# Tasks: G-MAN v1.0 — Native Desktop GUI

## Review Workload Forecast

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

| Unit | Goal | Base | PR |
|------|------|------|-----|
| 1 | Go sidecar | feature/gman-v1 | #1 |
| 2 | Tauri shell | #1 branch | #2 |
| 3 | Svelte frontend | #2 branch | #3 |
| 4 | Build + E2E | #3 branch | #4 |

## Phase 0: Monorepo + Toolchain

- [x] 0.1 Move `cmd/`, `internal/`, `go.mod`, `go.sum` to `/core/`. `go test ./...` — 0 regressions.
- [x] 0.2 Install Rust, Node 22, pnpm. Scaffold `/app/` (Svelte 5 + TS + Vite + Tailwind + `@tauri-apps/api`).
- [x] 0.3 Skeleton dirs: `app/src/lib/{components,stores,lib,types}`, `routes/`. Create `/scripts/build.sh` + root `Makefile`.

## Phase 1: Go Sidecar (JSON-RPC + Streaming)

- [x] 1.1 **TEST** `jsonrpc_test.go`: request parsing, dispatch, errors, NDJSON framing, ready notification.
- [x] 1.2 **IMPL** `jsonrpc.go`: JSON-RPC 2.0 stdin/stdout, NDJSON framing, handler dispatch, flush.
- [x] 1.3 **TEST** `domain_test.go`: `StreamEvent` + `StreamRun()` channel contract.
- [x] 1.4 **IMPL** Add `StreamEvent` + `StreamRun(...) <-chan StreamEvent` to `agent.go`.
- [x] 1.5 **IMPL** Add `json:"..."` tags to `Session`, `ChatMessage`, `Grant`, `ToolResult`.
- [x] 1.6 **TEST** `ollama/client_test.go`: `StreamRun()` → token events + `done`.
- [x] 1.7 **IMPL** Add `StreamRun()` to `ollama/client.go`.
- [x] 1.8 **TEST** `orchestrator_test.go`: `HandleMessageStream()` → `<-chan StreamEvent`.
- [x] 1.9 **IMPL** Add `HandleMessageStream(...)` → `<-chan StreamEvent` to `orchestrator.go`.
- [x] 1.10 **IMPL** `cmd/gman-server/main.go`: reuse DI from `cmd/gman`, JSON-RPC loop.
- [x] 1.11 `go test ./...` from `/core/` — 0 regressions, all new tests pass.

## Phase 2: Tauri Shell (Rust)

- [x] 2.1 `tauri.conf.json`: frameless 420×700, CSP, `externalBin` → `gman-core-$TARGET_TRIPLE`.
- [x] 2.2 `Cargo.toml`: `tauri` v2, `tauri-plugin-shell`. Capabilities: shell, notification, window.
- [x] 2.3 `main.rs`: spawn sidecar, restart on crash, stdout→Tauri events, invoke→stdin relay.
- [x] 2.4 Tray: Show/Hide/Quit. Close→minimize. Icon in `icons/`.

## Phase 3: Svelte 5 Frontend

- [x] 3.1 `jsonrpc.ts`: typed RPC messages. `ipc.ts` + `events.ts`: typed invoke/listen wrappers.
- [x] 3.2 `chat.ts`, `permissions.ts`, `settings.ts`: messages `$state`, pending grants, theme/model.
- [x] 3.3 `ChatView.svelte`: bubbles + auto-scroll + typing dots. `ChatInput.svelte`: textarea + send.
- [x] 3.4 `OnboardingWizard.svelte`: 3-step, save config, skip if exists.
- [x] 3.5 `PermissionDialog.svelte`: modal Allow/Deny. `FilePreview.svelte`: highlight + copy.
- [x] 3.6 `+page.svelte`: chat + toggle (Ctrl+Shift+G). `App.svelte`: wizard vs chat, theme.
- [x] 3.7 `vite.config.ts`, `tailwind.config.ts`, `app.css` with G-MAN colors.

## Phase 4: Build + E2E

- [x] 4.1 `scripts/build.sh`: cross-compile (`amd64`/`arm64`) → `binaries/`. `Makefile`: wire build targets.
- [x] 4.2 Bundler: `.AppImage`, `.deb`, `.rpm`. Verify bundle <30MB, launch <3s.
- [x] 4.3 Playwright shell: launch, tray, close-to-tray, crash recovery.
- [x] 4.4 Playwright transport: request/response, streaming, error, not-ready.
- [x] 4.5 Playwright chat+wizard: send+stream, file preview, permission, toggle, first-launch, model-pull.
- [x] 4.6 Distribution: AppImage exec, deb install, update check. Update `/README.md`.
