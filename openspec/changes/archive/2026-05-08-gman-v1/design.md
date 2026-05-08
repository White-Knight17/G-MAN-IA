# Design: G-MAN v1.0 — Native Desktop GUI

## Technical Approach

Replace Bubbletea TUI with Tauri v2 + Svelte 5 desktop shell. Go core becomes a JSON-RPC 2.0 sidecar on stdin/stdout. All domain/application/infrastructure layers unchanged. `cmd/gman` preserved as TUI fallback. New `cmd/gman-server` entry point with NDJSON transport. Rust shell spawns sidecar, relays IPC, manages tray/window. Svelte 5 drives chat, permissions, onboarding via typed RPC.

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|----------|--------|----------|-----------|
| IPC protocol | JSON-RPC 2.0 NDJSON over stdin/stdout | gRPC, WebSocket, Unix socket | Zero-dependency transport; no port binding; works inside Tauri sidecar model |
| Sidecar entry point | New `cmd/gman-server/main.go` | Modifying `cmd/gman` | Preserves TUI fallback; additive change only |
| `StreamRun()` signature | `StreamRun(ctx, input, session) <-chan StreamEvent` | Callback-based, SSE | Go channels are idiomatic; single consumer; backpressure via buffer |
| UI framework | Svelte 5 (runes) | React, Vue, Solid | Minimal bundle; `$state`/`$derived`/`$effect` map naturally to JSON-RPC events |
| Rust shell scope | ~150 LOC relay only | Full Rust backend | Go core owns all logic; Rust is a thin proxy to avoid duplication |
| Window config | Frameless 420×700, always-on-top optional | Native title bar | Sidebar assistant aesthetic; consistent with companion UX |

## Data Flow

```
User types message
  │
  ▼
Svelte ChatView ──[invoke('relay_request')]──► Rust main.rs
  │                                                │
  │                                          write JSON-RPC to Go stdin
  │                                                │
  │                                                ▼
  │                                         Go jsonrpc server
  │                                           │
  │                                     ChatOrchestrator.HandleMessage()
  │                                           │
  │                                     OllamaClient.StreamRun()
  │                                           │
  │                                     NDJSON chunks to stdout
  │                                           │
  ▼                                                │
Svelte <──[emit('notification')]──◄── Rust reads stdout
(streaming text)                             (line by line)
```

Go sends `ready` notification on startup (`{"jsonrpc":"2.0","method":"ready","params":{"version":"1.0.0"}}`). Rust reads it, then loads the Svelte window.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/gman-server/main.go` | **Create** | Stdio RPC server entry point; reuses same DI as `cmd/gman` |
| `internal/transport/jsonrpc.go` | **Create** | JSON-RPC 2.0 server: reads NDJSON from stdin, writes to stdout; dispatches to handler map |
| `internal/transport/jsonrpc_test.go` | **Create** | Unit tests for request parsing, method dispatch, error formatting |
| `internal/domain/agent.go` | **Modify** | Add `StreamRun(ctx, input, session) (<-chan StreamEvent, error)` + `StreamEvent` type |
| `internal/domain/session.go` | **Modify** | Add `json:"..."` struct tags to `Session`, `ChatMessage`, `Grant` |
| `internal/infrastructure/ollama/client.go` | **Modify** | Implement `StreamRun()` — POST `/api/chat` with `stream:true`, emit NDJSON chunks |
| `internal/application/orchestrator.go` | **Modify** | Add `HandleMessageStream()` method returning `<-chan StreamEvent` |
| `cmd/gman/main.go` | **Preserve** | No changes — TUI fallback |
| `app/src-tauri/main.rs` | **Create** | Window config, system tray, sidecar spawn, stdin/stdout relay |
| `app/src-tauri/Cargo.toml` | **Create** | Tauri v2 deps: `tauri`, `tauri-plugin-shell`, `tauri-plugin-notification` |
| `app/src/lib/RpcClient.svelte.ts` | **Create** | Typed wrapper: `invoke('relay_request', {method, params}) → Promise<RpcResponse>` |
| `app/src/routes/+page.svelte` | **Create** | Main chat view: message list, input bar, typing indicator |
| `app/src/lib/NotificationHandler.svelte` | **Create** | Listens to Tauri events, renders permission dialogs, file previews |
| `Makefile` | **Create** | `build-core`, `build-ui`, `build` — cross-platform `.AppImage/.deb/.rpm` |
| `scripts/build.sh` | **Create** | Tauri bundler orchestration with sidecar naming convention |

## Interfaces / Contracts

### JSON-RPC 2.0 Messages

**Request (Svelte → Go)**:
```json
{"jsonrpc":"2.0","id":1,"method":"agent.chat","params":{"input":"fix my hyprland config"}}
```

**Streaming notification (Go → Svelte)**:
```json
{"jsonrpc":"2.0","method":"agent.token","params":{"token":"I'll","session_id":"abc"}}
{"jsonrpc":"2.0","method":"agent.token","params":{"token":" help","session_id":"abc"}}
```

**Tool permission request (Go → Svelte)**:
```json
{"jsonrpc":"2.0","method":"permission.request","params":{"id":"p1","tool":"write_file","path":"/home/user/.config/hypr/hyprland.conf","mode":"rw"}}
```

**Permission response (Svelte → Go)**:
```json
{"jsonrpc":"2.0","id":1,"result":{"granted":true,"id":"p1"}}
```

### Go Sidecar Main Loop (sketch)

```go
scanner := bufio.NewScanner(os.Stdin)
encoder := json.NewEncoder(os.Stdout)
writeNotification(encoder, "ready", map[string]string{"version": "1.0.0"})
for scanner.Scan() {
    var req jsonrpc.Request
    json.Unmarshal(scanner.Bytes(), &req)
    encoder.Encode(handler.Dispatch(req))
}
```

### StreamEvent type

```go
type StreamEvent struct {
    Type    string // "token", "tool_call", "permission_request", "done", "error"
    Token   string
    Tool    string
    Path    string
    Content string
    Error   string
}
```

### Svelte RPC Client (concept)

```typescript
let reqId = $state(0);
const call = (m: string, p: any) => invoke('relay_request', {jsonrpc:'2.0', id: ++reqId, method: m, params: p});
```

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Go unit | jsonrpc transport, StreamRun, NDJSON | `go test ./internal/transport/` — stdio pipes |
| Go integration | Full RPC loop with mock Agent | `go test ./cmd/gman-server/` |
| Go regression | Existing 85 tests | `go test ./...` — must pass 0 regressions |
| E2E | 5/5 tool prompts via GUI | Playwright + real Tauri binary |

Key components: `App.svelte` → `OnboardingWizard`, `ChatView` (message list + input + typing indicator), `FilePreview`, `PermissionDialog`, `SettingsPanel`. All use Svelte 5 runes for state.

## Migration / Rollout

No migration. `cmd/gman` preserved as TUI fallback. Delete `app/`, `cmd/gman-server/`, `internal/transport/` to revert.

## Open Questions

- [ ] Wayland tray fallback: Tauri tray API vs. DBus direct for Hyprland/waybar
- [ ] Sidecar binary naming convention per arch for Tauri bundler
- [ ] Permission dialog timeout: auto-deny after 30s or block indefinitely?
