package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Title   lipgloss.Style
	Label   lipgloss.Style
	Value   lipgloss.Style
	Border  lipgloss.Style
	Hint    lipgloss.Style
	Error   lipgloss.Style
	Success lipgloss.Style
}

var DefaultTheme = Theme{
	Title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1")),
	Label:   lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#89B4FA")),
	Value:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F2CDCD")),
	Border:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1),
	Hint:    lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#CBA6F7")),
	Error:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F38BA8")),
	Success: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1")),
}
