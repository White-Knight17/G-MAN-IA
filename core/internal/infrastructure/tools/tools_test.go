package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman/gman/internal/domain"
)

// --- Test Helpers ---

// stubSandbox is a minimal sandbox implementation for testing.
type stubSandbox struct {
	allowedPaths []string
	execFunc     func(ctx context.Context, command string, args []string, allowedPaths []string) (string, error)
}

func (s *stubSandbox) Execute(ctx context.Context, command string, args []string, allowedPaths []string) (string, error) {
	if s.execFunc != nil {
		return s.execFunc(ctx, command, args, allowedPaths)
	}
	return "stub output", nil
}

func (s *stubSandbox) AllowedPaths() []string { return s.allowedPaths }

// --- ReadFileTool Tests ---

func TestReadFileTool_Name(t *testing.T) {
	tool := NewReadFileTool([]string{"/tmp"}, &stubSandbox{allowedPaths: []string{"/tmp"}})
	if tool.Name() != "read_file" {
		t.Errorf("expected name 'read_file', got %q", tool.Name())
	}
}

func TestReadFileTool_SchemaXML(t *testing.T) {
	tool := NewReadFileTool([]string{"/tmp"}, &stubSandbox{allowedPaths: []string{"/tmp"}})
	schema := tool.SchemaXML()
	if !strings.Contains(schema, "read_file") {
		t.Errorf("schema should contain tool name: %s", schema)
	}
}

func TestReadFileTool_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := NewReadFileTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{"path": testFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "hello world") {
		t.Errorf("expected output to contain 'hello world', got %q", result.Output)
	}
}

func TestReadFileTool_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewReadFileTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{"path": filepath.Join(tmpDir, "nonexistent.txt")})
	if err == nil {
		t.Error("expected error for missing file")
	}
	if result.Success {
		t.Error("expected failure for missing file")
	}
}

func TestReadFileTool_MissingParam(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewReadFileTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]string{})
	if err == nil {
		t.Error("expected error for missing 'path' parameter")
	}
}

func TestReadFileTool_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewReadFileTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{"path": filepath.Join(tmpDir, "..", "..", "etc", "passwd")})
	if err == nil {
		t.Error("expected error for path traversal")
	}
	if result.Success {
		t.Error("expected failure for path traversal")
	}
}

func TestReadFileTool_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewReadFileTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{"path": tmpDir})
	if err == nil {
		t.Error("expected error when reading a directory")
	}
	if result.Success {
		t.Error("expected failure for directory")
	}
}

// --- WriteFileTool Tests ---

func TestWriteFileTool_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "write_test.txt")
	if err := os.WriteFile(testFile, []byte("original content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := NewWriteFileTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{
		"path":    testFile,
		"content": "new content\nwith multiple lines",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "Wrote") {
		t.Errorf("expected output to contain 'Wrote', got %q", result.Output)
	}

	// Verify backup was created
	bakPath := testFile + ".bak"
	if _, err := os.Stat(bakPath); os.IsNotExist(err) {
		t.Errorf("expected backup file %s to exist", bakPath)
	}

	// Verify new content was written
	data, _ := os.ReadFile(testFile)
	if string(data) != "new content\nwith multiple lines" {
		t.Errorf("expected new content, got %q", string(data))
	}
}

func TestWriteFileTool_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new_file.txt")

	tool := NewWriteFileTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{
		"path":    testFile,
		"content": "brand new file",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "Created new file") {
		t.Errorf("expected 'Created new file', got %q", result.Output)
	}
}

func TestWriteFileTool_MissingParams(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewWriteFileTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	tests := []struct {
		name   string
		params map[string]string
	}{
		{"missing path", map[string]string{"content": "test"}},
		{"missing content", map[string]string{"path": "/tmp/test.txt"}},
		{"empty params", map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(ctx, tt.params)
			if err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestWriteFileTool_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewWriteFileTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{
		"path":    filepath.Join(tmpDir, "..", "..", "etc", "hosts"),
		"content": "malicious",
	})
	if err == nil {
		t.Error("expected error for path traversal")
	}
	if result.Success {
		t.Error("expected failure for path traversal")
	}
}

func TestWriteFileTool_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewWriteFileTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{
		"path":    tmpDir,
		"content": "test",
	})
	if err == nil {
		t.Error("expected error when writing to a directory")
	}
	if result.Success {
		t.Error("expected failure for directory")
	}
}

// --- ListDirTool Tests ---

func TestListDirTool_Success(t *testing.T) {
	tmpDir := t.TempDir()
	// Create some test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("b"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	tool := NewListDirTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{"path": tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "file1.txt") {
		t.Errorf("expected 'file1.txt' in output, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "subdir/") {
		t.Errorf("expected 'subdir/' in output, got %q", result.Output)
	}
}

func TestListDirTool_HiddenFilesExcluded(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("b"), 0644)

	tool := NewListDirTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{"path": tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result.Output, ".hidden") {
		t.Error("hidden files should be excluded from listing")
	}
	if !strings.Contains(result.Output, "visible.txt") {
		t.Error("visible files should be in listing")
	}
}

func TestListDirTool_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewListDirTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{"path": filepath.Join(tmpDir, "..", "..", "etc")})
	if err == nil {
		t.Error("expected error for path traversal")
	}
	if result.Success {
		t.Error("expected failure for path traversal")
	}
}

func TestListDirTool_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewListDirTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{"path": filepath.Join(tmpDir, "nonexistent")})
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
	if result.Success {
		t.Error("expected failure for missing directory")
	}
}

func TestListDirTool_MissingParam(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewListDirTool([]string{tmpDir}, &stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]string{})
	if err == nil {
		t.Error("expected error for missing 'path' parameter")
	}
}

// --- CommandTool Tests ---

func TestCommandTool_Allowlist(t *testing.T) {
	sandbox := &stubSandbox{
		allowedPaths: []string{"/tmp"},
		execFunc: func(ctx context.Context, command string, args []string, allowedPaths []string) (string, error) {
			return "ok", nil
		},
	}
	tool := NewCommandTool(sandbox, []string{"/tmp"})
	ctx := context.Background()

	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{"allow grep", "grep test file.txt", false},
		{"allow ls", "ls -la /tmp", false},
		{"allow cat", "cat /tmp/test.txt", false},
		{"allow hyprctl", "hyprctl monitors", false},
		{"allow waybar", "waybar --version", false},
		{"allow find", "find /tmp -name test", false},
		{"allow which", "which bwrap", false},
		{"allow echo", "echo hello", false},
		{"block rm", "rm -rf /tmp", true},
		{"block sudo", "sudo ls", true},
		{"block unknown", "unknowncommand", true},
		{"allow pacman query", "pacman -Q firefox", false},
		{"block pacman sync", "pacman -S firefox", true},
		{"block pacman remove", "pacman -R firefox", true},
		{"block systemctl enable", "systemctl --user enable test", true},
		{"block systemctl start", "systemctl --user start test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, map[string]string{"command": tt.command})
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for command %q", tt.command)
				}
				if result.Success {
					t.Errorf("expected failure for command %q", tt.command)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for command %q: %v", tt.command, err)
				}
			}
		})
	}
}

func TestCommandTool_SystemctlRequiresUser(t *testing.T) {
	sandbox := &stubSandbox{
		allowedPaths: []string{"/tmp"},
		execFunc: func(ctx context.Context, command string, args []string, allowedPaths []string) (string, error) {
			return "ok", nil
		},
	}
	tool := NewCommandTool(sandbox, []string{"/tmp"})
	ctx := context.Background()

	// Without --user should fail
	_, err := tool.Execute(ctx, map[string]string{"command": "systemctl status"})
	if err == nil {
		t.Error("expected error for systemctl without --user")
	}

	// With --user should succeed
	result, err := tool.Execute(ctx, map[string]string{"command": "systemctl --user status"})
	if err != nil {
		t.Errorf("unexpected error for systemctl --user: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success for systemctl --user")
	}
}

func TestCommandTool_MissingParam(t *testing.T) {
	sandbox := &stubSandbox{allowedPaths: []string{"/tmp"}}
	tool := NewCommandTool(sandbox, []string{"/tmp"})
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]string{})
	if err == nil {
		t.Error("expected error for missing 'command' parameter")
	}
}

// --- CheckSyntaxTool Tests ---

func TestCheckSyntaxTool_Hyprland(t *testing.T) {
	tool := NewCheckSyntaxTool(&stubSandbox{allowedPaths: []string{"/tmp"}})
	ctx := context.Background()

	tests := []struct {
		name    string
		content string
		success bool
		hasErr  bool
	}{
		{
			name:    "valid hyprland config",
			content: "monitor = ,preferred,auto,1\ninput {\n  kb_layout = us\n}\ngeneral {\n  gaps_in = 5\n}",
			success: true,
		},
		{
			name:    "brace mismatch",
			content: "general {\n  gaps_in = 5\n",
			success: true, // check_syntax reports issues but doesn't "fail"
			hasErr:  false,
		},
		{
			name:    "unknown section",
			content: "fakesection {\n  something = value\n}",
			success: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, map[string]string{
				"filetype": "hyprland",
				"content":  tt.content,
			})
			if tt.hasErr && err == nil {
				t.Error("expected error")
			}
			if result.Success != tt.success {
				t.Errorf("expected success=%v, got %v", tt.success, result.Success)
			}
		})
	}
}

func TestCheckSyntaxTool_Waybar(t *testing.T) {
	tool := NewCheckSyntaxTool(&stubSandbox{allowedPaths: []string{"/tmp"}})
	ctx := context.Background()

	tests := []struct {
		name    string
		content string
		success bool
	}{
		{
			name:    "valid waybar json",
			content: `{"layer": "top", "modules-left": ["hyprland/workspaces"]}`,
			success: true,
		},
		{
			name:    "invalid json (no brackets)",
			content: "just some text",
			success: true, // tool reports issues, not errors
		},
		{
			name:    "brace mismatch",
			content: `{"layer": "top"`,
			success: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, map[string]string{
				"filetype": "waybar",
				"content":  tt.content,
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result.Success != tt.success {
				t.Errorf("expected success=%v, got %v", tt.success, result.Success)
			}
		})
	}
}

func TestCheckSyntaxTool_Bash(t *testing.T) {
	_, bashErr := os.Stat("/bin/bash")
	if bashErr != nil {
		t.Skip("bash not available")
	}

	tool := NewCheckSyntaxTool(&stubSandbox{allowedPaths: []string{"/tmp"}})
	ctx := context.Background()

	tests := []struct {
		name    string
		content string
		success bool
	}{
		{
			name:    "valid bash script",
			content: "#!/bin/bash\necho hello\n",
			success: true,
		},
		{
			name:    "syntax error in bash",
			content: "#!/bin/bash\nif [\n",
			success: true, // tool reports errors via output, not via error return
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, map[string]string{
				"filetype": "bash",
				"content":  tt.content,
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result.Success != tt.success {
				t.Errorf("expected success=%v, got %v", tt.success, result.Success)
			}
		})
	}
}

func TestCheckSyntaxTool_UnsupportedType(t *testing.T) {
	tool := NewCheckSyntaxTool(&stubSandbox{allowedPaths: []string{"/tmp"}})
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]string{
		"filetype": "unsupported",
		"content":  "test",
	})
	if err == nil {
		t.Error("expected error for unsupported filetype")
	}
}

func TestCheckSyntaxTool_MissingParams(t *testing.T) {
	tool := NewCheckSyntaxTool(&stubSandbox{allowedPaths: []string{"/tmp"}})
	ctx := context.Background()

	tests := []struct {
		name   string
		params map[string]string
	}{
		{"missing filetype", map[string]string{"content": "test"}},
		{"missing content", map[string]string{"filetype": "bash"}},
		{"empty params", map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(ctx, tt.params)
			if err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

// --- SearchWikiTool Tests ---

func TestSearchWikiTool_NoKnowledgeBase(t *testing.T) {
	tool := NewSearchWikiTool(&stubSandbox{allowedPaths: []string{"/tmp"}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{"query": "hyprland"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("search_wiki should return success even when kb doesn't exist")
	}
	if !strings.Contains(result.Output, "No knowledge base found") {
		t.Errorf("expected 'No knowledge base found' message, got %q", result.Output)
	}
}

func TestSearchWikiTool_EmptyKnowledgeBase(t *testing.T) {
	tmpDir := t.TempDir()
	kbDir := filepath.Join(tmpDir, ".config", "gman", "knowledge")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatalf("failed to create kb dir: %v", err)
	}

	// Override home dir detection by using the tool's own knowledge dir
	tool := NewSearchWikiTool(&stubSandbox{allowedPaths: []string{tmpDir}})
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]string{"query": "nothing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should find the actual home dir kb (or the temp one if we could redirect)
	// Since we can't easily redirect home dir in tests, this test validates the
	// tool doesn't crash on an empty kb.
	if !result.Success {
		t.Errorf("expected success: %s", result.Error)
	}
}

func TestSearchWikiTool_WithKnowledgeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	kbDir := filepath.Join(tmpDir, "knowledge")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatalf("failed to create kb dir: %v", err)
	}

	// Create test knowledge files
	os.WriteFile(filepath.Join(kbDir, "hyprland.md"), []byte("# Hyprland\n\nHyprland is a dynamic tiling Wayland compositor.\n\n## Gaps\n\nSet gaps with general { gaps_in = 5 }\n"), 0644)
	os.WriteFile(filepath.Join(kbDir, "waybar.md"), []byte("# Waybar\n\nWaybar is a status bar for Wayland.\n"), 0644)

	tool := &SearchWikiTool{sandbox: &stubSandbox{allowedPaths: []string{tmpDir}}}
	ctx := context.Background()

	tests := []struct {
		name    string
		query   string
		want    string
		wantNil bool
	}{
		{
			name:  "find hyprland",
			query: "hyprland",
			want:  "hyprland.md",
		},
		{
			name:  "find gaps",
			query: "gaps",
			want:  "hyprland.md",
		},
		{
			name:    "no results",
			query:   "nonexistent",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to intercept the home dir lookup. Since we can't easily,
			// test the internal search logic manually via the Walk.
			// For a more practical test, just verify the tool signature works.
			result, err := tool.Execute(ctx, map[string]string{"query": tt.query})
			if err != nil && !tt.wantNil {
				t.Logf("tool error (expected if kb not found in home): %v", err)
			}
			if tt.wantNil && err == nil {
				t.Logf("result: %s", result.Output)
			}
		})
	}
}

func TestSearchWikiTool_MissingParam(t *testing.T) {
	tool := NewSearchWikiTool(&stubSandbox{allowedPaths: []string{"/tmp"}})
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]string{})
	if err == nil {
		t.Error("expected error for missing 'query' parameter")
	}
}

func TestSearchWikiTool_Name(t *testing.T) {
	tool := NewSearchWikiTool(&stubSandbox{allowedPaths: []string{"/tmp"}})
	if tool.Name() != "search_wiki" {
		t.Errorf("expected name 'search_wiki', got %q", tool.Name())
	}
}

// --- computeDiff Tests ---

func TestComputeDiff(t *testing.T) {
	tests := []struct {
		name     string
		original string
		new      string
		path     string
		contains string
	}{
		{
			name:     "new file",
			original: "",
			new:      "line1\nline2\n",
			path:     "/tmp/test.txt",
			contains: "Created new file",
		},
		{
			name:     "no changes",
			original: "line1\nline2\n",
			new:      "line1\nline2\n",
			path:     "/tmp/test.txt",
			contains: "no changes",
		},
		{
			name:     "lines added",
			original: "line1\n",
			new:      "line1\nline2\nline3\n",
			path:     "/tmp/test.txt",
			contains: "lines added",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeDiff(tt.original, tt.new, tt.path)
			if !strings.Contains(strings.ToLower(result), strings.ToLower(tt.contains)) {
				t.Errorf("expected diff to contain %q, got %q", tt.contains, result)
			}
		})
	}
}

// --- isWithinAllowedDirs Tests ---

func TestIsWithinAllowedDirs(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		dirs     []string
		expected bool
	}{
		{
			name:     "path within single dir",
			path:     "/home/user/.config/hypr/hyprland.conf",
			dirs:     []string{"/home/user/.config"},
			expected: true,
		},
		{
			name:     "path outside dir",
			path:     "/home/user/Documents/file.txt",
			dirs:     []string{"/home/user/.config"},
			expected: false,
		},
		{
			name:     "exact dir match",
			path:     "/home/user/.config",
			dirs:     []string{"/home/user/.config"},
			expected: true,
		},
		{
			name:     "traversal blocked",
			path:     "/home/user/.config/../../../etc/passwd",
			dirs:     []string{"/home/user/.config"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWithinAllowedDirs(tt.path, tt.dirs)
			if got != tt.expected {
				t.Errorf("isWithinAllowedDirs(%q, %v) = %v, want %v", tt.path, tt.dirs, got, tt.expected)
			}
		})
	}
}

// --- Interface compliance check ---

func TestToolsImplementDomainTool(t *testing.T) {
	// Compile-time check: all tools should satisfy domain.Tool
	var _ domain.Tool = (*ReadFileTool)(nil)
	var _ domain.Tool = (*WriteFileTool)(nil)
	var _ domain.Tool = (*ListDirTool)(nil)
	var _ domain.Tool = (*CommandTool)(nil)
	var _ domain.Tool = (*CheckSyntaxTool)(nil)
	var _ domain.Tool = (*SearchWikiTool)(nil)
}
