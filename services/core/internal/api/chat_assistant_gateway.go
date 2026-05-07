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
	assistantStreamFlushChars           = 160
	assistantStreamFlushIntervalMs      = 24
)

const assistantContentFallback = "I couldn't produce a clean assistant response. Try again, or check the selected model/runtime."

type modelRuntimePromptBudget struct {
	ThreadMessages         int
	IncludedMessages       int
	TruncatedMessages      int
	TranscriptChars        int
	MemoryChars            int
	CrossThreadMemoryChars int
	ObservationMemoryChars int
	AttachmentChars        int
	UserChars              int
	SystemChars            int
	TotalChars             int
	Compacted              bool
	AttachmentsTrimmed     bool
}

func (s *Server) buildChatLLMMessages(ctx context.Context, th *chat.ThreadDetail) (system string, user string) {
	transcript := s.chat.BuildTranscript(th.Messages, chatTranscriptTurns)
	sys := s.chatOperatorSystemPrompt()
	if s.gateway != nil {
		sys += "\n\n" + s.gateway.ChatSystemSupplement()
	}
	user = "---\nTHREAD TITLE: " + th.Title + "\n---\n" + transcript
	if memoryContext := buildPersistedThreadMemoryContext(th.Messages, chatTranscriptTurns, chatThreadMemoryContextMaxMessages, chatThreadMemoryContextMaxRunes); memoryContext != "" {
		user += "\n\n---\nEARLIER THREAD MEMORY\n" + memoryContext
	}
	if crossThreadMemory := s.buildCrossThreadChatContext(ctx, th.ID, chatCrossThreadContextMaxMessages, chatCrossThreadContextMaxRunes); crossThreadMemory != "" {
		user += "\n\n---\nRELATED CHAT MEMORY\n" + crossThreadMemory
	}
	if observationMemory := s.buildMemoryObservationContext(ctx, th.DossierID, chatMemoryObservationMaxItems, chatMemoryObservationMaxRunes); observationMemory != "" {
		user += "\n\n---\nMEMORY OBSERVATIONS\n" + observationMemory
	}
	if att := s.buildThreadAttachmentContext(ctx, th); att != "" {
		user += "\n\n---\nATTACHMENTS CONTEXT\n" + att
	}
	return sys, user
}

func (s *Server) buildModelRuntimePlainChatMessages(ctx context.Context, th *chat.ThreadDetail) ([]ModelRuntimeChatMessage, modelRuntimePromptBudget) {
	budget := modelRuntimePromptBudget{ThreadMessages: len(th.Messages)}
	systemBase := s.chatOperatorSystemPrompt()
	systemSections := []string{}

	start := 0
	if len(th.Messages) > modelRuntimePlainChatMessages {
		start = len(th.Messages) - modelRuntimePlainChatMessages
		budget.Compacted = true
	}

	if title := strings.TrimSpace(th.Title); title != "" {
		systemSections = append(systemSections, "Thread title: "+trimSummary(title, 160))
	}
	if memoryContext := buildPersistedThreadMemoryContext(th.Messages, modelRuntimePlainChatMessages, 6, modelRuntimePlainChatMemoryMax); memoryContext != "" {
		budget.MemoryChars = len(memoryContext)
		systemSections = append(systemSections, "Earlier thread memory:\n"+memoryContext)
	}
	if crossThreadMemory := s.buildCrossThreadChatContext(ctx, th.ID, 4, 800); crossThreadMemory != "" {
		budget.CrossThreadMemoryChars = len(crossThreadMemory)
		systemSections = append(systemSections, "Related chat memory:\n"+crossThreadMemory)
	}
	if observationMemory := s.buildMemoryObservationContext(ctx, th.DossierID, 4, 800); observationMemory != "" {
		budget.ObservationMemoryChars = len(observationMemory)
		systemSections = append(systemSections, "Memory observations:\n"+observationMemory)
	}
	if budget.Compacted {
		systemSections = append(systemSections, "Recent chat context was compacted for local model runtime latency. Answer only the latest operator turn.")
	}
	system := assembleBoundedSystemPrompt(systemBase, systemSections, modelRuntimePlainChatSystemMax)
	budget.SystemChars = len(system)

	messages := []ModelRuntimeChatMessage{{Role: "system", Content: system}}
	for _, msg := range th.Messages[start:] {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		if role != "assistant" {
			role = "user"
		}
		content := strings.TrimSpace(msg.Content)
		if len(content) > modelRuntimePlainChatMessageMax {
			content = trimSummary(content, modelRuntimePlainChatMessageMax)
			budget.TruncatedMessages++
			budget.Compacted = true
		}
		if content == "" {
			continue
		}
		messages = append(messages, ModelRuntimeChatMessage{Role: role, Content: content})
		budget.TranscriptChars += len(content)
		budget.IncludedMessages++
	}

	if att := strings.TrimSpace(s.buildThreadAttachmentContext(ctx, th)); att != "" {
		trimmed := trimSummary(att, modelRuntimePlainChatAttachmentMax)
		if len(trimmed) < len(att) {
			budget.AttachmentsTrimmed = true
			budget.Compacted = true
		}
		budget.AttachmentChars = len(trimmed)
		attachmentMessage := "Relevant attachment excerpts:\n" + trimmed
		if len(messages) > 1 && messages[len(messages)-1].Role == "user" {
			messages[len(messages)-1].Content = trimSummary(messages[len(messages)-1].Content+"\n\n"+attachmentMessage, modelRuntimePlainChatUserMax)
		} else {
			messages = append(messages, ModelRuntimeChatMessage{Role: "user", Content: attachmentMessage})
		}
	}

	totalUserChars := 0
	for i := range messages {
		if messages[i].Role == "user" {
			if len(messages[i].Content) > modelRuntimePlainChatUserMax {
				messages[i].Content = trimSummary(messages[i].Content, modelRuntimePlainChatUserMax)
				budget.Compacted = true
			}
			totalUserChars += len(messages[i].Content)
		}
	}
	budget.UserChars = totalUserChars
	budget.TotalChars = budget.SystemChars + budget.UserChars

	return messages, budget
}

func assembleBoundedSystemPrompt(base string, sections []string, max int) string {
	base = strings.TrimSpace(base)
	cleanSections := make([]string, 0, len(sections))
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section != "" {
			cleanSections = append(cleanSections, section)
		}
	}
	if len(cleanSections) == 0 {
		return trimSummary(base, max)
	}
	suffix := strings.Join(cleanSections, "\n\n")
	if max <= 0 {
		return strings.TrimSpace(base + "\n\n" + suffix)
	}
	separator := "\n\n"
	suffixBudget := len(separator) + len(suffix)
	if suffixBudget >= max {
		return trimSummary(suffix, max)
	}
	baseBudget := max - suffixBudget - len("…")
	if baseBudget < 0 {
		baseBudget = 0
	}
	base = trimSummary(base, baseBudget)
	return strings.TrimSpace(base + separator + suffix)
}

func (s *Server) forcedToolCallMismatch(forcedModel string, toolCalls []map[string]any) (bool, []string) {
	forcedModel = strings.TrimSpace(forcedModel)
	if forcedModel == "" || len(toolCalls) == 0 || s.gateway == nil {
		return false, nil
	}
	selected := make([]string, 0, len(toolCalls))
	for _, call := range toolCalls {
		fn, _ := call["function"].(map[string]any)
		name := strings.TrimSpace(asString(fn["name"]))
		if name == "" {
			selected = append(selected, "")
			return true, selected
		}
		toolID, _, resolved := s.gateway.ResolveChatFunctionName(name)
		if !resolved {
			selected = append(selected, name)
			return true, selected
		}
		resolvedModel := gateway.ChatModelName(toolID)
		selected = append(selected, resolvedModel)
		if resolvedModel != forcedModel {
			return true, selected
		}
	}
	return false, selected
}

func chatActivityState(activity map[string]any, fallback string) string {
	if activity == nil {
		return fallback
	}
	state := strings.TrimSpace(asString(activity["executionState"]))
	if state == "" {
		return fallback
	}
	return state
}

func modelRuntimePromptBudgetMap(b modelRuntimePromptBudget) map[string]any {
	return map[string]any{
		"threadMessages":         b.ThreadMessages,
		"includedMessages":       b.IncludedMessages,
		"truncatedMessages":      b.TruncatedMessages,
		"transcriptChars":        b.TranscriptChars,
		"memoryChars":            b.MemoryChars,
		"crossThreadMemoryChars": b.CrossThreadMemoryChars,
		"observationMemoryChars": b.ObservationMemoryChars,
		"attachmentChars":        b.AttachmentChars,
		"userChars":              b.UserChars,
		"systemChars":            b.SystemChars,
		"totalChars":             b.TotalChars,
		"compacted":              b.Compacted,
		"attachmentsTrimmed":     b.AttachmentsTrimmed,
	}
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

	if !gateway.ShouldAttachChatTools(lastUserContent) {
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

	loadToolCatalog()
	pushStage("tools_attached", map[string]any{"tools": toolNames, "count": len(toolNames)})
	forcedModel := gateway.ForcedChatModelName(lastUserContent)

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
	if forcedModel != gateway.ChatModelName("desktop.open") {
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
	if forcedModel == gateway.ChatModelName("desktop.open") {
		pushStage("deterministic_desktop_shortcut", map[string]any{"reason": "explicit desktop or terminal intent"})
		gwActivity := map[string]any{
			"userRequestSummary": trimSummary(lastUserContent, 500),
			"toolManifest":       manifests,
			"stages":             stages,
			"toolCallEmitted":    false,
		}
		trackedActivity = gwActivity
		var final strings.Builder
		s.runChatFSDeterministicFallback(ctx, corr, lastUserContent, forcedModel, pushStage, gwActivity, &final)
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
	if forcedModel == gateway.ChatModelName("repo.inspect") {
		pushStage("deterministic_repo_inspect_shortcut", map[string]any{"reason": "explicit repo orientation intent"})
		gwActivity := map[string]any{
			"userRequestSummary": trimSummary(lastUserContent, 500),
			"toolManifest":       manifests,
			"stages":             stages,
			"toolCallEmitted":    false,
		}
		trackedActivity = gwActivity
		var final strings.Builder
		s.runChatFSDeterministicFallback(ctx, corr, lastUserContent, forcedModel, pushStage, gwActivity, &final)
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
	model := ol.ModelForChat(ctx)
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

	sys, userBody := s.buildChatLLMMessages(ctx, th)
	msgs := []map[string]any{
		{"role": "system", "content": sys},
		{"role": "user", "content": userBody},
	}

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

		if forcedModel != "" && toolEmitted {
			if mismatch, selected := s.forcedToolCallMismatch(forcedModel, toolCalls); mismatch {
				pushStage("forced_tool_mismatch_discarded", map[string]any{
					"turn":     turn,
					"forced":   forcedModel,
					"selected": selected,
				})
				gwActivity["modelToolCallsDiscarded"] = true
				gwActivity["discardedToolCalls"] = selected
				fallbackOnly := strings.Builder{}
				fallbackRan = s.runChatFSDeterministicFallback(ctx, corr, lastUserContent, forcedModel, pushStage, gwActivity, &fallbackOnly)
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
				final.WriteString(forgeAuthorityToolOmissionMessage(forcedModel))
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
				fallbackRan = s.runChatFSDeterministicFallback(ctx, corr, lastUserContent, forcedModel, pushStage, gwActivity, &fallbackOnly)
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
				if forcedModel != "" {
					if strings.TrimSpace(content) != "" {
						pushStage("model_prose_discarded", map[string]any{
							"reason": "forced tool route omitted tool_calls; model prose cannot decide FORGE capability or availability",
						})
						gwActivity["modelProseDiscarded"] = true
					}
					final.WriteString(forgeAuthorityToolOmissionMessage(forcedModel))
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

	if emit != nil {
		if am := s.completeAssistantWithNativeOllamaStream(ctx, threadID, userMessageID, th, lastUserContent, ollamaAdapter, corr, getManifests, stages, pushStage, emit, requestStart, perf); am != nil {
			return am
		}
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

	messages, promptBudget := s.buildModelRuntimePlainChatMessages(ctx, th)
	preflightTrace, preflightReason := s.modelRuntimeChatPreflight(ctx, meta)
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
	lastFlush := time.Time{}
	emittedFirst := false
	emittedChars := 0
	streamCut := false
	firstTokenMs := int64(0)
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
		if token.Done {
			return nil
		}
		if strings.TrimSpace(token.Text) == "" {
			return nil
		}
		if firstTokenMs == 0 {
			firstTokenMs = time.Since(modelStart).Milliseconds()
		}
		rawStream.WriteString(token.Text)
		flushVisible(false)
		return nil
	})
	flushVisible(true)
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

	content, contentWarnings := sanitizeAssistantVisibleContent(result.Content)
	if content == "" {
		stripped, _ := stripSyntheticTranscriptContinuation(rawStream.String())
		content = strings.TrimSpace(stripped)
	}
	if content == "" {
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

	messages, promptBudget := s.buildModelRuntimePlainChatMessages(ctx, th)
	preflightTrace, preflightReason := s.modelRuntimeChatPreflight(ctx, meta)
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

	content, contentWarnings := sanitizeAssistantVisibleContent(result.Content)
	if content == "" {
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

func forgeAuthorityToolOmissionMessage(forcedModel string) string {
	toolID := strings.TrimSpace(forcedModel)
	if toolID == "" {
		toolID = "requested tool"
	}
	return fmt.Sprintf("FORGE authority boundary: the request was routed to `%s`, but the model returned prose instead of a governed tool call. I discarded the model prose because the model does not decide what FORGE can access or execute. No gateway action ran for this message; retrying should go through FORGE preflight, gateway policy, and approval/capability checks.", toolID)
}

func deterministicNoToolChatReply(th *chat.ThreadDetail, content string) (string, bool) {
	normalized := normalizeAssistantIntent(content)
	switch normalized {
	case "what is your name", "whats your name", "who are you", "what are you":
		return "I am FORGE.", true
	case "what is my name", "whats my name", "who am i":
		if name := latestOperatorNameFromThread(th); name != "" {
			return "Your name is " + name + ".", true
		}
		return "I don't have your name in this thread.", true
	}
	if isWeatherWithoutLocationQuery(normalized) {
		return "What city or ZIP code should I check for the weather?", true
	}
	return "", false
}

func latestOperatorNameFromThread(th *chat.ThreadDetail) string {
	if th == nil {
		return ""
	}
	for i := len(th.Messages) - 1; i >= 0; i-- {
		msg := th.Messages[i]
		if msg.Role != "user" {
			continue
		}
		if name := parseOperatorNameClaim(msg.Content); name != "" {
			return name
		}
	}
	return ""
}

func parseOperatorNameClaim(content string) string {
	text := strings.TrimSpace(content)
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	idx := strings.LastIndex(lower, "my name is")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(text[idx+len("my name is"):])
	rest = strings.Trim(rest, " \t\r\n:,-")
	if rest == "" {
		return ""
	}
	fields := strings.Fields(rest)
	parts := make([]string, 0, 3)
	for _, field := range fields {
		token := strings.Trim(field, `"'()[]{}<>`)
		token = strings.TrimRight(token, ".!,?;:")
		if token == "" {
			break
		}
		parts = append(parts, token)
		if strings.ContainsAny(field, ".!,?;:") || len(parts) >= 3 {
			break
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func normalizeAssistantIntent(content string) string {
	s := strings.ToLower(strings.TrimSpace(content))
	s = strings.Trim(s, " \t\r\n?!.,")
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "what's", "whats")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}

func isWeatherWithoutLocationQuery(normalized string) bool {
	if normalized == "" || !strings.Contains(normalized, "weather") {
		return false
	}
	if strings.Contains(normalized, " in ") || strings.Contains(normalized, " for ") || strings.Contains(normalized, " at ") {
		return false
	}
	if strings.Contains(normalized, "weather today") || strings.Contains(normalized, "weather looking like today") || strings.Contains(normalized, "forecast today") {
		return true
	}
	return normalized == "weather" || normalized == "what is the weather" || normalized == "whats the weather"
}

func sanitizeAssistantVisibleContent(content string) (string, []string) {
	out := strings.TrimSpace(content)
	warnings := []string(nil)
	var stripped bool

	out, stripped = stripDelimitedAssistantBlock(out, "<think>", "</think>")
	if stripped {
		warnings = append(warnings, "stripped_hidden_thinking_block")
	}
	out, stripped = stripDelimitedAssistantBlock(out, "<thinking>", "</thinking>")
	if stripped {
		warnings = append(warnings, "stripped_hidden_thinking_block")
	}
	out, stripped = stripLeadingReasoningScaffold(out)
	if stripped {
		warnings = append(warnings, "stripped_reasoning_scaffold")
	}
	if idx := assistantLineMarkerIndex(out, []string{"TRACEABILITY"}); idx >= 0 {
		out = strings.TrimSpace(out[:idx])
		warnings = append(warnings, "stripped_traceability_scaffold")
	}
	if strippedOut, cut := stripSyntheticTranscriptContinuation(out); cut {
		out = strings.TrimSpace(strippedOut)
		warnings = append(warnings, "stripped_synthetic_transcript_turn")
	}
	out, stripped = normalizeAssistantVisibleIdentity(out)
	if stripped {
		warnings = append(warnings, "normalized_model_identity")
	}

	return strings.TrimSpace(out), warnings
}

func stripSyntheticTranscriptContinuation(content string) (string, bool) {
	markers := []string{
		"USER",
		"YOU",
		"ASSISTANT",
		"FORGE",
		"OPERATOR",
	}
	if idx := assistantLineMarkerIndex(content, markers); idx >= 0 {
		return strings.TrimSpace(content[:idx]), true
	}
	return content, false
}

func normalizeAssistantVisibleIdentity(content string) (string, bool) {
	out := strings.TrimSpace(content)
	lower := strings.ToLower(out)
	prefixes := []string{
		"i am phi",
		"i'm phi",
		"my name is phi",
		"this is phi",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			if idx := strings.Index(out, "."); idx >= 0 {
				rest := strings.TrimSpace(out[idx+1:])
				if rest == "" {
					return "I am FORGE.", true
				}
				return "I am FORGE. " + rest, true
			}
			return "I am FORGE.", true
		}
	}
	return out, false
}

func stripDelimitedAssistantBlock(content, openMarker, closeMarker string) (string, bool) {
	out := strings.TrimSpace(content)
	stripped := false
	for {
		lower := strings.ToLower(out)
		start := strings.Index(lower, strings.ToLower(openMarker))
		if start < 0 {
			return strings.TrimSpace(out), stripped
		}
		endRel := strings.Index(lower[start+len(openMarker):], strings.ToLower(closeMarker))
		if endRel < 0 {
			if start == 0 {
				return "", true
			}
			return strings.TrimSpace(out[:start]), true
		}
		end := start + len(openMarker) + endRel + len(closeMarker)
		out = strings.TrimSpace(out[:start] + "\n" + out[end:])
		stripped = true
	}
}

func stripLeadingReasoningScaffold(content string) (string, bool) {
	out := strings.TrimSpace(content)
	if out == "" {
		return "", false
	}
	lower := strings.ToLower(out)
	reasoningMarkers := []string{
		"thinking process:",
		"reasoning process:",
		"internal reasoning:",
		"chain of thought:",
		"first, the user said:",
		"user's latest input:",
		"we need to answer:",
		"we need answer:",
		"my response should",
		"analysis:",
		"reasoning:",
	}
	leaksReasoning := false
	for _, marker := range reasoningMarkers {
		if strings.HasPrefix(lower, marker) {
			leaksReasoning = true
			break
		}
	}
	if !leaksReasoning {
		return out, false
	}
	if idx, markerLen := assistantFinalMarkerIndex(out); idx >= 0 {
		return strings.TrimSpace(out[idx+markerLen:]), true
	}
	return "", true
}

func assistantFinalMarkerIndex(content string) (int, int) {
	lower := strings.ToLower(content)
	for _, marker := range []string{"final answer:", "final:", "answer:", "response:"} {
		if strings.HasPrefix(lower, marker) {
			return 0, len(marker)
		}
		pattern := "\n" + marker
		if idx := strings.Index(lower, pattern); idx >= 0 {
			return idx + 1, len(marker)
		}
	}
	return -1, 0
}

func assistantLineMarkerIndex(content string, markers []string) int {
	offset := 0
	for _, line := range strings.SplitAfter(content, "\n") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
		for _, marker := range markers {
			if strings.EqualFold(trimmed, marker) || strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(marker)+":") {
				return offset
			}
		}
		offset += len(line)
	}
	return -1
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
