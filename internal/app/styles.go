package app

import "github.com/charmbracelet/lipgloss"

var (
	accent = lipgloss.Color("39")
	muted  = lipgloss.Color("245")
	green  = lipgloss.Color("42")
	yellow = lipgloss.Color("226")
	red    = lipgloss.Color("203")
	bg     = lipgloss.Color("0")

	appStyle            = lipgloss.NewStyle().Padding(1, 2)
	titleStyle          = lipgloss.NewStyle().Bold(true).Foreground(accent)
	subtleStyle         = lipgloss.NewStyle().Foreground(muted)
	dangerStyle         = lipgloss.NewStyle().Foreground(red).Bold(true)
	selectedStyle       = lipgloss.NewStyle().Foreground(accent).Bold(true)
	dangerSelectedStyle = lipgloss.NewStyle().Foreground(yellow).Bold(true)
	goodStyle           = lipgloss.NewStyle().Foreground(green).Bold(true)
	warnStyle           = lipgloss.NewStyle().Foreground(yellow).Bold(true)
	errorStyle          = lipgloss.NewStyle().Foreground(red).Bold(true)
	boxStyle            = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(1, 2)
	panelStyle          = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(1, 2)
	sidebarStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(1, 2)
	badgeStyle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(accent).Padding(0, 1)
)
