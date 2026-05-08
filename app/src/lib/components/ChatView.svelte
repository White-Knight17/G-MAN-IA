<script lang="ts">
  import type { Message } from "$lib/stores/chat.svelte";

  interface Props {
    messages: Message[];
    isThinking: boolean;
    onsend?: (text: string) => void;
  }

  let { messages, isThinking, onsend }: Props = $props();

  let inputText = $state("");
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
    if (!trimmed || isThinking) return;
    onsend?.(trimmed);
    inputText = "";
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
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
        <div
          class="message-row"
          class:right={isUser || isTool}
          class:left={!isUser && !isTool}
        >
          <div
            class="bubble"
            data-role={msg.role}
            data-typing={msg.streaming ? "true" : "false"}
            class:user-bubble={isUser}
            class:assistant-bubble={!isUser && !isTool}
            class:tool-bubble={isTool}
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
    <textarea
      placeholder="Ask G-MAN anything..."
      bind:value={inputText}
      onkeydown={handleKeydown}
      disabled={isThinking}
      rows="2"
    ></textarea>
    <button
      onclick={handleSend}
      disabled={isThinking || !inputText.trim()}
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
  }

  .assistant-bubble {
    background: var(--gman-surface, #24283b);
    color: var(--gman-text, #c0caf5);
    border-bottom-left-radius: 0.25rem;
  }

  .tool-bubble {
    background: transparent;
    border: 1px dashed var(--gman-muted, #565f89);
    color: var(--gman-muted, #565f89);
    font-size: 0.75rem;
    font-style: italic;
    padding: 0.375rem 0.75rem;
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
    transition: opacity 0.15s;
  }

  .input-bar button:hover:not(:disabled) {
    opacity: 0.9;
  }

  .input-bar button:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
</style>
