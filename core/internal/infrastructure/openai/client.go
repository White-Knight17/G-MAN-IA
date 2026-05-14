// Package openai provides an HTTP client adapter for OpenAI-compatible APIs.
// It implements domain.Agent, translating ReAct loop requests into chat
// completion requests and returning streaming responses.
//
// Supports: OpenAI, DeepSeek, Groq, and any OpenAI-compatible API.
package openai

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

// Default parameters for the OpenAI client.
const (
	DefaultTimeout = 120 * time.Second
)

// Predefined base URLs for known providers.
var DefaultBaseURLs = map[string]string{
	"openai":   "https://api.openai.com",
	"deepseek": "https://api.deepseek.com",
	"groq":     "https://api.groq.com/openai",
}

// chatMessage represents a message in the OpenAI chat format.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest is the JSON body sent to the chat completions endpoint.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// Non-streaming response
type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Streaming response chunk
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

// Client implements domain.Agent for OpenAI-compatible APIs.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	tools      []domain.Tool
	httpClient *http.Client
}

// NewClient creates a new OpenAI-compatible client.
// apiKey is the provider API key.
// baseURL is the API base URL (e.g., "https://api.openai.com").
// model is the model name (e.g., "gpt-4o", "deepseek-chat").
func NewClient(apiKey, baseURL, model string, tools []domain.Tool) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		tools:      tools,
		httpClient: &http.Client{Timeout: DefaultTimeout},
	}
}

// StreamRun executes the agent loop with streaming output via SSE.
func (c *Client) StreamRun(ctx context.Context, input string, session *domain.Session) (<-chan domain.StreamEvent, error) {
	messages := c.buildMessages(input, session)

	body := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal stream request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("openai: create stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		ch := make(chan domain.StreamEvent, 2)
		ch <- domain.StreamEvent{
			Type:  "error",
			Error: fmt.Sprintf("openai: API returned status %d: %s", resp.StatusCode, string(body)),
		}
		close(ch)
		return ch, nil
	}

	ch := make(chan domain.StreamEvent, 64)
	go c.readSSEStream(ctx, resp.Body, ch)
	return ch, nil
}

// readSSEStream reads SSE (Server-Sent Events) from the streaming response.
func (c *Client) readSSEStream(ctx context.Context, body io.ReadCloser, ch chan<- domain.StreamEvent) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ch <- domain.StreamEvent{Type: "error", Error: ctx.Err().Error()}
			return
		default:
		}

		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			ch <- domain.StreamEvent{Type: "done"}
			return
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				ch <- domain.StreamEvent{Type: "token", Content: content}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- domain.StreamEvent{Type: "error", Error: fmt.Sprintf("openai: stream read error: %v", err)}
	}
}

// Run executes a non-streaming agent call.
func (c *Client) Run(ctx context.Context, input string, session *domain.Session) (string, error) {
	messages := c.buildMessages(input, session)

	body := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("openai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("openai: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai: API returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("openai: decode response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("openai: API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("openai: empty response from model")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

// Tools returns the set of tools available to this agent.
func (c *Client) Tools() []domain.Tool {
	return c.tools
}

// buildMessages constructs the messages array for the OpenAI API call.
func (c *Client) buildMessages(input string, session *domain.Session) []chatMessage {
	messages := make([]chatMessage, 0, len(session.Messages)+2)

	// System prompt
	messages = append(messages, chatMessage{
		Role:    "system",
		Content: c.buildSystemPrompt(),
	})

	// Session history
	for _, msg := range session.Messages {
		messages = append(messages, chatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Current user input
	if input != "" {
		messages = append(messages, chatMessage{
			Role:    "user",
			Content: input,
		})
	}

	return messages
}

// buildSystemPrompt constructs the system prompt.
func (c *Client) buildSystemPrompt() string {
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
