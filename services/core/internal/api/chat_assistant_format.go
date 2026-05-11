package api

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"forge/projectforge/services/core/internal/gateway"
)

func formatToolResult(gatewayToolID string, res *gateway.Result) string {
	switch gatewayToolID {
	case "fs.mkdir":
		return fmt.Sprintf("Directory created at %v", res.Data["path"])
	case "fs.list":
		count := res.Data["count"]
		path := res.Data["path"]
		entries, _ := json.MarshalIndent(res.Data["entries"], "", "  ")
		return fmt.Sprintf("Listed %v entries in %v:\n```json\n%s\n```", count, path, string(entries))
	case "fs.read":
		path := res.Data["path"]
		text, _ := res.Data["text"].(string)
		size := res.Data["size"]
		if len(text) > 4000 {
			text = text[:4000] + "\n… (truncated)"
		}
		return fmt.Sprintf("File %v (%v bytes):\n```\n%s\n```", path, size, text)
	case "fs.write":
		if files, ok := res.Data["files"]; ok {
			count := res.Data["count"]
			bytes := res.Data["bytes"]
			encoded, _ := json.MarshalIndent(files, "", "  ")
			if len(encoded) > 0 {
				return fmt.Sprintf("Wrote %v files (%v bytes):\n```json\n%s\n```", count, bytes, string(encoded))
			}
			return fmt.Sprintf("Wrote %v files (%v bytes)", count, bytes)
		}
		return fmt.Sprintf("Wrote %v bytes to %v", res.Data["bytes"], res.Data["path"])
	case "proc.run":
		stdout, _ := res.Data["stdout"].(string)
		stderr, _ := res.Data["stderr"].(string)
		exitCode := res.Data["exitCode"]
		cmd := res.Data["command"]
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Command: %v\nExit code: %v", cmd, exitCode))
		if strings.TrimSpace(stdout) != "" {
			out := stdout
			if len(out) > 4000 {
				out = out[:4000] + "\n… (truncated)"
			}
			b.WriteString(fmt.Sprintf("\n\nStdout:\n```\n%s\n```", out))
		}
		if strings.TrimSpace(stderr) != "" {
			errOut := stderr
			if len(errOut) > 2000 {
				errOut = errOut[:2000] + "\n… (truncated)"
			}
			b.WriteString(fmt.Sprintf("\n\nStderr:\n```\n%s\n```", errOut))
		}
		return b.String()
	case "net.fetch":
		body, _ := res.Data["body"].(string)
		urlValue := res.Data["url"]
		statusCode := res.Data["statusCode"]
		if len(body) > 4000 {
			body = body[:4000] + "\n... (truncated)"
		}
		return fmt.Sprintf("Fetched %v (status %v):\n```html\n%s\n```", urlValue, statusCode, body)
	case "web.search":
		query := res.Data["query"]
		results, _ := res.Data["results"].([]map[string]any)
		if results == nil {
			if raw, ok := res.Data["results"].([]any); ok {
				results = make([]map[string]any, 0, len(raw))
				for _, item := range raw {
					if rec, ok := item.(map[string]any); ok {
						results = append(results, rec)
					}
				}
			}
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Web search for %q returned %d result(s).", query, len(results)))
		for i, result := range results {
			title, _ := result["title"].(string)
			urlValue, _ := result["url"].(string)
			snippet, _ := result["snippet"].(string)
			b.WriteString(fmt.Sprintf("\n\n%d. %s\n%s", i+1, strings.TrimSpace(title), strings.TrimSpace(urlValue)))
			if strings.TrimSpace(snippet) != "" {
				b.WriteString("\n")
				b.WriteString(strings.TrimSpace(snippet))
			}
		}
		return b.String()
	case "desktop.open":
		if target, ok := res.Data["target"].(string); ok && strings.TrimSpace(target) != "" {
			return fmt.Sprintf("Opened desktop target: %s", target)
		}
		if urlValue, ok := res.Data["url"].(string); ok && strings.TrimSpace(urlValue) != "" {
			return fmt.Sprintf("Opened browser URL: %s", urlValue)
		}
		return "Desktop open request completed."
	case "git.status":
		output, _ := res.Data["output"].(string)
		available, _ := res.Data["available"].(bool)
		if !available {
			return "This directory is not a git repository."
		}
		return fmt.Sprintf("```\n%s\n```", strings.TrimSpace(output))
	default:
		resJSON, _ := json.MarshalIndent(res.Data, "", "  ")
		return fmt.Sprintf("Gateway ok: %s\n```json\n%s\n```", res.Message, string(resJSON))
	}
}

func pathAllowed(workspace, userPath string) bool {
	wsAbs, err := filepath.Abs(workspace)
	if err != nil {
		return false
	}
	wsAbs = filepath.Clean(wsAbs)
	p := strings.TrimSpace(userPath)
	if p == "" {
		return false
	}
	var target string
	if filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
		target = filepath.Clean(p)
	} else {
		target = filepath.Clean(filepath.Join(wsAbs, p))
	}
	tAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	tAbs = filepath.Clean(tAbs)
	if tAbs == wsAbs {
		return true
	}
	rel, err := filepath.Rel(wsAbs, tAbs)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, "..") {
		return false
	}
	return !strings.HasPrefix(rel, string(filepath.Separator))
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, ok := args[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func domainForTool(toolID string) string {
	switch {
	case strings.HasPrefix(toolID, "fs."):
		return "filesystem"
	case strings.HasPrefix(toolID, "git."):
		return "git"
	case strings.HasPrefix(toolID, "proc."):
		return "process"
	case strings.HasPrefix(toolID, "net."):
		return "network"
	case strings.HasPrefix(toolID, "system."):
		return "system"
	case strings.HasPrefix(toolID, "desktop."):
		return "desktop"
	case toolID == "secret.get":
		return "secret"
	default:
		return "general"
	}
}

// collectToolCallsFromOllamaMessage normalizes Ollama / OpenAI chat message shapes into tool call records.
func collectToolCallsFromOllamaMessage(msg map[string]any) []map[string]any {
	if msg == nil {
		return nil
	}
	var out []map[string]any
	if tc, ok := msg["tool_calls"].([]any); ok {
		for _, item := range tc {
			if rec, ok := item.(map[string]any); ok {
				out = append(out, rec)
			}
		}
	}
	if len(out) == 0 {
		if fc, ok := msg["function_call"].(map[string]any); ok {
			out = append(out, map[string]any{"function": fc})
		}
	}
	return out
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func trimSummary(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
