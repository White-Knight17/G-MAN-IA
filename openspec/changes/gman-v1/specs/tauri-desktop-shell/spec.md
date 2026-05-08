# tauri-desktop-shell Specification

## Purpose

Tauri v2 native desktop window with system tray, minimal chrome, and Go sidecar lifecycle management on Linux.

## Requirements

### Requirement: Tauri Desktop Shell

The system MUST provide a Tauri v2 application window. The system tray SHALL include Show, Hide, and Quit menu items. The window chrome SHALL be minimal: G-MAN branding header with minimize and close buttons. The Svelte 5 frontend SHALL load via Tauri WebView. The Go sidecar binary MUST be spawned on startup; its liveness SHALL be monitored via exit-code polling and restarted within 2 seconds on crash. Closing the window SHALL minimize to tray, not quit the app.

#### Scenario: Launch and sidecar startup

- GIVEN the user launches the G-MAN binary
- WHEN the app starts
- THEN a window appears with G-MAN branding, the system tray icon appears, the Go sidecar process spawns and passes health check, and the Svelte frontend renders in the WebView

#### Scenario: Tray show/hide toggle

- GIVEN the app is running and hidden to tray
- WHEN the user clicks "Show" in the system tray menu
- THEN the window restores to its last position and size
- AND clicking "Hide" minimizes it back to tray without terminating the sidecar

#### Scenario: Sidecar crash recovery

- GIVEN the Go sidecar process is running
- WHEN it exits with a non-zero code unexpectedly
- THEN Tauri detects the exit within 500ms and spawns a new sidecar process within 2 seconds, logging the restart event

#### Scenario: Window close minimizes to tray

- GIVEN the app window is visible
- WHEN the user clicks the close button (X)
- THEN the window hides to system tray, the sidecar continues running, and Quit is only available via tray menu
