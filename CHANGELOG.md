# Changelog

## [1.0.0](https://github.com/White-Knight17/G-MAN-IA/compare/v0.10.1...v1.0.0) (2026-05-08)


### ⚠ BREAKING CHANGES

* Release Please configuration enforces v1.0.0 baseline. Adds bump-minor-pre-major and release-as: 1.0.0 to prevent 0.x drift.

### Features

* add CHANGELOG, ROADMAP and session context for AI agents ([4349b51](https://github.com/White-Knight17/G-MAN-IA/commit/4349b5176378dd60f28ce30cb7316f211842a3ca))
* add domain interfaces (Agent, Tool, Sandbox, Permission, Session) ([e4ccaa0](https://github.com/White-Knight17/G-MAN-IA/commit/e4ccaa02495e4655167c8a661557909fc35d9098))
* add model tool-use verification (3-model test, llama3.2:3b passes) ([8496885](https://github.com/White-Knight17/G-MAN-IA/commit/84968851a37a89878840a57c69c99bd86745b9da))
* **app:** add ChatOrchestrator with ReAct agent loop ([3de7750](https://github.com/White-Knight17/G-MAN-IA/commit/3de775073ae76c2cfb680c1cf2c54f477b32de0c))
* **app:** add GrantManager for session-scoped permission grants ([56c5835](https://github.com/White-Knight17/G-MAN-IA/commit/56c583576bf294bc1155e284e75962927cc9786f))
* **app:** add ToolExecutor with case-insensitive XML parsing ([b1356bf](https://github.com/White-Knight17/G-MAN-IA/commit/b1356bf7d147572ce6dfb6be6b4ba8830e43185e))
* **build:** add build pipeline, E2E tests, and v1.0 README ([077416b](https://github.com/White-Knight17/G-MAN-IA/commit/077416b08d267baafdf9b0e151015bf3b258597c))
* **build:** add build pipeline, E2E tests, and v1.0 README ([965d6e5](https://github.com/White-Knight17/G-MAN-IA/commit/965d6e50a54ad0f2f6753675aa4280c861852450))
* **domain:** initialize Go module for harvey ([4b54570](https://github.com/White-Knight17/G-MAN-IA/commit/4b54570b722f775bb8014233fef3cad6110839a4))
* force Release Please to v1.0.0 with config ([718aed7](https://github.com/White-Knight17/G-MAN-IA/commit/718aed74cf9e73aa5022cdd8498fce57f51efc4d))
* **infra:** add in-memory permission repository ([5a40632](https://github.com/White-Knight17/G-MAN-IA/commit/5a40632695f7d1c447bf9ceb7f3050e0deaa2a1b))
* **infra:** add Ollama HTTP client implementing Agent interface ([9111cd1](https://github.com/White-Knight17/G-MAN-IA/commit/9111cd123618b568cd927664eb472322c59ca288))
* **sandbox:** add Bubblewrap sandbox implementation ([946e70a](https://github.com/White-Knight17/G-MAN-IA/commit/946e70a8adef38ff633b8c0ab0a845efc8cb543c))
* **sidecar:** add JSON-RPC transport, streaming, and gman-server entry point ([71ed570](https://github.com/White-Knight17/G-MAN-IA/commit/71ed5701a6f14d47d72dfe66d0c95812bd2e2639))
* switch to lightweight text commands and add documentation ([d7eed36](https://github.com/White-Knight17/G-MAN-IA/commit/d7eed36cc295259bab436fd98365eb9fe30f5527))
* **tauri:** add Tauri v2 desktop shell with sidecar relay and system tray ([1a8e846](https://github.com/White-Knight17/G-MAN-IA/commit/1a8e846e1542963dc296aead23d9a842f56f689a))
* **tools:** add command tool with allowlist/blocklist ([eebc165](https://github.com/White-Knight17/G-MAN-IA/commit/eebc16505130d5e5f6ed09c0a627f071e660ce0c))
* **tools:** add filesystem tools (read_file, write_file, list_dir) ([cf01c61](https://github.com/White-Knight17/G-MAN-IA/commit/cf01c618d2cccaf7d1dccfc91f968c818c77b610))
* **tools:** add syntax check and wiki search tools ([ab54eb7](https://github.com/White-Knight17/G-MAN-IA/commit/ab54eb7993d5da68de96f6e7aab8a4769d146702))
* **transport:** add JSON-RPC 2.0 server over stdin/stdout ([c128065](https://github.com/White-Knight17/G-MAN-IA/commit/c12806527c05228a728ae6f5773c27827bd58aeb))
* **tui:** add Bubbletea model with chat loop, views, and async orchestrator ([0726fe6](https://github.com/White-Knight17/G-MAN-IA/commit/0726fe6d22bb4272c535bc94ba7f761845de91f0))
* **tui:** add Lipgloss styles and keyboard bindings ([7feefdd](https://github.com/White-Knight17/G-MAN-IA/commit/7feefdd4578c940e2d3d898893063293255b0c6e))
* **verify:** add model tool-use verification for deepseek-r1:1.5b ([c8101c2](https://github.com/White-Knight17/G-MAN-IA/commit/c8101c2adc25be976e269e83bf9883a117f94ca0))
* wire dependency injection in main.go with graceful shutdown ([18b9c89](https://github.com/White-Knight17/G-MAN-IA/commit/18b9c892a546867cbf536eeef04cf66384e316aa))


### Bug Fixes

* **build:** add pnpm install to CI build job and Makefile ([5f2cfa6](https://github.com/White-Knight17/G-MAN-IA/commit/5f2cfa69e3c31c433aabcf26f893b90d7109db22))
* **build:** make dev depends on build-core ([f2688b1](https://github.com/White-Knight17/G-MAN-IA/commit/f2688b11573b852767cf9ba79dc339a16e358e23))
* **build:** remove target triple from externalBin name ([81389ef](https://github.com/White-Knight17/G-MAN-IA/commit/81389ef04e64b0eed27e69240d4e689a52e28c11))
* **ci:** add Tauri system deps and pnpm version to CI workflow ([55a1823](https://github.com/White-Knight17/G-MAN-IA/commit/55a1823c8ab682ed1c3dcf17146946fe1881a733))
* **ci:** add Tauri system deps and pnpm version to CI workflow ([4b53523](https://github.com/White-Knight17/G-MAN-IA/commit/4b535232f7767f6445898e55219aebb2a98e44d0))
* **ci:** add Tauri system deps and pnpm version to CI workflow ([b47b185](https://github.com/White-Knight17/G-MAN-IA/commit/b47b185cf6f6f0cb3070aa9bbec0feeb873c8c1e))
* **sandbox:** correct Landlock syscalls and bwrap compatibility ([69cd7d1](https://github.com/White-Knight17/G-MAN-IA/commit/69cd7d1dd2d1f4112b5af957f20a894489ddd9e9))
* **tauri:** correct sidecar binary path in dev mode ([ccb5d8e](https://github.com/White-Knight17/G-MAN-IA/commit/ccb5d8e8ea5e4e9a1cbf13c72209719ec65d0efd))
* **tauri:** multi-path sidecar discovery + CI Go build ([71985fe](https://github.com/White-Knight17/G-MAN-IA/commit/71985fe14768d40e91dfa480c1e22127088412f8))

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
