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
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/gentleman/programas/harvey/internal/domain"
)

// ToolExecutor parses XML <tool_call> blocks from LLM responses, performs
// permission checks, and routes to the correct domain.Tool implementation.
//
// Responsibilities:
//   - Parse XML tool calls (case-insensitive tool name matching)
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

// toolCallXML represents the parsed structure of a <tool_call> XML block.
// The Name element is explicit; all other child elements are captured
// in Params and converted to a map later.
type toolCallXML struct {
	Name   string     `xml:"name"`
	Params []paramXML `xml:",any"`
}

// paramXML captures any XML element inside <tool_call> other than <name>.
// XMLName identifies the element name (e.g., "path", "cmd", "query").
type paramXML struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

// Execute parses a <tool_call> XML string, performs permission checks,
// and routes the call to the appropriate domain.Tool implementation.
//
// Steps:
//   1. Parse XML to extract tool name (case-insensitive) and parameters
//   2. Look up the tool by name
//   3. Check permissions via domain.PermissionRepository
//   4. Execute the tool with a 30-second timeout
//   5. Return the ToolResult
//
// Errors:
//   - Malformed XML returns an error ToolResult
//   - Unknown tool name returns an error ToolResult
//   - Permission denied returns an error ToolResult
//   - Context cancellation returns the error
func (e *ToolExecutor) Execute(ctx context.Context, session *domain.Session, toolCallXML string) (domain.ToolResult, error) {
	// Step 1: Parse XML
	name, params, err := e.parseToolCall(toolCallXML)
	if err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("parse error: %v", err),
		}, err
	}

	// Step 2: Find tool (case-insensitive)
	tool, ok := e.toolIndex[strings.ToLower(name)]
	if !ok {
		err := fmt.Errorf("unknown tool: %q (available: %s)", name, e.availableToolNames())
		return domain.ToolResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Step 3: Check permissions
	if !e.checkPermission(tool.Name(), params) {
		err := fmt.Errorf("permission denied for %s", tool.Name())
		return domain.ToolResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Step 4: Execute with timeout
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

// parseToolCall extracts the tool name and parameters from a <tool_call> XML string.
// Tool name matching is case-insensitive (llama3.2:3b workaround).
func (e *ToolExecutor) parseToolCall(xmlStr string) (string, map[string]string, error) {
	// Trim any text before/after the XML block
	xmlStr = strings.TrimSpace(xmlStr)

	oi := strings.Index(xmlStr, "<tool_call>")
	ci := strings.Index(xmlStr, "</tool_call>")
	if oi == -1 || ci == -1 || oi >= ci {
		return "", nil, fmt.Errorf("no valid <tool_call> block found")
	}

	block := xmlStr[oi : ci+len("</tool_call>")]

	var call toolCallXML
	decoder := xml.NewDecoder(strings.NewReader(block))
	if err := decoder.Decode(&call); err != nil {
		return "", nil, fmt.Errorf("xml decode: %w", err)
	}

	name := strings.TrimSpace(call.Name)
	if name == "" {
		return "", nil, fmt.Errorf("tool name is empty")
	}

	params := make(map[string]string, len(call.Params))
	for _, p := range call.Params {
		// Use lowercase param name for consistency
		key := strings.ToLower(p.XMLName.Local)
		params[key] = strings.TrimSpace(p.Value)
	}

	return name, params, nil
}

// checkPermission validates that a tool has the required permissions.
// Write operations (write_file) require rw grants.
// Read operations (read_file, list_dir, check_syntax) require ro grants.
// Non-filesystem operations (run_command, search_docs) need no grant.
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
