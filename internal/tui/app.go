package tui

import (
	"context"
	"fmt"
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
	spinner      spinner.Model

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
	a.usernameInput.Placeholder = "Enter your O'Reilly email"
	a.usernameInput.Focus()
	a.usernameInput.CharLimit = 100
	a.usernameInput.Width = 50
	a.usernameInput.Prompt = "Email: "

	// Initialize password input
	a.passwordInput = textinput.New()
	a.passwordInput.Placeholder = "Enter your O'Reilly password"
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

// authView renders the authentication view
func (a *App) authView() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	header := headerStyle.Render("Welcome to Goreilly!")
	subHeader := lipgloss.NewStyle().MarginBottom(2).Render("Please enter your O'Reilly credentials to continue.")

	// Create a styled box for the login form
	formStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Margin(1, 0)

	// Style for input labels
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).MarginRight(2)

	// Only show cursor in the active input field
	usernameInput := a.usernameInput
	passwordInput := a.passwordInput

	if a.activeInput == "username" {
		usernameInput.Focus()
		passwordInput.Blur()
	} else {
		usernameInput.Blur()
		passwordInput.Focus()
	}

	// Render the username and password inputs
	inputs := []string{
		fmt.Sprintf("%s\n%s", 
			labelStyle.Render("Email"),
			usernameInput.View(),
		),
		"",
		fmt.Sprintf("%s\n%s",
			labelStyle.Render("Password"),
			passwordInput.View(),
		),
	}

	// Add loading spinner if authenticating
	if a.isLoading {
		inputs = append(inputs, "", fmt.Sprintf("  %s Authenticating...", a.spinner.View()))
	}

	// Add status message if any
	if a.message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).MarginTop(1)
		inputs = append(inputs, "", msgStyle.Render(a.message))
	}

	// Add help text
	helpText := lipgloss.NewStyle().Faint(true).Render("↑/↓: Navigate • Enter: Login • Tab: Switch Field • q: Quit")
	inputs = append(inputs, "", helpText)

	// Combine everything
	form := formStyle.Render(strings.Join(inputs, "\n"))

	// Show error if any
	if a.err != nil {
		errMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("Error: %v", a.err))
		form = fmt.Sprintf("%s\n\n%s", form, errMsg)
	}

	return fmt.Sprintf("%s\n%s\n\n%s", header, subHeader, form)
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
				return authError{fmt.Errorf("%s", errMsg)}
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
	a.err = fmt.Errorf(msg)
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
