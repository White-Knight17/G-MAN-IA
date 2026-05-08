// G-MAN v1.0 — TypeScript IPC Client
// JSON-RPC 2.0 transport via Tauri invoke() bridge

import { invoke } from "@tauri-apps/api/core";

// ── Types ──────────────────────────────────────────────────────────────────

export type JSONRPCRequest = {
  jsonrpc: "2.0";
  id: number;
  method: string;
  params: Record<string, unknown>;
};

export type JSONRPCResponse = {
  jsonrpc: "2.0";
  id: number;
  result?: unknown;
  error?: { code: number; message: string };
};

export type StreamEvent = {
  type: "token" | "tool_call" | "tool_result" | "error" | "done";
  data?: unknown;
};

type JSONRPCNotification = {
  jsonrpc: "2.0";
  method: string;
  params: Record<string, unknown>;
};

// ── State ──────────────────────────────────────────────────────────────────

let nextId = 1;

// ── relayRequest ───────────────────────────────────────────────────────────

/**
 * Sends a JSON-RPC 2.0 request through the Tauri invoke bridge.
 * The Rust layer writes the request to Go's stdin and reads the response from stdout.
 *
 * @throws Error if the JSON-RPC response contains an error or if the sidecar is not ready.
 */
export async function relayRequest(
  method: string,
  params: Record<string, unknown>,
): Promise<JSONRPCResponse> {
  const id = nextId++;
  const request: JSONRPCRequest = {
    jsonrpc: "2.0",
    id,
    method,
    params,
  };

  const response = await invoke<JSONRPCResponse>("relay_request", request);

  // If the response contains an error, throw
  if (response.error) {
    throw new Error(response.error.message);
  }

  return response;
}

// ── streamChat ─────────────────────────────────────────────────────────────

/**
 * Opens a streaming chat connection through the Tauri invoke bridge.
 * Yields StreamEvent tokens as NDJSON lines arrive from the Go sidecar.
 *
 * @param message The user's message text
 * @yields StreamEvent tokens, tool calls, tool results, errors, and done signal
 * @throws Error if the sidecar is not ready or connection fails
 */
export async function* streamChat(
  message: string,
): AsyncGenerator<StreamEvent> {
  const raw = await invoke<string>("stream_chat", { input: message });

  // Parse NDJSON: one JSON-RPC notification per line
  const lines = raw.split("\n").map((l) => l.trim()).filter((l) => l.length > 0);

  for (const line of lines) {
    const notification: JSONRPCNotification = JSON.parse(line);

    switch (notification.method) {
      case "stream.token":
        yield { type: "token", data: notification.params.token };
        break;

      case "stream.tool_call":
        yield {
          type: "tool_call",
          data: {
            tool: notification.params.tool,
            path: notification.params.path,
          },
        };
        break;

      case "stream.tool_result":
        yield { type: "tool_result", data: notification.params.content };
        break;

      case "stream.error":
        yield { type: "error", data: notification.params.error };
        break;

      case "stream.done":
        yield { type: "done" };
        return;

      default:
        // Unknown notification — skip
        break;
    }
  }
}
