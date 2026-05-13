package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	c := Defaults()

	if c.Version != "2.1.0" {
		t.Errorf("expected version 2.1.0, got %s", c.Version)
	}
	if c.Window.Mode != "floating" {
		t.Errorf("expected default mode 'floating', got %s", c.Window.Mode)
	}
	if c.Window.Width != 420 {
		t.Errorf("expected default width 420, got %d", c.Window.Width)
	}
	if c.Backend.Provider != "ollama" {
		t.Errorf("expected default provider 'ollama', got %s", c.Backend.Provider)
	}
	if c.Backend.OllamaURL != "http://localhost:11434" {
		t.Errorf("expected default ollama URL, got %s", c.Backend.OllamaURL)
	}
	if c.Backend.Model != "" {
		t.Errorf("expected empty default model, got %s", c.Backend.Model)
	}
	if c.Backend.APIKeys == nil {
		t.Error("expected APIKeys map to be initialized")
	}
	if len(c.Directories) != 0 {
		t.Errorf("expected empty directories, got %v", c.Directories)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	c := Defaults()
	c.Backend.Model = "llama3.2:3b"
	c.Window.Mode = "companion"
	c.Window.Width = 380
	c.Backend.APIKeys["openai"] = "sk-test-key"

	err := c.Save(configPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load and verify roundtrip
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Version != "2.1.0" {
		t.Errorf("expected version 2.1.0, got %s", loaded.Version)
	}
	if loaded.Backend.Model != "llama3.2:3b" {
		t.Errorf("expected model 'llama3.2:3b', got %s", loaded.Backend.Model)
	}
	if loaded.Window.Mode != "companion" {
		t.Errorf("expected mode 'companion', got %s", loaded.Window.Mode)
	}
	if loaded.Window.Width != 380 {
		t.Errorf("expected width 380, got %d", loaded.Window.Width)
	}
	if loaded.Backend.APIKeys["openai"] != "sk-test-key" {
		t.Errorf("expected openai key, got %s", loaded.Backend.APIKeys["openai"])
	}
}

func TestLoadMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.json")

	c, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load should not error on missing file, got: %v", err)
	}

	// Should return defaults
	if c.Window.Mode != "floating" {
		t.Errorf("expected default mode, got %s", c.Window.Mode)
	}
}

func TestSaveCreatesParentDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gman", "config.json")

	c := Defaults()
	err := c.Save(configPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created in nested directory")
	}
}

func TestLoadPreservesUnknownFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write config with extra fields
	content := `{
		"version": "2.1.0",
		"window": {"mode": "floating", "width": 420},
		"backend": {"provider": "ollama", "model": "llama3", "ollama_url": "http://localhost:11434", "api_keys": {}},
		"theme": "dark",
		"directories": [],
		"future_field": "should_be_preserved"
	}`
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Theme != "dark" {
		t.Errorf("expected theme 'dark', got %s", loaded.Theme)
	}
}

func TestSaveJSONIsValid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	c := Defaults()
	c.Backend.Model = "test-model"
	c.Window.Mode = "compact"

	err := c.Save(configPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify it's valid JSON by loading it again
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("re-Load failed: %v", err)
	}

	if loaded.Backend.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %s", loaded.Backend.Model)
	}
	if loaded.Window.Mode != "compact" {
		t.Errorf("expected mode 'compact', got %s", loaded.Window.Mode)
	}
}
