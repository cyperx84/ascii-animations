// Package theme provides Dracula color palette and lipgloss styles for the TUI.
package theme

import "github.com/charmbracelet/lipgloss"

// Dracula palette colors.
var (
	BgColor      = lipgloss.Color("#282a36")
	FgColor      = lipgloss.Color("#f8f8f2")
	CurrentLine  = lipgloss.Color("#44475a")
	CommentColor = lipgloss.Color("#6272a4")
	CyanColor    = lipgloss.Color("#8be9fd")
	GreenColor   = lipgloss.Color("#50fa7b")
	OrangeColor  = lipgloss.Color("#ffb86c")
	PinkColor    = lipgloss.Color("#ff79c6")
	PurpleColor  = lipgloss.Color("#bd93f9")
	RedColor     = lipgloss.Color("#ff5555")
	YellowColor  = lipgloss.Color("#f1fa8c")
)

// Shared styles used across the TUI.
var (
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(PurpleColor).
		MarginBottom(1)

	MenuItem = lipgloss.NewStyle().
			Foreground(FgColor).
			PaddingLeft(2)

	SelectedItem = lipgloss.NewStyle().
			Foreground(GreenColor).
			Bold(true).
			PaddingLeft(1)

	Header = lipgloss.NewStyle().
		Foreground(CyanColor).
		Bold(true)

	Footer = lipgloss.NewStyle().
		Foreground(CommentColor).
		MarginTop(1)

	Help = lipgloss.NewStyle().
		Foreground(CommentColor)

	AnimName = lipgloss.NewStyle().
			Foreground(PinkColor).
			Bold(true)

	AnimDesc = lipgloss.NewStyle().
			Foreground(OrangeColor)

	Lib = lipgloss.NewStyle().
		Foreground(YellowColor)

	Source = lipgloss.NewStyle().
		Foreground(CommentColor).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(CurrentLine).
		Padding(0, 1)

	ExportMsg = lipgloss.NewStyle().
			Foreground(GreenColor).
			Bold(true)

	Preview = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PurpleColor).
		Padding(1)

	InputStyle = lipgloss.NewStyle().
			Foreground(CyanColor).
			Bold(true)
)
