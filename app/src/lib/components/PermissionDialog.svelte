<script lang="ts">
  interface Props {
    tool: string;
    path: string;
    mode: "ro" | "rw";
    onallow: () => void;
    ondeny: () => void;
  }

  let { tool, path, mode, onallow, ondeny }: Props = $props();

  let isRead = $derived(mode === "ro");
  let action = $derived(isRead ? "read" : "write");
</script>

<div class="overlay" role="dialog" aria-modal="true">
  <div class="dialog">
    <div class="icon">
      {#if isRead}
        👁️
      {:else}
        ✏️
      {/if}
    </div>

    <h3>Allow G-MAN to {action} this file?</h3>

    <div class="details">
      <span class="tool-name">{tool}</span>
      <code class="path">{path}</code>
      <span class="mode-label">{isRead ? "read-only" : "read/write"}</span>
    </div>

    <div class="actions">
      <button class="deny-btn" onclick={ondeny}>
        Deny
      </button>
      <button class="allow-btn" onclick={onallow}>
        Allow
      </button>
    </div>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
    animation: fadeIn 0.2s ease;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .dialog {
    background: var(--gman-surface, #24283b);
    border: 1px solid var(--gman-border, #1e2030);
    border-radius: 1rem;
    padding: 1.5rem;
    max-width: 24rem;
    width: 90%;
    text-align: center;
    animation: scaleIn 0.2s ease;
  }

  @keyframes scaleIn {
    from { transform: scale(0.95); opacity: 0; }
    to { transform: scale(1); opacity: 1; }
  }

  .dialog h3 {
    margin: 0.75rem 0 0.5rem 0;
    font-size: 1rem;
    font-weight: 600;
    color: var(--gman-text, #c0caf5);
  }

  .icon {
    font-size: 2rem;
  }

  .details {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
    margin: 1rem 0;
  }

  .tool-name {
    font-size: 0.75rem;
    color: var(--gman-accent, #3b82f6);
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .path {
    font-size: 0.75rem;
    color: var(--gman-muted, #565f89);
    background: var(--gman-bg, #1a1b26);
    padding: 0.375rem 0.5rem;
    border-radius: 0.375rem;
    word-break: break-all;
  }

  .mode-label {
    font-size: 0.75rem;
    color: var(--gman-muted, #565f89);
  }

  .actions {
    display: flex;
    gap: 0.75rem;
    justify-content: center;
    margin-top: 1.25rem;
  }

  .actions button {
    padding: 0.5rem 1.5rem;
    border-radius: 0.5rem;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    border: none;
    transition: opacity 0.15s;
  }

  .deny-btn {
    background: var(--gman-bg, #1a1b26);
    color: #f87171;
    border: 1px solid #f87171 !important;
  }

  .deny-btn:hover {
    background: rgba(248, 113, 113, 0.1);
  }

  .allow-btn {
    background: #22c55e;
    color: #fff;
  }

  .allow-btn:hover {
    opacity: 0.9;
  }
</style>
