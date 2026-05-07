# Design: Harvey Core — Local AI Assistant for Arch Linux + Hyprland

## Architecture Overview

Clean/Hexagonal Go monorepo. Domain core defines interfaces; infrastructure implements them; UI drives use cases through ports. Dependency injection at `cmd/harvey/main.go` wires adapters → domain → application.

```
┌─────────────────────────────────────────────────────────┐
│  UI (Bubbletea TUI)                                     │
│  cmd/harvey/main.go ──wires──► ChatOrchestrator         │
├─────────────────────────────────────────────────────────┤
│  APPLICATION: ChatOrchestrator, ToolExecutor, GrantMgr  │
│       │           │            │                        │
│  DOMAIN PORTS (interfaces):                             │
│   ◄── Agent ──►  ◄── Tool ──►  ◄── Sandbox ──►         │
│   ◄── PermissionRepo ──►                                │
│       │           │            │                        │
│  INFRASTRUCTURE (adapters):                             │
│   OllamaClient  FilesystemTool  BubblewrapSandbox       │
│   (HTTP+NDJSON) (path-validated)  LandlockSandbox       │
│   WikiClient    InMemoryPermissions                     │
└─────────────────────────────────────────────────────────┘
```

**Rule**: Domain NEVER imports infrastructure/UI. Infrastructure implements domain interfaces.

## Key Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|----------|--------|----------|-----------|
| Language | Go 1.26 | Python, Hybrid | Single static binary (~15MB). Landlock via `golang.org/x/sys/unix`. Bubbletea TUI. |
| LLM client | Direct HTTP to Ollama `/api/chat` | `langchaingo`, Python ollama lib | API is one POST endpoint. ~50 lines of Go. Zero deps. |
| Sandbox primary | Bubblewrap subprocess for commands | Pure Landlock | Battle-tested (Flatpak). Landlock as in-process hardening layer. |
| Permission storage | In-memory map, session-scoped | SQLite, JSON file | Grants expire on exit by design. No persistence attack surface. |
| Tool-call format | XML (`<tool_call>`) | JSON, Markdown code fences | 1.5B models handle XML more reliably than JSON. Easier to partial-parse during streaming. |
| TUI library | Bubbletea + Lip Gloss + Bubbles | Textual (Python), tview | Go-native. Charmbracelet ecosystem is unmatched for TUI. Mouse support, viewport, styling. |
| Context management | Dotfile map (tree of ~/.config with summaries) | Full file contents | 8K token cap. Map compresses ~50 config files into ~2K tokens. |

## Domain Layer (`internal/domain/`)

| Interface / Type | Kind | Purpose |
|-----------------|------|---------|
| `Agent` | Interface | ReAct loop. `Run(ctx, session, msg) (response, error)`. Internal: calls LLM, parses XML, invokes tools, feeds results back. |
| `Tool` | Interface | `Name() string`, `Description() string`, `Schema() ToolSchema`, `Execute(ctx, input) (ToolResult, error)` |
| `Sandbox` | Interface | `Execute(ctx, paths []string, command string, env map[string]string) ([]byte, error)` |
| `PermissionRepository` | Interface | `Grant(dir, mode)`, `Revoke(dir)`, `Check(dir, mode) bool`, `List() []Grant` |
| `Session` | Value object | `ID string`, `Grants []Grant`, `History []ChatMessage`, `CreatedAt time.Time` |
| `Grant` | Value object | `Path string` (resolved, absolute), `Mode GrantMode` (ro/rw), `GrantedAt time.Time` |
| `ToolCall` | Value object | `Name string`, `Params map[string]string`, `Reason string` |
| `ToolResult` | Value object | `Output string`, `Error error`, `Duration time.Duration` |
| `ChatMessage` | Value object | `Role string` (system/user/assistant/tool), `Content string`, `Timestamp time.Time` |

## Application Layer (`internal/application/`)

| Use Case | Struct | Responsibility |
|----------|--------|----------------|
| `ChatOrchestrator` | `ChatOrchestrator` | Bridges TUI ↔ Agent. Manages `Session.History`. Trims to 8K tokens using dotfile map. Appends system prompt. Calls `Agent.Run()`. Returns streaming channel. |
| `ToolExecutor` | `ToolExecutor` | Parses XML `<tool_call>` blocks from LLM output. Routes to registered `Tool` implementations. Wraps execution in `Sandbox` + permission check. Returns `ToolResult`. |
| `GrantManager` | `GrantManager` | Session-scoped grant lifecycle. Emits `GrantRequested` events for TUI modal. Validates paths before granting. |

## Infrastructure Layer (`internal/infrastructure/`)

| Adapter | Package | Implements | Key Detail |
|---------|---------|------------|------------|
| `OllamaClient` | `ollama/` | Domain `Agent` (via application) | POST `/api/chat`. `bufio.Scanner` on response body for NDJSON. Token channel `chan string`. Timeout via `context.WithTimeout`. |
| `BubblewrapSandbox` | `sandbox/` | Domain `Sandbox` | Constructs `bwrap` args: `--unshare-all`, `--ro-bind` for allowed paths, `--bind` for writable, `--tmpfs /tmp`, `--no-network`. `exec.CommandContext`. |
| `LandlockSandbox` | `sandbox/` | Domain `Sandbox` | `unix.LandlockAddRule()` + `unix.LandlockRestrictSelf()`. Called once at startup. Immutable after. |
| `FilesystemTool` | `tools/` | Domain `Tool` | `read_file`, `write_file` (`.bak` + diff via `go-diff`), `list_dir`. All paths go through `filepath.Clean` + `filepath.EvalSymlinks` before validation. |
| `WikiClient` | `tools/` | Domain `Tool` | `search_docs`: `curl`-based local Arch Wiki dump search. |
| `CommandTool` | `tools/` | Domain `Tool` | `run_command` + `check_syntax`. Hardcoded allowlist: `hyprctl`, `systemctl --user`, `journalctl --user`, `grep`, `ls`, `cat`, `pacman -Q`. |
| `InMemoryPermissionRepo` | `permission/` | Domain `PermissionRepository` | `sync.RWMutex`-protected `map[string]GrantMode`. |

## UI Layer (`internal/ui/tui/`)

**Bubbletea Model**:
```
┌──────────────────────────────────────────┐
│ Harvey — Arch Assistant    [Grants: 2]   │ ← title bar
├────────────────────┬─────────────────────┤
│ Chat History       │ File Preview / Diff │
│ (viewport scroll)  │ (syntax highlighted)│
│                    │                     │
│ > streaming text… █│                     │
├────────────────────┴─────────────────────┤
│ > Type your question…           [Enter]  │ ← input bar
└──────────────────────────────────────────┘
```

- **Components**: `chatView` (messages with Bubbletea `Viewport`), `filePreview` (split pane, diff coloring), `inputBar` (Bubbles `TextInput`), `grantModal` (overlay with `[Allow] [Deny]`)
- **Communication**: TUI subscribes to `ChatOrchestrator.Stream()` channel for token deltas. Sends user messages via `ChatOrchestrator.Send(msg)`. Grant modal triggered by `GrantManager` channel.
- **Key bindings**: `Enter` submit, `Tab`/`Shift+Tab` focus, `Ctrl+C`/`q` quit, `Ctrl+D` file preview toggle, `PgUp/PgDn` scroll chat

## Data Flow: Typical Interaction

```
User: "organize my hyprland config"
  │
  ▼
TUI.InputBar ──► ChatOrchestrator.Send(msg)
  │
  ▼
ChatOrchestrator ──► Agent.Run() ──► OllamaClient (POST /api/chat, stream)
  │                                      │
  │                              NDJSON tokens ←── Ollama
  ▼                                      │
ToolExecutor.ParseXML(stream)◄───────────┘
  │
  │ <tool_call><name>list_dir</name><path>~/.config/hypr</path></tool_call>
  ▼
ToolExecutor → PermissionRepo.Check(path, "ro") → GrantManager (if needed)
  │
  ▼
Tool.Execute() ──► Sandbox.Execute() ──► Landlock (in-process) or bwrap (subprocess)
  │
  ▼
ToolResult ──► Agent.Run() ──► OllamaClient (second call with tool result)
  │
  │ <tool_call><name>write_file</name><path>~/.config/hypr/hyprland.conf</path>…</tool_call>
  │
  ▼
[Permission check → Grant modal → User allows → write + .bak + diff]
  │
  ▼
ToolResult ──► Agent ──► OllamaClient ──► "Done. I organized your config…"
  │
  ▼
ChatOrchestrator.Stream() ──► TUI.ChatView.Append(text)
```

## Tool Call XML Format

| Tool | XML Request | XML Response |
|------|------------|--------------|
| `read_file` | `<tool_call><name>read_file</name><path>~/.config/hypr/hyprland.conf</path></tool_call>` | `<tool_result><output>…contents…</output></tool_result>` |
| `write_file` | `<tool_call><name>write_file</name><path>~/.config/hypr/hyprland.conf</path><content>…</content></tool_call>` | `<tool_result><output>wrote 52 lines</output><diff>@@ -1,3 +1,3 @@…</diff></tool_result>` |
| `list_dir` | `<tool_call><name>list_dir</name><path>~/.config/hypr</path></tool_call>` | `<tool_result><output>hyprland.conf\nhyprlock.conf</output></tool_result>` |
| `run_command` | `<tool_call><name>run_command</name><cmd>hyprctl monitors</cmd></tool_call>` | `<tool_result><output>Monitor eDP-1: …</output></tool_result>` |
| `check_syntax` | `<tool_call><name>check_syntax</name><path>~/.config/hypr/hyprland.conf</path></tool_call>` | `<tool_result><output>OK</output><errors></errors></tool_result>` |
| `search_docs` | `<tool_call><name>search_docs</name><query>hyprland gaps</query></tool_call>` | `<tool_result><output>Arch Wiki: …</output></tool_result>` |

Errors: `<tool_result><error>ErrFileNotFound: /home/user/.config/sway/config</error></tool_result>`

## Module Structure

```
cmd/harvey/main.go                 ← Entry point: flag parsing, signal handling, DI wiring
internal/
  domain/
    agent.go                       ← Agent interface (ReAct loop contract)
    tool.go                        ← Tool interface, ToolSchema, ToolCall, ToolResult
    sandbox.go                     ← Sandbox interface
    permission.go                  ← PermissionRepository interface, Grant, GrantMode
    session.go                     ← Session, ChatMessage value objects
  application/
    orchestrator.go                ← ChatOrchestrator: session mgmt, 8K trim, dotfile map
    executor.go                    ← ToolExecutor: XML parse, route, sandbox wrap
    grantmgr.go                    ← GrantManager: grant lifecycle, events
  infrastructure/
    ollama/
      client.go                    ← OllamaClient: HTTP POST, NDJSON scan, token chan
    sandbox/
      bubblewrap.go                ← BubblewrapSandbox: bwrap args, subprocess
      landlock.go                  ← LandlockSandbox: syscall restrict, startup init
    tools/
      filesystem.go                ← FilesystemTool: read/write(list)/list + path val
      command.go                   ← CommandTool: run_command + check_syntax (allowlist)
      wiki.go                      ← WikiClient: Arch Wiki search (local dump)
    permission/
      memory.go                    ← InMemoryPermissionRepo: map + RWMutex
  ui/tui/
    model.go                       ← Bubbletea Model: Init, Update, View
    views/
      chat.go                      ← ChatView (Viewport + Lip Gloss styles)
      filepreview.go               ← FilePreview (diff coloring, syntax)
      input.go                     ← InputBar (Bubbles TextInput)
      grantmodal.go                ← GrantModal overlay
    styles.go                      ← Lip Gloss theme (colors, borders)
```

## Testing Strategy

| Layer | Approach | Tools |
|-------|----------|-------|
| Domain | Unit: mock all interfaces. Table-driven tests for Agent loop XML parsing edge cases. | `go test`, `testing` stdlib |
| Application | Unit: mock `Agent`, `Sandbox`, `PermissionRepository`. Test ChatOrchestrator 8K trim, ToolExecutor routing. | `go test` |
| Infrastructure | Integration: real Ollama (health check), real bwrap subprocess (path isolation), Landlock syscall (no-bypass test). Golden files for NDJSON parsing. | `go test` + `testdata/` |
| TUI | Component: `teatest` for Bubbletea model interactions. Test resize, grant modal, streaming render. | `teatest` |
| E2E | Full agent loop with real Ollama + deepseek-r1:1.5b + sandbox. 20-prompt Hyprland test suite. | Manual + script |
