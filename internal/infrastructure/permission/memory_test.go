package permission_test

import (
	"testing"

	"github.com/gentleman/programas/harvey/internal/domain"
	"github.com/gentleman/programas/harvey/internal/infrastructure/permission"
)

func TestInMemoryPermissionRepo_Grant(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		mode      domain.PermissionMode
		wantErr   bool
		wantCount int
	}{
		{
			name:      "grant read-only access",
			path:      "/home/user/.config/hypr",
			mode:      domain.PermissionRead,
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:      "grant read-write access",
			path:      "/home/user/.config/waybar",
			mode:      domain.PermissionWrite,
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:      "empty path should error",
			path:      "",
			mode:      domain.PermissionRead,
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := permission.NewInMemoryPermissionRepo()
			err := repo.Grant(tt.path, tt.mode)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if count := repo.Count(); count != tt.wantCount {
				t.Errorf("Count() = %d, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestInMemoryPermissionRepo_Upgrade(t *testing.T) {
	repo := permission.NewInMemoryPermissionRepo()

	// Grant ro first
	err := repo.Grant("/home/user/.config/hypr", domain.PermissionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Grant rw later — should upgrade
	err = repo.Grant("/home/user/.config/hypr", domain.PermissionWrite)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have 1 grant (upgraded)
	if repo.Count() != 1 {
		t.Errorf("Count() = %d, want 1", repo.Count())
	}

	// Should have rw access now
	if !repo.Check("/home/user/.config/hypr", domain.PermissionWrite) {
		t.Error("should have rw access after upgrade")
	}
}

func TestInMemoryPermissionRepo_Check(t *testing.T) {
	repo := permission.NewInMemoryPermissionRepo()
	repo.Grant("/home/user/.config/hypr", domain.PermissionRead)
	repo.Grant("/home/user/.config/waybar", domain.PermissionWrite)

	tests := []struct {
		name    string
		path    string
		mode    domain.PermissionMode
		wantOK  bool
	}{
		{
			name:   "ro check on ro path passes",
			path:   "/home/user/.config/hypr",
			mode:   domain.PermissionRead,
			wantOK: true,
		},
		{
			name:   "rw check on ro path fails",
			path:   "/home/user/.config/hypr",
			mode:   domain.PermissionWrite,
			wantOK: false,
		},
		{
			name:   "ro check on rw path passes (escalation)",
			path:   "/home/user/.config/waybar",
			mode:   domain.PermissionRead,
			wantOK: true,
		},
		{
			name:   "rw check on rw path passes",
			path:   "/home/user/.config/waybar",
			mode:   domain.PermissionWrite,
			wantOK: true,
		},
		{
			name:   "ungranted path fails",
			path:   "/etc/passwd",
			mode:   domain.PermissionRead,
			wantOK: false,
		},
		{
			name:   "ungranted path rw fails",
			path:   "/etc/shadow",
			mode:   domain.PermissionWrite,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.Check(tt.path, tt.mode)
			if result != tt.wantOK {
				t.Errorf("Check(%q, %q) = %v, want %v", tt.path, tt.mode, result, tt.wantOK)
			}
		})
	}
}

func TestInMemoryPermissionRepo_Revoke(t *testing.T) {
	repo := permission.NewInMemoryPermissionRepo()
	repo.Grant("/home/user/.config/hypr", domain.PermissionRead)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "revoke existing grant",
			path:    "/home/user/.config/hypr",
			wantErr: false,
		},
		{
			name:    "revoke non-existent grant",
			path:    "/home/user/.config/kitty",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Revoke(tt.path)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	// After revoking /hypr, it should not be checkable
	if repo.Check("/home/user/.config/hypr", domain.PermissionRead) {
		t.Error("revoked path should not pass check")
	}
}

func TestInMemoryPermissionRepo_ListGrants(t *testing.T) {
	repo := permission.NewInMemoryPermissionRepo()
	repo.Grant("/path/a", domain.PermissionRead)
	repo.Grant("/path/b", domain.PermissionWrite)

	grants := repo.ListGrants()

	if len(grants) != 2 {
		t.Errorf("ListGrants() = %d grants, want 2", len(grants))
	}

	found := map[string]domain.PermissionMode{}
	for _, g := range grants {
		found[g.Path] = g.Mode
	}

	if mode, ok := found["/path/a"]; !ok || mode != domain.PermissionRead {
		t.Errorf("/path/a grant missing or wrong mode: %v", ok)
	}
	if mode, ok := found["/path/b"]; !ok || mode != domain.PermissionWrite {
		t.Errorf("/path/b grant missing or wrong mode: %v", ok)
	}
}

func TestInMemoryPermissionRepo_Concurrent(t *testing.T) {
	repo := permission.NewInMemoryPermissionRepo()

	// Concurrent grants and checks
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(i int) {
			path := "/path/" + string(rune('a'+i%26))
			repo.Grant(path, domain.PermissionRead)
			repo.Check(path, domain.PermissionRead)
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	if repo.Count() > 26 {
		t.Errorf("max 26 unique paths, got %d", repo.Count())
	}
}

func TestInMemoryPermissionRepo_Clear(t *testing.T) {
	repo := permission.NewInMemoryPermissionRepo()
	repo.Grant("/a", domain.PermissionRead)
	repo.Grant("/b", domain.PermissionWrite)

	repo.Clear()

	if repo.Count() != 0 {
		t.Errorf("Count() after clear = %d, want 0", repo.Count())
	}
	if len(repo.ListGrants()) != 0 {
		t.Error("ListGrants after clear should be empty")
	}
}
