package domain

// PermissionMode represents the access level granted to a directory.
// ro = read-only (list and read files, no writes)
// rw = read-write (full access including writes and file creation)
type PermissionMode string

const (
	// PermissionRead allows listing and reading files in the granted path.
	PermissionRead PermissionMode = "ro"

	// PermissionWrite allows full read-write access to the granted path.
	PermissionWrite PermissionMode = "rw"
)

// PermissionRepository manages session-scoped directory grants.
// Grants are per-directory, per-mode, and expire when the session ends.
// Implementations are responsible for thread safety.
//
// Implementations: InMemoryPermissionRepo (sync.RWMutex + map).
type PermissionRepository interface {
	// Grant authorizes access to the given path with the specified mode.
	// Paths should be absolute and resolved before calling.
	// Returns an error if the path is invalid or already granted.
	Grant(path string, mode PermissionMode) error

	// Revoke removes a previously granted permission.
	// Returns an error if no grant exists for the path.
	Revoke(path string) error

	// Check verifies whether a given path has the specified mode granted.
	// Returns true if access is allowed, false otherwise.
	// ro access is satisfied by either ro or rw grants.
	// rw access requires an explicit rw grant.
	Check(path string, mode PermissionMode) bool

	// ListGrants returns all active grants in the current session.
	ListGrants() []Grant
}

// Grant represents a single permission grant for a directory path.
// Grants are session-scoped and discarded on process exit.
type Grant struct {
	// Path is the absolute, resolved directory path that this grant covers.
	Path string `json:"path"`

	// Mode is the access level granted (ro or rw).
	Mode PermissionMode `json:"mode"`

	// GrantedAt is the timestamp when this grant was created.
	// Used for audit and display purposes in the TUI.
	GrantedAt string `json:"granted_at"` // ISO 8601 timestamp
}
