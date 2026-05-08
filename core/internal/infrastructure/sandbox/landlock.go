package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Landlock filesystem access rights (from linux/landlock.h).
// These control which file operations are permitted on allowed paths.
const (
	landlockAccessFSExecute    = 1 << 0  // LANDLOCK_ACCESS_FS_EXECUTE
	landlockAccessFSWriteFile  = 1 << 1  // LANDLOCK_ACCESS_FS_WRITE_FILE
	landlockAccessFSReadFile   = 1 << 2  // LANDLOCK_ACCESS_FS_READ_FILE
	landlockAccessFSReadDir    = 1 << 3  // LANDLOCK_ACCESS_FS_READ_DIR
	landlockAccessFSRemoveDir  = 1 << 4  // LANDLOCK_ACCESS_FS_REMOVE_DIR
	landlockAccessFSRemoveFile = 1 << 5  // LANDLOCK_ACCESS_FS_REMOVE_FILE
	landlockAccessFSMakeChar   = 1 << 6  // LANDLOCK_ACCESS_FS_MAKE_CHAR
	landlockAccessFSMakeDir    = 1 << 7  // LANDLOCK_ACCESS_FS_MAKE_DIR
	landlockAccessFSMakeReg    = 1 << 8  // LANDLOCK_ACCESS_FS_MAKE_REG
	landlockAccessFSMakeSock   = 1 << 9  // LANDLOCK_ACCESS_FS_MAKE_SOCK
	landlockAccessFSMakeFifo   = 1 << 10 // LANDLOCK_ACCESS_FS_MAKE_FIFO
	landlockAccessFSMakeBlock  = 1 << 11 // LANDLOCK_ACCESS_FS_MAKE_BLOCK
	landlockAccessFSMakeSym    = 1 << 12 // LANDLOCK_ACCESS_FS_MAKE_SYM
	landlockAccessFSRefer      = 1 << 13 // LANDLOCK_ACCESS_FS_REFER
	landlockAccessFSTruncate   = 1 << 14 // LANDLOCK_ACCESS_FS_TRUNCATE
)

// LandlockSandbox implements domain.Sandbox using the Linux Landlock LSM.
// It restricts the CURRENT Go process's file access to only the allowed paths.
// Once applied via landlock_restrict_self(), the rules are immutable for the
// lifetime of the process — no escalation, no bypass.
//
// IMPORTANT: Because Landlock is in-process, it applies to the Go process itself.
// It should be applied at startup before any file operations on paths that may
// not be in the allowlist. It complements Bubblewrap (subprocess isolation) by
// hardening the tool-runner process.
type LandlockSandbox struct {
	allowedPaths   []string
	enforced       bool
	rulesetFD      int
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
		rulesetFD:    -1,
	}
}

// Apply activates Landlock restrictions for the current process.
// After this call, the process can only access files within the allowedPaths.
// This is a ONE-WAY operation — rules cannot be changed after enforcement.
//
// Uses raw syscalls (landlock_create_ruleset, landlock_add_rule,
// landlock_restrict_self) because golang.org/x/sys does not currently
// provide high-level Go wrappers for the Landlock API.
func (s *LandlockSandbox) Apply() error {
	// All allowed access rights — we need all of them to permit read/write/execute
	// operations within the allowed paths.
	handledAccess := uint64(
		landlockAccessFSExecute |
			landlockAccessFSWriteFile |
			landlockAccessFSReadFile |
			landlockAccessFSReadDir |
			landlockAccessFSRemoveDir |
			landlockAccessFSRemoveFile |
			landlockAccessFSMakeChar |
			landlockAccessFSMakeDir |
			landlockAccessFSMakeReg |
			landlockAccessFSMakeSock |
			landlockAccessFSMakeFifo |
			landlockAccessFSMakeBlock |
			landlockAccessFSMakeSym |
			landlockAccessFSRefer |
			landlockAccessFSTruncate,
	)

	// Step 1: Create a ruleset
	rulesetAttr := unix.LandlockRulesetAttr{
		Access_fs: handledAccess,
	}

	rulesetFD, err := landlockCreateRuleset(&rulesetAttr, uint32(unsafe.Sizeof(rulesetAttr)), 0)
	if err != nil {
		return fmt.Errorf("landlock: create_ruleset failed: %w (kernel supports Landlock? need CONFIG_SECURITY_LANDLOCK=y)", err)
	}
	s.rulesetFD = rulesetFD

	// Step 2: Add rules for each allowed path
	for _, p := range s.allowedPaths {
		abs, err := filepath.Abs(p)
		if err != nil {
			unix.Close(rulesetFD)
			return fmt.Errorf("landlock: cannot resolve path %q: %w", p, err)
		}

		// For Landlock, the directory must exist and we need an open fd for it.
		info, statErr := os.Stat(abs)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			unix.Close(rulesetFD)
			return fmt.Errorf("landlock: cannot stat %q: %w", abs, statErr)
		}
		if !info.IsDir() {
			continue
		}

		// Open the directory to get a file descriptor for Parent_fd.
		dirFD, err := unix.Open(abs, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
		if err != nil {
			unix.Close(rulesetFD)
			return fmt.Errorf("landlock: cannot open directory %q: %w", abs, err)
		}

		pathBeneath := unix.LandlockPathBeneathAttr{
			Allowed_access: handledAccess,
			Parent_fd:      int32(dirFD),
		}

		addErr := landlockAddRule(rulesetFD, unix.LANDLOCK_RULE_PATH_BENEATH, &pathBeneath, 0)
		unix.Close(dirFD) // Parent_fd is no longer needed after add_rule

		if addErr != nil {
			// Retry with minimal access flags (read-only) as a fallback
			retryFD, retryOpenErr := unix.Open(abs, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
			if retryOpenErr != nil {
				unix.Close(rulesetFD)
				return fmt.Errorf("landlock: add_rule failed for %q: %w", abs, addErr)
			}

			pathBeneathRetry := unix.LandlockPathBeneathAttr{
				Allowed_access: landlockAccessFSReadFile |
					landlockAccessFSReadDir |
					landlockAccessFSExecute,
				Parent_fd: int32(retryFD),
			}
			retryErr := landlockAddRule(rulesetFD, unix.LANDLOCK_RULE_PATH_BENEATH, &pathBeneathRetry, 0)
			unix.Close(retryFD)

			if retryErr != nil {
				unix.Close(rulesetFD)
				return fmt.Errorf("landlock: add_rule failed for %q: %w (first error: %v)", abs, retryErr, addErr)
			}
		}
	}

	// Step 3: Enforce the ruleset
	if err := landlockRestrictSelf(rulesetFD, 0); err != nil {
		unix.Close(rulesetFD)
		return fmt.Errorf("landlock: restrict_self failed: %w (process may need CAP_SYS_ADMIN or to run as root)", err)
	}

	// Ruleset is now enforced; the fd is no longer needed.
	unix.Close(rulesetFD)
	s.rulesetFD = -1
	s.enforced = true

	return nil
}

// landlockCreateRuleset invokes the landlock_create_ruleset syscall.
// Returns the ruleset file descriptor or an error.
func landlockCreateRuleset(attr *unix.LandlockRulesetAttr, size uint32, flags uint32) (int, error) {
	fd, _, errno := unix.Syscall(
		unix.SYS_LANDLOCK_CREATE_RULESET,
		uintptr(unsafe.Pointer(attr)),
		uintptr(size),
		uintptr(flags),
	)
	if errno != 0 {
		return -1, fmt.Errorf("syscall landlock_create_ruleset: %w", errno)
	}
	return int(fd), nil
}

// landlockAddRule invokes the landlock_add_rule syscall.
// Uses Syscall6 because it requires 4 arguments.
func landlockAddRule(rulesetFD int, ruleType uint32, ruleAttr *unix.LandlockPathBeneathAttr, flags uint32) error {
	_, _, errno := unix.Syscall6(
		unix.SYS_LANDLOCK_ADD_RULE,
		uintptr(rulesetFD),
		uintptr(ruleType),
		uintptr(unsafe.Pointer(ruleAttr)),
		uintptr(flags),
		0,
		0,
	)
	if errno != 0 {
		return fmt.Errorf("syscall landlock_add_rule: %w", errno)
	}
	return nil
}

// landlockRestrictSelf invokes the landlock_restrict_self syscall.
func landlockRestrictSelf(rulesetFD int, flags uint32) error {
	_, _, errno := unix.Syscall(
		unix.SYS_LANDLOCK_RESTRICT_SELF,
		uintptr(rulesetFD),
		uintptr(flags),
		0,
	)
	if errno != 0 {
		return fmt.Errorf("syscall landlock_restrict_self: %w", errno)
	}
	return nil
}

// Execute validates that the requested path is within allowed paths.
// For Landlock, the actual file access enforcement is done by the kernel
// via the Landlock rules applied in Apply(). This method provides the
// pre-validation layer to catch obvious violations early.
func (s *LandlockSandbox) Execute(ctx context.Context, command string, args []string, allowedPaths []string) (string, error) {
	// Validate paths are within allowedPaths
	for _, raw := range append(args, allowedPaths...) {
		if len(raw) == 0 {
			continue
		}
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
