package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"golang.org/x/net/publicsuffix"
	"golang.org/x/time/rate"
)

// Default headers
const (
	userAgent        = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	acceptHeader     = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"
	acceptLangHeader = "en-US,en;q=0.5"
	connectionHeader = "keep-alive"
)

// Client represents an HTTP client with retry and rate limiting capabilities
type Client struct {
	baseURL     string
	client      *http.Client
	headers     map[string]string
	rateLimiter *rate.Limiter
	retryPolicy *RetryPolicy
	logger      *log.Logger
}

// SetDefaultHeader sets a default header that will be included in all requests
func (c *Client) SetDefaultHeader(key, value string) {
	if c.headers == nil {
		c.headers = make(map[string]string)
	}
	c.headers[key] = value
}

// RetryPolicy defines the retry behavior for failed requests
type RetryPolicy struct {
	// MaxRetries is the maximum number of retries
	MaxRetries int
	// RetryableStatusCodes is a list of status codes that should be retried
	RetryableStatusCodes []int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
}

// ShouldRetry checks if a status code should be retried
func (r *RetryPolicy) ShouldRetry(statusCode int) bool {
	for _, code := range r.RetryableStatusCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}

// CalculateBackoff calculates the backoff duration for a retry attempt
func (r *RetryPolicy) CalculateBackoff(attempt int) time.Duration {
	if r.InitialBackoff == 0 {
		r.InitialBackoff = 100 * time.Millisecond
	}
	if r.MaxBackoff == 0 {
		r.MaxBackoff = 5 * time.Second
	}

	backoff := float64(r.InitialBackoff) * math.Pow(2, float64(attempt))
	jitter := 0.5 + rand.Float64()
	delay := time.Duration(backoff * jitter)

	// Cap the delay at the maximum allowed
	if delay > r.MaxBackoff {
		delay = r.MaxBackoff
	}

	return delay
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:           3,
		RetryableStatusCodes: []int{500, 502, 503, 504},
		InitialBackoff:       100 * time.Millisecond,
		MaxBackoff:           5 * time.Second,
	}
}

// New creates a new HTTP client with the specified configuration
func New(baseURL string, opts ...Option) *Client {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		DisableCompression: false,
		ForceAttemptHTTP2:  true,
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
	}

	httpClient := &http.Client{
		Jar:       jar,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Preserve headers during redirects
			if len(via) > 0 {
				req.Header = via[0].Header.Clone()
			}
			return nil
		},
		Timeout: 60 * time.Second,
	}

	return NewWithHTTPClient(baseURL, httpClient, opts...)
}

// NewWithHTTPClient creates a new client with a custom HTTP client
func NewWithHTTPClient(baseURL string, httpClient *http.Client, opts ...Option) *Client {
	c := &Client{
		baseURL:     baseURL,
		client:      httpClient,
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 10), // 10 requests per second
		retryPolicy: DefaultRetryPolicy(),
		headers:     make(map[string]string),
	}

	// Set default headers
	c.SetDefaultHeader("User-Agent", userAgent)
	c.SetDefaultHeader("Accept", acceptHeader)
	c.SetDefaultHeader("Accept-Language", acceptLangHeader)
	c.SetDefaultHeader("Connection", connectionHeader)

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Option configures the Client
type Option func(*Client)

// GetHTTPClient exposes the underlying http.Client for advanced use.
func (c *Client) GetHTTPClient() *http.Client {
	return c.client
}

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

// WithRetryPolicy sets the retry policy for the client
func WithRetryPolicy(policy *RetryPolicy) Option {
	return func(c *Client) {
		c.retryPolicy = policy
	}
}

// newRequest creates a new HTTP request with the given method, path, and body
func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// do performs the HTTP request with retry and rate limiting
func (c *Client) do(req *http.Request) (*http.Response, error) {
	// Apply rate limiting
	if err := c.rateLimiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Create a function to get a fresh body reader for retries
	var bodyBytes []byte
	var err error

	if req.Body != nil {
		// Read the body once and store it for retries
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		// Close the original body
		req.Body.Close()

		// Create a new body reader that can be reset
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// Set GetBody function to allow the body to be read multiple times
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}

		// Set ContentLength if not already set
		if req.ContentLength == 0 && len(bodyBytes) > 0 {
			req.ContentLength = int64(len(bodyBytes))
		}
	}

	// Send the request with retries
	var resp *http.Response

	for attempt := 0; attempt <= c.retryPolicy.MaxRetries; attempt++ {
		// Reset the request body for retries
		if req.GetBody != nil {
			var reqErr error
			if req.Body, reqErr = req.GetBody(); reqErr != nil {
				return nil, fmt.Errorf("failed to reset request body: %w", reqErr)
			}
		}

		resp, err = c.client.Do(req)

		// If no error and status code is not in retryable status codes, return the response
		if err == nil && !c.retryPolicy.ShouldRetry(resp.StatusCode) {
			return resp, nil
		}

		// If we get here, we need to retry
		if attempt < c.retryPolicy.MaxRetries {
			// Calculate backoff duration
			backoff := c.retryPolicy.CalculateBackoff(attempt)
			if c.logger != nil {
				status := 0
				if resp != nil {
					status = resp.StatusCode
				}
				c.logger.Printf("Request failed (attempt %d/%d), retrying in %v: %v (status: %d)\n",
					attempt+1, c.retryPolicy.MaxRetries, backoff, err, status)
			}

			// Wait for the backoff duration or context cancellation
			select {
			case <-time.After(backoff):
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}
	}

	// If we've exhausted all retries, return the last error
	if err != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", c.retryPolicy.MaxRetries+1, err)
	}

	return resp, nil
}

// Get performs a GET request to the specified path with optional headers
func (c *Client) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	// Add custom headers if provided
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.do(req)
}

// Post sends a POST request with retry and rate limiting
func (c *Client) Post(ctx context.Context, path, contentType string, body io.Reader) (*http.Response, error) {
	headers := map[string]string{
		"Content-Type": contentType,
	}
	return c.PostWithHeaders(ctx, path, headers, body)
}

// PostWithHeaders sends a POST request with custom headers and retry logic
func (c *Client) PostWithHeaders(ctx context.Context, path string, headers map[string]string, body io.Reader) (*http.Response, error) {
	// Read the entire body into memory so we can reuse it for retries
	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}
	}

	url := c.baseURL + path

	// Create a new request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set the body
	if len(bodyBytes) > 0 {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	}

	// Set default headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Add/override with provided headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Ensure we have a content type if there's a body
	if len(bodyBytes) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Add connection headers if not present
	if req.Header.Get("Connection") == "" {
		req.Header.Set("Connection", "keep-alive")
	}

	// Add accept header if not present
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}

	// Add user agent if not present
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", userAgent)
	}

	// Make the request
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// doWithRetry sends an HTTP request with retry logic
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i <= c.retryPolicy.MaxRetries; i++ {
		resp, err = c.client.Do(req)
		if err == nil {
			// Check if we should retry based on status code
			if !c.retryPolicy.ShouldRetry(resp.StatusCode) {
				return resp, nil
			}
		}

		// If this is the last attempt, break and return the error
		if i == c.retryPolicy.MaxRetries {
			break
		}

		// Calculate backoff and wait
		backoff := c.retryPolicy.CalculateBackoff(i)
		time.Sleep(backoff)
	}

	return nil, err
}

// isRetryableError checks if the error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Add logic to determine if the error is retryable
	// For example, network timeouts, temporary network errors, etc.
	// Add more conditions as needed
	return true
}

// GetCookies returns the cookies for the given URL from the client's cookie jar
func (c *Client) GetCookies(urlStr string) []*http.Cookie {
	if c.client.Jar == nil {
		return nil
	}

	url, err := url.Parse(urlStr)
	if err != nil {
		return nil
	}

	return c.client.Jar.Cookies(url)
}
