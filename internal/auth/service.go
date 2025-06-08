package auth

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tuannvm/goreilly/internal/config"
)

type Service struct {
	config *config.Config
}

// Token represents the authentication token
// This is a placeholder - update with actual O'Reilly token structure
// based on their API documentation
//
// Example:
// type Token struct {
// 	AccessToken string `json:"access_token"`
// 	TokenType   string `json:"token_type"`
// 	ExpiresIn   int    `json:"expires_in"`
// }
type Token struct {
	APIKey string `json:"api_key"`
}

// NewService creates a new authentication service
func NewService(cfg *config.Config) (*Service, error) {
	return &Service{
		config: cfg,
	}, nil
}

// Authenticate authenticates with O'Reilly API using the provided API key
func (s *Service) Authenticate(ctx context.Context, apiKey string) (*Token, error) {
	// TODO: Implement actual O'Reilly API authentication
	// This is a placeholder that just validates the key is not empty
	if apiKey == "" {
		return nil, ErrInvalidToken
	}

	token := &Token{
		APIKey: apiKey,
	}

	// Save the token
	if err := s.saveToken(token); err != nil {
		return nil, err
	}

	// Update config
	s.config.APIKey = apiKey
	if err := s.config.Save(); err != nil {
		return nil, err
	}

	return token, nil
}

// GetToken returns the current authentication token
func (s *Service) GetToken() (*Token, error) {
	tokenPath, err := s.tokenPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotAuthenticated
		}
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// IsAuthenticated checks if the user is authenticated
func (s *Service) IsAuthenticated() bool {
	_, err := s.GetToken()
	return err == nil
}

// Logout removes the authentication token
func (s *Service) Logout() error {
	tokenPath, err := s.tokenPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(tokenPath); err == nil {
		return os.Remove(tokenPath)
	}

	return nil
}

func (s *Service) saveToken(token *Token) error {
	tokenPath, err := s.tokenPath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(token)
	if err != nil {
		return err
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		return err
	}

	return os.WriteFile(tokenPath, data, 0600)
}

func (s *Service) tokenPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "koreilly", "token.json"), nil
}

// Errors
var (
	ErrNotAuthenticated = NewAuthError("not authenticated")
	ErrInvalidToken    = NewAuthError("invalid token")
)

type AuthError struct {
	message string
}

func NewAuthError(message string) *AuthError {
	return &AuthError{message: message}
}

func (e *AuthError) Error() string {
	return e.message
}
