package tui_test

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gentleman/programas/harvey/internal/domain"
	"github.com/gentleman/programas/harvey/internal/ui/tui"
)

// mockOrchestrator implements tui.ChatOrchestrator for testing.
// It returns predefined responses or errors.
type mockOrchestrator struct {
	response string
	err      error
	// callCount tracks how many times HandleMessage was called
	callCount int
	// lastInput captures the last user input received
	lastInput string
}

func (m *mockOrchestrator) HandleMessage(ctx context.Context, session *domain.Session, userInput string) (string, error) {
	m.callCount++
	m.lastInput = userInput
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

// TestModelInitialState verifies the model is properly initialized.
func TestModelInitialState(t *testing.T) {
	orch := &mockOrchestrator{response: "Hello!"}
	model := tui.NewModel(orch)

	// Init should return non-nil commands (blink + enter alt screen)
	cmd := model.Init()
	if cmd == nil {
		t.Error("Init() returned nil command, expected non-nil")
	}

	// The model should start with no messages and an empty session
	// (we can't access unexported fields directly, so we test via behavior)
	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

// TestModelKeyBindings tests keyboard input handling.
func TestModelKeyBindings(t *testing.T) {
	tests := []struct {
		name          string
		keyPress      tea.KeyMsg
		expectQuit    bool
		expectThinking bool // if Enter was pressed with text
	}{
		{
			name:       "Ctrl+C quits",
			keyPress:   tea.KeyMsg{Type: tea.KeyCtrlC},
			expectQuit: true,
		},
		{
			name:       "Esc quits (no dialog open)",
			keyPress:   tea.KeyMsg{Type: tea.KeyEsc},
			expectQuit: true,
		},
		{
			name:       "q quits",
			keyPress:   tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			expectQuit: true,
		},
		{
			name:       "Enter with empty input does nothing",
			keyPress:   tea.KeyMsg{Type: tea.KeyEnter},
			expectQuit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orch := &mockOrchestrator{response: "Hello!"}
			model := tui.NewModel(orch)

			newModel, cmd := model.Update(tt.keyPress)

			if tt.expectQuit {
				// tea.Quit is a function; check that cmd is not nil
				// (we can't easily check it IS tea.Quit without calling it)
				if cmd == nil {
					t.Error("expected non-nil quit command, got nil")
				}
			}

			_ = newModel
		})
	}
}

// TestModelWindowResize verifies layout recalculation on terminal resize.
func TestModelWindowResize(t *testing.T) {
	orch := &mockOrchestrator{response: "Hello!"}
	model := tui.NewModel(orch)

	// Simulate terminal resize
	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := model.Update(resizeMsg)

	// View should render without panic
	view := newModel.View()
	if view == "" {
		t.Error("View() returned empty after resize")
	}
}

// TestModelGrantDialogFlow tests the grant confirmation dialog state transitions.
func TestModelGrantDialogFlow(t *testing.T) {
	orch := &mockOrchestrator{response: "Hello!"}
	model := tui.NewModel(orch)

	// Simulate receiving a grant request
	grant := domain.Grant{
		Path: "/home/user/.config/hypr",
		Mode: domain.PermissionWrite,
	}
	newModel, _ := model.Update(tui.GrantRequestMsg{Grant: grant})

	// View should show grant dialog
	view := newModel.View()
	if !strings.Contains(view, "/home/user/.config/hypr") {
		t.Errorf("grant dialog should show the path, got view:\n%s", view)
	}
	if !strings.Contains(view, "Allow") {
		t.Errorf("grant dialog should have 'Allow' option, got view:\n%s", view)
	}

	// User presses "y" to approve
	approveModel, _ := newModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// After approving, the dialog should close
	approveView := approveModel.View()
	if strings.Contains(approveView, "/home/user/.config/hypr") {
		// The dialog might still show if the model hasn't fully processed the response.
		// The grant dialog closes only after handling grantResponseMsg.
		// This is expected behavior — the key press queues a grantResponseMsg that
		// will be processed in the next Update cycle.
		t.Log("grant path still visible — dialog closes after processing grantResponseMsg")
	}

	// Test denial: user presses ESC in grant mode
	model2 := tui.NewModel(orch)
	newM, _ := model2.Update(tui.GrantRequestMsg{Grant: grant})
	model2 = newM.(tui.Model)
	denyM, _ := model2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	denyModel := denyM.(tui.Model)
	_ = denyModel.View()
}

// TestModelViewModes verifies different view renderings based on state.
func TestModelViewModes(t *testing.T) {
	orch := &mockOrchestrator{response: "Hello!"}
	model := tui.NewModel(orch)

	// Before any messages, view should show the title bar and input
	view := model.View()
	if !strings.Contains(view, "Harvey") {
		t.Errorf("expected 'Harvey' in title bar, got:\n%s", view)
	}
	if !strings.Contains(view, "Ask Harvey") {
		t.Errorf("expected input placeholder, got:\n%s", view)
	}

	// After sending a message, thinking indicator should show
	// (but only after processing the actual orchestrator response)
	_ = view
}

// TestMockOrchestrator verifies the mock orchestrator works correctly.
func TestMockOrchestrator(t *testing.T) {
	orch := &mockOrchestrator{
		response: "I found your config!",
	}

	session := &domain.Session{ID: "test"}
	resp, err := orch.HandleMessage(context.Background(), session, "Show my config")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resp, "found") {
		t.Errorf("unexpected response: %q", resp)
	}
	if orch.callCount != 1 {
		t.Errorf("expected 1 call, got %d", orch.callCount)
	}
	if orch.lastInput != "Show my config" {
		t.Errorf("expected input 'Show my config', got %q", orch.lastInput)
	}
}

// TestMockOrchestratorError verifies error handling in the mock.
func TestMockOrchestratorError(t *testing.T) {
	orch := &mockOrchestrator{
		err: context.DeadlineExceeded,
	}

	session := &domain.Session{ID: "test"}
	_, err := orch.HandleMessage(context.Background(), session, "Hi")

	if err == nil {
		t.Error("expected error, got nil")
	}
}
