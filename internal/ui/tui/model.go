package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gentleman/programas/harvey/internal/domain"
)

// ChatOrchestrator is the interface the TUI uses to send messages to
// the ReAct agent loop. The concrete *application.ChatOrchestrator
// implements this interface. Defined here (UI layer) to satisfy the
// Clean Architecture dependency rule: UI depends on domain ports,
// not on application/infrastructure.
type ChatOrchestrator interface {
	// HandleMessage processes a user message through the full ReAct loop
	// and returns the final assistant response.
	HandleMessage(ctx context.Context, session *domain.Session, userInput string) (string, error)
}

// Model is the top-level Bubbletea model for the Harvey TUI.
// It owns all UI state: chat history, input, viewport, grant dialogs,
// and layout dimensions.
type Model struct {
	// Chat state
	messages []domain.ChatMessage
	input    textinput.Model
	viewport viewport.Model
	thinking bool // true while waiting for LLM response (shows spinner)

	// Split view dimensions
	chatWidth    int
	previewWidth int
	showPreview  bool

	// Grant dialog state
	showGrantDialog bool
	pendingGrant    domain.Grant
	grantApproved   chan bool // channel to send grant response back

	// Dependencies (injected at construction)
	orchestrator ChatOrchestrator
	session      *domain.Session
	styles       Styles
	keys         KeyMap

	// Layout state
	width  int
	height int
	ready  bool
	err    error
}

// NewModel creates a new TUI Model with the given orchestrator.
// It initializes the text input, viewport, and a fresh Session.
func NewModel(orchestrator ChatOrchestrator) Model {
	ti := textinput.New()
	ti.Placeholder = "Ask Harvey..."
	ti.CharLimit = 2000
	ti.Focus()

	vp := viewport.New(0, 0)

	session := &domain.Session{
		ID:        generateSessionID(),
		Messages:  make([]domain.ChatMessage, 0),
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	return Model{
		input:        ti,
		viewport:     vp,
		orchestrator: orchestrator,
		session:      session,
		styles:       DefaultStyles(),
		keys:         DefaultKeyMap(),
		chatWidth:    70,
		previewWidth: 30,
		showPreview:  false,
	}
}

// generateSessionID creates a simple unique session identifier.
// Uses nanosecond timestamp — sufficient for a local single-instance TUI.
func generateSessionID() string {
	return time.Now().UTC().Format("20060102-150405.000000000")
}

// Init initializes the model and returns the initial command.
// Starts the text input cursor blink and enters the alternate screen buffer.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tea.EnterAltScreen,
	)
}
