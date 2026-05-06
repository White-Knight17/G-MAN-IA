package domain

// Session represents a single interactive session between the user and the agent.
// It holds the conversation history, active permission grants, and session metadata.
// Sessions are ephemeral — all data is discarded on process exit.
type Session struct {
	// ID uniquely identifies this session (UUID v4 recommended).
	ID string

	// Messages holds the full conversation history in chronological order.
	// Includes system, user, assistant, and tool messages.
	Messages []ChatMessage

	// Grants holds the active permission grants for this session.
	// Populated by the GrantManager as the user approves tool access requests.
	Grants []Grant

	// StartedAt records when the session was created.
	StartedAt string // ISO 8601 timestamp
}

// ChatMessage represents a single message in the conversation history.
// Maps directly to the Ollama /api/chat message format (system, user, assistant, tool).
type ChatMessage struct {
	// Role identifies the message author.
	// Valid values: "system", "user", "assistant", "tool"
	Role string

	// Content is the message body.
	// For user messages: the raw input text.
	// For assistant messages: the LLM's response (may include <tool_call> XML).
	// For tool messages: the serialized ToolResult.
	// For system messages: the initial system prompt.
	Content string

	// Timestamp records when the message was created.
	Timestamp string // ISO 8601 timestamp
}
