package domain

import (
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
