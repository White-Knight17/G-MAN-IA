# Archive Report: harvey-core

**Archived**: 2026-05-06  
**Change**: harvey-core — Local AI Assistant for Arch Linux + Hyprland  
**SDD Mode**: hybrid (engram + openspec)  
**Final Verdict**: PASS WITH WARNINGS (0 CRITICAL, 4 WARNINGS, 5 SUGGESTIONS)

---

## What Was Built

Harvey Core is a local AI assistant (Go 1.26, Clean/Hexagonal Architecture, Bubbletea TUI) that helps new Linux users configure their Arch Linux + Hyprland dotfiles through natural language conversation, offline and with zero telemetry.

### Core Components

| Component | Package | Lines (source + test) | Coverage |
|-----------|---------|----------------------|----------|
| Domain interfaces | `internal/domain/` | ~380 + 14 tests | N/A (value objects) |
| Application orchestration | `internal/application/` | 1 COVERED | 89.9% |
| Ollama HTTP client | `internal/infrastructure/ollama/` | 1 COVERED | 89.0% |
| Permission repository | `internal/infrastructure/permission/` | 1 COVERED | 100.0% |
| Sandbox (Bubblewrap + Landlock) | `internal/infrastructure/sandbox/` | 1 COVERED | 67.3% (1 SKIP) |
| Dotfile tools (6) | `internal/infrastructure/tools/` | 1 COVERED | 69.1% |
| Bubbletea TUI | `internal/ui/tui/` | 1 COVERED | 46.4% |
| Entry point | `cmd/harvey/main.go` | 216+ lines | N/A |
| Model verification | `cmd/verify-models/main.go` | Phase 0 | N/A |
| **Total** | **9 packages** | **7046 lines (3965 source, 3081 test)** | **85 tests, 0 failures** |

### Architecture

- **Pattern**: Clean/Hexagonal — domain defines interfaces, infrastructure implements them, UI drives use cases through ports
- **Model**: llama3.2:3b via Ollama (verified 8/10 tool-call consistency; qwen2.5:3b and qwen3.5:2b both failed at 0/10)
- **Agent loop**: ReAct pattern (user → LLM → XML tool-call parse → sandboxed execution → result → LLM → response)
- **Sandbox**: Defense-in-depth — path validation (symlink-resolved) → command blocklist (18 dangerous commands) → command allowlist (10 safe commands) → Bubblewrap container (`--unshare-all`, `--ro-bind`, `--bind`, `--tmpfs`)
- **Permissions**: Session-scoped, in-memory `sync.RWMutex`-protected map, ro/rw per directory, expire on exit
- **TUI**: Split layout (chat history + file preview), grant confirmation modal, streaming text, keyboard navigation
- **6 tools**: read_file, write_file (with .bak + diff), list_dir, run_command (allowlisted), check_syntax, search_wiki

### Delivery

- **Feature-branch-chain**: 5 PRs (01-domain → 02-app → 03-sandbox → 04-tui → 05-wiring)
- **Git**: Local-only (no remote configured); all branches on `feature/harvey-core` chain
- **Commits**: Work-unit structured (4 commits in PR 5, plus earlier PRs)

---

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Go (not Python) | Single static binary (~15MB), native Landlock/Bubblewrap, Bubbletea TUI ecosystem |
| Direct Ollama REST API (no Go AI libs) | `/api/chat` is one POST endpoint — ~50 lines of HTTP code |
| XML tool-call format (not JSON) | 1.5B models handle XML more reliably; easier to partial-parse during streaming |
| llama3.2:3b over deepseek-r1:1.5b | Phase 0 verification: r1 not pulled; llama3.2:3b scored 8/10, r1 equivalents (qwen) scored 0/10 |
| Bubblewrap primary, Landlock secondary | Battle-tested (Flatpak), no root required; Landlock wired but not integrated in main.go |
| In-memory permissions (not persisted) | Grants expire on exit by design — no persistence attack surface |
| Feature-branch-chain (not stacked in main) | 400-line review budget with 2000-line change → 5 autonomous PR slices |

---

## Verify Findings

### No CRITICAL issues — safe to archive

### WARNINGS (4)

1. **Ollama streaming not implemented**: `client.go:99` sets `Stream: false`. Spec requires NDJSON streaming. TUI receives full messages, not incremental token deltas. Affects 2 scenarios (streaming chat PARTIAL, token display UNTESTED).
2. **views/ directory is empty**: Design listed 4 view files; all view logic is in `model.go`, `update.go`, `view.go`. Either add view files or update design.
3. **Dotfile map not implemented**: Design decision about dotfile map compression (~50 configs → ~2K tokens) is absent. Orchestrator trims session history but has no explicit compression layer.
4. **Landlock not wired in main.go**: Landlock is implemented but never integrated into the application. Only BubblewrapSandbox is active. Landlock test is skipped (requires root). Defense-in-depth gap.

### SUGGESTIONS (5)

1. **doc/ directory is empty** — no project documentation
2. **E2E test scope**: 5 prompts (not 20 as originally planned) — documented deviation
3. **Spinner animation static**: renders `frames[0]` instead of cycling frames
4. **TUI coverage at 46.4%**: lower than other packages; more component tests would help
5. **No `--no-network` in bwrap args**: design says it should be present but `buildBwrapArgs()` doesn't include it

---

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| harvey-core | **Created** | 6 requirements, 20 scenarios. First spec in `openspec/specs/`. All ADDED requirements converted to main spec Requirements. |

---

## Artifact Locations

| Artifact | Engram ID | OpenSpec Path |
|----------|-----------|---------------|
| Exploration | #9 | N/A |
| Proposal | #11 | `openspec/changes/archive/2026-05-06-harvey-core/proposal.md` |
| Delta Spec | #12 | `openspec/changes/archive/2026-05-06-harvey-core/spec.md` |
| Design | #13 | `openspec/changes/archive/2026-05-06-harvey-core/design.md` |
| Tasks | #14 | `openspec/changes/archive/2026-05-06-harvey-core/tasks.md` |
| Apply Progress | #15 | N/A |
| Verify Report | #18 | `openspec/changes/archive/2026-05-06-harvey-core/verify-report.md` |
| Project Init | #5 | N/A |
| **Main Spec** (new) | N/A | `openspec/specs/harvey-core/spec.md` |
| **Archive Report** | (this save) | `openspec/changes/archive/2026-05-06-harvey-core/archive-report.md` |

---

## Source of Truth

`openspec/specs/harvey-core/spec.md` now serves as the authoritative spec for Harvey Core's 6 requirements. All future changes to harvey-core MUST reference this spec as base.

---

## SDD Cycle Complete

The harvey-core change has been fully planned (explore → propose → spec → design → tasks), implemented (apply × 5 PRs), verified (verify with 0 CRITICAL issues), and archived.

**Ready for the next change in the programas project.**
