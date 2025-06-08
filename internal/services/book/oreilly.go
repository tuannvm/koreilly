package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tuannvm/goreilly/internal/client"
)

const (
	defaultBaseURL = "https://learning.oreilly.com/api/v2"
	loginEndpoint  = "/auth/login/"
)

// OReillyService handles authentication with O'Reilly's API
type OReillyService struct {
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
	Email    string `json:"email"`
	Password string `json:"password"`
}

// NewOReillyService creates a new O'Reilly API service
func NewOReillyService() *OReillyService {
	return &OReillyService{
		client: client.New(defaultBaseURL),
	}
}

// Login authenticates with O'Reilly's API using email and password
func (s *OReillyService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	loginReq := LoginRequest{
		Email:    email,
		Password: password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling login request: %w", err)
	}

	// Set a reasonable timeout for the login request
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Make the login request
	reqBody := bytes.NewReader(jsonData)
	resp, err := s.client.Post(ctx, loginEndpoint, "application/json", reqBody)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed with status: %s", resp.Status)
	}

	// Parse the response
	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("error decoding login response: %w", err)
	}

	if loginResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	return &loginResp, nil
}

// ValidateToken checks if the current token is still valid
func (s *OReillyService) ValidateToken(ctx context.Context, token string) (bool, error) {
	// TODO: Implement token validation
	// This would typically make a request to a protected endpoint
	// and check if the token is still valid
	return true, nil
}
