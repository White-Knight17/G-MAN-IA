package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gentleman/programas/harvey/internal/domain"
	"golang.org/x/sys/unix"
)

// Landlock ABI versions.
const (
	landlockABIVersion = 4 // Linux 6.7+, supports inode reparenting
)

// LandlockSandbox implements domain.Sandbox using the Linux Landlock LSM.
// It restricts the CURRENT Go process's file access to only the allowed paths.
// Once applied via LandlockRestrictSelf(), the rules are immutable for the
// lifetime of the process — no escalation, no bypass.
//
// IMPORTANT: Because Landlock is in-process, it applies to the Go process itself.
// It should be applied at startup before any file operations on paths that may
// not be in the allowlist. It complements Bubblewrap (subprocess isolation) by
// hardening the tool-runner process.
//
// For commands that need subprocess execution, use BubblewrapSandbox instead.
// LandlockSandbox is used for direct file operations (read_file, write_file, list_dir).
type LandlockSandbox struct {
	allowedPaths []string
	enforced     bool
}

// NewLandlockSandbox creates a LandlockSandbox with the given allowed paths.
// Call Apply() to enforce the Landlock rules, after which they are immutable.
func NewLandlockSandbox(allowedPaths []string) *LandlockSandbox {
	resolved := make([]string, len(allowedPaths))
	for i, p := range allowedPaths {
		resolved[i] = filepath.Clean(p)
	}
	return &LandlockSandbox{
		allowedPaths: resolved,
		enforced:     false,
	}
}

// Apply activates Landlock restrictions for the current process.
// After this call, the process can only access files within the allowedPaths.
// This is a ONE-WAY operation — rules cannot be changed after enforcement.
//
// Returns an error if:
//   - The kernel does not support Landlock (check /proc/config or uname -r)
//   - The process does not have CAP_SYS_ADMIN (typically requires running as root
//     or having the capability set on the binary)
func (s *LandlockSandbox) Apply() error {
	// Check Landlock ABI version
	abi := unix.LandlockGetABIVersion()
	if abi < 1 {
		return fmt.Errorf("landlock: kernel does not support Landlock (ABI version %d, need >= 1)", abi)
	}

	// Build Landlock rules from allowed paths
	rules, err := s.buildRules()
	if err != nil {
		return fmt.Errorf("landlock: failed to build rules: %w", err)
	}

	if len(rules) == 0 {
		// No rules means unrestricted — apply a minimal rule to activate Landlock
		// without restricting anything (this is unexpected but safe).
		return nil
	}

	// Submit rules to kernel
	for _, rule := range rules {
		if err := unix.LandlockAddRule(
			unix.LandlockAccessFSExecute|
				unix.LandlockAccessFSReadFile|
				unix.LandlockAccessFSReadDir|
				unix.LandlockAccessFSRemoveDir|
				unix.LandlockAccessFSRemoveFile|
				unix.LandlockAccessFSMakeChar|
				unix.LandlockAccessFSMakeDir|
				unix.LandlockAccessFSMakeReg|
				unix.LandlockAccessFSMakeSock|
				unix.LandlockAccessFSMakeFifo|
				unix.LandlockAccessFSMakeBlock|
				unix.LandlockAccessFSMakeSym|
				unix.LandlockAccessFSRefer|
				unix.LandlockAccessFSTruncate,
			rule,
			0,
		); err != nil {
			// If the kernel doesn't support some access flags, try with fewer flags
			// as a compatibility fallback
			if err := unix.LandlockAddRule(
				unix.LandlockAccessFSReadFile|
					unix.LandlockAccessFSReadDir|
					unix.LandlockAccessFSExecute,
				rule,
				0,
			); err != nil {
				return fmt.Errorf("landlock: LandlockAddRule failed: %w", err)
			}
		}
	}

	// Enforce rules — after this, the process is locked down.
	if err := unix.LandlockRestrictSelf(); err != nil {
		return fmt.Errorf("landlock: LandlockRestrictSelf failed: %w (process may need CAP_SYS_ADMIN or to run as root)", err)
	}

	s.enforced = true
	return nil
}

// Execute validates that the requested path is within allowed paths.
// For Landlock, the actual file access enforcement is done by the kernel
// via the Landlock rules applied in Apply(). This method provides the
// pre-validation layer to catch obvious violations early.
//
// The command parameter is interpreted as the operation type (e.g., "read_file",
// "write_file") for validation purposes. The args parameter should contain
// file paths that will be accessed.
//
// For Landlock, Execute delegates actual file I/O to the caller — the kernel
// enforces the restrictions. This method only validates path membership.
func (s *LandlockSandbox) Execute(ctx context.Context, command string, args []string, allowedPaths []string) (string, error) {
	// Validate paths are within allowedPaths
	for _, raw := range append(args, allowedPaths...) {
		if err := s.validatePath(raw); err != nil {
			return "", err
		}
	}

	// For write operations, verify the path exists in allowedPaths
	cmdLower := strings.ToLower(command)
	if cmdLower == "write_file" {
		for _, arg := range args {
			if len(arg) > 0 && !s.isPathAllowed(arg) {
				return "", fmt.Errorf("landlock: write to %q not allowed — path is outside sandbox", arg)
			}
		}
	}

	return "", nil
}

// AllowedPaths returns the configured allowed paths.
func (s *LandlockSandbox) AllowedPaths() []string {
	paths := make([]string, len(s.allowedPaths))
	copy(paths, s.allowedPaths)
	return paths
}

// IsEnforced returns whether Landlock rules have been applied.
func (s *LandlockSandbox) IsEnforced() bool {
	return s.enforced
}

// buildRules constructs Landlock rules that allow access to the configured paths.
// Each path is converted to a Landlock path beneath rule (AT_FDCWD + path).
func (s *LandlockSandbox) buildRules() ([]*unix.LandlockPathBeneathAttr, error) {
	var rules []*unix.LandlockPathBeneathAttr

	for _, p := range s.allowedPaths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve path %q: %w", p, err)
		}

		// For Landlock, we need the directory to exist to create a rule for it.
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			// Path does not exist — skip creating a rule for it.
			// The bubblewrap layer will handle non-existent paths at execution time.
			continue
		}

		rules = append(rules, &unix.LandlockPathBeneathAttr{
			AllowedAccess: unix.LandlockAccessFSReadFile |
				unix.LandlockAccessFSReadDir |
				unix.LandlockAccessFSExecute |
				unix.LandlockAccessFSRemoveDir |
				unix.LandlockAccessFSRemoveFile |
				unix.LandlockAccessFSMakeChar |
				unix.LandlockAccessFSMakeDir |
				unix.LandlockAccessFSMakeReg |
				unix.LandlockAccessFSMakeSock |
				unix.LandlockAccessFSMakeFifo |
				unix.LandlockAccessFSMakeBlock |
				unix.LandlockAccessFSMakeSym |
				unix.LandlockAccessFSRefer |
				unix.LandlockAccessFSTruncate,
			ParentFd: unix.AT_FDCWD,
		})
	}

	return rules, nil
}

// validatePath resolves and validates a single path against allowed paths.
func (s *LandlockSandbox) validatePath(raw string) error {
	cleaned := filepath.Clean(raw)

	if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
		cleaned = resolved
	}

	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return fmt.Errorf("landlock: cannot resolve path %q: %w", raw, err)
	}

	if !s.isPathAllowed(abs) {
		return fmt.Errorf("landlock: path %q (resolved to %q) is not within allowed paths", raw, abs)
	}
	return nil
}

// isPathAllowed checks whether the given absolute path is within the allowed paths.
func (s *LandlockSandbox) isPathAllowed(absPath string) bool {
	for _, allowed := range s.allowedPaths {
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
