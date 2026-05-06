// Package main runs model tool-use verification against Ollama-hosted models.
// Sends shorter prompts in non-streaming mode to test XML tool-call capability.
// Usage: go run ./cmd/verify-models/
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
	Error   string        `json:"error,omitempty"`
}

type prompt struct {
	name    string
	content string
}

type result struct {
	name     string
	valid    bool
	latency  time.Duration
	response string
	err      error
}

type modelReport struct {
	model      string
	total      int
	valid      int
	avgLatency time.Duration
	verdict    string
	results    []result
}

// systemPrompt defines the tool-calling instructions in a compact format.
const systemPrompt = `You are an AI assistant. You have tools: read_file, write_file, list_dir, run_command.
When you need to use a tool, respond with a single XML block:
<tool_call>
<name>tool_name</name>
<param>value</param>
</tool_call>
Always use this EXACT format. Only output ONE <tool_call> block.`

// prompts are short, focused requests to minimize generation time.
var prompts = []prompt{
	{name: "read_file", content: "Show me my hyprland config at /home/user/.config/hypr/hyprland.conf"},
	{name: "list_dir", content: "List files in /home/user/.config/hypr/"},
	{name: "write_file", content: "Write 'gaps=5' to /home/user/.config/hypr/hyprland.conf"},
	{name: "run_command", content: "Check connected monitors with hyprctl"},
	{name: "read_file2", content: "Read /home/user/.config/waybar/config"},
	{name: "list_dir2", content: "List the directory /home/user/.config/"},
	{name: "write_file2", content: "Write 'border_size=2' to /home/user/.config/hypr/hyprland.conf"},
	{name: "run_command2", content: "Check waybar status with systemctl --user"},
	{name: "read_file3", content: "Read /home/user/.config/kitty/kitty.conf"},
	{name: "list_dir3", content: "List files in /home/user/.config/kitty/"},
}

func main() {
	var models string
	var timeout int
	flag.StringVar(&models, "models", "llama3.2:3b,qwen2.5:3b,qwen3.5:2b", "comma-separated model names")
	flag.IntVar(&timeout, "timeout", 30, "timeout in seconds per prompt")
	flag.Parse()

	modelList := strings.Split(models, ",")

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  Harvey Model Tool-Use Verification v2")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  Models: %s\n", strings.Join(modelList, ", "))
	fmt.Printf("  Prompts/model: %d, Timeout: %ds, Threshold: >=70%%\n\n", len(prompts), timeout)

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	var reports []modelReport
	allFailed := true

	for _, model := range modelList {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}

		fmt.Printf("--- %s ---\n", model)
		report := testModel(client, model, "http://localhost:11434", time.Duration(timeout)*time.Second)
		reports = append(reports, report)

		pct := float64(report.valid) / float64(report.total) * 100
		fmt.Printf("  %d/%d valid (%.0f%%), avg %s, %s\n\n",
			report.valid, report.total, pct, report.avgLatency.Round(time.Millisecond), report.verdict)
		if report.verdict == "PASS" {
			allFailed = false
		}
	}

	// Summary table
	fmt.Println("=== SUMMARY ===")
	fmt.Printf("%-16s | %-10s | %-12s | %-6s\n", "Model", "Valid/Tot", "Avg Latency", "Verdict")
	fmt.Printf("%s-+-%s-+-%s-+-%s\n", strings.Repeat("-", 16), strings.Repeat("-", 10), strings.Repeat("-", 12), strings.Repeat("-", 6))
	for _, r := range reports {
		fmt.Printf("%-16s | %d/%-9d | %-12s | %-6s\n",
			r.model, r.valid, r.total, r.avgLatency.Round(time.Millisecond).String(), r.verdict)
	}

	// Detail
	fmt.Printf("\n=== DETAILS ===\n")
	for _, r := range reports {
		fmt.Printf("\n%s (%s):\n", r.model, r.verdict)
		for _, res := range r.results {
			s := "[PASS]"
			if !res.valid {
				s = "[FAIL]"
			}
			fmt.Printf("  %s %-20s %-10s", s, res.name, res.latency.Round(time.Millisecond).String())
			if res.err != nil {
				fmt.Printf(" ERR: %v", res.err)
			} else if !res.valid {
				snip := strings.TrimSpace(res.response)
				if len(snip) > 120 {
					snip = snip[:120] + "..."
				}
				fmt.Printf(" -> %q", snip)
			}
			fmt.Println()
		}
	}

	if allFailed {
		fmt.Printf("\n*** ALL MODELS FAILED — STOP. ***\n")
		os.Exit(1)
	}
	fmt.Printf("\n=== ≥1 model passed. Proceed. ===\n")
}

func testModel(client *http.Client, model, baseURL string, timeout time.Duration) modelReport {
	r := modelReport{model: model, total: len(prompts), results: make([]result, len(prompts))}
	var totalLat time.Duration

	for i, p := range prompts {
		lat, resp, err := sendChat(client, baseURL, model, p.content, timeout)
		res := result{name: p.name, latency: lat, response: resp, err: err}
		if err == nil {
			res.valid = isValidToolCall(resp)
		}
		r.results[i] = res
		if res.valid {
			r.valid++
		}
		totalLat += lat

		status := "."
		if res.valid {
			status = "✓"
		} else if err != nil {
			status = "✗"
		} else {
			status = "✗"
		}
		fmt.Printf("  [%s] %s (%s)\n", status, p.name, lat.Round(time.Millisecond))
	}
	if r.total > 0 {
		r.avgLatency = totalLat / time.Duration(r.total)
	}
	if float64(r.valid)/float64(r.total) >= 0.70 {
		r.verdict = "PASS"
	} else {
		r.verdict = "FAIL"
	}
	return r
}

func sendChat(client *http.Client, baseURL, model, userPrompt string, timeout time.Duration) (time.Duration, string, error) {
	req := ollamaChatRequest{
		Model: model,
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream: false,
	}

	body, _ := json.Marshal(req)
	url := strings.TrimRight(baseURL, "/") + "/api/chat"

	start := time.Now()
	httpResp, err := client.Post(url, "application/json", bytes.NewReader(body))
	lat := time.Since(start)

	if err != nil {
		return lat, "", err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, 2*1024*1024))
	if err != nil {
		return lat, "", err
	}

	if httpResp.StatusCode != 200 {
		return lat, "", fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(respBody[:min(len(respBody), 200)]))
	}

	var chatResp ollamaChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return lat, "", fmt.Errorf("parse: %w", err)
	}

	if chatResp.Error != "" {
		return lat, "", fmt.Errorf("ollama: %s", chatResp.Error)
	}

	return lat, chatResp.Message.Content, nil
}

func isValidToolCall(resp string) bool {
	resp = strings.TrimSpace(resp)
	if resp == "" {
		return false
	}

	oi := strings.Index(resp, "<tool_call>")
	ci := strings.Index(resp, "</tool_call>")
	if oi == -1 || ci == -1 || oi >= ci {
		return false
	}

	block := resp[oi : ci+len("</tool_call>")]

	ni := strings.Index(block, "<name>")
	ne := strings.Index(block, "</name>")
	if ni == -1 || ne == -1 || ni >= ne {
		return false
	}

	name := strings.TrimSpace(block[ni+len("<name>") : ne])

	known := map[string]bool{
		"read_file": true, "write_file": true, "list_dir": true, "run_command": true,
	}
	if !known[name] {
		return false
	}

	// Check basic XML well-formedness for key tags
	for _, t := range []string{"tool_call", "name"} {
		if strings.Count(block, "<"+t+">") != strings.Count(block, "</"+t+">") {
			return false
		}
	}

	return true
}
