// Package tui implements the Bubbletea terminal user interface for G-MAN.
// It provides a split-pane chat interface with streaming LLM responses,
// file preview panel, and grant confirmation dialog.
//
// Architecture: the TUI depends on domain types and a ChatOrchestrator
// interface defined locally. It does NOT import from the application layer.
// The concrete *application.ChatOrchestrator is injected at wiring time (cmd/gman/main.go).
package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all Lipgloss styles used throughout the TUI.
// A default dark theme with blue accent is provided by DefaultStyles().
type Styles struct {
	// Base layout styles
	App     lipgloss.Style // full app background
	Chat    lipgloss.Style // chat panel (left side)
	Preview lipgloss.Style // file preview panel (right side)
	Input   lipgloss.Style // input bar at bottom
	Title   lipgloss.Style // top title bar

	// Message role styles
	UserMsg    lipgloss.Style // "You: ..." style
	GMANMsg  lipgloss.Style // "G-MAN: ..." style
	SystemMsg  lipgloss.Style // system messages
	ErrorMsg   lipgloss.Style // error messages

	// Component styles
	GrantDialog lipgloss.Style // grant confirmation modal overlay
	Spinner     lipgloss.Style // thinking indicator
	Divider     lipgloss.Style // vertical divider between panels

	// Grant dialog button styles
	GrantAllowBtn lipgloss.Style
	GrantDenyBtn  lipgloss.Style

	// Semantic colors
	Accent  lipgloss.Color // primary accent (blue)
	Success lipgloss.Color // green
	Error   lipgloss.Color // red
	Muted   lipgloss.Color // gray for timestamps, metadata
	Bg      lipgloss.Color // background
	Fg      lipgloss.Color // foreground text
}

// DefaultStyles returns a dark-themed style set with blue accent.
// Feels like a terminal assistant — unobtrusive but clear.
func DefaultStyles() Styles {
	accent := lipgloss.Color("39")   // bright blue
	success := lipgloss.Color("120") // green
	errColor := lipgloss.Color("196") // red
	muted := lipgloss.Color("241")   // gray
	bg := lipgloss.Color("0")        // black background
	fg := lipgloss.Color("252")      // light gray text

	s := Styles{
		Accent:  accent,
		Success: success,
		Error:   errColor,
		Muted:   muted,
		Bg:      bg,
		Fg:      fg,
	}

	// App: full terminal background
	s.App = lipgloss.NewStyle().
		Background(bg).
		Foreground(fg)

	// Title bar
	s.Title = lipgloss.NewStyle().
		Background(accent).
		Foreground(lipgloss.Color("15")). // white text on blue
		Bold(true).
		Padding(0, 1)

	// Chat panel
	s.Chat = lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Padding(0, 1)

	// Preview panel
	s.Preview = lipgloss.NewStyle().
		Background(lipgloss.Color("232")). // very dark gray
		Foreground(muted).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(muted).
		Padding(0, 1)

	// Input bar
	s.Input = lipgloss.NewStyle().
		Background(lipgloss.Color("236")). // dark gray
		Foreground(fg).
		Padding(0, 1)

	// Message styles
	s.UserMsg = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true)

	s.GMANMsg = lipgloss.NewStyle().
		Foreground(fg)

	s.SystemMsg = lipgloss.NewStyle().
		Foreground(muted).
		Italic(true)

	s.ErrorMsg = lipgloss.NewStyle().
		Foreground(errColor).
		Bold(true)

	// Spinner: thinking indicator (light blue, pulsing feel)
	s.Spinner = lipgloss.NewStyle().
		Foreground(accent)

	// Vertical divider between chat and preview
	s.Divider = lipgloss.NewStyle().
		Foreground(muted)

	// Grant dialog: centered modal overlay
	s.GrantDialog = lipgloss.NewStyle().
		Background(lipgloss.Color("234")). // dark background
		Foreground(fg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		Align(lipgloss.Center)

	// Grant dialog buttons
	s.GrantAllowBtn = lipgloss.NewStyle().
		Background(success).
		Foreground(lipgloss.Color("0")). // black text on green
		Bold(true).
		Padding(0, 2)

	s.GrantDenyBtn = lipgloss.NewStyle().
		Background(errColor).
		Foreground(lipgloss.Color("15")). // white text on red
		Bold(true).
		Padding(0, 2)

	return s
}
