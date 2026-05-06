// Package domain defines the core interfaces and value objects for Harvey.
// These interfaces form the domain layer in a Clean Architecture, with
// zero dependencies on infrastructure, UI, or external libraries.
//
// Dependency rule: inner layers (domain) NEVER import from outer layers
// (application, infrastructure, UI).
package domain

import "context"

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

	// Tools returns the set of tools available to this agent.
	Tools() []Tool
}
