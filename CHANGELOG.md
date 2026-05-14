# Changelog

## [2.1.0](https://github.com/White-Knight17/G-MAN-IA/compare/v2.0.0...v2.1.0) (2026-05-14)


### Features

* accept any API provider in /api command ([f103c40](https://github.com/White-Knight17/G-MAN-IA/commit/f103c409161a441880c34b96be1e837d6c97b2f1))
* add OpenAI-compatible API backend with multi-provider support ([b3e3f2b](https://github.com/White-Knight17/G-MAN-IA/commit/b3e3f2b42e0a76deec507da2325becad47053fb6))
* **companion-mode:** add slash commands and JSON-RPC handlers ([4bee3b2](https://github.com/White-Knight17/G-MAN-IA/commit/4bee3b250366ec44f79ec58f13ed563027c2d55c))
* **companion-mode:** Material UI, config migration, E2E tests ([77be6aa](https://github.com/White-Knight17/G-MAN-IA/commit/77be6aa375b1d0926b498e29f61a6762712b539e))
* v2.1.0 Companion Mode — sidebar, slash commands, multi-provider APIs ([e6f95bb](https://github.com/White-Knight17/G-MAN-IA/commit/e6f95bb0293b6a2cebfaecbf8a594c6d55dd8138))


### Bug Fixes

* add missing oncommand prop to ChatView ([32349db](https://github.com/White-Knight17/G-MAN-IA/commit/32349db3a38584222bf0128f6f550cdccbc0c36e))
* add missing Tauri commands, improve UX, auto-detect Ollama ([0d75f48](https://github.com/White-Knight17/G-MAN-IA/commit/0d75f486a1fa98ce0911c16ad73c8a342285815b))
* auto-set correct model when switching API provider ([785421c](https://github.com/White-Knight17/G-MAN-IA/commit/785421c812b8394bec787c3f4196f2a3d8fefe17))
* create binaries directory in build-core target ([c0a500f](https://github.com/White-Knight17/G-MAN-IA/commit/c0a500fe80b0a25a57a5b970628b7d6227d9b442))
* force browser conditions in Vite to prevent SSR mount error ([13740ae](https://github.com/White-Knight17/G-MAN-IA/commit/13740ae21f695e57583c9f412288e3a7e0212af4))
* make relay_request and stream_chat async to prevent UI freeze ([303994c](https://github.com/White-Knight17/G-MAN-IA/commit/303994cc4dfb32a4d61e9e5bb8b2dcd6ec90d77b))
* rename Tauri commands to match frontend invoke names ([ff51d6a](https://github.com/White-Knight17/G-MAN-IA/commit/ff51d6ace3baa6e6fd56eaecfefb40f8e0be158a))
* resolve Svelte 5 lifecycle_function_unavailable error ([ee97fc5](https://github.com/White-Knight17/G-MAN-IA/commit/ee97fc5a24f61fc253cbe4c40b268d4659fe638a))
* rewrite Tauri relay commands to properly handle notifications ([94b0bb9](https://github.com/White-Knight17/G-MAN-IA/commit/94b0bb94cf1a21f7978f14b429521f0d73d054e5))
* use dynamic imports to prevent SSR mount error ([b6ae964](https://github.com/White-Knight17/G-MAN-IA/commit/b6ae9642824a4db2eb1c1ee24507b55f76bb1b71))

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
