import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/svelte";
import ChatView from "../components/ChatView.svelte";
import type { Message } from "../stores/chat.svelte";
import { createChatStore } from "../stores/chat.svelte";

// ============================================================================
// ChatView component tests
// ============================================================================

describe("ChatView", () => {
  it("renders welcome message when no messages exist", () => {
    const store = createChatStore();

    render(ChatView, {
      props: {
        messages: store.messages,
        isThinking: store.isThinking,
      },
    });

    expect(screen.getByText(/Hi! I'm G-MAN/)).toBeInTheDocument();
    expect(screen.getByText(/Linux assistant/)).toBeInTheDocument();
  });

  it("renders user message as right-aligned bubble", () => {
    const messages: Message[] = [
      {
        id: "1",
        role: "user",
        content: "Hello G-MAN",
        timestamp: 1000,
      },
    ];

    render(ChatView, {
      props: { messages, isThinking: false },
    });

    expect(screen.getByText("Hello G-MAN")).toBeInTheDocument();
    // The user bubble should exist (role-based data attribute)
    const userBubble = document.querySelector('[data-role="user"]');
    expect(userBubble).toBeInTheDocument();
  });

  it("renders assistant message as left-aligned bubble", () => {
    const messages: Message[] = [
      {
        id: "1",
        role: "assistant",
        content: "Let me check that for you",
        timestamp: 1000,
      },
    ];

    render(ChatView, {
      props: { messages, isThinking: false },
    });

    expect(screen.getByText("Let me check that for you")).toBeInTheDocument();
    const assistantBubble = document.querySelector('[data-role="assistant"]');
    expect(assistantBubble).toBeInTheDocument();
  });

  it("shows typing indicator when isThinking is true", () => {
    const messages: Message[] = [
      {
        id: "1",
        role: "assistant",
        content: "",
        timestamp: 1000,
        streaming: true,
      },
    ];

    render(ChatView, {
      props: { messages, isThinking: true },
    });

    // Should have a loading/typing indicator
    const indicator = document.querySelector('[data-typing="true"]');
    expect(indicator).toBeInTheDocument();
  });

  it("hides typing indicator when isThinking is false", () => {
    const messages: Message[] = [
      {
        id: "1",
        role: "assistant",
        content: "Here's your answer",
        timestamp: 1000,
        streaming: false,
      },
    ];

    render(ChatView, {
      props: { messages, isThinking: false },
    });

    const indicator = document.querySelector('[data-typing="true"]');
    expect(indicator).toBeNull();
  });

  it("renders input bar with textarea and send button", () => {
    const store = createChatStore();

    render(ChatView, {
      props: {
        messages: store.messages,
        isThinking: store.isThinking,
      },
    });

    // Input area should exist
    const textarea = screen.getByPlaceholderText(/Ask G-MAN/);
    expect(textarea).toBeInTheDocument();

    const sendButton = screen.getByRole("button", { name: /send/i });
    expect(sendButton).toBeInTheDocument();
  });

  it("renders tool call messages with special styling", () => {
    const messages: Message[] = [
      {
        id: "1",
        role: "tool",
        content: "Reading file: /home/user/.config/hypr/hyprland.conf",
        timestamp: 1000,
      },
    ];

    render(ChatView, {
      props: { messages, isThinking: false },
    });

    const toolBubble = document.querySelector('[data-role="tool"]');
    expect(toolBubble).toBeInTheDocument();
    expect(
      screen.getByText(/Reading file/),
    ).toBeInTheDocument();
  });

  it("renders send button disabled when no text and not thinking", () => {
    const onSend = vi.fn();

    render(ChatView, {
      props: {
        messages: [],
        isThinking: false,
        onsend: onSend,
      },
    });

    const sendButton = screen.getByRole("button", { name: /send/i });
    expect(sendButton).toBeInTheDocument();
    expect(sendButton).toBeDisabled();
  });

  it("disables input when isThinking is true", () => {
    const messages: Message[] = [
      {
        id: "1",
        role: "assistant",
        content: "",
        timestamp: 1000,
        streaming: true,
      },
    ];

    render(ChatView, {
      props: { messages, isThinking: true },
    });

    const textarea = screen.getByPlaceholderText(/Ask G-MAN/);
    expect(textarea).toBeDisabled();
  });

  it("renders multiple messages in order", () => {
    const messages: Message[] = [
      {
        id: "1",
        role: "user",
        content: "First",
        timestamp: 1000,
      },
      {
        id: "2",
        role: "assistant",
        content: "Response 1",
        timestamp: 2000,
      },
      {
        id: "3",
        role: "user",
        content: "Second",
        timestamp: 3000,
      },
      {
        id: "4",
        role: "assistant",
        content: "Response 2",
        timestamp: 4000,
      },
    ];

    render(ChatView, {
      props: { messages, isThinking: false },
    });

    expect(screen.getByText("First")).toBeInTheDocument();
    expect(screen.getByText("Response 1")).toBeInTheDocument();
    expect(screen.getByText("Second")).toBeInTheDocument();
    expect(screen.getByText("Response 2")).toBeInTheDocument();
  });

  it("applies user-bubble class to user messages", () => {
    const messages: Message[] = [
      { id: "1", role: "user", content: "Hello", timestamp: 1000 },
    ];
    render(ChatView, { props: { messages, isThinking: false } });
    const bubble = document.querySelector(".user-bubble");
    expect(bubble).toBeInTheDocument();
    expect(bubble?.getAttribute("data-role")).toBe("user");
  });

  it("applies assistant-bubble class to assistant messages", () => {
    const messages: Message[] = [
      { id: "1", role: "assistant", content: "Hi there", timestamp: 1000 },
    ];
    render(ChatView, { props: { messages, isThinking: false } });
    const bubble = document.querySelector(".assistant-bubble");
    expect(bubble).toBeInTheDocument();
    expect(bubble?.getAttribute("data-role")).toBe("assistant");
  });

  it("applies command-bubble class to command-result messages", () => {
    const messages: Message[] = [
      { id: "1", role: "command-result", content: "**Help output**", timestamp: 1000 },
    ];
    render(ChatView, { props: { messages, isThinking: false } });
    const bubble = document.querySelector(".command-bubble");
    expect(bubble).toBeInTheDocument();
    expect(bubble?.getAttribute("data-role")).toBe("command-result");
  });

  it("renders command result as full-width", () => {
    const messages: Message[] = [
      { id: "1", role: "command-result", content: "Result", timestamp: 1000 },
    ];
    render(ChatView, { props: { messages, isThinking: false } });
    const row = document.querySelector(".full-width");
    expect(row).toBeInTheDocument();
  });
});
