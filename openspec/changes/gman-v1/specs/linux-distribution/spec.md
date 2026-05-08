# linux-distribution Specification

## Purpose

Linux packaging via Tauri bundler producing .AppImage, .deb, and .rpm with embedded Go sidecar. Auto-update via Tauri updater checking GitHub Releases.

## Requirements

### Requirement: Linux Distribution

The system MUST produce distributable packages via Tauri bundler: `.AppImage`, `.deb`, and `.rpm`. The Go sidecar binary SHALL be embedded as a Tauri sidecar with automatic architecture detection (amd64/arm64). The compressed bundle size MUST be under 30MB. The Tauri updater SHALL check GitHub Releases for new versions and notify the user. A manual update check SHALL be available from settings.

#### Scenario: AppImage launch

- GIVEN the `.AppImage` file is downloaded and made executable (`chmod +x`)
- WHEN the user runs `./G-MAN_1.0.0_amd64.AppImage`
- THEN the G-MAN window opens, the system tray icon appears, and the Go sidecar starts within 3 seconds

#### Scenario: Debian package install

- GIVEN the `.deb` package is downloaded
- WHEN the user runs `sudo dpkg -i gman_1.0.0_amd64.deb`
- THEN a `.desktop` entry is created at `/usr/share/applications/`, the binary is at `/usr/bin/gman`, and the app launches from the system app launcher

#### Scenario: Auto-update notification

- GIVEN G-MAN v1.0.0 is running
- WHEN the Tauri updater detects a newer release on GitHub Releases (e.g., v1.0.1)
- THEN a notification appears: "Update available: v1.0.1" with a "Download and Install" button; clicking it downloads the update and applies it

#### Scenario: Manual update check

- GIVEN the app is running
- WHEN the user opens Settings and clicks "Check for updates"
- THEN the updater queries the GitHub Releases API; if up-to-date it shows "You're on the latest version"; if an update is available it shows the notification from the auto-update scenario
