// Package main is the entry point for the G-MAN JSON-RPC sidecar.
// It communicates with the Tauri shell over stdin/stdout using NDJSON
// framing. Supports Ollama (local) and OpenAI-compatible APIs (remote).
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/gentleman/gman/internal/application"
	gmanconfig "github.com/gentleman/gman/internal/infrastructure/config"
	"github.com/gentleman/gman/internal/domain"
	"github.com/gentleman/gman/internal/infrastructure/ollama"
	"github.com/gentleman/gman/internal/infrastructure/openai"
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
	configPath := filepath.Join(expandHome("~/.config"), "gman", "config.json")

	// Load config or use defaults
	cfg, err := gmanconfig.Load(configPath)
	if err != nil {
		cfg = gmanconfig.Defaults()
	}

	allowedDirs := []string{
		expandHome("~/.config"),
		expandHome("~/.local"),
	}

	// Infrastructure
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

	grantMgr := application.NewGrantManager(permRepo)
	toolExec := application.NewToolExecutor(toolList, bwSandbox, permRepo)

	// Agent — may be swapped when provider changes
	var agentMu sync.RWMutex
	agent := buildAgent(cfg.Backend, toolList)
	orchestrator := application.NewChatOrchestrator(agent, toolExec, grantMgr, application.WithMaxIterations(5))

	// Rebuild orchestrator on provider change
	updateAgent := func() {
		agentMu.Lock()
		defer agentMu.Unlock()
		newCfg, err := gmanconfig.Load(configPath)
		if err != nil {
			return
		}
		newAgent := buildAgent(newCfg.Backend, toolList)
		orchestrator = application.NewChatOrchestrator(newAgent, toolExec, grantMgr, application.WithMaxIterations(5))
	}

	// JSON-RPC server
	server := transport.NewServer(os.Stdin, os.Stdout)

	// "ping"
	server.Handle("ping", func(req transport.Request) transport.Response {
		return transport.Response{Result: "pong"}
	})

	// "agent.chat"
	server.Handle("agent.chat", func(req transport.Request) transport.Response {
		var params struct {
			Input     string `json:"input"`
			SessionID string `json:"session_id,omitempty"`
		}
		_ = json.Unmarshal(req.Params, &params)

		session := &domain.Session{
			ID:        params.SessionID,
			Messages:  make([]domain.ChatMessage, 0),
			Grants:    make([]domain.Grant, 0),
			StartedAt: time.Now().UTC().Format(time.RFC3339),
		}

		agentMu.RLock()
		o := orchestrator
		agentMu.RUnlock()

		response, err := o.HandleMessage(context.Background(), session, params.Input)
		if err != nil {
			return transport.Response{Error: &transport.Error{Code: transport.InternalError, Message: err.Error()}}
		}
		return transport.Response{Result: map[string]interface{}{"message": response, "session_id": session.ID}}
	})

	// "agent.stream"
	server.Handle("agent.stream", func(req transport.Request) transport.Response {
		var params struct {
			Input     string `json:"input"`
			SessionID string `json:"session_id,omitempty"`
		}
		_ = json.Unmarshal(req.Params, &params)

		session := &domain.Session{
			ID:        params.SessionID,
			Messages:  make([]domain.ChatMessage, 0),
			Grants:    make([]domain.Grant, 0),
			StartedAt: time.Now().UTC().Format(time.RFC3339),
		}

		agentMu.RLock()
		o := orchestrator
		agentMu.RUnlock()

		ch, err := o.HandleMessageStream(context.Background(), session, params.Input)
		if err != nil {
			return transport.Response{Error: &transport.Error{Code: transport.InternalError, Message: err.Error()}}
		}

		for evt := range ch {
			server.SendNotification("agent.event", map[string]interface{}{
				"type": evt.Type, "content": evt.Content, "error": evt.Error, "session_id": session.ID,
			})
		}

		return transport.Response{Result: map[string]interface{}{"session_id": session.ID, "status": "complete"}}
	})

	// "permission.grant"
	server.Handle("permission.grant", func(req transport.Request) transport.Response {
		var params struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
		}
		_ = json.Unmarshal(req.Params, &params)
		var mode domain.PermissionMode
		switch params.Mode {
		case "ro":
			mode = domain.PermissionRead
		case "rw":
			mode = domain.PermissionWrite
		default:
			return transport.Response{Error: &transport.Error{Code: transport.InvalidParams, Message: "mode must be 'ro' or 'rw'"}}
		}
		_ = permRepo.Grant(params.Path, mode)
		return transport.Response{Result: "granted"}
	})

	// "permission.list"
	server.Handle("permission.list", func(req transport.Request) transport.Response {
		return transport.Response{Result: permRepo.ListGrants()}
	})

	// "model.list" — Ollama only
	server.Handle("model.list", func(req transport.Request) transport.Response {
		ollamaClient := ollama.NewOllamaClient("", nil, "http://localhost:11434")
		models, err := ollamaClient.ListModels(context.Background())
		if err != nil {
			return transport.Response{Error: &transport.Error{Code: transport.InternalError, Message: err.Error()}}
		}
		return transport.Response{Result: map[string]interface{}{"models": models}}
	})

	// "model.pull" — Ollama only
	server.Handle("model.pull", func(req transport.Request) transport.Response {
		var params struct{ Model string `json:"model"` }
		_ = json.Unmarshal(req.Params, &params)
		if params.Model == "" {
			return transport.Response{Error: &transport.Error{Code: transport.InvalidParams, Message: "model required"}}
		}
		ollamaClient := ollama.NewOllamaClient("", nil, "http://localhost:11434")
		progressCh := make(chan ollama.PullProgress, 10)
		go func() {
			_ = ollamaClient.PullModel(context.Background(), params.Model, progressCh)
			close(progressCh)
		}()
		for p := range progressCh {
			server.SendNotification("model.pull.progress", map[string]interface{}{
				"model": params.Model, "status": p.Status, "completed": p.Completed, "total": p.Total, "percent": p.Percent,
			})
		}
		return transport.Response{Result: map[string]interface{}{"status": "complete", "model": params.Model}}
	})

	// "config.get"
	server.Handle("config.get", func(req transport.Request) transport.Response {
		cfg, err := gmanconfig.Load(configPath)
		if err != nil {
			return transport.Response{Error: &transport.Error{Code: transport.InternalError, Message: err.Error()}}
		}
		return transport.Response{Result: safeConfigMap(cfg)}
	})

	// "config.set"
	server.Handle("config.set", func(req transport.Request) transport.Response {
		cfg, err := gmanconfig.Load(configPath)
		if err != nil {
			return transport.Response{Error: &transport.Error{Code: transport.InternalError, Message: err.Error()}}
		}

		var params struct {
			Theme    string            `json:"theme,omitempty"`
			Model    string            `json:"model,omitempty"`
			Provider string            `json:"provider,omitempty"`
			OllamaURL string          `json:"ollama_url,omitempty"`
			APIKey   string            `json:"api_key,omitempty"`
			BaseURL  string            `json:"base_url,omitempty"`
			Window   map[string]any    `json:"window,omitempty"`
		}
		_ = json.Unmarshal(req.Params, &params)

		providerChanged := false

		if params.Theme != "" {
			cfg.Theme = params.Theme
		}
		if params.Model != "" {
			cfg.Backend.Model = params.Model
		}
		if params.Provider != "" && params.Provider != cfg.Backend.Provider {
			cfg.Backend.Provider = params.Provider
			providerChanged = true
		}
		if params.OllamaURL != "" {
			cfg.Backend.OllamaURL = params.OllamaURL
		}
		if params.APIKey != "" {
			if cfg.Backend.APIKeys == nil {
				cfg.Backend.APIKeys = make(map[string]string)
			}
			cfg.Backend.APIKeys[cfg.Backend.Provider] = params.APIKey
			providerChanged = true // API key set means we're switching to remote
		}
		if params.BaseURL != "" {
			if cfg.Backend.BaseURLs == nil {
				cfg.Backend.BaseURLs = make(map[string]string)
			}
			cfg.Backend.BaseURLs[cfg.Backend.Provider] = params.BaseURL
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
			return transport.Response{Error: &transport.Error{Code: transport.InternalError, Message: err.Error()}}
		}

		if providerChanged {
			updateAgent()
		}

		return transport.Response{Result: map[string]bool{"ok": true}}
	})

	// "provider.list" — list supported remote providers
	server.Handle("provider.list", func(req transport.Request) transport.Response {
		return transport.Response{Result: []map[string]string{
			{"id": "ollama", "name": "Ollama (Local)", "base_url": "http://localhost:11434", "type": "local"},
			{"id": "openai", "name": "OpenAI", "base_url": "https://api.openai.com", "type": "remote"},
			{"id": "deepseek", "name": "DeepSeek", "base_url": "https://api.deepseek.com", "type": "remote"},
			{"id": "groq", "name": "Groq", "base_url": "https://api.groq.com/openai", "type": "remote"},
		}}
	})

	// Signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	server.SendNotification("ready", map[string]string{"version": "2.1.0"})
	return server.Serve(ctx)
}

// buildAgent creates the appropriate agent based on config.
func buildAgent(backend gmanconfig.Backend, tools []domain.Tool) domain.Agent {
	switch backend.Provider {
	case "ollama", "":
		url := backend.OllamaURL
		if url == "" {
			url = "http://localhost:11434"
		}
		model := backend.Model
		if model == "" {
			model = "llama3.2:3b"
		}
		return ollama.NewOllamaClient(model, tools, url)

	default:
		// Remote API provider (openai, deepseek, groq, etc.)
		apiKey := ""
		if backend.APIKeys != nil {
			apiKey = backend.APIKeys[backend.Provider]
		}

		// Determine base URL
		baseURL := ""
		if backend.BaseURLs != nil {
			baseURL = backend.BaseURLs[backend.Provider]
		}
		if baseURL == "" {
			baseURL = openai.DefaultBaseURLs[backend.Provider]
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com" // fallback
		}

		model := backend.Model
		if model == "" {
			model = "gpt-4o"
		}

		return openai.NewClient(apiKey, baseURL, model, tools)
	}
}

// expandHome expands ~ to $HOME.
func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	if path == "~" {
		home, _ := os.UserHomeDir()
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
		"window":      map[string]interface{}{"mode": c.Window.Mode},
	}
}
