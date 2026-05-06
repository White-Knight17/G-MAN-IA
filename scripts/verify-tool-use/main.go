// verify-tool-use connects to Ollama (localhost:11434) and sends 20 structured
// prompts to deepseek-r1:1.5b, testing its ability to produce valid XML tool calls
// in the <tool_call> format. It reports tool-call consistency and logs each
// response for manual inspection.
//
// Usage: go run scripts/verify-tool-use/main.go
package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	ollamaURL = "http://localhost:11434/api/chat"
	model     = "deepseek-r1:1.5b"
)

// ToolCall is the expected XML structure from the model.
type ToolCall struct {
	XMLName xml.Name `xml:"tool_call"`
	Name    string   `xml:"name"`
	Path    string   `xml:"path,omitempty"`
	Content string   `xml:"content,omitempty"`
	Cmd     string   `xml:"cmd,omitempty"`
	Query   string   `xml:"query,omitempty"`
}

// toolDef describes a tool the model can call via XML.
type toolDef struct {
	Name        string
	Description string
	XMLTemplate string
}

var tools = []toolDef{
	{
		Name:        "read_file",
		Description: "Read contents of a file at the given path",
		XMLTemplate: `<tool_call><name>read_file</name><path>/full/path</path></tool_call>`,
	},
	{
		Name:        "write_file",
		Description: "Write content to a file (creates .bak backup)",
		XMLTemplate: `<tool_call><name>write_file</name><path>/full/path</path><content>text</content></tool_call>`,
	},
	{
		Name:        "list_dir",
		Description: "List directory contents at the given path",
		XMLTemplate: `<tool_call><name>list_dir</name><path>/full/path</path></tool_call>`,
	},
	{
		Name:        "run_command",
		Description: "Run an allowlisted command in a sandbox",
		XMLTemplate: `<tool_call><name>run_command</name><cmd>command args</cmd></tool_call>`,
	},
	{
		Name:        "check_syntax",
		Description: "Validate config file syntax",
		XMLTemplate: `<tool_call><name>check_syntax</name><path>/full/path</path></tool_call>`,
	},
	{
		Name:        "search_docs",
		Description: "Search the local Arch Wiki documentation",
		XMLTemplate: `<tool_call><name>search_docs</name><query>search terms</query></tool_call>`,
	},
}

// 20 prompts exercising all 6 tool definitions across varying scenarios.
var prompts = []string{
	"Read the file /home/user/.config/hypr/hyprland.conf",
	"List all files in the directory /home/user/.config/waybar",
	"Run the command: hyprctl monitors",
	"Search the Arch Wiki for 'hyprland window rules'",
	"Validate the syntax of /home/user/.config/kitty/kitty.conf",
	"Write 'include ~/.config/hypr/extra.conf' to /home/user/.config/hypr/hyprland.conf",
	"Read the file /home/user/.config/fish/config.fish",
	"List the directory /home/user/.config/wofi",
	"Run: systemctl --user status pipewire",
	"Search docs for 'pacman hooks'",
	"Check syntax of /home/user/.config/sway/config",
	"Read the PAM configuration at /etc/pam.d/system-auth",
	"List files in /home/user/.config/nvim/lua",
	"Run: journalctl --user -n 5",
	"Write 'bindsym $mod+Return exec kitty' to /home/user/.config/sway/config",
	"Search for 'systemd user units' in the wiki",
	"List directory /home/user/.config/mako",
	"Run: grep -r 'exec' /home/user/.config/hypr/",
	"Validate /home/user/.config/nvim/init.lua syntax",
	"Read /home/user/.config/waybar/style.css",
}

type ollamaReq struct {
	Model    string    `json:"model"`
	Stream   bool      `json:"stream"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResp struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

func buildSystemPrompt() string {
	var sb strings.Builder
	sb.WriteString("You are Harvey, an AI assistant for Arch Linux with Hyprland.\n")
	sb.WriteString("You have access to tools. When you need to use a tool, output EXACTLY this XML format:\n")
	sb.WriteString("<tool_call><name>tool_name</name><param>value</param></tool_call>\n\n")
	sb.WriteString("Available tools:\n")
	for _, t := range tools {
		sb.WriteString(fmt.Sprintf("- %s: %s. Format: %s\n", t.Name, t.Description, t.XMLTemplate))
	}
	sb.WriteString("\nIMPORTANT: Output ONLY the XML tool call, nothing else. No markdown, no backticks.\n")
	return sb.String()
}

func sendPrompt(client *http.Client, sysPrompt, userPrompt string) (string, error) {
	req := ollamaReq{
		Model:  model,
		Stream: false,
		Messages: []message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequest("POST", ollamaURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	var or ollamaResp
	if err := json.Unmarshal(respBody, &or); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return or.Message.Content, nil
}

func containsToolCall(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "<tool_call>") &&
		strings.Contains(lower, "name") &&
		strings.Contains(lower, "</tool_call>")
}

func parseToolCalls(content string) ([]ToolCall, error) {
	// The model may wrap XML in markdown or add extra text.
	// Extract all <tool_call>...</tool_call> blocks.
	type Wrapped struct {
		XMLName xml.Name
		Inner   []byte `xml:",innerxml"`
	}

	var calls []ToolCall
	d := xml.NewDecoder(strings.NewReader(content))

	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		if start, ok := tok.(xml.StartElement); ok && start.Name.Local == "tool_call" {
			var tc ToolCall
			if err := d.DecodeElement(&tc, &start); err != nil {
				continue
			}
			calls = append(calls, tc)
		}
	}

	if len(calls) == 0 {
		return nil, fmt.Errorf("no <tool_call> elements found")
	}
	return calls, nil
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║  Harvey Model Tool-Use Verification             ║")
	fmt.Println("║  Model: deepseek-r1:1.5b  |  Prompts: 20       ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()

	client := &http.Client{Timeout: 60 * time.Second}

	// Health check
	hcResp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		fmt.Printf("❌ Ollama not reachable: %v\n", err)
		return
	}
	hcResp.Body.Close()
	if hcResp.StatusCode != 200 {
		fmt.Printf("❌ Ollama returned status %d\n", hcResp.StatusCode)
		return
	}
	fmt.Println("✓ Ollama reachable at localhost:11434")

	systemPrompt := buildSystemPrompt()

	passed := 0
	validXML := 0
	totalCalls := 0
	startTime := time.Now()

	for i, prompt := range prompts {
		fmt.Printf("\n[%2d/%2d] Prompt: %s\n", i+1, len(prompts), truncate(prompt, 60))

		content, err := sendPrompt(client, systemPrompt, prompt)
		if err != nil {
			fmt.Printf("     ❌ Error: %v\n", err)
			continue
		}

		truncated := truncate(content, 100)
		fmt.Printf("     Response: %s\n", truncated)

		if !containsToolCall(content) {
			fmt.Printf("     ⚠️  No <tool_call> tags detected\n")
			continue
		}

		passed++ // Has tool_call keywords

		calls, err := parseToolCalls(content)
		if err != nil {
			fmt.Printf("     ⚠️  XML parse error: %v\n", err)
			continue
		}

		totalCalls += len(calls)
		validXML++ // Valid XML parsed

		for _, call := range calls {
			fmt.Printf("     ✓ Tool: %s", call.Name)
			switch {
			case call.Path != "":
				fmt.Printf(" | Path: %s", truncate(call.Path, 40))
			case call.Cmd != "":
				fmt.Printf(" | Cmd: %s", truncate(call.Cmd, 40))
			case call.Query != "":
				fmt.Printf(" | Query: %s", truncate(call.Query, 40))
			}
			fmt.Println()
		}
	}

	elapsed := time.Since(startTime)

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  VERIFICATION RESULTS")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Printf("  Model:            %s\n", model)
	fmt.Printf("  Total prompts:    %d\n", len(prompts))
	fmt.Printf("  Has tool_call:    %d/%d (%.0f%%)\n", passed, len(prompts),
		float64(passed)/float64(len(prompts))*100)
	fmt.Printf("  Valid XML:        %d/%d (%.0f%%)\n", validXML, len(prompts),
		float64(validXML)/float64(len(prompts))*100)
	fmt.Printf("  Total tool calls: %d\n", totalCalls)
	fmt.Printf("  Elapsed:          %s\n", elapsed.Round(time.Millisecond))
	fmt.Println()

	xmlRatio := float64(validXML) / float64(len(prompts))

	switch {
	case xmlRatio >= 0.85:
		fmt.Println("✅ VERDICT: Model passes — ≥85% valid XML. Proceed with domain interfaces.")
	case xmlRatio >= 0.60:
		fmt.Println("⚠️  VERDICT: Marginal — ≥60% but <85%. Consider a larger model or adjust tool prompts.")
	default:
		fmt.Println("❌ VERDICT: Model fails — <60% valid XML. Do NOT proceed with domain interfaces until resolved.")
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
