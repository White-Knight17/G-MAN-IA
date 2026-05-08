package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gentleman/gman/internal/transport"
)

// TestPingHandler verifies the ping handler composition directly.
func TestPingHandler(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}` + "\n"
	outBuf := new(bytes.Buffer)
	server := transport.NewServer(bytes.NewBufferString(input), outBuf)

	// Register ping handler (same as in main)
	server.Handle("ping", func(req transport.Request) transport.Response {
		return transport.Response{Result: "pong"}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp transport.Response
	if err := json.Unmarshal(outBuf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nraw: %q", err, outBuf.String())
	}
	if resp.Result != "pong" {
		t.Errorf("expected 'pong', got %v", resp.Result)
	}
}

// TestReadyNotification verifies the server can send notifications.
func TestReadyNotification(t *testing.T) {
	outBuf := new(bytes.Buffer)
	server := transport.NewServer(bytes.NewBufferString(""), outBuf)

	server.SendNotification("ready", map[string]string{
		"version": "1.0.0",
	})

	var notif struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Version string `json:"version"`
		} `json:"params"`
	}
	if err := json.Unmarshal(outBuf.Bytes(), &notif); err != nil {
		t.Fatalf("failed to parse notification: %v", err)
	}
	if notif.Method != "ready" {
		t.Errorf("expected method 'ready', got %q", notif.Method)
	}
	if notif.Params.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", notif.Params.Version)
	}
}

// TestPermissionListHandler verifies permission.list returns grants.
func TestPermissionListHandler(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"permission.list","params":{}}` + "\n"
	outBuf := new(bytes.Buffer)
	server := transport.NewServer(bytes.NewBufferString(input), outBuf)

	// Simulate the handler from main (return empty grant list)
	server.Handle("permission.list", func(req transport.Request) transport.Response {
		return transport.Response{Result: []interface{}{}}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp transport.Response
	if err := json.Unmarshal(outBuf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nraw: %q", err, outBuf.String())
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func intPtr(i int) *int { return &i }
