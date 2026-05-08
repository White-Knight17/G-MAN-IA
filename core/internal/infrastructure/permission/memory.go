// Package permission provides an in-memory implementation of the
// domain.PermissionRepository interface for session-scoped grants.
//
// Grants are stored in a sync.RWMutex-protected map and are automatically
// discarded when the process exits — no persistence, no attack surface.
package permission

import (
	"fmt"
	"sync"

	"github.com/gentleman/gman/internal/domain"
)

// InMemoryPermissionRepo implements domain.PermissionRepository using
// a thread-safe in-memory map. Grants are session-scoped and discarded
// on process exit by design.
//
// Guarantees:
//   - Thread-safe: all methods are protected by sync.RWMutex
//   - Path normalization: paths are stored as-is (caller must resolve)
//   - ro/rw escalation: Check for "ro" succeeds if any grant (ro or rw) exists
//   - rw requirement: Check for "rw" only succeeds with an explicit rw grant
type InMemoryPermissionRepo struct {
	mu     sync.RWMutex
	grants map[string]domain.PermissionMode // path → mode
}

// NewInMemoryPermissionRepo creates an empty permission repository.
func NewInMemoryPermissionRepo() *InMemoryPermissionRepo {
	return &InMemoryPermissionRepo{
		grants: make(map[string]domain.PermissionMode),
	}
}

// Grant authorizes access to the given path with the specified mode.
// If a grant already exists for the path, it is upgraded or overwritten.
// Implements domain.PermissionRepository.Grant().
func (r *InMemoryPermissionRepo) Grant(path string, mode domain.PermissionMode) error {
	if path == "" {
		return fmt.Errorf("permission: path must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// If a grant already exists, keep the broader permission
	if existing, ok := r.grants[path]; ok {
		if existing == domain.PermissionWrite || mode == domain.PermissionWrite {
			r.grants[path] = domain.PermissionWrite
		}
		return nil
	}

	r.grants[path] = mode
	return nil
}

// Revoke removes a previously granted permission.
// Returns an error if no grant exists for the path.
// Implements domain.PermissionRepository.Revoke().
func (r *InMemoryPermissionRepo) Revoke(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.grants[path]; !ok {
		return fmt.Errorf("permission: no grant for path %q", path)
	}

	delete(r.grants, path)
	return nil
}

// Check verifies whether the given path has the specified mode granted.
// ro access is satisfied by either ro or rw grants.
// rw access requires an explicit rw grant.
// Implements domain.PermissionRepository.Check().
func (r *InMemoryPermissionRepo) Check(path string, mode domain.PermissionMode) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	granted, ok := r.grants[path]
	if !ok {
		return false
	}

	// ro can be satisfied by any grant level
	if mode == domain.PermissionRead {
		return true
	}

	// rw requires explicit rw
	return granted == domain.PermissionWrite
}

// ListGrants returns a snapshot of all active grants.
// Each grant includes the path, mode, and current timestamp.
// Implements domain.PermissionRepository.ListGrants().
func (r *InMemoryPermissionRepo) ListGrants() []domain.Grant {
	r.mu.RLock()
	defer r.mu.RUnlock()

	grants := make([]domain.Grant, 0, len(r.grants))
	for path, mode := range r.grants {
		grants = append(grants, domain.Grant{
			Path:      path,
			Mode:      mode,
			GrantedAt: "", // populated by GrantManager or caller
		})
	}
	return grants
}

// Clear removes all grants. Useful for session teardown or testing.
func (r *InMemoryPermissionRepo) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.grants = make(map[string]domain.PermissionMode)
}

// Count returns the number of active grants. Useful for testing.
func (r *InMemoryPermissionRepo) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.grants)
}
