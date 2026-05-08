package tui

import "github.com/gentleman/gman/internal/domain"

// Custom message types for async communication between goroutines
// and the Bubbletea event loop. These are sent via tea.Cmd and received
// in the Update method.

// orchestratorResponseMsg is sent when the ChatOrchestrator successfully
// returns a response from the LLM.
type orchestratorResponseMsg struct {
	response string
}

// orchestratorErrorMsg is sent when the ChatOrchestrator encounters
// an error during the ReAct loop.
type orchestratorErrorMsg struct {
	err error
}

// GrantRequestMsg is sent when the permission system requests a grant
// that the user hasn't approved yet. Triggers the grant dialog modal.
// Exported for testing.
type GrantRequestMsg struct {
	Grant domain.Grant
}

// grantResponseMsg is sent when the user responds to a grant dialog
// (either "y" to allow or "n" to deny).
type grantResponseMsg struct {
	approved bool
}
