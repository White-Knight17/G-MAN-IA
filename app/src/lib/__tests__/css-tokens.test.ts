import { describe, it, expect } from "vitest";
import { readFileSync } from "fs";
import path from "path";

// CSS token tests — verify design tokens are defined in app.css
// These are structural tests for the CSS custom properties.

describe("CSS elevation tokens", () => {
  let cssContent: string;

  beforeAll(() => {
    cssContent = readFileSync(
      path.resolve(__dirname, "../../../src/app.css"),
      "utf-8",
    );
  });

  it("defines elevation-1 shadow token", () => {
    expect(cssContent).toContain("--gman-elevation-1");
  });

  it("defines elevation-2 shadow token", () => {
    expect(cssContent).toContain("--gman-elevation-2");
  });

  it("defines elevation-3 shadow token", () => {
    expect(cssContent).toContain("--gman-elevation-3");
  });

  it("defines elevation-4 shadow token", () => {
    expect(cssContent).toContain("--gman-elevation-4");
  });

  it("elevation tokens contain rgba shadow values", () => {
    expect(cssContent).toMatch(/--gman-elevation-1:.*rgba\(0,?\s*0,?\s*0/);
    expect(cssContent).toMatch(/--gman-elevation-2:.*rgba\(0,?\s*0,?\s*0/);
  });
});

describe("CSS spacing tokens", () => {
  let cssContent: string;

  beforeAll(() => {
    cssContent = readFileSync(
      path.resolve(__dirname, "../../../src/app.css"),
      "utf-8",
    );
  });

  it("defines spacing tokens xs through xl", () => {
    expect(cssContent).toContain("--gman-space-xs");
    expect(cssContent).toContain("--gman-space-sm");
    expect(cssContent).toContain("--gman-space-md");
    expect(cssContent).toContain("--gman-space-lg");
    expect(cssContent).toContain("--gman-space-xl");
  });

  it("spacing tokens use pixel values", () => {
    expect(cssContent).toMatch(/--gman-space-xs:\s*4px/);
    expect(cssContent).toMatch(/--gman-space-sm:\s*8px/);
    expect(cssContent).toMatch(/--gman-space-md:\s*16px/);
    expect(cssContent).toMatch(/--gman-space-lg:\s*24px/);
    expect(cssContent).toMatch(/--gman-space-xl:\s*32px/);
  });
});

describe("CSS typography tokens", () => {
  let cssContent: string;

  beforeAll(() => {
    cssContent = readFileSync(
      path.resolve(__dirname, "../../../src/app.css"),
      "utf-8",
    );
  });

  it("defines typography font tokens", () => {
    expect(cssContent).toContain("--gman-font-title");
    expect(cssContent).toContain("--gman-font-body");
    expect(cssContent).toContain("--gman-font-caption");
  });
});

describe("CSS button transitions", () => {
  let cssContent: string;

  beforeAll(() => {
    cssContent = readFileSync(
      path.resolve(__dirname, "../../../src/app.css"),
      "utf-8",
    );
  });

  it("defines .gman-btn transition class", () => {
    expect(cssContent).toContain(".gman-btn");
    expect(cssContent).toContain("transition:");
  });

  it("defines hover state with elevation", () => {
    expect(cssContent).toContain(".gman-btn:hover");
    expect(cssContent).toContain("var(--gman-elevation-1)");
  });

  it("defines active state with scale", () => {
    expect(cssContent).toContain(".gman-btn:active");
    expect(cssContent).toContain("scale(0.98)");
  });
});
