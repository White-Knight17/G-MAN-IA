import "./app.css";
import App from "./App.svelte";
import { mount } from "svelte";

// Only mount on client — Vite may evaluate this in SSR context during dev
let app: ReturnType<typeof mount> | undefined;

if (!import.meta.env.SSR) {
  app = mount(App, {
    target: document.getElementById("app")!,
  });
}

export default app;
