# Tasks: Harvey Core — Local AI Assistant

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1500–2000 (22 files + tests) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | 5 PRs (feature-branch-chain) |
| Delivery strategy | auto-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Domain interfaces + model validation | PR 1 → feature/harvey-core | Base: feature/harvey-core. No deps. ~150 lines. |
| 2 | Core orchestration: application + LLM client + permissions | PR 2 → PR 1 branch | Base: PR 1. Orchestrator, ToolExecutor, GrantMgr, OllamaClient, InMemoryPermissionRepo. ~350 lines. |
| 3 | Sandboxing + 6 tools | PR 3 → PR 2 branch | Base: PR 2. Landlock, Bubblewrap, FilesystemTool, CommandTool, WikiClient. ~450 lines. |
| 4 | Bubbletea TUI | PR 4 → PR 3 branch | Base: PR 3. Model + 4 views + styles. ~350 lines. |
| 5 | DI wiring + go.mod + all tests | PR 5 → PR 4 branch | Base: PR 4. main.go, go.mod, domain/app/infra/TUI/E2E tests. ~500+ lines. |

## Phase 0: Model Tool-Use Verification

- [x] 0.1 Create `cmd/verify-models/main.go` — test llama3.2:3b, qwen2.5:3b, qwen3.5:2b XML tool-call capability (10 prompts each). Results: llama3.2:3b PASS (8/10, 80%), qwen2.5:3b FAIL (0/10), qwen3.5:2b FAIL (0/10)

## Phase 1: Domain Interfaces

- [x] 1.1 Create `internal/domain/tool.go` — Tool interface, ToolSchema, ToolCall, ToolResult types
- [x] 1.2 Create `internal/domain/sandbox.go` — Sandbox interface (Execute method)
- [x] 1.3 Create `internal/domain/permission.go` — PermissionRepository interface, Grant struct, GrantMode enum (ro/rw)
- [x] 1.4 Create `internal/domain/session.go` — Session, ChatMessage value objects
- [x] 1.5 Create `internal/domain/agent.go` — Agent interface: Run(ctx, session, msg) (response, error)

## Phase 2: Application Layer

- [x] 2.1 Create `internal/application/orchestrator.go` — ChatOrchestrator: session history mgmt, 8K token trim, dotfile map, system prompt injection, streaming channel
- [x] 2.2 Create `internal/application/executor.go` — ToolExecutor: XML `<tool_call>` parser, tool routing, sandbox + permission wrapping, 30s timeout, retry logic
- [x] 2.3 Create `internal/application/grantmgr.go` — GrantManager: grant lifecycle, GrantRequested event channel for TUI modal, path validation

## Phase 3: Infrastructure Adapters

- [x] 3.1 Create `internal/infrastructure/permission/memory.go` — InMemoryPermissionRepo: sync.RWMutex + map[string]GrantMode, implements PermissionRepository
- [x] 3.2 Create `internal/infrastructure/ollama/client.go` — OllamaClient: POST /api/chat, NDJSON scan via bufio.Scanner, token chan string, health check with model availability
- [x] 3.3 Create `internal/infrastructure/sandbox/landlock.go` — LandlockSandbox: LandlockAddRule + LandlockRestrictSelf at startup, read-only for allowed paths
- [x] 3.4 Create `internal/infrastructure/sandbox/bubblewrap.go` — BubblewrapSandbox: bwrap args (--unshare-all, --ro-bind, --bind, --tmpfs), exec.CommandContext, command blocklist + path traversal defense
- [x] 3.5 Create `internal/infrastructure/tools/filesystem.go` — FilesystemTool: read_file, write_file (with .bak + diff summary), list_dir, path validation (Clean + EvalSymlinks)
- [x] 3.6 Create `internal/infrastructure/tools/command.go` — CommandTool: run_command (allowlisted + flag-blocklisted), integrated with BubblewrapSandbox
- [x] 3.7 Create `internal/infrastructure/tools/wiki.go` — CheckSyntaxTool (hyprland/waybar/bash) + SearchWikiTool (local .md knowledge base)

## Phase 4: Bubbletea TUI

- [ ] 4.1 Create `internal/ui/tui/styles.go` — Lip Gloss theme (colors, borders, ANSI styles)
- [ ] 4.2 Create `internal/ui/tui/views/chat.go` — ChatView: Viewport with streaming token append, scroll
- [ ] 4.3 Create `internal/ui/tui/views/filepreview.go` — FilePreview: split pane, diff coloring, toggle
- [ ] 4.4 Create `internal/ui/tui/views/input.go` — InputBar: Bubbles TextInput, Enter to submit
- [ ] 4.5 Create `internal/ui/tui/views/grantmodal.go` — GrantModal: overlay with [Allow] [Deny], arrow-key navigation
- [ ] 4.6 Create `internal/ui/tui/model.go` — Bubbletea Model: Init/Update/View, split layout, Tab/Shift+Tab focus, Ctrl+C/q quit, resize reflow

## Phase 5: DI Wiring & Module Setup

- [ ] 5.1 Run `go mod init harvey` — add deps: bubbletea, lipgloss, bubbless, golang.org/x/sys, go-diff
- [ ] 5.2 Create `cmd/harvey/main.go` — DI wiring: instantiate adapters → domain → application, flag parsing (model, ollama URL), signal handling, startup health check, launch TUI

## Phase 6: Testing

- [ ] 6.1 Write `internal/domain/*_test.go` — table-driven tests: XML parse edge cases, ToolCall/ToolResult marshaling, mock Agent loop
- [x] 6.2 Write `internal/application/*_test.go` — unit tests: ChatOrchestrator 8K trim, ToolExecutor routing + timeout, GrantManager events
- [x] 6.3 Write `internal/infrastructure/*_test.go` — integration tests: Ollama health check, NDJSON golden files, bwrap path isolation, Landlock no-bypass (sandbox tests + tools tests complete; Landlock enforcement skipped without root)
- [ ] 6.4 Write `internal/ui/tui/*_test.go` — teatest component tests: model resize reflow, grant modal interaction, streaming render
- [ ] 6.5 Create `scripts/e2e-hyprland-test.sh` — 20-prompt E2E suite with real Ollama + sandbox; validates spec scenarios GIVEN/WHEN/THEN
