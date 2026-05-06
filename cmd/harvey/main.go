// Package main is the entry point for Harvey, an AI-powered assistant for
// Arch Linux + Hyprland. It wires together domain interfaces, application
// use cases, and infrastructure adapters, then launches the Bubbletea TUI.
//
// Architecture (Clean/Hexagonal):
//
//	cmd/harvey/main.go  ← composition root
//	      │
//	      ├──► infrastructure adapters (ollama, sandbox, tools, permission)
//	      │
//	      ├──► application use cases (orchestrator, executor, grant manager)
//	      │
//	      └──► UI (Bubbletea TUI)
//
// Dependency injection: all adapters are instantiated here and injected
// into application use cases. The TUI receives the orchestrator through
// its local ChatOrchestrator interface — no direct infrastructure imports.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gentleman/programas/harvey/internal/application"
	"github.com/gentleman/programas/harvey/internal/domain"
	"github.com/gentleman/programas/harvey/internal/infrastructure/ollama"
	"github.com/gentleman/programas/harvey/internal/infrastructure/permission"
	"github.com/gentleman/programas/harvey/internal/infrastructure/sandbox"
	"github.com/gentleman/programas/harvey/internal/infrastructure/tools"
	"github.com/gentleman/programas/harvey/internal/ui/tui"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Harvey crashed: %v", err)
	}
}

// run encapsulates the full startup lifecycle:
//
//  1. Parse flags
//  2. Expand and validate allowed directories
//  3. Instantiate infrastructure adapters (permissions, sandbox, tools, agent)
//  4. Wire application use cases (GrantManager, ToolExecutor, ChatOrchestrator)
//  5. Print startup banner
//  6. Health check Ollama connectivity and model availability
//  7. Create TUI model
//  8. Handle OS signals for graceful shutdown
//  9. Run the Bubbletea program
func run() error {
	// ---------------------------------------------------------------------------
	// Step 1: Parse command-line flags
	// ---------------------------------------------------------------------------
	modelName := flag.String("model", "llama3.2:3b", "Ollama model to use")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama API URL")
	allowedDirsFlag := flag.String("allowed-dirs", "~/.config,~/.local", "Comma-separated allowed directories")
	flag.Parse()

	// ---------------------------------------------------------------------------
	// Step 2: Expand and validate allowed directories
	// ---------------------------------------------------------------------------
	allowedDirs, err := parseAllowedDirs(*allowedDirsFlag)
	if err != nil {
		return fmt.Errorf("invalid allowed-dirs: %w", err)
	}

	// ---------------------------------------------------------------------------
	// Step 3: Instantiate infrastructure adapters
	// ---------------------------------------------------------------------------

	// 3a. Permission repository (session-scoped, in-memory)
	permRepo := permission.NewInMemoryPermissionRepo()

	// 3b. Sandbox (Bubblewrap for command isolation)
	bwSandbox := sandbox.NewBubblewrapSandbox(allowedDirs)

	// 3c. Tools (each implements domain.Tool)
	toolList := []domain.Tool{
		tools.NewReadFileTool(allowedDirs, bwSandbox),
		tools.NewWriteFileTool(allowedDirs, bwSandbox),
		tools.NewListDirTool(allowedDirs, bwSandbox),
		tools.NewCommandTool(bwSandbox, allowedDirs),
		tools.NewCheckSyntaxTool(bwSandbox),
		tools.NewSearchWikiTool(bwSandbox),
	}

	// 3d. Agent (Ollama HTTP client implementing domain.Agent)
	agent := ollama.NewOllamaClient(*modelName, toolList, *ollamaURL)

	// ---------------------------------------------------------------------------
	// Step 4: Wire application use cases
	// ---------------------------------------------------------------------------
	grantMgr := application.NewGrantManager(permRepo)
	toolExec := application.NewToolExecutor(toolList, bwSandbox, permRepo)
	orchestrator := application.NewChatOrchestrator(
		agent,
		toolExec,
		grantMgr,
		application.WithMaxIterations(5),
	)

	// ---------------------------------------------------------------------------
	// Step 5: Startup banner
	// ---------------------------------------------------------------------------
	printBanner(*modelName, allowedDirs)

	// ---------------------------------------------------------------------------
	// Step 6: Health check
	// ---------------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Print("Checking Ollama connectivity... ")
	if err := agent.HealthCheck(ctx); err != nil {
		fmt.Println("FAIL")
		return fmt.Errorf("startup health check failed: %w", err)
	}
	fmt.Println("OK")

	// ---------------------------------------------------------------------------
	// Step 7: Create TUI model
	// ---------------------------------------------------------------------------
	tuiModel := tui.NewModel(orchestrator)
	p := tea.NewProgram(tuiModel, tea.WithAltScreen())

	// ---------------------------------------------------------------------------
	// Step 8: Signal handling for graceful shutdown
	// ---------------------------------------------------------------------------
	// Bubbletea v1.3.x does not have RunWithContext. Instead, we intercept
	// SIGINT/SIGTERM in a goroutine and send tea.Quit() to the program.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		p.Send(tea.Quit()) // graceful exit through Bubbletea
	}()

	// ---------------------------------------------------------------------------
	// Step 9: Run the Bubbletea program
	// ---------------------------------------------------------------------------
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}

	return nil
}

// parseAllowedDirs splits a comma-separated list of directories, expands ~ to
// $HOME, resolves to absolute paths, and validates that each path exists.
// Returns a cleaned slice of absolute paths or an error.
func parseAllowedDirs(raw string) ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	parts := strings.Split(raw, ",")
	seen := make(map[string]bool)
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Expand ~ to $HOME
		if strings.HasPrefix(part, "~/") {
			part = filepath.Join(home, part[2:])
		} else if part == "~" {
			part = home
		}

		// Resolve to absolute path
		abs, err := filepath.Abs(part)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve %q: %w", part, err)
		}

		// Validate path exists (or at least its parent)
		info, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("directory does not exist: %s", abs)
			}
			return nil, fmt.Errorf("cannot stat %s: %w", abs, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("not a directory: %s", abs)
		}

		// Deduplicate
		if seen[abs] {
			continue
		}
		seen[abs] = true
		result = append(result, abs)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid directories specified")
	}

	return result, nil
}

// printBanner prints a startup banner showing model and allowed directories.
func printBanner(model string, dirs []string) {
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║        Harvey — Arch AI Assistant        ║")
	fmt.Println("╠══════════════════════════════════════════╣")
	fmt.Printf("║  Model:   %-30s ║\n", model)
	for i, d := range dirs {
		if i == 0 {
			fmt.Printf("║  Dirs:    %-30s ║\n", d)
		} else {
			fmt.Printf("║           %-30s ║\n", d)
		}
	}
	fmt.Println("╚══════════════════════════════════════════╝")
}
