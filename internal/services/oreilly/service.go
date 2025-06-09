package oreilly

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tuannvm/goreilly/internal/client"
	"golang.org/x/net/publicsuffix"
)

const (
	defaultBaseURL    = "https://www.oreilly.com"
	safariBaseURL     = "https://learning.oreilly.com"
	loginPage         = "/member/auth/login/"
	loginEntryURL     = "/member/auth/login/"
	profileURL        = "/profile/"
	apiBaseURL        = "https://api.oreilly.com"
	userAgent         = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	acceptHeader      = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"
	acceptLangHeader  = "en-US,en;q=0.5"
	connectionHeader  = "keep-alive"
	upgradeInsecure  = "1"
	contentTypeForm   = "application/x-www-form-urlencoded"
)

// Service represents the O'Reilly service
type Service struct {
	client    *client.Client
	sessionID string
	jwtToken  string
	baseURL   string // Track the current base URL
}

// loginResponse represents the response from the login API
type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	RedirectURI string `json:"redirect_uri"`
}

// LoginResponse represents the response from the O'Reilly login API
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// LoginRequest represents the request payload for O'Reilly login
type LoginRequest struct {
	Csrfmiddlewaretoken string `form:"csrfmiddlewaretoken"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	Next                string `form:"next,omitempty"`
	RememberMe          bool   `form:"remember_me"`
}

// NewService creates a new O'Reilly service
func NewService() (*Service, error) {
	// Create a cookie jar to handle cookies automatically
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Create a custom HTTP client with cookie support and disabled SSL verification
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Skip SSL verification
	}

	httpClient := &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Preserve headers during redirects
			req.Header = via[0].Header.Clone()
			return nil
		},
	}

	// Create a custom client with our HTTP client
	c := client.NewWithHTTPClient("", httpClient) // Empty base URL since we'll handle it in the methods

	// Set default headers
	c.SetDefaultHeader("User-Agent", userAgent)
	c.SetDefaultHeader("Accept", acceptHeader)
	c.SetDefaultHeader("Accept-Language", acceptLangHeader)
	c.SetDefaultHeader("Connection", connectionHeader)
	c.SetDefaultHeader("Upgrade-Insecure-Requests", upgradeInsecure)

	return &Service{
		client:  c,
		baseURL: defaultBaseURL,
	}, nil
}

// getCSRFToken retrieves the CSRF token from the login page
func (s *Service) getCSRFToken(ctx context.Context) (string, error) {
	// Set headers for the login page request
	headers := map[string]string{
		"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"User-Agent": userAgent,
	}

	// Use the client's Get method with headers
	resp, err := s.client.Get(ctx, s.baseURL+loginPage, headers)
	if err != nil {
		return "", fmt.Errorf("failed to fetch login page: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read login page: %w", err)
	}

	if len(body) == 0 {
		return "", fmt.Errorf("received empty response body from login page")
	}

	// Debug: Save the response to a file for inspection
	// os.WriteFile("login_page.html", body, 0644)


	// Look for CSRF token in the HTML
	csrfToken, err := extractCSRFToken(string(body))
	if err != nil {
		// Try to get more debug info
		dbgInfo := string(body)
		if len(dbgInfo) > 500 {
			dbgInfo = dbgInfo[:500] + "..."
		}
		return "", fmt.Errorf("failed to extract CSRF token (status %d): %w\nResponse (first 500 chars):\n%s", 
			resp.StatusCode, err, dbgInfo)
	}

	if csrfToken == "" {
		return "", fmt.Errorf("empty CSRF token found in response")
	}

	return csrfToken, nil
}

// extractCSRFToken extracts the CSRF token from the login page HTML
func extractCSRFToken(body string) (string, error) {
	// Try different patterns to find the CSRF token
	patterns := []*regexp.Regexp{
		// Look for: <input type="hidden" name="csrfmiddlewaretoken" value="TOKEN">
		regexp.MustCompile(`<input[^>]+name=["']csrfmiddlewaretoken["'][^>]+value=["']([^"']+)`),
		// Look for: <input name="csrfmiddlewaretoken" value="TOKEN" type="hidden">
		regexp.MustCompile(`<input[^>]+value=["']([^"']+)["'][^>]+name=["']csrfmiddlewaretoken["']`),
		// Look for: <input type='hidden' name='csrfmiddlewaretoken' value='TOKEN'/>
		regexp.MustCompile(`<input[^>]+name=['"]csrfmiddlewaretoken['"][^>]+value=['"]([^'"]+)`),
		// Look for CSRF token in meta tag
		regexp.MustCompile(`<meta[^>]+name=["']csrf-token['"][^>]+content=["']([^"']+)`),
		// Look for CSRF token in cookies (if we're handling cookies)
		regexp.MustCompile(`(?i)csrftoken=([^;]+)`),
	}

	// First try to find in the HTML body
	for _, re := range patterns {
		matches := re.FindStringSubmatch(body)
		if len(matches) > 1 {
			token := strings.TrimSpace(matches[1])
			if token != "" {
				return token, nil
			}
		}
	}

	// Try to find any form input that might contain the token
	re := regexp.MustCompile(`<input[^>]+name=["']([^"']+)["'][^>]+value=["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(body, -1)
	for _, match := range matches {
		if len(match) >= 3 && strings.Contains(strings.ToLower(match[1]), "csrf") {
			// Found a potential CSRF token in a form input
			return strings.TrimSpace(match[2]), nil
		}
	}

	// Try to look for any hidden input
	hiddenRe := regexp.MustCompile(`<input[^>]+type=["']hidden["'][^>]+>`)
	hiddenInputs := hiddenRe.FindAllString(body, -1)
	for _, input := range hiddenInputs {
		if nameMatch := regexp.MustCompile(`name=["']([^"']+)["']`).FindStringSubmatch(input); len(nameMatch) > 1 {
			if strings.Contains(strings.ToLower(nameMatch[1]), "csrf") {
				if valueMatch := regexp.MustCompile(`value=["']([^"']+)["']`).FindStringSubmatch(input); len(valueMatch) > 1 {
					return strings.TrimSpace(valueMatch[1]), nil
				}
			}
		}
	}

	// Return a more helpful error message
	debugLen := 500
	if len(body) < debugLen {
		debugLen = len(body)
	}
	
	return "", fmt.Errorf("CSRF token not found in response. First %d chars: %s", 
		debugLen, body[:debugLen])
}

// Login authenticates with O'Reilly and returns a token
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	// First, make a GET request to get the login page and set cookies
	loginPageURL := "https://www.oreilly.com/member/auth/login/"
	
	initialHeaders := map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.5",
		"Connection":      "keep-alive",
		"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/115.0",
	}

	// Get the login page to set initial cookies
	_, err := s.client.Get(ctx, loginPageURL, initialHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to load login page: %w", err)
	}

	// Now make the login request
	loginURL := "https://www.oreilly.com/member/auth/login/"

	// Prepare form data
	formData := url.Values{}
	formData.Set("email", email)
	formData.Set("password", password)
	formData.Set("next", "/")

	// Set up the request headers
	headers := map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Accept":        "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Origin":        "https://www.oreilly.com",
		"Referer":       loginPageURL,
		"User-Agent":    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/115.0",
		"Cache-Control": "no-cache",
	}

	// Make the login request
	resp, err := s.client.PostWithHeaders(
		ctx,
		loginURL,
		headers,
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read login response: %w", err)
	}

	// Check for successful login (302 redirect on success)
	if resp.StatusCode != http.StatusFound {
		// Try to extract error message from response
		errMsg := ""
		if len(body) > 200 {
			errMsg = string(body[:200]) + "..."
		} else {
			errMsg = string(body)
		}
		return nil, fmt.Errorf("login failed with status %d: %s", resp.StatusCode, errMsg)
	}

	// Check if we were redirected to the home page (successful login)
	location, err := resp.Location()
	if err == nil && location.Path == "/" {
		// Get the JWT token from cookies
		cookies := resp.Cookies()
		var jwtToken string
		for _, cookie := range cookies {
			if cookie.Name == "jwt_token" || cookie.Name == "auth_token" {
				jwtToken = cookie.Value
				break
			}
		}

		if jwtToken == "" {
			return nil, fmt.Errorf("JWT token not found in cookies")
		}

		// Save the JWT token for future requests
		s.jwtToken = jwtToken

		// Verify the login by accessing the profile page
		if err := s.verifyLogin(ctx); err != nil {
			return nil, fmt.Errorf("login verification failed: %w", err)
		}

		// Return the token
		return &LoginResponse{
			AccessToken: jwtToken,
			TokenType:   "Bearer",
			ExpiresIn:   3600, // Default expiration time
		}, nil
	}

	return nil, fmt.Errorf("login failed: unexpected redirect to %s", location)
}

// verifyLogin verifies that the login was successful by accessing the profile page
func (s *Service) verifyLogin(ctx context.Context) error {
	// Switch to the Safari base URL for profile access
	oldBaseURL := s.baseURL
	s.baseURL = safariBaseURL
	defer func() { s.baseURL = oldBaseURL }()

	resp, err := s.client.Get(ctx, s.baseURL+profileURL, nil)
	if err != nil {
		return fmt.Errorf("failed to access profile page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to verify login, status: %d", resp.StatusCode)
	}

	// Check if the account is expired
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read profile response: %w", err)
	}

	if strings.Contains(string(body), `"user_type":"Expired"`) {
		return fmt.Errorf("account subscription has expired")
	}

	return nil
}
