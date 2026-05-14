import { defineConfig } from "vitest/config";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import path from "path";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte()],
  clearScreen: false,
  // Disable SSR entirely — this is a Tauri client-only app
  ssr: {
    noExternal: true,
  },
  server: {
    port: 1420,
    strictPort: true,
    watch: {
      ignored: ["**/src-tauri/**"],
    },
  },
  resolve: {
    conditions: process.env.VITEST ? ["browser"] : [],
    alias: {
      $lib: path.resolve("./src/lib"),
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    include: ["src/**/*.{test,spec}.{ts,svelte.ts}"],
    setupFiles: ["src/test-setup.ts"],
    css: false,
  },
});
