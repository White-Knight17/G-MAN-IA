// Package main is the entry point for the G-MAN JSON-RPC sidecar.
// It communicates with the Tauri shell over stdin/stdout using NDJSON
// framing. The sidecar contains all domain logic (agent, tools, sandbox,
// permissions) exposed as remote method calls.
//
// Architecture:
//
//	cmd/gman-server/main.go  ← composition root
//	      │
//	      ├──► transport.Server (JSON-RPC 2.0 stdin/stdout)
//	      │
//	      ├──► infrastructure adapters (ollama, sandbox, tools, permission)
//	      │
//	      └──► application use cases (orchestrator, executor, grant manager)
//
// This is the NEW entry point; cmd/gman/main.go is preserved as TUI fallback.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gentleman/gman/internal/application"
	gmanconfig "github.com/gentleman/gman/internal/infrastructure/config"
	"github.com/gentleman/gman/internal/domain"
	"github.com/gentleman/gman/internal/infrastructure/ollama"
	"github.com/gentleman/gman/internal/infrastructure/permission"
	"github.com/gentleman/gman/internal/infrastructure/sandbox"
	"github.com/gentleman/gman/internal/infrastructure/tools"
	"github.com/gentleman/gman/internal/transport"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("G-MAN sidecar crashed: %v", err)
	}
}

// run wires dependencies and starts the JSON-RPC server loop.
func run() error {
	// ---------------------------------------------------------------------------
	// Default configuration (hardcoded for sidecar; config UI lives in Svelte)
	// ---------------------------------------------------------------------------
	modelName := "llama3.2:3b"
	ollamaURL := "http://localhost:11434"
	allowedDirs := []string{
		expandHome("~/.config"),
		expandHome("~/.local"),
	}

	// Load persistent config if available
	configPath := filepath.Join(expandHome("~/.config"), "gman", "config.json")
	persistedCfg, _ := gmanconfig.Load(configPath)
	if persistedCfg.Backend.Model != "" {
		modelName = persistedCfg.Backend.Model
	}
	if persistedCfg.Backend.OllamaURL != "" {
		ollamaURL = persistedCfg.Backend.OllamaURL
	}

	// ---------------------------------------------------------------------------
	// Infrastructure adapters
	// ---------------------------------------------------------------------------
	permRepo := permission.NewInMemoryPermissionRepo()
	bwSandbox := sandbox.NewBubblewrapSandbox(allowedDirs)

	toolList := []domain.Tool{
		tools.NewReadFileTool(allowedDirs, bwSandbox),
		tools.NewWriteFileTool(allowedDirs, bwSandbox),
		tools.NewListDirTool(allowedDirs, bwSandbox),
		tools.NewCommandTool(bwSandbox, allowedDirs),
		tools.NewCheckSyntaxTool(bwSandbox),
		tools.NewSearchWikiTool(bwSandbox),
	}

	agent := ollama.NewOllamaClient(modelName, toolList, ollamaURL)

	// ---------------------------------------------------------------------------
	// Application use cases
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
	// JSON-RPC server
	// ---------------------------------------------------------------------------
	server := transport.NewServer(os.Stdin, os.Stdout)

	// "ping" — health check
	server.Handle("ping", func(req transport.Request) transport.Response {
		return transport.Response{Result: "pong"}
	})

	// "agent.chat" — non-streaming chat
	server.Handle("agent.chat", func(req transport.Request) transport.Response {
		var params struct {
			Input     string `json:"input"`
			SessionID string `json:"session_id,omitempty"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: "invalid params: " + err.Error(),
				},
			}
		}

		session := &domain.Session{
			ID:        params.SessionID,
			Messages:  make([]domain.ChatMessage, 0),
			Grants:    make([]domain.Grant, 0),
			StartedAt: time.Now().UTC().Format(time.RFC3339),
		}

		ctx := context.Background()
		response, err := orchestrator.HandleMessage(ctx, session, params.Input)
		if err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InternalError,
					Message: err.Error(),
				},
			}
		}

		return transport.Response{
			Result: map[string]interface{}{
				"message":    response,
				"session_id": session.ID,
			},
		}
	})

	// "agent.stream" — streaming chat
	server.Handle("agent.stream", func(req transport.Request) transport.Response {
		var params struct {
			Input     string `json:"input"`
			SessionID string `json:"session_id,omitempty"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: "invalid params: " + err.Error(),
				},
			}
		}

		session := &domain.Session{
			ID:        params.SessionID,
			Messages:  make([]domain.ChatMessage, 0),
			Grants:    make([]domain.Grant, 0),
			StartedAt: time.Now().UTC().Format(time.RFC3339),
		}

		ctx := context.Background()
		ch, err := orchestrator.HandleMessageStream(ctx, session, params.Input)
		if err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InternalError,
					Message: err.Error(),
				},
			}
		}

		// Stream events as notifications to the client
		for evt := range ch {
			server.SendNotification("agent.event", map[string]interface{}{
				"type":       evt.Type,
				"content":    evt.Content,
				"error":      evt.Error,
				"session_id": session.ID,
			})
		}

		return transport.Response{
			Result: map[string]interface{}{
				"session_id": session.ID,
				"status":     "complete",
			},
		}
	})

	// "permission.grant" — grant directory access
	server.Handle("permission.grant", func(req transport.Request) transport.Response {
		var params struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: "invalid params: " + err.Error(),
				},
			}
		}

		var mode domain.PermissionMode
		switch params.Mode {
		case "ro":
			mode = domain.PermissionRead
		case "rw":
			mode = domain.PermissionWrite
		default:
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: "mode must be 'ro' or 'rw'",
				},
			}
		}

		if err := permRepo.Grant(params.Path, mode); err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InternalError,
					Message: err.Error(),
				},
			}
		}

		return transport.Response{Result: "granted"}
	})

	// "permission.list" — list active grants
	server.Handle("permission.list", func(req transport.Request) transport.Response {
		grants := permRepo.ListGrants()
		return transport.Response{Result: grants}
	})

	// "model.list" — list available Ollama models
	server.Handle("model.list", func(req transport.Request) transport.Response {
		ctx := context.Background()
		models, err := agent.ListModels(ctx)
		if err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InternalError,
					Message: err.Error(),
				},
			}
		}
		return transport.Response{Result: map[string]interface{}{
			"models": models,
		}}
	})

	// "model.pull" — start pulling a model (streams progress via notifications)
	server.Handle("model.pull", func(req transport.Request) transport.Response {
		var params struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: err.Error(),
				},
			}
		}
		if params.Model == "" {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: "model parameter is required",
				},
			}
		}

		progressCh := make(chan ollama.PullProgress, 10)

		go func() {
			ctx := context.Background()
			err := agent.PullModel(ctx, params.Model, progressCh)
			close(progressCh)
			if err != nil {
				server.SendNotification("model.pull.error", map[string]interface{}{
					"model": params.Model,
					"error": err.Error(),
				})
				return
			}
		}()

		// Stream progress as notifications
		for p := range progressCh {
			server.SendNotification("model.pull.progress", map[string]interface{}{
				"model":     params.Model,
				"status":    p.Status,
				"completed": p.Completed,
				"total":     p.Total,
				"percent":   p.Percent,
			})
		}

		return transport.Response{Result: map[string]interface{}{
			"status": "complete",
			"model":  params.Model,
		}}
	})

	// "config.get" — return current config (API keys masked)
	server.Handle("config.get", func(req transport.Request) transport.Response {
		cfg, err := gmanconfig.Load(configPath)
		if err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InternalError,
					Message: err.Error(),
				},
			}
		}
		return transport.Response{Result: safeConfigMap(cfg)}
	})

	// "config.set" — update config fields and persist
	server.Handle("config.set", func(req transport.Request) transport.Response {
		cfg, err := gmanconfig.Load(configPath)
		if err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InternalError,
					Message: err.Error(),
				},
			}
		}

		var params struct {
			Theme     string            `json:"theme,omitempty"`
			Model     string            `json:"model,omitempty"`
			Provider  string            `json:"provider,omitempty"`
			OllamaURL string            `json:"ollama_url,omitempty"`
			APIKey    string            `json:"api_key,omitempty"`
			Window    map[string]any    `json:"window,omitempty"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InvalidParams,
					Message: err.Error(),
				},
			}
		}

		if params.Theme != "" {
			cfg.Theme = params.Theme
		}
		if params.Model != "" {
			cfg.Backend.Model = params.Model
		}
		if params.Provider != "" {
			cfg.Backend.Provider = params.Provider
		}
		if params.OllamaURL != "" {
			cfg.Backend.OllamaURL = params.OllamaURL
		}
		if params.APIKey != "" {
			if cfg.Backend.APIKeys == nil {
				cfg.Backend.APIKeys = make(map[string]string)
			}
			cfg.Backend.APIKeys[cfg.Backend.Provider] = params.APIKey
		}
		if params.Window != nil {
			if mode, ok := params.Window["mode"].(string); ok {
				cfg.Window.Mode = mode
			}
			if width, ok := params.Window["width"].(float64); ok {
				cfg.Window.Width = int(width)
			}
		}

		if err := cfg.Save(configPath); err != nil {
			return transport.Response{
				Error: &transport.Error{
					Code:    transport.InternalError,
					Message: err.Error(),
				},
			}
		}

		return transport.Response{Result: map[string]bool{"ok": true}}
	})

	// ---------------------------------------------------------------------------
	// Signal handling
	// ---------------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	// ---------------------------------------------------------------------------
	// Send ready notification and serve
	// ---------------------------------------------------------------------------
	server.SendNotification("ready", map[string]string{
		"version": "1.0.0",
	})

	return server.Serve(ctx)
}

// expandHome expands ~ to $HOME.
func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	return path
}

// safeConfigMap returns a config representation without exposing API key values.
func safeConfigMap(c gmanconfig.Config) map[string]interface{} {
	hasKey := false
	for _, v := range c.Backend.APIKeys {
		if v != "" {
			hasKey = true
			break
		}
	}
	return map[string]interface{}{
		"provider":    c.Backend.Provider,
		"model":       c.Backend.Model,
		"has_api_key": hasKey,
		"theme":       c.Theme,
		"window": map[string]interface{}{
			"mode": c.Window.Mode,
		},
	}
}
