package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gentleman/programas/harvey/internal/domain"
)

// allowlist defines the set of commands that may be executed through the sandbox.
// Each entry is the base command name (e.g., "grep", "ls").
var allowlist = map[string]bool{
	"grep":       true,
	"ls":         true,
	"cat":        true,
	"pacman":     true, // only query mode allowed (see pacmanFlags check)
	"systemctl":  true, // only --user flag allowed
	"hyprctl":    true,
	"waybar":     true,
	"find":       true,
	"which":      true,
	"echo":       true,
	"journalctl": true, // only --user flag allowed
}

// flagBlocklist defines command+flag combinations that are blocked.
// For example, "pacman --sync" is blocked because it would install packages.
var flagBlocklist = map[string][]string{
	"pacman":    {"--sync", "-S", "--remove", "-R"},
	"systemctl": {"--global", "--system", "enable", "disable", "start", "stop", "restart", "mask", "unmask"},
}

// CommandTool executes allowlisted shell commands through the sandbox.
// The command and its flags/arguments are provided in the "command" param.
// The tool validates the command against the allowlist and flag blocklist
// before dispatching it to the sandbox for execution.
//
// Schema XML:
//
//	<tool_call>
//	  <name>run_command</name>
//	  <command>ls -la /home/user/.config</command>
//	</tool_call>
type CommandTool struct {
	sandbox     domain.Sandbox
	allowedDirs []string
}

// NewCommandTool creates a CommandTool with the given sandbox and allowed directories.
func NewCommandTool(sandbox domain.Sandbox, allowedDirs []string) *CommandTool {
	return &CommandTool{
		sandbox:     sandbox,
		allowedDirs: allowedDirs,
	}
}

// Name returns the tool identifier.
func (t *CommandTool) Name() string { return "run_command" }

// Description returns a human-readable tool description for the LLM prompt.
func (t *CommandTool) Description() string {
	return "Runs an allowlisted shell command inside the sandbox. Available: grep, ls, cat, pacman (query only), systemctl --user, hyprctl, waybar, find, which, echo."
}

// SchemaXML returns the XML schema the LLM must follow when calling this tool.
func (t *CommandTool) SchemaXML() string {
	return `<tool_call><name>run_command</name><command>command and arguments here</command></tool_call>`
}

// Execute runs the command specified in params["command"] through the sandbox.
// The command is split into the executable name and its arguments, validated
// against the allowlist and flag blocklist, then dispatched to the sandbox.
func (t *CommandTool) Execute(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
	rawCommand, exists := params["command"]
	if !exists || rawCommand == "" {
		return domain.ToolResult{
			Success: false,
			Error:   "run_command: missing required parameter 'command'",
		}, fmt.Errorf("run_command: missing parameter 'command'")
	}

	// Split command into executable and args
	parts := strings.Fields(rawCommand)
	if len(parts) == 0 {
		return domain.ToolResult{
			Success: false,
			Error:   "run_command: empty command",
		}, fmt.Errorf("run_command: empty command")
	}

	exe := parts[0]
	args := parts[1:]

	// Validate command is allowlisted
	baseExe := filepath.Base(exe)
	exeLower := strings.ToLower(baseExe)
	if !allowlist[exeLower] {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("run_command: command %q is not in the allowlist (allowed: grep, ls, cat, pacman, systemctl, hyprctl, waybar, find, which, echo)", exe),
		}, fmt.Errorf("run_command: command %q not allowed", exe)
	}

	// Validate flag restrictions
	if err := t.validateFlags(exeLower, args); err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("run_command: %v", err),
		}, err
	}

	// Special handling for systemctl: enforce --user flag
	if exeLower == "systemctl" {
		hasUser := false
		for _, a := range args {
			if a == "--user" {
				hasUser = true
				break
			}
		}
		if !hasUser {
			return domain.ToolResult{
				Success: false,
				Error:   "run_command: systemctl requires --user flag (system-wide operations are not allowed)",
			}, fmt.Errorf("run_command: systemctl requires --user")
		}
	}

	// Special handling for journalctl: enforce --user flag
	if exeLower == "journalctl" {
		hasUser := false
		for _, a := range args {
			if a == "--user" {
				hasUser = true
				break
			}
		}
		if !hasUser {
			return domain.ToolResult{
				Success: false,
				Error:   "run_command: journalctl requires --user flag (system-wide logs are not allowed)",
			}, fmt.Errorf("run_command: journalctl requires --user")
		}
	}

	// Execute through sandbox
	output, err := t.sandbox.Execute(ctx, exe, args, t.allowedDirs)
	if err != nil {
		return domain.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("run_command: execution failed: %v", err),
		}, err
	}

	return domain.ToolResult{
		Success: true,
		Output:  output,
	}, nil
}

// validateFlags checks the command and its arguments against the flag blocklist.
// For example, "pacman --sync" would be blocked because --sync is in the
// pacman flag blocklist.
func (t *CommandTool) validateFlags(exeLower string, args []string) error {
	blockedFlags, hasRestrictions := flagBlocklist[exeLower]
	if !hasRestrictions {
		return nil
	}

	for _, arg := range args {
		for _, blocked := range blockedFlags {
			if strings.EqualFold(arg, blocked) ||
				(strings.HasPrefix(arg, blocked) && (len(arg) == len(blocked) || arg[len(blocked)] == '=')) {
				return fmt.Errorf("flag %q is blocked for command %q (only query operations are allowed)", arg, exeLower)
			}
		}
	}

	return nil
}
