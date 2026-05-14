# companion-window Specification

## Purpose

Always-on-top resizable sidebar with compact mode, hotkey toggle, and floating/companion mode switching for the G-MAN desktop app.

## Requirements

### Requirement: Dual Window Modes

The system MUST support two window modes: "companion" (full-height right-edge sidebar, always-on-top) and "floating" (centered 420x700 window). The default mode on first launch SHALL be "floating". The user MUST be able to switch between modes via settings or a command.

#### Scenario: Switch to companion mode

- GIVEN the app is running in floating mode (420x700 centered)
- WHEN the user switches to companion mode via settings or command
- THEN the window resizes to full screen height, anchors to the right edge, becomes always-on-top, and its width is between 320px and 600px

#### Scenario: Switch to floating mode

- GIVEN the app is running in companion mode
- WHEN the user switches to floating mode
- THEN the window resizes to 420x700, centers on screen, and loses always-on-top status

#### Scenario: Default mode on first launch

- GIVEN the app has never been launched before
- WHEN the app starts
- THEN the window opens in floating mode (420x700 centered)

### Requirement: Global Hotkey Toggle

The system MUST register a global hotkey (Ctrl+Shift+G) that shows or hides the window regardless of which application is focused. The hotkey MUST work in both companion and floating modes.

#### Scenario: Show hidden window via hotkey

- GIVEN the app is running and the window is hidden
- WHEN the user presses Ctrl+Shift+G from any application
- THEN the window becomes visible and focused

#### Scenario: Hide visible window via hotkey

- GIVEN the app window is visible and focused
- WHEN the user presses Ctrl+Shift+G
- THEN the window hides but the app continues running in the system tray

### Requirement: Companion Mode Resizable Width

The system MUST allow the user to resize the window width in companion mode between 320px (minimum) and 600px (maximum). The height MUST remain fixed at full screen height.

#### Scenario: Resize companion window within bounds

- GIVEN the app is in companion mode
- WHEN the user drags the left edge of the window
- THEN the width changes between 320px and 600px, and the height remains full screen

#### Scenario: Resize hits minimum bound

- GIVEN the companion window is at 320px width
- WHEN the user attempts to shrink it further
- THEN the width stays at 320px

### Requirement: Compact Collapsed State

The system MUST support a compact/collapsed state where the window reduces to a thin bar (~40px wide) at the right edge. The window MUST expand back to its previous width on hover or a hotkey press.

#### Scenario: Collapse to compact state

- GIVEN the app is in companion mode with a width of 400px
- WHEN the user triggers collapse (via button or hotkey)
- THEN the window shrinks to approximately 40px wide at the right edge, showing only a thin indicator bar

#### Scenario: Expand on hover

- GIVEN the window is in compact collapsed state (~40px)
- WHEN the user moves the mouse cursor over the thin bar
- THEN the window expands back to its previous width (400px)

#### Scenario: Expand on hotkey

- GIVEN the window is in compact collapsed state
- WHEN the user presses the expand hotkey
- THEN the window expands back to its previous width

### Requirement: Mode and Position Persistence

The system MUST remember the last window mode (companion/floating), position, and width across application restarts.

#### Scenario: Restore companion mode after restart

- GIVEN the user was in companion mode at 450px width when they last closed the app
- WHEN the app restarts
- THEN the window opens in companion mode at 450px width, anchored to the right edge

#### Scenario: Restore floating mode after restart

- GIVEN the user was in floating mode when they last closed the app
- WHEN the app restarts
- THEN the window opens in floating mode (420x700 centered)
