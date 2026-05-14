import { describe, it, expect } from "vitest";
import { parseCommand, isSlashCommand } from "../commands";

describe("isSlashCommand", () => {
  it("returns true for text starting with /", () => {
    expect(isSlashCommand("/help")).toBe(true);
    expect(isSlashCommand("/clear")).toBe(true);
    expect(isSlashCommand("/model llama3")).toBe(true);
  });

  it("returns false for text not starting with /", () => {
    expect(isSlashCommand("hello")).toBe(false);
    expect(isSlashCommand("what is /help")).toBe(false);
    expect(isSlashCommand("")).toBe(false);
  });

  it("returns false for text with only whitespace before /", () => {
    expect(isSlashCommand(" /help")).toBe(false);
  });
});

describe("parseCommand", () => {
  it("parses simple command without args", () => {
    const result = parseCommand("/help");
    expect(result).toEqual({ cmd: "help", args: [] });
  });

  it("parses command with single arg", () => {
    const result = parseCommand("/model llama3.2:3b");
    expect(result).toEqual({ cmd: "model", args: ["llama3.2:3b"] });
  });

  it("parses command with multiple args", () => {
    const result = parseCommand("/api openai sk-123");
    expect(result).toEqual({ cmd: "api", args: ["openai", "sk-123"] });
  });

  it("lowercases command name", () => {
    const result = parseCommand("/HELP");
    expect(result.cmd).toBe("help");
  });

  it("preserves case in args", () => {
    const result = parseCommand("/model Llama3.2:3B");
    expect(result.args).toEqual(["Llama3.2:3B"]);
  });

  it("handles extra whitespace", () => {
    const result = parseCommand("/model   llama3   ");
    expect(result).toEqual({ cmd: "model", args: ["llama3"] });
  });

  it("returns empty args for command with no args", () => {
    const result = parseCommand("/clear");
    expect(result.args).toEqual([]);
  });
});
