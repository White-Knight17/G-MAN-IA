// Package sandbox provides defense-in-depth execution isolation for tool operations.
// BubblewrapSandbox uses bubblewrap (bwrap) containers for subprocess isolation;
// LandlockSandbox applies Landlock LSM rules within the Go process itself.
package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gentleman/programas/harvey/internal/domain"
)

// Default bwrap binary location.
const bwrapBin = "/usr/bin/bwrap"

// Default timeout for sandboxed command execution.
const defaultSandboxTimeout = 30 * time.Second

// Commands that are NEVER allowed inside the sandbox, regardless of allowlist.
var blocklist = []string{
	"rm", "dd", "mkfs", "mkfs.ext4", "mkfs.btrfs", "mkfs.xfs",
	"sudo", "su", "chmod", "chown", "mount", "umount",
	"reboot", "shutdown", "poweroff", "halt",
	"fdisk", "parted", "sfdisk",
}

// BubblewrapSandbox implements domain.Sandbox using bubblewrap (bwrap).
// It containerizes every subprocess with --unshare-all, read-only rootfs bind-mounts,
// writable allowed-path binds, and --no-network. A command blocklist rejects dangerous
// commands before they reach bwrap.
//
// Architecture:
//   - Path validation: resolves and normalizes paths, rejects traversal attempts.
//   - bwrap container: isolated PID/net namespaces, read-only /usr and /bin.
//   - 30-second per-command timeout via context.
type BubblewrapSandbox struct {
	allowedPaths []string
}

// NewBubblewrapSandbox creates a BubblewrapSandbox with the given allowed paths.
// These paths will be bind-mounted into the container as writable mounts.
func NewBubblewrapSandbox(allowedPaths []string) *BubblewrapSandbox {
	resolved := make([]string, len(allowedPaths))
	for i, p := range allowedPaths {
		resolved[i] = filepath.Clean(p)
	}
	return &BubblewrapSandbox{allowedPaths: resolved}
}

// Execute runs a command inside the bubblewrap container.
//
// Steps:
//  1. Validate the command is not in the global blocklist.
//  2. Build bwrap arguments: user namespace isolation, ro-binds for system dirs,
//     writable binds for allowed paths, tmpfs /tmp, no-network.
//  3. Run the command via exec.CommandContext with a 30-second timeout.
//  4. Return combined stdout+stderr.
//
// The command string is the binary to execute (e.g., "ls", "grep", "hyprctl").
// args contains the arguments to the command.
func (s *BubblewrapSandbox) Execute(ctx context.Context, command string, args []string, allowedPaths []string) (string, error) {
	// Step 1: Blocklist validation
	if err := s.validateCommand(command); err != nil {
		return "", err
	}

	// Step 1.5: Validate paths are within allowed paths
	if err := s.validatePaths(allowedPaths); err != nil {
		return "", err
	}

	// Step 2: Build bwrap arguments
	bwrapArgs := s.buildBwrapArgs(allowedPaths)
	bwrapArgs = append(bwrapArgs, command)
	bwrapArgs = append(bwrapArgs, args...)

	// Step 3: Execute with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, defaultSandboxTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, bwrapBin, bwrapArgs...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if ctxWithTimeout.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("sandbox: command timed out after %v: %s", defaultSandboxTimeout, command)
		}
		return string(output), fmt.Errorf("sandbox: bwrap failed: %w (output: %s)", err, string(output))
	}

	return string(output), nil
}

// AllowedPaths returns the current set of paths permitted by this sandbox.
func (s *BubblewrapSandbox) AllowedPaths() []string {
	paths := make([]string, len(s.allowedPaths))
	copy(paths, s.allowedPaths)
	return paths
}

// validateCommand checks that the command is not in the blocklist.
// The check is performed against the base name of the command (e.g., "rm", "sudo").
func (s *BubblewrapSandbox) validateCommand(command string) error {
	base := filepath.Base(command)
	for _, blocked := range blocklist {
		if strings.EqualFold(base, blocked) {
			return fmt.Errorf("sandbox: command %q is blocked — potentially destructive operations are not allowed", command)
		}
	}
	return nil
}

// validatePaths ensures each path is within the sandbox's allowedPaths.
// It resolves and normalizes each path, then checks it has one of the allowed
// paths as a prefix. This prevents path traversal attacks.
func (s *BubblewrapSandbox) validatePaths(paths []string) error {
	for _, raw := range paths {
		// Resolve and normalize to prevent traversal (e.g., ../etc, symlinks)
		cleaned := filepath.Clean(raw)

		// Resolve symlinks where possible — EvalSymlinks fails on non-existent paths,
		// which is acceptable (the subsequent operation will also fail).
		if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
			cleaned = resolved
		}

		abs, err := filepath.Abs(cleaned)
		if err != nil {
			return fmt.Errorf("sandbox: cannot resolve path %q: %w", raw, err)
		}

		// Check that the resolved path is within one of the allowed paths
		if !s.isPathAllowed(abs) {
			return fmt.Errorf("sandbox: path %q (resolved to %q) is not within allowed paths: %v",
				raw, abs, s.allowedPaths)
		}
	}
	return nil
}

// isPathAllowed checks whether the given absolute path has one of the
// sandbox's allowedPaths as a directory prefix. This prevents traversal
// attacks like /etc/passwd when only ~/.config is allowed.
func (s *BubblewrapSandbox) isPathAllowed(absPath string) bool {
	for _, allowed := range s.allowedPaths {
		allowedAbs, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		// Ensure the allowed dir is a directory prefix of the target path.
		// Use strings.HasPrefix after ensuring both are clean paths.
		rel, err := filepath.Rel(allowedAbs, absPath)
		if err != nil {
			continue
		}
		// If relative path does not start with "..", it's within the allowed dir.
		if !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}

// buildBwrapArgs constructs the arguments for the bwrap command.
// It creates a minimal container with:
//   - --unshare-all for full namespace isolation
//   - --ro-bind for read-only system mounts (/usr, /bin, /lib, /lib64, /etc)
//   - --bind for writable allowed paths
//   - --tmpfs /tmp for a clean temp filesystem
//   - --no-network to prevent network access
//   - --dev /dev for basic device access
//   - --proc /proc for process info
func (s *BubblewrapSandbox) buildBwrapArgs(allowedPaths []string) []string {
	args := []string{
		"--unshare-all",
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/bin", "/bin",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/lib64", "/lib64",
		"--ro-bind", "/etc", "/etc",
		"--tmpfs", "/tmp",
		"--no-network",
		"--dev", "/dev",
		"--proc", "/proc",
	}

	// Bind-mount each allowed path as writable inside the container.
	// If a path doesn't exist on the host, bwrap will fail — this is the expected behavior.
	for _, p := range allowedPaths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		args = append(args, "--bind", abs, abs)
	}

	return args
}
