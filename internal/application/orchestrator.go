package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gentleman/programas/harvey/internal/domain"
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
//   3. Response is parsed for <tool_call> XML blocks
//   4. If a tool call is found, the orchestrator:
//      a. Extracts the XML tool call
//      b. Calls ToolExecutor.Execute() for sandboxed execution
//      c. Formats the result as a <tool_result> XML block
//      d. Feeds the result back to Agent.Run() (loop)
//   5. When the LLM responds without a tool call, the loop ends
//
// Safety mechanisms:
//   - Max 5 iterations prevent infinite loops
//   - After 3 consecutive XML parse errors, the orchestrator
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
// executor handles XML parsing, permission checks, and tool routing.
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

		// Try to extract a tool call from the response
		toolXML, hasToolCall := extractToolCallXML(response)
		if !hasToolCall {
			// No tool call — this is the final response
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

		// Execute the tool through the sandboxed executor
		result, err := o.executor.Execute(ctx, session, toolXML)
		// Format the result as XML for the LLM
		resultXML := formatToolResult(result)
		session.Messages = append(session.Messages, domain.ChatMessage{
			Role:      "tool",
			Content:   resultXML,
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

// extractToolCallXML attempts to extract a <tool_call>...</tool_call> block
// from an LLM response. Returns the complete XML string and true if found.
func extractToolCallXML(response string) (string, bool) {
	oi := strings.Index(response, "<tool_call>")
	ci := strings.Index(response, "</tool_call>")
	if oi == -1 || ci == -1 || oi >= ci {
		return "", false
	}
	return response[oi : ci+len("</tool_call>")], true
}

// formatToolResult serializes a ToolResult into an XML <tool_result> block
// that the LLM can parse.
//
// On success:
//
//	<tool_result><output>...content...</output></tool_result>
//
// On failure:
//
//	<tool_result><error>...description...</error></tool_result>
func formatToolResult(result domain.ToolResult) string {
	if result.Success {
		return fmt.Sprintf("<tool_result>\n<output>%s</output>\n</tool_result>",
			xmlEscape(result.Output))
	}
	return fmt.Sprintf("<tool_result>\n<error>%s</error>\n</tool_result>",
		xmlEscape(result.Error))
}

// xmlEscape escapes special XML characters in text content.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// Agent returns the primary agent for inspection or health checks.
func (o *ChatOrchestrator) Agent() domain.Agent {
	return o.agent
}
