package oreilly

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/tuannvm/goreilly/internal/client"
)

const (
	// O'Reilly's web login page
	defaultBaseURL = "https://www.oreilly.com"
	loginPage     = "/member/auth/login/"
	loginAPI      = "/member/auth/login/"
)

// Service handles authentication with O'Reilly's API
type Service struct {
	client *client.Client
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
}

// NewService creates a new O'Reilly API service
func NewService() *Service {
	return &Service{
		client: client.New(defaultBaseURL),
	}
}

// getCSRFToken fetches the CSRF token from the login page
func (s *Service) getCSRFToken(ctx context.Context) (string, error) {
	// First, get the login page to get the CSRF token
	resp, err := s.client.Get(ctx, loginPage)
	if err != nil {
		return "", fmt.Errorf("failed to get login page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse the HTML to find the CSRF token
	token, err := extractCSRFToken(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to extract CSRF token: %w", err)
	}

	return token, nil
}

// extractCSRFToken extracts the CSRF token from the login page HTML
func extractCSRFToken(body io.Reader) (string, error) {
	// This is a simplified example - you might need to adjust the parsing
	// based on the actual HTML structure of O'Reilly's login page
	tokenRe := regexp.MustCompile(`name=['"]csrfmiddlewaretoken['"][^>]*value=['"]([^'"]+)['"]`)
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	matches := tokenRe.FindStringSubmatch(string(bodyBytes))
	if len(matches) < 2 {
		return "", fmt.Errorf("CSRF token not found in response")
	}

	return matches[1], nil
}

// Login authenticates with O'Reilly's website using email and password
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	// First, get the CSRF token
	csrfToken, err := s.getCSRFToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSRF token: %w", err)
	}

	// Prepare form data
	formData := url.Values{
		"csrfmiddlewaretoken": []string{csrfToken},
		"email":              []string{email},
		"password":           []string{password},
		"next":               []string{"/"}, // Redirect after login
	}

	// Set up the request
	reqBody := strings.NewReader(formData.Encode())
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Referer":     defaultBaseURL + loginPage,
		"Origin":      defaultBaseURL,
	}

	// Make the login request
	resp, err := s.client.PostWithHeaders(ctx, loginAPI, headers, reqBody)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful login (302 redirect on success)
	if resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Check for session cookie
	sessionCookie := ""
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "sessionid" || cookie.Name == "oreilly_media_remember_me" {
			sessionCookie = cookie.Value
			break
		}
	}

	if sessionCookie == "" {
		return nil, fmt.Errorf("no session cookie found in response")
	}

	// Return the session token
	return &LoginResponse{
		AccessToken: sessionCookie,
		TokenType:   "session",
	}, nil
}
