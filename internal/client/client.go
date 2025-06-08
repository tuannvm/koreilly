package client

import (
	"context"
	"io"
	"math/rand"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// Client is a custom HTTP client with retry and rate limiting
// capabilities for interacting with the O'Reilly API.
type Client struct {
	client      *http.Client
	baseURL     string
	rateLimiter *rate.Limiter
	retryPolicy RetryPolicy
}

// RetryPolicy defines the retry behavior for failed requests
type RetryPolicy struct {
	MaxRetries int
	MinDelay   time.Duration
	MaxDelay   time.Duration
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: 3,
		MinDelay:   100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
	}
}

// New creates a new HTTP client with the specified configuration
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		client:      &http.Client{Timeout: 30 * time.Second},
		baseURL:     baseURL,
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 10), // 10 requests per second
		retryPolicy: DefaultRetryPolicy(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Option configures the Client
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.client = httpClient
	}
}

// WithRateLimit sets a custom rate limit
func WithRateLimit(limit rate.Limit, burst int) Option {
	return func(c *Client) {
		c.rateLimiter = rate.NewLimiter(limit, burst)
	}
}

// WithRetryPolicy sets a custom retry policy
func WithRetryPolicy(policy RetryPolicy) Option {
	return func(c *Client) {
		c.retryPolicy = policy
	}
}

// Get performs an HTTP GET request with retries
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	return c.doWithRetry(req)
}

// Post performs an HTTP POST request with retries
func (c *Client) Post(ctx context.Context, path, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	return c.doWithRetry(req)
}

// doWithRetry executes the request with retries based on the retry policy
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= c.retryPolicy.MaxRetries; attempt++ {
		// Wait for rate limiter
		if err := c.rateLimiter.Wait(req.Context()); err != nil {
			return nil, err
		}

		// Execute the request
		resp, err = c.client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// Close the response body if it exists
		if resp != nil {
			resp.Body.Close()
		}

		// Don't retry if we've hit max retries or the error is not retryable
		if attempt == c.retryPolicy.MaxRetries || !isRetryableError(err) {
			break
		}

		// Calculate backoff
		backoff := c.calculateBackoff(attempt)
		time.Sleep(backoff)
	}

	return nil, err
}

// isRetryableError checks if the error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Add more conditions as needed
	return true
}

// calculateBackoff calculates the backoff duration
func (c *Client) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff with jitter
	backoff := c.retryPolicy.MinDelay * time.Duration(1<<uint(attempt))
	if backoff > c.retryPolicy.MaxDelay {
		backoff = c.retryPolicy.MaxDelay
	}

	// Add jitter
	jitter := time.Duration(0.75 * float64(backoff))
	return backoff/2 + time.Duration(rand.Int63n(int64(jitter)))
}
