# Spec: Harvey Core

Local AI assistant for Arch Linux + Hyprland. Offline, single static Go binary with Bubbletea TUI, Ollama-backed ReAct agent loop, and defense-in-depth sandboxed dotfile tools.

## Requirements

### Requirement: ReAct Agent Loop

The system MUST implement a ReAct agent loop that orchestrates user→LLM→tools→LLM cycles. Each message cycle SHALL parse XML tool calls from the LLM response, execute them sandboxed, and feed results back. The loop MUST enforce a 30-second per-step timeout and SHALL retry once on timeout or malformed output. The agent MUST use the configured model and MAY fall back to a secondary model if the primary fails 3 consecutive tool-call parses.

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

### Requirement: Ollama HTTP Client

The system MUST provide an HTTP client for Ollama `/api/chat` supporting NDJSON streaming responses. The client SHALL verify Ollama connectivity and model availability at startup. On connection errors the client MUST return structured errors; on partial stream errors it SHALL return whatever tokens were received plus an error marker.

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

### Requirement: Sandboxed Execution

The system MUST run all file and command operations inside a defense-in-depth sandbox. Path validation SHALL resolve and verify paths are within `~/.config`. Landlock SHALL restrict filesystem access at the syscall level. Bubblewrap MUST containerize all `run_command` subprocesses with `--no-network` and a read-only rootfs. The sandbox MISN'T allow writes outside allowed paths and MUST refuse disallowed commands.

#### Scenario: Allowed path write succeeds

- GIVEN the user has granted rw on `~/.config/hypr`
- WHEN `write_file` targets `~/.config/hypr/hyprland.conf`
- THEN the sandbox allows the write through all three layers

#### Scenario: System file write blocked

- GIVEN no grant exists for `/etc`
- WHEN a tool attempts `write_file /etc/hosts`
- THEN path validation rejects it before Landlock or Bubblewrap are invoked

#### Scenario: Disallowed command refused

- GIVEN the command allowlist is `[hyprctl, systemctl --user, journalctl]`
- WHEN `run_command` attempts `rm -rf /`
- THEN Bubblewrap refuses the command and returns `ErrCommandNotAllowed`

### Requirement: Session-Scoped Permissions

The system MUST enforce explicit user grants before any write or list operation on a directory. Grants SHALL be per-directory, per-mode (ro/rw), and expire on session exit. The TUI MUST prompt the user for confirmation when a tool requests access to an ungranted directory. Grants MAY be revoked mid-session.

#### Scenario: First-time directory access prompts grant

- GIVEN no grant exists for `~/.config/hypr`
- WHEN a tool requests write access to that directory
- THEN the TUI displays a grant confirmation dialog showing the path and mode (rw)

#### Scenario: Grant expires on exit

- GIVEN the user granted rw on `~/.config/waybar` during the current session
- WHEN the session ends (process exit or `Ctrl+C`)
- THEN all grants are discarded; next session requires new confirmation

#### Scenario: Revoke mid-session

- GIVEN an active grant for `~/.config/kitty`
- WHEN the user issues `revoke kitty` via the TUI
- THEN subsequent tool access to that directory triggers a new grant prompt

### Requirement: Bubbletea Terminal UI

The system MUST provide a Bubbletea TUI with a split layout: chat history on the left, file preview/diff on the right. The TUI SHALL render streaming LLM tokens incrementally and display a grant confirmation dialog as a modal. Keyboard navigation MUST support `Tab`/`Shift+Tab` for focus cycling, `Enter` to submit, and `Ctrl+C`/`q` to quit. The layout SHALL reflow correctly on terminal resize.

#### Scenario: Streaming token display

- GIVEN the agent is processing a user message
- WHEN Ollama streams token deltas
- THEN each delta is appended to the chat panel's current message with visible cursor

#### Scenario: Terminal resize reflows layout

- GIVEN the TUI is running at 80×24
- WHEN the terminal resizes to 120×40
- THEN the split layout reflows proportionally without truncation or glitching

#### Scenario: Grant confirmation modal

- GIVEN a tool requests access to an ungranted directory
- WHEN the permission model emits a grant request
- THEN the TUI displays a modal overlay with path, mode, and `[Allow] [Deny]` buttons navigable via arrow keys

### Requirement: Dotfile Tools

The system MUST provide six sandboxed tools. `read_file` SHALL return file contents; `write_file` SHALL create a `.bak` copy before writing and return a diff; `list_dir` SHALL enumerate directory entries excluding hidden files by default; `run_command` MUST execute only allowlisted commands in Bubblewrap and return stdout/stderr; `check_syntax` SHALL validate config syntax via allowlisted linters; `search_wiki` SHALL grep the local Arch Wiki dump. All tools MUST return structured errors on failure.

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
