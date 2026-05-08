import { describe, it, expect, vi, beforeEach } from "vitest";
import { invoke } from "@tauri-apps/api/core";

// The module under test — does NOT exist yet (RED)
import {
  relayRequest,
  streamChat,
  type JSONRPCRequest,
  type JSONRPCResponse,
  type StreamEvent,
} from "../rpc";

// ── Mock Tauri invoke ──
vi.mock("@tauri-apps/api/core", () => ({
  invoke: vi.fn(),
}));

beforeEach(() => {
  vi.clearAllMocks();
});

// ============================================================================
// relayRequest tests
// ============================================================================

describe("relayRequest", () => {
  it("sends a JSON-RPC formatted request via Tauri invoke", async () => {
    const mockResponse: JSONRPCResponse = {
      jsonrpc: "2.0",
      id: 1,
      result: { reply: "G-MAN here!" },
    };

    vi.mocked(invoke).mockResolvedValueOnce(mockResponse);

    const result = await relayRequest("agent.chat", {
      input: "hello",
      session_id: "abc",
    });

    expect(invoke).toHaveBeenCalledTimes(1);

    // Verify the request was properly formatted
    const callArgs = vi.mocked(invoke).mock.calls[0];
    expect(callArgs[0]).toBe("relay_request");
    const payload: JSONRPCRequest = callArgs[1] as JSONRPCRequest;
    expect(payload.jsonrpc).toBe("2.0");
    expect(payload.method).toBe("agent.chat");
    expect(payload.params).toEqual({ input: "hello", session_id: "abc" });
    expect(payload.id).toBeGreaterThan(0);

    expect(result).toBe(mockResponse);
  });

  it("increments request IDs across multiple calls", async () => {
    const mockResponse: JSONRPCResponse = {
      jsonrpc: "2.0",
      id: 1,
      result: "ok",
    };

    vi.mocked(invoke).mockResolvedValue(mockResponse);

    await relayRequest("method.a", {});
    const call1Args = vi.mocked(invoke).mock.calls[0];
    const id1 = (call1Args[1] as JSONRPCRequest).id;

    await relayRequest("method.b", {});
    const call2Args = vi.mocked(invoke).mock.calls[1];
    const id2 = (call2Args[1] as JSONRPCRequest).id;

    expect(id2).toBeGreaterThan(id1);
  });

  it("returns the parsed response result on success", async () => {
    const mockResponse: JSONRPCResponse = {
      jsonrpc: "2.0",
      id: 1,
      result: { data: "config content", path: "/home/user/.config" },
    };

    vi.mocked(invoke).mockResolvedValueOnce(mockResponse);

    const result = await relayRequest("tool.read_file", {
      path: "/home/user/.config/hypr/hyprland.conf",
    });

    expect(result).toEqual(mockResponse);
  });

  it("throws on JSON-RPC error response", async () => {
    const mockResponse: JSONRPCResponse = {
      jsonrpc: "2.0",
      id: 1,
      error: { code: -32000, message: "Model not found" },
    };

    vi.mocked(invoke).mockResolvedValueOnce(mockResponse);

    await expect(
      relayRequest("agent.chat", { input: "hello" }),
    ).rejects.toThrow("Model not found");
  });

  it("throws when Tauri invoke fails (sidecar not ready)", async () => {
    vi.mocked(invoke).mockRejectedValueOnce(new Error("sidecar not ready"));

    await expect(
      relayRequest("agent.chat", { input: "hello" }),
    ).rejects.toThrow("sidecar not ready");
  });

  it("handles empty params object", async () => {
    const mockResponse: JSONRPCResponse = {
      jsonrpc: "2.0",
      id: 1,
      result: { healthy: true },
    };

    vi.mocked(invoke).mockResolvedValueOnce(mockResponse);

    const result = await relayRequest("health.ping", {});

    const callArgs = vi.mocked(invoke).mock.calls[0];
    const payload = callArgs[1] as JSONRPCRequest;
    expect(payload.params).toEqual({});
    expect(result).toEqual(mockResponse);
  });
});

// ============================================================================
// streamChat tests
// ============================================================================

describe("streamChat", () => {
  it("yields StreamEvent tokens as they arrive from NDJSON chunks", async () => {
    // Simulate NDJSON chunk: multiple lines of events
    const chunk = [
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: "Hello", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: " world", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.done",
        params: { session_id: "abc" },
      }),
    ].join("\n");

    // Mock invoke to return the first chunk
    vi.mocked(invoke).mockResolvedValueOnce(chunk);

    const events: StreamEvent[] = [];
    for await (const event of streamChat("Hello")) {
      events.push(event);
    }

    expect(events).toHaveLength(3);

    expect(events[0]).toEqual({
      type: "token",
      data: "Hello",
    });

    expect(events[1]).toEqual({
      type: "token",
      data: " world",
    });

    expect(events[2]).toEqual({
      type: "done",
    });
  });

  it("yields tool_call events", async () => {
    const chunk = [
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.tool_call",
        params: {
          tool: "read_file",
          path: "/home/user/.config/hypr/hyprland.conf",
          session_id: "abc",
        },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.tool_result",
        params: {
          content: "file contents here...",
          session_id: "abc",
        },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.done",
        params: { session_id: "abc" },
      }),
    ].join("\n");

    vi.mocked(invoke).mockResolvedValueOnce(chunk);

    const events: StreamEvent[] = [];
    for await (const event of streamChat("read my hyprland config")) {
      events.push(event);
    }

    expect(events).toHaveLength(3);

    expect(events[0]).toEqual({
      type: "tool_call",
      data: { tool: "read_file", path: "/home/user/.config/hypr/hyprland.conf" },
    });

    expect(events[1]).toEqual({
      type: "tool_result",
      data: "file contents here...",
    });

    expect(events[2]).toEqual({ type: "done" });
  });

  it("yields error events on stream errors", async () => {
    const chunk = [
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: "I'll", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.error",
        params: { error: "Model timeout", session_id: "abc" },
      }),
    ].join("\n");

    vi.mocked(invoke).mockResolvedValueOnce(chunk);

    const events: StreamEvent[] = [];
    for await (const event of streamChat("hello")) {
      events.push(event);
    }

    expect(events).toHaveLength(2);
    expect(events[0]).toEqual({ type: "token", data: "I'll" });
    expect(events[1]).toEqual({
      type: "error",
      data: "Model timeout",
    });
  });

  it("throws when Tauri invoke itself fails", async () => {
    vi.mocked(invoke).mockRejectedValueOnce(
      new Error("sidecar not ready"),
    );

    await expect(async () => {
      for await (const _ of streamChat("hello")) {
        // should throw before yielding
      }
    }).rejects.toThrow("sidecar not ready");
  });

  it("handles empty stream (no tokens, immediate done)", async () => {
    const chunk = JSON.stringify({
      jsonrpc: "2.0",
      method: "stream.done",
      params: { session_id: "abc" },
    });

    vi.mocked(invoke).mockResolvedValueOnce(chunk);

    const events: StreamEvent[] = [];
    for await (const event of streamChat("empty query")) {
      events.push(event);
    }

    expect(events).toHaveLength(1);
    expect(events[0]).toEqual({ type: "done" });
  });
});
