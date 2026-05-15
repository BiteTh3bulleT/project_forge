package api

import (
	"context"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/chat"
)

func (s *Server) completeAssistantWithoutTools(
	ctx context.Context,
	threadID, userMessageID int64,
	th *chat.ThreadDetail,
	lastUserContent string,
	ollamaAdapter adapters.Adapter,
	corr string,
	stages []map[string]any,
	pushStage func(string, map[string]any),
	emit func(event string, payload map[string]any),
	requestedModelID string,
) *chat.Message {
	requestStart := time.Now()
	perf := classifyChatPerformance(lastUserContent)
	recordStage := func(stage string, data map[string]any) {
		row := map[string]any{"stage": stage, "atMs": time.Now().UnixMilli()}
		for k, v := range data {
			row[k] = v
		}
		stages = append(stages, row)
		pushStage(stage, data)
	}
	recordStage("hyperlane_classified", map[string]any{
		"intent":             perf.Intent,
		"contextBudgetClass": perf.ContextBudgetClass,
		"outputMode":         perf.OutputMode,
		"noModel":            perf.NoModel,
		"confidence":         perf.Confidence,
		"reason":             perf.Reason,
		"hyperlaneMs":        perf.HyperlaneMs,
	})

	var manifests []map[string]any
	manifestLoaded := false
	getManifests := func() []map[string]any {
		if !manifestLoaded {
			if s.gateway != nil {
				manifests = s.gateway.ChatToolManifests()
			}
			manifestLoaded = true
		}
		return manifests
	}

	if perf.NoModel {
		text := s.renderNoModelChatReply(ctx, perf, th, lastUserContent)
		if strings.TrimSpace(text) == "" {
			text = assistantContentFallback
		}
		recordStage("deterministic_no_model_reply", map[string]any{"reason": perf.Reason, "intent": perf.Intent})
		trace := chatHyperlaneNoModelTrace(requestStart, perf)
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", text, map[string]any{
			"replyToUserMessageId": userMessageID,
			"correlationId":        corr,
			"modelSkipped":         true,
			"chatLatencyTrace":     trace,
			"toolPipeline":         map[string]any{"stages": stages},
			"toolGatewayActivity": map[string]any{
				"userRequestSummary":  trimSummary(lastUserContent, 500),
				"stages":              stages,
				"toolCallEmitted":     false,
				"executionState":      "skipped",
				"latencyTrace":        trace,
				"contextBudgetClass":  perf.ContextBudgetClass,
				"outputMode":          perf.OutputMode,
				"hyperlaneIntentType": trace["hyperlane_intent_type"],
				"hyperlaneRoute":      trace["hyperlane_route"],
				"gatewayAvoided":      true,
				"modelruntimeAvoided": true,
			},
		})
		return am
	}

	manifests = getManifests()
	if s.modelRuntime != nil {
		recordStage("runtime_primary", map[string]any{"reason": "plain chat prefers model runtime"})
		if am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "model-runtime-first plain chat path", requestedModelID, requestStart, perf); am != nil {
			return am
		} else if strings.TrimSpace(reason) != "" {
			recordStage("runtime_fallback", map[string]any{"reason": "model runtime plain-chat path failed: " + reason})
		}
	}

	if emit != nil {
		if am := s.completeAssistantWithNativeOllamaStream(ctx, threadID, userMessageID, th, lastUserContent, ollamaAdapter, corr, getManifests, stages, pushStage, emit, requestStart, perf); am != nil {
			return am
		}
	}

	ol, ok := ollamaAdapter.(adapters.Ollama)
	if !ok {
		am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama adapter not registered", requestedModelID, requestStart, perf)
		if am != nil {
			return am
		}
		trace := chatLatencyTrace(requestStart, perf, map[string]any{"modelruntime_ms": int64(0), "gateway_execution_ms": int64(0)})
		am, _ = s.chat.AppendMessage(ctx, threadID, "assistant", "Chat completion requires the Ollama adapter in this runtime, and model runtime fallback failed: "+reason, map[string]any{
			"failure": true, "replyToUserMessageId": userMessageID, "correlationId": corr,
			"toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages}, "chatLatencyTrace": trace,
			"toolGatewayActivity": map[string]any{
				"userRequestSummary": trimSummary(lastUserContent, 500),
				"toolManifest":       manifests,
				"stages":             stages,
				"toolCallEmitted":    false,
				"executionState":     "error",
				"failureReason":      "ollama adapter not registered; model runtime fallback failed: " + reason,
				"latencyTrace":       trace,
			},
		})
		return am
	}

	baseURL := ol.BaseURLForChat(ctx)
	model := ol.ModelForChat(ctx)
	if strings.TrimSpace(model) == "" {
		am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama model is not configured", requestedModelID, requestStart, perf)
		if am != nil {
			return am
		}
		trace := chatLatencyTrace(requestStart, perf, map[string]any{"modelruntime_ms": int64(0), "gateway_execution_ms": int64(0)})
		am, _ = s.chat.AppendMessage(ctx, threadID, "assistant", "ollama model is not configured in Settings, and model runtime fallback failed: "+reason, map[string]any{
			"replyToUserMessageId": userMessageID, "correlationId": corr,
			"toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages}, "chatLatencyTrace": trace,
			"toolGatewayActivity": map[string]any{
				"userRequestSummary": trimSummary(lastUserContent, 500),
				"toolManifest":       manifests,
				"stages":             stages,
				"toolCallEmitted":    false,
				"executionState":     "error",
				"failureReason":      "ollama model is not configured; model runtime fallback failed: " + reason,
				"latencyTrace":       trace,
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
		am, reason := s.completeAssistantWithModelRuntime(ctx, threadID, userMessageID, th, lastUserContent, corr, manifests, stages, "ollama /api/chat failed: "+err.Error(), requestedModelID, requestStart, perf)
		if am != nil {
			return am
		}
		text := "Ollama /api/chat failed: " + err.Error()
		if strings.TrimSpace(reason) != "" {
			text += " (model runtime fallback failed: " + reason + ")"
		}
		trace := chatLatencyTrace(requestStart, perf, map[string]any{"modelruntime_ms": int64(0), "gateway_execution_ms": int64(0)})
		am, _ = s.chat.AppendMessage(ctx, threadID, "assistant", text, map[string]any{
			"failure": true, "replyToUserMessageId": userMessageID, "correlationId": corr,
			"toolManifest": manifests, "toolPipeline": map[string]any{"stages": stages}, "chatLatencyTrace": trace,
			"toolGatewayActivity": map[string]any{
				"userRequestSummary": trimSummary(lastUserContent, 500),
				"toolManifest":       manifests,
				"stages":             stages,
				"toolCallEmitted":    false,
				"executionState":     "error",
				"failureReason":      err.Error(),
				"latencyTrace":       trace,
			},
		})
		return am
	}

	content := ""
	if msg, _ := raw["message"].(map[string]any); msg != nil {
		content = strings.TrimSpace(asString(msg["content"]))
	}
	content, contentWarnings := sanitizeAssistantVisibleContent(content)
	if content == "" {
		content = assistantContentFallback
	}
	trace := chatLatencyTrace(requestStart, perf, map[string]any{
		"modelruntime_ms":      int64(0),
		"gateway_execution_ms": int64(0),
		"model_calls_avoided":  0,
	})
	metadata := map[string]any{
		"replyToUserMessageId": userMessageID,
		"correlationId":        corr,
		"ollamaOk":             true,
		"toolsBypassed":        true,
		"chatLatencyTrace":     trace,
		"toolManifest":         manifests,
		"toolPipeline":         map[string]any{"stages": stages},
		"toolGatewayActivity": map[string]any{
			"userRequestSummary": trimSummary(lastUserContent, 500),
			"toolManifest":       manifests,
			"stages":             stages,
			"toolCallEmitted":    false,
			"executionState":     "skipped",
			"latencyTrace":       trace,
		},
	}
	if len(contentWarnings) > 0 {
		metadata["assistantContentWarnings"] = contentWarnings
	}

	am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", content, metadata)
	return am
}

func (s *Server) completeAssistantWithNativeOllamaStream(
	ctx context.Context,
	threadID, userMessageID int64,
	th *chat.ThreadDetail,
	lastUserContent string,
	ollamaAdapter adapters.Adapter,
	corr string,
	getManifests func() []map[string]any,
	stages []map[string]any,
	pushStage func(string, map[string]any),
	emit func(event string, payload map[string]any),
	requestStart time.Time,
	perf chatPerformanceDecision,
) *chat.Message {
	ol, ok := ollamaAdapter.(adapters.Ollama)
	if !ok {
		pushStage("ollama_native_stream_unavailable", map[string]any{"reason": "ollama adapter not registered"})
		return nil
	}
	baseURL := ol.BaseURLForChat(ctx)
	model := ol.ModelForChat(ctx)
	if strings.TrimSpace(model) == "" {
		pushStage("ollama_native_stream_unavailable", map[string]any{"reason": "ollama model is not configured"})
		return nil
	}

	runtimeMessages, promptBudget := s.buildModelRuntimePlainChatMessages(ctx, th)
	messages := make([]map[string]any, 0, len(runtimeMessages))
	for _, msg := range runtimeMessages {
		messages = append(messages, map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	pushStage("ollama_native_stream_start", map[string]any{
		"model":        model,
		"promptBudget": modelRuntimePromptBudgetMap(promptBudget),
	})

	var rawStream strings.Builder
	lastFlush := time.Time{}
	emittedFirst := false
	emittedChars := 0
	streamCut := false
	flushVisible := func(force bool) {
		visible, cut := stripSyntheticTranscriptContinuation(rawStream.String())
		if cut {
			streamCut = true
		}
		emitLimit := len(visible)
		if !force && !streamCut {
			const markerLookbehind = 16
			if emitLimit > markerLookbehind {
				emitLimit -= markerLookbehind
			} else if emittedFirst {
				emitLimit = emittedChars
			}
		}
		if emitLimit <= emittedChars {
			return
		}
		now := time.Now()
		if !force && emittedFirst && emitLimit-emittedChars < assistantStreamFlushChars && now.Sub(lastFlush) < time.Duration(assistantStreamFlushIntervalMs)*time.Millisecond {
			return
		}
		chunk := visible[emittedChars:emitLimit]
		emit("token", map[string]any{"text": chunk})
		emittedChars = emitLimit
		emittedFirst = true
		lastFlush = now
	}
	content, streamMeta, err := ol.StreamChat(ctx, baseURL, model, messages, 120*time.Second, func(token string) error {
		if streamCut {
			return nil
		}
		rawStream.WriteString(token)
		flushVisible(false)
		return nil
	})
	flushVisible(true)
	if err != nil {
		pushStage("ollama_native_stream_error", map[string]any{"error": err.Error()})
		return nil
	}
	if streamCut {
		pushStage("ollama_native_stream_truncated", map[string]any{"reason": "synthetic transcript continuation"})
	}
	content, contentWarnings := sanitizeAssistantVisibleContent(content)
	if content == "" {
		content = assistantContentFallback
	}
	trace := chatLatencyTraceWithPrompt(requestStart, perf, promptBudget, map[string]any{
		"modelruntime_ms":      int64(0),
		"gateway_execution_ms": int64(0),
		"model_calls_avoided":  1,
	})
	metadata := map[string]any{
		"replyToUserMessageId": userMessageID,
		"correlationId":        corr,
		"ollamaOk":             true,
		"ollamaStream":         true,
		"chatLatencyTrace":     trace,
		"streamBatching": map[string]any{
			"flushChars":      assistantStreamFlushChars,
			"flushIntervalMs": assistantStreamFlushIntervalMs,
		},
		"toolsBypassed":       true,
		"modelRuntimeSkipped": true,
		"toolManifest":        getManifests(),
		"toolPipeline":        map[string]any{"stages": stages},
		"toolGatewayActivity": map[string]any{
			"userRequestSummary": trimSummary(lastUserContent, 500),
			"toolManifest":       getManifests(),
			"stages":             stages,
			"toolCallEmitted":    false,
			"executionState":     "skipped",
			"fallback":           "ollama_native_stream",
			"model":              model,
			"promptBudget":       modelRuntimePromptBudgetMap(promptBudget),
			"latencyTrace":       trace,
			"contextBudgetClass": perf.ContextBudgetClass,
			"outputMode":         perf.OutputMode,
		},
		"ollamaMetadata": streamMeta,
	}
	if len(contentWarnings) > 0 {
		metadata["assistantContentWarnings"] = contentWarnings
	}
	am, err := s.chat.AppendMessage(ctx, threadID, "assistant", content, metadata)
	if err != nil {
		pushStage("ollama_native_stream_save_failed", map[string]any{"error": err.Error()})
		return nil
	}
	pushStage("ollama_native_stream_done", map[string]any{"messageId": am.ID})
	return am
}
