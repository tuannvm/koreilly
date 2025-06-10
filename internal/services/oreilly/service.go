package oreilly

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
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

// getCSRFToken retrieves the CSRF token from the main page
func (s *Service) getCSRFToken(ctx context.Context) (string, error) {
	headers := map[string]string{
		"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"User-Agent": userAgent,
	}

	// Try to get the CSRF token from the login page first
	loginPageURL := "https://www.oreilly.com/member/auth/login/"
	resp, err := s.client.Get(ctx, loginPageURL, headers)
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

	// Look for CSRF token in the HTML
	csrfToken, err := extractCSRFToken(string(body))
	if err == nil && csrfToken != "" {
		return csrfToken, nil
	}

	// If we didn't find the CSRF token in the login page, try the main page
	resp, err = s.client.Get(ctx, "https://www.oreilly.com", headers)
	if err != nil {
		return "", fmt.Errorf("failed to fetch homepage: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read homepage: %w", err)
	}

	if len(body) == 0 {
		return "", fmt.Errorf("received empty response body from homepage")
	}

	// Look for CSRF token in the HTML
	csrfToken, err = extractCSRFToken(string(body))
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

// extractCSRFToken extracts the CSRF token from the HTML
func extractCSRFToken(body string) (string, error) {
	// Try different patterns to find the CSRF token
	patterns := []*regexp.Regexp{
		// Look for: <input type="hidden" name="csrfmiddlewaretoken" value="TOKEN">
		regexp.MustCompile(`<input[^>]+name=["']csrfmiddlewaretoken["'][^>]+value=["']([^"']+)`),
		// Look for: <input name="csrfmiddlewaretoken" value="TOKEN" type="hidden">
		regexp.MustCompile(`<input[^>]+value=["']([^"']+)["'][^>]+name=["']csrfmiddlewaretoken["']`),
		// Look for: <meta name="csrf-token" content="TOKEN">
		regexp.MustCompile(`<meta[^>]+name=["']csrf-token["'][^>]+content=["']([^"']+)`),
		// Look for: window.csrfToken = "TOKEN"
		regexp.MustCompile(`window\.csrfToken\s*=\s*["']([^"']+)`),
		// Look for: "csrfToken":"TOKEN"
		regexp.MustCompile(`"csrfToken"\s*:\s*"([^"]+)"`),
		// Look for: <meta name="csrfmiddlewaretoken" content="TOKEN">
		regexp.MustCompile(`<meta[^>]+name=["']csrfmiddlewaretoken["'][^>]+content=["']([^"']+)`),
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

// Login authenticates with O'Reilly using email and password
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	log.Println("Starting login process...")
	
	// First, get the CSRF token (this will also set the necessary cookies)
	log.Println("Getting CSRF token...")
	csrfToken, err := s.getCSRFToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSRF token: %w", err)
	}
	log.Printf("Got CSRF token: %s...", csrfToken[:10])

	// Prepare the login form data
	formData := url.Values{
		"csrfmiddlewaretoken": {csrfToken},
		"email":               {email},
		"password":            {password},
		"next":                {"/"},
		"remember_me":         {"true"},
	}

	// Make the login request
	loginURL := "https://www.oreilly.com/member/auth/login/"
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Accept":       "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Origin":       "https://www.oreilly.com",
		"Referer":      "https://www.oreilly.com/member/auth/login/",
		"Accept-Language": "en-US,en;q=0.9",
		"DNT":           "1",
		"Connection":    "keep-alive",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Dest": "document",
		"Sec-Fetch-Mode": "navigate",
		"Sec-Fetch-Site": "same-origin",
		"Sec-Fetch-User": "?1",
		"Cache-Control": "no-cache",
	}

	// Log the request for debugging
	log.Printf("Sending login request to %s", loginURL)
	log.Printf("Request headers: %+v", headers)
	log.Printf("Form data: %+v", formData)

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

	// Read the response body for debugging
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*4))
	bodyStr := string(body)
	log.Printf("Login response status: %d", resp.StatusCode)
	log.Printf("Response headers: %+v", resp.Header)
	log.Printf("Response body (first 500 chars): %s", bodyStr[:min(500, len(bodyStr))])

	// Check for common error cases in the response body
	switch {
	case strings.Contains(bodyStr, "The email or password you entered is incorrect"):
		return nil, fmt.Errorf("invalid email or password")
	case strings.Contains(bodyStr, "This account is inactive"):
		return nil, fmt.Errorf("account is inactive")
	case strings.Contains(bodyStr, "Too many failed login attempts"):
		return nil, fmt.Errorf("too many failed login attempts - please try again later")
	case strings.Contains(bodyStr, "CSRF verification failed"):
		return nil, fmt.Errorf("session expired - please try again")
	case strings.Contains(bodyStr, "<title>Sign In</title>"):
		return nil, fmt.Errorf("login failed - please check your credentials")
	case strings.Contains(bodyStr, "<form"):
		// If we got a form back, it likely means the login failed but the server didn't provide a specific error
		return nil, fmt.Errorf("login failed - please check your credentials")
	}

	// Check if login was successful (should be a redirect to the home page or profile)
	log.Printf("Checking if login was successful...")
	if resp.StatusCode != http.StatusFound {
		log.Printf("Unexpected status code: %d (expected %d)", resp.StatusCode, http.StatusFound)
		// Try to extract error message from the response
		errMsg := "login failed - please check your credentials"
		
		// Try to find error message in common locations
		errorPatterns := []*regexp.Regexp{
			regexp.MustCompile(`<div[^>]*class=["'][^"']*error[^"']*["'][^>]*>([^<]+)</div>`),
			regexp.MustCompile(`<p[^>]*class=["'][^"']*error[^"']*["'][^>]*>([^<]+)</p>`),
			regexp.MustCompile(`<div[^>]*role=["']alert["'][^>]*>([^<]+)</div>`),
			regexp.MustCompile(`<p[^>]*role=["']alert["'][^>]*>([^<]+)</p>`),
		}

		for _, pattern := range errorPatterns {
			if match := pattern.FindStringSubmatch(bodyStr); len(match) > 1 {
				if msg := strings.TrimSpace(match[1]); msg != "" {
					errMsg = msg
					log.Printf("Found error message in response: %s", errMsg)
					break
				}
			}
		}

		return nil, fmt.Errorf("%s", errMsg)
	}

	// Get the location header to check for successful login
	location, err := resp.Location()
	if err != nil {
		// If we can't get the location, it might still be a successful login
		log.Printf("Warning: Failed to get redirect location: %v", err)
	} else {
		log.Printf("Redirected to: %s", location.String())
		// Check if we were redirected back to the login page (failed login)
		if strings.Contains(location.Path, "/auth/login/") {
			return nil, fmt.Errorf("login failed - invalid credentials")
		}
	}

	// Extract the JWT token from cookies
	var jwtToken string
	for _, cookie := range s.client.GetCookies("https://www.oreilly.com") {
		log.Printf("Cookie: %s=%s", cookie.Name, cookie.Value)
		if cookie.Name == "orm-jwt" {
			jwtToken = cookie.Value
			break
		}
	}

	if jwtToken == "" {
		return nil, fmt.Errorf("no JWT token found in cookies after login")
	}

	// Verify login by accessing profile page
	profileURL := "https://www.oreilly.com/member/profile/"
	profileResp, err := s.client.Get(ctx, profileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to verify login: %w", err)
	}
	defer profileResp.Body.Close()

	if profileResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to verify login, status: %d", profileResp.StatusCode)
	}

	log.Printf("Successfully logged in as %s", email)
	return &LoginResponse{
		AccessToken: jwtToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // Default expiration time
	}, nil
}

// verifyLogin verifies that the login was successful by accessing the profile page
func (s *Service) verifyLogin(ctx context.Context) error {
	// Try to access the profile page
	profileURL := "https://www.oreilly.com/member/profile/"
	headers := map[string]string{
		"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
	}

	resp, err := s.client.Get(ctx, profileURL, headers)
	if err != nil {
		return fmt.Errorf("failed to verify login: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read profile page: %w", err)
	}

	// Convert body to string for easier inspection
	bodyStr := string(body)

	// Check for common error cases
	switch {
	case strings.Contains(bodyStr, `"user_type":"Expired"`):
		return fmt.Errorf("account subscription has expired")
	case strings.Contains(bodyStr, "signin"):
		return fmt.Errorf("login failed: you are not signed in")
	case resp.StatusCode >= 400:
		return fmt.Errorf("login verification failed with status %d", resp.StatusCode)
	}

	// If we got here, the login was successful
	return nil
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
