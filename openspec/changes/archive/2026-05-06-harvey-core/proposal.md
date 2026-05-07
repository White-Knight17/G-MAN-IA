# Proposal: Harvey Core — Local AI Assistant for Arch Linux + Hyprland

## Intent

New Linux users on Arch + Hyprland struggle with config editing via terminal. Harvey is an offline local AI assistant that reads, explains, and safely modifies dotfiles (~/.config only). Single binary, zero telemetry, zero internet.

## Scope

### In Scope
- ReAct agent loop: user prompt → Ollama (deepseek-r1:1.5b) → tool-call parsing (XML) → sandboxed execution → response
- 6 tools: read_file, write_file (with .bak), list_dir, run_command (allowlisted), check_syntax, search_docs
- Defense-in-depth sandbox: path validation → Landlock in-process → Bubblewrap subprocess for commands
- Session-scoped permission grants (ro/rw per directory, expire on exit, grant UI indicator)
- Bubbletea TUI: split layout (chat history + file preview/diff), streaming token display, input bar
- Context window management: 8K token cap, dotfile map for context compression
- Go 1.26 single static binary (~15MB), zero runtime deps

### Out of Scope
- Multi-model support, Web/GUI, system-file editing (/etc), auto package install, persistent history, plugins

## Capabilities

### New Capabilities
- `agent-loop`: ReAct agent orchestrating user→LLM→tools→LLM cycle with structured XML tool-call parsing
- `ollama-client`: HTTP client for `/api/chat` with NDJSON streaming, no external Go libraries
- `sandbox`: Landlock + Bubblewrap + path validation defense-in-depth; kernel-enforced path restrictions
- `permission-model`: Session-scoped directory grants, grant UI, expire-on-exit, explicit before write
- `tui`: Bubbletea terminal UI with Lip Gloss styling, chat/file-preview split, mouse support
- `dotfile-tools`: 6 sandboxed tools — read, write (with .bak + diff), list, run (allowlisted), check config syntax, search wikis

### Modified Capabilities
- None (greenfield)

## Approach

**Stack**: Go 1.26 + Bubbletea + direct Ollama REST API. No external Go AI libraries — the API is a single POST endpoint.

**Architecture**: Clean/Hexagonal. Domain defines `Agent`, `Tool`, `Sandbox` interfaces. Infrastructure adapts Ollama HTTP, Landlock syscalls, Bubblewrap exec. UI (Bubbletea) drives application use cases through ports.

**Monorepo**: `cmd/harvey/` → `internal/{domain,application,infrastructure,ui}/`

**Phase 0 (pre-code)**: Pull deepseek-r1:1.5b, run 20-prompt capability test for tool-call consistency. If <80%, fallback to llama3.2:3b.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/harvey/` | New | Entry point, CLI flags, signal handling |
| `internal/domain/` | New | Agent loop, Tool interface, Sandbox port, Permission model |
| `internal/application/` | New | Chat orchestration, tool execution, grant use cases |
| `internal/infrastructure/ollama/` | New | HTTP client + NDJSON stream parser |
| `internal/infrastructure/sandbox/` | New | Landlock bindings + Bubblewrap executor |
| `internal/infrastructure/filesystem/` | New | Path-validated file operations |
| `internal/ui/tui/` | New | Bubbletea model, views, components |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| 1.5B model inconsistent tool-call formatting | Med | Pre-test 20+ prompts; fallback to 3B model |
| R1 CoT reasoning adds unacceptable latency | Med | Disable thinking mode; test llama3.2:1b as fast alt |
| Corrupted config leaves user without GUI | Low | Always `.bak`; show diff before apply; `undo` command |
| Landlock Go syscall bugs | Low | Bubblewrap-first; add Landlock as hardening after bubblewrap tested |

## Rollback Plan

- **Binary**: delete `harvey` — no system files modified
- **Configs**: `.bak` files created on every write; `cp .bak → original` restores
- **Sandbox**: Landlock process-scoped (dies with process); Bubblewrap leaves no state
- **Model**: `ollama rm deepseek-r1:1.5b` if pulled during verification

## Dependencies

- Ollama v0.23.1+ at `localhost:11434` (installed)
- deepseek-r1:1.5b (not pulled — first action)
- Bubblewrap at `/usr/bin/bwrap` (available)
- Linux 5.13+ with Landlock, user namespaces, seccomp (on 7.0.3)
- Go 1.26+ (installed)

## Success Criteria

- [ ] Agent completes full ReAct loop: ask → tool-call → execute → respond
- [ ] All 6 tools function under sandbox; path traversal blocked at all 3 layers
- [ ] Write ops create `.bak`, require explicit grant, show diff before apply
- [ ] TUI renders split layout with streaming tokens, grant indicator, input bar
- [ ] Model tool-call consistency ≥80% on 20-prompt Hyprland config test suite
- [ ] Static binary: `go build -o harvey cmd/harvey/main.go` — runs standalone
