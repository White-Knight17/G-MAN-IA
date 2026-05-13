import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  needsMigration,
  parseLocalStorageConfig,
  buildSetConfigParams,
  runMigration,
} from "../migration";

// ============================================================================
// Migration: needsMigration
// ============================================================================

describe("needsMigration", () => {
  it("returns true when config exists and not yet migrated", () => {
    const getItem = vi.fn((key: string) => {
      if (key === "gman-config") return '{"backend":"ollama"}';
      return null;
    });
    expect(needsMigration(getItem)).toBe(true);
  });

  it("returns false when config does not exist", () => {
    const getItem = vi.fn(() => null);
    expect(needsMigration(getItem)).toBe(false);
  });

  it("returns false when already migrated", () => {
    const getItem = vi.fn((key: string) => {
      if (key === "gman-config") return '{"backend":"ollama"}';
      if (key === "gman-migrated") return "true";
      return null;
    });
    expect(needsMigration(getItem)).toBe(false);
  });
});

// ============================================================================
// Migration: parseLocalStorageConfig
// ============================================================================

describe("parseLocalStorageConfig", () => {
  it("parses valid config JSON", () => {
    const raw = JSON.stringify({
      backend: "ollama",
      model: "llama3.2:3b",
      directories: ["~/.config"],
      theme: "dark",
    });
    const result = parseLocalStorageConfig(raw);
    expect(result).toEqual({
      backend: "ollama",
      model: "llama3.2:3b",
      directories: ["~/.config"],
      theme: "dark",
    });
  });

  it("returns null for invalid JSON", () => {
    expect(parseLocalStorageConfig("not json")).toBeNull();
  });

  it("returns null for null input", () => {
    expect(parseLocalStorageConfig(null)).toBeNull();
  });

  it("uses defaults for missing fields", () => {
    const raw = JSON.stringify({});
    const result = parseLocalStorageConfig(raw);
    expect(result).toEqual({
      backend: "ollama",
      model: "",
      directories: [],
      theme: "dark",
    });
  });
});

// ============================================================================
// Migration: buildSetConfigParams
// ============================================================================

describe("buildSetConfigParams", () => {
  it("maps localStorage config to RPC params", () => {
    const config = {
      backend: "ollama",
      model: "llama3.2:3b",
      directories: ["~/.config"],
      theme: "light",
    };
    const params = buildSetConfigParams(config);
    expect(params).toEqual({
      provider: "ollama",
      model: "llama3.2:3b",
      theme: "light",
      window: { mode: "floating" },
    });
  });

  it("always sets window mode to floating", () => {
    const config = {
      backend: "openai",
      model: "gpt-4",
      directories: [],
      theme: "dark",
    };
    const params = buildSetConfigParams(config);
    expect(params.window).toEqual({ mode: "floating" });
  });
});

// ============================================================================
// Migration: runMigration
// ============================================================================

describe("runMigration", () => {
  it("skips migration when not needed", async () => {
    const getItem = vi.fn(() => null);
    const setItem = vi.fn();
    const setConfig = vi.fn();

    const result = await runMigration({ getItem, setItem, setConfig });
    expect(result).toEqual({ success: true, migrated: false });
    expect(setConfig).not.toHaveBeenCalled();
  });

  it("migrates successfully when config exists", async () => {
    const getItem = vi.fn((key: string) => {
      if (key === "gman-config")
        return JSON.stringify({
          backend: "ollama",
          model: "llama3",
          directories: [],
          theme: "dark",
        });
      return null;
    });
    const setItem = vi.fn();
    const setConfig = vi.fn().mockResolvedValue({ ok: true });

    const result = await runMigration({ getItem, setItem, setConfig });
    expect(result).toEqual({ success: true, migrated: true });
    expect(setConfig).toHaveBeenCalledWith({
      provider: "ollama",
      model: "llama3",
      theme: "dark",
      window: { mode: "floating" },
    });
    expect(setItem).toHaveBeenCalledWith("gman-migrated", "true");
  });

  it("returns error when config parsing fails", async () => {
    const getItem = vi.fn((key: string) => {
      if (key === "gman-config") return "invalid json";
      return null;
    });
    const setItem = vi.fn();
    const setConfig = vi.fn();

    const result = await runMigration({ getItem, setItem, setConfig });
    expect(result.success).toBe(false);
    expect(result.migrated).toBe(false);
    expect(result.error).toContain("parse");
    expect(setConfig).not.toHaveBeenCalled();
  });

  it("returns error when setConfig fails", async () => {
    const getItem = vi.fn((key: string) => {
      if (key === "gman-config")
        return JSON.stringify({
          backend: "ollama",
          model: "llama3",
          directories: [],
          theme: "dark",
        });
      return null;
    });
    const setItem = vi.fn();
    const setConfig = vi.fn().mockRejectedValue(new Error("RPC timeout"));

    const result = await runMigration({ getItem, setItem, setConfig });
    expect(result.success).toBe(false);
    expect(result.migrated).toBe(false);
    expect(result.error).toBe("RPC timeout");
  });

  it("returns error when setConfig returns ok=false", async () => {
    const getItem = vi.fn((key: string) => {
      if (key === "gman-config")
        return JSON.stringify({
          backend: "ollama",
          model: "llama3",
          directories: [],
          theme: "dark",
        });
      return null;
    });
    const setItem = vi.fn();
    const setConfig = vi.fn().mockResolvedValue({ ok: false });

    const result = await runMigration({ getItem, setItem, setConfig });
    expect(result.success).toBe(false);
    expect(result.error).toContain("ok=false");
  });
});
