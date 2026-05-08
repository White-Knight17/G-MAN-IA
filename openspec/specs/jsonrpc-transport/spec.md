# jsonrpc-transport Specification

## Purpose

JSON-RPC 2.0 communication protocol between Svelte frontend and Go sidecar via Tauri Rust relay over stdin/stdout with NDJSON framing.

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
