package api

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"forge/projectforge/services/core/internal/gateway"
)

type toolDispatchResult struct {
	args            map[string]any
	state           string
	text            string
	failureReason   string
	executionResult any
}

func normalizeChatInvokeArgs(args map[string]any) (paths []string, input map[string]any) {
	input = map[string]any{}
	if raw, ok := args["input"]; ok {
		switch typed := raw.(type) {
		case map[string]any:
			for k, v := range typed {
				input[k] = v
			}
		case string:
			s := strings.TrimSpace(typed)
			if s != "" {
				input["query"] = s
			}
		case fmt.Stringer:
			s := strings.TrimSpace(typed.String())
			if s != "" {
				input["query"] = s
			}
		}
	}
	if p := strings.TrimSpace(stringArg(args, "path")); p != "" {
		paths = append(paths, normalizeChatPathAlias(p))
	}
	if raw, ok := args["paths"]; ok {
		switch typed := raw.(type) {
		case []any:
			for _, x := range typed {
				if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
					paths = append(paths, normalizeChatPathAlias(s))
				}
			}
		case []string:
			for _, s := range typed {
				if strings.TrimSpace(s) != "" {
					paths = append(paths, normalizeChatPathAlias(s))
				}
			}
		}
	}
	reserved := map[string]bool{"path": true, "paths": true, "input": true}
	for k, v := range args {
		if reserved[k] {
			continue
		}
		if _, exists := input[k]; !exists {
			input[k] = v
		}
	}
	return paths, input
}

func normalizeChatPathAlias(raw string) string {
	p := filepath.ToSlash(strings.TrimSpace(raw))
	p = strings.Trim(p, `"'`)
	if p == "" || strings.HasPrefix(p, "~") {
		return p
	}
	trimmed := strings.TrimLeft(p, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || !strings.EqualFold(parts[0], "downloads") {
		return p
	}
	out := []string{"~", "Downloads"}
	if len(parts) > 1 {
		out = append(out, parts[1:]...)
	}
	return filepath.ToSlash(filepath.Join(out...))
}

func (s *Server) dispatchToolCall(ctx context.Context, corr string, threadID int64, functionName, argsStr, lastUserContent string, pushStage func(string, map[string]any)) toolDispatchResult {
	toolID, laneID, ok := s.gateway.ResolveChatFunctionName(functionName)
	if !ok {
		pushStage("resolve_failed", map[string]any{"function": functionName})
		return toolDispatchResult{
			args:          map[string]any{},
			state:         "error",
			text:          fmt.Sprintf("Unknown function %q.", functionName),
			failureReason: "not in gateway chat catalog",
		}
	}

	var args map[string]any
	if strings.TrimSpace(argsStr) != "" {
		if len(argsStr) > chatToolArgumentMaxBytes {
			pushStage("tool_args_too_large", map[string]any{
				"bytes": len(argsStr),
				"max":   chatToolArgumentMaxBytes,
			})
			return toolDispatchResult{
				args:          map[string]any{},
				state:         "error",
				text:          "Tool call arguments were too large.",
				failureReason: fmt.Sprintf("tool arguments too large: %d > %d bytes", len(argsStr), chatToolArgumentMaxBytes),
				executionResult: map[string]any{
					"argumentBytes": len(argsStr),
					"maxBytes":      chatToolArgumentMaxBytes,
				},
			}
		}
		if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
			pushStage("tool_args_error", map[string]any{"error": err.Error()})
			return toolDispatchResult{
				args:          map[string]any{},
				state:         "error",
				text:          "Tool call had invalid argument JSON.",
				failureReason: "invalid JSON: " + err.Error(),
			}
		}
	}
	if args == nil {
		args = map[string]any{}
	}

	paths, input := normalizeChatInvokeArgs(args)
	input = enrichDesktopOpenInputFromUser(toolID, input, lastUserContent)

	if toolID == "fs.write" && gateway.IsVideoGameJournalWebpageIntent(lastUserContent) && len(paths) > 0 && !isHTMLWritePath(paths[0]) {
		pushStage("stale_write_path_rejected", map[string]any{
			"path":   paths[0],
			"reason": "webpage request cannot write to non-HTML path",
		})
		return toolDispatchResult{
			args:          args,
			state:         "denied",
			text:          "Refused: this request asks for a webpage, but the model selected a non-HTML path. No file was written.",
			failureReason: "webpage request selected non-HTML path",
			executionResult: map[string]any{
				"rejectedPath": paths[0],
				"reason":       "webpage request selected non-HTML path",
			},
		}
	}

	if toolID == "fs.mkdir" && gateway.IsCompositeFilesystemWorkflow(lastUserContent) {
		path := ""
		if len(paths) > 0 {
			path = paths[0]
		}
		pushStage("composite_mkdir_suppressed", map[string]any{
			"path":   path,
			"reason": "parent directory creation deferred to approved fs.write",
		})
		return toolDispatchResult{
			args:  args,
			state: "ok",
			text:  "Skipped standalone mkdir: this request also creates or writes a file. Parent directory creation is deferred to fs.write so approval covers the whole filesystem operation.",
			executionResult: map[string]any{
				"suppressed": true,
				"toolId":     toolID,
				"path":       path,
				"reason":     "composite filesystem workflow; mkdir deferred to fs.write",
			},
		}
	}

	if toolID == "fs.read" && len(paths) > 0 {
		if content, meta, ok, err := s.resolveThreadAttachmentRead(ctx, threadID, paths[0]); err != nil {
			pushStage("attachment_read_error", map[string]any{"error": err.Error()})
		} else if ok {
			pushStage("attachment_read_resolved", map[string]any{"artifactId": meta["artifactId"], "path": meta["path"]})
			text := fmt.Sprintf("Attachment %v (%v bytes):\n```\n%s\n```", meta["path"], meta["size"], content)
			return toolDispatchResult{args: args, state: "ok", text: text, executionResult: meta}
		}
	}

	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if !pathAllowed(s.cfg.WorkspaceDir, p) {
			pushStage("path_precheck_failed", map[string]any{"path": p})
			return toolDispatchResult{
				args:          args,
				state:         "denied",
				text:          "Refused: every path must resolve inside the configured workspace.",
				failureReason: "path rejected (outside workspace or traversal)",
			}
		}
	}

	if toolID == "fs.list" && len(paths) == 0 {
		paths = []string{"."}
	}

	gwReq := gateway.Request{
		ToolID:        toolID,
		LaneID:        laneID,
		Domain:        domainForTool(toolID),
		Action:        "invoke",
		CorrelationID: corr,
		Paths:         paths,
		Input:         input,
		Initiator:     "chat",
		DryRun:        false,
		Metadata: map[string]any{
			"chatUserRequest": strings.TrimSpace(lastUserContent),
		},
	}

	pushStage("backend_dispatch", map[string]any{"toolId": gwReq.ToolID, "laneId": gwReq.LaneID, "paths": gwReq.Paths})

	res, gerr := s.gateway.Execute(ctx, gwReq)
	if gerr != nil {
		pushStage("gateway_error", map[string]any{"error": gerr.Error()})
		return toolDispatchResult{args: args, state: "error", text: "Gateway error: " + gerr.Error(), failureReason: gerr.Error()}
	}
	if res == nil {
		return toolDispatchResult{args: args, state: "error", text: "Gateway returned no result.", failureReason: "nil gateway result"}
	}

	resMap, _ := json.Marshal(res)
	pushStage("execution_result", map[string]any{"status": res.Status, "json": trimSummary(string(resMap), 8000)})

	if res.Status == gateway.StatusOK {
		text := formatToolResult(toolID, res)
		return toolDispatchResult{args: args, state: "ok", text: text, executionResult: res.Data}
	}

	reason := strings.TrimSpace(res.DeniedReason)
	if reason == "" {
		reason = strings.TrimSpace(res.Message)
	}

	outData := map[string]any{}
	if res.Data != nil {
		for k, v := range res.Data {
			outData[k] = v
		}
	}
	outData["gatewayStatus"] = res.Status
	outData["policyOutcome"] = res.PolicyOutcome

	var text string
	switch res.Status {
	case gateway.StatusNeedsApprov:
		text = fmt.Sprintf("This action needs operator approval before it can run: %s", reason)
		if v, ok := outData["approvalRequestId"]; ok && v != nil {
			text += fmt.Sprintf(" (approval request #%v)", v)
		}
	case gateway.StatusDenied:
		text = fmt.Sprintf("Gateway denied this action (policy): %s", reason)
	default:
		text = fmt.Sprintf("Gateway %s: %s", res.Status, reason)
	}

	return toolDispatchResult{
		args:            args,
		state:           res.Status,
		text:            text,
		failureReason:   reason,
		executionResult: outData,
	}
}

func (s *Server) resolveThreadAttachmentRead(ctx context.Context, threadID int64, requestedPath string) (content string, meta map[string]any, ok bool, err error) {
	query := strings.TrimSpace(requestedPath)
	if query == "" {
		return "", nil, false, nil
	}
	queryBase := filepath.Base(query)
	th, err := s.chat.GetThread(ctx, threadID)
	if err != nil {
		return "", nil, false, err
	}
	seen := map[int64]struct{}{}
	for i := len(th.Messages) - 1; i >= 0; i-- {
		msg := th.Messages[i]
		for _, artifactID := range messageAttachmentIDs(msg.Metadata) {
			if artifactID <= 0 {
				continue
			}
			if _, exists := seen[artifactID]; exists {
				continue
			}
			seen[artifactID] = struct{}{}
			art, getErr := s.artifacts.GetByID(ctx, artifactID)
			if getErr != nil {
				continue
			}
			fileName := strings.TrimSpace(filepath.Base(art.FilePath))
			title := strings.TrimSpace(art.Title)
			if !strings.EqualFold(query, fileName) && !strings.EqualFold(queryBase, fileName) &&
				!strings.EqualFold(query, title) && !strings.EqualFold(queryBase, title) {
				continue
			}
			text, _, textual, readErr := s.artifacts.ReadArtifactText(ctx, artifactID)
			if readErr != nil {
				return "", nil, false, readErr
			}
			if !textual {
				return "", nil, false, fmt.Errorf("attachment %d is not textual", artifactID)
			}
			if len(text) > 4000 {
				text = text[:4000] + "\n… (truncated)"
			}
			return text, map[string]any{
				"path":       art.FilePath,
				"size":       len(text),
				"bytes":      len(text),
				"text":       text,
				"artifactId": artifactID,
				"source":     "chat_attachment",
			}, true, nil
		}
	}
	return "", nil, false, nil
}

func enrichDesktopOpenInputFromUser(toolID string, input map[string]any, lastUserContent string) map[string]any {
	if toolID != "desktop.open" {
		return input
	}
	text := strings.TrimSpace(lastUserContent)
	if text == "" {
		return input
	}
	if input == nil {
		input = map[string]any{}
	}
	if v, ok := input["query"].(string); ok && strings.TrimSpace(v) != "" {
		return input
	}
	input["query"] = text
	return input
}
