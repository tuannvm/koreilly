package tui

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tuannvm/goreilly/internal/auth"
	"github.com/tuannvm/goreilly/internal/config"
)

// App represents the main TUI application
type App struct {
	cfg      *config.Config
	authSvc  *auth.Service
	current  string
	quitting bool

	// Sub-models
	usernameInput textinput.Model
	passwordInput textinput.Model
	spinner       spinner.Model

	// States
	err         error
	activeInput string // 'username' or 'password'
	isLoading   bool
	message     string
}

// NewApp creates a new TUI application
func NewApp(cfg *config.Config, authSvc *auth.Service) (*App, error) {
	a := &App{
		cfg:         cfg,
		authSvc:     authSvc,
		current:     "auth",
		activeInput: "username",
	}

	// Initialize username input
	a.usernameInput = textinput.New()
	a.usernameInput.Placeholder = "Legacy O'Reilly email (non-SSO)"
	a.usernameInput.Focus()
	a.usernameInput.CharLimit = 100
	a.usernameInput.Width = 50
	a.usernameInput.Prompt = "Email: "

	// Initialize password input
	a.passwordInput = textinput.New()
	a.passwordInput.Placeholder = "Legacy password"
	a.passwordInput.CharLimit = 100
	a.passwordInput.Width = 50
	a.passwordInput.EchoMode = textinput.EchoPassword
	a.passwordInput.EchoCharacter = '•'
	a.passwordInput.Prompt = "Password: "

	// Initialize spinner
	a.spinner = spinner.New()
	a.spinner.Spinner = spinner.Dot
	a.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return a, nil
}

// Run starts the TUI application
func (a *App) Run(ctx context.Context) error {
	p := tea.NewProgram(a, tea.WithAltScreen())

	// Start the spinner
	a.spinner, _ = a.spinner.Update(spinner.Tick())

	// Run the program
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to start TUI: %w", err)
	}

	return nil
}

// Init initializes the TUI
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.spinner.Tick,
		textinput.Blink,
	)
}

// Update handles updates
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return a, tea.Quit
		case key.Matches(msg, keys.Enter):
			if a.current == "auth" {
				if a.activeInput == "username" {
					a.activeInput = "password"
					return a, nil
				}
				return a.handleAuth()
			}
		case key.Matches(msg, keys.Tab):
			if a.current == "auth" {
				if a.activeInput == "username" {
					a.activeInput = "password"
				} else {
					a.activeInput = "username"
				}
				return a, nil
			}
		}

	// Handle authentication response
	case authError:
		a.err = msg.err
		a.current = "auth"
		return a, nil

	case *auth.Token:
		// Successful authentication
		a.current = "main"
		a.err = nil
		return a, tea.Quit // For now, just quit on successful auth
	}

	// Update the current input
	var cmd tea.Cmd
	if a.current == "auth" {
		if a.activeInput == "username" {
			a.usernameInput, cmd = a.usernameInput.Update(msg)
		} else {
			a.passwordInput, cmd = a.passwordInput.Update(msg)
		}
	}

	return a, cmd
}

// View renders the TUI
func (a *App) View() string {
	if a.quitting {
		return "Goodbye!\n"
	}

	var s string

	switch a.current {
	case "loading":
		return fmt.Sprintf("\n   %s Authenticating... (press q to quit)\n", a.spinner.View())
	case "auth":
		s = a.authView()
	default:
		s = "Loading...\n"
	}

	if a.err != nil {
		s += "\nError: " + a.err.Error() + "\n"
	}

	s += "\nPress q to quit.\n"

	return s
}

// sanitizeError removes HTML and other unwanted characters from error messages
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	clean := re.ReplaceAllString(err.Error(), "")
	// Replace multiple spaces/newlines with a single space
	re = regexp.MustCompile(`\s+`)
	clean = re.ReplaceAllString(clean, " ")
	// Trim spaces
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return "An unknown error occurred"
	}
	return clean
}

// authView renders the authentication view
func (a *App) authView() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	header := headerStyle.Render("Welcome to Goreilly!")

	instructions := `
Google Chrome (SSO flow):

  1. Visit https://learning.oreilly.com and sign in with your organisation’s SSO.
  2. Install the free “EditThisCookie” extension from the Chrome Web Store
     (or any tool that can export cookies).
  3. Click the 🍪 icon → Export → select “Netscape” format.
     This saves all cookies for learning.oreilly.com, including the “orm-jwt”.
  4. Move the exported file somewhere convenient, e.g. ~/Downloads/oreilly_cookies.txt
  5. In your terminal run:
        goreilly cookie import ~/Downloads/oreilly_cookies.txt
  6. Restart Goreilly and you’re ready to search & download.

(Users with legacy non-SSO accounts may still log in with email/password using the old flow.)`

	return fmt.Sprintf("%s\n\n%s\n", header, strings.TrimSpace(instructions))
}

// handleAuth handles the authentication flow
func (a *App) handleAuth() (tea.Model, tea.Cmd) {
	// Get credentials from inputs
	username := strings.TrimSpace(a.usernameInput.Value())
	password := strings.TrimSpace(a.passwordInput.Value())

	// Validate inputs
	if username == "" {
		a.setError("Email is required")
		a.usernameInput.Focus()
		return a, nil
	}
	if password == "" {
		a.setError("Password is required")
		a.passwordInput.Focus()
		return a, nil
	}

	// Clear any previous errors and messages
	a.clearError()
	a.message = ""
	a.isLoading = true

	// Start authentication in a goroutine
	return a, tea.Batch(
		a.spinner.Tick,
		func() tea.Msg {
			// This will block, so we run it in a goroutine
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Show a message that we're attempting to log in
			a.message = "Attempting to log in..."

			// Call the auth service
			token, err := a.authSvc.Authenticate(ctx, username, password)
			if err != nil {
				// Check for common error types and provide user-friendly messages
				errMsg := err.Error()
				if strings.Contains(errMsg, "invalid email or password") {
					errMsg = "Invalid email or password. Please try again."
				} else if strings.Contains(errMsg, "too many failed attempts") {
					errMsg = "Too many failed login attempts. Please try again later."
				} else if strings.Contains(errMsg, "account is inactive") {
					errMsg = "This account is inactive. Please contact support."
				}
				return authError{errors.New(errMsg)}
			}

			// Show success message
			a.message = fmt.Sprintf("Login successful! Welcome, %s", username)

			// Here you would typically transition to the main app view
			// For now, we'll just show a success message
			time.Sleep(2 * time.Second) // Show success message briefly

			return token // Return the token on success
		},
	)
}

// Helper function to set an error message
func (a *App) setError(msg string) {
	a.err = fmt.Errorf("%s", msg)
	a.isLoading = false
}

// Helper function to clear any error
func (a *App) clearError() {
	a.err = nil
}

// keys defines the key bindings for the application
var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch field"),
	),
}

type keyMap struct {
	Quit  key.Binding
	Enter key.Binding
	Tab   key.Binding
}

// authError wraps an authentication error for the TUI
type authError struct {
	err error
}

// Error returns the error message
func (a authError) Error() string {
	return a.err.Error()
}
