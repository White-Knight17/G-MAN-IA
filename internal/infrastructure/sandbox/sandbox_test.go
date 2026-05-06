package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// skipIfNoBwrap skips the test if bwrap is not available.
func skipIfNoBwrap(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath(bwrapBin); err != nil {
		t.Skipf("bwrap not found at %s: %v", bwrapBin, err)
	}
}

// TestBubblewrap_Blocklist validates that dangerous commands are rejected
// before reaching bwrap.
func TestBubblewrap_Blocklist(t *testing.T) {
	skipIfNoBwrap(t)

	sandbox := NewBubblewrapSandbox([]string{"/tmp"})
	ctx := context.Background()

	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "block rm",
			command: "rm",
			args:    []string{"-rf", "/tmp/test"},
			wantErr: true,
			errMsg:  "blocked",
		},
		{
			name:    "block dd",
			command: "dd",
			args:    []string{"if=/dev/zero", "of=/tmp/test"},
			wantErr: true,
			errMsg:  "blocked",
		},
		{
			name:    "block mkfs",
			command: "mkfs",
			args:    []string{"ext4", "/dev/sda"},
			wantErr: true,
			errMsg:  "blocked",
		},
		{
			name:    "block sudo",
			command: "sudo",
			args:    []string{"ls"},
			wantErr: true,
			errMsg:  "blocked",
		},
		{
			name:    "block chmod",
			command: "chmod",
			args:    []string{"777", "/tmp/test"},
			wantErr: true,
			errMsg:  "blocked",
		},
		{
			name:    "block mount",
			command: "mount",
			args:    []string{"/dev/sda", "/mnt"},
			wantErr: true,
			errMsg:  "blocked",
		},
		{
			name:    "block reboot",
			command: "reboot",
			args:    nil,
			wantErr: true,
			errMsg:  "blocked",
		},
		{
			name:    "block shutdown",
			command: "shutdown",
			args:    []string{"-h", "now"},
			wantErr: true,
			errMsg:  "blocked",
		},
		{
			name:    "allow echo",
			command: "echo",
			args:    []string{"hello"},
			wantErr: false,
		},
		{
			name:    "allow ls",
			command: "ls",
			args:    []string{"/tmp"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sandbox.Execute(ctx, tt.command, tt.args, sandbox.AllowedPaths())
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errMsg)) {
					t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestBubblewrap_PathTraversal validates that path traversal attempts are rejected.
func TestBubblewrap_PathTraversal(t *testing.T) {
	skipIfNoBwrap(t)

	tmpDir, err := os.MkdirTemp("", "harvey-bwrap-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sandbox := NewBubblewrapSandbox([]string{tmpDir})
	ctx := context.Background()

	// Create a file inside the allowed dir for safe commands
	allowedFile := filepath.Join(tmpDir, "allowed.txt")
	if err := os.WriteFile(allowedFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "cat allowed file",
			command: "cat",
			args:    []string{allowedFile},
			wantErr: false,
		},
		{
			name:    "cat file outside allowed dir (traversal)",
			command: "cat",
			args:    []string{filepath.Join(tmpDir, "..", "..", "etc", "passwd")},
			wantErr: true,
		},
		{
			name:    "cat direct system file",
			command: "cat",
			args:    []string{"/etc/passwd"},
			wantErr: true,
		},
		{
			name:    "echo to allowed file",
			command: "echo",
			args:    []string{"hello"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sandbox.Execute(ctx, tt.command, tt.args, sandbox.AllowedPaths())
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for path traversal, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestBubblewrap_SafeCommands validates that safe commands execute successfully.
func TestBubblewrap_SafeCommands(t *testing.T) {
	skipIfNoBwrap(t)

	tmpDir, err := os.MkdirTemp("", "harvey-bwrap-safe")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sandbox := NewBubblewrapSandbox([]string{tmpDir})
	ctx := context.Background()

	tests := []struct {
		name      string
		command   string
		args      []string
		wantError bool
		contains  string
	}{
		{
			name:      "echo hello",
			command:   "echo",
			args:      []string{"hello", "world"},
			wantError: false,
			contains:  "hello world",
		},
		{
			name:      "ls allowed dir",
			command:   "ls",
			args:      []string{tmpDir},
			wantError: false,
		},
		{
			name:      "cat allowed file",
			command:   "cat",
			args:      []string{filepath.Join(tmpDir, "test.txt")},
			wantError: true, // file doesn't exist, will fail but not due to blocklist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := sandbox.Execute(ctx, tt.command, tt.args, sandbox.AllowedPaths())
			if !tt.wantError {
				if err != nil {
					t.Errorf("unexpected error for %s: %v", tt.command, err)
				}
				if tt.contains != "" && !strings.Contains(output, tt.contains) {
					t.Errorf("expected output to contain %q, got %q", tt.contains, output)
				}
			}
		})
	}
}

// TestBubblewrap_AllowedPaths verifies AllowedPaths returns the configured paths.
func TestBubblewrap_AllowedPaths(t *testing.T) {
	paths := []string{"/tmp", "/home/user/.config"}
	sandbox := NewBubblewrapSandbox(paths)
	got := sandbox.AllowedPaths()

	if len(got) != len(paths) {
		t.Errorf("expected %d paths, got %d", len(paths), len(got))
	}
}

// TestLandlock_SkipWithoutSupport skips Landlock tests when not running with
// appropriate permissions or kernel support.
func TestLandlock_SkipWithoutSupport(t *testing.T) {
	s := NewLandlockSandbox([]string{"/tmp"})
	err := s.Apply()
	if err != nil {
		t.Skipf("Landlock not available: %v", err)
	}
}

// TestLandlock_PathValidation tests the path validation logic without requiring
// Landlock enforcement (which needs root/CAP_SYS_ADMIN).
func TestLandlock_PathValidation(t *testing.T) {
	s := NewLandlockSandbox([]string{"/tmp"})

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "path within /tmp",
			path:    "/tmp/test.txt",
			wantErr: false,
		},
		{
			name:    "path outside allowed dirs",
			path:    "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "path traversal attempt",
			path:    "/tmp/../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "nested path within allowed",
			path:    "/tmp/harvey/subdir/file",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.validatePath(tt.path)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for path %q, got nil", tt.path)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for path %q: %v", tt.path, err)
			}
		})
	}
}

// TestLandlock_IsPathAllowed tests the path containment logic.
func TestLandlock_IsPathAllowed(t *testing.T) {
	s := NewLandlockSandbox([]string{"/home/user/.config"})

	tests := []struct {
		name    string
		path    string
		allowed bool
	}{
		{
			name:    "exact allowed dir",
			path:    "/home/user/.config",
			allowed: true,
		},
		{
			name:    "file in allowed dir",
			path:    "/home/user/.config/hypr/hyprland.conf",
			allowed: true,
		},
		{
			name:    "sibling of allowed dir",
			path:    "/home/user/Documents",
			allowed: false,
		},
		{
			name:    "system path",
			path:    "/etc",
			allowed: false,
		},
		{
			name:    "traversal from allowed dir",
			path:    "/home/user/.config/../../../etc/passwd",
			allowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.isPathAllowed(tt.path)
			if got != tt.allowed {
				t.Errorf("isPathAllowed(%q) = %v, want %v", tt.path, got, tt.allowed)
			}
		})
	}
}

// TestBubblewrap_ValidatePaths tests path normalization and traversal detection.
func TestBubblewrap_ValidatePaths(t *testing.T) {
	s := NewBubblewrapSandbox([]string{"/home/user/.config"})

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "clean path in allowed dir",
			path:    "/home/user/.config/hypr/hyprland.conf",
			wantErr: false,
		},
		{
			name:    "path with dots (but still within)",
			path:    "/home/user/.config/hypr/./hyprland.conf",
			wantErr: false,
		},
		{
			name:    "traversal to /etc",
			path:    "/home/user/.config/../../../etc",
			wantErr: true,
		},
		{
			name:    "direct system path",
			path:    "/etc/shadow",
			wantErr: true,
		},
		{
			name:    "root path",
			path:    "/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.validatePaths([]string{tt.path})
			if tt.wantErr && err == nil {
				t.Errorf("expected error for path %q, got nil", tt.path)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for path %q: %v", tt.path, err)
			}
		})
	}
}
