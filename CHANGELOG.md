# Changelog

## v2.1.0 (2026-05-13) — Companion Mode

### 🎉 G-MAN becomes a desktop companion

Three major pillars: always-visible sidebar modes, slash commands, and multi-provider API support (OpenAI, DeepSeek, Groq).

### Added
- **Companion Mode**: always-on-top sidebar (380px), floating (420×700), compact (48px bar) window modes
- **Global hotkey**: `Ctrl+Shift+G` to show/hide G-MAN from anywhere
- **Slash commands**: `/help`, `/clear`, `/model`, `/model <name>`, `/models <name>`, `/api <provider> <key>`
- **Command palette**: autocomplete popup when typing `/` in chat input
- **Multi-provider API**: OpenAI-compatible client supporting OpenAI, DeepSeek, Groq, and custom endpoints
- **Auto-detect models**: correct model set automatically when switching providers (deepseek-v4-pro, gpt-4o, etc.)
- **Ollama auto-detect**: shows available local models on startup
- **Settings button**: ⚙ icon in titlebar to re-run the onboarding wizard
- **Material UI refresh**: CSS elevation tokens (4 levels), 8px spacing grid, typography hierarchy, button transitions
- **Config persistence**: `~/.config/gman/config.json` with migration from localStorage
- **Config migration**: automatic localStorage → config.json on first v2.1.0 launch
- **Provider list**: `provider.list` JSON-RPC endpoint

### Technical
- **452 tests** (Go 12 packages + Svelte 108 tests) — 0 failures
- **Go**: new `openai.Client` implementing `domain.Agent` with SSE streaming
- **Rust**: async Tauri commands (`relay_request`, `stream_chat`) to prevent UI freeze
- **Svelte 5**: fixed SSR lifecycle issues with `conditions: ["browser"]` in Vite config
- **Build**: sidecar binary path fix in Makefile

---

## v2.0.0 (2026-05-08) — Desktop Revolution

### 💥 BREAKING: TUI → Desktop GUI

G-MAN leaves the terminal. The Bubbletea TUI is replaced by a native desktop app with Tauri v2 + Svelte 5. The Go core continues as a JSON-RPC 2.0 sidecar over stdin/stdout.

## v1.0.0 (2026-05-08) — Desktop Assistant

### 🎉 G-MAN goes GUI

Replaced the Bubbletea TUI with a native desktop app built with Tauri v2 + Svelte 5. The Go core remains unchanged — it now runs as a JSON-RPC 2.0 sidecar communicating over stdin/stdout (zero network exposure).

### Added
- **Tauri v2 desktop shell**: frameless 420×700 window, system tray (Show/Hide/Quit), left-click toggle
- **Svelte 5 frontend**: chat bubbles with streaming typing indicator, auto-scroll, welcome message
- **Onboarding wizard**: 3-step first-run setup (AI backend, allowed directories, theme)
- **Permission dialogs**: modal overlay with Allow/Deny + 30s auto-deny timeout
- **JSON-RPC 2.0 transport**: NDJSON over stdin/stdout, no open ports
- **Streaming API**: `StreamRun()` channel-based method on Agent interface + Ollama NDJSON streaming
- **Build pipeline**: Makefile with 10 targets, Tauri bundler (.deb, .AppImage, .rpm)
- **CI/CD matrix**: 4 jobs (Go lint/vet, Go test, Rust test, Svelte test, Build with artifact upload)
- **Release Please**: automated versioning with major/minor tags

### Changed
- **System prompt**: lightweight text commands (READ/WRITE/LIST/RUN/CHECK/SEARCH) replace XML — 85% fewer tokens, 100% E2E accuracy with llama3.2:3b
- **Module**: `github.com/gentleman/gman` (renamed from harvey)
- **Config path**: `~/.config/gman/` (was `~/.config/harvey/`)
- **Architecture**: monorepo `/core/` (Go) + `/app/` (Tauri + Svelte)

### Removed
- Bubbletea TUI (preserved as fallback in `core/cmd/gman/main.go`)

### Technical
- **342 tests** (Go 265 + Svelte 57 + Rust 12 + E2E 9)
- **0 failures**
- **Go vet**: clean
- Clean/Hexagonal Architecture preserved — domain layer unchanged from v0.9

---

## v0.9.0 (2026-05-06) — TUI Alpha

### Initial Release
- ReAct agent loop with llama3.2:3b via Ollama
- 6 sandboxed dotfile tools (read_file, write_file with .bak, list_dir, run_command with allowlist, check_syntax, search_wiki)
- Bubblewrap sandbox with defense-in-depth (path validation + blocklist)
- Session-scoped permission grants
- Bubbletea TUI with chat/file-preview split
- 85 tests, 0 failures
- CI/CD with GitHub Actions + Release Please
