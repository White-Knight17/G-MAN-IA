# G-MAN Session Context

For AI agents: read this first when starting a new session on this project.

## Project Identity

- Name: G-MAN (Gentleman AI Manager)
- Repo: github.com/White-Knight17/G-MAN-IA
- Version: v1.0.0
- Module: github.com/gentleman/gman
- Stack: Go 1.26 + Rust 1.95 + Svelte 5 + TypeScript

## What G-MAN Is

A local, private desktop AI assistant for Linux (Arch/Hyprland tested).
Helps newcomers configure dotfiles, run safe commands, and learn Linux.

## Architecture (Monorepo)

```
core/                    Go (Clean/Hexagonal)
  cmd/gman/              TUI fallback (Bubbletea)
  cmd/gman-server/       JSON-RPC sidecar (v1.0 main entry)
  internal/domain/       Agent, Tool, Sandbox, Permission, Session
  internal/application/  ChatOrchestrator, ToolExecutor, GrantManager
  internal/infrastructure/ Ollama, Bubblewrap, Landlock, 6 tools, permissions
  internal/transport/    JSON-RPC 2.0 NDJSON over stdin/stdout

app/                     Tauri v2 + Svelte 5
  src-tauri/             Rust shell: sidecar spawn, IPC relay, system tray
  src/                   Svelte 5: ChatView, OnboardingWizard, PermissionDialog
```

## Key Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| Go core as sidecar | JSON-RPC stdin/stdout | Zero ports, zero network, Go domain unchanged |
| UI framework | Svelte 5 | 2KB runtime, compiler-based reactivity |
| IPC format | Text commands (READ/WRITE/etc) | 85% fewer tokens than XML, 100% E2E with llama3.2:3b |
| Model | llama3.2:3b via Ollama | CPU-only viable, 80% tool accuracy |
| Sandbox | Bubblewrap + path validation | Defense-in-depth, kernel-level isolation |

## Testing

- Total: 342 tests (Go 265 + Svelte 57 + Rust 12 + E2E 9)
- Strict TDD enabled
- Commands: make test-all, make test-core, make test-ui

## Build

- make dev: build Go sidecar + launch Tauri dev server
- make build: full production build
- make bundle: .deb, .AppImage, .rpm

## CI/CD

- .github/workflows/ci.yml: lint, test (Go+Rust+Svelte), build
- .github/workflows/release-please.yml: auto releases with major/minor tags

## Known Issues (v1.0.0)

- FilePreview component not implemented (deferred to v1.1)
- Onboarding wizard does not validate Ollama connectivity
- No Ctrl+Shift+G keyboard toggle
- Settings panel missing
- Config in localStorage (not config.json)
- Wayland tray compatibility untested

## SDD Artifacts

- openspec/specs/ (6 domain specs)
- openspec/changes/archive/ (harvey-core + gman-v1)

## User Preferences

- Spanish communication (Rioplatense)
- Auto mode (run all SDD phases)
- Hybrid artifact store (engram + openspec)
- Auto-chain delivery with feature-branch-chain
