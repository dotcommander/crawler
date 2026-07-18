package ui

import (
	"charm.land/lipgloss/v2"
)

var (
	// Color palette - cyberpunk/sci-fi theme
	primaryColor   = lipgloss.Color("#00ff41") // Matrix green
	secondaryColor = lipgloss.Color("#39ff14") // Neon green
	accentColor    = lipgloss.Color("#ff006e") // Hot pink
	warningColor   = lipgloss.Color("#ffb700") // Amber
	errorColor     = lipgloss.Color("#ff0040") // Red
	dimColor       = lipgloss.Color("#606060") // Dim gray

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(1, 2).
			MarginBottom(1).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(secondaryColor)

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			MarginBottom(1)

	// Progress bar styles
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Background(lipgloss.Color("#1a1a1a"))

	// Status styles
	StatusStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	// URL styles
	URLStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Underline(true)

	// Stat styles
	StatLabelStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Width(20)

	StatValueStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Width(15).
			Align(lipgloss.Right)

	// Animation styles
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	// Log styles
	LogStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			PaddingLeft(2)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true)
)

// Spinner frames for animation
var SpinnerFrames = []string{
	"⠋",
	"⠙",
	"⠹",
	"⠸",
	"⠼",
	"⠴",
	"⠦",
	"⠧",
	"⠇",
	"⠏",
}

// Alternative sci-fi spinner
var SciFiSpinner = []string{
	"◜",
	"◠",
	"◝",
	"◞",
	"◡",
	"◟",
}

// Progress indicators
var ProgressIndicators = []string{
	"▱▱▱▱▱▱▱▱▱▱",
	"▰▱▱▱▱▱▱▱▱▱",
	"▰▰▱▱▱▱▱▱▱▱",
	"▰▰▰▱▱▱▱▱▱▱",
	"▰▰▰▰▱▱▱▱▱▱",
	"▰▰▰▰▰▱▱▱▱▱",
	"▰▰▰▰▰▰▱▱▱▱",
	"▰▰▰▰▰▰▰▱▱▱",
	"▰▰▰▰▰▰▰▰▱▱",
	"▰▰▰▰▰▰▰▰▰▱",
	"▰▰▰▰▰▰▰▰▰▰",
}
