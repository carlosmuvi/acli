package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap holds every binding acli uses. Help is rendered from these.
type keyMap struct {
	// global
	Quit   key.Binding
	Help   key.Binding
	Doctor key.Binding

	// emulator list
	Launch   key.Binding
	ColdBoot key.Binding
	Wipe     key.Binding
	Kill     key.Binding
	Logs     key.Binding
	Shot     key.Binding
	Refresh  key.Binding

	// logcat
	Filter   key.Binding
	Level    key.Binding
	PauseLog key.Binding
	ClearLog key.Binding
	Back     key.Binding
}

var keys = keyMap{
	Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Doctor: key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "doctor")),

	Launch:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "launch")),
	ColdBoot: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "cold boot")),
	Wipe:     key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "wipe+launch")),
	Kill:     key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "kill")),
	Logs:     key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "logcat")),
	Shot:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "screenshot")),
	Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),

	Filter:   key.NewBinding(key.WithKeys("f", "/"), key.WithHelp("f", "filter")),
	Level:    key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "min level")),
	PauseLog: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "pause")),
	ClearLog: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "clear")),
	Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
}
