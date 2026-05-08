package build_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// projectRoot walks up from cwd to find go.work.
func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("project root not found — missing go.work")
		}
		dir = parent
	}
}

// --- Makefile tests ---

func TestMakefileExists(t *testing.T) {
	root := projectRoot(t)
	info, err := os.Stat(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("Makefile missing: %v", err)
	}
	if info.IsDir() {
		t.Fatal("Makefile is a directory")
	}
}

func TestMakefileTargets(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	content := string(data)

	targets := []string{"build-core", "build-ui", "build",
		"bundle", "clean", "test-all", "test-core", "test-ui", "dev", "install"}
	for _, tgt := range targets {
		found := strings.Contains(content, tgt+":") || strings.Contains(content, tgt+" :")
		if !found {
			t.Errorf("Makefile missing target %q", tgt)
		}
	}
	if !strings.Contains(content, ".PHONY:") {
		t.Error("Makefile missing .PHONY")
	}
}

func TestMakefileBuildCoreOutputPath(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	if !strings.Contains(string(data), "binaries") {
		t.Error("build-core must output to app/src-tauri/binaries/")
	}
}

func TestMakeBuildCoreProducesBinary(t *testing.T) {
	root := projectRoot(t)
	binDir := filepath.Join(root, "app", "src-tauri", "binaries")

	// Ensure clean start
	os.RemoveAll(binDir)
	os.MkdirAll(binDir, 0755)
	defer os.RemoveAll(binDir)

	cmd := exec.Command("make", "build-core")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make build-core: %v\n%s", err, string(out))
	}

	entries, err := os.ReadDir(binDir)
	if err != nil {
		t.Fatalf("read binaries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no binary produced in app/src-tauri/binaries/")
	}
}

// --- build.sh tests ---

func TestBuildScriptExistsAndExecutable(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "scripts", "build.sh")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("build.sh missing: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("build.sh not executable — run: chmod +x scripts/build.sh")
	}
}

func TestBuildScriptContent(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "scripts", "build.sh"))
	if err != nil {
		t.Fatalf("read build.sh: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"#!/bin/bash",
		"set -euo pipefail",
		"go build",
		"gman-core",
		"pnpm build",
		"pnpm tauri build",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("build.sh missing %q", want)
		}
	}
}

// --- tauri.conf.json tests ---

func TestTauriConfBundlerSection(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "app", "src-tauri", "tauri.conf.json"))
	if err != nil {
		t.Fatalf("read tauri.conf.json: %v", err)
	}
	content := string(data)

	checks := []string{
		`"active": true`,
		`"targets": "all"`,
		`"icon"`,
		`"linux"`,
		`"deb"`,
		`"appimage"`,
		`"externalBin"`,
		`"binaries/gman-core`,
		`"copyright"`,
		`"category"`,
		`"shortDescription"`,
		`"longDescription"`,
	}
	for _, ck := range checks {
		if !strings.Contains(content, ck) {
			t.Errorf("tauri.conf.json bundle section missing %q", ck)
		}
	}

	// Must mention ollama dependency
	if !strings.Contains(content, "ollama") {
		t.Error("tauri.conf.json should list ollama as a deb dependency")
	}

	// Must mention Gentleman Programming copyright
	if !strings.Contains(content, "Gentleman") {
		t.Error("tauri.conf.json missing Gentleman Programming copyright")
	}
}

// Verify externalBin path uses correct naming convention
func TestTauriConfExternalBinNaming(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "app", "src-tauri", "tauri.conf.json"))
	if err != nil {
		t.Fatalf("read tauri.conf.json: %v", err)
	}
	content := string(data)

	// externalBin must reference binaries/gman-core with a target triple suffix
	if !strings.Contains(content, `"externalBin"`) {
		t.Fatal("tauri.conf.json missing externalBin field")
	}
	if !strings.Contains(content, "gman-core") {
		t.Error("externalBin must reference gman-core binary")
	}
}

// --- CI workflow tests ---

func TestCIWorkflowHasRequiredJobs(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci.yml: %v", err)
	}
	content := string(data)

	// Must define test-go, test-rust, test-frontend, and build jobs
	jobs := []string{"test-go", "test-rust", "test-frontend", "build"}
	for _, job := range jobs {
		if !strings.Contains(content, job+":") {
			t.Errorf("ci.yml missing job %q", job)
		}
	}

	// Build job must depend on test jobs
	if !strings.Contains(content, "needs:") {
		t.Error("ci.yml build job must use 'needs:' to depend on test jobs")
	}

	// Must upload artifact
	if !strings.Contains(content, "upload-artifact") {
		t.Error("ci.yml must upload build artifacts")
	}
}

func TestCIWorkflowUsesSetupGo(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci.yml: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "actions/setup-go@v5") {
		t.Error("ci.yml must use actions/setup-go@v5 for Go test job")
	}
	if !strings.Contains(content, `go-version: "1.26"`) {
		t.Error("ci.yml must specify Go 1.26")
	}
}

func TestCIWorkflowUsesPnpmAndNode(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci.yml: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "pnpm/action-setup") {
		t.Error("ci.yml must use pnpm/action-setup for frontend jobs")
	}
	if !strings.Contains(content, "actions/setup-node") {
		t.Error("ci.yml must use actions/setup-node for frontend jobs")
	}
}

func TestCIWorkflowUsesRustToolchain(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci.yml: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "dtolnay/rust-toolchain") {
		t.Error("ci.yml must use dtolnay/rust-toolchain for Rust test job")
	}
}

// --- .gitignore tests ---

func TestGitignoreRequiredPatterns(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	content := string(data)

	patterns := []string{
		"app/src-tauri/target/",
		"app/dist/",
		"app/node_modules/",
		"app/src-tauri/binaries/",
		"app/test-results/",
	}
	for _, pat := range patterns {
		if !strings.Contains(content, pat) {
			t.Errorf(".gitignore missing entry %q", pat)
		}
	}
}
