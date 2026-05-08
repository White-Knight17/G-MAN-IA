// Package domain defines the core interfaces and value objects for G-MAN.
// These interfaces form the domain layer in a Clean Architecture, with
// zero dependencies on infrastructure, UI, or external libraries.
//
// Dependency rule: inner layers (domain) NEVER import from outer layers
// (application, infrastructure, UI).
package domain

import "context"

// StreamEvent represents a single event emitted during a streaming agent run.
// Events are sent through a channel and consumed by the transport layer
// for real-time delivery to the frontend.
//
// Type values:
//   - "token": a text token from the LLM (Content holds the token)
//   - "tool_call": the LLM requested a tool execution (Content holds JSON)
//   - "tool_result": a tool execution completed (Content holds the result)
//   - "done": the streaming run completed successfully
//   - "error": an error occurred (Error holds the message)
//   - "permission_request": a tool needs user permission (Content holds JSON)
type StreamEvent struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Agent encapsulates the ReAct (Reason + Act) agent loop.
// It receives user input within a session context, orchestrates LLM calls,
// parses XML tool-call responses, invokes tools, feeds results back to
// the LLM, and returns the final response.
//
// Implementations handle:
//   - Prompt construction (system prompt + history + user message)
//   - Token streaming via a return channel or streaming channel
//   - 30-second per-step timeout
//   - Retry on malformed output
//   - Model fallback on consecutive parse failures
type Agent interface {
	// Run executes the agent loop for the given input within the session.
	// It returns the final text response or an error if the loop fails.
	Run(ctx context.Context, input string, session *Session) (string, error)

	// StreamRun executes the agent loop with streaming output.
	// It returns a channel that emits StreamEvents (tokens, tool calls,
	// tool results, done/error) and an optional error if the initial
	// setup fails. The channel is closed when the stream is complete.
	StreamRun(ctx context.Context, input string, session *Session) (<-chan StreamEvent, error)

	// Tools returns the set of tools available to this agent.
	Tools() []Tool
}
