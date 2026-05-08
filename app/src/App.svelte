<script lang="ts">
  import ChatView from "$lib/components/ChatView.svelte";
  import OnboardingWizard from "$lib/components/OnboardingWizard.svelte";
  import PermissionDialog from "$lib/components/PermissionDialog.svelte";
  import { createChatStore } from "$lib/stores/chat.svelte";

  // ── State ──────────────────────────────────────────────────────────────

  let chatStore = $state(createChatStore());
  let onboarded = $state(false);
  let showWizard = $state(false);

  // Check for existing config on mount
  $effect(() => {
    const config = localStorage.getItem("gman-config");
    if (config) {
      onboarded = true;
    } else {
      showWizard = true;
    }
  });

  // ── Handlers ───────────────────────────────────────────────────────────

  function handleWizardFinish(config: {
    backend: string;
    model: string;
    directories: string[];
    theme: string;
  }) {
    // Save config to localStorage
    localStorage.setItem("gman-config", JSON.stringify(config));
    showWizard = false;
    onboarded = true;

    // Apply theme
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
    // "system" uses default CSS which respects system preference
  }

  async function handleSendMessage(text: string) {
    try {
      await chatStore.sendMessage(text);
    } catch (err) {
      // Error already handled by the store
    }
  }

  // Apply saved theme on mount
  $effect(() => {
    const config = localStorage.getItem("gman-config");
    if (config) {
      try {
        const parsed = JSON.parse(config);
        if (parsed.theme) {
          applyTheme(parsed.theme);
        }
      } catch {}
    } else {
      // Default dark theme
      document.documentElement.classList.add("theme-dark");
    }
  });
</script>

<div class="app-shell">
  <!-- Titlebar -->
  <header class="titlebar">
    <div class="brand">
      <span class="logo">🤖</span>
      <span class="title">G-MAN</span>
    </div>
    <div class="window-controls">
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
        onsend={handleSendMessage}
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
    transition: background 0.1s, color 0.1s;
  }

  .window-controls button:hover {
    background: var(--gman-border);
    color: var(--gman-text);
  }

  .close-btn:hover {
    background: #ef4444 !important;
    color: #fff !important;
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
