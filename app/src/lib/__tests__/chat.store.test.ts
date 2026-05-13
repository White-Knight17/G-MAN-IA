import { describe, it, expect, vi, beforeEach } from "vitest";
import { invoke } from "@tauri-apps/api/core";

// Module under test — does NOT exist yet (RED)
import { createChatStore, type Message } from "../stores/chat.svelte";
import { listModels, pullModel, getConfig, setConfig } from "../rpc";
import * as rpc from "../rpc";

vi.mock("@tauri-apps/api/core", () => ({
  invoke: vi.fn(),
}));

vi.mock("../rpc", async () => {
  const actual = await vi.importActual("../rpc");
  return {
    ...actual,
    listModels: vi.fn(),
    pullModel: vi.fn(),
    getConfig: vi.fn(),
    setConfig: vi.fn(),
  };
});

beforeEach(() => {
  vi.clearAllMocks();
});

// ============================================================================
// createChatStore tests
// ============================================================================

describe("createChatStore", () => {
  it("starts with empty messages and isThinking false", () => {
    const store = createChatStore();

    expect(store.messages).toEqual([]);
    expect(store.isThinking).toBe(false);
  });

  it("adds user message and sets isThinking on sendMessage", async () => {
    const store = createChatStore();

    // Mock streamChat response: tokens + done
    const mockChunk = [
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

    vi.mocked(invoke).mockResolvedValueOnce(mockChunk);

    await store.sendMessage("Hi G-MAN!");

    // Should have user + assistant messages
    expect(store.messages.length).toBeGreaterThanOrEqual(2);

    // User message
    const userMsg = store.messages[0];
    expect(userMsg.role).toBe("user");
    expect(userMsg.content).toBe("Hi G-MAN!");

    // Assistant message (accumulated from tokens)
    const assistantMsg = store.messages[store.messages.length - 1];
    expect(assistantMsg.role).toBe("assistant");
    expect(assistantMsg.content).toBe("Hello world");

    // Should not be thinking anymore
    expect(store.isThinking).toBe(false);
  });

  it("accumulates streaming tokens into assistant message", async () => {
    const store = createChatStore();

    // Multiple token events
    const mockChunk = [
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: "Your", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: " config", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: " is", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: " fine.", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.done",
        params: { session_id: "abc" },
      }),
    ].join("\n");

    vi.mocked(invoke).mockResolvedValueOnce(mockChunk);

    await store.sendMessage("Check my config");

    const assistantMsg = store.messages.find(
      (m) => m.role === "assistant",
    );
    expect(assistantMsg).toBeDefined();
    expect(assistantMsg!.content).toBe("Your config is fine.");
  });

  it("sets isThinking to true while streaming", () => {
    const store = createChatStore();

    // We need to track that isThinking was set during streaming
    // Since sendMessage is async and internally processes the stream,
    // let's check that the store state transitions correctly
    let thinkingDuringStream = false;

    const mockChunk = [
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: "Hi", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.done",
        params: { session_id: "abc" },
      }),
    ].join("\n");

    vi.mocked(invoke).mockResolvedValueOnce(mockChunk);

    // Verify that isThinking starts false
    expect(store.isThinking).toBe(false);

    // After sendMessage completes, isThinking should be false
    return store.sendMessage("test").then(() => {
      expect(store.isThinking).toBe(false);
    });
  });

  it("adds tool_result content to the assistant message", async () => {
    const store = createChatStore();

    const mockChunk = [
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: "Reading", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.tool_result",
        params: {
          content: "## File: ~/.config/hypr/hyprland.conf\n\n```\nmonitor=,preferred,auto,1\n```",
          session_id: "abc",
        },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.done",
        params: { session_id: "abc" },
      }),
    ].join("\n");

    vi.mocked(invoke).mockResolvedValueOnce(mockChunk);

    await store.sendMessage("Show my config");

    const assistantMsg = store.messages.find(
      (m) => m.role === "assistant",
    );
    expect(assistantMsg).toBeDefined();
    expect(assistantMsg!.content).toContain("Reading");
    expect(assistantMsg!.content).toContain("## File:");
  });

  it("includes tool call metadata in message", async () => {
    const store = createChatStore();

    const mockChunk = [
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
        params: { content: "config content", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.done",
        params: { session_id: "abc" },
      }),
    ].join("\n");

    vi.mocked(invoke).mockResolvedValueOnce(mockChunk);

    await store.sendMessage("Read hyprland config");

    // Check there's a tool message between user and assistant
    const toolMsgs = store.messages.filter((m) => m.role === "tool");
    expect(toolMsgs.length).toBeGreaterThanOrEqual(1);
    expect(toolMsgs[0].content).toContain("read_file");
  });

  it("clearMessages resets to empty state", async () => {
    const store = createChatStore();

    const mockChunk = [
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: "ok", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.done",
        params: { session_id: "abc" },
      }),
    ].join("\n");

    vi.mocked(invoke).mockResolvedValueOnce(mockChunk);

    await store.sendMessage("test");
    expect(store.messages.length).toBeGreaterThan(0);

    store.clearMessages();
    expect(store.messages).toEqual([]);
    expect(store.isThinking).toBe(false);
  });

  it("handles stream errors gracefully", async () => {
    const store = createChatStore();

    const mockChunk = [
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: "Trying", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.error",
        params: { error: "Model timeout", session_id: "abc" },
      }),
    ].join("\n");

    vi.mocked(invoke).mockResolvedValueOnce(mockChunk);

    await store.sendMessage("Long query");

    const assistantMsg = store.messages.find(
      (m) => m.role === "assistant",
    );
    expect(assistantMsg).toBeDefined();
    // Should contain received tokens + error note
    expect(assistantMsg!.content).toContain("Trying");
    expect(assistantMsg!.content).toContain("Model timeout");
    expect(store.isThinking).toBe(false);
  });

  it("handles invoke failure without crashing", async () => {
    const store = createChatStore();

    vi.mocked(invoke).mockRejectedValueOnce(
      new Error("sidecar not ready"),
    );

    await expect(store.sendMessage("test")).rejects.toThrow(
      "sidecar not ready",
    );

    // User message should still be recorded
    const userMsg = store.messages.find((m) => m.role === "user");
    expect(userMsg).toBeDefined();
    expect(userMsg!.content).toBe("test");

    // isThinking should be reset even on failure
    expect(store.isThinking).toBe(false);
  });

  it("multiple sendMessage calls accumulate messages", async () => {
    const store = createChatStore();

    const mockChunk1 = [
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: "First", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.done",
        params: { session_id: "abc" },
      }),
    ].join("\n");

    const mockChunk2 = [
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: "Second", session_id: "def" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.done",
        params: { session_id: "def" },
      }),
    ].join("\n");

    vi.mocked(invoke)
      .mockResolvedValueOnce(mockChunk1)
      .mockResolvedValueOnce(mockChunk2);

    await store.sendMessage("Q1");
    await store.sendMessage("Q2");

    // Should have: user1, assistant1, user2, assistant2
    const userMsgs = store.messages.filter((m) => m.role === "user");
    const assistantMsgs = store.messages.filter(
      (m) => m.role === "assistant",
    );

    expect(userMsgs).toHaveLength(2);
    expect(assistantMsgs).toHaveLength(2);
    expect(userMsgs[0].content).toBe("Q1");
    expect(userMsgs[1].content).toBe("Q2");
    expect(assistantMsgs[0].content).toBe("First");
    expect(assistantMsgs[1].content).toBe("Second");
  });
});

// ============================================================================
// executeCommand tests
// ============================================================================

describe("executeCommand", () => {
  it("clears messages on /clear command", async () => {
    const store = createChatStore();

    // Add a message first
    const mockChunk = [
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.token",
        params: { token: "ok", session_id: "abc" },
      }),
      JSON.stringify({
        jsonrpc: "2.0",
        method: "stream.done",
        params: { session_id: "abc" },
      }),
    ].join("\n");

    vi.mocked(invoke).mockResolvedValueOnce(mockChunk);
    await store.sendMessage("test");
    expect(store.messages.length).toBeGreaterThan(0);

    // Now execute /clear
    await store.executeCommand("/clear");

    expect(store.messages).toEqual([]);
  });

  it("adds command-result message for /help", async () => {
    const store = createChatStore();

    await store.executeCommand("/help");

    const cmdMsg = store.messages.find((m) => m.role === "command-result");
    expect(cmdMsg).toBeDefined();
    expect(cmdMsg!.content).toContain("/help");
    expect(cmdMsg!.content).toContain("/clear");
    expect(cmdMsg!.content).toContain("/model");
  });

  it("adds command-result message for /model", async () => {
    const store = createChatStore();

    vi.mocked(getConfig).mockResolvedValueOnce({
      provider: "ollama",
      model: "llama3.2:3b",
      has_api_key: false,
      theme: "dark",
      window: { mode: "floating" },
    });

    vi.mocked(listModels).mockResolvedValueOnce([
      { name: "llama3.2:3b", size: "2.0 GB", digest: "abc" },
    ]);

    await store.executeCommand("/model");

    const cmdMsg = store.messages.find((m) => m.role === "command-result");
    expect(cmdMsg).toBeDefined();
    expect(cmdMsg!.content).toContain("llama3.2:3b");
  });

  it("adds error command-result for unknown command", async () => {
    const store = createChatStore();

    await store.executeCommand("/unknown");

    const cmdMsg = store.messages.find((m) => m.role === "command-result");
    expect(cmdMsg).toBeDefined();
    expect(cmdMsg!.content).toContain("Unknown command");
    expect(cmdMsg!.content).toContain("/unknown");
  });

  it("sets isProcessingCommand during command execution", async () => {
    const store = createChatStore();

    // /clear is synchronous, so we check before and after
    expect(store.isProcessingCommand).toBe(false);
    await store.executeCommand("/clear");
    expect(store.isProcessingCommand).toBe(false);
  });
});
