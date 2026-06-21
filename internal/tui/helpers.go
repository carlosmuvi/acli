package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// matches reports whether a key message triggers a binding.
func matches(msg tea.KeyMsg, b key.Binding) bool {
	return key.Matches(msg, b)
}

// truncate shortens s to at most w columns, appending an ellipsis. It is
// rune-based and assumes mostly single-width text (true for logcat output).
func truncate(s string, w int) string {
	if w <= 1 {
		return s
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

// footerHelp renders a dim key-hint line.
func footerHelp(items ...string) string {
	return footerStyle.Render(strings.Join(items, "   "))
}
