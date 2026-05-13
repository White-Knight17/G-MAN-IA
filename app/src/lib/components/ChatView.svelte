<script lang="ts">
  import type { Message } from "$lib/stores/chat.svelte";
  import { isSlashCommand, parseCommand } from "$lib/commands";

  interface Props {
    messages: Message[];
    isThinking: boolean;
    isProcessingCommand?: boolean;
    onsend?: (text: string) => void;
    oncommand?: (text: string) => void;
  }

  let { messages, isThinking, isProcessingCommand = false, onsend, oncommand }: Props = $props();

  let inputText = $state("");
  let showCommandPalette = $state(false);
  let messagesContainer: HTMLDivElement;

  // Auto-scroll when new messages arrive
  $effect(() => {
    // Track message count to trigger scroll
    const _count = messages.length;
    if (messagesContainer) {
      messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }
  });

  function handleSend() {
    const trimmed = inputText.trim();
    if (!trimmed || isThinking || isProcessingCommand) return;

    if (isSlashCommand(trimmed)) {
      oncommand?.(trimmed);
    } else {
      onsend?.(trimmed);
    }
    inputText = "";
    showCommandPalette = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  function handleInput() {
    // Show command palette when typing a slash command
    showCommandPalette = isSlashCommand(inputText.trim());
  }

  const availableCommands = [
    { name: "help", desc: "Show available commands" },
    { name: "clear", desc: "Clear chat history" },
    { name: "model", desc: "Show current model" },
    { name: "models", desc: "Pull a model from Ollama" },
  ];

  function getMatchingCommands(text: string) {
    const parsed = parseCommand(text);
    if (!parsed.cmd) return availableCommands;
    return availableCommands.filter((c) =>
      c.name.startsWith(parsed.cmd),
    );
  }

  function selectCommand(cmd: { name: string; desc: string }) {
    inputText = "/" + cmd.name + " ";
    showCommandPalette = false;
  }
</script>

<div class="chat-view">
  <!-- Message list -->
  <div class="messages" bind:this={messagesContainer}>
    {#if messages.length === 0}
      <div class="welcome">
        <h2>🤖 Hi! I'm G-MAN</h2>
        <p>Your Linux assistant. How can I help?</p>
      </div>
    {:else}
      {#each messages as msg (msg.id)}
        {@const isUser = msg.role === "user"}
        {@const isTool = msg.role === "tool"}
        {@const isCommandResult = msg.role === "command-result"}
        <div
          class="message-row"
          class:right={isUser || isTool}
          class:left={!isUser && !isTool}
          class:full-width={isCommandResult}
        >
          <div
            class="bubble"
            data-role={msg.role}
            data-typing={msg.streaming ? "true" : "false"}
            class:user-bubble={isUser}
            class:assistant-bubble={!isUser && !isTool && !isCommandResult}
            class:tool-bubble={isTool}
            class:command-bubble={isCommandResult}
          >
            {#if msg.content}
              <div class="message-content">
                {@html msg.content}
              </div>
            {/if}
            {#if msg.streaming && isThinking}
              <span class="typing-dots">
                <span class="dot">.</span>
                <span class="dot">.</span>
                <span class="dot">.</span>
              </span>
            {/if}
          </div>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Input bar -->
  <div class="input-bar">
    <div class="input-wrapper">
      <textarea
        placeholder="Ask G-MAN anything... or type / for commands"
        bind:value={inputText}
        onkeydown={handleKeydown}
        oninput={handleInput}
        disabled={isThinking || isProcessingCommand}
        rows="2"
      ></textarea>
      {#if showCommandPalette}
        <div class="command-palette">
          {#each getMatchingCommands(inputText) as cmd}
            <button
              class="command-item"
              onclick={() => selectCommand(cmd)}
            >
              <span class="command-name">/{cmd.name}</span>
              <span class="command-desc">{cmd.desc}</span>
            </button>
          {/each}
        </div>
      {/if}
    </div>
    <button
      onclick={handleSend}
      disabled={isThinking || isProcessingCommand || !inputText.trim()}
      aria-label="Send"
    >
      ▶
    </button>
  </div>
</div>

<style>
  .chat-view {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--gman-bg, #1a1b26);
    color: var(--gman-text, #c0caf5);
  }

  .messages {
    flex: 1;
    overflow-y: auto;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .welcome {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    text-align: center;
    opacity: 0.7;
    padding: 2rem;
  }

  .welcome h2 {
    font-size: 1.25rem;
    font-weight: 600;
    margin: 0 0 0.5rem 0;
  }

  .welcome p {
    font-size: 0.875rem;
    color: var(--gman-muted, #565f89);
  }

  .message-row {
    display: flex;
    max-width: 85%;
  }

  .message-row.right {
    align-self: flex-end;
    justify-content: flex-end;
  }

  .message-row.left {
    align-self: flex-start;
    justify-content: flex-start;
  }

  .bubble {
    padding: 0.625rem 0.875rem;
    border-radius: 0.75rem;
    word-break: break-word;
    line-height: 1.5;
    font-size: 0.875rem;
  }

  .user-bubble {
    background: var(--gman-accent, #3b82f6);
    color: #fff;
    border-bottom-right-radius: 0.25rem;
    box-shadow: var(--gman-elevation-1, 0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.24));
    transition: box-shadow 0.15s ease, transform 0.1s ease;
  }

  .user-bubble:hover {
    box-shadow: var(--gman-elevation-2, 0 3px 6px rgba(0, 0, 0, 0.16), 0 3px 6px rgba(0, 0, 0, 0.23));
  }

  .assistant-bubble {
    background: var(--gman-surface, #24283b);
    color: var(--gman-text, #c0caf5);
    border-bottom-left-radius: 0.25rem;
    box-shadow: var(--gman-elevation-1, 0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.24));
    transition: box-shadow 0.15s ease, transform 0.1s ease;
  }

  .assistant-bubble:hover {
    box-shadow: var(--gman-elevation-2, 0 3px 6px rgba(0, 0, 0, 0.16), 0 3px 6px rgba(0, 0, 0, 0.23));
  }

  .tool-bubble {
    background: transparent;
    border: 1px dashed var(--gman-muted, #565f89);
    color: var(--gman-muted, #565f89);
    font-size: 0.75rem;
    font-style: italic;
    padding: 0.375rem 0.75rem;
  }

  .command-bubble {
    background: var(--gman-surface, #1e2030);
    border: 1px solid var(--gman-border, #292e42);
    color: var(--gman-text, #c0caf5);
    font-family: "JetBrains Mono", "Fira Code", monospace;
    font-size: 0.8125rem;
    white-space: pre-wrap;
    max-width: 100%;
    box-shadow: var(--gman-elevation-2, 0 3px 6px rgba(0, 0, 0, 0.16), 0 3px 6px rgba(0, 0, 0, 0.23));
  }

  .message-row.full-width {
    max-width: 100%;
    align-self: stretch;
  }

  .typing-dots {
    display: inline-flex;
    gap: 0.15rem;
    margin-left: 0.25rem;
  }

  .typing-dots .dot {
    animation: blink 1.4s infinite both;
    font-weight: bold;
    font-size: 1.25rem;
    line-height: 1;
  }

  .typing-dots .dot:nth-child(2) {
    animation-delay: 0.2s;
  }

  .typing-dots .dot:nth-child(3) {
    animation-delay: 0.4s;
  }

  @keyframes blink {
    0%, 80%, 100% {
      opacity: 0;
    }
    40% {
      opacity: 1;
    }
  }

  .input-bar {
    display: flex;
    gap: 0.5rem;
    padding: 0.75rem;
    border-top: 1px solid var(--gman-border, #1e2030);
    background: var(--gman-surface, #24283b);
  }

  .input-bar textarea {
    flex: 1;
    background: var(--gman-input, #1a1b26);
    color: var(--gman-text, #c0caf5);
    border: 1px solid var(--gman-border, #1e2030);
    border-radius: 0.5rem;
    padding: 0.5rem 0.75rem;
    font-size: 0.8125rem;
    font-family: inherit;
    resize: none;
    outline: none;
  }

  .input-bar textarea:focus {
    border-color: var(--gman-accent, #3b82f6);
  }

  .input-bar textarea:disabled {
    opacity: 0.5;
  }

  .input-bar button {
    background: var(--gman-accent, #3b82f6);
    color: #fff;
    border: none;
    border-radius: 0.5rem;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    cursor: pointer;
    box-shadow: var(--gman-elevation-1, 0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.24));
    transition: background 0.15s ease, box-shadow 0.15s ease, transform 0.1s ease, filter 0.15s ease;
  }

  .input-bar button:hover:not(:disabled) {
    transform: scale(1.02);
    filter: brightness(1.1);
    box-shadow: var(--gman-elevation-2, 0 3px 6px rgba(0, 0, 0, 0.16), 0 3px 6px rgba(0, 0, 0, 0.23));
  }

  .input-bar button:active:not(:disabled) {
    transform: scale(0.98);
    filter: brightness(0.9);
    box-shadow: none;
  }

  .input-bar button:disabled {
    opacity: 0.4;
    cursor: not-allowed;
    box-shadow: none;
  }

  .input-wrapper {
    flex: 1;
    position: relative;
  }

  .input-wrapper textarea {
    width: 100%;
    box-sizing: border-box;
  }

  .command-palette {
    position: absolute;
    bottom: 100%;
    left: 0;
    right: 0;
    background: var(--gman-surface, #24283b);
    border: 1px solid var(--gman-border, #1e2030);
    border-radius: 0.5rem;
    margin-bottom: 0.25rem;
    max-height: 200px;
    overflow-y: auto;
    box-shadow: var(--gman-elevation-2, 0 3px 6px rgba(0, 0, 0, 0.16));
    z-index: 10;
  }

  .command-item {
    display: flex;
    flex-direction: column;
    width: 100%;
    padding: 0.5rem 0.75rem;
    background: transparent;
    border: none;
    color: var(--gman-text, #c0caf5);
    text-align: left;
    cursor: pointer;
    font-size: 0.8125rem;
  }

  .command-item:hover {
    background: var(--gman-accent, #3b82f6);
    color: #fff;
  }

  .command-name {
    font-weight: 600;
    font-family: "JetBrains Mono", "Fira Code", monospace;
  }

  .command-desc {
    font-size: 0.75rem;
    opacity: 0.7;
  }
</style>
