package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gentleman/gman/internal/domain"
)

// Update processes incoming messages and returns the updated model
// and any commands to execute. It implements tea.Model.Update.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// --- Keyboard input ---
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	// --- Window resize ---
	case tea.WindowSizeMsg:
		return m.handleWindowResize(msg)

	// --- Async orchestrator responses ---
	case orchestratorResponseMsg:
		return m.handleOrchestratorResponse(msg)

	case orchestratorErrorMsg:
		return m.handleOrchestratorError(msg)

	// --- Grant dialog events ---
	case GrantRequestMsg:
		return m.handleGrantRequest(msg)

	case grantResponseMsg:
		return m.handleGrantResponse(msg)

	// --- Spinner tick ---
	case spinnerTickMsg:
		if m.thinking {
			return m, m.spinnerTick()
		}
		return m, nil

	// --- Default: delegate to focused component ---
	default:
		return m.handleDefault(msg)
	}
}

// handleKeyMsg processes all keyboard input, routing based on
// whether the grant dialog is open or the normal chat mode is active.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Grant dialog mode: only y/n/esc are relevant
	if m.showGrantDialog {
		return m.handleGrantDialogKey(msg)
	}

	// Normal mode key bindings
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Send):
		return m.handleSend()

	case key.Matches(msg, m.keys.TogglePreview):
		m.showPreview = !m.showPreview
		m.recalculateLayout()
		return m, nil

	case key.Matches(msg, m.keys.ScrollUp):
		m.viewport.LineUp(1)
		return m, nil

	case key.Matches(msg, m.keys.ScrollDown):
		m.viewport.LineDown(1)
		return m, nil
	}

	// Default: delegate to text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// handleSend sends the user's current input to the orchestrator
// asynchronously and adds the user message to the chat history.
func (m Model) handleSend() (tea.Model, tea.Cmd) {
	userInput := strings.TrimSpace(m.input.Value())
	if userInput == "" {
		return m, nil
	}

	// Start thinking state
	m.thinking = true

	// Add user message to display and session
	now := time.Now().UTC().Format(time.RFC3339)
	msg := domain.ChatMessage{
		Role:      "user",
		Content:   userInput,
		Timestamp: now,
	}
	m.messages = append(m.messages, msg)
	m.session.Messages = append(m.session.Messages, msg)

	// Clear input
	m.input.SetValue("")
	m.input, _ = m.input.Update(nil) // reset cursor position

	// Launch async LLM call
	return m, tea.Batch(
		callOrchestrator(m.orchestrator, m.session, userInput),
		m.spinnerTick(),
	)
}

// handleGrantDialogKey processes key presses when the grant dialog modal is visible.
func (m Model) handleGrantDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Yes):
		return m, func() tea.Msg {
			return grantResponseMsg{approved: true}
		}

	case key.Matches(msg, m.keys.No):
		return m, func() tea.Msg {
			return grantResponseMsg{approved: false}
		}
	}
	return m, nil
}

// handleWindowResize recalculates the split-pane layout when the terminal resizes.
func (m Model) handleWindowResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	if !m.ready {
		// First resize: mark as ready for proper rendering
		m.ready = true
	}

	m.recalculateLayout()
	return m, nil
}

// handleOrchestratorResponse processes a successful LLM response.
func (m Model) handleOrchestratorResponse(msg orchestratorResponseMsg) (tea.Model, tea.Cmd) {
	m.thinking = false
	m.err = nil

	now := time.Now().UTC().Format(time.RFC3339)
	m.messages = append(m.messages, domain.ChatMessage{
		Role:      "assistant",
		Content:   msg.response,
		Timestamp: now,
	})

	m.updateViewport()
	return m, nil
}

// handleOrchestratorError processes an LLM call failure.
func (m Model) handleOrchestratorError(msg orchestratorErrorMsg) (tea.Model, tea.Cmd) {
	m.thinking = false
	m.err = msg.err

	now := time.Now().UTC().Format(time.RFC3339)
	errContent := fmt.Sprintf("Error: %v", msg.err)
	m.messages = append(m.messages, domain.ChatMessage{
		Role:      "system",
		Content:   errContent,
		Timestamp: now,
	})

	m.updateViewport()
	return m, nil
}

// handleGrantRequest shows the grant confirmation dialog.
func (m Model) handleGrantRequest(msg GrantRequestMsg) (tea.Model, tea.Cmd) {
	m.showGrantDialog = true
	m.pendingGrant = msg.Grant
	return m, nil
}

// handleGrantResponse processes the user's response to the grant dialog.
func (m Model) handleGrantResponse(msg grantResponseMsg) (tea.Model, tea.Cmd) {
	m.showGrantDialog = false

	if m.grantApproved != nil {
		m.grantApproved <- msg.approved
	}

	return m, nil
}

// handleDefault delegates unhandled messages to the focused input component.
func (m Model) handleDefault(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// recalculateLayout adjusts the chat and preview panel widths
// based on the current terminal width and preview visibility.
func (m *Model) recalculateLayout() {
	availableWidth := m.width

	if availableWidth <= 0 {
		availableWidth = 80 // sensible default before first resize
	}

	if m.showPreview && availableWidth > 40 {
		// 70/30 split
		m.chatWidth = (availableWidth * 70) / 100
		m.previewWidth = availableWidth - m.chatWidth - 1 // -1 for divider
		if m.previewWidth < 10 {
			m.previewWidth = 10
			m.chatWidth = availableWidth - m.previewWidth - 1
		}
	} else {
		m.chatWidth = availableWidth
		m.previewWidth = 0
	}

	// Update viewport dimensions
	headerHeight := 1  // title bar
	inputHeight := 3   // input bar
	viewportHeight := m.height - headerHeight - inputHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	m.viewport.Width = m.chatWidth
	m.viewport.Height = viewportHeight
}

// updateViewport sets the viewport content from the message list.
func (m *Model) updateViewport() {
	content := m.renderChatContent()
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

// renderChatContent produces the full chat panel text for the viewport.
func (m *Model) renderChatContent() string {
	var sb strings.Builder
	for _, msg := range m.messages {
		sb.WriteString(m.renderMessage(msg))
		sb.WriteString("\n")
	}
	return sb.String()
}

// renderMessage formats a single chat message with role prefix and styling.
func (m *Model) renderMessage(msg domain.ChatMessage) string {
	roleDisplay := m.roleDisplay(msg.Role)
	style := m.roleStyle(msg.Role)

	return style.Render(fmt.Sprintf("%s %s", roleDisplay, msg.Content))
}

// roleDisplay returns the display prefix for a message role.
func (m *Model) roleDisplay(role string) string {
	switch role {
	case "user":
		return "[You]"
	case "assistant":
		return "[G-MAN]"
	case "system":
		return "[System]"
	case "tool":
		return "[Tool]"
	default:
		return "[" + role + "]"
	}
}

// roleStyle returns the Lipgloss style for a message role.
func (m *Model) roleStyle(role string) lipgloss.Style {
	switch role {
	case "user":
		return m.styles.UserMsg
	case "assistant":
		return m.styles.GMANMsg
	case "system":
		return m.styles.SystemMsg
	default:
		return m.styles.GMANMsg
	}
}

// spinnerTick returns a command that sends a tick message for the spinner.
func (m Model) spinnerTick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return spinnerTickMsg{t}
	})
}

// spinnerTickMsg is a tick message for the thinking spinner animation.
type spinnerTickMsg struct {
	time.Time
}

// callOrchestrator launches ChatOrchestrator.HandleMessage in a goroutine
// and returns the result as a tea.Msg. This keeps the TUI responsive
// during potentially long-running LLM calls.
func callOrchestrator(
	orchestrator ChatOrchestrator,
	session *domain.Session,
	input string,
) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		response, err := orchestrator.HandleMessage(ctx, session, input)
		if err != nil {
			return orchestratorErrorMsg{err: err}
		}
		return orchestratorResponseMsg{response: response}
	}
}
