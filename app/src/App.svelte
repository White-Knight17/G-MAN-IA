<script lang="ts">
  import ChatView from "$lib/components/ChatView.svelte";
  import OnboardingWizard from "$lib/components/OnboardingWizard.svelte";
  import { createChatStore } from "$lib/stores/chat.svelte";
  import { listModels } from "$lib/rpc";

  // ── Initialization (synchronous, no lifecycle needed) ──────────────────

  const initialConfig = localStorage.getItem("gman-config");
  const initialTheme = initialConfig ? JSON.parse(initialConfig).theme : "dark";

  if (initialTheme === "dark") {
    document.documentElement.classList.add("theme-dark");
  } else if (initialTheme === "light") {
    document.documentElement.classList.add("theme-light");
  }

  // ── State ──────────────────────────────────────────────────────────────

  let chatStore = $state(createChatStore());
  let onboarded = $state(initialConfig !== null);
  let showWizard = $state(initialConfig === null);
  let ollamaReady = $state(false);

  // Auto-detect Ollama models on mount (async, non-blocking)
  (async () => {
    try {
      const models = await listModels();
      if (models.length > 0) {
        ollamaReady = true;
        chatStore.addCommandResult(
          `🦙 Ollama detected — ${models.length} model(s) available:\n` +
          models.map(m => `  • ${m.name} (${m.size})`).join("\n") +
          "\n\nType /model to switch, or /models <name> to download more."
        );
      }
    } catch {
      // Ollama not running or not installed — user will see connection error on first chat
    }
  })();

  // ── Handlers ───────────────────────────────────────────────────────────

  function handleWizardFinish(config: {
    backend: string;
    model: string;
    directories: string[];
    theme: string;
  }) {
    localStorage.setItem("gman-config", JSON.stringify(config));
    showWizard = false;
    onboarded = true;
    applyTheme(config.theme);
  }

  function applyTheme(theme: string) {
    const root = document.documentElement;
    root.classList.remove("theme-dark", "theme-light");
    if (theme === "dark") {
      root.classList.add("theme-dark");
    } else if (theme === "light") {
      root.classList.add("theme-light");
    }
  }

  async function handleSendMessage(text: string) {
    try {
      await chatStore.sendMessage(text);
    } catch (err) {
      // Error already handled by the store
    }
  }
</script>

<div class="app-shell">
  <!-- Titlebar -->
  <header class="titlebar">
    <div class="brand">
      <span class="logo">🤖</span>
      <span class="title">G-MAN</span>
    </div>
    <div class="window-controls">
      <button class="settings-btn" aria-label="Settings" onclick={() => showWizard = true}>
        ⚙
      </button>
      <button class="minimize-btn" aria-label="Minimize">─</button>
      <button class="close-btn" aria-label="Close">✕</button>
    </div>
  </header>

  <!-- Content -->
  <main class="content">
    {#if showWizard}
      <OnboardingWizard onfinish={handleWizardFinish} />
    {:else if onboarded}
      <ChatView
        messages={chatStore.messages}
        isThinking={chatStore.isThinking}
        isProcessingCommand={chatStore.isProcessingCommand}
        onsend={handleSendMessage}
        oncommand={(text) => chatStore.executeCommand(text)}
      />
    {:else}
      <div class="loading">
        <p>Loading...</p>
      </div>
    {/if}
  </main>
</div>

<style>
  :global(*) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(body) {
    font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI",
      Roboto, sans-serif;
    overflow: hidden;
    user-select: none;
  }

  :global(.theme-dark),
  :root {
    --gman-bg: #1a1b26;
    --gman-surface: #24283b;
    --gman-text: #c0caf5;
    --gman-muted: #565f89;
    --gman-accent: #3b82f6;
    --gman-border: #1e2030;
    --gman-input: #1a1b26;
  }

  :global(.theme-light) {
    --gman-bg: #f5f5f5;
    --gman-surface: #e8e8e8;
    --gman-text: #1a1b26;
    --gman-muted: #888;
    --gman-accent: #3b82f6;
    --gman-border: #ddd;
    --gman-input: #fff;
  }

  :global(body) {
    background: var(--gman-bg);
    color: var(--gman-text);
  }

  .app-shell {
    display: flex;
    flex-direction: column;
    height: 100vh;
    width: 100vw;
    overflow: hidden;
  }

  .titlebar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    height: 32px;
    padding: 0 0.75rem;
    background: var(--gman-surface);
    border-bottom: 1px solid var(--gman-border);
    -webkit-app-region: drag;
    flex-shrink: 0;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 0.375rem;
  }

  .logo {
    font-size: 0.875rem;
  }

  .title {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--gman-text);
    letter-spacing: 0.02em;
  }

  .window-controls {
    display: flex;
    gap: 0.25rem;
    -webkit-app-region: no-drag;
  }

  .window-controls button {
    background: transparent;
    border: none;
    color: var(--gman-muted);
    width: 28px;
    height: 22px;
    border-radius: 0.25rem;
    cursor: pointer;
    font-size: 0.75rem;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.15s ease, color 0.15s ease, transform 0.1s ease;
  }

  .window-controls button:hover {
    background: var(--gman-surface-hover, #2f3348);
    color: var(--gman-text);
    transform: scale(1.02);
  }

  .window-controls button:active {
    transform: scale(0.98);
    background: var(--gman-border, #1e2030);
  }

  .close-btn:hover {
    background: #ef4444 !important;
    color: #fff !important;
  }

  .settings-btn {
    font-size: 0.75rem;
    margin-right: 0.25rem;
  }

  .content {
    flex: 1;
    overflow: hidden;
  }

  .loading {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--gman-muted);
    font-size: 0.875rem;
  }
</style>
