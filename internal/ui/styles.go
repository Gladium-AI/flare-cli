package ui

import "github.com/charmbracelet/lipgloss"

// Style definitions for terminal output.
var (
	Bold    = lipgloss.NewStyle().Bold(true)
	Dim     = lipgloss.NewStyle().Faint(true)
	Success = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	Error   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	Info    = lipgloss.NewStyle().Foreground(lipgloss.Color("4")) // blue
	Label   = lipgloss.NewStyle().Bold(true).Width(30)
)
