package domain

import "context"

// Sandbox provides a defense-in-depth execution boundary for tool operations.
// All file and command operations from tools MUST go through the sandbox
// to prevent unauthorized access outside allowed paths.
//
// The sandbox uses two layers:
//   1. Path validation: resolves and verifies paths are within allowedPaths
//      before any operation.
//   2. Syscall-level restriction: Landlock or Bubblewrap to enforce
//      filesystem isolation at the kernel level.
//
// Implementations: LandlockSandbox, BubblewrapSandbox.
type Sandbox interface {
	// Execute runs a command within the sandbox with the given arguments.
	// The allowedPaths parameter specifies which directories the command
	// is permitted to access. Returns stdout output or an error.
	//
	// For LandlockSandbox: allowedPaths is checked against pre-registered
	// Landlock rules applied at startup (immutable after LandlockRestrictSelf).
	//
	// For BubblewrapSandbox: allowedPaths are mapped as --ro-bind or --bind
	// mounts in the bwrap container.
	Execute(ctx context.Context, command string, args []string, allowedPaths []string) (string, error)

	// AllowedPaths returns the set of paths currently permitted by this sandbox.
	// For Landlock, this reflects the rules registered before LandlockRestrictSelf.
	AllowedPaths() []string
}
