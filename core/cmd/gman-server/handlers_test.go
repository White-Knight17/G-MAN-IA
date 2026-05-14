package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gentleman/gman/internal/infrastructure/config"
	"github.com/gentleman/gman/internal/infrastructure/ollama"
	"github.com/gentleman/gman/internal/transport"
)

// TestModelListHandler verifies model.list returns parsed models.
func TestModelListHandler(t *testing.T) {
	// Create a mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]interface{}{
				{"name": "llama3.2:3b", "size": 2032891662, "digest": "abc123"},
			},
		})
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("test", nil, server.URL)

	input := `{"jsonrpc":"2.0","id":1,"method":"model.list","params":{}}` + "\n"
	outBuf := new(bytes.Buffer)
	srv := transport.NewServer(bytes.NewBufferString(input), outBuf)

	srv.Handle("model.list", func(req transport.Request) transport.Response {
		ctx := context.Background()
		models, err := client.ListModels(ctx)
		if err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InternalError,
					Message: err.Error(),
				},
			}
		}
		return transport.Response{Result: map[string]interface{}{
			"models": models,
		}}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	// Skip the first line (could be notification)
	lines := bytes.Split(outBuf.Bytes(), []byte("\n"))
	var resp transport.Response
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var parsed transport.Response
		if err := json.Unmarshal(line, &parsed); err != nil {
			continue
		}
		if parsed.ID != nil {
			resp = parsed
			break
		}
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result to be map, got %T", resp.Result)
	}

	models, ok := resultMap["models"].([]interface{})
	if !ok {
		t.Fatalf("expected models to be array, got %T", resultMap["models"])
	}
	if len(models) != 1 {
		t.Errorf("expected 1 model, got %d", len(models))
	}
}

// TestConfigGetHandler verifies config.get returns config without API keys.
func TestConfigGetHandler(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write a test config
	cfg := config.Defaults()
	cfg.Backend.Provider = "ollama"
	cfg.Backend.Model = "llama3.2:3b"
	cfg.Theme = "dark"
	cfg.Save(configPath)

	input := `{"jsonrpc":"2.0","id":1,"method":"config.get","params":{}}` + "\n"
	outBuf := new(bytes.Buffer)
	srv := transport.NewServer(bytes.NewBufferString(input), outBuf)

	srv.Handle("config.get", func(req transport.Request) transport.Response {
		loaded, err := config.Load(configPath)
		if err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InternalError,
					Message: err.Error(),
				},
			}
		}
		return transport.Response{Result: safeConfigMap(loaded)}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	lines := bytes.Split(outBuf.Bytes(), []byte("\n"))
	var resp transport.Response
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var parsed transport.Response
		if err := json.Unmarshal(line, &parsed); err != nil {
			continue
		}
		if parsed.ID != nil {
			resp = parsed
			break
		}
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp.Result)
	}

	// Verify provider is present
	if resultMap["provider"] != "ollama" {
		t.Errorf("expected provider 'ollama', got %v", resultMap["provider"])
	}
	// Verify has_api_key is present (security: never expose actual keys)
	if _, hasKey := resultMap["has_api_key"]; !hasKey {
		t.Error("expected has_api_key field in response")
	}
}

// TestConfigSetHandler verifies config.set updates and persists config.
func TestConfigSetHandler(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write initial config
	cfg := config.Defaults()
	cfg.Theme = "dark"
	cfg.Backend.Model = "llama3.2:3b"
	cfg.Save(configPath)

	input := `{"jsonrpc":"2.0","id":1,"method":"config.set","params":{"theme":"light"}}` + "\n"
	outBuf := new(bytes.Buffer)
	srv := transport.NewServer(bytes.NewBufferString(input), outBuf)

	srv.Handle("config.set", func(req transport.Request) transport.Response {
		loaded, err := config.Load(configPath)
		if err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InternalError,
					Message: err.Error(),
				},
			}
		}

		var params struct {
			Theme    string `json:"theme,omitempty"`
			Model    string `json:"model,omitempty"`
			Provider string `json:"provider,omitempty"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: err.Error(),
				},
			}
		}

		if params.Theme != "" {
			loaded.Theme = params.Theme
		}
		if params.Model != "" {
			loaded.Backend.Model = params.Model
		}
		if params.Provider != "" {
			loaded.Backend.Provider = params.Provider
		}

		if err := loaded.Save(configPath); err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InternalError,
					Message: err.Error(),
				},
			}
		}

		return transport.Response{Result: map[string]bool{"ok": true}}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	lines := bytes.Split(outBuf.Bytes(), []byte("\n"))
	var resp transport.Response
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var parsed transport.Response
		if err := json.Unmarshal(line, &parsed); err != nil {
			continue
		}
		if parsed.ID != nil {
			resp = parsed
			break
		}
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Verify the file was actually updated
	updated, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load updated config: %v", err)
	}
	if updated.Theme != "light" {
		t.Errorf("expected theme 'light', got %q", updated.Theme)
	}
}

// TestModelPullHandler verifies model.pull starts pull and returns immediately.
func TestModelPullHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher := w.(http.Flusher)
		responses := []string{
			`{"status":"pulling manifest"}`,
			`{"status":"success"}`,
		}
		for _, resp := range responses {
			w.Write([]byte(resp + "\n"))
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("test", nil, server.URL)

	input := `{"jsonrpc":"2.0","id":1,"method":"model.pull","params":{"model":"llama3.2:3b"}}` + "\n"
	outBuf := new(bytes.Buffer)
	srv := transport.NewServer(bytes.NewBufferString(input), outBuf)

	srv.Handle("model.pull", func(req transport.Request) transport.Response {
		var params struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: err.Error(),
				},
			}
		}
		if params.Model == "" {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: "model parameter is required",
				},
			}
		}

		progressCh := make(chan ollama.PullProgress, 10)

		go func() {
			ctx := context.Background()
			err := client.PullModel(ctx, params.Model, progressCh)
			close(progressCh)
			if err != nil {
				srv.SendNotification("model.pull.error", map[string]interface{}{
					"model": params.Model,
					"error": err.Error(),
				})
				return
			}
		}()

		// Read progress and send as notifications
		for p := range progressCh {
			srv.SendNotification("model.pull.progress", map[string]interface{}{
				"model":     params.Model,
				"status":    p.Status,
				"completed": p.Completed,
				"total":     p.Total,
				"percent":   p.Percent,
			})
		}

		return transport.Response{Result: map[string]interface{}{
			"status": "started",
			"model":  params.Model,
		}}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	// Verify we got the response
	if !bytes.Contains(outBuf.Bytes(), []byte(`"started"`)) {
		t.Errorf("expected 'started' in response, got: %s", outBuf.String())
	}
}

// TestModelPullHandler_MissingModel verifies model.pull rejects empty model name.
func TestModelPullHandler_MissingModel(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"model.pull","params":{}}` + "\n"
	outBuf := new(bytes.Buffer)
	srv := transport.NewServer(bytes.NewBufferString(input), outBuf)

	srv.Handle("model.pull", func(req transport.Request) transport.Response {
		var params struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: err.Error(),
				},
			}
		}
		if params.Model == "" {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: "model parameter is required",
				},
			}
		}
		return transport.Response{Result: map[string]interface{}{"status": "started"}}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	lines := bytes.Split(outBuf.Bytes(), []byte("\n"))
	var resp transport.Response
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var parsed transport.Response
		if err := json.Unmarshal(line, &parsed); err != nil {
			continue
		}
		if parsed.ID != nil {
			resp = parsed
			break
		}
	}

	if resp.Error == nil {
		t.Fatal("expected error for missing model parameter")
	}
	if resp.Error.Message != "model parameter is required" {
		t.Errorf("expected 'model parameter is required', got %q", resp.Error.Message)
	}
}

// Ensure os package is used (for temp dir tests in other files)
var _ = os.TempDir
