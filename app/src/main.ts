// Entry point — only runs in browser context
// Dynamic imports prevent Vite SSR from evaluating mount()

async function bootstrap() {
  // Guard: only run in browser
  if (typeof window === "undefined") return;

  const { mount } = await import("svelte");
  const { default: App } = await import("./App.svelte");
  await import("./app.css");

  mount(App, {
    target: document.getElementById("app")!,
  });
}

bootstrap();
