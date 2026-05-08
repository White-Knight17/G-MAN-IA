package domain

import "context"

// Tool represents a capability that the agent can invoke during a ReAct loop.
// Each tool has a name, a description for the LLM system prompt, an XML schema
// that the LLM must follow when calling it, and an Execute method that performs
// the actual work within the sandbox.
//
// Implementations include: FilesystemTool (read_file, write_file, list_dir),
// CommandTool (run_command, check_syntax), and WikiClient (search_docs).
type Tool interface {
	// Name returns the tool identifier used in <name> XML tags.
	// Must be lowercase, underscore-separated (e.g., "read_file", "list_dir").
	Name() string

	// Description returns a human-readable explanation of what the tool does,
	// suitable for injection into the LLM system prompt.
	Description() string

	// SchemaXML returns the XML schema definition that the LLM must follow
	// when constructing a <tool_call> block for this tool.
	// Example: "<tool_call><name>read_file</name><path>/absolute/path</path></tool_call>"
	SchemaXML() string

	// Execute performs the tool's action with the given parameters.
	// The params map contains parameter names mapped from the XML block
	// (e.g., {"path": "/home/user/.config/hypr/hyprland.conf"}).
	// Returns a ToolResult indicating success or failure.
	Execute(ctx context.Context, params map[string]string) (ToolResult, error)
}

// ToolResult captures the outcome of a tool execution.
// On success, Output contains the result and Error is empty.
// On failure, Error contains the error description.
type ToolResult struct {
	Success bool   // whether the tool executed successfully
	Output  string // result content on success
	Error   string // error description on failure
}
