<script lang="ts">
  // G-MAN v1.0 — Desktop GUI
  // Minimal skeleton for PR #2 (Tauri shell integration)
  // Full chat UI comes in PR #3
</script>

<main>
  <h1>G-MAN</h1>
  <p>Loading...</p>
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    background: #1a1b26;
    color: #c0caf5;
    font-family: system-ui, sans-serif;
  }

  main {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;
  }

  h1 {
    font-size: 1.5rem;
    font-weight: 700;
    margin: 0;
  }

  p {
    color: #565f89;
    margin-top: 0.5rem;
  }
</style>
