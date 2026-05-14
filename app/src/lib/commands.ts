// Slash command parser for G-MAN companion mode.
// Pure functions — no side effects, easy to test.

export type ParsedCommand = {
  cmd: string;
  args: string[];
};

/**
 * Checks if the input text is a slash command.
 * Returns true only if the text starts with "/" (no leading whitespace).
 */
export function isSlashCommand(text: string): boolean {
  return text.startsWith("/");
}

/**
 * Parses a slash command string into its command name and arguments.
 * The command name is lowercased; arguments preserve their original case.
 *
 * @param text The raw input text (e.g., "/model llama3.2:3b")
 * @returns ParsedCommand with cmd (lowercased) and args array
 */
export function parseCommand(text: string): ParsedCommand {
  if (!text.startsWith("/")) {
    return { cmd: "", args: [] };
  }

  const withoutSlash = text.slice(1).trim();
  const parts = withoutSlash.split(/\s+/).filter((p) => p.length > 0);

  if (parts.length === 0) {
    return { cmd: "", args: [] };
  }

  return {
    cmd: parts[0].toLowerCase(),
    args: parts.slice(1),
  };
}
