import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/svelte";
import OnboardingWizard from "../components/OnboardingWizard.svelte";

describe("OnboardingWizard", () => {
  it("shows step 1 by default", () => {
    render(OnboardingWizard, {
      props: {
        onfinish: vi.fn(),
      },
    });

    expect(screen.getByText(/Choose your AI/)).toBeInTheDocument();
  });

  it("shows progress indicator with step count", () => {
    render(OnboardingWizard, {
      props: {
        onfinish: vi.fn(),
      },
    });

    expect(screen.getByText(/Step 1/)).toBeInTheDocument();
    expect(screen.getByText(/3/)).toBeInTheDocument();
  });

  it("has Skip and Next buttons on first step", () => {
    render(OnboardingWizard, {
      props: {
        onfinish: vi.fn(),
      },
    });

    expect(screen.getByRole("button", { name: /skip/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /next/i })).toBeInTheDocument();
  });

  it("advances to step 2 when Next is clicked", async () => {
    render(OnboardingWizard, {
      props: {
        onfinish: vi.fn(),
      },
    });

    const nextBtn = screen.getByRole("button", { name: /next/i });
    await fireEvent.click(nextBtn);

    expect(screen.getByText(/Allowed directories/)).toBeInTheDocument();
    expect(screen.getByText(/Step 2/)).toBeInTheDocument();
  });

  it("advances to step 3 when Next is clicked on step 2", async () => {
    render(OnboardingWizard, {
      props: {
        onfinish: vi.fn(),
      },
    });

    // Step 1 → Step 2
    await fireEvent.click(screen.getByRole("button", { name: /next/i }));

    // Step 2 → Step 3
    const nextBtn = screen.getByRole("button", { name: /next/i });
    await fireEvent.click(nextBtn);

    expect(screen.getByText(/Theme/i)).toBeInTheDocument();
    expect(screen.getByText(/Step 3/)).toBeInTheDocument();
  });

  it("calls onfinish when Finish is clicked on step 3", async () => {
    const onFinish = vi.fn();

    render(OnboardingWizard, {
      props: {
        onfinish: onFinish,
      },
    });

    // Navigate to step 3
    await fireEvent.click(screen.getByRole("button", { name: /next/i }));
    await fireEvent.click(screen.getByRole("button", { name: /next/i }));

    const finishBtn = screen.getByRole("button", { name: /finish/i });
    expect(finishBtn).toBeInTheDocument();

    await fireEvent.click(finishBtn);
    expect(onFinish).toHaveBeenCalledTimes(1);
  });

  it("calls onfinish when Skip is clicked", async () => {
    const onFinish = vi.fn();

    render(OnboardingWizard, {
      props: {
        onfinish: onFinish,
      },
    });

    const skipBtn = screen.getByRole("button", { name: /skip/i });
    await fireEvent.click(skipBtn);
    expect(onFinish).toHaveBeenCalledTimes(1);
  });

  it("shows Back button on step 2 and 3", async () => {
    render(OnboardingWizard, {
      props: {
        onfinish: vi.fn(),
      },
    });

    // No Back on step 1
    expect(
      screen.queryByRole("button", { name: /back/i }),
    ).not.toBeInTheDocument();

    // Step 1 → Step 2
    await fireEvent.click(screen.getByRole("button", { name: /next/i }));
    expect(screen.getByRole("button", { name: /back/i })).toBeInTheDocument();

    // Step 2 → Step 3
    await fireEvent.click(screen.getByRole("button", { name: /next/i }));
    expect(screen.getByRole("button", { name: /back/i })).toBeInTheDocument();
  });

  it("returns to previous step when Back is clicked", async () => {
    render(OnboardingWizard, {
      props: {
        onfinish: vi.fn(),
      },
    });

    // Step 1 → Step 2
    await fireEvent.click(screen.getByRole("button", { name: /next/i }));
    expect(screen.getByText(/Allowed directories/)).toBeInTheDocument();

    // Step 2 → Step 1
    await fireEvent.click(screen.getByRole("button", { name: /back/i }));
    expect(screen.getByText(/Choose your AI/)).toBeInTheDocument();
  });

  it("shows Ollama option in step 1", () => {
    render(OnboardingWizard, {
      props: {
        onfinish: vi.fn(),
      },
    });

    expect(screen.getByText(/Ollama/)).toBeInTheDocument();
    expect(screen.getByText(/local/)).toBeInTheDocument();
  });

  it("shows API key option in step 1", () => {
    render(OnboardingWizard, {
      props: {
        onfinish: vi.fn(),
      },
    });

    expect(screen.getByText("🔑 API Key")).toBeInTheDocument();
  });

  it("shows directory checkboxes in step 2", async () => {
    render(OnboardingWizard, {
      props: {
        onfinish: vi.fn(),
      },
    });

    // Navigate to step 2
    await fireEvent.click(screen.getByRole("button", { name: /next/i }));

    expect(screen.getByText("~/.config")).toBeInTheDocument();
    expect(screen.getByText("~/.local")).toBeInTheDocument();
  });

  it("shows theme options in step 3", async () => {
    render(OnboardingWizard, {
      props: {
        onfinish: vi.fn(),
      },
    });

    // Navigate to step 3
    await fireEvent.click(screen.getByRole("button", { name: /next/i }));
    await fireEvent.click(screen.getByRole("button", { name: /next/i }));

    expect(screen.getByText(/Dark/)).toBeInTheDocument();
    expect(screen.getByText(/Light/)).toBeInTheDocument();
    expect(screen.getByText(/System/)).toBeInTheDocument();
  });
});
