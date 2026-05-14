# Delta for tauri-desktop-shell

## ADDED Requirements

### Requirement: Companion Window Configuration

The system MUST support dynamic window configuration for dual modes: "companion" (always-on-top, right-edge anchored, full-height, resizable width 320-600px) and "floating" (centered, 420x700, not always-on-top). A Tauri command `set_companion_mode(bool)` MUST resize and reposition the window accordingly. A Tauri command `toggle_window()` MUST show or hide the window. The global shortcut Ctrl+Shift+G MUST be registered to call `toggle_window()`.

#### Scenario: Activate companion mode via Tauri command

- GIVEN the window is in floating mode (420x700 centered)
- WHEN `set_companion_mode(true)` is called
- THEN the window resizes to full height, anchors to the right edge, sets alwaysOnTop=true, and width is set to the last saved companion width (or 400px default)

#### Scenario: Toggle window visibility via global hotkey

- GIVEN the app is running with Ctrl+Shift+G registered
- WHEN the user presses Ctrl+Shift+G from any focused application
- THEN if the window is visible it hides; if hidden it shows and focuses

#### Scenario: Compact mode resize

- GIVEN the window is in companion mode
- WHEN the compact state is activated
- THEN the window width shrinks to approximately 40px while maintaining full height

## MODIFIED Requirements

### Requirement: Tauri Desktop Shell

The system MUST provide a Tauri v2 application window. The system tray SHALL include Show, Hide, and Quit menu items. The window chrome SHALL be minimal: G-MAN branding header with minimize and close buttons. The Svelte 5 frontend SHALL load via Tauri WebView. The Go sidecar binary MUST be spawned on startup; its liveness SHALL be monitored via exit-code polling and restarted within 2 seconds on crash. Closing the window SHALL minimize to tray, not quit the app. The window MUST support dynamic resizing and alwaysOnTop toggling for companion mode. The global shortcut Ctrl+Shift+G MUST be registered at startup for window toggle.
(Previously: Fixed 420x700 centered window with no alwaysOnTop or dynamic resize support.)

#### Scenario: Launch and sidecar startup

- GIVEN the user launches the G-MAN binary
- WHEN the app starts
- THEN a window appears with G-MAN branding, the system tray icon appears, the Go sidecar process spawns and passes health check, and the Svelte frontend renders in the WebView

#### Scenario: Tray show/hide toggle

- GIVEN the app is running and hidden to tray
- WHEN the user clicks "Show" in the system tray menu
- THEN the window restores to its last position, size, and mode (companion or floating)

#### Scenario: Sidecar crash recovery

- GIVEN the Go sidecar process is running
- WHEN it exits with a non-zero code unexpectedly
- THEN Tauri detects the exit within 500ms and spawns a new sidecar process within 2 seconds, logging the restart event

#### Scenario: Window close minimizes to tray

- GIVEN the app window is visible
- WHEN the user clicks the close button (X)
- THEN the window hides to system tray, the sidecar continues running, and Quit is only available via tray menu
