package wiremock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
)

const (
	wiremockAdminURN         = "__admin"
	wiremockAdminMappingsURN = "__admin/mappings"
)

// A Client implements requests to the wiremock server.
type Client struct {
	url       string
	stubs     map[*testing.T][]*StubRule
	stubMutex sync.Mutex
}

// NewClient returns *Client.
func NewClient(url string) *Client {
	return &Client{
		url:   url,
		stubs: make(map[*testing.T][]*StubRule),
	}
}

// StubFor creates a new stub mapping.
func (c *Client) StubFor(stubRule *StubRule) error {
	requestBody, err := stubRule.MarshalJSON()
	if err != nil {
		return fmt.Errorf("build stub request error: %s", err.Error())
	}

	res, err := http.Post(fmt.Sprintf("%s/%s", c.url, wiremockAdminMappingsURN), "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("stub request error: %s", err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("read response error: %s", err.Error())
		}

		return fmt.Errorf("bad response status: %d, response: %s", res.StatusCode, string(bodyBytes))
	}

	return nil
}

// StubForTest creates a new stub mapping for given test t
func (c *Client) StubForTest(t *testing.T, stubRule *StubRule) {
	requestBody, err := stubRule.MarshalJSON()
	if err != nil {
		t.Fatalf("build stub request error: %s", err)
	}

	res, err := http.Post(fmt.Sprintf("%s/%s", c.url, wiremockAdminMappingsURN), "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("stub request error: %s", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Errorf("close response body error: %s", err)
		}
	}()

	if res.StatusCode != http.StatusCreated {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("read response error: %s", err)
		}
		t.Fatalf("bad response status: %d, response: %s", res.StatusCode, string(bodyBytes))
	}

	c.stubMutex.Lock()
	defer c.stubMutex.Unlock()

	if len(c.stubs[t]) == 0 {
		c.stubs[t] = make([]*StubRule, 0)
	}
	c.stubs[t] = append(c.stubs[t], stubRule)
}

// Clear deletes all stub mappings.
func (c *Client) Clear() error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%s", c.url, wiremockAdminMappingsURN), nil)
	if err != nil {
		return fmt.Errorf("build cleare Request error: %s", err.Error())
	}

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("clear Request error: %s", err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response status: %d", res.StatusCode)
	}

	return nil
}

// ClearForTest deletes all stub mappings of given test t
func (c *Client) ClearForTest(t *testing.T) {
	c.stubMutex.Lock()
	defer c.stubMutex.Unlock()

	if len(c.stubs[t]) == 0 {
		return
	}

	for _, stubRule := range c.stubs[t] {
		if err := c.DeleteStub(stubRule); err != nil {
			t.Fatalf("delete stub error: %s", err)
		}
	}

	delete(c.stubs, t)
}

// Reset restores stub mappings to the defaults defined back in the backing store.
func (c *Client) Reset() error {
	res, err := http.Post(fmt.Sprintf("%s/%s/reset", c.url, wiremockAdminMappingsURN), "application/json", nil)
	if err != nil {
		return fmt.Errorf("reset Request error: %s", err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("read response error: %s", err.Error())
		}

		return fmt.Errorf("bad response status: %d, response: %s", res.StatusCode, string(bodyBytes))
	}

	return nil
}

// ResetAllScenarios resets back to start of the state of all configured scenarios.
func (c *Client) ResetAllScenarios() error {
	res, err := http.Post(fmt.Sprintf("%s/%s/scenarios/reset", c.url, wiremockAdminURN), "application/json", nil)
	if err != nil {
		return fmt.Errorf("reset all scenarios Request error: %s", err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("read response error: %s", err.Error())
		}

		return fmt.Errorf("bad response status: %d, response: %s", res.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetCountRequests gives count requests by criteria.
func (c *Client) GetCountRequests(r *Request) (int64, error) {
	requestBody, err := r.MarshalJSON()
	if err != nil {
		return 0, fmt.Errorf("get count requests: build error: %s", err.Error())
	}

	res, err := http.Post(fmt.Sprintf("%s/%s/requests/count", c.url, wiremockAdminURN), "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, fmt.Errorf("get count requests: %s", err.Error())
	}
	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, fmt.Errorf("get count requests: read response error: %s", err.Error())
	}

	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("get count requests: bad response status: %d, response: %s", res.StatusCode, string(bodyBytes))
	}

	var countRequestsResponse struct {
		Count int64 `json:"count"`
	}

	err = json.Unmarshal(bodyBytes, &countRequestsResponse)
	if err != nil {
		return 0, fmt.Errorf("get count requests: read json error: %s", err.Error())
	}

	return countRequestsResponse.Count, nil
}

// Verify checks count of request sent.
func (c *Client) Verify(r *Request, expectedCount int64) (bool, error) {
	actualCount, err := c.GetCountRequests(r)
	if err != nil {
		return false, err
	}

	return actualCount == expectedCount, nil
}

// VerifyForTest checks count of request sent.
func (c *Client) VerifyForTest(t *testing.T, r *Request, expectedCount int64) bool {
	actualCount, err := c.GetCountRequests(r)
	if err != nil {
		t.Fatalf("get count requests error: %s", err)
	}

	if actualCount != expectedCount {
		t.Errorf("number of request is not match, expect %d, actual %d", expectedCount, actualCount)
		return false
	}
	return true
}

// DeleteStubByID deletes stub by id.
func (c *Client) DeleteStubByID(id string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%s/%s", c.url, wiremockAdminMappingsURN, id), nil)
	if err != nil {
		return fmt.Errorf("delete stub by id: build request error: %s", err.Error())
	}

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("delete stub by id: request error: %s", err.Error())
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("read response error: %s", err.Error())
		}

		return fmt.Errorf("bad response status: %d, response: %s", res.StatusCode, string(bodyBytes))
	}

	return nil
}

// DeleteStub deletes stub mapping.
func (c *Client) DeleteStub(s *StubRule) error {
	return c.DeleteStubByID(s.UUID())
}

// BuildTestEndpoint returns endpoint and expectAPIPath for given test t
func (c Client) BuildTestEndpoint(t *testing.T, apiPath string) (endpoint, expectAPIPath string) {
	if string(apiPath[0]) != "/" {
		apiPath = "/" + apiPath
	}
	expectAPIPath = fmt.Sprintf("/%p%s", t, apiPath)
	endpoint = fmt.Sprintf("%s%s", c.url, expectAPIPath)
	return
}

// URL ...
func (c *Client) URL() string {
	return c.url
}
