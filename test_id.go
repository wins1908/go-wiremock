package wiremock

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

const (
	// TestIDRequestHeader ...
	// do not upper case "ID", request header is case insensitive
	TestIDRequestHeader = "X-Wiremock-Test-Id"
	// TestIDJSONField ...
	TestIDJSONField = "wiremock_test_id"

	testIDCtxKey = "test_id"
)

// ContextForTest set test ID into context
func ContextForTest(ctx context.Context, t *testing.T) context.Context {
	return ContextWithTestID(ctx, CreateTestID(t))
}

// RequestForTest ...
func RequestForTest(req *http.Request, t *testing.T) *http.Request {
	req.Header.Set(TestIDRequestHeader, CreateTestID(t))
	return req
}

// StubRuleForTest ...
func StubRuleForTest(t *testing.T, rule *StubRule) *StubRule {
	return rule.WithHeader(TestIDRequestHeader, EqualTo(CreateTestID(t)))
}

// HeaderForTest ...
func HeaderForTest(headers map[string]interface{}, t *testing.T) {
	headers[TestIDJSONField] = CreateTestID(t)
}

// ContextWithTestID set test ID into context
func ContextWithTestID(ctx context.Context, testID string) context.Context {
	return context.WithValue(ctx, testIDCtxKey, testID)
}

// TestIDFromContext ...
func TestIDFromContext(ctx context.Context) string {
	testID, _ := ctx.Value(testIDCtxKey).(string)
	return testID
}

// TestIDToContextMiddleware ...
func TestIDToContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		if testID := req.Header.Get(TestIDRequestHeader); testID != "" {
			req = req.WithContext(ContextWithTestID(ctx, testID))
		}

		next.ServeHTTP(w, req)
	})
}

// TestIDToOutgoingRequestHeaderMiddleware ...
func TestIDToOutgoingRequestHeaderMiddleware(next http.RoundTripper) http.RoundTripper {
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if testID := TestIDFromContext(req.Context()); testID != "" {
			req.Header.Set(TestIDRequestHeader, testID)
		}
		return next.RoundTrip(req)
	})
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// CreateTestID ...
func CreateTestID(t *testing.T) string {
	return fmt.Sprintf("%s:%p", t.Name(), t)
}
