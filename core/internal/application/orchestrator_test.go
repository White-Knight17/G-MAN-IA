package application_test

import (
	"context"
	"strings"
	"testing"

	"github.com/gentleman/gman/internal/application"
	"github.com/gentleman/gman/internal/domain"
)

// stubAgent implements domain.Agent for testing the orchestrator.
type stubAgent struct {
	tools     []domain.Tool
	responses []string // sequence of responses for multi-turn tests
	callCount int
	err       error
}

func (a *stubAgent) Run(ctx context.Context, input string, session *domain.Session) (string, error) {
	if a.err != nil {
		return "", a.err
	}
	if a.callCount >= len(a.responses) {
		return "no more responses", nil
	}
	resp := a.responses[a.callCount]
	a.callCount++
	return resp, nil
}

func (a *stubAgent) StreamRun(ctx context.Context, input string, session *domain.Session) (<-chan domain.StreamEvent, error) {
	ch := make(chan domain.StreamEvent, 64)
	go func() {
		result, err := a.Run(ctx, input, session)
		if err != nil {
			ch <- domain.StreamEvent{Type: "error", Error: err.Error()}
		} else {
			ch <- domain.StreamEvent{Type: "token", Content: result}
		}
		ch <- domain.StreamEvent{Type: "done"}
		close(ch)
	}()
	return ch, nil
}

func (a *stubAgent) Tools() []domain.Tool { return a.tools }

func TestChatOrchestrator_SimpleResponse_NoToolCall(t *testing.T) {
	agent := &stubAgent{
		responses: []string{"This is a direct answer, no tools needed."},
	}

	perms := newStubPermissionRepo()
	tools := []domain.Tool{}
	sandbox := &stubSandbox{}
	executor := application.NewToolExecutor(tools, sandbox, perms)
	grantMgr := application.NewGrantManager(perms)

	orch := application.NewChatOrchestrator(agent, executor, grantMgr)

	session := &domain.Session{ID: "test"}
	response, err := orch.HandleMessage(context.Background(), session, "What's a good terminal font?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(response, "direct answer") {
		t.Errorf("unexpected response: %q", response)
	}

	// Verify session history
	if len(session.Messages) != 2 { // user + assistant
		t.Errorf("expected 2 messages in session, got %d", len(session.Messages))
	}
	if session.Messages[0].Role != "user" {
		t.Errorf("first message role = %q, want 'user'", session.Messages[0].Role)
	}
	if session.Messages[1].Role != "assistant" {
		t.Errorf("second message role = %q, want 'assistant'", session.Messages[1].Role)
	}
}

func TestChatOrchestrator_ToolCallOneTurn(t *testing.T) {
	agent := &stubAgent{
		responses: []string{
			"Let me read your config.\nREAD: /home/user/.config/hypr/hyprland.conf",
			"Here's the summary of your config: it looks good!",
		},
	}

	perms := newStubPermissionRepo()
	perms.Grant("/home/user/.config/hypr/hyprland.conf", domain.PermissionRead)

	tools := []domain.Tool{
		&stubTool{
			name:        "read_file",
			description: "Reads a file",
			schema:      `<tool_call><name>read_file</name><path>/absolute/path</path></tool_call>`,
			execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
				return domain.ToolResult{Success: true, Output: "# Hyprland config\nmonitor=eDP-1,1920x1080,0x0,1"}, nil
			},
		},
	}

	sandbox := &stubSandbox{}
	grantMgr := application.NewGrantManager(perms)
	executor := application.NewToolExecutor(tools, sandbox, perms)

	orch := application.NewChatOrchestrator(agent, executor, grantMgr)

	session := &domain.Session{ID: "test"}
	response, err := orch.HandleMessage(context.Background(), session, "Show me my config")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(response, "good") {
		t.Errorf("expected final response to contain 'good', got: %q", response)
	}

	// Verify session history has user → assistant(tool_call) → tool → assistant
	if len(session.Messages) != 4 {
		t.Errorf("expected 4 messages, got %d", len(session.Messages))
		for i, m := range session.Messages {
			t.Logf("  [%d] role=%s content=%q", i, m.Role, m.Content[:min(80, len(m.Content))])
		}
	}
}

func TestChatOrchestrator_MaxIterations(t *testing.T) {
	// Agent always returns a tool command, causing an infinite loop
	agent := &stubAgent{
		responses: []string{
			"READ: /tmp/a",
			"READ: /tmp/b",
			"READ: /tmp/c",
			"READ: /tmp/d",
			"READ: /tmp/e",
			"Should not reach this — max iterations is 5",
		},
	}

	perms := newStubPermissionRepo()
	tools := []domain.Tool{
		&stubTool{
			name:        "read_file",
			description: "Reads a file",
			schema:      `<tool_call><name>read_file</name><path>/path</path></tool_call>`,
			execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
				return domain.ToolResult{Success: true, Output: "content"}, nil
			},
		},
	}
	for _, p := range []string{"/tmp/a", "/tmp/b", "/tmp/c", "/tmp/d", "/tmp/e"} {
		perms.Grant(p, domain.PermissionRead)
	}

	sandbox := &stubSandbox{}
	grantMgr := application.NewGrantManager(perms)
	executor := application.NewToolExecutor(tools, sandbox, perms)

	orch := application.NewChatOrchestrator(agent, executor, grantMgr,
		application.WithMaxIterations(5))

	session := &domain.Session{ID: "test"}
	_, err := orch.HandleMessage(context.Background(), session, "Go!")

	if err == nil {
		t.Error("expected error for max iterations, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "max iterations") {
		t.Errorf("error should mention 'max iterations', got: %v", err)
	}
}

func TestChatOrchestrator_AgentError(t *testing.T) {
	agent := &stubAgent{
		err: context.DeadlineExceeded,
	}

	perms := newStubPermissionRepo()
	tools := []domain.Tool{}
	sandbox := &stubSandbox{}
	executor := application.NewToolExecutor(tools, sandbox, perms)
	grantMgr := application.NewGrantManager(perms)

	orch := application.NewChatOrchestrator(agent, executor, grantMgr)

	session := &domain.Session{ID: "test"}
	_, err := orch.HandleMessage(context.Background(), session, "Hi!")

	if err == nil {
		t.Error("expected error for agent failure, got nil")
	}
}

func TestChatOrchestrator_ToolErrorRecovery(t *testing.T) {
	// Agent calls a tool, tool fails, agent retries with different approach
	agent := &stubAgent{
		responses: []string{
			"READ: /nonexistent",
			"That file doesn't exist. Would you like me to check another path?",
		},
	}

	perms := newStubPermissionRepo()
	perms.Grant("/nonexistent", domain.PermissionRead)

	tools := []domain.Tool{
		&stubTool{
			name:        "read_file",
			description: "Reads a file",
			schema:      `<tool_call><name>read_file</name><path>/path</path></tool_call>`,
			execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
				return domain.ToolResult{Success: false, Error: "file not found"}, nil
			},
		},
	}

	sandbox := &stubSandbox{}
	grantMgr := application.NewGrantManager(perms)
	executor := application.NewToolExecutor(tools, sandbox, perms)

	orch := application.NewChatOrchestrator(agent, executor, grantMgr)

	session := &domain.Session{ID: "test"}
	response, err := orch.HandleMessage(context.Background(), session, "Read my config")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(response, "doesn't exist") {
		t.Errorf("expected agent to handle error gracefully, got: %q", response)
	}
}

// =============================================================================
// HandleMessageStream Tests
// =============================================================================

// streamStubAgent emits predefined StreamEvents.
type streamStubAgent struct {
	events []domain.StreamEvent
	err    error
}

func (a *streamStubAgent) Run(ctx context.Context, input string, session *domain.Session) (string, error) {
	return "not used", nil
}

func (a *streamStubAgent) StreamRun(ctx context.Context, input string, session *domain.Session) (<-chan domain.StreamEvent, error) {
	if a.err != nil {
		return nil, a.err
	}
	ch := make(chan domain.StreamEvent, len(a.events))
	go func() {
		for _, e := range a.events {
			ch <- e
		}
		close(ch)
	}()
	return ch, nil
}

func (a *streamStubAgent) Tools() []domain.Tool { return nil }

func TestChatOrchestrator_HandleMessageStream_SimpleResponse(t *testing.T) {
	agent := &streamStubAgent{
		events: []domain.StreamEvent{
			{Type: "token", Content: "Hello"},
			{Type: "token", Content: " world"},
			{Type: "done"},
		},
	}

	perms := newStubPermissionRepo()
	tools := []domain.Tool{}
	sandbox := &stubSandbox{}
	executor := application.NewToolExecutor(tools, sandbox, perms)
	grantMgr := application.NewGrantManager(perms)

	orch := application.NewChatOrchestrator(agent, executor, grantMgr)

	session := &domain.Session{ID: "test"}
	ch, err := orch.HandleMessageStream(context.Background(), session, "Hi!")
	if err != nil {
		t.Fatalf("HandleMessageStream failed: %v", err)
	}

	var events []domain.StreamEvent
	for evt := range ch {
		events = append(events, evt)
	}

	// Should have token events + done event
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(events))
	}

	hasDone := false
	hasToken := false
	for _, evt := range events {
		if evt.Type == "done" {
			hasDone = true
		}
		if evt.Type == "token" {
			hasToken = true
		}
	}
	if !hasToken {
		t.Error("expected at least one token event")
	}
	if !hasDone {
		t.Error("expected a done event")
	}
}

func TestChatOrchestrator_HandleMessageStream_WithToolCall(t *testing.T) {
	// Agent emits: tool_call → tool_result should be intercepted
	agent := &streamStubAgent{
		events: []domain.StreamEvent{
			{Type: "token", Content: "Let me check that file."},
			{Type: "tool_call", Content: `READ: /home/user/.config/test`},
			{Type: "token", Content: "The file contains config data."},
			{Type: "done"},
		},
	}

	perms := newStubPermissionRepo()
	perms.Grant("/home/user/.config/test", domain.PermissionRead)

	tools := []domain.Tool{
		&stubTool{
			name:        "read_file",
			description: "Reads a file",
			schema:      `<tool_call><name>read_file</name><path>/path</path></tool_call>`,
			execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
				return domain.ToolResult{Success: true, Output: "config content"}, nil
			},
		},
	}

	sandbox := &stubSandbox{}
	grantMgr := application.NewGrantManager(perms)
	executor := application.NewToolExecutor(tools, sandbox, perms)

	orch := application.NewChatOrchestrator(agent, executor, grantMgr)

	session := &domain.Session{ID: "test"}
	ch, err := orch.HandleMessageStream(context.Background(), session, "Check my config")
	if err != nil {
		t.Fatalf("HandleMessageStream failed: %v", err)
	}

	var events []domain.StreamEvent
	for evt := range ch {
		events = append(events, evt)
	}

	// Should have token, tool_call, tool_result, more token, done
	hasToolResult := false
	hasDone := false
	for _, evt := range events {
		if evt.Type == "tool_result" {
			hasToolResult = true
		}
		if evt.Type == "done" {
			hasDone = true
		}
	}
	if !hasToolResult {
		t.Error("expected a tool_result event")
	}
	if !hasDone {
		t.Error("expected a done event")
	}
}

func TestChatOrchestrator_HandleMessageStream_NoToolCommand(t *testing.T) {
	// Agent emits plain text response without tool commands
	agent := &streamStubAgent{
		events: []domain.StreamEvent{
			{Type: "token", Content: "I don't need any tools for this."},
			{Type: "done"},
		},
	}

	perms := newStubPermissionRepo()
	tools := []domain.Tool{}
	sandbox := &stubSandbox{}
	executor := application.NewToolExecutor(tools, sandbox, perms)
	grantMgr := application.NewGrantManager(perms)

	orch := application.NewChatOrchestrator(agent, executor, grantMgr)

	session := &domain.Session{ID: "test"}
	ch, err := orch.HandleMessageStream(context.Background(), session, "What time is it?")
	if err != nil {
		t.Fatalf("HandleMessageStream failed: %v", err)
	}

	var events []domain.StreamEvent
	for evt := range ch {
		events = append(events, evt)
	}

	// Should have tokens and done (no tool_result since no tool was called)
	for _, evt := range events {
		if evt.Type == "tool_result" {
			t.Error("expected no tool_result event for non-tool response")
		}
	}
}
