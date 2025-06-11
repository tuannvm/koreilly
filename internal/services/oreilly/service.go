package oreilly

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/tuannvm/goreilly/internal/client"
	"golang.org/x/net/publicsuffix"
)

const (
	defaultBaseURL   = "https://www.oreilly.com"
	safariBaseURL    = "https://learning.oreilly.com"
	loginPage        = "/member/auth/login/"
	loginEntryURL    = "/member/auth/login/"
	profileURL       = "/profile/"
	apiBaseURL       = "https://api.oreilly.com"
	userAgent        = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	acceptHeader     = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"
	acceptLangHeader = "en-US,en;q=0.5"
	connectionHeader = "keep-alive"
	upgradeInsecure  = "1"
	contentTypeForm  = "application/x-www-form-urlencoded"
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

// getCSRFToken is now unused (CSRF is not required in new login flow)

// Login authenticates with O'Reilly using email and password with SafariBooks multi-step method.
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	// Step 1: GET the unified login page to establish cookies
	unifiedLoginURL := "https://learning.oreilly.com/login/unified/?next=/home/"
	req, err := http.NewRequestWithContext(ctx, "GET", unifiedLoginURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct GET for unified login: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := s.client.GetHTTPClient().Do(req)
	if err != nil {
		log.Printf("GET unified login page failed: %v", err)
		return nil, fmt.Errorf("failed to GET unified login page: %w", err)
	}
	log.Printf("[Unified Login GET] Status: %v", resp.Status)
	for k, v := range resp.Header {
		log.Printf("[Unified Login GET] Header: %s=%v", k, v)
	}
	for i, c := range resp.Cookies() {
		log.Printf("[Unified Login GET] Cookie[%d]: %s=%s", i, c.Name, c.Value)
	}
	_ = resp.Body.Close()

	// Step 2: POST creds as JSON to real login endpoint with redirect_uri
	loginURL := "https://www.oreilly.com/member/auth/login/"
	redirectUri := "https://api.oreilly.com%2Fhome%2F" // URL-encoded /home/
	loginPayload := fmt.Sprintf(`{"email":%q,"password":%q,"redirect_uri":%q}`, email, password, redirectUri)
	headers := map[string]string{
		"Accept":       "application/json, text/javascript, */*; q=0.01",
		"Content-Type": "application/json",
		"User-Agent":   userAgent,
		"Referer":      unifiedLoginURL,
	}
	log.Printf("Posting JSON login to %s", loginURL)
	resp, err = s.client.PostWithHeaders(ctx, loginURL, headers, strings.NewReader(loginPayload))
	if err != nil {
		log.Printf("[Login JSON POST] Network error: %v", err)
		return nil, fmt.Errorf("login JSON POST failed: %w", err)
	}
	log.Printf("[Login JSON POST] Status: %v", resp.Status)
	for k, v := range resp.Header {
		log.Printf("[Login JSON POST] Header: %s=%v", k, v)
	}
	for i, c := range resp.Cookies() {
		log.Printf("[Login JSON POST] Cookie[%d]: %s=%s", i, c.Name, c.Value)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[Login JSON POST] Error reading body: %v", err)
		return nil, fmt.Errorf("failed to read JSON login response: %w", err)
	}
	log.Printf("[Login JSON POST] Status: %d, Body (first 350): %.350s", resp.StatusCode, string(body))
	if resp.StatusCode != 200 {
		if strings.Contains(string(body), "inactive") {
			return nil, fmt.Errorf("login failed: account is inactive")
		}
		if strings.Contains(string(body), "incorrect") || strings.Contains(string(body), "Invalid") {
			return nil, fmt.Errorf("login failed: invalid email or password")
		}
		return nil, fmt.Errorf("login failed: unexpected response status %d", resp.StatusCode)
	}

	// Parse JSON; expect {"access_token": "...", "token_type": "...", "redirect_uri": ...}
	var parsed struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		RedirectUri string `json:"redirect_uri"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("could not parse login response: %w", err)
	}
	if parsed.AccessToken == "" || parsed.RedirectUri == "" {
		return nil, fmt.Errorf("login: missing token or redirect URI")
	}

	// Step 3: GET the redirect URI to finalize the session (sets cookies)
	finalizeURL := parsed.RedirectUri
	log.Printf("Following login redirect (finalize session): %s", finalizeURL)
	req2, err := http.NewRequestWithContext(ctx, "GET", finalizeURL, nil)
	if err != nil {
		log.Printf("[Finalize GET] Build error: %v", err)
		return nil, fmt.Errorf("failed to build finalize session GET: %w", err)
	}
	req2.Header.Set("User-Agent", userAgent)
	resp2, err := s.client.GetHTTPClient().Do(req2)
	if err != nil {
		log.Printf("[Finalize GET] Network error: %v", err)
		return nil, fmt.Errorf("GET finalize redirect failed: %w", err)
	}
	log.Printf("[Finalize GET] Status: %v", resp2.Status)
	for k, v := range resp2.Header {
		log.Printf("[Finalize GET] Header: %s=%v", k, v)
	}
	for i, c := range resp2.Cookies() {
		log.Printf("[Finalize GET] Cookie[%d]: %s=%s", i, c.Name, c.Value)
	}
	_ = resp2.Body.Close()

	// Step 4: Look for jwt token in cookies
	var jwtToken string
	for _, cookie := range s.client.GetCookies("https://learning.oreilly.com") {
		log.Printf("Final Check Cookie: %s=%s", cookie.Name, cookie.Value)
		if cookie.Name == "orm-jwt" {
			jwtToken = cookie.Value
			break
		}
	}
	if jwtToken == "" {
		// As fallback, return access_token (valid for API use, not browser flows)
		jwtToken = parsed.AccessToken
	}
	if jwtToken == "" {
		return nil, fmt.Errorf("login failed: no valid token found after login flow")
	}
	return &LoginResponse{
		AccessToken: jwtToken,
		TokenType:   parsed.TokenType,
		ExpiresIn:   3600,
	}, nil
}

// verifyLogin verifies that the login was successful by accessing the profile page
func (s *Service) verifyLogin(ctx context.Context) error {
	// Try to access the profile page
	profileURL := "https://www.oreilly.com/member/profile/"
	headers := map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
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
