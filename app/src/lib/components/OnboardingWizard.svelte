<script lang="ts">
  interface Props {
    onfinish: (config: {
      backend: string;
      model: string;
      directories: string[];
      theme: string;
    }) => void;
  }

  let { onfinish }: Props = $props();

  let currentStep = $state(0); // 0-indexed: step 1, 2, 3
  const totalSteps = 3;

  // Step 1: AI backend
  let selectedBackend = $state<"ollama" | "api">("ollama");
  let apiKey = $state("");
  let apiEndpoint = $state("");
  let modelName = $state("deepseek-r1:1.5b");

  // Step 2: Allowed directories
  let directories = $state([
    { path: "~/.config", label: ".config (Hyprland, waybar, etc.)", checked: true },
    { path: "~/.local", label: ".local (user applications)", checked: true },
    { path: "", label: "Custom path...", checked: false, custom: true },
  ]);

  // Step 3: Theme
  let selectedTheme = $state<"dark" | "light" | "system">("dark");

  const stepTitles = [
    "Choose your AI",
    "Allowed directories",
    "Theme",
  ];

  function nextStep() {
    if (currentStep < totalSteps - 1) {
      currentStep++;
    }
  }

  function prevStep() {
    if (currentStep > 0) {
      currentStep--;
    }
  }

  function handleSkip() {
    // Use defaults
    onfinish({
      backend: "ollama",
      model: "deepseek-r1:1.5b",
      directories: ["~/.config", "~/.local"],
      theme: "dark",
    });
  }

  function handleFinish() {
    const dirs = directories
      .filter((d) => d.checked || (d.custom && d.path.trim()))
      .map((d) => (d.custom ? d.path.trim() : d.path))
      .filter(Boolean);

    onfinish({
      backend: selectedBackend === "api" ? "api" : "ollama",
      model: selectedBackend === "api" ? modelName || "gpt-4" : modelName || "deepseek-r1:1.5b",
      directories: dirs.length > 0 ? dirs : ["~/.config", "~/.local"],
      theme: selectedTheme,
    });
  }
</script>

<div class="wizard">
  <!-- Progress bar -->
  <div class="progress">
    <div class="progress-bar" style="width: {((currentStep + 1) / totalSteps) * 100}%"></div>
  </div>
  <div class="progress-text">
    Step {currentStep + 1} of {totalSteps}
  </div>

  <!-- Step content -->
  <div class="step-content">
    <h2>{stepTitles[currentStep]}</h2>

    {#if currentStep === 0}
      <!-- Step 1: Choose AI -->
      <div class="options-group">
        <label class="option-card" class:selected={selectedBackend === "ollama"}>
          <input
            type="radio"
            bind:group={selectedBackend}
            value="ollama"
          />
          <div class="option-info">
            <strong>🦙 Ollama (Local)</strong>
            <span>Run models locally on your machine</span>
          </div>
        </label>

        <label class="option-card" class:selected={selectedBackend === "api"}>
          <input
            type="radio"
            bind:group={selectedBackend}
            value="api"
          />
          <div class="option-info">
            <strong>🔑 API Key</strong>
            <span>Connect to external API provider</span>
          </div>
        </label>
      </div>

      {#if selectedBackend === "api"}
        <div class="api-fields">
          <input
            type="text"
            placeholder="API Key"
            bind:value={apiKey}
            class="input-field"
          />
          <input
            type="text"
            placeholder="Endpoint URL (optional)"
            bind:value={apiEndpoint}
            class="input-field"
          />
          <input
            type="text"
            placeholder="Model name"
            bind:value={modelName}
            class="input-field"
          />
        </div>
      {:else}
        <div class="api-fields">
          <input
            type="text"
            placeholder="Model name (e.g., deepseek-r1:1.5b)"
            bind:value={modelName}
            class="input-field"
          />
        </div>
      {/if}

    {:else if currentStep === 1}
      <!-- Step 2: Allowed directories -->
      <div class="options-group">
        {#each directories as dir}
          <label class="option-card" class:selected={dir.checked}>
            <input
              type="checkbox"
              bind:checked={dir.checked}
            />
            <div class="option-info">
              <strong>{dir.path || "Custom path"}</strong>
              <span>{dir.label}</span>
            </div>
          </label>
          {#if dir.custom && dir.checked}
            <input
              type="text"
              placeholder="/home/user/my-project"
              bind:value={dir.path}
              class="input-field"
            />
          {/if}
        {/each}
      </div>

    {:else if currentStep === 2}
      <!-- Step 3: Theme -->
      <div class="options-group theme-group">
        <label class="option-card theme-card" class:selected={selectedTheme === "dark"}>
          <input type="radio" bind:group={selectedTheme} value="dark" />
          <div class="theme-preview dark-preview">
            <span>🌙 Dark</span>
          </div>
        </label>

        <label class="option-card theme-card" class:selected={selectedTheme === "light"}>
          <input type="radio" bind:group={selectedTheme} value="light" />
          <div class="theme-preview light-preview">
            <span>☀️ Light</span>
          </div>
        </label>

        <label class="option-card theme-card" class:selected={selectedTheme === "system"}>
          <input type="radio" bind:group={selectedTheme} value="system" />
          <div class="theme-preview system-preview">
            <span>💻 System</span>
          </div>
        </label>
      </div>
    {/if}
  </div>

  <!-- Navigation -->
  <div class="nav">
    <button class="skip-btn" onclick={handleSkip}>
      Skip
    </button>
    <div class="nav-right">
      {#if currentStep > 0}
        <button class="back-btn" onclick={prevStep}>
          Back
        </button>
      {/if}
      {#if currentStep < totalSteps - 1}
        <button class="next-btn" onclick={nextStep}>
          Next
        </button>
      {:else}
        <button class="finish-btn" onclick={handleFinish}>
          Finish
        </button>
      {/if}
    </div>
  </div>
</div>

<style>
  .wizard {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--gman-bg, #1a1b26);
    color: var(--gman-text, #c0caf5);
    padding: 1.5rem;
  }

  .progress {
    height: 3px;
    background: var(--gman-surface, #24283b);
    border-radius: 3px;
    margin-bottom: 0.375rem;
    overflow: hidden;
  }

  .progress-bar {
    height: 100%;
    background: var(--gman-accent, #3b82f6);
    transition: width 0.3s ease;
    border-radius: 3px;
  }

  .progress-text {
    font-size: 0.75rem;
    color: var(--gman-muted, #565f89);
    margin-bottom: 1.5rem;
  }

  .step-content {
    flex: 1;
    overflow-y: auto;
  }

  .step-content h2 {
    font-size: 1.125rem;
    font-weight: 600;
    margin: 0 0 1rem 0;
  }

  .options-group {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .option-card {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    padding: 0.75rem;
    border: 1px solid var(--gman-border, #1e2030);
    border-radius: 0.625rem;
    cursor: pointer;
    transition: border-color 0.15s, background 0.15s;
  }

  .option-card:hover {
    border-color: var(--gman-accent, #3b82f6);
  }

  .option-card.selected {
    border-color: var(--gman-accent, #3b82f6);
    background: rgba(59, 130, 246, 0.08);
  }

  .option-info {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .option-info strong {
    font-size: 0.875rem;
    font-weight: 500;
  }

  .option-info span {
    font-size: 0.75rem;
    color: var(--gman-muted, #565f89);
  }

  .api-fields {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-top: 0.75rem;
  }

  .input-field {
    width: 100%;
    background: var(--gman-surface, #24283b);
    color: var(--gman-text, #c0caf5);
    border: 1px solid var(--gman-border, #1e2030);
    border-radius: 0.5rem;
    padding: 0.5rem 0.75rem;
    font-size: 0.8125rem;
    font-family: inherit;
    outline: none;
    box-sizing: border-box;
  }

  .input-field:focus {
    border-color: var(--gman-accent, #3b82f6);
  }

  .theme-group {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .theme-card {
    align-items: center;
  }

  .theme-preview {
    padding: 0.75rem;
    border-radius: 0.5rem;
    font-size: 0.875rem;
    font-weight: 500;
  }

  .dark-preview {
    background: #1a1b26;
    color: #c0caf5;
  }

  .light-preview {
    background: #f5f5f5;
    color: #1a1b26;
  }

  .system-preview {
    background: linear-gradient(135deg, #1a1b26 50%, #f5f5f5 50%);
    color: #c0caf5;
  }

  .nav {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 1.5rem;
    padding-top: 1rem;
    border-top: 1px solid var(--gman-border, #1e2030);
  }

  .nav-right {
    display: flex;
    gap: 0.5rem;
  }

  .nav button {
    padding: 0.5rem 1rem;
    border-radius: 0.5rem;
    font-size: 0.8125rem;
    font-weight: 500;
    cursor: pointer;
    border: none;
    transition: opacity 0.15s;
  }

  .skip-btn {
    background: transparent;
    color: var(--gman-muted, #565f89);
  }

  .skip-btn:hover {
    color: var(--gman-text, #c0caf5);
  }

  .back-btn {
    background: transparent;
    color: var(--gman-muted, #565f89);
    border: 1px solid var(--gman-border, #1e2030) !important;
  }

  .back-btn:hover {
    color: var(--gman-text, #c0caf5);
    border-color: var(--gman-muted, #565f89) !important;
  }

  .next-btn, .finish-btn {
    background: var(--gman-accent, #3b82f6);
    color: #fff;
  }

  .next-btn:hover, .finish-btn:hover {
    opacity: 0.9;
  }
</style>
