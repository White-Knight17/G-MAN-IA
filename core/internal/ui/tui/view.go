package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the full TUI layout. It implements tea.Model.View.
//
// Three possible render modes:
//  1. Grant dialog: modal overlay asking for permission confirmation
//  2. Error state: error banner at the top
//  3. Normal mode: split pane — chat on left, file preview on right (optional)
func (m Model) View() string {
	// If grant dialog is open, show modal overlay
	if m.showGrantDialog {
		return m.renderGrantDialog()
	}

	// Build layout from top to bottom
	var sections []string

	// Title bar
	sections = append(sections, m.renderTitleBar())

	// Main content area: chat + optional preview
	sections = append(sections, m.renderMainPanel())

	// Error banner (if any)
	if m.err != nil {
		sections = append(sections, m.renderErrorBanner())
	}

	// Input bar
	sections = append(sections, m.renderInputBar())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderTitleBar renders the top bar with app name and session info.
func (m Model) renderTitleBar() string {
	title := " G-MAN — Arch Assistant "
	grantCount := m.grantCount()

	right := fmt.Sprintf(" Grants: %d ", grantCount)

	// Pad title to fill width
	availableWidth := m.width
	if availableWidth <= 0 {
		availableWidth = 80
	}

	padding := availableWidth - lipgloss.Width(title) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}

	bar := title + strings.Repeat(" ", padding) + right
	return m.styles.Title.Width(availableWidth).Render(bar)
}

// renderMainPanel renders the chat viewport and optional file preview.
func (m Model) renderMainPanel() string {
	chatContent := m.renderChatViewport()

	if m.showPreview && m.previewWidth > 0 {
		previewContent := m.renderPreviewPanel()
		divider := m.styles.Divider.Render("│")

		// Set exact widths
		chatCol := m.styles.Chat.Width(m.chatWidth).Render(chatContent)
		previewCol := m.styles.Preview.Width(m.previewWidth).Render(previewContent)

		return lipgloss.JoinHorizontal(lipgloss.Top, chatCol, divider, previewCol)
	}

	return m.styles.Chat.Width(m.chatWidth).Render(chatContent)
}

// renderChatViewport renders the chat message list with the viewport.
func (m Model) renderChatViewport() string {
	return m.viewport.View()
}

// renderPreviewPanel renders the file preview on the right side.
// Shows content of the last file G-MAN read or wrote, if available.
func (m Model) renderPreviewPanel() string {
	content := m.getPreviewContent()
	if content == "" {
		emptyMsg := m.styles.SystemMsg.Render("No file preview")
		return m.styles.Preview.Render(emptyMsg)
	}

	// Apply basic syntax highlighting (just the content)
	return m.styles.Preview.Render(content)
}

// getPreviewContent extracts file preview content from the last relevant tool result.
// Currently shows the last tool output as a preview.
func (m Model) getPreviewContent() string {
	// Scan messages in reverse to find the last tool result
	for i := len(m.messages) - 1; i >= 0; i-- {
		msg := m.messages[i]
		if msg.Role == "tool" {
			// Extract content from tool result XML
			return m.extractToolOutput(msg.Content)
		}
	}
	return ""
}

// extractToolOutput strips XML tags from a tool result to show raw content.
func (m Model) extractToolOutput(toolXML string) string {
	// Strip <tool_result> and <output> tags for preview
	s := toolXML
	s = strings.ReplaceAll(s, "<tool_result>", "")
	s = strings.ReplaceAll(s, "</tool_result>", "")
	s = strings.ReplaceAll(s, "<output>", "")
	s = strings.ReplaceAll(s, "</output>", "")
	s = strings.ReplaceAll(s, "<error>", "")
	s = strings.ReplaceAll(s, "</error>", "")
	s = strings.ReplaceAll(s, "<diff>", "")
	s = strings.ReplaceAll(s, "</diff>", "")
	s = strings.TrimSpace(s)

	// Truncate very long previews
	const maxPreview = 2000
	if len(s) > maxPreview {
		s = s[:maxPreview] + "\n... (truncated)"
	}

	return s
}

// renderGrantDialog renders the modal overlay for permission confirmation.
func (m Model) renderGrantDialog() string {
	// Build dialog content
	path := m.pendingGrant.Path
	mode := string(m.pendingGrant.Mode)
	modeDisplay := "read-only"
	if mode == "rw" {
		modeDisplay = "read & write"
	}

	dialogContent := fmt.Sprintf(
		"Allow G-MAN to access %s?\n\n"+
			"  Path: %s\n"+
			"  Mode: %s\n\n"+
			"  [Y] Allow    [N] Deny",
		modeDisplay, path, mode,
	)

	dialog := m.styles.GrantDialog.Render(dialogContent)

	// Center the dialog on screen
	availW := m.width
	availH := m.height
	if availW <= 0 {
		availW = 80
	}
	if availH <= 0 {
		availH = 24
	}

	// Calculate vertical padding to center
	dialogHeight := lipgloss.Height(dialog)
	topPad := (availH - dialogHeight) / 2
	if topPad < 0 {
		topPad = 0
	}

	// Render the overlay: background + centered dialog
	overlay := strings.Repeat("\n", topPad) + lipgloss.PlaceHorizontal(availW, lipgloss.Center, dialog)

	return m.styles.App.Render(overlay)
}

// renderErrorBanner renders a red error strip at the bottom of the chat area.
func (m Model) renderErrorBanner() string {
	if m.err == nil {
		return ""
	}
	errText := fmt.Sprintf(" ⚠ %v ", m.err)
	return m.styles.ErrorMsg.Width(m.width).Render(errText)
}

// renderInputBar renders the text input area at the bottom.
func (m Model) renderInputBar() string {
	// Show thinking indicator if waiting for LLM
	if m.thinking {
		spinner := m.renderSpinner()
		thinkingText := fmt.Sprintf(" %s G-MAN is thinking...", spinner)
		return m.styles.Input.Width(m.width).Render(thinkingText)
	}

	return m.styles.Input.Width(m.width).Render(m.input.View())
}

// renderSpinner returns a simple ASCII spinner frame for the thinking state.
func (m Model) renderSpinner() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	// Use a simple animation based on the nanosecond timestamp
	// This is a simple frame cycler — a more sophisticated approach
	// would track frame state in the model
	return m.styles.Spinner.Render(frames[0])
}

// grantCount returns the number of active grants for display in the title bar.
func (m Model) grantCount() int {
	if m.session == nil {
		return 0
	}
	return len(m.session.Grants)
}
