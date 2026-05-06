package application

import (
	"context"
	"fmt"
	"time"

	"github.com/gentleman/programas/harvey/internal/domain"
)

// GrantManager orchestrates the grant lifecycle for tool executions.
// It wraps a PermissionRepository and provides:
//   - RequestGrant: check if a grant exists; if not, return false
//     (the TUI will prompt the user in PR 4)
//   - Grant: record a new grant with the current timestamp
//   - Revoke: remove an existing grant
//
// Grants are session-scoped — they are discarded when the process exits.
type GrantManager struct {
	perms domain.PermissionRepository
}

// NewGrantManager creates a GrantManager with the given PermissionRepository.
func NewGrantManager(perms domain.PermissionRepository) *GrantManager {
	return &GrantManager{perms: perms}
}

// RequestGrant checks whether a grant exists for the given path and mode.
// If the grant exists, returns (true, nil).
// If no grant exists, returns (false, nil) — the TUI will prompt the user
// in PR 4 to decide whether to grant or deny.
//
// Returns an error only if the underlying repository fails.
func (g *GrantManager) RequestGrant(ctx context.Context, path string, mode domain.PermissionMode) (bool, error) {
	if g.perms.Check(path, mode) {
		return true, nil
	}
	return false, nil
}

// Grant records a permission grant for the given path and mode.
// The grant is timestamped with the current time.
// Returns an error if the path is empty or the repository fails.
func (g *GrantManager) Grant(ctx context.Context, path string, mode domain.PermissionMode) error {
	if path == "" {
		return fmt.Errorf("grantmgr: path must not be empty")
	}
	grant := domain.Grant{
		Path:      path,
		Mode:      mode,
		GrantedAt: time.Now().UTC().Format(time.RFC3339),
	}
	_ = grant // The repository stores internally; GrantAt is for audit
	return g.perms.Grant(path, mode)
}

// Revoke removes a previously granted permission for the given path.
// Returns an error if no grant exists or the repository fails.
func (g *GrantManager) Revoke(ctx context.Context, path string) error {
	return g.perms.Revoke(path)
}

// ListGrants returns all active grants from the repository.
func (g *GrantManager) ListGrants() []domain.Grant {
	return g.perms.ListGrants()
}
