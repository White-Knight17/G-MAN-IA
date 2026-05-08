# Roadmap

## v1.1.0 — Polish & Completeness

**Target**: Complete the 6 deferred CRITICAL items from v1.0.0 verify phase.

### Features
- [ ] **FilePreview component**: show file content/diff in the right panel when G-MAN reads/writes files
- [ ] **Onboarding Ollama validation**: actually test Ollama connectivity and model availability in the wizard (not just UI mocks)
- [ ] **Ctrl+Shift+G keyboard toggle**: global hotkey to show/hide G-MAN sidebar
- [ ] **Settings panel**: re-run onboarding wizard, manual update check, theme switcher
- [ ] **Config file persistence**: move from localStorage to `~/.config/gman/config.json` via JSON-RPC
- [ ] **Streaming UX**: true token-by-token rendering in chat (currently simulated)

### Technical
- [ ] **golangci-lint**: add to CI for static analysis
- [ ] **Full Tauri GUI E2E tests**: Playwright tests with actual Tauri window (currently sidecar-only)
- [ ] **Tauri auto-updater**: verify end-to-end with GitHub Releases
- [ ] **Unified config model**: single `Config` struct shared between Go and frontend
- [ ] **Wayland tray compatibility**: test and fix system tray on Hyprland/Wayland

---

## v1.2.0 — Cloud & Knowledge

- [ ] **Remote API support**: OpenAI, Anthropic, Groq API keys as alternative to local Ollama
- [ ] **Knowledge base browser**: UI for managing `.md` files in `~/.config/gman/knowledge/`
- [ ] **Conversation history**: persist and search past conversations
- [ ] **Multi-model selector**: switch between local/remote models from settings

---

## v2.0.0 — Ecosystem

- [ ] **Plugin system**: community tools and extensions
- [ ] **Windows support**: Tauri cross-compilation
- [ ] **Internationalization**: multi-language UI
- [ ] **Voice input/output**: speech-to-text and TTS
