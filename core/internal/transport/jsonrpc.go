// Package transport implements a JSON-RPC 2.0 server over stdin/stdout
// with NDJSON framing (one JSON object per line). This is the Go sidecar
// transport that communicates with the Tauri shell.
//
// The server reads JSON-RPC requests (and notifications) from stdin,
// dispatches them to registered handlers, and writes responses to stdout.
// Streaming events are written as JSON-RPC notifications via SendNotification.
package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// JSON-RPC 2.0 standard error codes.
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// Request represents a JSON-RPC 2.0 request.
// ID is nil for notifications (no response expected).
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
// Error is nil on success, non-nil on failure.
// ID is a pointer to allow null IDs for parse errors.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int   `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HandlerFunc is the function signature for JSON-RPC method handlers.
// Handlers receive the parsed request and return a response.
// For notifications (nil ID), the response is discarded.
type HandlerFunc func(req Request) Response

// Server is a JSON-RPC 2.0 server that communicates over stdin/stdout
// using NDJSON (Newline-Delimited JSON) framing.
//
// Requests are read line-by-line from stdin. Each line must be a
// complete JSON object. Responses are written as single JSON lines
// to stdout. Streaming events are sent as notifications.
//
// The server is safe for concurrent notification writes while serving.
type Server struct {
	handlers map[string]HandlerFunc
	stdin    *bufio.Scanner
	stdout   *json.Encoder
	stdoutW  io.Writer // kept for Sync/flushing
	mu       sync.Mutex
	maxSize  int
}

// NewServer creates a JSON-RPC server that reads from stdin and writes to stdout.
// stdin and stdout are typically os.Stdin and os.Stdout for the sidecar use case,
// but can be any io.Reader/io.Writer for testing.
const defaultMaxSize = 1 * 1024 * 1024 // 1MB soft limit for request payloads

func NewServer(stdin io.Reader, stdout io.Writer) *Server {
	s := &Server{
		handlers: make(map[string]HandlerFunc),
		stdin:    bufio.NewScanner(stdin),
		stdout:   json.NewEncoder(stdout),
		stdoutW:  stdout,
		maxSize:  defaultMaxSize,
	}
	// Scanner buffer is set larger than maxSize to allow reading oversized
	// lines so we can return proper error responses instead of crashing.
	scannerMax := 100 * 1024 * 1024 // 100MB absolute max
	buf := make([]byte, 0, 64*1024)
	s.stdin.Buffer(buf, scannerMax)
	return s
}

// Handle registers a handler function for the given JSON-RPC method.
// If a handler already exists for the method, it is overwritten.
func (s *Server) Handle(method string, fn HandlerFunc) {
	s.handlers[method] = fn
}

// Serve reads JSON-RPC requests from stdin in a loop and dispatches them
// to registered handlers. It blocks until stdin is closed or the context
// is cancelled. Each line on stdin must be a valid JSON object (single
// request) or JSON array (batch request).
//
// Notifications (requests without an ID) are dispatched but no response
// is written. Errors during parsing generate JSON-RPC error responses
// (for non-notification requests).
func (s *Server) Serve(ctx context.Context) error {
	// Check context before entering the loop
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	for s.stdin.Scan() {
		// Check context cancellation between lines
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := s.stdin.Bytes()

		// Check if input exceeds soft limit
		if len(line) > s.maxSize {
			s.processOversizedLine(line)
			continue
		}

		s.processLine(line)
	}

	if err := s.stdin.Err(); err != nil {
		return fmt.Errorf("stdin read error: %w", err)
	}
	return nil
}

// SendNotification writes a JSON-RPC 2.0 notification (request without ID)
// to stdout as an NDJSON line. Notifications are used for streaming events,
// readiness signals, and permission requests.
//
// The notification is written as:
//
//	{"jsonrpc":"2.0","method":"stream.token","params":{...}}
func (s *Server) SendNotification(method string, params interface{}) error {
	notif := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.stdout.Encode(notif)
}

// processOversizedLine handles lines that exceed the max size limit.
// It attempts to extract the request ID from the oversized payload
// to return a proper error response.
func (s *Server) processOversizedLine(line []byte) {
	// Try to extract the ID from the oversized payload
	var partial struct {
		ID *int `json:"id"`
	}
	_ = json.Unmarshal(line[:min(len(line), 1024)], &partial)
	s.writeError(partial.ID, ParseError, "Request too large")
}

// processLine handles a single NDJSON line from stdin.
// It detects whether the line is a single request or a batch (array).
func (s *Server) processLine(line []byte) {
	trimmed := trimSpace(line)
	if len(trimmed) == 0 {
		return
	}

	// Check if it's a batch (JSON array)
	if trimmed[0] == '[' {
		s.processBatch(trimmed)
		return
	}

	// Single request
	s.processRequest(trimmed)
}

// processRequest parses a single JSON-RPC request, dispatches it,
// and writes the response (if not a notification).
func (s *Server) processRequest(data []byte) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		s.writeError(nil, ParseError, "Parse error")
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		if req.ID != nil {
			s.writeError(req.ID, InvalidRequest, "Invalid Request: jsonrpc must be '2.0'")
		}
		return
	}

	// Find handler
	handler, ok := s.handlers[req.Method]
	if !ok {
		if req.ID != nil {
			s.writeError(req.ID, MethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
		}
		return
	}

	// Dispatch to handler
	resp := handler(req)

	// Notifications (nil ID): no response expected
	if req.ID == nil {
		return
	}

	// Ensure response has correct JSONRPC version and ID
	resp.JSONRPC = "2.0"
	resp.ID = req.ID

	s.writeResponse(resp)
}

// processBatch handles a JSON-RPC 2.0 batch request (array of requests).
// Each element is processed independently. Responses are collected into
// a batch response array. Notifications within the batch produce no
// response element.
func (s *Server) processBatch(data []byte) {
	var requests []json.RawMessage
	if err := json.Unmarshal(data, &requests); err != nil {
		s.writeError(nil, ParseError, "Parse error: invalid batch JSON")
		return
	}

	if len(requests) == 0 {
		s.writeError(nil, InvalidRequest, "Invalid Request: empty batch")
		return
	}

	// Collect responses for non-notification requests
	var responses []Response
	for _, raw := range requests {
		var req Request
		if err := json.Unmarshal(raw, &req); err != nil {
			// For individual parse errors in batch, return error response
			responses = append(responses, newErrorResponse(nil, ParseError, "Parse error"))
			continue
		}

		// Validate JSON-RPC version
		if req.JSONRPC != "2.0" {
			if req.ID != nil {
				responses = append(responses, newErrorResponse(req.ID, InvalidRequest, "Invalid Request"))
			}
			continue
		}

		// Notifications in batch: no response
		if req.ID == nil {
			// Still dispatch the handler for side effects
			if handler, ok := s.handlers[req.Method]; ok {
				handler(req)
			}
			continue
		}

		handler, ok := s.handlers[req.Method]
		if !ok {
			responses = append(responses, newErrorResponse(req.ID, MethodNotFound, fmt.Sprintf("Method not found: %s", req.Method)))
			continue
		}

		resp := handler(req)
		resp.JSONRPC = "2.0"
		resp.ID = req.ID // both are *int
		responses = append(responses, resp)
	}

	// Only write if there are responses (batch could be all notifications)
	if len(responses) > 0 {
		s.writeBatchResponses(responses)
	}
}

// writeResponse writes a single JSON-RPC response as an NDJSON line.
func (s *Server) writeResponse(resp Response) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stdout.Encode(resp)
}

// writeBatchResponses writes a batch response array as an NDJSON line.
func (s *Server) writeBatchResponses(responses []Response) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stdout.Encode(responses)
}

// writeError writes a JSON-RPC error response.
// If id is nil (e.g., parse error where we can't extract the ID),
// the response is written with a null ID as per JSON-RPC 2.0 spec.
// For notifications with a known nil ID, we skip the response.
func (s *Server) writeError(id *int, code int, message string) {
	resp := newErrorResponse(id, code, message)
	s.writeResponse(resp)
}

// newErrorResponse creates a Response with the given error code and message.
// id may be nil (for parse errors where the request ID couldn't be determined).
func newErrorResponse(id *int, code int, message string) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// trimSpace trims whitespace from a byte slice.
func trimSpace(b []byte) []byte {
	start := 0
	end := len(b)
	for start < end && (b[start] == ' ' || b[start] == '\t' || b[start] == '\r' || b[start] == '\n') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t' || b[end-1] == '\r' || b[end-1] == '\n') {
		end--
	}
	return b[start:end]
}
