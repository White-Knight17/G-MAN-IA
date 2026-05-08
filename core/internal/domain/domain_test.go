package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// ChatMessage Tests
// =============================================================================

func TestChatMessageCreation(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		content string
	}{
		{
			name:    "user message",
			role:    "user",
			content: "Show my Hyprland config",
		},
		{
			name:    "assistant message with tool call",
			role:    "assistant",
			content: "<tool_call><name>read_file</name><path>/home/user/.config/hypr/hyprland.conf</path></tool_call>",
		},
		{
			name:    "system prompt",
			role:    "system",
			content: "You are G-MAN, an Arch Linux assistant.",
		},
		{
			name:    "tool result",
			role:    "tool",
			content: "<tool_result><output># Hyprland config...</output></tool_result>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ChatMessage{
				Role:      tt.role,
				Content:   tt.content,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}

			if msg.Role != tt.role {
				t.Errorf("Role = %q, want %q", msg.Role, tt.role)
			}
			if msg.Content != tt.content {
				t.Errorf("Content = %q, want %q", msg.Content, tt.content)
			}
			if msg.Timestamp == "" {
				t.Error("Timestamp should not be empty")
			}
		})
	}
}

func TestChatMessageValidRoles(t *testing.T) {
	validRoles := []string{"system", "user", "assistant", "tool"}

	for _, role := range validRoles {
		t.Run("role_"+role, func(t *testing.T) {
			msg := ChatMessage{
				Role:      role,
				Content:   "test content",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			// ChatMessage does not validate roles — it's a plain struct.
			// This test ensures all expected roles can be assigned.
			if msg.Role != role {
				t.Errorf("expected role %q, got %q", role, msg.Role)
			}
		})
	}
}

// =============================================================================
// Session Tests
// =============================================================================

func TestSessionCreation(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{
			name: "with UUID",
			id:   "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "with nanosecond timestamp",
			id:   "20260101-120000.000000000",
		},
		{
			name: "empty ID allowed",
			id:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := Session{
				ID:        tt.id,
				Messages:  make([]ChatMessage, 0),
				Grants:    make([]Grant, 0),
				StartedAt: time.Now().UTC().Format(time.RFC3339),
			}

			if session.ID != tt.id {
				t.Errorf("ID = %q, want %q", session.ID, tt.id)
			}
			if session.Messages == nil {
				t.Error("Messages should not be nil after make()")
			}
			if session.Grants == nil {
				t.Error("Grants should not be nil after make()")
			}
			if session.StartedAt == "" {
				t.Error("StartedAt should not be empty")
			}
		})
	}
}

func TestSessionMessageAppend(t *testing.T) {
	session := Session{
		ID:        "test-session",
		Messages:  make([]ChatMessage, 0),
		Grants:    make([]Grant, 0),
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Append user message
	session.Messages = append(session.Messages, ChatMessage{
		Role:      "user",
		Content:   "hello",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if len(session.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(session.Messages))
	}

	// Append assistant response
	session.Messages = append(session.Messages, ChatMessage{
		Role:      "assistant",
		Content:   "Hi! How can I help with your config?",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if len(session.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(session.Messages))
	}

	// Verify message order is preserved
	if session.Messages[0].Role != "user" {
		t.Errorf("first message role = %q, want 'user'", session.Messages[0].Role)
	}
	if session.Messages[1].Role != "assistant" {
		t.Errorf("second message role = %q, want 'assistant'", session.Messages[1].Role)
	}
}

func TestSessionGrantTracking(t *testing.T) {
	session := Session{
		ID:        "test-session",
		Grants:    make([]Grant, 0),
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Add a grant
	session.Grants = append(session.Grants, Grant{
		Path:      "/home/user/.config/hypr",
		Mode:      PermissionRead,
		GrantedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if len(session.Grants) != 1 {
		t.Errorf("expected 1 grant, got %d", len(session.Grants))
	}

	// Add another grant
	session.Grants = append(session.Grants, Grant{
		Path:      "/home/user/.config/waybar",
		Mode:      PermissionWrite,
		GrantedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if len(session.Grants) != 2 {
		t.Errorf("expected 2 grants, got %d", len(session.Grants))
	}
}

// =============================================================================
// Grant Tests
// =============================================================================

func TestGrantCreation(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)

	tests := []struct {
		name string
		path string
		mode PermissionMode
	}{
		{
			name: "read grant for hyprland config",
			path: "/home/user/.config/hypr",
			mode: PermissionRead,
		},
		{
			name: "write grant for waybar config",
			path: "/home/user/.config/waybar",
			mode: PermissionWrite,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grant := Grant{
				Path:      tt.path,
				Mode:      tt.mode,
				GrantedAt: now,
			}

			if grant.Path != tt.path {
				t.Errorf("Path = %q, want %q", grant.Path, tt.path)
			}
			if grant.Mode != tt.mode {
				t.Errorf("Mode = %q, want %q", grant.Mode, tt.mode)
			}
			if grant.GrantedAt != now {
				t.Errorf("GrantedAt = %q, want %q", grant.GrantedAt, now)
			}
		})
	}
}

// =============================================================================
// PermissionMode Tests
// =============================================================================

func TestPermissionModeConstants(t *testing.T) {
	tests := []struct {
		name     string
		mode     PermissionMode
		expected string
	}{
		{
			name:     "read permission",
			mode:     PermissionRead,
			expected: "ro",
		},
		{
			name:     "write permission",
			mode:     PermissionWrite,
			expected: "rw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.mode) != tt.expected {
				t.Errorf("PermissionMode = %q, want %q", tt.mode, tt.expected)
			}
		})
	}
}

func TestPermissionModeDistinct(t *testing.T) {
	if PermissionRead == PermissionWrite {
		t.Error("PermissionRead and PermissionWrite should be different values")
	}
	if PermissionRead == "" {
		t.Error("PermissionRead should not be empty string")
	}
	if PermissionWrite == "" {
		t.Error("PermissionWrite should not be empty string")
	}
}

func TestPermissionModeEmptyStringIsNotAValidMode(t *testing.T) {
	// The zero value of PermissionMode is "" — it should be treated as invalid/ungranted.
	var emptyMode PermissionMode
	if emptyMode != "" {
		t.Errorf("zero value PermissionMode should be %q, got %q", "", emptyMode)
	}
	if emptyMode == PermissionRead {
		t.Error("zero value PermissionMode should NOT equal PermissionRead")
	}
	if emptyMode == PermissionWrite {
		t.Error("zero value PermissionMode should NOT equal PermissionWrite")
	}
}

// =============================================================================
// ToolResult Tests
// =============================================================================

func TestToolResultSuccess(t *testing.T) {
	result := ToolResult{
		Success: true,
		Output:  "hyprland.conf\nhyprlock.conf\nhypridle.conf",
		Error:   "",
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Output == "" {
		t.Error("Output should not be empty on success")
	}
	if result.Error != "" {
		t.Error("Error should be empty on success")
	}
}

func TestToolResultError(t *testing.T) {
	result := ToolResult{
		Success: false,
		Output:  "",
		Error:   "file not found: /home/user/.config/sway/config",
	}

	if result.Success {
		t.Error("Success should be false")
	}
	if result.Error == "" {
		t.Error("Error should not be empty on failure")
	}
}

func TestToolResultEmpty(t *testing.T) {
	// Zero-value ToolResult
	var result ToolResult

	if result.Success {
		t.Error("zero-value Success should be false")
	}
	if result.Output != "" {
		t.Errorf("zero-value Output should be empty, got %q", result.Output)
	}
	if result.Error != "" {
		t.Errorf("zero-value Error should be empty, got %q", result.Error)
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestChatMessageLongContent(t *testing.T) {
	// Simulate a large config file content
	longContent := ""
	for i := 0; i < 5000; i++ {
		longContent += "x"
	}

	msg := ChatMessage{
		Role:      "assistant",
		Content:   longContent,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if len(msg.Content) != 5000 {
		t.Errorf("expected content length 5000, got %d", len(msg.Content))
	}
}

func TestGrantEmptyPath(t *testing.T) {
	// Grant with empty path — struct allows it, validation is caller's responsibility.
	grant := Grant{
		Path:      "",
		Mode:      PermissionRead,
		GrantedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if grant.Path != "" {
		t.Errorf("expected empty path, got %q", grant.Path)
	}
}

// =============================================================================
// JSON Serialization Tests (Task 1.5)
// =============================================================================

func TestSessionJSONSerialization(t *testing.T) {
	session := Session{
		ID:        "abc-123",
		Messages:  []ChatMessage{},
		Grants:    []Grant{},
		StartedAt: "2026-01-01T00:00:00Z",
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded Session
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.ID != "abc-123" {
		t.Errorf("expected ID 'abc-123', got %q", decoded.ID)
	}
	if decoded.StartedAt != "2026-01-01T00:00:00Z" {
		t.Errorf("expected StartedAt, got %q", decoded.StartedAt)
	}
}

func TestChatMessageJSONSerialization(t *testing.T) {
	msg := ChatMessage{
		Role:      "user",
		Content:   "Hello, G-MAN!",
		Timestamp: "2026-01-01T00:00:00Z",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded ChatMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Role != "user" {
		t.Errorf("expected Role 'user', got %q", decoded.Role)
	}
	if decoded.Content != "Hello, G-MAN!" {
		t.Errorf("expected Content preserved, got %q", decoded.Content)
	}
}

func TestGrantJSONSerialization(t *testing.T) {
	grant := Grant{
		Path:      "/home/user/.config",
		Mode:      PermissionRead,
		GrantedAt: "2026-01-01T00:00:00Z",
	}

	data, err := json.Marshal(grant)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded Grant
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Path != "/home/user/.config" {
		t.Errorf("expected Path, got %q", decoded.Path)
	}
	if decoded.Mode != PermissionRead {
		t.Errorf("expected Mode 'ro', got %q", decoded.Mode)
	}
}

func TestToolResultJSONSerialization(t *testing.T) {
	result := ToolResult{
		Success: true,
		Output:  "file contents here",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded ToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if !decoded.Success {
		t.Error("expected Success true")
	}
	if decoded.Output != "file contents here" {
		t.Errorf("expected Output, got %q", decoded.Output)
	}
}

func TestStreamEventJSONRoundTrip(t *testing.T) {
	event := StreamEvent{
		Type:    "token",
		Content: "Hello, world!",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded StreamEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Type != "token" {
		t.Errorf("expected type 'token', got %q", decoded.Type)
	}
	if decoded.Content != "Hello, world!" {
		t.Errorf("expected content, got %q", decoded.Content)
	}

	// Verify JSON keys use lowercase
	raw := string(data)
	if !strings.Contains(raw, `"type"`) {
		t.Errorf("expected JSON key 'type' in output: %s", raw)
	}
	if !strings.Contains(raw, `"content"`) {
		t.Errorf("expected JSON key 'content' in output: %s", raw)
	}
}

// =============================================================================
// StreamEvent Tests
// =============================================================================

func TestStreamEventTypes(t *testing.T) {
	tests := []struct {
		name       string
		eventType  string
		content    string
		errMsg     string
		wantValid  bool
	}{
		{
			name:      "token event",
			eventType: "token",
			content:   "Hello",
			wantValid: true,
		},
		{
			name:      "tool_call event",
			eventType: "tool_call",
			content:   `{"name":"read_file","params":{"path":"/tmp/test"}}`,
			wantValid: true,
		},
		{
			name:      "tool_result event",
			eventType: "tool_result",
			content:   "file contents here",
			wantValid: true,
		},
		{
			name:      "done event",
			eventType: "done",
			content:   "",
			wantValid: true,
		},
		{
			name:      "error event",
			eventType: "error",
			errMsg:    "model not found",
			wantValid: true,
		},
		{
			name:      "permission_request event",
			eventType: "permission_request",
			content:   `{"tool":"write_file","path":"/etc/config"}`,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := StreamEvent{
				Type:    tt.eventType,
				Content: tt.content,
				Error:   tt.errMsg,
			}

			if event.Type != tt.eventType {
				t.Errorf("Type = %q, want %q", event.Type, tt.eventType)
			}
			if event.Content != tt.content {
				t.Errorf("Content = %q, want %q", event.Content, tt.content)
			}
			if event.Error != tt.errMsg {
				t.Errorf("Error = %q, want %q", event.Error, tt.errMsg)
			}
		})
	}
}

func TestStreamEventJSONSerialization(t *testing.T) {
	// Verify JSON round-trip for NDJSON transport
	// This test will be properly implemented after JSON tags are added (task 1.5)
	event := StreamEvent{
		Type:    "token",
		Content: "Hello, world!",
	}

	if event.Type != "token" {
		t.Errorf("expected type 'token', got %q", event.Type)
	}
}

// mockStreamAgent implements Agent with StreamRun for testing the interface contract.
type mockStreamAgent struct {
	events []StreamEvent
	err    error
}

func (m *mockStreamAgent) Run(ctx context.Context, input string, session *Session) (string, error) {
	return "mock", nil
}

func (m *mockStreamAgent) StreamRun(ctx context.Context, input string, session *Session) (<-chan StreamEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan StreamEvent, len(m.events))
	go func() {
		for _, e := range m.events {
			ch <- e
		}
		close(ch)
	}()
	return ch, nil
}

func (m *mockStreamAgent) Tools() []Tool { return nil }

func TestAgentStreamRunInterface(t *testing.T) {
	// Compile-time verification that mockStreamAgent implements Agent
	var _ Agent = &mockStreamAgent{}

	ctx := context.Background()
	session := &Session{ID: "test"}
	agent := &mockStreamAgent{
		events: []StreamEvent{
			{Type: "token", Content: "Hi"},
			{Type: "done"},
		},
	}

	ch, err := agent.StreamRun(ctx, "hello", session)
	if err != nil {
		t.Fatalf("StreamRun failed: %v", err)
	}

	var events []StreamEvent
	for evt := range ch {
		events = append(events, evt)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != "token" {
		t.Errorf("first event type: expected 'token', got %q", events[0].Type)
	}
	if events[1].Type != "done" {
		t.Errorf("last event type: expected 'done', got %q", events[1].Type)
	}
}

func TestAgentStreamRunError(t *testing.T) {
	agent := &mockStreamAgent{
		err: fmt.Errorf("connection refused"),
	}

	ctx := context.Background()
	session := &Session{ID: "test"}

	ch, err := agent.StreamRun(ctx, "input", session)
	if err == nil {
		t.Error("expected error for connection refused")
	}
	if ch != nil {
		t.Error("expected nil channel on error")
	}
}

func TestStreamEventChannelContract(t *testing.T) {
	// Simulate a StreamRun implementation that emits events and closes the channel.
	// This proves the channel contract: events emitted, then channel closed.
	ch := make(chan StreamEvent, 64)

	go func() {
		ch <- StreamEvent{Type: "token", Content: "Hello"}
		ch <- StreamEvent{Type: "token", Content: " world"}
		ch <- StreamEvent{Type: "done"}
		close(ch)
	}()

	var events []StreamEvent
	for evt := range ch {
		events = append(events, evt)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Type != "token" || events[0].Content != "Hello" {
		t.Errorf("event 0: expected token 'Hello', got %s/%q", events[0].Type, events[0].Content)
	}
	if events[1].Type != "token" || events[1].Content != " world" {
		t.Errorf("event 1: expected token ' world', got %s/%q", events[1].Type, events[1].Content)
	}
	if events[2].Type != "done" {
		t.Errorf("event 2: expected done, got %s", events[2].Type)
	}
}
