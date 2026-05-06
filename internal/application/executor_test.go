package application_test

import (
	"context"
	"strings"
	"testing"

	"github.com/gentleman/programas/harvey/internal/application"
	"github.com/gentleman/programas/harvey/internal/domain"
)

// stubTool is a domain.Tool implementation for testing.
type stubTool struct {
	name        string
	description string
	schema      string
	execute     func(ctx context.Context, params map[string]string) (domain.ToolResult, error)
}

func (s *stubTool) Name() string                { return s.name }
func (s *stubTool) Description() string          { return s.description }
func (s *stubTool) SchemaXML() string            { return s.schema }
func (s *stubTool) Execute(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
	if s.execute != nil {
		return s.execute(ctx, params)
	}
	return domain.ToolResult{Success: true, Output: "default output"}, nil
}

// stubSandbox implements domain.Sandbox for testing.
type stubSandbox struct {
	allowedPaths []string
}

func (s *stubSandbox) Execute(ctx context.Context, command string, args []string, allowedPaths []string) (string, error) {
	return "sandbox output", nil
}

func (s *stubSandbox) AllowedPaths() []string {
	return s.allowedPaths
}

// stubPermissionRepo implements domain.PermissionRepository for testing.
type stubPermissionRepo struct {
	grants map[string]domain.PermissionMode
}

func newStubPermissionRepo() *stubPermissionRepo {
	return &stubPermissionRepo{grants: make(map[string]domain.PermissionMode)}
}

func (p *stubPermissionRepo) Grant(path string, mode domain.PermissionMode) error {
	p.grants[path] = mode
	return nil
}

func (p *stubPermissionRepo) Revoke(path string) error {
	if _, ok := p.grants[path]; !ok {
		return &stubError{"no grant for path"}
	}
	delete(p.grants, path)
	return nil
}

type stubError struct{ msg string }

func (e *stubError) Error() string { return e.msg }

func (p *stubPermissionRepo) Check(path string, mode domain.PermissionMode) bool {
	granted, ok := p.grants[path]
	if !ok {
		return false
	}
	if mode == domain.PermissionRead {
		return true
	}
	return granted == domain.PermissionWrite
}

func (p *stubPermissionRepo) ListGrants() []domain.Grant {
	grants := make([]domain.Grant, 0, len(p.grants))
	for path, mode := range p.grants {
		grants = append(grants, domain.Grant{Path: path, Mode: mode})
	}
	return grants
}

func TestToolExecutor_ParseAndExecute(t *testing.T) {
	tests := []struct {
		name       string
		toolCallXML string
		tools      []domain.Tool
		preGrants  map[string]domain.PermissionMode
		input      string
		wantOutput string
		wantSuccess bool
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "valid read_file tool call",
			toolCallXML: `<tool_call><name>read_file</name><path>/home/user/.config/hypr/hyprland.conf</path></tool_call>`,
			tools: []domain.Tool{
				&stubTool{
					name:        "read_file",
					description: "Reads a file",
					schema:      `<tool_call><name>read_file</name><path>/absolute/path</path></tool_call>`,
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						return domain.ToolResult{Success: true, Output: "file contents"}, nil
					},
				},
			},
			preGrants: map[string]domain.PermissionMode{
				"/home/user/.config/hypr/hyprland.conf": domain.PermissionRead,
			},
			wantOutput:  "file contents",
			wantSuccess: true,
		},
		{
			name:        "case-insensitive tool name Llama3.2 workaround",
			toolCallXML: `<tool_call><name>List_dir</name><path>/home/user/.config</path></tool_call>`,
			tools: []domain.Tool{
				&stubTool{
					name:        "list_dir",
					description: "Lists a directory",
					schema:      `<tool_call><name>list_dir</name><path>/absolute/path</path></tool_call>`,
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						return domain.ToolResult{Success: true, Output: params["path"]}, nil
					},
				},
			},
			preGrants: map[string]domain.PermissionMode{
				"/home/user/.config": domain.PermissionRead,
			},
			wantOutput:  "/home/user/.config",
			wantSuccess: true,
		},
		{
			name:        "unknown tool name",
			toolCallXML: `<tool_call><name>unknown_tool</name><path>/tmp</path></tool_call>`,
			tools: []domain.Tool{
				&stubTool{name: "read_file", description: "Reads a file", schema: "<tool_call></tool_call>"},
			},
			wantErr:   true,
			errSubstr: "unknown tool",
		},
		{
			name:        "permission denied for write without rw grant",
			toolCallXML: `<tool_call><name>write_file</name><path>/home/user/.config/hypr/hyprland.conf</path></tool_call>`,
			tools: []domain.Tool{
				&stubTool{
					name:        "write_file",
					description: "Writes a file",
					schema:      `<tool_call><name>write_file</name><path>/absolute/path</path><content>...</content></tool_call>`,
				},
			},
			preGrants: map[string]domain.PermissionMode{
				"/home/user/.config/hypr/hyprland.conf": domain.PermissionRead, // only ro
			},
			wantErr:   true,
			errSubstr: "permission denied",
		},
		{
			name:        "write_file succeeds with rw grant",
			toolCallXML: `<tool_call><name>write_file</name><path>/home/user/.config/hypr/hyprland.conf</path><content>new content</content></tool_call>`,
			tools: []domain.Tool{
				&stubTool{
					name:        "write_file",
					description: "Writes a file",
					schema:      `<tool_call><name>write_file</name><path>/absolute/path</path><content>...</content></tool_call>`,
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						return domain.ToolResult{Success: true, Output: "wrote file"}, nil
					},
				},
			},
			preGrants: map[string]domain.PermissionMode{
				"/home/user/.config/hypr/hyprland.conf": domain.PermissionWrite,
			},
			wantOutput:  "wrote file",
			wantSuccess: true,
		},
		{
			name:        "no permission check for non-filesystem tools",
			toolCallXML: `<tool_call><name>run_command</name><cmd>hyprctl monitors</cmd></tool_call>`,
			tools: []domain.Tool{
				&stubTool{
					name:        "run_command",
					description: "Runs a command",
					schema:      `<tool_call><name>run_command</name><cmd>command</cmd></tool_call>`,
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						return domain.ToolResult{Success: true, Output: "command output"}, nil
					},
				},
			},
			// No grants needed
			wantOutput:  "command output",
			wantSuccess: true,
		},
		{
			name:        "malformed XML without tool_call wrapper",
			toolCallXML: `<name>read_file</name><path>/tmp</path>`,
			tools: []domain.Tool{
				&stubTool{name: "read_file", description: "Reads a file", schema: ""},
			},
			wantErr:   true,
			errSubstr: "no valid",
		},
		{
			name:        "empty tool_call block",
			toolCallXML: `<tool_call></tool_call>`,
			tools: []domain.Tool{
				&stubTool{name: "read_file", description: "Reads a file", schema: ""},
			},
			wantErr:   true,
			errSubstr: "name is empty",
		},
		{
			name:        "valid tool_call with extra text before/after",
			toolCallXML: `Some text before <tool_call><name>read_file</name><path>/tmp/test</path></tool_call> and text after`,
			tools: []domain.Tool{
				&stubTool{
					name:        "read_file",
					description: "Reads a file",
					schema:      `<tool_call><name>read_file</name><path>/absolute/path</path></tool_call>`,
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						return domain.ToolResult{Success: true, Output: "ok"}, nil
					},
				},
			},
			preGrants: map[string]domain.PermissionMode{
				"/tmp/test": domain.PermissionRead,
			},
			wantOutput:  "ok",
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := newStubPermissionRepo()
			for path, mode := range tt.preGrants {
				perms.Grant(path, mode)
			}

			sandbox := &stubSandbox{}
			executor := application.NewToolExecutor(tt.tools, sandbox, perms)
			session := &domain.Session{ID: "test"}

			result, err := executor.Execute(context.Background(), session, tt.toolCallXML)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if err != nil && tt.errSubstr != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errSubstr)) {
					t.Errorf("error %q doesn't contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.wantSuccess)
			}
			if result.Output != tt.wantOutput {
				t.Errorf("Output = %q, want %q", result.Output, tt.wantOutput)
			}
		})
	}
}

func TestToolExecutor_ToolExecutionError(t *testing.T) {
	tools := []domain.Tool{
		&stubTool{
			name:        "read_file",
			description: "Reads a file",
			schema:      `<tool_call><name>read_file</name><path>/path</path></tool_call>`,
			execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
				return domain.ToolResult{}, context.DeadlineExceeded
			},
		},
	}

	perms := newStubPermissionRepo()
	perms.Grant("/path", domain.PermissionRead)

	sandbox := &stubSandbox{}
	executor := application.NewToolExecutor(tools, sandbox, perms)

	xml := `<tool_call><name>read_file</name><path>/path</path></tool_call>`
	result, err := executor.Execute(context.Background(), &domain.Session{ID: "test"}, xml)

	if err == nil {
		t.Error("expected error for tool execution failure")
	}
	if result.Success {
		t.Error("expected Success=false for failed tool execution")
	}
}
