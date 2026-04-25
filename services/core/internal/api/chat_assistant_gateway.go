package api

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/gateway"
)

func (s *Server) buildChatLLMMessages(ctx context.Context, th *chat.ThreadDetail) (system string, user string) {
	transcript := s.chat.BuildTranscript(th.Messages, chatTranscriptTurns)
	sys := s.chatOperatorSystemPrompt()
	if s.gateway != nil {
		sys += "\n\n" + s.gateway.ChatSystemSupplement()
	}
	user = "---\nTHREAD TITLE: " + th.Title + "\n---\n" + transcript
	if att := s.buildThreadAttachmentContext(ctx, th); att != "" {
		user += "\n\n---\nATTACHMENTS CONTEXT\n" + att
	}
	return sys, user
}

func (s *Server) completeAssistantWithGatewayTools(
	ctx context.Context,
	threadID, userMessageID int64,
	th *chat.ThreadDetail,
	lastUserContent string,
	ollamaAdapter adapters.Adapter,
	dryRun bool,
	emit func(event string, payload map[string]any),
	requestedModelID string,
) *chat.Message {
	if decision, ok := parseChatApprovalDirective(lastUserContent); ok {
		return s.handleChatApprovalDirective(ctx, threadID, userMessageID, decision)
	}
	if probe := s.maybeRespondGatewayStatusProbe(ctx, threadID, userMessageID, th, lastUserContent); probe != nil {
		return probe
	}

	corr := "chat-tools-" + strconv.FormatInt(userMessageID, 10)

	ollamaTools := s.gateway.ChatOllamaToolDefs()
	manifests := s.gateway.ChatToolManifests()
	toolNames := s.gateway.ChatToolModelNames()

	stages := []map[string]any{}
	var trackedActivity map[string]any
	pushStage := func(stage string, data map[string]any) {
		row := map[string]any{"stage": stage, "atMs": time.Now().UnixMilli()}
		for k, v := range data {
			row[k] = v
		}
		stages = append(stages, row)
		if trackedActivity != nil {
			trackedActivity["stages"] = stages
		}
		if emit != nil {
			emit("agent_stage", row)
		}
		_ = s.log.Emit(ctx, "chat.tool.pipeline", map[string]any{
			"correlationId": corr, "threadId": threadID, "userMessageId": userMessageID,
			"stage": stage, "detail": data,
		})
	}

	pushStage("request_received", map[string]any{"userChars": len(lastUserContent)})

	if !gateway.ShouldAttachChatTools(lastUserContent) {
		pushStage("tools_skipped", map[string]any{"reason": "non_operational_turn"})
		return s.completeAssistantWithoutTools(ctx, threadID, userMessageID, th, lastUserContent, ollamaAdapter, corr, stages, pushStage, requestedModelID)
	}

	if s.gateway == nil {
		pushStage("tools_unavailable", map[string]any{"reason": "gateway_nil"})
		text := "The execution gateway is not available in this runtime. I cannot execute tools from chat."
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", text, map[string]any{
			"failure": true, "replyToUserMessageId": userMessageID, "correlationId": corr,
			"toolsUnavailable": true, "toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages},
			"toolGatewayActivity": map[string]any{
				"userRequestSummary": trimSummary(lastUserContent, 500),
				"toolManifest":       manifests,
				"stages":             stages,
				"executionState":     "unavailable",
				"failureReason":      "gateway not loaded in this process",
			},
		})
		return am
	}

	pushStage("tools_attached", map[string]any{"tools": toolNames, "count": len(toolNames)})

	if dryRun {
		pushStage("dry_run", map[string]any{"note": "no_model_no_gateway"})
		text := fmt.Sprintf("Dry run: would attach %d gateway tools and skip Ollama plus gateway execution.", len(ollamaTools))
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", text, map[string]any{
			"dryRun": true, "replyToUserMessageId": userMessageID, "correlationId": corr,
			"toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages},
			"toolGatewayActivity": map[string]any{
				"userRequestSummary": trimSummary(lastUserContent, 500),
				"toolManifest":       manifests,
				"stages":             stages,
				"executionState":     "dry_run",
			},
		})
		return am
	}

	if _, _, _, combined := gateway.ParseCombinedMkdirAndWrite(lastUserContent); combined {
		pushStage("deterministic_combined_shortcut", map[string]any{"reason": "multi-step filesystem intent"})
		gwActivity := map[string]any{
			"userRequestSummary": trimSummary(lastUserContent, 500),
			"toolManifest":       manifests,
			"stages":             stages,
			"toolCallEmitted":    false,
		}
		trackedActivity = gwActivity
		var final strings.Builder
		s.runDeterministicMkdirThenWrite(ctx, corr, lastUserContent, pushStage, gwActivity, &final)
		if final.Len() == 0 {
			final.WriteString("(no deterministic output)")
		}
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", final.String(), map[string]any{
			"replyToUserMessageId": userMessageID,
			"correlationId":        corr,
			"ollamaSkipped":        true,
			"toolManifest":         manifests,
			"toolPipeline":         map[string]any{"stages": stages},
			"toolGatewayActivity":  gwActivity,
		})
		_ = s.log.Emit(ctx, "chat.message.assistant", map[string]any{"threadId": threadID, "messageId": am.ID, "ok": true, "tools": true, "deterministicShortcut": true})
		return am
	}
	if _, _, ok := gateway.ParsePythonBannerScriptIntent(lastUserContent); ok {
		pushStage("deterministic_python_banner_shortcut", map[string]any{"reason": "explicit script creation intent"})
		gwActivity := map[string]any{
			"userRequestSummary": trimSummary(lastUserContent, 500),
			"toolManifest":       manifests,
			"stages":             stages,
			"toolCallEmitted":    false,
		}
		trackedActivity = gwActivity
		var final strings.Builder
		s.runChatFSDeterministicFallback(ctx, corr, lastUserContent, "", pushStage, gwActivity, &final)
		if final.Len() == 0 {
			final.WriteString("(no deterministic output)")
		}
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", final.String(), map[string]any{
			"replyToUserMessageId": userMessageID,
			"correlationId":        corr,
			"ollamaSkipped":        true,
			"toolManifest":         manifests,
			"toolPipeline":         map[string]any{"stages": stages},
			"toolGatewayActivity":  gwActivity,
		})
		_ = s.log.Emit(ctx, "chat.message.assistant", map[string]any{"threadId": threadID, "messageId": am.ID, "ok": true, "tools": true, "deterministicShortcut": true})
		return am
	}

	ol, ok := ollamaAdapter.(adapters.Ollama)
	if !ok {
		pushStage("adapter_mismatch", map[string]any{"detail": "ollama concrete type required for /api/chat tools"})
		pushStage("runtime_fallback", map[string]any{"reason": "ollama adapter not registered"})
		am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama adapter not registered", requestedModelID)
		if am != nil {
			return am
		}
		am, _ = s.chat.AppendMessage(ctx, threadID, "assistant", "Chat tool completion requires the Ollama adapter, and model runtime fallback failed: "+reason, map[string]any{
			"failure": true, "replyToUserMessageId": userMessageID, "correlationId": corr,
			"toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages},
			"toolGatewayActivity": map[string]any{
				"userRequestSummary": trimSummary(lastUserContent, 500),
				"toolManifest":       manifests,
				"stages":             stages,
				"executionState":     "unavailable",
				"failureReason":      "ollama adapter not registered; model runtime fallback failed: " + reason,
			},
		})
		return am
	}

	baseURL := ol.BaseURLForChat(ctx)
	model := ol.ModelForChat(ctx)
	if strings.TrimSpace(model) == "" {
		pushStage("runtime_fallback", map[string]any{"reason": "ollama model is not configured"})
		am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama model is not configured", requestedModelID)
		if am != nil {
			return am
		}
		am, _ = s.chat.AppendMessage(ctx, threadID, "assistant", "ollama model is not configured in Settings, and model runtime fallback failed: "+reason, map[string]any{
			"replyToUserMessageId": userMessageID, "correlationId": corr,
			"toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages},
			"toolGatewayActivity": map[string]any{
				"userRequestSummary": trimSummary(lastUserContent, 500),
				"toolManifest":       manifests,
				"stages":             stages,
				"executionState":     "error",
				"failureReason":      "ollama model is not configured; model runtime fallback failed: " + reason,
			},
		})
		return am
	}

	sys, userBody := s.buildChatLLMMessages(ctx, th)
	msgs := []map[string]any{
		{"role": "system", "content": sys},
		{"role": "user", "content": userBody},
	}

	forcedModel := gateway.ForcedChatModelName(lastUserContent)
	var toolChoice any
	if forcedModel != "" {
		toolChoice = map[string]any{
			"type":     "function",
			"function": map[string]any{"name": forcedModel},
		}
		pushStage("forced_tool_choice", map[string]any{"mode": "function", "reason": "heuristic_match", "tool": forcedModel})
	}
	gwActivity := map[string]any{
		"userRequestSummary": trimSummary(lastUserContent, 500),
		"toolManifest":       manifests,
		"stages":             stages,
		"toolCallEmitted":    false,
	}
	trackedActivity = gwActivity
	if forcedModel != "" {
		gwActivity["forcedToolRequested"] = forcedModel
	}

	var final strings.Builder
	const maxToolTurns = 6
	callSummaries := make([]map[string]any, 0, 8)
	toolCallsExecuted := 0
	toolCallEmittedEver := false
	modelTurns := 0
	fallbackRan := false
	terminalState := "skipped"
	stopLoop := false

	for turn := 0; turn < maxToolTurns; turn++ {
		modelTurns = turn + 1
		var turnToolChoice any
		if turn == 0 {
			turnToolChoice = toolChoice
		}
		raw, err := ol.OllamaChat(ctx, baseURL, model, msgs, ollamaTools, turnToolChoice, 180*time.Second)
		if err != nil {
			pushStage("ollama_chat_error", map[string]any{"turn": turn, "error": err.Error()})
			pushStage("runtime_fallback", map[string]any{"reason": "ollama /api/chat failed", "turn": turn})
			am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama /api/chat failed: "+err.Error(), requestedModelID)
			if am != nil {
				return am
			}
			text := "Ollama /api/chat failed: " + err.Error()
			if strings.TrimSpace(reason) != "" {
				text += " (model runtime fallback failed: " + reason + ")"
			}
			am, _ = s.chat.AppendMessage(ctx, threadID, "assistant", text, map[string]any{
				"failure": true, "replyToUserMessageId": userMessageID, "correlationId": corr,
				"toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages},
				"toolGatewayActivity": map[string]any{
					"userRequestSummary": trimSummary(lastUserContent, 500),
					"toolManifest":       manifests,
					"stages":             stages,
					"executionState":     "error",
					"failureReason":      err.Error(),
					"modelTurns":         modelTurns,
					"toolCallsExecuted":  toolCallsExecuted,
				},
			})
			return am
		}

		rawJSON, _ := json.Marshal(raw)
		rawSnippet := string(rawJSON)
		if len(rawSnippet) > 14000 {
			rawSnippet = rawSnippet[:14000] + "…(truncated)"
		}
		pushStage("model_raw_response", map[string]any{"turn": turn, "json": rawSnippet})

		msg, _ := raw["message"].(map[string]any)
		toolCalls := collectToolCallsFromOllamaMessage(msg)
		content := ""
		if msg != nil {
			content = strings.TrimSpace(asString(msg["content"]))
		}
		toolEmitted := len(toolCalls) > 0
		if toolEmitted {
			toolCallEmittedEver = true
		}
		pushStage("tool_call_check", map[string]any{"turn": turn, "emitted": toolEmitted, "count": len(toolCalls)})

		assistantMsg := map[string]any{"role": "assistant"}
		if content != "" {
			assistantMsg["content"] = content
		}
		if len(toolCalls) > 0 {
			assistantMsg["tool_calls"] = toolCalls
		}
		if content != "" || len(toolCalls) > 0 {
			msgs = append(msgs, assistantMsg)
		}

		if !toolEmitted {
			if turn == 0 {
				fallbackOnly := strings.Builder{}
				fallbackRan = s.runChatFSDeterministicFallback(ctx, corr, lastUserContent, forcedModel, pushStage, gwActivity, &fallbackOnly)
				if fallbackRan {
					if strings.TrimSpace(content) != "" {
						pushStage("model_prose_discarded", map[string]any{
							"reason": "deterministic fallback executed; retaining verified gateway output only",
						})
					}
					final.Reset()
					final.WriteString(fallbackOnly.String())
					terminalState = "ok"
					stopLoop = true
					break
				}
				if forcedModel != "" {
					if strings.TrimSpace(content) == "" {
						final.WriteString("(empty model response)")
					} else {
						final.WriteString(content)
					}
					final.WriteString("\n\n---\nFORGE: No tool_calls were returned for this turn, so nothing ran through the gateway from the model. If the text above claims directories, files, or \"gateway ok\" results, that is not a verified tool outcome for this message.")
					terminalState = "model_omitted_tool_calls"
					gwActivity["failureReason"] = "Ollama returned no tool_calls while a tool was forced; prose is not verified gateway output unless a deterministic fallback ran."
					stopLoop = true
					break
				}
			}
			if strings.TrimSpace(content) == "" {
				final.WriteString("(empty response)")
			} else {
				final.WriteString(content)
			}
			if toolCallEmittedEver {
				terminalState = "ok"
			} else {
				terminalState = "skipped"
			}
			stopLoop = true
			break
		}

		for idx, call := range toolCalls {
			fn, _ := call["function"].(map[string]any)
			name := strings.TrimSpace(asString(fn["name"]))

			// Ollama may return arguments as string or object.
			var argsStr string
			switch v := fn["arguments"].(type) {
			case string:
				argsStr = v
			case map[string]any:
				b, _ := json.Marshal(v)
				argsStr = string(b)
			default:
				argsStr = ""
			}
			pushStage("tool_args", map[string]any{"turn": turn, "index": idx, "name": name, "arguments": argsStr})
			if emit != nil {
				emit("tool_call", map[string]any{
					"turn":      turn,
					"index":     idx,
					"modelName": name,
					"arguments": argsStr,
				})
			}

			toolID, _, resolved := s.gateway.ResolveChatFunctionName(name)
			if !resolved {
				summary := map[string]any{
					"turn":   turn,
					"index":  idx,
					"name":   name,
					"state":  "error",
					"reason": fmt.Sprintf("unknown tool %q (not in gateway chat catalog)", name),
				}
				callSummaries = append(callSummaries, summary)
				gwActivity["failureReason"] = summary["reason"]
				final.Reset()
				final.WriteString(fmt.Sprintf("Unknown tool %q — not exposed from the gateway for chat.", name))
				terminalState = "error"
				stopLoop = true
				break
			}

			result := s.dispatchToolCall(ctx, corr, threadID, name, argsStr, lastUserContent, pushStage)
			summary := map[string]any{
				"turn":        turn,
				"index":       idx,
				"modelName":   name,
				"gatewayTool": toolID,
				"args":        result.args,
				"state":       result.state,
			}
			if result.failureReason != "" {
				summary["reason"] = result.failureReason
			}
			if result.executionResult != nil {
				summary["result"] = result.executionResult
			}
			callSummaries = append(callSummaries, summary)
			toolCallsExecuted++
			if emit != nil {
				emit("tool_result", map[string]any{
					"turn":          turn,
					"index":         idx,
					"modelName":     name,
					"gatewayTool":   toolID,
					"state":         result.state,
					"text":          result.text,
					"failureReason": result.failureReason,
					"result":        result.executionResult,
				})
			}

			gwActivity["toolSelected"] = toolID
			gwActivity["toolArgs"] = result.args
			gwActivity["executionResult"] = result.executionResult
			if result.failureReason != "" {
				gwActivity["failureReason"] = result.failureReason
			}

			toolResultPayload := map[string]any{
				"state":         result.state,
				"text":          result.text,
				"failureReason": result.failureReason,
				"result":        result.executionResult,
			}
			toolResultJSON, _ := json.Marshal(toolResultPayload)
			toolMsg := map[string]any{
				"role":      "tool",
				"name":      toolID,
				"tool_name": toolID,
				"content":   string(toolResultJSON),
			}
			if id := strings.TrimSpace(asString(call["id"])); id != "" {
				toolMsg["tool_call_id"] = id
			}
			msgs = append(msgs, toolMsg)

			if result.state != "ok" {
				if strings.TrimSpace(result.text) != "" {
					final.Reset()
					final.WriteString(result.text)
				} else if strings.TrimSpace(result.failureReason) != "" {
					final.Reset()
					final.WriteString(result.failureReason)
				}
				terminalState = result.state
				stopLoop = true
				break
			}
		}
		if stopLoop {
			break
		}
	}

	if !stopLoop {
		terminalState = "iteration_limit"
		if final.Len() == 0 {
			final.WriteString("Stopped after reaching the maximum tool/model loop depth for this turn. Refine the request or rerun.")
		}
	}
	if final.Len() == 0 {
		final.WriteString("(empty response)")
	}

	gwActivity["toolCallEmitted"] = toolCallEmittedEver || fallbackRan
	gwActivity["executionState"] = terminalState
	gwActivity["modelTurns"] = modelTurns
	gwActivity["toolCallsExecuted"] = toolCallsExecuted
	if fallbackRan {
		gwActivity["modelContentDiscarded"] = true
	}
	if len(callSummaries) > 0 {
		gwActivity["toolCalls"] = callSummaries
	}

	am, err := s.chat.AppendMessage(ctx, threadID, "assistant", final.String(), map[string]any{
		"replyToUserMessageId": userMessageID,
		"correlationId":        corr,
		"ollamaOk":             true,
		"toolManifest":         manifests,
		"toolPipeline":         map[string]any{"stages": stages},
		"toolGatewayActivity":  gwActivity,
	})
	if err != nil {
		return nil
	}
	_ = s.log.Emit(ctx, "chat.message.assistant", map[string]any{
		"threadId":          threadID,
		"messageId":         am.ID,
		"ok":                true,
		"tools":             toolCallEmittedEver || fallbackRan,
		"modelTurns":        modelTurns,
		"toolCallsExecuted": toolCallsExecuted,
	})
	return am
}

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
		paths = append(paths, p)
	}
	if raw, ok := args["paths"]; ok {
		switch typed := raw.(type) {
		case []any:
			for _, x := range typed {
				if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
					paths = append(paths, strings.TrimSpace(s))
				}
			}
		case []string:
			for _, s := range typed {
				if strings.TrimSpace(s) != "" {
					paths = append(paths, strings.TrimSpace(s))
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

func (s *Server) completeAssistantWithoutTools(
	ctx context.Context,
	threadID, userMessageID int64,
	th *chat.ThreadDetail,
	lastUserContent string,
	ollamaAdapter adapters.Adapter,
	corr string,
	stages []map[string]any,
	pushStage func(string, map[string]any),
	requestedModelID string,
) *chat.Message {
	recordStage := func(stage string, data map[string]any) {
		row := map[string]any{"stage": stage, "atMs": time.Now().UnixMilli()}
		for k, v := range data {
			row[k] = v
		}
		stages = append(stages, row)
		pushStage(stage, data)
	}

	manifests := []map[string]any{}
	if s.gateway != nil {
		manifests = s.gateway.ChatToolManifests()
	}

	if s.modelRuntime != nil {
		recordStage("runtime_primary", map[string]any{"reason": "plain chat prefers model runtime"})
		if am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "model-runtime-first plain chat path", requestedModelID); am != nil {
			return am
		} else if strings.TrimSpace(reason) != "" {
			recordStage("runtime_fallback", map[string]any{"reason": "model runtime plain-chat path failed: " + reason})
		}
	}

	ol, ok := ollamaAdapter.(adapters.Ollama)
	if !ok {
		am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama adapter not registered", requestedModelID)
		if am != nil {
			return am
		}
		am, _ = s.chat.AppendMessage(ctx, threadID, "assistant", "Chat completion requires the Ollama adapter in this runtime, and model runtime fallback failed: "+reason, map[string]any{
			"failure": true, "replyToUserMessageId": userMessageID, "correlationId": corr,
			"toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages},
			"toolGatewayActivity": map[string]any{
				"userRequestSummary": trimSummary(lastUserContent, 500),
				"toolManifest":       manifests,
				"stages":             stages,
				"toolCallEmitted":    false,
				"executionState":     "error",
				"failureReason":      "ollama adapter not registered; model runtime fallback failed: " + reason,
			},
		})
		return am
	}

	baseURL := ol.BaseURLForChat(ctx)
	model := ol.ModelForChat(ctx)
	if strings.TrimSpace(model) == "" {
		am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama model is not configured", requestedModelID)
		if am != nil {
			return am
		}
		am, _ = s.chat.AppendMessage(ctx, threadID, "assistant", "ollama model is not configured in Settings, and model runtime fallback failed: "+reason, map[string]any{
			"replyToUserMessageId": userMessageID, "correlationId": corr,
			"toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages},
			"toolGatewayActivity": map[string]any{
				"userRequestSummary": trimSummary(lastUserContent, 500),
				"toolManifest":       manifests,
				"stages":             stages,
				"toolCallEmitted":    false,
				"executionState":     "error",
				"failureReason":      "ollama model is not configured; model runtime fallback failed: " + reason,
			},
		})
		return am
	}

	sys, userBody := s.buildChatLLMMessages(ctx, th)
	msgs := []map[string]any{
		{"role": "system", "content": sys},
		{"role": "user", "content": userBody},
	}
	raw, err := ol.OllamaChat(ctx, baseURL, model, msgs, nil, nil, 120*time.Second)
	if err != nil {
		am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama /api/chat failed: "+err.Error(), requestedModelID)
		if am != nil {
			return am
		}
		text := "Ollama /api/chat failed: " + err.Error()
		if strings.TrimSpace(reason) != "" {
			text += " (model runtime fallback failed: " + reason + ")"
		}
		am, _ = s.chat.AppendMessage(ctx, threadID, "assistant", text, map[string]any{
			"failure": true, "replyToUserMessageId": userMessageID, "correlationId": corr,
			"toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages},
			"toolGatewayActivity": map[string]any{
				"userRequestSummary": trimSummary(lastUserContent, 500),
				"toolManifest":       manifests,
				"stages":             stages,
				"toolCallEmitted":    false,
				"executionState":     "error",
				"failureReason":      err.Error(),
			},
		})
		return am
	}

	content := ""
	if msg, _ := raw["message"].(map[string]any); msg != nil {
		content = strings.TrimSpace(asString(msg["content"]))
	}
	if content == "" {
		content = "Acknowledged. Ready for your next instruction."
	}

	am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", content, map[string]any{
		"replyToUserMessageId": userMessageID,
		"correlationId":        corr,
		"ollamaOk":             true,
		"toolsBypassed":        true,
		"toolManifest":         manifests,
		"toolPipeline":         map[string]any{"stages": stages},
		"toolGatewayActivity": map[string]any{
			"userRequestSummary": trimSummary(lastUserContent, 500),
			"toolManifest":       manifests,
			"stages":             stages,
			"toolCallEmitted":    false,
			"executionState":     "skipped",
		},
	})
	return am
}

func (s *Server) completeAssistantWithModelRuntime(
	ctx context.Context,
	threadID, userMessageID int64,
	th *chat.ThreadDetail,
	lastUserContent, corr string,
	manifests []map[string]any,
	stages []map[string]any,
	fallbackReason string,
	requestedModelID string,
) (*chat.Message, string) {
	if s.modelRuntime == nil {
		return nil, "model runtime is unavailable"
	}

	workspaceID := strings.TrimSpace(s.cfg.WorkspaceDir)
	meta := ModelRuntimeRequestMeta{
		CorrelationID: strings.TrimSpace(corr),
		WorkspaceID:   workspaceID,
	}

	modelID, resolveReason := s.resolveChatModelRuntimeModel(ctx, meta, requestedModelID)
	if strings.TrimSpace(modelID) == "" {
		return nil, resolveReason
	}

	sys, userBody := s.buildChatLLMMessages(ctx, th)
	result, err := s.modelRuntime.Chat(ctx, ModelRuntimeChatRequest{
		ModelID: modelID,
		Messages: []ModelRuntimeChatMessage{
			{Role: "system", Content: sys},
			{Role: "user", Content: userBody},
		},
		Actor:  "chat",
		Source: "chat_assistant",
		Meta:   meta,
		Metadata: map[string]any{
			"entrypoint": "api.chat",
			"fallback":   "model_runtime",
		},
	})
	if err != nil {
		_, code, message := mapModelRuntimeError(err)
		if strings.TrimSpace(code) != "" {
			return nil, message + " (" + code + ")"
		}
		return nil, message
	}

	content := strings.TrimSpace(result.Content)
	if content == "" {
		content = "Acknowledged. Ready for your next instruction."
	}

	activity := map[string]any{
		"userRequestSummary": trimSummary(lastUserContent, 500),
		"toolManifest":       manifests,
		"stages":             stages,
		"toolPipeline":       map[string]any{"stages": stages},
		"toolCallEmitted":    false,
		"executionState":     "skipped",
		"fallback":           "model_runtime",
		"modelId":            nonEmpty(strings.TrimSpace(result.ModelID), modelID),
		"backend":            strings.TrimSpace(result.Backend),
	}
	if requested := strings.TrimSpace(requestedModelID); requested != "" {
		activity["requestedModelId"] = requested
	}
	if trimmed := strings.TrimSpace(fallbackReason); trimmed != "" {
		activity["fallbackReason"] = trimmed
	}

	metadata := map[string]any{
		"replyToUserMessageId": userMessageID,
		"correlationId":        corr,
		"modelRuntimeOk":       true,
		"modelRuntimeModelId":  nonEmpty(strings.TrimSpace(result.ModelID), modelID),
		"modelRuntimeBackend":  strings.TrimSpace(result.Backend),
		"toolManifest":         manifests,
		"toolPipeline":         map[string]any{"stages": stages},
		"toolGatewayActivity":  activity,
	}
	if requested := strings.TrimSpace(requestedModelID); requested != "" {
		metadata["modelRuntimeRequestedModelId"] = requested
	}
	if trimmed := strings.TrimSpace(result.AuditID); trimmed != "" {
		metadata["modelRuntimeAuditId"] = trimmed
	}
	if len(result.Warnings) > 0 {
		metadata["modelRuntimeWarnings"] = append([]string(nil), result.Warnings...)
	}

	am, err := s.chat.AppendMessage(ctx, threadID, "assistant", content, metadata)
	if err != nil {
		return nil, "assistant reply could not be saved"
	}
	return am, ""
}

func (s *Server) resolveChatModelRuntimeModel(ctx context.Context, meta ModelRuntimeRequestMeta, requestedModelID string) (string, string) {
	if s.modelRuntime == nil {
		return "", "model runtime is unavailable"
	}

	models, err := s.modelRuntime.ListModels(ctx, ModelRuntimeListRequest{Meta: meta})
	if err != nil {
		_, code, message := mapModelRuntimeError(err)
		if strings.TrimSpace(code) != "" {
			return "", message + " (" + code + ")"
		}
		return "", message
	}
	if len(models) == 0 {
		return "", "no managed models are registered in model runtime"
	}

	requested := strings.TrimSpace(requestedModelID)
	if requested != "" {
		for _, model := range models {
			if !strings.EqualFold(strings.TrimSpace(model.ID), requested) {
				continue
			}
			if !modelRuntimeModelSupportsChat(model) {
				return "", "requested model does not advertise chat/completion capability"
			}
			if !modelRuntimeModelUsableForChat(model) {
				return "", "requested model is not available for chat"
			}
			return strings.TrimSpace(model.ID), ""
		}
		return "", fmt.Sprintf("requested model %q is not registered in model runtime", requested)
	}

	defaultID := strings.TrimSpace(s.cfg.ModelDefaultID)
	if defaultID != "" {
		for _, model := range models {
			if !strings.EqualFold(strings.TrimSpace(model.ID), defaultID) {
				continue
			}
			if !modelRuntimeModelSupportsChat(model) {
				return "", "default model does not advertise chat/completion capability"
			}
			if !modelRuntimeModelUsableForChat(model) {
				return "", "default model is not available for chat"
			}
			return strings.TrimSpace(model.ID), ""
		}
	}

	type candidate struct {
		model ModelRuntimeModel
		rank  int
	}
	candidates := make([]candidate, 0, len(models))
	for _, model := range models {
		if !modelRuntimeModelSupportsChat(model) || !modelRuntimeModelUsableForChat(model) {
			continue
		}
		candidates = append(candidates, candidate{
			model: model,
			rank:  modelRuntimeChatStatusRank(model.Status),
		})
	}
	if len(candidates) == 0 {
		return "", "no chat-capable available model is registered in model runtime"
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].rank != candidates[j].rank {
			return candidates[i].rank > candidates[j].rank
		}
		return strings.ToLower(strings.TrimSpace(candidates[i].model.ID)) < strings.ToLower(strings.TrimSpace(candidates[j].model.ID))
	})
	return strings.TrimSpace(candidates[0].model.ID), ""
}

func modelRuntimeModelSupportsChat(model ModelRuntimeModel) bool {
	if len(model.Capabilities) == 0 {
		return true
	}
	for _, capability := range model.Capabilities {
		switch strings.ToLower(strings.TrimSpace(capability)) {
		case "chat", "completion":
			return true
		}
	}
	return false
}

func modelRuntimeModelUsableForChat(model ModelRuntimeModel) bool {
	switch strings.ToLower(strings.TrimSpace(model.Status)) {
	case "disabled", "archived", "unavailable", "error":
		return false
	default:
		return true
	}
}

func modelRuntimeChatStatusRank(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "loaded":
		return 5
	case "available":
		return 4
	case "verified":
		return 3
	case "imported":
		return 2
	case "loading", "unloading":
		return 1
	default:
		return 0
	}
}

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
