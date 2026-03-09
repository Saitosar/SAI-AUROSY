package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is the SAI AUROSY API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Option configures the client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// New creates a new client with API key authentication.
// baseURL is the Control Plane base URL including the API path, e.g.:
//   - http://localhost:8080/v1 (standalone)
//   - https://api.example.com/api/v1 (behind proxy)
// If baseURL does not end with /v1, /v1 is appended.
func New(baseURL, apiKey string, opts ...Option) *Client {
	u := baseURL
	if u != "" && u[len(u)-1] == '/' {
		u = u[:len(u)-1]
	}
	if u != "" && len(u) >= 3 && u[len(u)-3:] != "/v1" {
		u = u + "/v1"
	}
	c := &Client{
		baseURL:    u,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// do performs an HTTP request with API key auth.
func (c *Client) do(ctx context.Context, method, path string, body interface{}, tenantID string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if tenantID != "" {
		q := req.URL.Query()
		q.Set("tenant_id", tenantID)
		req.URL.RawQuery = q.Encode()
	}
	return c.httpClient.Do(req)
}

// doJSON performs a request and decodes the JSON response.
func (c *Client) doJSON(ctx context.Context, method, path string, body interface{}, tenantID string, out interface{}) error {
	resp, err := c.do(ctx, method, path, body, tenantID)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Body: string(b)}
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// APIError represents an API error response.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Body)
}
