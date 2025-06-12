package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BookDelegate renders one list item per vertical row, using Title() (Title - Author).
type BookDelegate struct{}

func (d BookDelegate) Height() int                               { return 1 }
func (d BookDelegate) Spacing() int                              { return 0 }
func (d BookDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d BookDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	book, ok := item.(BookItem)
	if !ok {
		return
	}
	selected := index == m.Index()
	style := lipgloss.NewStyle()
	if selected {
		style = style.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("62")).Bold(true)
	} else {
		style = style.Foreground(lipgloss.Color("12"))
	}
	fmt.Fprint(w, style.Render(book.TitleText))
}
