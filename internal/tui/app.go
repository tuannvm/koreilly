package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/tuannvm/goreilly/internal/auth"
	"github.com/tuannvm/goreilly/internal/services/oreilly"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BookItem wraps a single search result and implements list.Item.
type BookItem struct {
	TitleText string
}

func (b BookItem) Title() string {
	return b.TitleText
}
func (b BookItem) Description() string { return "" }
func (b BookItem) FilterValue() string { return b.TitleText }

// searchResultMsg carries items or an error from the async search.
type searchResultMsg struct {
	items []BookItem
	err   error
}

// downloadRequestMsg is sent when the user selects a book to download.
type downloadRequestMsg struct {
	Slug  string
	Title string
}

// App is the interactive search TUI model.
type App struct {
	authSvc     *auth.Service
	searchInput textinput.Model
	spinner     spinner.Model
	books       list.Model
	inList      bool
	err         error
}

// NewApp constructs the App, setting up inputs and list.
func NewApp(authSvc *auth.Service) *App {
	a := &App{authSvc: authSvc}

	// Search box
	a.searchInput = textinput.New()
	a.searchInput.Placeholder = "Enter search query"
	a.searchInput.Prompt = "Search: "
	a.searchInput.Width = 40
	a.searchInput.CharLimit = 100
	a.searchInput.Focus()

	// Spinner
	a.spinner = spinner.New()
	a.spinner.Spinner = spinner.Dot

	// Interactive list (use BookDelegate for always vertical scrolling)
	delegate := BookDelegate{}
	a.books = list.New(nil, delegate, 60, 10)
	a.books.Title = "Results (↑/↓, Enter: select, Esc: search, q: quit)"
	a.books.SetFilteringEnabled(false)

	return a
}

// Init runs any startup commands.
func (a *App) Init() tea.Cmd {
	return tea.Batch(a.spinner.Tick, textinput.Blink)
}

// Add a Run method for compatibility with app.Run()
func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Update handles all messages: key events and search results.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle keyboard
	case tea.KeyMsg:
		// Quit anytime
		if key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))) {
			return a, tea.Quit
		}

		// If list view is active, route keys to list
		if a.inList {
			booksModel, cmd := a.books.Update(msg)
			a.books = booksModel

			// Enter to select a book
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				if it, ok := a.books.SelectedItem().(BookItem); ok {
					return a, func() tea.Msg {
						return downloadRequestMsg{Title: it.TitleText}
					}
				}
			}
			// Esc returns to search input
			if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
				a.inList = false
				a.searchInput.Focus()
				return a, nil
			}

			return a, cmd
		}

		// Otherwise handle search input
		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			// Kick off async search
			return a, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()

				tok, err := a.authSvc.GetToken()
				if err != nil {
					return searchResultMsg{nil, err}
				}
				svc, _ := oreilly.NewService()
				res, err := svc.SearchBooks(ctx, tok.AccessToken, a.searchInput.Value(), 10)
				if err != nil {
					return searchResultMsg{nil, err}
				}

				items := make([]BookItem, len(res.Results))
				for i, r := range res.Results {
					items[i] = BookItem{
						TitleText: r.Title,
					}
				}
				return searchResultMsg{items, nil}
			}
		}

		// Any other key updates the search input
		var cmd tea.Cmd
		a.searchInput, cmd = a.searchInput.Update(msg)
		return a, cmd

	// Search results came back
	case searchResultMsg:
		a.err = msg.err
		if msg.err != nil {
			a.inList = false
		} else {
			// Populate and show list
			listItems := make([]list.Item, len(msg.items))
			for i, it := range msg.items {
				listItems[i] = it
			}
			a.books.SetItems(listItems)
			a.inList = true
		}
		return a, nil

	// Download requested (stub)
	case downloadRequestMsg:
		a.err = fmt.Errorf("Download requested: %s", msg.Title)
		return a, nil
	}

	return a, nil
}

// View renders the TUI based on current state.
func (a *App) View() string {
	header := lipgloss.NewStyle().Bold(true).
		Render("Search O’Reilly (q: quit)")
	input := a.searchInput.View()
	if a.err != nil {
		input += "\n\n" +
			lipgloss.NewStyle().Foreground(lipgloss.Color("196")).
				Render(a.err.Error())
	}

	if a.inList {
		return fmt.Sprintf("%s\n\n%s\n\n%s",
			header, input, a.books.View())
	}

	placeholder := lipgloss.NewStyle().Faint(true).
		Render("Type a query and press Enter…")
	return fmt.Sprintf("%s\n\n%s\n\n%s",
		header, input, placeholder)
}
