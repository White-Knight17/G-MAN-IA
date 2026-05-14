// Config migration: localStorage → config.json via RPC
// Pure functions for testability — side effects injected.

export type LocalStorageConfig = {
  backend: string;
  model: string;
  directories: string[];
  theme: string;
};

export type MigrationResult = {
  success: boolean;
  migrated: boolean;
  error?: string;
};

/**
 * Checks if migration is needed.
 * Returns true if localStorage has config AND migration flag is not set.
 */
export function needsMigration(
  getItem: (key: string) => string | null,
): boolean {
  const hasConfig = getItem("gman-config") !== null;
  const alreadyMigrated = getItem("gman-migrated") === "true";
  return hasConfig && !alreadyMigrated;
}

/**
 * Parses localStorage config into the format expected by config.set RPC.
 */
export function parseLocalStorageConfig(
  raw: string | null,
): LocalStorageConfig | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw);
    return {
      backend: parsed.backend ?? "ollama",
      model: parsed.model ?? "",
      directories: parsed.directories ?? [],
      theme: parsed.theme ?? "dark",
    };
  } catch {
    return null;
  }
}

/**
 * Builds the config.set RPC params from a parsed localStorage config.
 */
export function buildSetConfigParams(
  config: LocalStorageConfig,
): Record<string, unknown> {
  return {
    provider: config.backend,
    model: config.model,
    theme: config.theme,
    window: { mode: "floating" },
  };
}

/**
 * Runs the full migration flow.
 * Accepts injected dependencies for testability.
 */
export async function runMigration(deps: {
  getItem: (key: string) => string | null;
  setItem: (key: string, value: string) => void;
  setConfig: (params: Record<string, unknown>) => Promise<{ ok: boolean }>;
}): Promise<MigrationResult> {
  if (!needsMigration(deps.getItem)) {
    return { success: true, migrated: false };
  }

  const raw = deps.getItem("gman-config");
  const config = parseLocalStorageConfig(raw);

  if (!config) {
    return {
      success: false,
      migrated: false,
      error: "Failed to parse localStorage config",
    };
  }

  try {
    const params = buildSetConfigParams(config);
    const result = await deps.setConfig(params);

    if (result.ok) {
      deps.setItem("gman-migrated", "true");
      return { success: true, migrated: true };
    } else {
      return {
        success: false,
        migrated: false,
        error: "config.set RPC returned ok=false",
      };
    }
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    return { success: false, migrated: false, error: msg };
  }
}
