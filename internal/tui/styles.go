package tui

import (
	"github.com/carlosmuvi/acli/internal/android"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorAccent = lipgloss.Color("13")  // magenta
	colorMuted  = lipgloss.Color("241") // grey
	colorOK     = lipgloss.Color("10")  // green
	colorBad    = lipgloss.Color("9")   // red

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(colorAccent).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	okStyle     = lipgloss.NewStyle().Foreground(colorOK)
	badStyle    = lipgloss.NewStyle().Foreground(colorBad)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorMuted)

	stateRunningStyle = lipgloss.NewStyle().Foreground(colorOK)
	stateStoppedStyle = lipgloss.NewStyle().Foreground(colorMuted)
)

// levelStyle colors a log line by severity.
func levelStyle(l android.Level) lipgloss.Style {
	switch l {
	case android.LevelError, android.LevelFatal:
		return lipgloss.NewStyle().Foreground(colorBad)
	case android.LevelWarn:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	case android.LevelInfo:
		return lipgloss.NewStyle().Foreground(colorOK)
	case android.LevelDebug:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // blue
	default:
		return mutedStyle
	}
}
