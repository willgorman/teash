package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	styleStatusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	styleHelp = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	styleCluster = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	stylePrompt = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))
)
