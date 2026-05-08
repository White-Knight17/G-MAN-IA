package transport_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gentleman/gman/internal/transport"
)

// =============================================================================
// Request Parsing Tests
// =============================================================================

func TestParseValidRequest(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	server.Handle("ping", func(req transport.Request) transport.Response {
		return transport.Response{JSONRPC: "2.0", ID: req.ID, Result: "pong"}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	resps := readResponses(t, out)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	resp := resps[0]
	if resp.Result != "pong" {
		t.Errorf("expected result 'pong', got %v", resp.Result)
	}
	if resp.ID == nil || *resp.ID != 1 {
		t.Errorf("expected ID 1, got %v", resp.ID)
	}
	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got %q", resp.JSONRPC)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	input := `this is not json` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	resps := readResponses(t, out)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	resp := resps[0]
	if resp.Error == nil {
		t.Fatal("expected error response for invalid JSON")
	}
	if resp.Error.Code != transport.ParseError {
		t.Errorf("expected ParseError code %d, got %d", transport.ParseError, resp.Error.Code)
	}
}

func TestMissingJSONRPCVersion(t *testing.T) {
	input := `{"id":1,"method":"ping"}` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	resps := readResponses(t, out)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	resp := resps[0]
	if resp.Error == nil {
		t.Fatal("expected error response for missing jsonrpc version")
	}
	if resp.Error.Code != transport.InvalidRequest {
		t.Errorf("expected InvalidRequest code %d, got %d", transport.InvalidRequest, resp.Error.Code)
	}
}

func TestInvalidJSONRPCVersion(t *testing.T) {
	input := `{"jsonrpc":"1.0","id":1,"method":"ping"}` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	resps := readResponses(t, out)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	resp := resps[0]
	if resp.Error == nil {
		t.Fatal("expected error response for wrong jsonrpc version")
	}
	if resp.Error.Code != transport.InvalidRequest {
		t.Errorf("expected InvalidRequest code %d, got %d", transport.InvalidRequest, resp.Error.Code)
	}
}

func TestNotificationNoResponse(t *testing.T) {
	input := `{"jsonrpc":"2.0","method":"log","params":{"msg":"hello"}}` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	var logged string
	server.Handle("log", func(req transport.Request) transport.Response {
		var params struct{ Msg string `json:"msg"` }
		json.Unmarshal(req.Params, &params)
		logged = params.Msg
		return transport.Response{}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	// Verify handler was called
	if logged != "hello" {
		t.Errorf("expected handler to log 'hello', got %q", logged)
	}

	// Verify NO response was written
	resps := readResponses(t, out)
	if len(resps) > 0 {
		t.Errorf("expected no response for notification, got %d responses", len(resps))
	}
}

func TestBatchRequests(t *testing.T) {
	input := `[{"jsonrpc":"2.0","id":1,"method":"ping","params":{}},{"jsonrpc":"2.0","id":2,"method":"ping","params":{}}]` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	server.Handle("ping", func(req transport.Request) transport.Response {
		return transport.Response{JSONRPC: "2.0", ID: req.ID, Result: "pong"}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	resps := readResponses(t, out)
	if len(resps) != 2 {
		t.Fatalf("expected 2 responses in batch, got %d", len(resps))
	}
	if *resps[0].ID != 1 {
		t.Errorf("expected ID 1 in first response, got %d", resps[0].ID)
	}
	if *resps[1].ID != 2 {
		t.Errorf("expected ID 2 in second response, got %d", resps[1].ID)
	}
}

func TestBatchWithNotifications(t *testing.T) {
	input := `[{"jsonrpc":"2.0","method":"log","params":{"msg":"test"}},{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}]` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	pinged := false
	server.Handle("log", func(req transport.Request) transport.Response {
		return transport.Response{}
	})
	server.Handle("ping", func(req transport.Request) transport.Response {
		pinged = true
		return transport.Response{JSONRPC: "2.0", ID: req.ID, Result: "pong"}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	if !pinged {
		t.Error("expected ping handler to be called")
	}

	resps := readResponses(t, out)
	// Only 1 response (notification in batch excluded)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response (notification excluded), got %d", len(resps))
	}
	if *resps[0].ID != 1 {
		t.Errorf("expected ID 1, got %d", resps[0].ID)
	}
}

func TestUnknownMethod(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"nonexistent","params":{}}` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	resps := readResponses(t, out)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	resp := resps[0]
	if resp.Error == nil {
		t.Fatal("expected error response for unknown method")
	}
	if resp.Error.Code != transport.MethodNotFound {
		t.Errorf("expected MethodNotFound code %d, got %d", transport.MethodNotFound, resp.Error.Code)
	}
}

func TestLargePayload(t *testing.T) {
	// Create a payload larger than 1MB
	largeContent := strings.Repeat("x", 2*1024*1024) // 2MB
	input := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{"data":"` + largeContent + `"}}` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	server.Handle("ping", func(req transport.Request) transport.Response {
		return transport.Response{JSONRPC: "2.0", ID: req.ID, Result: "pong"}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	resps := readResponses(t, out)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	resp := resps[0]
	if resp.Error == nil {
		t.Fatal("expected error response for large payload")
	}
	if resp.Error.Code != transport.ParseError {
		t.Errorf("expected ParseError for oversized payload, got code %d", resp.Error.Code)
	}
}

func TestMultipleNDJSONLines(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"echo","params":{"text":"hello"}}` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	server.Handle("ping", func(req transport.Request) transport.Response {
		return transport.Response{JSONRPC: "2.0", ID: req.ID, Result: "pong"}
	})
	server.Handle("echo", func(req transport.Request) transport.Response {
		var params struct{ Text string `json:"text"` }
		json.Unmarshal(req.Params, &params)
		return transport.Response{JSONRPC: "2.0", ID: req.ID, Result: params.Text}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	resps := readResponses(t, out)
	if len(resps) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(resps))
	}
	if resps[0].Result != "pong" {
		t.Errorf("first response: expected 'pong', got %v", resps[0].Result)
	}
	if resps[1].Result != "hello" {
		t.Errorf("second response: expected 'hello', got %v", resps[1].Result)
	}
}

// =============================================================================
// Stream / Notification Writing Tests
// =============================================================================

func TestSendNotification(t *testing.T) {
	server, out := newTestServer(strings.NewReader(""))

	err := server.SendNotification("stream.token", map[string]string{
		"token": "Hello",
	})
	if err != nil {
		t.Fatalf("SendNotification failed: %v", err)
	}

	// Read the notification from stdout buffer
	resps := readResponses(t, out)
	if len(resps) == 0 {
		t.Fatal("expected at least one line in output")
	}
	// The notification is not a Response type, so readResponses won't parse it.
	// Read raw output instead.
	output := out.Buf.String()
	t.Logf("raw output: %q", output)

	var notif struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Token string `json:"token"`
		} `json:"params"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &notif); err != nil {
		t.Fatalf("failed to parse notification: %q: %v", output, err)
	}
	if notif.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %q", notif.JSONRPC)
	}
	if notif.Method != "stream.token" {
		t.Errorf("expected method 'stream.token', got %q", notif.Method)
	}
	if notif.Params.Token != "Hello" {
		t.Errorf("expected token 'Hello', got %q", notif.Params.Token)
	}
}

func TestHandlerReturnsError(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"fail","params":{}}` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	server.Handle("fail", func(req transport.Request) transport.Response {
		return transport.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &transport.Error{Code: -32000, Message: "something went wrong"},
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	resps := readResponses(t, out)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	resp := resps[0]
	if resp.Error == nil {
		t.Fatal("expected error response")
	}
	if resp.Error.Code != -32000 {
		t.Errorf("expected error code -32000, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "something went wrong" {
		t.Errorf("expected error message 'something went wrong', got %q", resp.Error.Message)
	}
}

func TestParamsParsing(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"greet","params":{"name":"World","count":42}}` + "\n"
	server, out := newTestServer(strings.NewReader(input))

	server.Handle("greet", func(req transport.Request) transport.Response {
		var params struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return transport.Response{
				JSONRPC: "2.0", ID: req.ID,
				Error: &transport.Error{Code: transport.InvalidParams, Message: err.Error()},
			}
		}
		return transport.Response{
			JSONRPC: "2.0", ID: req.ID,
			Result: map[string]interface{}{"greeting": params.Name, "count": params.Count},
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	resps := readResponses(t, out)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	resp := resps[0]
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp.Result)
	}
	if result["greeting"] != "World" {
		t.Errorf("expected greeting 'World', got %v", result["greeting"])
	}
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

func TestServeContextCancel(t *testing.T) {
	// stdin closes immediately (empty), cancel before Serve starts
	server, _ := newTestServer(strings.NewReader(""))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Serve

	err := server.Serve(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// =============================================================================
// Helpers
// =============================================================================

// testOutput captures server stdout and provides post-hoc line reading.
type testOutput struct {
	Buf *bytes.Buffer
}

func newTestOutput() *testOutput {
	return &testOutput{Buf: new(bytes.Buffer)}
}

func (to *testOutput) Write(p []byte) (int, error) {
	return to.Buf.Write(p)
}

// newTestServer creates a Server with buffer-based stdout for testing.
func newTestServer(stdin io.Reader) (*transport.Server, *testOutput) {
	out := newTestOutput()
	server := transport.NewServer(stdin, out)
	return server, out
}

// readResponses reads all JSON-RPC response objects from the test output buffer.
// It handles both single Response objects and batch arrays.
func readResponses(t *testing.T, to *testOutput) []transport.Response {
	t.Helper()
	var responses []transport.Response
	raw := strings.TrimSpace(to.Buf.String())
	if raw == "" {
		return responses
	}

	// Try as single line containing batch array first
	if strings.HasPrefix(raw, "[") {
		var batch []transport.Response
		if err := json.Unmarshal([]byte(raw), &batch); err == nil {
			return batch
		}
	}

	// Read line by line
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var resp transport.Response
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			// Might be a notification (no "id" field) — skip for response parsing
			continue
		}
		responses = append(responses, resp)
	}
	return responses
}
