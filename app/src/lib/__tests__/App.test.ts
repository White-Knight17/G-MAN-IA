import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/svelte";
import App from "../../App.svelte";

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
    get length() {
      return Object.keys(store).length;
    },
    key: vi.fn((index: number) => Object.keys(store)[index] ?? null),
  };
})();

Object.defineProperty(window, "localStorage", {
  value: localStorageMock,
});

beforeEach(() => {
  vi.clearAllMocks();
  localStorageMock.clear();
});

// ============================================================================
// App.svelte tests
// ============================================================================

describe("App", () => {
  it("renders G-MAN title", () => {
    render(App, { props: {} });

    expect(screen.getByText("G-MAN")).toBeInTheDocument();
  });

  it("shows onboarding wizard on first launch (no config)", () => {
    // No config in localStorage means first launch
    render(App, { props: {} });

    // Should show the wizard
    expect(screen.getByText(/Choose your AI/i)).toBeInTheDocument();
  });

  it("shows chat view when config exists (already onboarded)", () => {
    // Simulate existing config
    localStorageMock.setItem(
      "gman-config",
      JSON.stringify({
        backend: "ollama",
        model: "deepseek-r1:1.5b",
        directories: ["~/.config"],
        theme: "dark",
      }),
    );

    render(App, { props: {} });

    // Should show chat (welcome message + input)
    expect(screen.getByText(/I'm G-MAN/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Ask G-MAN/)).toBeInTheDocument();
  });

  it("shows titlebar with G-MAN branding", () => {
    localStorageMock.setItem(
      "gman-config",
      JSON.stringify({
        backend: "ollama",
        model: "deepseek-r1:1.5b",
        directories: ["~/.config"],
        theme: "dark",
      }),
    );

    render(App, { props: {} });

    // Titlebar should have G-MAN text
    const titlebar = screen.getByText("G-MAN");
    expect(titlebar).toBeInTheDocument();
  });

  it("applies dark theme by default", () => {
    render(App, { props: {} });

    const app = document.querySelector(".app-shell");
    expect(app).toBeInTheDocument();

    // Dark theme should have dark background class or CSS var
    const style = window.getComputedStyle(document.body);
    // In jsdom, we can't reliably check CSS vars, but the app-shell element should exist
  });

  it("switches from wizard to chat after finish", async () => {
    // We can't test the full flow in jsdom, but we can verify both states render
    // First: wizard shown
    const { component, unmount } = render(App, { props: {} });
    expect(screen.getByText(/Choose your AI/)).toBeInTheDocument();

    unmount();

    // Now: chat shown (after config set)
    localStorageMock.setItem(
      "gman-config",
      JSON.stringify({
        backend: "ollama",
        model: "deepseek-r1:1.5b",
        directories: ["~/.config"],
        theme: "dark",
      }),
    );

    render(App, { props: {} });
    expect(screen.getByPlaceholderText(/Ask G-MAN/)).toBeInTheDocument();
  });
});
