// G-MAN v1.0 — Chat Store (Svelte 5 runes)
// Manages message history, streaming state, and RPC integration

import { streamChat, listModels, pullModel, getConfig, setConfig } from "../rpc";

// ── Types ──────────────────────────────────────────────────────────────────

export type MessageRole = "user" | "assistant" | "system" | "tool" | "command-result";

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
  let isProcessingCommand = $state(false);

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

  function addCommandResult(content: string, success: boolean = true) {
    const cmdMsg: Message = {
      id: nextMsgId(),
      role: "command-result",
      content,
      timestamp: Date.now(),
    };
    messages = [...messages, cmdMsg];
  }

  async function executeCommand(text: string) {
    const trimmed = text.trim();
    if (!trimmed.startsWith("/")) return;

    isProcessingCommand = true;
    try {
      const parts = trimmed.slice(1).split(/\s+/);
      const cmd = parts[0].toLowerCase();
      const args = parts.slice(1);

      switch (cmd) {
        case "clear":
          clearMessages();
          break;

        case "help":
          addCommandResult(formatHelp());
          break;

        case "model":
          await executeModelCommand(args);
          break;

        case "models":
          if (args.length === 0) {
            addCommandResult("Usage: /models <name>\nExample: /models qwen2.5:3b", false);
          } else {
            await pullModelCommand(args[0]);
          }
          break;

        case "api":
          if (args.length < 2) {
            addCommandResult("Usage: /api <provider> <key>\nExample: /api openai sk-xxx...", false);
          } else {
            await setApiKeyCommand(args[0], args.slice(1).join(" "));
          }
          break;

        default:
          addCommandResult(`Unknown command: /${cmd}\nType /help for available commands.`, false);
          break;
      }
    } finally {
      isProcessingCommand = false;
    }
  }

  async function pullModelCommand(name: string) {
    addCommandResult(`⬇️ Pulling **${name}** from Ollama...`);
    try {
      const result = await pullModel(name);
      addCommandResult(`✅ **${name}** downloaded successfully.\nRun /model to see available models.`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      addCommandResult(`❌ Failed to pull ${name}: ${msg}`, false);
    }
  }

  async function setApiKeyCommand(provider: string, key: string) {
    if (!provider || !key) {
      addCommandResult(`❌ Usage: /api <provider> <key>\nExample: /api openai sk-xxx...`, false);
      return;
    }
    try {
      const params: Record<string, string> = { provider: provider.toLowerCase(), api_key: key };
      await setConfig(params);
      addCommandResult(`✅ API key set for **${provider}**.\nRun /model to see current configuration.\n\nTip: G-MAN auto-detects base URLs for openai, deepseek, groq.\nUse /config for advanced settings.`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      addCommandResult(`❌ Failed to set API key: ${msg}`, false);
    }
  }

  async function executeModelCommand(args: string[]) {
    try {
      const config = await getConfig();
      const isLocal = config.provider === "ollama" || !config.provider;
      let output = `**Provider**: ${config.provider || "ollama"}\n`;
      output += `**Model**: ${config.model || (isLocal ? "llama3.2:3b" : "gpt-4o")}\n`;
      output += `**API Key**: ${config.has_api_key ? "🔑 Set" : "❌ Not set"}\n\n`;

      if (isLocal) {
        try {
          const models = await listModels();
          if (models.length > 0) {
            output += "Local Ollama models:\n";
            for (const m of models) {
              const active = m.name === config.model ? " (active)" : "";
              output += `  • ${m.name} — ${m.size}${active}\n`;
            }
          } else {
            output += "No local models found. Run /models <name> to pull one.";
          }
        } catch {
          output += "⚠️ Ollama not running. Start it with: systemctl --user start ollama";
        }
      } else {
        output += "Using remote API. Supported providers: openai, deepseek, groq\n";
        output += "Run /api <provider> <key> to switch provider.";
      }

      addCommandResult(output);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      addCommandResult(`Error fetching models: ${msg}`, false);
    }
  }

  function formatHelp(): string {
    return [
      "**Available Commands:**",
      "",
      "/help — Show this help message",
      "/clear — Clear chat history",
      "/model — Show current model and available models",
      "/models <name> — Pull a model from Ollama",
      "/api <provider> <key> — Set remote API key (openai, deepseek, groq, etc.)",
      "",
      "Type a message (without /) to chat with G-MAN.",
    ].join("\n");
  }

  return {
    get messages() {
      return messages;
    },
    get isThinking() {
      return isThinking;
    },
    get isProcessingCommand() {
      return isProcessingCommand;
    },
    sendMessage,
    clearMessages,
    executeCommand,
    addCommandResult,
  };
}
