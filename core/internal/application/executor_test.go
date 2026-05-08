package application_test

import (
	"context"
	"strings"
	"testing"

	"github.com/gentleman/gman/internal/application"
	"github.com/gentleman/gman/internal/domain"
)

// stubTool is a domain.Tool implementation for testing.
type stubTool struct {
	name        string
	description string
	schema      string
	execute     func(ctx context.Context, params map[string]string) (domain.ToolResult, error)
}

func (s *stubTool) Name() string               { return s.name }
func (s *stubTool) Description() string         { return s.description }
func (s *stubTool) SchemaXML() string           { return s.schema }
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
		name        string
		response    string
		tools       []domain.Tool
		preGrants   map[string]domain.PermissionMode
		wantOutput  string
		wantSuccess bool
		wantErr     bool
		errSubstr   string
	}{
		{
			name:     "plain text response with no command",
			response: "Hello! I can help you configure your system.",
			tools: []domain.Tool{
				&stubTool{name: "read_file", description: "Reads a file", schema: ""},
			},
			wantOutput:  "Hello! I can help you configure your system.",
			wantSuccess: true,
		},
		{
			name:     "READ command",
			response: "Let me read that file for you.\nREAD: /home/user/.config/hypr/hyprland.conf",
			tools: []domain.Tool{
				&stubTool{
					name:        "read_file",
					description: "Reads a file",
					schema:      "",
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
			name:     "case-insensitive command matching (lowercase read)",
			response: "read: /home/user/.config",
			tools: []domain.Tool{
				&stubTool{
					name:        "list_dir",
					description: "Lists a directory",
					schema:      "",
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						return domain.ToolResult{Success: true, Output: params["path"]}, nil
					},
				},
			},
			preGrants: map[string]domain.PermissionMode{
				"/home/user/.config": domain.PermissionRead,
			},
			// "read:" starts with READ which maps to read_file, but the stub tool is list_dir.
			// The executor will try to execute read_file and fail (unknown tool).
			// Let's test with the right mapping: use "LIST:" for list_dir.
			wantErr:   true,
			errSubstr: "unknown tool",
		},
		{
			name:     "LIST command (case-insensitive)",
			response: "list: /home/user/.config",
			tools: []domain.Tool{
				&stubTool{
					name:        "list_dir",
					description: "Lists a directory",
					schema:      "",
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
			name:     "WRITE command with END marker",
			response: "WRITE: /home/user/.config/hypr/hyprland.conf\nmonitor=eDP-1,1920x1080,0x0,1\nEND",
			tools: []domain.Tool{
				&stubTool{
					name:        "write_file",
					description: "Writes a file",
					schema:      "",
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						if params["content"] != "monitor=eDP-1,1920x1080,0x0,1" {
							return domain.ToolResult{Success: false, Error: "unexpected content"}, nil
						}
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
			name:     "WRITE command with multiline content",
			response: "WRITE: /tmp/test.conf\nline one\nline two\nline three\nEND\nMore text ignored",
			tools: []domain.Tool{
				&stubTool{
					name:        "write_file",
					description: "Writes a file",
					schema:      "",
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						if params["content"] != "line one\nline two\nline three" {
							return domain.ToolResult{Success: false, Error: "unexpected content"}, nil
						}
						return domain.ToolResult{Success: true, Output: "wrote file"}, nil
					},
				},
			},
			preGrants: map[string]domain.PermissionMode{
				"/tmp/test.conf": domain.PermissionWrite,
			},
			wantOutput:  "wrote file",
			wantSuccess: true,
		},
		{
			name:     "RUN command",
			response: "RUN: hyprctl monitors",
			tools: []domain.Tool{
				&stubTool{
					name:        "run_command",
					description: "Runs a command",
					schema:      "",
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						if params["command"] != "hyprctl monitors" {
							return domain.ToolResult{Success: false, Error: "wrong command"}, nil
						}
						return domain.ToolResult{Success: true, Output: "command output"}, nil
					},
				},
			},
			// No grants needed
			wantOutput:  "command output",
			wantSuccess: true,
		},
		{
			name:     "CHECK command with content and END",
			response: "CHECK: hyprland\nmonitor=,DP-1,1920x1080@144,0x0,1\nEND",
			tools: []domain.Tool{
				&stubTool{
					name:        "check_syntax",
					description: "Checks syntax",
					schema:      "",
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						if params["filetype"] != "hyprland" {
							return domain.ToolResult{Success: false, Error: "wrong filetype"}, nil
						}
						if params["content"] != "monitor=,DP-1,1920x1080@144,0x0,1" {
							return domain.ToolResult{Success: false, Error: "wrong content"}, nil
						}
						return domain.ToolResult{Success: true, Output: "Valid config"}, nil
					},
				},
			},
			wantOutput:  "Valid config",
			wantSuccess: true,
		},
		{
			name:     "SEARCH command",
			response: "SEARCH: waybar config",
			tools: []domain.Tool{
				&stubTool{
					name:        "search_wiki",
					description: "Searches wiki",
					schema:      "",
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						if params["query"] != "waybar config" {
							return domain.ToolResult{Success: false, Error: "wrong query"}, nil
						}
						return domain.ToolResult{Success: true, Output: "Found 3 results"}, nil
					},
				},
			},
			wantOutput:  "Found 3 results",
			wantSuccess: true,
		},
		{
			name:     "unknown command keyword treated as conversational",
			response: "UNKNOWN: something",
			tools: []domain.Tool{
				&stubTool{name: "read_file", description: "Reads", schema: ""},
			},
			wantOutput:  "UNKNOWN: something",
			wantSuccess: true,
		},
		{
			name:     "permission denied for write without rw grant",
			response: "WRITE: /home/user/.config/hypr/hyprland.conf\nnew content\nEND",
			tools: []domain.Tool{
				&stubTool{
					name:        "write_file",
					description: "Writes a file",
					schema:      "",
				},
			},
			preGrants: map[string]domain.PermissionMode{
				"/home/user/.config/hypr/hyprland.conf": domain.PermissionRead, // only ro
			},
			wantErr:   true,
			errSubstr: "permission denied",
		},
		{
			name:     "write_file succeeds with rw grant",
			response: "WRITE: /home/user/.config/hypr/hyprland.conf\nnew content\nEND",
			tools: []domain.Tool{
				&stubTool{
					name:        "write_file",
					description: "Writes a file",
					schema:      "",
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
			name:     "no permission check for non-filesystem tools (RUN)",
			response: "RUN: hyprctl monitors",
			tools: []domain.Tool{
				&stubTool{
					name:        "run_command",
					description: "Runs a command",
					schema:      "",
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
			name:     "multiple commands — first one wins",
			response: "READ: /tmp/first\nREAD: /tmp/second",
			tools: []domain.Tool{
				&stubTool{
					name:        "read_file",
					description: "Reads a file",
					schema:      "",
					execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
						return domain.ToolResult{Success: true, Output: "first file content"}, nil
					},
				},
			},
			preGrants: map[string]domain.PermissionMode{
				"/tmp/first": domain.PermissionRead,
			},
			wantOutput:  "first file content",
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

			result, err := executor.Execute(context.Background(), session, tt.response)

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
			schema:      "",
			execute: func(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
				return domain.ToolResult{}, context.DeadlineExceeded
			},
		},
	}

	perms := newStubPermissionRepo()
	perms.Grant("/path", domain.PermissionRead)

	sandbox := &stubSandbox{}
	executor := application.NewToolExecutor(tools, sandbox, perms)

	response := "READ: /path"
	result, err := executor.Execute(context.Background(), &domain.Session{ID: "test"}, response)

	if err == nil {
		t.Error("expected error for tool execution failure")
	}
	if result.Success {
		t.Error("expected Success=false for failed tool execution")
	}
}
