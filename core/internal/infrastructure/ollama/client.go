// Package ollama provides an HTTP client adapter for Ollama's /api/chat endpoint.
// OllamaClient implements the domain.Agent interface, translating ReAct loop
// requests into non-streaming LLM calls and returning responses.
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gentleman/gman/internal/domain"
)

// Default parameters for the Ollama client.
const (
	DefaultBaseURL = "http://localhost:11434"
	DefaultModel   = "llama3.2:3b"
	DefaultTimeout = 120 * time.Second
)

// Predefined errors returned by the client.
var (
	ErrEmptyResponse  = fmt.Errorf("ollama: empty response from model")
	ErrNotOK          = fmt.Errorf("ollama: non-200 status code")
	ErrConnectionRefused = fmt.Errorf("ollama: connection refused — is Ollama running?")
)

// chatRequest is the JSON body sent to Ollama's /api/chat endpoint.
type chatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

// chatResponse is the JSON body returned by /api/chat in non-streaming mode.
type chatResponse struct {
	Message ollamaMessage `json:"message"`
	Error   string        `json:"error,omitempty"`
}

// ollamaMessage represents a single message in the conversation.
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaClient implements domain.Agent by calling the Ollama HTTP API.
// It builds the messages array from the session history, injects the
// system prompt with tool schemas, and sends non-streaming requests.
type OllamaClient struct {
	model      string
	tools      []domain.Tool
	baseURL    string
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama client with the given configuration.
// model is the Ollama model name (e.g., "llama3.2:3b").
// tools are the domain.Tool implementations available to the LLM.
// baseURL is the Ollama server URL (default: http://localhost:11434).
func NewOllamaClient(model string, tools []domain.Tool, baseURL string) *OllamaClient {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &OllamaClient{
		model:    model,
		tools:    tools,
		baseURL:  strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: DefaultTimeout},
	}
}

// StreamRun executes the agent loop with streaming output.
// It sends a streaming chat request to Ollama (stream: true), reads the
// NDJSON response body line-by-line, and emits StreamEvents containing
// token deltas. The channel is closed when the stream ends or on error.
//
// Implements domain.Agent.StreamRun().
func (c *OllamaClient) StreamRun(ctx context.Context, input string, session *domain.Session) (<-chan domain.StreamEvent, error) {
	messages := c.buildMessages(input, session)

	body := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal stream request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("ollama: create stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Drain and close body to allow connection reuse
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		ch := make(chan domain.StreamEvent, 2)
		ch <- domain.StreamEvent{
			Type:  "error",
			Error: fmt.Sprintf("ollama: API returned status %d", resp.StatusCode),
		}
		close(ch)
		return ch, nil
	}

	ch := make(chan domain.StreamEvent, 64)
	go c.readStream(ctx, resp.Body, ch)
	return ch, nil
}

// readStream reads NDJSON lines from the Ollama streaming response body
// and emits StreamEvents on the channel. It detects when a line is the
// final "done" message and closes the channel.
func (c *OllamaClient) readStream(ctx context.Context, body io.ReadCloser, ch chan<- domain.StreamEvent) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	// Ollama streaming responses can be large; set adequate buffer
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ch <- domain.StreamEvent{Type: "error", Error: ctx.Err().Error()}
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var streamResp struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			Done  bool   `json:"done"`
			Error string `json:"error,omitempty"`
		}

		if err := json.Unmarshal(line, &streamResp); err != nil {
			// Skip malformed lines, but don't crash
			continue
		}

		if streamResp.Error != "" {
			ch <- domain.StreamEvent{Type: "error", Error: streamResp.Error}
			return
		}

		if streamResp.Done {
			ch <- domain.StreamEvent{Type: "done"}
			return
		}

		content := streamResp.Message.Content
		if content != "" {
			ch <- domain.StreamEvent{Type: "token", Content: content}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- domain.StreamEvent{Type: "error", Error: fmt.Sprintf("ollama: stream read error: %v", err)}
	}
}

// Tools returns the set of tools available to this agent.
// Implements domain.Agent.Tools().
func (c *OllamaClient) Tools() []domain.Tool {
	return c.tools
}

// Run executes the agent loop by sending the session context to Ollama
// and returning the model's response. It builds the full messages array
// from the session history, prepends the system prompt, and appends the
// current user input (if non-empty).
//
// On the first call, input contains the user's message.
// On subsequent ReAct loop iterations, input may be empty — the session
// history already contains the tool call and tool result messages.
//
// Implements domain.Agent.Run().
func (c *OllamaClient) Run(ctx context.Context, input string, session *domain.Session) (string, error) {
	messages := c.buildMessages(input, session)

	body := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("ollama: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("ollama: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: API returned status %d", resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("ollama: decode response: %w", err)
	}

	if chatResp.Error != "" {
		return "", fmt.Errorf("ollama: API error: %s", chatResp.Error)
	}

	content := strings.TrimSpace(chatResp.Message.Content)
	if content == "" {
		return "", ErrEmptyResponse
	}

	return content, nil
}

// buildMessages constructs the messages array for the Ollama API call.
// It includes:
//   1. The system prompt with tool schemas and instructions
//   2. The session conversation history
//   3. The current user input (if non-empty)
func (c *OllamaClient) buildMessages(input string, session *domain.Session) []ollamaMessage {
	messages := make([]ollamaMessage, 0, len(session.Messages)+2)

	// System prompt always goes first
	messages = append(messages, ollamaMessage{
		Role:    "system",
		Content: c.buildSystemPrompt(),
	})

	// Session history
	for _, msg := range session.Messages {
		messages = append(messages, ollamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Current user input (only if non-empty — on ReAct iterations this may be empty)
	if input != "" {
		messages = append(messages, ollamaMessage{
			Role:    "user",
			Content: input,
		})
	}

	return messages
}

// buildSystemPrompt constructs a lightweight system prompt that instructs
// the LLM to use simple text-based commands instead of heavy XML schemas.
// This reduces prompt tokens significantly (~300 vs ~2000 for XML),
// which is critical for CPU-only small models like llama3.2:3b.
func (c *OllamaClient) buildSystemPrompt() string {
	var sb strings.Builder

	sb.WriteString("You are G-MAN, a friendly Linux assistant for Arch Linux + Hyprland users.\n")
	sb.WriteString("You help configure dotfiles, explain settings, and run safe commands.\n\n")

	if len(c.tools) > 0 {
		sb.WriteString("When you need to perform an action, use these commands on their own line:\n")
		sb.WriteString("READ: /path/to/file — read a file\n")
		sb.WriteString("WRITE: /path/to/file — write new content (content on next lines, end with END)\n")
		sb.WriteString("LIST: /path/to/dir — list directory contents\n")
		sb.WriteString("RUN: command — run a safe command\n")
		sb.WriteString("CHECK: filetype — check config syntax (hyprland, waybar, or bash). Content on next lines, end with END\n")
		sb.WriteString("SEARCH: query — search local wiki for information\n\n")
		sb.WriteString("IMPORTANT:\n")
		sb.WriteString("- Only use commands when you need to read or modify files\n")
		sb.WriteString("- Explain what you're doing before using a command\n")
		sb.WriteString("- Never use RUN for dangerous commands (rm, sudo, etc.)\n")
		sb.WriteString("- For WRITE and CHECK, put the content on the lines after the command, then END on its own line\n")
		sb.WriteString("- Be concise and helpful\n")
	} else {
		sb.WriteString("You do not have any tools available. Answer the user directly.\n")
	}

	return sb.String()
}

// HealthCheck verifies that Ollama is running and the configured model is available.
// It calls GET /api/tags and checks if c.model is in the response.
// Returns an error if Ollama is unreachable or the model is not found.
func (c *OllamaClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("health check: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check: %w: status %d", ErrNotOK, resp.StatusCode)
	}

	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return fmt.Errorf("health check: decode response: %w", err)
	}

	for _, m := range tagsResp.Models {
		if strings.HasPrefix(m.Name, c.model) {
			return nil
		}
	}

	return fmt.Errorf("health check: model %q not found — run 'ollama pull %s'", c.model, c.model)
}
