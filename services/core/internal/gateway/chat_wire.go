package gateway

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

const chatModelPrefix = "forge_"

// legacyChatToolAliases maps older model-facing names to gateway tool IDs.
var legacyChatToolAliases = map[string]string{
	"filesystem_create_directory": "fs.mkdir",
	"filesystem_list_directory":   "fs.list",
	"filesystem_read_file":        "fs.read",
	"filesystem_write_file":       "fs.write",
	"shell_run_scoped":            "proc.run",
	"git_status":                  "git.status",
}

// ChatModelName returns a stable OpenAI-safe function name for a gateway tool ID.
func ChatModelName(toolID string) string {
	return chatModelPrefix + strings.ReplaceAll(toolID, ".", "_")
}

// DefaultChatLane maps a gateway tool ID to a built-in action lane that can carry it.
// Returns ("", false) when the tool is not exposed through chat (no safe lane mapping).
func DefaultChatLane(toolID string) (string, bool) {
	switch toolID {
	case "fs.write":
		return "fs.write.bounded", true
	case "git.commit", "git.checkout", "git.stash", "git.apply_patch":
		return "git.write", true
	case "system.service_status", "system.service_control", "system.logs":
		return "system.privileged", true
	case "net.interfaces", "net.dns_lookup", "net.connectivity", "net.fetch":
		return "network.inspect", true
	case "desktop.notify", "desktop.open":
		return "desktop.session", true
	case "proc.terminate":
		return "proc.run", true
	case "secret.get":
		return "", false
	}
	switch toolID {
	case "fs.read", "fs.list", "fs.mkdir", "repo.inspect",
		"git.status", "git.diff", "git.branch",
		"validate.project_context", "proc.run":
		return toolID, true
	case "fs.rename", "fs.copy", "fs.delete", "fs.chmod":
		return "fs.write.bounded", true
	default:
		return "", false
	}
}

func chatGenericParametersSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Optional single target path (workspace-relative or absolute inside workspace). Prefer this OR paths[], not both, unless the tool needs multiple paths.",
			},
			"paths": map[string]any{
				"type":        "array",
				"description": "Target paths when the tool needs zero or more filesystem locations (order may matter for multi-path tools).",
				"items":       map[string]any{"type": "string"},
			},
			"input": map[string]any{
				"type":                 "object",
				"description":          "Tool-specific fields (e.g. command, contents, timeoutMs, cwd, pid, service name). See the gateway tool description.",
				"additionalProperties": true,
			},
		},
		"additionalProperties": false,
	}
}

// ChatOllamaToolDefs builds one OpenAI-style function definition per gateway tool
// that has a resolvable chat lane.
func (g *Gateway) ChatOllamaToolDefs() []map[string]any {
	infos := g.Tools()
	sort.Slice(infos, func(i, j int) bool { return infos[i].ID < infos[j].ID })
	out := make([]map[string]any, 0, len(infos))
	for _, ti := range infos {
		if _, ok := DefaultChatLane(ti.ID); !ok {
			continue
		}
		model := ChatModelName(ti.ID)
		desc := chatToolDescription(ti)
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        model,
				"description": desc,
				"parameters":  chatGenericParametersSchema(),
			},
		})
	}
	return out
}

func chatToolDescription(ti ToolInfo) string {
	var b strings.Builder
	b.WriteString(ti.Description)
	b.WriteString("\n\n")
	b.WriteString("FORGE gateway tool id: ")
	b.WriteString(ti.ID)
	b.WriteString("\nDomain: ")
	b.WriteString(ti.Domain)
	b.WriteString(" · action: ")
	b.WriteString(ti.Action)
	b.WriteString("\nRisk: ")
	b.WriteString(ti.RiskClass)
	b.WriteString(" · execution level: ")
	b.WriteString(ti.ExecutionLevel)
	if ti.WriteIntent {
		b.WriteString(" · write intent: yes")
	} else {
		b.WriteString(" · write intent: no")
	}
	if ti.Executes {
		b.WriteString(" · executes subprocess: yes")
	}
	if ti.UsesNetwork {
		b.WriteString(" · uses network: yes")
	}
	b.WriteString("\n\nInvoke with JSON arguments { \"path\"?: string, \"paths\"?: string[], \"input\"?: object }.")
	b.WriteString(" Put tool-specific fields inside \"input\" (e.g. {\"command\":\"go version\"} for proc.run, {\"contents\":\"...\"} for fs.write).")
	b.WriteString(" Paths are workspace-relative unless absolute and still inside the workspace.")
	if ti.ID == "desktop.open" {
		b.WriteString(" For desktop.open use one of: path/paths for files, input.url for URLs, or input.application for desktop apps (e.g. {\"input\":{\"application\":\"konsole\"}}).")
	}
	return b.String()
}

// ChatToolManifests returns a JSON-serializable manifest of tools attached to chat.
func (g *Gateway) ChatToolManifests() []map[string]any {
	infos := g.Tools()
	sort.Slice(infos, func(i, j int) bool { return infos[i].ID < infos[j].ID })
	out := make([]map[string]any, 0, len(infos))
	for _, ti := range infos {
		lane, ok := DefaultChatLane(ti.ID)
		if !ok {
			continue
		}
		out = append(out, map[string]any{
			"modelName":     ChatModelName(ti.ID),
			"gatewayToolId": ti.ID,
			"gatewayLaneId": lane,
			"domain":        ti.Domain,
			"action":        ti.Action,
			"description":   ti.Description,
			"riskClass":     ti.RiskClass,
			"writeIntent":   ti.WriteIntent,
		})
	}
	return out
}

// ChatToolModelNames lists model function names attached to chat (forge_*).
func (g *Gateway) ChatToolModelNames() []string {
	infos := g.Tools()
	out := make([]string, 0, len(infos))
	for _, ti := range infos {
		if _, ok := DefaultChatLane(ti.ID); !ok {
			continue
		}
		out = append(out, ChatModelName(ti.ID))
	}
	sort.Strings(out)
	return out
}

// ChatSystemSupplement documents dynamic chat tools for the operator model.
func (g *Gateway) ChatSystemSupplement() string {
	names := g.ChatToolModelNames()
	head := `TOOLING (FORGE gateway)
Every registered gateway tool that has a chat lane is exposed as an OpenAI-style function.
Function names are ` + "`forge_<toolId with dots as underscores>`" + ` (example: ` + "`forge_fs_mkdir`" + ` for tool id ` + "`fs.mkdir`" + `).
Legacy names like ` + "`filesystem_create_directory`" + ` are still accepted.

Arguments for every tool:
- Optional ` + "`path`" + ` (string) OR ` + "`paths`" + ` (string[]) for filesystem/git targets.
- Optional ` + "`input`" + ` (object) for tool-specific fields (command, contents, timeoutMs, cwd, etc.).

Execution always goes through gateway.Execute with permission profiles and action lanes.
If the gateway denies a call or a tool is not listed, say so plainly — never invent results.
Tools not listed in the manifest are not callable from this chat runtime.
`
	if len(names) == 0 {
		return head + "\nNo gateway tools are currently exposed to chat (lane mapping missing)."
	}
	b, _ := json.MarshalIndent(names, "", "  ")
	return head + "\nAttached tools (" + strconv.Itoa(len(names)) + "): \n```json\n" + string(b) + "\n```\n"
}

// ResolveChatFunctionName maps an OpenAI function name (or legacy alias) to gateway tool and lane.
func (g *Gateway) ResolveChatFunctionName(name string) (toolID, laneID string, ok bool) {
	if id, hit := legacyChatToolAliases[strings.TrimSpace(name)]; hit {
		lane, ok2 := DefaultChatLane(id)
		if !ok2 || g.tools[id] == nil {
			return "", "", false
		}
		return id, lane, true
	}
	if !strings.HasPrefix(name, chatModelPrefix) {
		return "", "", false
	}
	suf := strings.TrimPrefix(name, chatModelPrefix)
	for id := range g.tools {
		if strings.ReplaceAll(id, ".", "_") == suf {
			lane, ok2 := DefaultChatLane(id)
			if !ok2 {
				return "", "", false
			}
			return id, lane, true
		}
	}
	return "", "", false
}
