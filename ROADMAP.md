# Roadmap

## v2.1.0 — Companion Mode 🎉 (DONE)

### Features
- [x] **Companion Mode**: always-on-top sidebar, floating/compact modes, Ctrl+Shift+G global hotkey
- [x] **Slash Commands**: `/help`, `/clear`, `/model`, `/models <name>`, `/api <provider> <key>`
- [x] **Material UI Refresh**: elevation tokens, 8px spacing grid, typography hierarchy, button transitions
- [x] **Multi-Provider API**: OpenAI, DeepSeek, Groq, and any OpenAI-compatible API
- [x] **Model Auto-Detect**: auto-set correct model when switching providers
- [x] **Config Migration**: localStorage → `~/.config/gman/config.json`
- [x] **Settings Button**: re-run onboarding wizard from the titlebar
- [x] **Auto-Detect Ollama**: shows available models on startup

### Technical
- [x] **FilePreview component**: show file content/diff in the right panel when G-MAN reads/writes files

---

## v2.2.0 — Polish & Completeness
> (formerly listed as v1.1.0 — renumbered due to release-please)
>
> **Target**: Complete the 6 deferred CRITICAL items from v1.0.0 verify phase.

### Features
- [ ] **Onboarding Ollama validation**: actually test Ollama connectivity and model availability in the wizard (not just UI mocks)
- [ ] **Streaming UX**: true token-by-token rendering in chat (currently full-response)

### Technical
- [ ] **golangci-lint**: add to CI for static analysis
- [ ] **Full Tauri GUI E2E tests**: Playwright tests with actual Tauri window (currently sidecar-only)
- [ ] **Tauri auto-updater**: verify end-to-end with GitHub Releases
- [ ] **Unified config model**: single `Config` struct shared between Go and frontend
- [ ] **Wayland tray compatibility**: test and fix system tray on Hyprland/Wayland

---

## v2.3.0 — Cloud & Knowledge
> (formerly listed as v1.2.0)
- [ ] **Knowledge base browser**: UI for managing `.md` files in `~/.config/gman/knowledge/`
- [ ] **Conversation history**: persist and search past conversations
- [ ] **Multi-model selector**: switch between local/remote models from settings
- [ ] **Provider management UI**: GUI for adding/removing API providers

---

## v3.0.0 — Ecosystem
> (formerly listed as v2.0.0)
- [ ] **Plugin system**: community tools and extensions
- [ ] **Windows support**: Tauri cross-compilation
- [ ] **Internationalization**: multi-language UI
- [ ] **Voice input/output**: speech-to-text and TTS
