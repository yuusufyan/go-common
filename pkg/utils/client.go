package utils

import (
	"context"
	"net/http"
	"time"

	"github.com/yuusufyan/go-common/pkg/logger"
)

// TracingClientConfig configures NewTracingClientWithConfig. Zero values fall back to the defaults.
type TracingClientConfig struct {
	// Timeout per request (default: 30s).
	Timeout time.Duration
	// MaxRetries is the total number of attempts on network / 5xx errors (default: 3).
	MaxRetries int
	// RetryDelay is the base backoff, multiplied by the attempt number (default: 100ms).
	RetryDelay time.Duration
	// Transport overrides the underlying http.RoundTripper (optional).
	Transport http.RoundTripper
}

// TracingClient is a custom HTTP client that automatically forwards trace headers
type TracingClient struct {
	client     *http.Client
	maxRetries int
	retryDelay time.Duration
}

// NewTracingClient creates a new HTTP client with default timeouts
func NewTracingClient(timeout time.Duration) *TracingClient {
	return NewTracingClientWithConfig(TracingClientConfig{Timeout: timeout})
}

// NewTracingClientWithConfig creates a new HTTP client from cfg.
func NewTracingClientWithConfig(cfg TracingClientConfig) *TracingClient {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 100 * time.Millisecond
	}
	return &TracingClient{
		client: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: cfg.Transport,
		},
		maxRetries: cfg.MaxRetries,
		retryDelay: cfg.RetryDelay,
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

	// A request with a body can only be retried if the body can be re-read
	maxRetries := tc.maxRetries
	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		maxRetries = 1
	}

	var resp *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		if i > 0 && req.GetBody != nil {
			body, bodyErr := req.GetBody()
			if bodyErr != nil {
				return nil, bodyErr
			}
			req.Body = body
		}

		resp, err = tc.client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// If it's a 5xx error or network error, retry after a short delay
		if i < maxRetries-1 {
			if resp != nil {
				resp.Body.Close()
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(i+1) * tc.retryDelay):
			}
		}
	}

	return resp, err
}

// Get is a helper for GET requests
func (tc *TracingClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return tc.Do(req)
}
