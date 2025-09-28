package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	vp      viewport.Model
	entries []string
}

func initialModel(entries []string) model {
	m := model{entries: entries}
	m.vp = viewport.New(0, 0)
	m.vp.SetContent(strings.Join(entries, "\n\n"))
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.vp.Width = msg.Width
		m.vp.Height = msg.Height - 2
	default:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	header := DefaultTheme.Title.Render("Pulse – Recent Entries")
	hint := DefaultTheme.Hint.Render("↑/↓ scroll • q to quit")
	body := DefaultTheme.Border.Render(m.vp.View())
	return header + "\n" + body + "\n" + hint
}

func Run(entries []string) error {
	_, err := tea.NewProgram(initialModel(entries), tea.WithAltScreen()).Run()
	return err
}
