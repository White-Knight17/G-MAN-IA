// Package application contains the use-case orchestrators that bridge
// the domain interfaces with the TUI. It implements the ReAct agent loop,
// tool execution with permission enforcement, and grant lifecycle management.
//
// Dependency rule: application depends on domain interfaces, NOT on
// infrastructure implementations. Infrastructure adapters are injected
// via constructors.
package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gentleman/programas/harvey/internal/domain"
)

// textCommand maps a lightweight text-based command (e.g. "READ") to
// a domain.Tool name (e.g. "read_file") and whether content follows
// on subsequent lines (WRITE, CHECK).
type textCommand struct {
	toolName   string // domain tool name (e.g. "read_file")
	hasContent bool   // whether content lines follow (ended by END)
}

// commandMap translates lightweight text commands to domain tool names.
// Commands are matched case-insensitively. Content-bearing commands
// (WRITE, CHECK) collect lines until an END marker is found.
var commandMap = map[string]textCommand{
	"READ":   {toolName: "read_file", hasContent: false},
	"WRITE":  {toolName: "write_file", hasContent: true},
	"LIST":   {toolName: "list_dir", hasContent: false},
	"RUN":    {toolName: "run_command", hasContent: false},
	"CHECK":  {toolName: "check_syntax", hasContent: true},
	"SEARCH": {toolName: "search_wiki", hasContent: false},
}

// ToolExecutor parses text-based tool commands from LLM responses, performs
// permission checks, and routes to the correct domain.Tool implementation.
//
// Responsibilities:
//   - Parse lightweight text commands (READ, WRITE, LIST, RUN, CHECK, SEARCH)
//   - Permission validation via domain.PermissionRepository
//   - Route to registered tool implementations
//   - Enforce 30-second per-tool timeout
//   - Return structured domain.ToolResult
type ToolExecutor struct {
	tools     []domain.Tool
	sandbox   domain.Sandbox
	perms     domain.PermissionRepository
	toolIndex map[string]domain.Tool // lowercase name → tool
}

// NewToolExecutor creates a ToolExecutor with the given dependencies.
// The toolIndex is built by lowercasing each tool's Name() for case-insensitive
// matching — this handles the llama3.2:3b case-sensitivity issue discovered
// during model verification.
func NewToolExecutor(tools []domain.Tool, sandbox domain.Sandbox, perms domain.PermissionRepository) *ToolExecutor {
	index := make(map[string]domain.Tool, len(tools))
	for _, t := range tools {
		index[strings.ToLower(t.Name())] = t
	}
	return &ToolExecutor{
		tools:     tools,
		sandbox:   sandbox,
		perms:     perms,
		toolIndex: index,
	}
}

// Execute parses an LLM response for text-based tool commands, performs
// permission checks, and routes the call to the appropriate domain.Tool.
//
// The response may contain conversational text mixed with commands.
// Commands are detected by line-starting keywords (READ:, WRITE:, etc.).
// If no command is found, the response is returned as-is (conversational).
//
// For WRITE and CHECK, content is collected from subsequent lines until
// an END marker is found on its own line.
//
// Errors:
//   - Unknown tool name returns an error ToolResult
//   - Permission denied returns an error ToolResult
//   - Context cancellation returns the error
func (e *ToolExecutor) Execute(ctx context.Context, session *domain.Session, response string) (domain.ToolResult, error) {
	lines := strings.Split(response, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		cmd, arg := parseCommand(line)
		if cmd == "" {
			continue
		}

		cmdUpper := strings.ToUpper(cmd)
		cmdInfo, ok := commandMap[cmdUpper]
		if !ok {
			continue
		}

		// Build parameter map depending on the command type
		params := e.buildParams(cmdUpper, arg, lines, i+1)

		return e.executeDomainTool(ctx, session, cmdInfo.toolName, params)
	}

	// No command found — treat as conversational response
	return domain.ToolResult{Success: true, Output: response}, nil
}

// buildParams constructs the parameter map for a text command based on
// its type. Content-bearing commands (WRITE, CHECK) collect text until END.
func (e *ToolExecutor) buildParams(cmdUpper string, arg string, lines []string, startIdx int) map[string]string {
	switch cmdUpper {
	case "READ":
		return map[string]string{"path": arg}
	case "WRITE":
		return map[string]string{
			"path":    arg,
			"content": collectContent(lines, startIdx),
		}
	case "LIST":
		return map[string]string{"path": arg}
	case "RUN":
		return map[string]string{"command": arg}
	case "CHECK":
		return map[string]string{
			"filetype": arg,
			"content":  collectContent(lines, startIdx),
		}
	case "SEARCH":
		return map[string]string{"query": arg}
	default:
		return map[string]string{}
	}
}

// executeDomainTool looks up a domain tool by name, checks permissions,
// and executes it with a 30-second timeout.
func (e *ToolExecutor) executeDomainTool(ctx context.Context, session *domain.Session, toolName string, params map[string]string) (domain.ToolResult, error) {
	// Find tool (case-insensitive)
	tool, ok := e.toolIndex[strings.ToLower(toolName)]
	if !ok {
		err := fmt.Errorf("unknown tool: %q (available: %s)", toolName, e.availableToolNames())
		return domain.ToolResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Check permissions
	if !e.checkPermission(tool.Name(), params) {
		err := fmt.Errorf("permission denied for %s", tool.Name())
		return domain.ToolResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, params)
	if err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("execution failed: %v", err),
		}, err
	}

	return result, nil
}

// parseCommand extracts a command keyword and its argument from a line.
// The line must start with a known keyword followed by ": " (colon + space).
// Returns ("READ", "/path/to/file") for "READ: /path/to/file".
// Returns ("", "") if the line doesn't match the command pattern.
func parseCommand(line string) (cmd string, arg string) {
	parts := strings.SplitN(line, ": ", 2)
	if len(parts) != 2 {
		return "", ""
	}
	cmd = strings.TrimSpace(parts[0])
	arg = strings.TrimSpace(parts[1])

	// Validate it's a known command (case-insensitive)
	if _, ok := commandMap[strings.ToUpper(cmd)]; !ok {
		return "", ""
	}

	return cmd, arg
}

// collectContent collects lines from the given start index until an END
// marker is found on its own line. The content lines are joined with
// newlines and trimmed.
func collectContent(lines []string, start int) string {
	var contentLines []string
	for j := start; j < len(lines); j++ {
		trimmed := strings.TrimSpace(lines[j])
		if trimmed == "END" {
			break
		}
		contentLines = append(contentLines, lines[j])
	}
	return strings.TrimSpace(strings.Join(contentLines, "\n"))
}

// checkPermission validates that a tool has the required permissions.
// Write operations (write_file) require rw grants.
// Read operations (read_file, list_dir, check_syntax) require ro grants.
// Non-filesystem operations (run_command, search_wiki) need no grant.
func (e *ToolExecutor) checkPermission(toolName string, params map[string]string) bool {
	path, hasPath := params["path"]
	if !hasPath {
		// No path parameter — no permission check needed
		return true
	}

	toolLower := strings.ToLower(toolName)
	switch toolLower {
	case "write_file":
		return e.perms.Check(path, domain.PermissionWrite)
	default:
		// read_file, list_dir, check_syntax — all read operations
		return e.perms.Check(path, domain.PermissionRead)
	}
}

// availableToolNames returns a comma-separated list of tool names for error messages.
func (e *ToolExecutor) availableToolNames() string {
	names := make([]string, 0, len(e.tools))
	for _, t := range e.tools {
		names = append(names, t.Name())
	}
	return strings.Join(names, ", ")
}
