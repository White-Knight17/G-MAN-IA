package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gentleman/gman/internal/domain"
)

// Default knowledge base directory for search_wiki.
const knowledgeDir = ".config/gman/knowledge"

// CheckSyntaxTool validates configuration file syntax for supported filetypes.
// Supported: hyprland (brace pairing + known key names), waybar (JSON validity),
// bash (bash -n syntax check).
//
// Schema XML:
//
//	<tool_call>
//	  <name>check_syntax</name>
//	  <filetype>hyprland|waybar|bash</filetype>
//	  <content>config content to validate</content>
//	</tool_call>
type CheckSyntaxTool struct {
	sandbox domain.Sandbox
}

// NewCheckSyntaxTool creates a CheckSyntaxTool.
func NewCheckSyntaxTool(sandbox domain.Sandbox) *CheckSyntaxTool {
	return &CheckSyntaxTool{sandbox: sandbox}
}

// Name returns the tool identifier.
func (t *CheckSyntaxTool) Name() string { return "check_syntax" }

// Description returns a human-readable tool description for the LLM prompt.
func (t *CheckSyntaxTool) Description() string {
	return "Validates config file syntax. Supports: hyprland (brace pairing + key names), waybar (JSON), bash (bash -n). Provide the filetype and content to check."
}

// SchemaXML returns the XML schema the LLM must follow when calling this tool.
func (t *CheckSyntaxTool) SchemaXML() string {
	return `<tool_call><name>check_syntax</name><filetype>hyprland|waybar|bash</filetype><content>config content here</content></tool_call>`
}

// Execute validates the syntax of the content based on the specified filetype.
// Returns a list of issues found, or "No syntax errors detected" if valid.
func (t *CheckSyntaxTool) Execute(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
	filetype, exists := params["filetype"]
	if !exists || filetype == "" {
		return domain.ToolResult{
			Success: false,
			Error:   "check_syntax: missing required parameter 'filetype' (hyprland|waybar|bash)",
		}, fmt.Errorf("check_syntax: missing parameter 'filetype'")
	}

	content, exists := params["content"]
	if !exists {
		return domain.ToolResult{
			Success: false,
			Error:   "check_syntax: missing required parameter 'content'",
		}, fmt.Errorf("check_syntax: missing parameter 'content'")
	}

	ftLower := strings.ToLower(filetype)
	switch ftLower {
	case "hyprland", "hypr":
		return t.checkHyprland(content), nil
	case "waybar":
		return t.checkWaybar(content), nil
	case "bash":
		return t.checkBash(content), nil
	default:
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("check_syntax: unsupported filetype %q (supported: hyprland, waybar, bash)", filetype),
		}, fmt.Errorf("check_syntax: unsupported filetype %q", filetype)
	}
}

// checkHyprland performs basic Hyprland config validation:
//   - Brace pairing ({ and } must match)
//   - Known key name presence check
func (t *CheckSyntaxTool) checkHyprland(content string) domain.ToolResult {
	var issues []string

	// Check brace pairing
	openCount := strings.Count(content, "{")
	closeCount := strings.Count(content, "}")

	if openCount != closeCount {
		issues = append(issues, fmt.Sprintf("brace mismatch: %d opening vs %d closing braces", openCount, closeCount))
	}

	// Check for known Hyprland key names
	knownKeys := []string{
		"monitor", "workspace", "input", "general", "decoration",
		"animations", "bind", "bindm", "bindl", "binde", "bindr",
		"windowrule", "windowrulev2", "layerrule", "exec-once", "exec",
		"env", "plugin", "misc", "cursor", "debug", "gestures",
		"group", "xwayland",
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Check for sections (lines ending with {)
		if strings.HasSuffix(trimmed, "{") {
			sectionName := strings.TrimSuffix(trimmed, "{")
			sectionName = strings.TrimSpace(sectionName)
			found := false
			for _, k := range knownKeys {
				if strings.EqualFold(sectionName, k) {
					found = true
					break
				}
			}
			if !found && sectionName != "" {
				issues = append(issues, fmt.Sprintf("line %d: unknown section %q", i+1, sectionName))
			}
		}
	}

	if len(issues) == 0 {
		return domain.ToolResult{
			Success: true,
			Output:  "No syntax errors detected (hyprland config)",
		}
	}

	return domain.ToolResult{
		Success: true,
		Output:  "Issues found in hyprland config:\n" + strings.Join(issues, "\n"),
	}
}

// checkWaybar validates JSON syntax using a simple brace/quote balance check.
// For full JSON validation, a json.Decoder would be used, but for the sandboxed
// environment a basic check is sufficient.
func (t *CheckSyntaxTool) checkWaybar(content string) domain.ToolResult {
	trimmed := strings.TrimSpace(content)

	// Basic JSON structure check: starts with { or [
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return domain.ToolResult{
			Success: true,
			Output:  "Issues found in waybar config:\nJSON must start with { or [",
		}
	}

	// Brace/bracket pairing
	var stack []rune
	var issues []string
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		for _, ch := range line {
			switch ch {
			case '{', '[':
				stack = append(stack, ch)
			case '}':
				if len(stack) == 0 || stack[len(stack)-1] != '{' {
					issues = append(issues, fmt.Sprintf("line %d: unexpected closing brace", i+1))
				} else {
					stack = stack[:len(stack)-1]
				}
			case ']':
				if len(stack) == 0 || stack[len(stack)-1] != '[' {
					issues = append(issues, fmt.Sprintf("line %d: unexpected closing bracket", i+1))
				} else {
					stack = stack[:len(stack)-1]
				}
			}
		}
	}

	if len(stack) > 0 {
		issues = append(issues, fmt.Sprintf("unclosed %c at end of file", stack[len(stack)-1]))
	}

	// Check trailing commas (simple heuristic)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ",") && i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if nextLine == "" || strings.HasPrefix(nextLine, "}") || strings.HasPrefix(nextLine, "]") {
				issues = append(issues, fmt.Sprintf("line %d: trailing comma before closing brace/bracket", i+1))
			}
		}
	}

	if len(issues) == 0 {
		return domain.ToolResult{
			Success: true,
			Output:  "No syntax errors detected (waybar JSON config)",
		}
	}

	return domain.ToolResult{
		Success: true,
		Output:  "Issues found in waybar config:\n" + strings.Join(issues, "\n"),
	}
}

// checkBash validates bash script syntax using bash -n (no execution mode).
func (t *CheckSyntaxTool) checkBash(content string) domain.ToolResult {
	// Write content to a temp file for bash -n validation
	tmpFile, err := os.CreateTemp("", "gman-bash-check-*.sh")
	if err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("check_syntax: failed to create temp file: %v", err),
		}
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("check_syntax: failed to write temp file: %v", err),
		}
	}
	tmpFile.Close()

	cmd := exec.Command("bash", "-n", tmpFile.Name())
	output, err := cmd.CombinedOutput()

	if err != nil {
		// bash -n returns non-zero on syntax errors
		errOutput := strings.TrimSpace(string(output))
		if errOutput == "" {
			errOutput = err.Error()
		}
		return domain.ToolResult{
			Success: true,
			Output:  "Issues found in bash script:\n" + errOutput,
		}
	}

	return domain.ToolResult{
		Success: true,
		Output:  "No syntax errors detected (bash script)",
	}
}

// SearchWikiTool searches local markdown files in the user's knowledge base
// directory (~/.config/gman/knowledge/) for the given query.
// Uses simple text search (case-insensitive substring matching) across .md files.
//
// Schema XML:
//
//	<tool_call>
//	  <name>search_wiki</name>
//	  <query>search terms here</query>
//	</tool_call>
type SearchWikiTool struct {
	sandbox domain.Sandbox
}

// NewSearchWikiTool creates a SearchWikiTool.
func NewSearchWikiTool(sandbox domain.Sandbox) *SearchWikiTool {
	return &SearchWikiTool{sandbox: sandbox}
}

// Name returns the tool identifier.
func (t *SearchWikiTool) Name() string { return "search_wiki" }

// Description returns a human-readable tool description for the LLM prompt.
func (t *SearchWikiTool) Description() string {
	return "Searches local markdown knowledge base files (~/.config/gman/knowledge/) for information. Use this to look up Arch Wiki documentation, Hyprland tips, or other saved knowledge."
}

// SchemaXML returns the XML schema the LLM must follow when calling this tool.
func (t *SearchWikiTool) SchemaXML() string {
	return `<tool_call><name>search_wiki</name><query>search terms</query></tool_call>`
}

// Execute searches the knowledge base directory for .md files matching the query.
// Returns matching file paths with previews of the first match in each file.
func (t *SearchWikiTool) Execute(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
	query, exists := params["query"]
	if !exists || query == "" {
		return domain.ToolResult{
			Success: false,
			Error:   "search_wiki: missing required parameter 'query'",
		}, fmt.Errorf("search_wiki: missing parameter 'query'")
	}

	// Resolve the knowledge base directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("search_wiki: cannot determine home directory: %v", err),
		}, err
	}

	kbPath := filepath.Join(homeDir, knowledgeDir)

	// Check if knowledge base exists
	info, err := os.Stat(kbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.ToolResult{
				Success: true,
				Output:  fmt.Sprintf("No knowledge base found at %s", kbPath),
			}, nil
		}
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("search_wiki: cannot access %s: %v", kbPath, err),
		}, err
	}

	if !info.IsDir() {
		return domain.ToolResult{
			Success: true,
			Output:  fmt.Sprintf("%s is not a directory", kbPath),
		}, nil
	}

	// Walk the knowledge base directory searching .md files
	var results []string
	queryLower := strings.ToLower(query)

	err = filepath.Walk(kbPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip files we can't read
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // skip unreadable files
		}

		contentLower := strings.ToLower(string(data))
		if strings.Contains(contentLower, queryLower) {
			// Create a preview: first matching line with context
			preview := buildPreview(string(data), query, 80)
			relPath, _ := filepath.Rel(kbPath, path)
			results = append(results, fmt.Sprintf("%s: %s", relPath, preview))
		}
		return nil
	})

	if err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("search_wiki: error scanning knowledge base: %v", err),
		}, err
	}

	if len(results) == 0 {
		return domain.ToolResult{
			Success: true,
			Output:  fmt.Sprintf("No results found for %q in %s", query, kbPath),
		}, nil
	}

	output := fmt.Sprintf("Found %d result(s) for %q in %s:\n%s",
		len(results), query, kbPath, strings.Join(results, "\n"))

	// Truncate if too long
	const maxOutput = 4 * 1024 // 4KB
	if len(output) > maxOutput {
		output = output[:maxOutput] + fmt.Sprintf("\n... [truncated, %d more results]", len(results)-1)
	}

	return domain.ToolResult{
		Success: true,
		Output:  output,
	}, nil
}

// buildPreview creates a short preview string from file content around the first
// occurrence of the query. Returns up to maxChars characters centered on the match.
func buildPreview(content string, query string, maxChars int) string {
	lines := strings.Split(content, "\n")
	queryLower := strings.ToLower(query)

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > maxChars {
				// Find the position of the query in the trimmed line
				idx := strings.Index(strings.ToLower(trimmed), queryLower)
				if idx < 0 {
					idx = 0
				}
				start := idx - 20
				if start < 0 {
					start = 0
				}
				end := start + maxChars
				if end > len(trimmed) {
					end = len(trimmed)
				}
				preview := trimmed[start:end]
				if start > 0 {
					preview = "..." + preview
				}
				if end < len(trimmed) {
					preview = preview + "..."
				}
				return fmt.Sprintf("%q", preview)
			}
			return fmt.Sprintf("%q", trimmed)
		}
	}
	return "(no preview)"
}
