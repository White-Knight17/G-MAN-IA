# jsonrpc-transport Specification

## Purpose

JSON-RPC 2.0 communication protocol between Svelte frontend and Go sidecar via Tauri Rust relay over stdin/stdout with NDJSON framing, including model management, configuration, and chat streaming methods.

## Requirements

### Requirement: JSON-RPC Transport

The system MUST use JSON-RPC 2.0 over stdin/stdout with NDJSON framing (one JSON object per line). The frontend SHALL send requests via Tauri `invoke()`; the Rust layer SHALL write the JSON-RPC request to Go's stdin. Go SHALL write responses and streaming events to stdout as NDJSON lines; the Rust layer SHALL emit them as Tauri events consumed by the frontend. During streaming responses, Go MUST flush after every NDJSON line for real-time delivery. If the sidecar is not ready, the Rust layer SHALL return an immediate error.

#### Scenario: Request/response cycle

- GIVEN the sidecar is running and ready
- WHEN the frontend sends `{"jsonrpc":"2.0","method":"chat.send","params":{...},"id":1}` via `invoke()`
- THEN Rust writes the JSON line to Go stdin, Go processes and writes `{"jsonrpc":"2.0","result":{...},"id":1}` to stdout, Rust emits a Tauri event with the result, and the frontend receives it

#### Scenario: Streaming token delivery

- GIVEN a chat request is in progress
- WHEN Go writes NDJSON lines `{"jsonrpc":"2.0","method":"stream.token","params":{"delta":"Hello"}}` and `{"jsonrpc":"2.0","method":"stream.token","params":{"delta":" world"}}` to stdout with flush after each
- THEN each line is emitted as a separate Tauri event and the UI renders tokens incrementally with <100ms apparent latency

#### Scenario: JSON-RPC error response

- GIVEN the sidecar is running
- WHEN Go encounters an error and writes `{"jsonrpc":"2.0","error":{"code":-32000,"message":"model not found"},"id":1}`
- THEN the Rust layer emits the error as a Tauri event and the frontend displays the error message

#### Scenario: Sidecar not ready

- GIVEN the Go sidecar has not completed initialization
- WHEN the frontend sends any JSON-RPC request via `invoke()`
- THEN the Rust layer returns an error immediately without writing to stdin, with message "sidecar not ready"

### Requirement: Model Management RPC Methods

The system MUST provide JSON-RPC method `model.list` that returns available Ollama models and the currently active model. The system MUST provide JSON-RPC method `model.pull` that triggers `ollama pull <name>` with streaming progress notifications via `SendNotification`.

#### Scenario: List available models

- GIVEN the frontend sends `{"jsonrpc":"2.0","method":"model.list","id":1}`
- WHEN the sidecar processes the request
- THEN it queries Ollama `/api/tags` and returns `{"jsonrpc":"2.0","result":{"models":["llama3","codellama"],"active":"llama3"},"id":1}`

#### Scenario: Pull model with streaming progress

- GIVEN the frontend sends `{"jsonrpc":"2.0","method":"model.pull","params":{"name":"codellama"},"id":2}`
- WHEN the pull is in progress
- THEN the sidecar sends notification events like `{"jsonrpc":"2.0","method":"pull.progress","params":{"name":"codellama","percent":45}}`
- AND on completion sends `{"jsonrpc":"2.0","method":"pull.complete","params":{"name":"codellama"}}`
- AND finally returns `{"jsonrpc":"2.0","result":{"status":"success"},"id":2}`

#### Scenario: Pull non-existent model

- GIVEN the frontend sends `{"jsonrpc":"2.0","method":"model.pull","params":{"name":"nonexistent"},"id":3}`
- WHEN Ollama reports the model does not exist
- THEN the sidecar returns `{"jsonrpc":"2.0","error":{"code":-32001,"message":"model 'nonexistent' not found on registry"},"id":3}`

### Requirement: Configuration RPC Methods

The system MUST provide JSON-RPC method `config.set_api_key` that stores an API key for a given provider (`openai`, `anthropic`, `groq`) in the sidecar's config.json. The system MUST provide JSON-RPC method `config.get` that returns the current configuration including active provider and whether a key is set (without exposing the key value).

#### Scenario: Set API key

- GIVEN the frontend sends `{"jsonrpc":"2.0","method":"config.set_api_key","params":{"provider":"openai","key":"sk-xxx"},"id":4}`
- WHEN the sidecar processes the request
- THEN the key is written to config.json under the `openai` provider entry
- AND the response is `{"jsonrpc":"2.0","result":{"status":"ok"},"id":4}`

#### Scenario: Get config hides key value

- GIVEN an API key has been set for `openai`
- WHEN the frontend sends `{"jsonrpc":"2.0","method":"config.get","id":5}`
- THEN the response includes `{"provider":"openai","has_api_key":true}` but NOT the actual key value

#### Scenario: Set invalid provider

- GIVEN the frontend sends `{"jsonrpc":"2.0","method":"config.set_api_key","params":{"provider":"invalid","key":"xxx"},"id":6}`
- WHEN the sidecar processes the request
- THEN it returns `{"jsonrpc":"2.0","error":{"code":-32002,"message":"unsupported provider: invalid"},"id":6}`
