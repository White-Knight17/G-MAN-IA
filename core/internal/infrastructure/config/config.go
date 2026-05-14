// Package config provides persistent configuration for G-MAN.
// Reads/writes ~/.config/gman/config.json with graceful defaults on missing file.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the top-level application configuration.
type Config struct {
	Version     string   `json:"version"`
	Window      Window   `json:"window"`
	Backend     Backend  `json:"backend"`
	Theme       string   `json:"theme"`
	Directories []string `json:"directories"`
}

// Window holds window mode configuration.
type Window struct {
	Mode     string `json:"mode"`
	Width    int    `json:"width"`
	Position Point  `json:"position"`
}

// Point represents an X/Y coordinate.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Backend holds AI provider configuration.
type Backend struct {
	Provider  string            `json:"provider"`
	Model     string            `json:"model"`
	OllamaURL string            `json:"ollama_url"`
	APIKeys   map[string]string `json:"api_keys"`
	BaseURLs  map[string]string `json:"base_urls"`
}

// Defaults returns a Config with sensible defaults.
func Defaults() Config {
	return Config{
		Version: "2.1.0",
		Window: Window{
			Mode:  "floating",
			Width: 420,
			Position: Point{
				X: 0,
				Y: 0,
			},
		},
		Backend: Backend{
			Provider:  "ollama",
			Model:     "",
			OllamaURL: "http://localhost:11434",
			APIKeys:   make(map[string]string),
			BaseURLs:  make(map[string]string),
		},
		Theme:       "dark",
		Directories: []string{},
	}
}

// Load reads configuration from the given path.
// If the file does not exist, returns defaults without error.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Defaults(), nil
		}
		return Config{}, err
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}

	// Ensure APIKeys is never nil
	if c.Backend.APIKeys == nil {
		c.Backend.APIKeys = make(map[string]string)
	}
	if c.Directories == nil {
		c.Directories = []string{}
	}

	return c, nil
}

// Save writes the configuration to the given path, creating parent directories as needed.
func (c Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
