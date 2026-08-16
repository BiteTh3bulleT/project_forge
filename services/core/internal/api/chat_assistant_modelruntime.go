package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/consensusgate"
	"forge/projectforge/services/core/internal/forgekernel/runtimeproposal"
)

func (s *Server) completeAssistantWithModelRuntimeStream(
	ctx context.Context,
	threadID, userMessageID int64,
	th *chat.ThreadDetail,
	lastUserContent, corr string,
	manifests []map[string]any,
	stages []map[string]any,
	requestedModelID string,
	requestStart time.Time,
	perf chatPerformanceDecision,
	emit func(event string, payload map[string]any),
) (*chat.Message, string) {
	streamRuntime, ok := s.modelRuntime.(modelRuntimeStreamingService)
	if !ok {
		return nil, "model runtime streaming is unavailable"
	}
	if emit == nil {
		return nil, "stream emitter is unavailable"
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

	messages, promptBudget, contextBinding, contextErr := s.prepareGovernedModelRuntimePrompt(ctx, th)
	if contextErr != nil {
		return nil, "FORGE-K Context Compiler failed: " + contextErr.Error()
	}
	preflightTrace, preflightReason := s.modelRuntimeChatStreamPreflight(ctx, meta)
	if strings.TrimSpace(preflightReason) != "" {
		return nil, preflightReason
	}
	stages = append(stages, map[string]any{
		"stage":  "model_runtime_stream_prompt_budget",
		"atMs":   time.Now().UnixMilli(),
		"budget": modelRuntimePromptBudgetMap(promptBudget),
	})
	emit("agent_stage", map[string]any{
		"stage":  "model_runtime_stream_start",
		"atMs":   time.Now().UnixMilli(),
		"model":  modelID,
		"budget": modelRuntimePromptBudgetMap(promptBudget),
	})

	var rawStream strings.Builder
	streamCut := false
	firstTokenMs := int64(0)

	modelStart := time.Now()
	result, err := streamRuntime.StreamChat(ctx, ModelRuntimeChatRequest{
		ModelID:       modelID,
		WorkloadClass: "INTERACTIVE_INFERENCE",
		Messages:      messages,
		MaxTokens:     chatOutputModeMaxTokens(perf.OutputMode),
		TimeoutMs:     modelRuntimePlainChatTimeoutMs,
		MaxAttempts:   modelRuntimePlainChatMaxAttempts,
		Actor:         "chat",
		Source:        "chat_assistant_stream",
		Meta:          meta,
		Metadata: map[string]any{
			"entrypoint":   "api.chat.stream",
			"fallback":     "model_runtime_stream",
			"promptBudget": modelRuntimePromptBudgetMap(promptBudget),
			"outputMode":   perf.OutputMode,
			"budgetClass":  perf.ContextBudgetClass,
		},
	}, func(token ModelRuntimeChatStreamToken) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if token.Done {
			return nil
		}
		if strings.TrimSpace(token.Reasoning) != "" {
			// Reasoning is untrusted driver output and cannot become visible
			// before the final runtime-proposal decision.
			return nil
		}
		if strings.TrimSpace(token.Text) == "" {
			return nil
		}
		if firstTokenMs == 0 {
			firstTokenMs = time.Since(modelStart).Milliseconds()
		}
		rawStream.WriteString(token.Text)
		return nil
	})
	modelRuntimeMs := time.Since(modelStart).Milliseconds()
	if err != nil {
		_, code, message := mapModelRuntimeError(err)
		if strings.TrimSpace(code) != "" {
			return nil, message + " (" + code + ")"
		}
		return nil, message
	}
	if streamCut {
		emit("agent_stage", map[string]any{"stage": "model_runtime_stream_truncated", "atMs": time.Now().UnixMilli(), "reason": "synthetic transcript continuation"})
	}

	driverOutput := result.Content
	if strings.TrimSpace(driverOutput) == "" {
		driverOutput = rawStream.String()
	}
	runtimeDecision, runtimeDecisionErr := decideRuntimeProposal(runtimeProposalRequest{
		SourceKind: runtimeproposal.SourceModelRuntime, WorkspacePath: workspaceID,
		ThreadID: threadID, UserMessageID: userMessageID, CorrelationID: corr,
		Prompt: messages, Output: driverOutput, Backend: result.Backend,
		ModelID: nonEmpty(strings.TrimSpace(result.ModelID), modelID), AuditID: result.AuditID,
		ExecutionID: result.ExecutionID, Proposal: result.Proposal,
		ContextBinding: contextBinding,
	})
	content := runtimeProposalFailureText
	if runtimeDecisionErr == nil {
		content = runtimeDecision.VisibleContent
	}
	content, contentWarnings := sanitizeAssistantVisibleContent(content)
	consensusCandidate, _ := sanitizeAssistantVisibleContent(driverOutput)
	consensusDecision := consensusgate.Gate(consensusgate.Input{
		Content:           consensusCandidate,
		Surface:           consensusgate.SurfaceChatFinal,
		WorkspaceID:       workspaceID,
		CorrelationID:     corr,
		ModelProposalOnly: result.Proposal != nil,
	})
	if runtimeDecisionErr == nil && runtimeDecision.Status == runtimeproposal.StatusAccepted {
		content = consensusDecision.Content
	}
	if strings.TrimSpace(content) == "" {
		content = assistantContentFallback
	}
	traceExtras := map[string]any{
		"modelruntime_ms":             modelRuntimeMs,
		"modelruntime_first_token_ms": firstTokenMs,
		"gateway_execution_ms":        int64(0),
		"model_calls_avoided":         0,
		"output_max_tokens":           chatOutputModeMaxTokens(perf.OutputMode),
	}
	for k, v := range preflightTrace {
		traceExtras[k] = v
	}
	trace := chatLatencyTraceWithPrompt(requestStart, perf, promptBudget, traceExtras)
	s.warnIfChatLatencyBudgetExceeded(ctx, threadID, userMessageID, corr, trace)
	activity := map[string]any{
		"userRequestSummary": trimSummary(lastUserContent, 500),
		"toolManifest":       manifests,
		"stages":             stages,
		"toolPipeline":       map[string]any{"stages": stages},
		"toolCallEmitted":    false,
		"executionState":     "skipped",
		"fallback":           "model_runtime_stream",
		"modelId":            nonEmpty(strings.TrimSpace(result.ModelID), modelID),
		"backend":            strings.TrimSpace(result.Backend),
		"promptBudget":       modelRuntimePromptBudgetMap(promptBudget),
		"latencyTrace":       trace,
		"contextBudgetClass": perf.ContextBudgetClass,
		"outputMode":         perf.OutputMode,
	}
	if requested := strings.TrimSpace(requestedModelID); requested != "" {
		activity["requestedModelId"] = requested
	}

	metadata := map[string]any{
		"replyToUserMessageId": userMessageID,
		"correlationId":        corr,
		"modelRuntimeOk":       true,
		"modelRuntimeStream":   true,
		"modelRuntimeModelId":  nonEmpty(strings.TrimSpace(result.ModelID), modelID),
		"modelRuntimeBackend":  strings.TrimSpace(result.Backend),
		"modelRuntimePrompt":   modelRuntimePromptBudgetMap(promptBudget),
		"chatLatencyTrace":     trace,
		"toolManifest":         manifests,
		"toolPipeline":         map[string]any{"stages": stages},
		"toolGatewayActivity":  activity,
		"consensusGate":        consensusDecision,
		"runtimeProposal":      runtimeProposalEvidence(runtimeDecision),
		"runtimeProposalError": runtimeProposalError(runtimeDecisionErr),
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
	if len(contentWarnings) > 0 {
		metadata["assistantContentWarnings"] = contentWarnings
	}

	am, err := s.chat.AppendMessage(ctx, threadID, "assistant", content, metadata)
	if err != nil {
		return nil, "assistant reply could not be saved"
	}
	emit("token", map[string]any{"text": content, "runtimeProposalDecision": runtimeDecision.DecisionDigest})
	emit("agent_stage", map[string]any{"stage": "model_runtime_stream_done", "atMs": time.Now().UnixMilli(), "messageId": am.ID})
	return am, ""
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
	requestStart time.Time,
	perf chatPerformanceDecision,
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

	messages, promptBudget, contextBinding, contextErr := s.prepareGovernedModelRuntimePrompt(ctx, th)
	if contextErr != nil {
		return nil, "FORGE-K Context Compiler failed: " + contextErr.Error()
	}
	preflightTrace, preflightReason := s.modelRuntimeChatPreflight(ctx, meta, modelID)
	if strings.TrimSpace(preflightReason) != "" {
		return nil, preflightReason
	}
	stages = append(stages, map[string]any{
		"stage":  "model_runtime_prompt_budget",
		"atMs":   time.Now().UnixMilli(),
		"budget": modelRuntimePromptBudgetMap(promptBudget),
	})
	modelStart := time.Now()
	result, err := s.modelRuntime.Chat(ctx, ModelRuntimeChatRequest{
		ModelID:       modelID,
		WorkloadClass: "INTERACTIVE_INFERENCE",
		Messages:      messages,
		MaxTokens:     chatOutputModeMaxTokens(perf.OutputMode),
		TimeoutMs:     modelRuntimePlainChatTimeoutMs,
		MaxAttempts:   modelRuntimePlainChatMaxAttempts,
		Actor:         "chat",
		Source:        "chat_assistant",
		Meta:          meta,
		Metadata: map[string]any{
			"entrypoint":   "api.chat",
			"fallback":     "model_runtime",
			"promptBudget": modelRuntimePromptBudgetMap(promptBudget),
			"outputMode":   perf.OutputMode,
			"budgetClass":  perf.ContextBudgetClass,
		},
	})
	modelRuntimeMs := time.Since(modelStart).Milliseconds()
	if err != nil {
		_, code, message := mapModelRuntimeError(err)
		if strings.TrimSpace(code) != "" {
			return nil, message + " (" + code + ")"
		}
		return nil, message
	}

	runtimeDecision, runtimeDecisionErr := decideRuntimeProposal(runtimeProposalRequest{
		SourceKind: runtimeproposal.SourceModelRuntime, WorkspacePath: workspaceID,
		ThreadID: threadID, UserMessageID: userMessageID, CorrelationID: corr,
		Prompt: messages, Output: result.Content, Backend: result.Backend,
		ModelID: nonEmpty(strings.TrimSpace(result.ModelID), modelID), AuditID: result.AuditID,
		ExecutionID: result.ExecutionID, Proposal: result.Proposal,
		ContextBinding: contextBinding,
	})
	content := runtimeProposalFailureText
	if runtimeDecisionErr == nil {
		content = runtimeDecision.VisibleContent
	}
	content, contentWarnings := sanitizeAssistantVisibleContent(content)
	consensusCandidate, _ := sanitizeAssistantVisibleContent(result.Content)
	consensusDecision := consensusgate.Gate(consensusgate.Input{
		Content:           consensusCandidate,
		Surface:           consensusgate.SurfaceChatFinal,
		WorkspaceID:       workspaceID,
		CorrelationID:     corr,
		ModelProposalOnly: result.Proposal != nil,
	})
	if runtimeDecisionErr == nil && runtimeDecision.Status == runtimeproposal.StatusAccepted {
		content = consensusDecision.Content
	}
	if strings.TrimSpace(content) == "" {
		content = assistantContentFallback
	}
	traceExtras := map[string]any{
		"modelruntime_ms":      modelRuntimeMs,
		"gateway_execution_ms": int64(0),
		"model_calls_avoided":  0,
		"output_max_tokens":    chatOutputModeMaxTokens(perf.OutputMode),
	}
	for k, v := range preflightTrace {
		traceExtras[k] = v
	}
	trace := chatLatencyTraceWithPrompt(requestStart, perf, promptBudget, traceExtras)
	s.warnIfChatLatencyBudgetExceeded(ctx, threadID, userMessageID, corr, trace)

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
		"promptBudget":       modelRuntimePromptBudgetMap(promptBudget),
		"latencyTrace":       trace,
		"contextBudgetClass": perf.ContextBudgetClass,
		"outputMode":         perf.OutputMode,
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
		"modelRuntimePrompt":   modelRuntimePromptBudgetMap(promptBudget),
		"chatLatencyTrace":     trace,
		"toolManifest":         manifests,
		"toolPipeline":         map[string]any{"stages": stages},
		"toolGatewayActivity":  activity,
		"consensusGate":        consensusDecision,
		"runtimeProposal":      runtimeProposalEvidence(runtimeDecision),
		"runtimeProposalError": runtimeProposalError(runtimeDecisionErr),
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
	if len(contentWarnings) > 0 {
		metadata["assistantContentWarnings"] = contentWarnings
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

	models, err := s.modelRuntime.ListModels(ctx, ModelRuntimeListRequest{Meta: meta, SkipDiscovery: true})
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
