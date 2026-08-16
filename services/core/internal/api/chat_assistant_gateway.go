package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/consensusgate"
	"forge/projectforge/services/core/internal/forgekernel/runtimeproposal"
	"forge/projectforge/services/core/internal/gateway"
)

const (
	modelRuntimePlainChatMessages       = 4
	modelRuntimePlainChatMessageMax     = 800
	modelRuntimePlainChatUserMax        = 6000
	modelRuntimePlainChatSystemMax      = 3200
	modelRuntimePlainChatMemoryMax      = 1000
	modelRuntimePlainChatAttachmentMax  = 1200
	modelRuntimePlainChatMaxOutputToken = 384
	modelRuntimePlainChatTimeoutMs      = 30000
	modelRuntimePlainChatMaxAttempts    = 1
	chatToolArgumentMaxBytes            = 32 << 10
	assistantStreamFlushChars           = 160
	assistantStreamFlushIntervalMs      = 24
)

const assistantContentFallback = "I couldn't produce a clean assistant response. Try again, or check the selected model/runtime."

func narrowOllamaToolsToForgeSelection(tools []map[string]any, selectedTool string) ([]map[string]any, bool) {
	selectedTool = strings.TrimSpace(selectedTool)
	if selectedTool == "" {
		return nil, false
	}
	for _, tool := range tools {
		fn, _ := tool["function"].(map[string]any)
		name, _ := fn["name"].(string)
		if strings.TrimSpace(name) == selectedTool {
			return []map[string]any{tool}, true
		}
	}
	return tools, false
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
	requestStart := time.Now()
	perf := classifyChatPerformance(lastUserContent)
	if decision, ok := parseChatApprovalDirective(lastUserContent); ok {
		return s.handleChatApprovalDirective(ctx, threadID, userMessageID, decision)
	}
	if am, handled := s.maybeRespondHyperlaneNoModel(ctx, threadID, userMessageID, lastUserContent); handled {
		return am
	}
	if probe := s.maybeRespondGatewayStatusProbe(ctx, threadID, userMessageID, th, lastUserContent); probe != nil {
		return probe
	}

	corr := "chat-tools-" + strconv.FormatInt(userMessageID, 10)

	var ollamaTools []map[string]any
	var manifests []map[string]any
	var toolNames []string
	toolCatalogLoaded := false
	loadToolCatalog := func() {
		if toolCatalogLoaded {
			return
		}
		toolCatalogLoaded = true
		if s.gateway == nil {
			return
		}
		ollamaTools = s.gateway.ChatOllamaToolDefs()
		manifests = s.gateway.ChatToolManifests()
		toolNames = s.gateway.ChatToolModelNames()
	}

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
	pushStage("hyperlane_classified", map[string]any{
		"intent":             perf.Intent,
		"contextBudgetClass": perf.ContextBudgetClass,
		"outputMode":         perf.OutputMode,
		"noModel":            perf.NoModel,
		"confidence":         perf.Confidence,
		"reason":             perf.Reason,
		"hyperlaneMs":        perf.HyperlaneMs,
	})

	if perf.NoModel {
		pushStage("tools_skipped", map[string]any{"reason": "hyperlane_no_model_route", "intent": perf.Intent})
		return s.completeAssistantWithoutTools(ctx, threadID, userMessageID, th, lastUserContent, ollamaAdapter, corr, stages, pushStage, emit, requestedModelID)
	}

	if !gateway.ShouldUseChatToolPipeline(lastUserContent) {
		pushStage("tools_skipped", map[string]any{"reason": "non_operational_turn"})
		return s.completeAssistantWithoutTools(ctx, threadID, userMessageID, th, lastUserContent, ollamaAdapter, corr, stages, pushStage, emit, requestedModelID)
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

	selectedTool := gateway.SelectChatToolName(lastUserContent)
	loadToolCatalog()
	if narrowed, ok := narrowOllamaToolsToForgeSelection(ollamaTools, selectedTool); ok {
		ollamaTools = narrowed
		toolNames = []string{selectedTool}
		pushStage("forge_tool_selected", map[string]any{"reason": "deterministic_intent_route", "tool": selectedTool})
	} else if selectedTool != "" {
		pushStage("tools_unavailable", map[string]any{"reason": "forge_selected_tool_missing_from_gateway_catalog", "tool": selectedTool})
		text := "FORGE selected a tool for this request, but that tool is not present in the governed gateway catalog. No model tool proposal was requested."
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", text, map[string]any{
			"failure": true, "replyToUserMessageId": userMessageID, "correlationId": corr,
			"toolsUnavailable": true, "toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages},
		})
		return am
	} else {
		ollamaTools = nil
		toolNames = nil
	}
	pushStage("tool_proposal_schema_attached", map[string]any{"tools": toolNames, "count": len(toolNames), "selectedBy": "forge"})

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
	if writes, ok := gateway.ParseSVGAssetWriteIntents(lastUserContent); ok {
		pushStage("deterministic_svg_shortcut", map[string]any{"reason": "explicit SVG asset creation intent", "count": len(writes)})
		gwActivity := map[string]any{
			"userRequestSummary": trimSummary(lastUserContent, 500),
			"toolManifest":       manifests,
			"stages":             stages,
			"toolCallEmitted":    false,
		}
		trackedActivity = gwActivity
		var final strings.Builder
		s.runDeterministicSVGWrites(ctx, corr, writes, pushStage, gwActivity, &final)
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
	if selectedTool != gateway.ChatModelName("desktop.open") {
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
	} else if _, _, ok := gateway.ParsePythonBannerScriptIntent(lastUserContent); ok {
		pushStage("deterministic_python_banner_shortcut_skipped", map[string]any{"reason": "remote terminal workflow uses desktop.open"})
	}
	if selectedTool == gateway.ChatModelName("desktop.open") {
		pushStage("deterministic_desktop_shortcut", map[string]any{"reason": "explicit desktop or terminal intent"})
		gwActivity := map[string]any{
			"userRequestSummary": trimSummary(lastUserContent, 500),
			"toolManifest":       manifests,
			"stages":             stages,
			"toolCallEmitted":    false,
		}
		trackedActivity = gwActivity
		var final strings.Builder
		s.runChatFSDeterministicFallback(ctx, corr, lastUserContent, selectedTool, pushStage, gwActivity, &final)
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
	if selectedTool == gateway.ChatModelName("repo.inspect") {
		pushStage("deterministic_repo_inspect_shortcut", map[string]any{"reason": "explicit repo orientation intent"})
		gwActivity := map[string]any{
			"userRequestSummary": trimSummary(lastUserContent, 500),
			"toolManifest":       manifests,
			"stages":             stages,
			"toolCallEmitted":    false,
		}
		trackedActivity = gwActivity
		var final strings.Builder
		s.runChatFSDeterministicFallback(ctx, corr, lastUserContent, selectedTool, pushStage, gwActivity, &final)
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
	if _, _, ok := gateway.ParseDownloadSorterScriptIntent(lastUserContent); ok {
		pushStage("deterministic_download_sorter_shortcut", map[string]any{"reason": "explicit Downloads sorter script intent"})
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
	if writePath, html, ok := gateway.ParseVideoGameJournalWebpageIntent(lastUserContent, s.latestGatewayFilesystemDir(ctx, th)); ok {
		pushStage("deterministic_webpage_shortcut", map[string]any{"reason": "same-directory webpage intent", "path": writePath})
		gwActivity := map[string]any{
			"userRequestSummary": trimSummary(lastUserContent, 500),
			"toolManifest":       manifests,
			"stages":             stages,
			"toolCallEmitted":    false,
		}
		trackedActivity = gwActivity
		var final strings.Builder
		s.runDeterministicWrite(ctx, corr, "webpage", writePath, html, pushStage, gwActivity, &final)
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
	if selectedTool == "" {
		pushStage("tools_skipped", map[string]any{"reason": "no_deterministic_forge_tool_selection"})
		return s.completeAssistantWithoutTools(ctx, threadID, userMessageID, th, lastUserContent, ollamaAdapter, corr, stages, pushStage, emit, requestedModelID)
	}

	ol, ok := ollamaAdapter.(adapters.Ollama)
	if !ok {
		pushStage("adapter_mismatch", map[string]any{"detail": "ollama concrete type required for /api/chat tools"})
		pushStage("runtime_fallback", map[string]any{"reason": "ollama adapter not registered"})
		am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama adapter not registered", requestedModelID, requestStart, perf)
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
	model, modelSource := s.resolveNativeOllamaChatModel(ctx, ol, requestedModelID)
	if strings.TrimSpace(model) == "" {
		pushStage("runtime_fallback", map[string]any{"reason": "ollama model is not configured"})
		am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama model is not configured", requestedModelID, requestStart, perf)
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
	pushStage("ollama_model_resolved", map[string]any{"model": model, "source": modelSource})

	sys, userBody, contextBinding, contextErr := s.prepareGovernedOllamaPrompt(ctx, th)
	if contextErr != nil {
		pushStage("context_compile_failed", map[string]any{"reason": contextErr.Error()})
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", "FORGE-K Context Compiler failed closed: "+contextErr.Error(), map[string]any{"failure": true, "replyToUserMessageId": userMessageID, "correlationId": corr})
		return am
	}
	msgs := []map[string]any{
		{"role": "system", "content": sys},
		{"role": "user", "content": userBody},
	}

	var toolChoice any
	if selectedTool != "" {
		toolChoice = map[string]any{
			"type":     "function",
			"function": map[string]any{"name": selectedTool},
		}
		pushStage("forge_tool_choice_enforced", map[string]any{"mode": "function", "reason": "deterministic_intent_route", "tool": selectedTool})
	}
	gwActivity := map[string]any{
		"userRequestSummary": trimSummary(lastUserContent, 500),
		"toolManifest":       manifests,
		"stages":             stages,
		"toolCallEmitted":    false,
	}
	trackedActivity = gwActivity
	if selectedTool != "" {
		gwActivity["forgeSelectedTool"] = selectedTool
	}

	var final strings.Builder
	const maxToolTurns = 6
	callSummaries := make([]map[string]any, 0, 8)
	toolCallsExecuted := 0
	toolCallEmittedEver := false
	finalFromModel := false
	var finalModelPrompt []map[string]any
	runtimeGatewayEvidenceRows := make([]runtimeGatewayEvidence, 0, 4)
	modelTurns := 0
	fallbackRan := false
	terminalState := "skipped"
	stopLoop := false

	for turn := 0; turn < maxToolTurns; turn++ {
		modelTurns = turn + 1
		turnPrompt := append([]map[string]any(nil), msgs...)
		var turnToolChoice any
		if turn == 0 {
			turnToolChoice = toolChoice
		}
		raw, err := ol.OllamaChat(ctx, baseURL, model, msgs, ollamaTools, turnToolChoice, 180*time.Second)
		if err != nil {
			pushStage("ollama_chat_error", map[string]any{"turn": turn, "error": err.Error()})
			pushStage("runtime_fallback", map[string]any{"reason": "ollama /api/chat failed", "turn": turn})
			am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama /api/chat failed: "+err.Error(), requestedModelID, requestStart, perf)
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
		pushStage("model_raw_response_bound", map[string]any{
			"turn":  turn,
			"bytes": len(rawJSON),
			"hash":  runtimeproposal.HashText(string(rawJSON)),
		})

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

		if selectedTool != "" && toolEmitted {
			if mismatch, selected := s.forgeToolCallMismatch(selectedTool, toolCalls); mismatch {
				pushStage("forced_tool_mismatch_discarded", map[string]any{
					"turn":     turn,
					"forced":   selectedTool,
					"selected": selected,
				})
				gwActivity["modelToolCallsDiscarded"] = true
				gwActivity["discardedToolCalls"] = selected
				fallbackOnly := strings.Builder{}
				fallbackRan = s.runChatFSDeterministicFallback(ctx, corr, lastUserContent, selectedTool, pushStage, gwActivity, &fallbackOnly)
				if fallbackRan {
					if strings.TrimSpace(content) != "" {
						pushStage("model_prose_discarded", map[string]any{
							"reason": "forced tool mismatch discarded; retaining verified gateway output only",
						})
					}
					final.Reset()
					final.WriteString(fallbackOnly.String())
					terminalState = chatActivityState(gwActivity, "ok")
					stopLoop = true
					break
				}
				final.WriteString(forgeAuthorityToolOmissionMessage(selectedTool))
				terminalState = "model_tool_mismatch"
				gwActivity["failureReason"] = "Ollama returned a different tool than the FORGE-forced gateway route; model tool call discarded because FORGE owns capability and availability decisions."
				stopLoop = true
				break
			}
		}

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
				fallbackRan = s.runChatFSDeterministicFallback(ctx, corr, lastUserContent, selectedTool, pushStage, gwActivity, &fallbackOnly)
				if fallbackRan {
					if strings.TrimSpace(content) != "" {
						pushStage("model_prose_discarded", map[string]any{
							"reason": "deterministic fallback executed; retaining verified gateway output only",
						})
					}
					final.Reset()
					final.WriteString(fallbackOnly.String())
					terminalState = chatActivityState(gwActivity, "ok")
					stopLoop = true
					break
				}
				if selectedTool != "" {
					if strings.TrimSpace(content) != "" {
						pushStage("model_prose_discarded", map[string]any{
							"reason": "forced tool route omitted tool_calls; model prose cannot decide FORGE capability or availability",
						})
						gwActivity["modelProseDiscarded"] = true
					}
					final.WriteString(forgeAuthorityToolOmissionMessage(selectedTool))
					terminalState = "model_omitted_tool_calls"
					gwActivity["failureReason"] = "Ollama returned no tool_calls while a tool was forced; model prose discarded because FORGE owns capability and availability decisions."
					stopLoop = true
					break
				}
			}
			if strings.TrimSpace(content) == "" {
				final.WriteString("(empty response)")
			} else {
				final.WriteString(content)
				finalFromModel = true
				finalModelPrompt = turnPrompt
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
			argsHash := runtimeproposal.HashText(argsStr)
			pushStage("tool_args_bound", map[string]any{"turn": turn, "index": idx, "name": name, "bytes": len(argsStr), "hash": argsHash})
			if emit != nil {
				emit("tool_call", map[string]any{
					"turn":          turn,
					"index":         idx,
					"modelName":     name,
					"argumentsHash": argsHash,
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
			if result.gatewayRequest != nil && result.gatewayResult != nil {
				runtimeGatewayEvidenceRows = append(runtimeGatewayEvidenceRows, runtimeGatewayEvidence{
					InvocationID: result.gatewayResult.InvocationID,
					ToolID:       result.gatewayResult.Tool,
					State:        result.state,
					AuditID:      result.auditID,
					Request:      result.gatewayRequest,
					Result:       result.gatewayResult,
				})
			}
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
	var runtimeDecision any
	var runtimeDecisionError string
	var consensusDecision any
	if finalFromModel {
		decision, decisionErr := decideRuntimeProposal(runtimeProposalRequest{
			SourceKind:    runtimeproposal.SourceNativeOllama,
			WorkspacePath: s.cfg.WorkspaceDir,
			ThreadID:      threadID, UserMessageID: userMessageID, CorrelationID: corr,
			Prompt: finalModelPrompt, Output: final.String(), Backend: baseURL, ModelID: model,
			GatewayEvidence: runtimeGatewayEvidenceRows,
			ContextBinding:  contextBinding,
		})
		candidate, _ := sanitizeAssistantVisibleContent(final.String())
		consensus := consensusgate.Gate(consensusgate.Input{
			Content: candidate, Surface: consensusgate.SurfaceChatFinal,
			WorkspaceID: s.cfg.WorkspaceDir, CorrelationID: corr, ModelProposalOnly: true,
		})
		visible := runtimeProposalFailureText
		if decisionErr == nil {
			visible = decision.VisibleContent
			if decision.Status == runtimeproposal.StatusAccepted {
				visible = consensus.Content
			}
		}
		if strings.TrimSpace(visible) == "" {
			visible = assistantContentFallback
		}
		final.Reset()
		final.WriteString(visible)
		runtimeDecision = runtimeProposalEvidence(decision)
		runtimeDecisionError = runtimeProposalError(decisionErr)
		consensusDecision = consensus
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
		"runtimeProposal":      runtimeDecision,
		"runtimeProposalError": runtimeDecisionError,
		"consensusGate":        consensusDecision,
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
