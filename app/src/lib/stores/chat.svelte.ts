// G-MAN v1.0 — Chat Store (Svelte 5 runes)
// Manages message history, streaming state, and RPC integration

import { streamChat } from "../rpc";

// ── Types ──────────────────────────────────────────────────────────────────

export type MessageRole = "user" | "assistant" | "system" | "tool";

export type Message = {
  id: string;
  role: MessageRole;
  content: string;
  timestamp: number;
  streaming?: boolean;
};

// ── ID generator ───────────────────────────────────────────────────────────

let _msgSeq = 0;
function nextMsgId(): string {
  return `msg-${Date.now()}-${++_msgSeq}`;
}

// ── createChatStore ────────────────────────────────────────────────────────

export function createChatStore() {
  let messages = $state<Message[]>([]);
  let isThinking = $state(false);

  async function sendMessage(text: string) {
    // Add user message immediately
    const userMsg: Message = {
      id: nextMsgId(),
      role: "user",
      content: text,
      timestamp: Date.now(),
    };
    messages = [...messages, userMsg];

    // Create placeholder for assistant response
    const assistantMsg: Message = {
      id: nextMsgId(),
      role: "assistant",
      content: "",
      timestamp: Date.now(),
      streaming: true,
    };
    messages = [...messages, assistantMsg];

    isThinking = true;

    try {
      let accumulated = "";

      for await (const event of streamChat(text)) {
        switch (event.type) {
          case "token":
            accumulated += event.data as string;
            // Update assistant message in-place
            messages = messages.map((m) =>
              m.id === assistantMsg.id
                ? { ...m, content: accumulated }
                : m,
            );
            break;

          case "tool_call": {
            // Record tool call as a separate message
            const tc = event.data as {
              tool: string;
              path: string;
            };
            const toolMsg: Message = {
              id: nextMsgId(),
              role: "tool",
              content: `Calling ${tc.tool}: ${tc.path}`,
              timestamp: Date.now(),
            };
            messages = [...messages, toolMsg];
            break;
          }

          case "tool_result": {
            const content = event.data as string;
            // Append tool result to assistant message
            accumulated +=
              "\n\n" + (content.length > 300
                ? content.slice(0, 300) + "\n... (truncated)"
                : content);
            messages = messages.map((m) =>
              m.id === assistantMsg.id
                ? { ...m, content: accumulated }
                : m,
            );
            break;
          }

          case "error": {
            const errorText = event.data as string;
            accumulated += `\n\n⚠️ Error: ${errorText}`;
            messages = messages.map((m) =>
              m.id === assistantMsg.id
                ? { ...m, content: accumulated, streaming: false }
                : m,
            );
            break;
          }

          case "done":
            // Mark streaming complete
            messages = messages.map((m) =>
              m.id === assistantMsg.id
                ? { ...m, streaming: false }
                : m,
            );
            break;
        }
      }
    } catch (err) {
      // On error, mark streaming complete and show error in message
      const errorMsg =
        err instanceof Error ? err.message : String(err);
      messages = messages.map((m) =>
        m.id === assistantMsg.id
          ? {
              ...m,
              content:
                (m.content || "") +
                `\n\n❌ Connection error: ${errorMsg}`,
              streaming: false,
            }
          : m,
      );
      throw err; // Re-throw so callers can handle if needed
    } finally {
      isThinking = false;
    }
  }

  function clearMessages() {
    messages = [];
    isThinking = false;
  }

  return {
    get messages() {
      return messages;
    },
    get isThinking() {
      return isThinking;
    },
    sendMessage,
    clearMessages,
  };
}
