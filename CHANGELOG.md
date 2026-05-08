# Changelog

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
