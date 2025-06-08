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
	input textinput.Model
	spinner spinner.Model

	// States
	err error
}

// NewApp creates a new TUI application
func NewApp(cfg *config.Config, authSvc *auth.Service) (*App, error) {
	a := &App{
		cfg:     cfg,
		authSvc: authSvc,
		current: "auth",
	}

	// Initialize input
	a.input = textinput.New()
	a.input.Placeholder = "Enter your O'Reilly API key"
	a.input.Focus()
	a.input.CharLimit = 100
	a.input.Width = 50

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
				return a.handleAuth()
			}
		}
	}

	// Update the current model
	var cmd tea.Cmd
	switch a.current {
	case "auth":
		a.input, cmd = a.input.Update(msg)
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
	return fmt.Sprintf(
		`%s

Enter your O'Reilly API key:

%s

%s`,
		"Welcome to KOReilly!",
		a.input.View(),
		"(Press Enter to continue, q to quit)",
	)
}

// handleAuth handles the authentication flow
func (a *App) handleAuth() (tea.Model, tea.Cmd) {
	apiKey := a.input.Value()
	if apiKey == "" {
		a.err = fmt.Errorf("API key cannot be empty")
		return a, nil
	}

	_, err := a.authSvc.Authenticate(context.Background(), apiKey)
	if err != nil {
		a.err = err
		return a, nil
	}

	a.current = "main"
	a.input.Reset()
	return a, nil
}

// keys defines the key bindings for the application
var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "submit"),
	),
}

type keyMap struct {
	Quit  key.Binding
	Enter key.Binding
}
