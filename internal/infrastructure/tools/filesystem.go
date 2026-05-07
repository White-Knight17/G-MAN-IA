// Package tools provides the six sandboxed dotfile tools that G-MAN's
// LLM agent can invoke during a ReAct loop. Each tool implements domain.Tool.
//
// Tools:
//   - ReadFileTool: reads file contents (max 10KB)
//   - WriteFileTool: writes file with .bak backup and returns diff summary
//   - ListDirTool: lists directory entries (non-recursive, names only)
//   - CommandTool: executes allowlisted shell commands through the sandbox
//   - CheckSyntaxTool: validates config file syntax (hyprland, waybar, bash)
//   - SearchWikiTool: searches local markdown knowledge base files
package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gentleman/gman/internal/domain"
)

// Common limits for filesystem tools.
const (
	// maxReadSize limits the number of bytes read_file will return.
	// Large configs should be truncated with a note.
	maxReadSize = 10 * 1024 // 10 KB
)

// ReadFileTool reads a file at the given path and returns its contents.
// Max 10KB per read. Path must be within allowed directories.
//
// Schema XML:
//
//	<tool_call>
//	  <name>read_file</name>
//	  <path>/absolute/path/to/file</path>
//	</tool_call>
type ReadFileTool struct {
	allowedDirs []string
	sandbox     domain.Sandbox
}

// NewReadFileTool creates a ReadFileTool with the given configuration.
// allowedDirs defines which directories this tool is permitted to read from.
func NewReadFileTool(allowedDirs []string, sandbox domain.Sandbox) *ReadFileTool {
	return &ReadFileTool{
		allowedDirs: allowedDirs,
		sandbox:     sandbox,
	}
}

// Name returns the tool identifier.
func (t *ReadFileTool) Name() string { return "read_file" }

// Description returns a human-readable tool description for the LLM prompt.
func (t *ReadFileTool) Description() string {
	return "Reads the contents of a file. Returns up to 10KB of text. Use this to inspect config files, scripts, or dotfiles."
}

// SchemaXML returns the XML schema the LLM must follow when calling this tool.
func (t *ReadFileTool) SchemaXML() string {
	return `<tool_call><name>read_file</name><path>/absolute/path/to/file</path></tool_call>`
}

// Execute reads the file at the path specified in params["path"].
// The path is validated to be within allowedDirs and the file must exist.
// Returns the file contents or an error.
func (t *ReadFileTool) Execute(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
	rawPath, exists := params["path"]
	if !exists || rawPath == "" {
		return domain.ToolResult{
			Success: false,
			Error:   "read_file: missing required parameter 'path'",
		}, fmt.Errorf("read_file: missing parameter 'path'")
	}

	// Validate path is within allowed dirs
	resolved, err := t.resolvePath(rawPath)
	if err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("read_file: %v", err),
		}, err
	}

	// Check file exists
	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			errMsg := fmt.Sprintf("file not found: %s", resolved)
			return domain.ToolResult{
				Success: false,
				Error:   errMsg,
			}, fmt.Errorf("%s", errMsg)
		}
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("read_file: cannot stat %s: %v", resolved, err),
		}, err
	}

	if info.IsDir() {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("read_file: %s is a directory, use list_dir instead", resolved),
		}, fmt.Errorf("read_file: path is a directory")
	}

	// Read file contents
	data, err := os.ReadFile(resolved)
	if err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("read_file: failed to read %s: %v", resolved, err),
		}, err
	}

	// Truncate if over maxReadSize
	content := string(data)
	if len(content) > maxReadSize {
		content = content[:maxReadSize] + fmt.Sprintf("\n... [truncated, %d bytes total]", len(data))
	}

	return domain.ToolResult{
		Success: true,
		Output:  content,
	}, nil
}

// WriteFileTool creates a .bak backup of the original file, writes new content,
// and returns a diff summary showing the changes.
//
// Schema XML:
//
//	<tool_call>
//	  <name>write_file</name>
//	  <path>/absolute/path/to/file</path>
//	  <content>new file contents here</content>
//	</tool_call>
type WriteFileTool struct {
	allowedDirs []string
	sandbox     domain.Sandbox
}

// NewWriteFileTool creates a WriteFileTool with the given configuration.
func NewWriteFileTool(allowedDirs []string, sandbox domain.Sandbox) *WriteFileTool {
	return &WriteFileTool{
		allowedDirs: allowedDirs,
		sandbox:     sandbox,
	}
}

// Name returns the tool identifier.
func (t *WriteFileTool) Name() string { return "write_file" }

// Description returns a human-readable tool description for the LLM prompt.
func (t *WriteFileTool) Description() string {
	return "Writes content to a file. Creates a .bak backup before writing. Returns a diff summary of changes. Requires write permission."
}

// SchemaXML returns the XML schema the LLM must follow when calling this tool.
func (t *WriteFileTool) SchemaXML() string {
	return `<tool_call><name>write_file</name><path>/absolute/path/to/file</path><content>new content</content></tool_call>`
}

// Execute writes content to the file at the path specified in params["path"].
// The original file is backed up as path.bak before the write.
// Returns a diff summary showing lines added, removed, and changed.
func (t *WriteFileTool) Execute(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
	rawPath, exists := params["path"]
	if !exists || rawPath == "" {
		return domain.ToolResult{
			Success: false,
			Error:   "write_file: missing required parameter 'path'",
		}, fmt.Errorf("write_file: missing parameter 'path'")
	}

	content, exists := params["content"]
	if !exists {
		return domain.ToolResult{
			Success: false,
			Error:   "write_file: missing required parameter 'content'",
		}, fmt.Errorf("write_file: missing parameter 'content'")
	}

	// Validate path is within allowed dirs
	resolved, err := t.resolvePath(rawPath)
	if err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("write_file: %v", err),
		}, err
	}

	// Check if it's a directory
	if info, err := os.Stat(resolved); err == nil && info.IsDir() {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("write_file: %s is a directory, cannot write", resolved),
		}, fmt.Errorf("write_file: path is a directory")
	}

	// Create .bak backup if the file exists
	var originalContent []byte
	if data, err := os.ReadFile(resolved); err == nil {
		originalContent = data
		bakPath := resolved + ".bak"
		if err := os.WriteFile(bakPath, data, 0644); err != nil {
			return domain.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("write_file: failed to create backup %s: %v", bakPath, err),
			}, err
		}
	}

	// Write new content
	if err := os.WriteFile(resolved, []byte(content), 0644); err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("write_file: failed to write %s: %v", resolved, err),
		}, err
	}

	// Compute diff summary
	diff := computeDiff(string(originalContent), content, resolved)

	return domain.ToolResult{
		Success: true,
		Output:  diff,
	}, nil
}

// ListDirTool lists directory contents (names only, non-recursive).
// Hidden files (starting with .) are excluded by default.
//
// Schema XML:
//
//	<tool_call>
//	  <name>list_dir</name>
//	  <path>/absolute/path/to/directory</path>
//	</tool_call>
type ListDirTool struct {
	allowedDirs []string
	sandbox     domain.Sandbox
}

// NewListDirTool creates a ListDirTool with the given configuration.
func NewListDirTool(allowedDirs []string, sandbox domain.Sandbox) *ListDirTool {
	return &ListDirTool{
		allowedDirs: allowedDirs,
		sandbox:     sandbox,
	}
}

// Name returns the tool identifier.
func (t *ListDirTool) Name() string { return "list_dir" }

// Description returns a human-readable tool description for the LLM prompt.
func (t *ListDirTool) Description() string {
	return "Lists the contents of a directory. Returns file and directory names (non-recursive, hidden files excluded)."
}

// SchemaXML returns the XML schema the LLM must follow when calling this tool.
func (t *ListDirTool) SchemaXML() string {
	return `<tool_call><name>list_dir</name><path>/absolute/path/to/directory</path></tool_call>`
}

// Execute lists the contents of the directory at the path in params["path"].
// Returns one entry per line, excluding hidden files (those starting with ".").
func (t *ListDirTool) Execute(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
	rawPath, exists := params["path"]
	if !exists || rawPath == "" {
		return domain.ToolResult{
			Success: false,
			Error:   "list_dir: missing required parameter 'path'",
		}, fmt.Errorf("list_dir: missing parameter 'path'")
	}

	// Validate path is within allowed dirs
	resolved, err := t.resolvePath(rawPath)
	if err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("list_dir: %v", err),
		}, err
	}

	entries, err := os.ReadDir(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("directory not found: %s", resolved),
			}, fmt.Errorf("list_dir: directory not found: %s", resolved)
		}
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("list_dir: cannot read %s: %v", resolved, err),
		}, err
	}

	var names []string
	for _, entry := range entries {
		// Exclude hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		suffix := ""
		if entry.IsDir() {
			suffix = "/"
		}
		names = append(names, entry.Name()+suffix)
	}

	output := strings.Join(names, "\n")
	if output == "" {
		output = "(empty directory)"
	}

	return domain.ToolResult{
		Success: true,
		Output:  output,
	}, nil
}

// pathHelpers shared by all filesystem tools.

// resolvePath normalizes and validates that a path is within the tool's allowedDirs.
// It handles path traversal detection, symlink resolution, and absolute path conversion.
func resolvePath(raw string, allowedDirs []string) (string, error) {
	cleaned := filepath.Clean(raw)

	// Resolve symlinks if possible
	if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
		cleaned = resolved
	}

	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path %q: %w", raw, err)
	}

	// Check path is within one of the allowed directories
	if !isWithinAllowedDirs(abs, allowedDirs) {
		return "", fmt.Errorf("path %q is outside allowed directories: %v", raw, allowedDirs)
	}

	return abs, nil
}

// isWithinAllowedDirs checks whether the given absolute path is within any
// of the allowed directories. Uses relative path comparison to prevent
// traversal attacks (e.g., /home/user/.config/../../../etc/passwd).
func isWithinAllowedDirs(absPath string, allowedDirs []string) bool {
	for _, allowed := range allowedDirs {
		allowedAbs, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(allowedAbs, absPath)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}

// computeDiff generates a human-readable diff summary between original and new content.
// Returns the diff as a string with line counts.
func computeDiff(original, new string, path string) string {
	origLines := strings.Split(original, "\n")
	newLines := strings.Split(new, "\n")

	// Handle empty original (new file)
	if original == "" {
		return fmt.Sprintf("Created new file %s (%d lines)", path, len(newLines))
	}

	added := len(newLines) - len(origLines)

	// Count changed lines (simple line-by-line comparison)
	changed := 0
	minLen := len(origLines)
	if len(newLines) < minLen {
		minLen = len(newLines)
	}
	for i := 0; i < minLen; i++ {
		if origLines[i] != newLines[i] {
			changed++
		}
	}

	// Lines only in the longer file count as changed
	remainingRemoved := 0
	if len(origLines) > len(newLines) {
		remainingRemoved = len(origLines) - len(newLines)
	}

	var parts []string
	if added > 0 {
		parts = append(parts, fmt.Sprintf("%d lines added", added))
	}
	if remainingRemoved > 0 || (original != "" && len(newLines) < len(origLines)) {
		parts = append(parts, fmt.Sprintf("%d lines removed", remainingRemoved))
	}
	if changed > 0 {
		parts = append(parts, fmt.Sprintf("%d lines changed", changed))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Wrote %s (no changes detected, %d lines)", path, len(newLines))
	}

	return fmt.Sprintf("Wrote %s: %s. Backup saved as %s.bak (%d lines → %d lines)",
		path, strings.Join(parts, ", "), path, len(origLines), len(newLines))
}

// resolvePath is a method wrapper for the tool structs.
func (t *ReadFileTool) resolvePath(raw string) (string, error) {
	return resolvePath(raw, t.allowedDirs)
}
func (t *WriteFileTool) resolvePath(raw string) (string, error) {
	return resolvePath(raw, t.allowedDirs)
}
func (t *ListDirTool) resolvePath(raw string) (string, error) {
	return resolvePath(raw, t.allowedDirs)
}
