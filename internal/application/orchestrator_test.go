package application_test

import (
	"context"
	"strings"
	"testing"

	"github.com/gentleman/programas/harvey/internal/application"
	"github.com/gentleman/programas/harvey/internal/domain"
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
			`<tool_call><name>read_file</name><path>/home/user/.config/hypr/hyprland.conf</path></tool_call>`,
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
	// Agent always returns a tool call, causing an infinite loop
	agent := &stubAgent{
		responses: []string{
			`<tool_call><name>read_file</name><path>/tmp/a</path></tool_call>`,
			`<tool_call><name>read_file</name><path>/tmp/b</path></tool_call>`,
			`<tool_call><name>read_file</name><path>/tmp/c</path></tool_call>`,
			`<tool_call><name>read_file</name><path>/tmp/d</path></tool_call>`,
			`<tool_call><name>read_file</name><path>/tmp/e</path></tool_call>`,
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
			`<tool_call><name>read_file</name><path>/nonexistent</path></tool_call>`,
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
