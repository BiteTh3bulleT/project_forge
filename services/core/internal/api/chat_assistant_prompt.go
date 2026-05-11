package api

import (
	"context"
	"strings"

	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/gateway"
)

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
