package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tuannvm/goreilly/internal/config"
	"github.com/tuannvm/goreilly/internal/services/oreilly"
)

type Service struct {
	config  *config.Config
	oreilly *oreilly.Service
}

// Token represents the O'Reilly authentication token
type Token struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn   int       `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// NewService creates a new authentication service
func NewService(cfg *config.Config) (*Service, error) {
	oreillySvc, err := oreilly.NewService()
	if err != nil {
		return nil, fmt.Errorf("failed to create O'Reilly service: %w", err)
	}

	return &Service{
		config:  cfg,
		oreilly: oreillySvc,
	}, nil
}

// Authenticate authenticates with O'Reilly API using username and password
func (s *Service) Authenticate(ctx context.Context, username, password string) (*Token, error) {
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	// Call the O'Reilly service to authenticate
	resp, err := s.oreilly.Login(ctx, username, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Create a new token with expiration
	token := &Token{
		AccessToken: resp.AccessToken,
		TokenType:   resp.TokenType,
		ExpiresIn:   resp.ExpiresIn,
		ExpiresAt:   time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
	}

	// Save the token
	if err := s.saveToken(token); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	// Update config with username (don't save password for security)
	s.config.Username = username
	if err := s.config.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
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
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "goreilly", "token.json"), nil
}

// Errors
func (s *Service) TokenFromCookieFile(cookiePath string) (*Token, error) {
	cookies, err := LoadCookieFile(cookiePath)
	if err != nil {
		return nil, fmt.Errorf("load cookie file: %w", err)
	}

	var jwt string
	var exp time.Time
	for _, c := range cookies {
		if c.Name == "orm-jwt" {
			jwt = c.Value
			exp = c.Expires
			break
		}
	}

	if jwt == "" {
		return nil, ErrInvalidToken
	}

	// If no expiry provided, assume one hour validity.
	if exp.IsZero() {
		exp = time.Now().Add(1 * time.Hour)
	}

	token := &Token{
		AccessToken: jwt,
		TokenType:   "Bearer",
		ExpiresIn:   int(time.Until(exp).Seconds()),
		ExpiresAt:   exp,
	}

	// Persist locally so the regular flow can reuse it.
	if err := s.saveToken(token); err != nil {
		return nil, err
	}

	return token, nil
}

// Errors
var (
	ErrNotAuthenticated   = NewAuthError("not authenticated")
	ErrInvalidToken       = NewAuthError("invalid token")
	ErrInvalidCredentials = NewAuthError("invalid username or password")
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
