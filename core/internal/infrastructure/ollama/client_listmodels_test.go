package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gentleman/gman/internal/infrastructure/ollama"
)

func TestOllamaClient_ListModels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]interface{}{
				{"name": "llama3.2:3b", "size": 2032891662, "digest": "abc123"},
				{"name": "qwen2.5:3b", "size": 1876543210, "digest": "def456"},
			},
		})
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("llama3.2:3b", nil, server.URL)
	ctx := context.Background()

	models, err := client.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	if models[0].Name != "llama3.2:3b" {
		t.Errorf("model[0].Name = %q, want 'llama3.2:3b'", models[0].Name)
	}
	if models[1].Name != "qwen2.5:3b" {
		t.Errorf("model[1].Name = %q, want 'qwen2.5:3b'", models[1].Name)
	}
	if !strings.Contains(models[0].Size, "GB") && !strings.Contains(models[0].Size, "MB") {
		t.Errorf("model[0].Size should be human-readable, got %q", models[0].Size)
	}
	if models[0].Digest != "abc123" {
		t.Errorf("model[0].Digest = %q, want 'abc123'", models[0].Digest)
	}
}

func TestOllamaClient_ListModels_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []interface{}{},
		})
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("test-model", nil, server.URL)
	ctx := context.Background()

	models, err := client.ListModels(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(models) != 0 {
		t.Errorf("expected 0 models, got %d", len(models))
	}
}

func TestOllamaClient_ListModels_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("test-model", nil, server.URL)
	ctx := context.Background()

	_, err := client.ListModels(ctx)
	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
}
