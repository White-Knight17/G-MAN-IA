package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gentleman/gman/internal/domain"
)

// Default configuration for the ChatOrchestrator.
const (
	DefaultMaxIterations       = 5    // prevent infinite agent loops
	DefaultMaxConsecutiveParseErrors = 3 // trigger model fallback threshold
)

// ChatOrchestrator implements the ReAct (Reason + Act) agent loop.
// It coordinates the full user→LLM→tools→LLM cycle:
//   1. User sends a message
//   2. Orchestrator calls Agent.Run() to get the LLM response
//   3. Response is scanned for text-based tool commands (READ:, WRITE:, etc.)
//   4. If a tool command is found, the orchestrator:
//      a. Passes the full response to ToolExecutor.Execute()
//      b. Formats the result as plain text for the LLM
//      c. Feeds the result back to Agent.Run() (loop)
//   5. When the LLM responds without a tool command, the loop ends
//
// Safety mechanisms:
//   - Max 5 iterations prevent infinite loops
//   - After 3 consecutive parse errors, the orchestrator
//     can switch to a fallback (model fallback support)
//   - 30-second per-tool timeout enforced by ToolExecutor
type ChatOrchestrator struct {
	agent         domain.Agent
	executor      *ToolExecutor
	grantMgr      *GrantManager
	maxIterations int
	parseErrors   int    // consecutive parse failure counter
	fallback      domain.Agent // optional fallback agent
}

// ChatOrchestratorOption is a functional option for configuring the orchestrator.
type ChatOrchestratorOption func(*ChatOrchestrator)

// WithMaxIterations sets the maximum number of ReAct loop iterations.
func WithMaxIterations(n int) ChatOrchestratorOption {
	return func(o *ChatOrchestrator) {
		o.maxIterations = n
	}
}

// WithFallback sets a fallback agent for model failover.
func WithFallback(agent domain.Agent) ChatOrchestratorOption {
	return func(o *ChatOrchestrator) {
		o.fallback = agent
	}
}

// NewChatOrchestrator creates a ChatOrchestrator with constructor-injected dependencies.
// agent is the primary LLM adapter (e.g., OllamaClient).
// executor handles text command parsing, permission checks, and tool routing.
// grantMgr manages session-scoped permission grants.
func NewChatOrchestrator(agent domain.Agent, executor *ToolExecutor, grantMgr *GrantManager, opts ...ChatOrchestratorOption) *ChatOrchestrator {
	o := &ChatOrchestrator{
		agent:         agent,
		executor:      executor,
		grantMgr:      grantMgr,
		maxIterations: DefaultMaxIterations,
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

// HandleMessage processes a user message through the full ReAct agent loop.
// It appends the user message to the session, iterates through agent runs
// and tool executions, and returns the final response.
//
// Returns the final text response from the agent, or an error if the loop
// fails or reaches the maximum number of iterations.
func (o *ChatOrchestrator) HandleMessage(ctx context.Context, session *domain.Session, userInput string) (string, error) {
	// Add user message to session history
	session.Messages = append(session.Messages, domain.ChatMessage{
		Role:      "user",
		Content:   userInput,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})

	input := userInput

	for i := 0; i < o.maxIterations; i++ {
		response, err := o.agent.Run(ctx, input, session)
		if err != nil {
			o.parseErrors++
			// Check if we should switch to fallback
			if o.parseErrors >= DefaultMaxConsecutiveParseErrors && o.fallback != nil {
				return o.tryFallback(ctx, session, input)
			}
			return "", fmt.Errorf("orchestrator: agent run failed (iteration %d): %w", i+1, err)
		}

		// Try to extract a tool command from the response
		if !hasToolCommand(response) {
			// No tool command — this is the final response
			o.parseErrors = 0
			session.Messages = append(session.Messages, domain.ChatMessage{
				Role:      "assistant",
				Content:   response,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
			return response, nil
		}

		// Add the assistant's tool-call request to session history
		session.Messages = append(session.Messages, domain.ChatMessage{
			Role:      "assistant",
			Content:   response,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})

		// Execute the tool through the sandboxed executor (full response)
		result, err := o.executor.Execute(ctx, session, response)
		// Format the result as plain text for the LLM
		resultText := formatToolResult(result)
		session.Messages = append(session.Messages, domain.ChatMessage{
			Role:      "tool",
			Content:   resultText,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})

		if err != nil {
			// Tool execution failed — feed the error back to the agent
			// The agent may try a different approach or report the error
			o.parseErrors++
		} else {
			o.parseErrors = 0
		}

		// On subsequent iterations, no new user input — the session history
		// already contains the tool call and tool result
		input = ""
	}

	return "", fmt.Errorf("orchestrator: max iterations (%d) reached without final response", o.maxIterations)
}

// tryFallback attempts the agent loop with the fallback agent.
// This is triggered when the primary agent fails 3 consecutive parse errors.
func (o *ChatOrchestrator) tryFallback(ctx context.Context, session *domain.Session, input string) (string, error) {
	// Reset parse errors for fallback attempt
	o.parseErrors = 0

	response, err := o.fallback.Run(ctx, input, session)
	if err != nil {
		return "", fmt.Errorf("orchestrator: fallback agent also failed: %w", err)
	}
	return response, nil
}

// hasToolCommand checks whether an LLM response contains a text-based
// tool command (READ:, WRITE:, LIST:, RUN:, CHECK:, or SEARCH: on a line).
// Returns true if at least one recognized command keyword is found.
func hasToolCommand(response string) bool {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for any recognized command prefix
		for _, cmd := range []string{"READ:", "WRITE:", "LIST:", "RUN:", "CHECK:", "SEARCH:"} {
			upperTrimmed := strings.ToUpper(trimmed)
			if strings.HasPrefix(upperTrimmed, cmd) {
				return true
			}
		}
	}
	return false
}

// formatToolResult serializes a ToolResult into a plain-text message
// that the LLM can understand.
//
// On success:
//
//	Tool result: ...output...
//
// On failure:
//
//	Tool error: ...description...
func formatToolResult(result domain.ToolResult) string {
	if result.Success {
		return fmt.Sprintf("Tool result:\n%s", result.Output)
	}
	return fmt.Sprintf("Tool error: %s", result.Error)
}

// HandleMessageStream processes a user message through the full ReAct agent
// loop with streaming output. It emits StreamEvents through the returned
// channel, passing through agent tokens and intercepting tool calls for
// execution.
//
// The channel is closed when the streaming loop completes or the maximum
// number of iterations is reached.
func (o *ChatOrchestrator) HandleMessageStream(ctx context.Context, session *domain.Session, userInput string) (<-chan domain.StreamEvent, error) {
	// Add user message to session history
	session.Messages = append(session.Messages, domain.ChatMessage{
		Role:      "user",
		Content:   userInput,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})

	ch := make(chan domain.StreamEvent, 64)

	go func() {
		defer close(ch)

		for i := 0; i < o.maxIterations; i++ {
			select {
			case <-ctx.Done():
				ch <- domain.StreamEvent{Type: "error", Error: ctx.Err().Error()}
				return
			default:
			}

			// Call agent StreamRun
			events, err := o.agent.StreamRun(ctx, userInput, session)
			if err != nil {
				ch <- domain.StreamEvent{Type: "error", Error: err.Error()}
				return
			}

			var fullResponse string
			var toolCallSeen bool

			for evt := range events {
				switch evt.Type {
				case "token":
					fullResponse += evt.Content
					ch <- evt
				case "tool_call":
					toolCallSeen = true
					ch <- evt

					// Execute the tool and emit tool_result
					result, execErr := o.executor.Execute(ctx, session, evt.Content)
					if execErr != nil {
						ch <- domain.StreamEvent{
							Type:    "tool_result",
							Content: result.Error,
							Error:   execErr.Error(),
						}
					} else {
						ch <- domain.StreamEvent{
							Type:    "tool_result",
							Content: result.Output,
						}
					}
				case "done":
					ch <- evt
				case "error":
					ch <- evt
				}
			}

			// If no tool was called and we got tokens, this was the final response
			if !toolCallSeen && fullResponse != "" {
				return
			}

			// Reset input for next iteration (session already has tool/result messages)
			userInput = ""
		}

		ch <- domain.StreamEvent{Type: "error", Error: "max iterations reached"}
	}()

	return ch, nil
}

// Agent returns the primary agent for inspection or health checks.
func (o *ChatOrchestrator) Agent() domain.Agent {
	return o.agent
}
