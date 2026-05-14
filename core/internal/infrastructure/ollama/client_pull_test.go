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

func TestOllamaClient_PullModel_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pull" {
			http.NotFound(w, r)
			return
		}

		// Verify request body
		var reqBody struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody.Model != "llama3.2:3b" {
			t.Errorf("expected model 'llama3.2:3b', got %q", reqBody.Model)
		}
		if !reqBody.Stream {
			t.Error("expected stream: true")
		}

		// Stream progress responses
		flusher := w.(http.Flusher)
		responses := []string{
			`{"status":"pulling manifest"}`,
			`{"status":"downloading","completed":100,"total":1000}`,
			`{"status":"downloading","completed":500,"total":1000}`,
			`{"status":"downloading","completed":1000,"total":1000}`,
			`{"status":"verifying sha256 digest"}`,
			`{"status":"success"}`,
		}
		for _, resp := range responses {
			w.Write([]byte(resp + "\n"))
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("test-model", nil, server.URL)
	ctx := context.Background()

	progressCh := make(chan ollama.PullProgress, 10)

	err := client.PullModel(ctx, "llama3.2:3b", progressCh)
	if err != nil {
		t.Fatalf("PullModel failed: %v", err)
	}
	close(progressCh)

	// Collect all progress events
	var events []ollama.PullProgress
	for p := range progressCh {
		events = append(events, p)
	}

	if len(events) < 3 {
		t.Fatalf("expected at least 3 progress events, got %d", len(events))
	}

	// First event should be "pulling manifest"
	if events[0].Status != "pulling manifest" {
		t.Errorf("first status = %q, want 'pulling manifest'", events[0].Status)
	}

	// Should have downloading events with progress
	var downloadEvents []ollama.PullProgress
	for _, e := range events {
		if e.Status == "downloading" {
			downloadEvents = append(downloadEvents, e)
		}
	}
	if len(downloadEvents) < 1 {
		t.Error("expected at least one downloading event")
	}

	// Last event should be success
	lastEvent := events[len(events)-1]
	if lastEvent.Status != "success" {
		t.Errorf("last status = %q, want 'success'", lastEvent.Status)
	}
}

func TestOllamaClient_PullModel_ConnectionError(t *testing.T) {
	// Use a server that closes immediately to simulate connection refused
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	client := ollama.NewOllamaClient("test-model", nil, server.URL)
	ctx := context.Background()

	progressCh := make(chan ollama.PullProgress, 1)

	err := client.PullModel(ctx, "nonexistent-model", progressCh)
	if err == nil {
		t.Fatal("expected error for connection refused, got nil")
	}
	if !strings.Contains(err.Error(), "refused") && !strings.Contains(err.Error(), "connect") {
		t.Errorf("error should mention connection issue, got: %v", err)
	}
}

func TestOllamaClient_PullModel_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("test-model", nil, server.URL)
	ctx := context.Background()

	progressCh := make(chan ollama.PullProgress, 1)

	err := client.PullModel(ctx, "nonexistent-model", progressCh)
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}
