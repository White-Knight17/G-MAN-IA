# Delta for chat-sidebar-ui

## ADDED Requirements

### Requirement: Slash Command Input Handling

The system MUST detect input starting with `/` in the chat input field and intercept it as a command. A command palette overlay SHALL appear showing matching commands as the user types. Known commands (`/model`, `/models`, `/api`, `/clear`, `/help`, `/ollamamodel`) MUST be dispatched to their handlers; unknown inputs MUST show an error suggestion. Command parsing MUST be case-insensitive.

#### Scenario: Command palette appears on slash

- GIVEN the chat input field is empty and focused
- WHEN the user types `/`
- THEN a command palette overlay appears listing all available commands with descriptions

#### Scenario: Command palette filters on partial match

- GIVEN the command palette is visible
- WHEN the user types `/mo`
- THEN the palette filters to show only `/model` and `/models`

#### Scenario: Unknown command shows error

- GIVEN the user types `/foobar` and presses Enter
- WHEN the input is processed
- THEN an error toast or inline message appears: "Unknown command. Type `/help` for available commands."

### Requirement: Material UI Elevation and Spacing

Message bubbles MUST use elevation (box-shadow) instead of flat backgrounds. Buttons MUST have hover/active states (ripple or scale effect). The layout MUST follow an 8px spacing grid. Typography MUST use consistent hierarchy (title, body, caption sizes). The color palette MUST remain Tokyo Night inspired (dark: #1a1b26, surface: #24283b, accent: #3b82f6). Light theme MUST also receive Material elevation treatment.

#### Scenario: Message bubbles show elevation

- GIVEN a message is rendered in the chat
- WHEN the message bubble is visible
- THEN it has a subtle box-shadow (elevation) distinguishing it from the background

#### Scenario: Button hover state

- GIVEN a button is visible in the UI (send, allow, deny)
- WHEN the user hovers over it
- THEN the button shows a visual change (scale, shadow, or color shift)

#### Scenario: 8px grid spacing

- GIVEN any UI section (input bar, message list, file preview)
- WHEN spacing is measured
- THEN all margins and padding are multiples of 8px

## MODIFIED Requirements

### Requirement: Chat Sidebar UI

The system MUST provide a Svelte 5 sidebar chat interface. Messages SHALL render as chat bubbles with distinct user (right-aligned, accent color) and assistant (left-aligned, neutral) styling. While Go processes a response, a typing indicator (animated dots) SHALL appear in the assistant bubble. A file preview panel SHALL display the last file content G-MAN read or wrote with syntax highlighting. Permission grant requests SHALL appear as modal overlay dialogs with the directory path and Allow/Deny buttons. The chat SHALL auto-scroll to the latest message. The sidebar visibility MUST be togglable via Ctrl+Shift+G. The input field MUST support slash command parsing with a command palette overlay. The UI MUST support two display modes: companion (full-height edge sidebar) and floating (centered window). Message bubbles MUST use Material elevation styling.
(Previously: Single-mode floating sidebar with flat bubble backgrounds and no slash command support.)

#### Scenario: Send message and receive streaming response

- GIVEN the sidebar is visible and the sidecar is connected
- WHEN the user types "Show my Hyprland config" and presses Enter
- THEN a user bubble appears immediately with the message, a typing indicator appears in the assistant bubble, tokens stream into the assistant bubble, the typing indicator disappears on completion, and the view auto-scrolls to the latest content

#### Scenario: File preview panel updates

- GIVEN G-MAN executes a `read_file` or `write_file` tool
- WHEN the tool result arrives
- THEN the file preview panel updates to show the file content with syntax highlighting (language detected from extension) and a copy button

#### Scenario: Permission grant modal

- GIVEN a tool requests access to `~/.config/hypr/` and no grant exists
- WHEN the permission system emits a grant request
- THEN a modal overlay appears showing the path, requested mode (ro/rw), and Allow/Deny buttons; clicking Allow saves the grant for the session; clicking Deny returns an error to the tool

#### Scenario: Keyboard shortcut toggle

- GIVEN the app window is focused
- WHEN the user presses Ctrl+Shift+G
- THEN if the sidebar is visible it hides; if hidden it slides in from the right edge
