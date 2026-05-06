package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keyboard bindings for the Harvey TUI.
// Keys adapt based on context (e.g., y/n only work in grant dialog).
type KeyMap struct {
	Send          key.Binding
	Quit          key.Binding
	ScrollUp      key.Binding
	ScrollDown    key.Binding
	Yes          key.Binding
	No           key.Binding
	TogglePreview key.Binding
}

// DefaultKeyMap returns the standard key bindings with vim-friendly alternatives.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "send message"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc", "ctrl+c", "q"),
			key.WithHelp("Esc/Ctrl+C/q", "quit"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("up", "ctrl+p", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down", "ctrl+n", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		Yes: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "allow"),
		),
		No: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("n", "deny"),
		),
		TogglePreview: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "toggle preview"),
		),
	}
}
