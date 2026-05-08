import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/svelte";
import PermissionDialog from "../components/PermissionDialog.svelte";

describe("PermissionDialog", () => {
  it("renders the path being requested", () => {
    render(PermissionDialog, {
      props: {
        tool: "write_file",
        path: "/home/user/.config/hypr/hyprland.conf",
        mode: "rw",
        onallow: vi.fn(),
        ondeny: vi.fn(),
      },
    });

    expect(screen.getByText(/write_file/)).toBeInTheDocument();
    expect(
      screen.getByText(/\/home\/user\/\.config\/hypr\/hyprland\.conf/),
    ).toBeInTheDocument();
  });

  it("renders Allow and Deny buttons", () => {
    render(PermissionDialog, {
      props: {
        tool: "read_file",
        path: "/home/user/.config/test.txt",
        mode: "ro",
        onallow: vi.fn(),
        ondeny: vi.fn(),
      },
    });

    const allowBtn = screen.getByRole("button", { name: /allow/i });
    const denyBtn = screen.getByRole("button", { name: /deny/i });

    expect(allowBtn).toBeInTheDocument();
    expect(denyBtn).toBeInTheDocument();
  });

  it("calls onallow when Allow is clicked", async () => {
    const onAllow = vi.fn();

    render(PermissionDialog, {
      props: {
        tool: "read_file",
        path: "/home/user/.config/test.txt",
        mode: "ro",
        onallow: onAllow,
        ondeny: vi.fn(),
      },
    });

    const allowBtn = screen.getByRole("button", { name: /allow/i });
    await fireEvent.click(allowBtn);
    expect(onAllow).toHaveBeenCalledTimes(1);
  });

  it("calls ondeny when Deny is clicked", async () => {
    const onDeny = vi.fn();

    render(PermissionDialog, {
      props: {
        tool: "read_file",
        path: "/home/user/.config/test.txt",
        mode: "ro",
        onallow: vi.fn(),
        ondeny: onDeny,
      },
    });

    const denyBtn = screen.getByRole("button", { name: /deny/i });
    await fireEvent.click(denyBtn);
    expect(onDeny).toHaveBeenCalledTimes(1);
  });

  it("shows read mode indicator", () => {
    render(PermissionDialog, {
      props: {
        tool: "read_file",
        path: "/home/user/.config/file.txt",
        mode: "ro",
        onallow: vi.fn(),
        ondeny: vi.fn(),
      },
    });

    expect(screen.getByText("read-only")).toBeInTheDocument();
  });

  it("shows write mode indicator", () => {
    render(PermissionDialog, {
      props: {
        tool: "write_file",
        path: "/home/user/.config/file.txt",
        mode: "rw",
        onallow: vi.fn(),
        ondeny: vi.fn(),
      },
    });

    expect(screen.getByText("read/write")).toBeInTheDocument();
  });

  it("shows Allow G-MAN to ... messaging", () => {
    render(PermissionDialog, {
      props: {
        tool: "run_command",
        path: "systemctl status",
        mode: "rw",
        onallow: vi.fn(),
        ondeny: vi.fn(),
      },
    });

    expect(screen.getByText(/Allow G-MAN/)).toBeInTheDocument();
  });
});
