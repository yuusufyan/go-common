package utils

import (
	"context"
	"net/http"
	"time"

	"github.com/yuusufyan/go-common/pkg/logger"
)

// TracingClient is a custom HTTP client that automatically forwards trace headers
type TracingClient struct {
	client *http.Client
}

// NewTracingClient creates a new HTTP client with default timeouts
func NewTracingClient(timeout time.Duration) *TracingClient {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &TracingClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Do executes an HTTP request with automatic tracing and retry logic
func (tc *TracingClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Extract TraceID and RequestID from context
	traceID, _ := ctx.Value(logger.TraceIDKey).(string)
	requestID, _ := ctx.Value(logger.RequestIDKey).(string)

	// Inject into headers
	if traceID != "" {
		req.Header.Set("X-Trace-ID", traceID)
	}
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	var resp *http.Response
	var err error

	// Retry logic: 3 attempts
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		resp, err = tc.client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// If it's a 5xx error or network error, retry after a short delay
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
		}
	}

	return resp, err
}

// Get is a helper for GET requests
func (tc *TracingClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return tc.Do(req)
}
