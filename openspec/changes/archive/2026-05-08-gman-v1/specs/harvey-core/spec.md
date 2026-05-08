# Delta for Harvey Core

## MODIFIED Requirements

### Requirement: ReAct Agent Loop

The system MUST implement a ReAct agent loop that orchestrates user→LLM→tools→LLM cycles. Each message cycle SHALL parse XML tool calls from the LLM response, execute them sandboxed, and feed results back. The loop MUST enforce a 30-second per-step timeout and SHALL retry once on timeout or malformed output. The agent MUST use the configured model and MAY fall back to a secondary model if the primary fails 3 consecutive tool-call parses. The agent SHALL also expose `StreamRun(ctx, input) <-chan StreamEvent` alongside the existing blocking `Run()`; StreamEvent types include `token`, `tool_call`, `tool_result`, `error`, and `done`.
(Previously: only blocking `Run()` method; now adds `StreamRun` channel-based variant.)

#### Scenario: Happy path tool execution

- GIVEN the agent receives "Show my Hyprland config"
- WHEN the LLM responds with `<tool_call><name>read_file</name><path>~/.config/hypr/hyprland.conf</path></tool_call>`
- THEN the agent parses the XML, executes read_file in the sandbox, and returns the file contents to the LLM for a final response

#### Scenario: Tool call timeout triggers retry

- GIVEN a tool call has been dispatched to the sandbox
- WHEN 30 seconds elapse without a result
- THEN the agent SHALL cancel the call, retry once, and if it fails again return an error to the LLM

#### Scenario: Consecutive parse failures trigger model fallback

- GIVEN the primary model returns malformed XML for 3 consecutive cycles
- WHEN the 3rd parse fails
- THEN the agent SHALL switch to the fallback model and log the transition

#### Scenario: StreamRun emits streaming events

- GIVEN the agent receives "List my config files" via `StreamRun`
- WHEN the LLM streams token deltas and a tool call
- THEN the channel emits `StreamEvent{Type:"token", Data:"List"}` followed by subsequent token events, then `StreamEvent{Type:"tool_call", Data:{...}}`, then `StreamEvent{Type:"tool_result", Data:{...}}`, and finally `StreamEvent{Type:"done"}` before closing

### Requirement: Ollama HTTP Client

The system MUST provide an HTTP client for Ollama `/api/chat` supporting NDJSON streaming responses with `stream:true`. The client SHALL verify Ollama connectivity and model availability at startup. On connection errors the client MUST return structured errors; on partial stream errors it SHALL return whatever tokens were received plus an error marker. The streamed tokens SHALL be mapped to the `StreamEvent` type for consumption by `StreamRun`.
(Previously: NDJSON streaming worked but lacked standardized `StreamEvent` envelope; now explicitly maps to the shared event type.)

#### Scenario: Successful streaming chat

- GIVEN Ollama is running and deepseek-r1:1.5b is pulled
- WHEN the client POSTs to `/api/chat` with a messages array
- THEN it receives NDJSON lines with `message.content` deltas and yields them via a Go channel

#### Scenario: Connection refused during operation

- GIVEN the client was connected and streaming
- WHEN Ollama crashes mid-stream
- THEN the client SHALL return accumulated tokens with an `ErrStreamInterrupted` sentinel and log the failure

#### Scenario: Model not available at startup

- GIVEN the primary model is not pulled
- WHEN the client performs startup health check
- THEN it SHALL return `ErrModelNotFound` and suggest `ollama pull`

#### Scenario: Streaming tokens use StreamEvent format

- GIVEN a chat request with `stream:true`
- WHEN Ollama returns NDJSON lines
- THEN each delta is wrapped as `StreamEvent{Type:"token", Data:delta}` and written to the channel, with a final `StreamEvent{Type:"done"}`

### Requirement: Dotfile Tools

The system MUST provide six sandboxed tools. `read_file` SHALL return file contents; `write_file` SHALL create a `.bak` copy before writing and return a diff; `list_dir` SHALL enumerate directory entries excluding hidden files by default; `run_command` MUST execute only allowlisted commands in Bubblewrap and return stdout/stderr; `check_syntax` SHALL validate config syntax via allowlisted linters; `search_wiki` SHALL grep the local Arch Wiki dump. All tools MUST return structured errors on failure. Tool results SHALL be wrappable as `StreamEvent{Type:"tool_result", Data:...}` for the streaming transport.
(Previously: tools returned plain structs; now also supports StreamEvent envelope.)

#### Scenario: write_file creates backup and diff

- GIVEN `~/.config/hypr/hyprland.conf` exists with 50 lines
- WHEN `write_file` replaces it with 52-line content
- THEN a `.bak` copy of the original is created, and a unified diff of the change is returned

#### Scenario: read_file on missing path

- GIVEN `~/.config/sway/config` does not exist
- WHEN `read_file` is called for that path
- THEN it SHALL return `ErrFileNotFound` with the resolved absolute path

#### Scenario: run_command outside allowlist

- GIVEN the allowlist is `[hyprctl, systemctl --user]`
- WHEN `run_command` attempts `sudo pacman -S kitty`
- THEN it SHALL return `ErrCommandNotAllowed` listing the attempted command

#### Scenario: check_syntax on valid config

- GIVEN `~/.config/hypr/hyprland.conf` is syntactically valid
- WHEN `check_syntax` runs `hyprctl validate` on it
- THEN it returns success with an empty error list

#### Scenario: search_wiki with no matches

- GIVEN the local Arch Wiki dump is indexed
- WHEN `search_wiki` queries "hyperland"
- THEN it SHALL return zero results with a did-you-mean suggestion if a close match exists

#### Scenario: Tool result via StreamEvent

- GIVEN `read_file` is called from `StreamRun`
- WHEN the tool completes successfully
- THEN the result is emitted as `StreamEvent{Type:"tool_result", Data:json.Marshal(result)}` to the streaming channel

## REMOVED Requirements

### Requirement: Bubbletea Terminal UI

(Reason: Replaced by `chat-sidebar-ui` (Tauri v2 + Svelte 5 GUI). The TUI codebase at `core/cmd/gman/` is preserved as a fallback entry point. All GUI interaction — chat, streaming display, grant modals, file preview — moves to the Svelte frontend.)
