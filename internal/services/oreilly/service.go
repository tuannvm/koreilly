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

// getCSRFToken retrieves the CSRF token from the O'Reilly website
// Since the login page is a JavaScript-rendered SPA, we need to make a POST request to the login endpoint
// with the correct headers and form data to get the authentication token
func (s *Service) getCSRFToken(ctx context.Context) (string, error) {
	loginPage := "https://learning.oreilly.com/accounts/login/"

	// Initial GET to fetch login page + cookies
	headers := map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
		"User-Agent":      userAgent,
	}

	log.Printf("Fetching login page to obtain CSRF token: %s", loginPage)
	resp, err := s.client.Get(ctx, loginPage, headers)
	if err != nil {
		return "", fmt.Errorf("failed to GET login page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed reading login page: %w", err)
	}

	// First check for csrftoken cookie (Django style)
	var token string
	for _, c := range resp.Cookies() {
		if c.Name == "csrftoken" {
			token = c.Value
			break
		}
	}

	// If not in cookie, fallback to hidden input parsing
	if token == "" {
		re := regexp.MustCompile(`name=['"]csrfmiddlewaretoken['"][^>]*value=['"]([^'"]+)`)
		m := re.FindStringSubmatch(string(body))
		if len(m) >= 2 {
			token = m[1]
		}
	}

	if token == "" {
		return "", fmt.Errorf("csrfmiddlewaretoken not found (cookie or form)")
	}

	log.Printf("Obtained CSRF token: %s", token[:min(10, len(token))])
	return token, nil
}

// Login authenticates with O'Reilly using email and password
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	// The login endpoint for O'Reilly's API
	loginURL := "https://learning.oreilly.com/accounts/login/"

	// Get fresh CSRF token & cookies first
	csrfToken, err := s.getCSRFToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve CSRF token: %w", err)
	}

	// Prepare the form data
	formData := url.Values{}
	formData.Set("csrfmiddlewaretoken", csrfToken)
	formData.Set("email", email)
	formData.Set("password", password)
	formData.Set("next", "/")

	// Set up headers for the login request
	headers := map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
		"Content-Type":    "application/x-www-form-urlencoded",
		"Origin":          "https://learning.oreilly.com",
		"Referer":         "https://learning.oreilly.com/accounts/login/",
	}

	// Make the login request
	log.Printf("Sending login request to %s", loginURL)
	resp, err := s.client.PostWithHeaders(ctx, loginURL, headers, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read login response: %w", err)
	}

	// Log the response status and headers for debugging
	log.Printf("Login response status: %s", resp.Status)
	log.Printf("Response headers: %+v", resp.Header)

	// Check for successful login by looking for the user menu in the response
	if !strings.Contains(string(body), "account-dropdown-button") {
		// If we don't see the account dropdown, the login likely failed
		// Try to extract an error message from the response
		errMsg := "login failed - unknown error"
		if strings.Contains(string(body), "Please enter a correct email and password") {
			errMsg = "invalid email or password"
		} else if strings.Contains(string(body), "This account is inactive") {
			errMsg = "account is inactive"
		}
		return nil, fmt.Errorf("login failed: %s", errMsg)
	}

	// If we get here, the login was successful
	// Extract the JWT token from cookies
	var jwtToken string
	for _, cookie := range s.client.GetCookies("https://learning.oreilly.com") {
		log.Printf("Found cookie: %s=%s", cookie.Name, cookie.Value)
		if cookie.Name == "orm-jwt" || cookie.Name == "orm-rt" {
			jwtToken = cookie.Value
			break
		}
	}

	if jwtToken == "" {
		return nil, fmt.Errorf("login successful but no JWT token found in cookies")
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
