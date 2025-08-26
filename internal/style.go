package internal

import "github.com/charmbracelet/lipgloss"

var (
	bold       = lipgloss.NewStyle().Bold(true)
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)
