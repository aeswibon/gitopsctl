package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#04B575")
	errorColor     = lipgloss.Color("#FF4C4C")
	warningColor   = lipgloss.Color("#F4C430")
	inactiveColor  = lipgloss.Color("#626262")
	accentColor    = lipgloss.Color("#EE6FF8")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(primaryColor).
			Padding(0, 1).
			MarginBottom(1)

	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(inactiveColor).
			Padding(0, 2).
			MarginRight(2)

	MainContentStyle = lipgloss.NewStyle().
				Padding(0, 1)

	DetailPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(primaryColor).
			Padding(1).
			MarginTop(1)

	StatusSyncedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(secondaryColor)

	StatusErrorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(errorColor)

	StatusSyncingStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(warningColor)

	StatusPendingStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(inactiveColor)

	BadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Padding(0, 1).
			MarginRight(1)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	KeyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(inactiveColor).
			MarginTop(1)
)
