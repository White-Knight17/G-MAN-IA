package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gentleman/gman/internal/domain"
	"github.com/gentleman/gman/internal/infrastructure/ollama"
)

// fakeTool is a minimal domain.Tool implementation for testing.
type fakeTool struct {
	name        string
	description string
	schema      string
}

func (f *fakeTool) Name() string                               { return f.name }
func (f *fakeTool) Description() string                        { return f.description }
func (f *fakeTool) SchemaXML() string                           { return f.schema }
func (f *fakeTool) Execute(ctx context.Context, params map[string]string) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Output: "fake result"}, nil
}

func newFakeTool(name string) *fakeTool {
	return &fakeTool{
		name:        name,
		description: "A test tool called " + name,
		schema:      "<tool_call><name>" + name + "</name><path>/some/path</path></tool_call>",
	}
}

// fakeSession creates a minimal session for testing.
func fakeSession() *domain.Session {
	return &domain.Session{
		ID:        "test-session",
		StartedAt: "2026-01-01T00:00:00Z",
	}
}

func TestOllamaClient_Run_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			http.NotFound(w, r)
			return
		}

		var reqBody struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			Stream bool `json:"stream"`
		}
		json.NewDecoder(r.Body).Decode(&reqBody)

		resp := map[string]interface{}{
			"model": reqBody.Model,
			"message": map[string]string{
				"role":    "assistant",
				"content": "Hello! I'm G-MAN. How can I help with your config?",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("test-model", nil, server.URL)
	ctx := context.Background()

	resp, err := client.Run(ctx, "Hi!", fakeSession())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == "" {
		t.Fatal("expected non-empty response")
	}
	if !strings.Contains(resp, "G-MAN") {
		t.Errorf("expected response to mention G-MAN, got: %q", resp)
	}
}

func TestOllamaClient_Run_Errors(t *testing.T) {
	tests := []struct {
		name       string
		setupServer func() *httptest.Server
		input      string
		wantErr    bool
		errSubstr  string
	}{
		{
			name: "connection refused",
			setupServer: func() *httptest.Server {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				s.Close() // close it to simulate connection refused
				return s
			},
			input:     "Hi!",
			wantErr:   true,
			errSubstr: "refused",
		},
		{
			name: "HTTP 500 error",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			input:     "Hi!",
			wantErr:   true,
			errSubstr: "status 500",
		},
		{
			name: "empty response body",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"message": map[string]string{"role": "assistant", "content": ""},
					})
				}))
			},
			input:     "Hi!",
			wantErr:   true,
			errSubstr: "empty",
		},
		{
			name: "API error in response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"message": map[string]string{"role": "assistant", "content": ""},
						"error":   "model not found",
					})
				}))
			},
			input:     "Hi!",
			wantErr:   true,
			errSubstr: "model not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			var baseURL string
			if tt.name == "connection refused" {
				baseURL = server.URL
			} else {
				baseURL = server.URL
			}

			client := ollama.NewOllamaClient("test-model", nil, baseURL)
			ctx := context.Background()

			_, err := client.Run(ctx, tt.input, fakeSession())
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errSubstr)) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestOllamaClient_SystemPrompt(t *testing.T) {
	var capturedMessages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&reqBody)
		capturedMessages = reqBody.Messages
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": map[string]string{"role": "assistant", "content": "OK"},
		})
	}))
	defer server.Close()

	tools := []domain.Tool{
		newFakeTool("read_file"),
		newFakeTool("write_file"),
	}

	client := ollama.NewOllamaClient("test-model", tools, server.URL)
	ctx := context.Background()

	_, err := client.Run(ctx, "read my config", fakeSession())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedMessages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(capturedMessages))
	}

	// First message should be system prompt
	if capturedMessages[0].Role != "system" {
		t.Errorf("first message role = %q, want 'system'", capturedMessages[0].Role)
	}

	sysPrompt := capturedMessages[0].Content
	if !strings.Contains(sysPrompt, "G-MAN") {
		t.Error("system prompt should mention G-MAN")
	}
	if !strings.Contains(sysPrompt, "READ:") {
		t.Error("system prompt should include READ: command format")
	}
	if !strings.Contains(sysPrompt, "WRITE:") {
		t.Error("system prompt should include WRITE: command format")
	}
	if !strings.Contains(sysPrompt, "LIST:") {
		t.Error("system prompt should include LIST: command format")
	}
	if !strings.Contains(sysPrompt, "RUN:") {
		t.Error("system prompt should include RUN: command format")
	}
	if !strings.Contains(sysPrompt, "CHECK:") {
		t.Error("system prompt should include CHECK: command format")
	}
	if !strings.Contains(sysPrompt, "SEARCH:") {
		t.Error("system prompt should include SEARCH: command format")
	}
	if !strings.Contains(sysPrompt, "END") {
		t.Error("system prompt should mention END marker")
	}

	// Second message should be user input
	if capturedMessages[1].Role != "user" {
		t.Errorf("second message role = %q, want 'user'", capturedMessages[1].Role)
	}
	if capturedMessages[1].Content != "read my config" {
		t.Errorf("user message = %q, want 'read my config'", capturedMessages[1].Content)
	}
}

func TestOllamaClient_SessionHistory(t *testing.T) {
	var capturedMessages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&reqBody)
		capturedMessages = reqBody.Messages
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": map[string]string{"role": "assistant", "content": "OK"},
		})
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("test-model", nil, server.URL)
	ctx := context.Background()

	session := fakeSession()
	session.Messages = []domain.ChatMessage{
		{Role: "user", Content: "what's my config?", Timestamp: "t1"},
		{Role: "assistant", Content: "which file?", Timestamp: "t2"},
	}

	_, err := client.Run(ctx, "hyprland.conf", session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: system + 2 history + 1 user = 4 messages
	if len(capturedMessages) != 4 {
		t.Fatalf("expected 4 messages, got %d: %+v", len(capturedMessages), capturedMessages)
	}

	if capturedMessages[1].Role != "user" {
		t.Errorf("message 1 role = %q, want 'user'", capturedMessages[1].Role)
	}
	if capturedMessages[2].Role != "assistant" {
		t.Errorf("message 2 role = %q, want 'assistant'", capturedMessages[2].Role)
	}
}

func TestOllamaClient_EmptyTools(t *testing.T) {
	var sysPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&reqBody)
		sysPrompt = reqBody.Messages[0].Content
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": map[string]string{"role": "assistant", "content": "OK"},
		})
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("test-model", nil, server.URL)
	ctx := context.Background()

	_, err := client.Run(ctx, "Hi!", fakeSession())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sysPrompt, "do not have any tools") {
		t.Errorf("empty tools system prompt should mention no tools, got: %q", sysPrompt)
	}
}

func TestOllamaClient_Tools(t *testing.T) {
	tools := []domain.Tool{
		newFakeTool("read_file"),
		newFakeTool("write_file"),
	}

	client := ollama.NewOllamaClient("test-model", tools, "http://localhost:11434")
	result := client.Tools()

	if len(result) != 2 {
		t.Errorf("Tools() returned %d tools, want 2", len(result))
	}
}

func TestOllamaClient_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []map[string]string{
					{"name": "llama3.2:3b"},
					{"name": "qwen2.5:3b"},
				},
			})
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("llama3.2:3b", nil, server.URL)
	ctx := context.Background()

	err := client.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck failed for existing model: %v", err)
	}
}

func TestOllamaClient_HealthCheck_ModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]string{
				{"name": "llama3.2:3b"},
			},
		})
	}))
	defer server.Close()

	client := ollama.NewOllamaClient("qwen2.5:3b", nil, server.URL)
	ctx := context.Background()

	err := client.HealthCheck(ctx)
	if err == nil {
		t.Error("HealthCheck should fail for missing model")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestOllamaClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	client := ollama.NewOllamaClient("test-model", nil, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.Run(ctx, "Hi!", fakeSession())
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
