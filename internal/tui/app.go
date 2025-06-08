package tui

import (
	"context"
	"fmt"

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
					a.passwordInput.Focus()
					return a, nil
				}
				return a.handleAuth()
			}
		case key.Matches(msg, keys.Tab):
			if a.current == "auth" && a.activeInput == "username" {
				a.activeInput = "password"
				a.passwordInput.Focus()
			} else if a.current == "auth" {
				a.activeInput = "username"
				a.usernameInput.Focus()
			}
			return a, nil
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
	var inputView string
	var helpText string

	if a.activeInput == "username" {
		helpText = "↑/↓: Navigate • Enter: Next • q: Quit"
		inputView = fmt.Sprintf(
			"%s\n\n%s",
			a.usernameInput.View() + " (email)",
			a.passwordInput.View() + " (password)",
		)
	} else {
		helpText = "↑/↓: Navigate • Enter: Login • Tab: Back • q: Quit"
		inputView = fmt.Sprintf(
			"%s\n\n%s",
			a.usernameInput.View() + " (email)",
			a.passwordInput.View() + " (password)",
		)
	}

	// Add error message if present
	if a.err != nil {
		helpText = fmt.Sprintf("Error: %s\n\n%s", a.err.Error(), helpText)
	}

	return fmt.Sprintf(
		"Welcome to Goreilly!\n\n" +
		"Please enter your O'Reilly credentials to continue.\n\n" +
		"%s\n\n%s\n",
		inputView,
		helpText,
	)
}

// handleAuth handles the authentication flow
func (a *App) handleAuth() (tea.Model, tea.Cmd) {
	username := a.usernameInput.Value()
	password := a.passwordInput.Value()

	// Validate inputs
	if username == "" {
		a.err = fmt.Errorf("email is required")
		a.activeInput = "username"
		a.usernameInput.Focus()
		return a, nil
	}

	if password == "" {
		a.err = fmt.Errorf("password is required")
		a.activeInput = "password"
		a.passwordInput.Focus()
		return a, nil
	}

	// Clear any previous errors
	a.err = nil

	// Show loading spinner
	a.current = "loading"

	// Start authentication in a goroutine
	return a, tea.Batch(
		a.spinner.Tick,
		func() tea.Msg {
			token, err := a.authSvc.Authenticate(context.Background(), username, password)
			if err != nil {
				// Clear password on error for security
				a.passwordInput.Reset()
				a.activeInput = "password"
				a.passwordInput.Focus()
				return authError{fmt.Errorf("authentication failed: %v", err)}
			}
			return token
		},
	)
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
