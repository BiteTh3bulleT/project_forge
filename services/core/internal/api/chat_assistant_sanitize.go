package api

import (
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/chat"
)

func forgeAuthorityToolOmissionMessage(forcedModel string) string {
	toolID := strings.TrimSpace(forcedModel)
	if toolID == "" {
		toolID = "requested tool"
	}
	return fmt.Sprintf("FORGE authority boundary: the request was routed to `%s`, but the model returned prose instead of a governed tool call. I discarded the model prose because the model does not decide what FORGE can access or execute. No gateway action ran for this message; retrying should go through FORGE preflight, gateway policy, and approval/capability checks.", toolID)
}

func deterministicNoToolChatReply(th *chat.ThreadDetail, content string) (string, bool) {
	normalized := normalizeAssistantIntent(content)
	if isAssistantIdentityQuery(normalized) {
		return "I am FORGE.", true
	}
	switch normalized {
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
	if trimmedFence := strings.TrimSpace(out); strings.HasSuffix(trimmedFence, "\n---") || trimmedFence == "---" {
		out = strings.TrimSpace(strings.TrimSuffix(trimmedFence, "\n---"))
		warnings = append(warnings, "stripped_synthetic_transcript_turn")
	}
	out, stripped = normalizeAssistantVisibleIdentity(out)
	if stripped {
		warnings = append(warnings, "normalized_model_identity")
	}

	return strings.TrimSpace(out), warnings
}

func stripSyntheticTranscriptContinuation(content string) (string, bool) {
	lower := strings.ToLower(content)
	for _, marker := range []string{
		"\n---\nthread title:",
		"\r\n---\r\nthread title:",
		"\n---\nuser:",
		"\r\n---\r\nuser:",
		"\n---\nassistant",
		"\r\n---\r\nassistant",
	} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			return strings.TrimSpace(content[:idx]), true
		}
	}
	for _, marker := range []string{
		"---\nthread title:",
		"---\r\nthread title:",
		"---\nuser:",
		"---\r\nuser:",
		"---\nassistant",
		"---\r\nassistant",
	} {
		if strings.HasPrefix(lower, marker) {
			return "", true
		}
	}
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
	if lower == "i am forge." || lower == "i am forge" {
		return "I am FORGE.", out != "I am FORGE."
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
