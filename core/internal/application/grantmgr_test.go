package application_test

import (
	"context"
	"testing"

	"github.com/gentleman/gman/internal/application"
	"github.com/gentleman/gman/internal/domain"
)

func TestGrantManager_RequestGrant_Exists(t *testing.T) {
	perms := newStubPermissionRepo()
	perms.Grant("/home/user/.config/hypr", domain.PermissionRead)

	gm := application.NewGrantManager(perms)
	ctx := context.Background()

	ok, err := gm.RequestGrant(ctx, "/home/user/.config/hypr", domain.PermissionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected grant to exist")
	}
}

func TestGrantManager_RequestGrant_Missing(t *testing.T) {
	perms := newStubPermissionRepo()
	gm := application.NewGrantManager(perms)
	ctx := context.Background()

	ok, err := gm.RequestGrant(ctx, "/home/user/.config/hypr", domain.PermissionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected grant to NOT exist (for TUI prompt)")
	}
}

func TestGrantManager_RequestGrant_RWRequiresRW(t *testing.T) {
	perms := newStubPermissionRepo()
	perms.Grant("/home/user/.config/hypr", domain.PermissionRead) // ro only

	gm := application.NewGrantManager(perms)
	ctx := context.Background()

	ok, err := gm.RequestGrant(ctx, "/home/user/.config/hypr", domain.PermissionWrite)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("rw request on ro-only grant should fail")
	}
}

func TestGrantManager_Grant(t *testing.T) {
	perms := newStubPermissionRepo()
	gm := application.NewGrantManager(perms)
	ctx := context.Background()

	err := gm.Grant(ctx, "/home/user/.config/hypr", domain.PermissionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !perms.Check("/home/user/.config/hypr", domain.PermissionRead) {
		t.Error("grant should be recorded in repository")
	}
}

func TestGrantManager_Grant_EmptyPath(t *testing.T) {
	perms := newStubPermissionRepo()
	gm := application.NewGrantManager(perms)
	ctx := context.Background()

	err := gm.Grant(ctx, "", domain.PermissionRead)
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestGrantManager_Revoke(t *testing.T) {
	perms := newStubPermissionRepo()
	perms.Grant("/home/user/.config/hypr", domain.PermissionRead)
	gm := application.NewGrantManager(perms)
	ctx := context.Background()

	err := gm.Revoke(ctx, "/home/user/.config/hypr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if perms.Check("/home/user/.config/hypr", domain.PermissionRead) {
		t.Error("revoked grant should not pass check")
	}
}

func TestGrantManager_Revoke_NonExistent(t *testing.T) {
	perms := newStubPermissionRepo()
	gm := application.NewGrantManager(perms)
	ctx := context.Background()

	err := gm.Revoke(ctx, "/nonexistent")
	if err == nil {
		t.Error("expected error for non-existent grant")
	}
}

func TestGrantManager_ListGrants(t *testing.T) {
	perms := newStubPermissionRepo()
	gm := application.NewGrantManager(perms)
	ctx := context.Background()

	gm.Grant(ctx, "/a", domain.PermissionRead)
	gm.Grant(ctx, "/b", domain.PermissionWrite)

	grants := gm.ListGrants()
	if len(grants) != 2 {
		t.Errorf("ListGrants() = %d, want 2", len(grants))
	}
}
