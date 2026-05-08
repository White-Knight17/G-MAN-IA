# Proposal: G-MAN v1.0 — Native Desktop GUI

## Intent

v0.9's TUI blocks non-technical users. v1.0 delivers a desktop companion — sidebar chat, system tray, notifications, onboarding wizard — with the Go core unchanged.

## Scope

### In Scope
- Tauri v2 + Svelte 5 GUI (was Bubbletea TUI)
- Go core as JSON-RPC sidecar (stdin/stdout IPC)
- Chat sidebar: streaming, typing indicators, history, permission dialogs
- System tray + desktop notifications + onboarding wizard
- Inline file preview (syntax highlighting, diff)
- `.AppImage`, `.deb`, `.rpm` (Linux, Arch-first)
- 5/5 E2E tool prompts via GUI (parity with v0.9)

### Out of Scope
Windows/macOS (v1.1), global hotkey, auto-updater, auto-start, knowledge-base UI, tool progress streaming, i18n.

## Capabilities

### New Capabilities
- `chat-ui`: Streaming chat, typing, tool status, permission dialogs
- `desktop-shell`: Tauri v2 window, tray, notifications, sidecar lifecycle, packaging
- `jsonrpc-transport`: JSON-RPC 2.0 over stdin/stdout, NDJSON, Rust relay
- `onboarding-wizard`: Ollama detection, model pull, workspace config, connectivity check
- `file-preview`: Syntax-highlighted code, diff view, copy button

### Modified Capabilities
None (domain/application/infrastructure unchanged).

## Approach

**Monorepo**: `/core/` (Go + `cmd/gman-server` + `internal/transport/`), `/app/` (Tauri shell, ~150 LOC), `/ui/` (Svelte 5 + Vite + Tailwind).

**Go** (additive): `StreamRun() <-chan StreamEvent` on orchestrator. `json` tags on Session types. New `cmd/gman-server` reads/writes NDJSON. `cmd/gman` preserved as TUI fallback.

**IPC**: RPC with `id` via Rust → Go stdin. NDJSON notifications via Go stdout → Rust → Svelte events.

| Area | Impact |
|------|--------|
| `core/cmd/gman-server/`, `core/internal/transport/` | New |
| `core/internal/application/`, `core/internal/domain/` | Add StreamRun, json tags |
| `core/cmd/gman/` | Preserved |
| `app/src-tauri/`, `ui/`, `Makefile` | New |

## Risks

| Risk | L | Mitigation |
|------|---|------------|
| stdout buffering | M | Encoder flush; NDJSON delimited |
| Wayland tray | M | Tauri abstraction; window-only fallback |
| Sidecar crash | M | Exit-code monitor; auto-restart |
| Ollama not installed | H | Onboarding detects + guides install |
| 85 test regressions | L | StreamRun additive; Run preserved |

## Rollback Plan

Preserve `cmd/gman/main.go` as TUI fallback. All changes additive. Delete `/app/`, `/ui/`, `cmd/gman-server/` to revert to v0.9.

## Dependencies

Node.js ≥20, Rust, Go 1.26, Ollama, pnpm

## Success Criteria

- [ ] 5/5 E2E tool prompts via GUI
- [ ] Onboarding <2 min, cold start <3s (8GB, SSD)
- [ ] Bundle <20MB compressed
- [ ] 0 Go test regressions (85 tests)
- [ ] First token <500ms of send
- [ ] System tray: KDE, Gnome-X11, Hyprland/waybar
