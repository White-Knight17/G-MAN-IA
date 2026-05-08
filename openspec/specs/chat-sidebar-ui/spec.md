# chat-sidebar-ui Specification

## Purpose

Svelte 5 conversational sidebar interface with streaming chat bubbles, file preview panel, permission grant dialogs, and keyboard shortcuts.

## Requirements

### Requirement: Chat Sidebar UI

The system MUST provide a Svelte 5 sidebar chat interface. Messages SHALL render as chat bubbles with distinct user (right-aligned, accent color) and assistant (left-aligned, neutral) styling. While Go processes a response, a typing indicator (animated dots) SHALL appear in the assistant bubble. A file preview panel SHALL display the last file content G-MAN read or wrote with syntax highlighting. Permission grant requests SHALL appear as modal overlay dialogs with the directory path and Allow/Deny buttons. The chat SHALL auto-scroll to the latest message. The sidebar visibility MUST be togglable via Ctrl+Shift+G.

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
