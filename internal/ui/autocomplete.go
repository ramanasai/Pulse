package ui

import (
	"database/sql"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ramanasai/pulse/internal/db"
)

// AutocompleteModel represents a text input with autocomplete functionality
type AutocompleteModel struct {
	input        textinput.Model
	suggestions  []string
	showing      bool
	selected     int
	db           *sql.DB
	source       AutocompleteSource
	style        lipgloss.Style
	maxSuggestions int
}

// AutocompleteSource defines where suggestions come from
type AutocompleteSource int

const (
	SourceProjects AutocompleteSource = iota
	SourceTags
	SourceCategories
	SourceBoth // Combined projects and tags
)

// AutocompleteMsg is a message to update suggestions
type AutocompleteMsg struct {
	Suggestions []string
}

// NewAutocomplete creates a new autocomplete input model
func NewAutocomplete(db *sql.DB, source AutocompleteSource, maxSuggestions int) AutocompleteModel {
	input := textinput.New()
	input.Placeholder = getPlaceholder(source)

	return AutocompleteModel{
		input:          input,
		db:             db,
		source:         source,
		maxSuggestions: maxSuggestions,
		style:          lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}
}

func getPlaceholder(source AutocompleteSource) string {
	switch source {
	case SourceProjects:
		return "Project name..."
	case SourceTags:
		return "Tags..."
	case SourceCategories:
		return "Category..."
	case SourceBoth:
		return "Project or tags..."
	default:
		return "Type..."
	}
}

// Update handles the autocomplete logic
func (m AutocompleteModel) Update(msg tea.Msg) (AutocompleteModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			if m.showing && len(m.suggestions) > 0 {
				// Select next suggestion
				m.selected = (m.selected + 1) % len(m.suggestions)
				return m, nil
			}
		case tea.KeyShiftTab:
			if m.showing && len(m.suggestions) > 0 {
				// Select previous suggestion
				m.selected = (m.selected - 1 + len(m.suggestions)) % len(m.suggestions)
				return m, nil
			}
		case tea.KeyEnter, tea.KeySpace:
			if m.showing && len(m.suggestions) > 0 {
				// Accept selected suggestion
				m.input.SetValue(m.suggestions[m.selected])
				m.showing = false
				m.selected = 0
				return m, nil
			}
		case tea.KeyEscape:
			if m.showing {
				m.showing = false
				m.selected = 0
				return m, nil
			}
		case tea.KeyRunes:
			// When typing, update suggestions
			oldValue := m.input.Value()
			m.input, cmd = m.input.Update(msg)

			if m.input.Value() != oldValue {
				return m, m.fetchSuggestions()
			}
			return m, cmd
		case tea.KeyBackspace:
			// When deleting, update suggestions
			oldValue := m.input.Value()
			m.input, cmd = m.input.Update(msg)

			if m.input.Value() != oldValue {
				return m, m.fetchSuggestions()
			}
			return m, cmd
		default:
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

	case AutocompleteMsg:
		m.suggestions = msg.Suggestions
		if len(m.suggestions) > 0 && m.input.Value() != "" {
			m.showing = true
			m.selected = 0
		} else {
			m.showing = false
			m.selected = 0
		}
		return m, nil

	default:
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, cmd
}

// fetchSuggestions retrieves suggestions based on current input
func (m AutocompleteModel) fetchSuggestions() tea.Cmd {
	return func() tea.Msg {
		if m.db == nil || m.input.Value() == "" {
			return AutocompleteMsg{Suggestions: []string{}}
		}

		var suggestions []string
		var err error

		query := strings.ToLower(m.input.Value())

		switch m.source {
		case SourceProjects:
			suggestions, err = db.SearchProjects(m.db, query, m.maxSuggestions)
		case SourceTags:
			suggestions, err = db.SearchTags(m.db, query, m.maxSuggestions)
		case SourceCategories:
			suggestions, err = db.SearchCategories(m.db, query, m.maxSuggestions)
		case SourceBoth:
			// Get both projects and tags
			projects, _ := db.SearchProjects(m.db, query, m.maxSuggestions/2)
			tags, _ := db.SearchTags(m.db, query, m.maxSuggestions/2)
			suggestions = append(projects, tags...)
		}

		if err != nil {
			return AutocompleteMsg{Suggestions: []string{}}
		}

		return AutocompleteMsg{Suggestions: suggestions}
	}
}

// View renders the autocomplete input and suggestions
func (m AutocompleteModel) View() string {
	var content strings.Builder

	// Render the input field
	content.WriteString(m.input.View())

	// Render suggestions if showing
	if m.showing && len(m.suggestions) > 0 {
		content.WriteString("\n")
		for i, suggestion := range m.suggestions {
			if i >= m.maxSuggestions {
				break
			}

			prefix := "  "
			if i == m.selected {
				prefix = "▶ "
				content.WriteString(m.style.Copy().Foreground(lipgloss.Color("12")).Render(prefix + suggestion))
			} else {
				content.WriteString(m.style.Render(prefix + suggestion))
			}
			content.WriteString("\n")
		}
	}

	return content.String()
}

// Value returns the current input value
func (m AutocompleteModel) Value() string {
	return m.input.Value()
}

// SetValue sets the input value
func (m AutocompleteModel) SetValue(value string) {
	m.input.SetValue(value)
}

// Focus focuses the input
func (m AutocompleteModel) Focus() {
	m.input.Focus()
	m.showing = false
	m.selected = 0
}

// Blur unfocuses the input
func (m AutocompleteModel) Blur() {
	m.input.Blur()
	m.showing = false
	m.selected = 0
}

// Focused returns whether the input is focused
func (m AutocompleteModel) Focused() bool {
	return m.input.Focused()
}

// Width sets the width of the input
func (m AutocompleteModel) SetWidth(width int) {
	m.input.Width = width
}

// Placeholder sets the placeholder text
func (m AutocompleteModel) SetPlaceholder(placeholder string) {
	m.input.Placeholder = placeholder
}

// Suggestions returns the current suggestions
func (m AutocompleteModel) Suggestions() []string {
	return m.suggestions
}

// Showing returns whether suggestions are currently displayed
func (m AutocompleteModel) Showing() bool {
	return m.showing
}