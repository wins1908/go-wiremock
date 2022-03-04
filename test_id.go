package wiremock

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

const (
	// TestIDHeader ...
	TestIDHeader = "X-Wiremock-Test-ID"

	testIDCtxKey = "test_id"
)

// ContextForTest set test ID into context
func ContextForTest(ctx context.Context, t *testing.T) context.Context {
	return context.WithValue(ctx, testIDCtxKey, CreateTestID(t))
}

// RequestForTest ...
func RequestForTest(req *http.Request, t *testing.T) *http.Request {
	req.Header.Set(TestIDHeader, CreateTestID(t))
	return req
}

// HeaderForTest ...
func HeaderForTest(headers map[string]interface{}, t *testing.T) {
	headers[TestIDHeader] = CreateTestID(t)
}

// TestIDToContextMiddleware ...
func TestIDToContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		if testID := req.Header.Get(TestIDHeader); testID != "" {
			req = req.WithContext(context.WithValue(ctx, testIDCtxKey, testID))
		}

		next.ServeHTTP(w, req)
	})
}

// TestIDToOutgoingRequestHeaderMiddleware ...
func TestIDToOutgoingRequestHeaderMiddleware(next http.RoundTripper) http.RoundTripper {
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if testID, ok := req.Context().Value(testIDCtxKey).(string); ok && testID != "" {
			req.Header.Set(TestIDHeader, testID)
		}
		return next.RoundTrip(req)
	})
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func CreateTestID(t *testing.T) string {
	return fmt.Sprintf("%s:%p", t.Name(), t)
}
