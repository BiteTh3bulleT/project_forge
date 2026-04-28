package gateway

import (
	"fmt"
	"hash/fnv"
	"strings"

	"forge/projectforge/services/core/internal/aios/hyperlane"
)

// ParseHyperlaneIntent classifies simple operator requests into CPU-only route
// proposals. It never executes tools, calls modelruntime, or mutates state.
func ParseHyperlaneIntent(user string) hyperlane.Intent {
	return ParseHyperlaneIntentWithDirectoryHint(user, "")
}

// SupportsNoModelHyperlaneRoute reports whether chat/process may answer the
// intent from structured state without modelruntime or gateway execution.
func SupportsNoModelHyperlaneRoute(intent hyperlane.Intent) bool {
	return hyperlane.SupportsNoModelRoute(intent)
}

// ParseHyperlaneIntentWithDirectoryHint allows gateway-derived follow-up context
// for deterministic template requests. The hint is still only a proposed path.
func ParseHyperlaneIntentWithDirectoryHint(user, dirHint string) hyperlane.Intent {
	raw := strings.TrimSpace(user)
	id := hyperlaneIntentID(raw)
	if raw == "" {
		return hyperlane.UnknownIntent(id, "empty input", nil)
	}
	lower := strings.Trim(strings.ToLower(raw), " \t\r\n?!.")

	if isModelruntimeStatusQuery(lower) {
		return routeIntent(id, hyperlane.IntentModelruntimeStatus, 0.91, "runtime", hyperlane.RouteModelruntimeStatus, false, false, false, "none", "modelruntime_status_query", nil, nil)
	}
	if isChatHistoryLookupQuery(lower) {
		return routeIntent(id, hyperlane.IntentChatHistoryLookup, 0.9, "operator", hyperlane.RouteChatHistoryLookup, false, false, false, "none", "chat_history_lookup_query", map[string]any{"query": raw}, nil)
	}
	if isChatMemoryInspectionQuery(lower) {
		return routeIntent(id, hyperlane.IntentChatMemoryInspection, 0.88, "operator", hyperlane.RouteChatMemoryInspector, false, false, false, "none", "chat_memory_inspection_query", nil, nil)
	}
	if isDiagnosticsQuery(lower) {
		return routeIntent(id, hyperlane.IntentDiagnosticsQuery, 0.89, "operator", hyperlane.RouteStructuredDiagnostics, false, false, false, "none", "diagnostics_query", nil, nil)
	}
	if isRestoreInspectionQuery(lower) {
		return routeIntent(id, hyperlane.IntentRestoreInspection, 0.87, "arterial", hyperlane.RouteRestoreInspector, false, false, false, "none", "restore_inspection_query", nil, nil)
	}
	if isDreamReportInspectionQuery(lower) {
		return routeIntent(id, hyperlane.IntentDreamReportInspection, 0.87, "lymphatic", hyperlane.RouteDreamReportInspector, false, false, false, "none", "dream_report_inspection_query", nil, nil)
	}
	if isStatusQuery(lower) {
		return routeIntent(id, hyperlane.IntentStatusQuery, 0.86, "operator", hyperlane.RouteStructuredStatus, false, false, false, "none", "status_query", nil, nil)
	}

	if dirHint != "" {
		if path, content, ok := ParseVideoGameJournalWebpageIntent(raw, dirHint); ok {
			return routeIntent(id, hyperlane.IntentGenerateTemplate, 0.81, "kernel", hyperlane.RouteGatewayFSWrite, true, false, true, "medium", "template_video_game_journal", map[string]any{
				"path":     path,
				"contents": content,
			}, nil)
		}
	}

	fallback := ParseFallbackIntent(raw)
	switch fallback.Type {
	case FallbackIntentMkdir:
		return routeIntent(id, hyperlane.IntentMkdir, fallback.Confidence, "kernel", hyperlane.RouteGatewayFSMkdir, true, false, fallback.RequiresApproval, "medium", "fallback_mkdir", map[string]any{"path": fallback.TargetPath}, fallback.Warnings)
	case FallbackIntentWriteFile:
		return routeIntent(id, hyperlane.IntentWriteFile, fallback.Confidence, "kernel", hyperlane.RouteGatewayFSWrite, true, false, fallback.RequiresApproval, "medium", "fallback_write_file", map[string]any{"path": fallback.TargetPath, "contents": fallback.Content}, fallback.Warnings)
	case FallbackIntentReadFile:
		return routeIntent(id, hyperlane.IntentReadFile, fallback.Confidence, "kernel", hyperlane.RouteGatewayFSRead, true, false, false, "low", "fallback_read_file", map[string]any{"path": fallback.TargetPath}, fallback.Warnings)
	case FallbackIntentListDirectory:
		return routeIntent(id, hyperlane.IntentListDirectory, fallback.Confidence, "kernel", hyperlane.RouteGatewayFSList, true, false, false, "low", "fallback_list_directory", map[string]any{"path": fallback.TargetPath}, fallback.Warnings)
	case FallbackIntentRunCommand:
		warnings := append([]string{"command routing proposal only; gateway policy must approve execution"}, fallback.Warnings...)
		return routeIntent(id, hyperlane.IntentRunCommand, fallback.Confidence, "kernel", hyperlane.RouteGatewayProcRun, true, false, true, "high", "fallback_run_command", map[string]any{"command": fallback.Command}, warnings)
	case FallbackIntentGenerateTemplate:
		return routeIntent(id, hyperlane.IntentGenerateTemplate, fallback.Confidence, "kernel", hyperlane.RouteGatewayFSWrite, true, false, true, "medium", "fallback_generate_template", map[string]any{"path": fallback.TargetPath, "contents": fallback.Content}, fallback.Warnings)
	}

	if route, confidence, matched := gatewayToolRouteHint(lower); route != "" {
		return routeIntent(id, hyperlane.IntentGatewayToolRequest, confidence, "kernel", route, true, false, route == hyperlane.RouteGatewayProcRun, riskClassForRoute(route), matched, nil, nil)
	}
	return hyperlane.UnknownIntent(id, rejectedReasonForUnknown(raw), nil)
}

func routeIntent(id string, typ hyperlane.IntentType, confidence float64, lane, route string, requiresGateway, requiresModel, requiresApproval bool, riskClass, matchedRule string, args map[string]any, warnings []string) hyperlane.Intent {
	warnings = append([]string(nil), warnings...)
	return hyperlane.Intent{
		ID:                   id,
		Type:                 typ,
		Confidence:           confidence,
		Lane:                 lane,
		Route:                route,
		RequiresGateway:      requiresGateway,
		RequiresModel:        requiresModel,
		RequiresApprovalHint: requiresApproval,
		RiskClass:            riskClass,
		Arguments:            args,
		Warnings:             warnings,
		MatchedRule:          matchedRule,
		Trace: hyperlane.IntentTrace{
			ParserVersion: hyperlane.ParserVersion,
			MatchedRule:   matchedRule,
			Confidence:    confidence,
			Route:         route,
			Warnings:      warnings,
		},
	}
}

func hyperlaneIntentID(input string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(input))))
	return fmt.Sprintf("hyperlane-intent-%016x", h.Sum64())
}

func isStatusQuery(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if s == "status" || s == "health" || s == "what mode are we in" || s == "what mode is forge in" {
		return true
	}
	return strings.Contains(s, "core status") ||
		strings.Contains(s, "forge status") ||
		strings.Contains(s, "system status") ||
		strings.Contains(s, "what is the status") ||
		strings.Contains(s, "is forge online") ||
		strings.Contains(s, "what is degraded")
}

func isDiagnosticsQuery(s string) bool {
	return strings.Contains(s, "diagnostic") ||
		strings.Contains(s, "diagnose") ||
		strings.Contains(s, "backend disabled") ||
		strings.Contains(s, "what is degraded") ||
		strings.Contains(s, "show degraded") ||
		strings.Contains(s, "why is forge slow")
}

func isChatMemoryInspectionQuery(s string) bool {
	return (strings.Contains(s, "cross chat") || strings.Contains(s, "cross-chat") || strings.Contains(s, "chat context") || strings.Contains(s, "chat memory") || strings.Contains(s, "conversation history")) &&
		(strings.Contains(s, "do we have") || strings.Contains(s, "implemented") || strings.Contains(s, "status") || strings.Contains(s, "active") || strings.Contains(s, "available") || strings.Contains(s, "memory") || strings.Contains(s, "context"))
}

func isChatHistoryLookupQuery(s string) bool {
	if !(strings.Contains(s, "what did i ask") || strings.Contains(s, "what did i say") || strings.Contains(s, "what was my request") || strings.Contains(s, "what did the user ask")) {
		return false
	}
	return strings.Contains(s, " on ") || strings.HasPrefix(s, "on ") || strings.Contains(s, "/")
}

func isRestoreInspectionQuery(s string) bool {
	return (strings.Contains(s, "restore") && (strings.Contains(s, "inspect") || strings.Contains(s, "score") || strings.Contains(s, "decision") || strings.Contains(s, "summary") || strings.Contains(s, "recent"))) ||
		strings.Contains(s, "latest context snapshot") ||
		strings.Contains(s, "show recent restore decisions")
}

func isDreamReportInspectionQuery(s string) bool {
	return strings.Contains(s, "dream") && (strings.Contains(s, "report") || strings.Contains(s, "latest") || strings.Contains(s, "review") || strings.Contains(s, "inspect"))
}

func isModelruntimeStatusQuery(s string) bool {
	return (strings.Contains(s, "modelruntime") || strings.Contains(s, "model runtime") || strings.Contains(s, "runtime registry") || strings.Contains(s, "model registry")) &&
		(strings.Contains(s, "status") || strings.Contains(s, "working") || strings.Contains(s, "online") || strings.Contains(s, "load") || strings.Contains(s, "available"))
}

func gatewayToolRouteHint(s string) (route string, confidence float64, matchedRule string) {
	switch {
	case wantsWebSearch(s):
		return hyperlane.RouteGatewayWebSearch, 0.72, "gateway_web_search"
	case wantsURLFetch(s):
		return hyperlane.RouteGatewayNetFetch, 0.72, "gateway_url_fetch"
	case wantsBrowserOpen(s):
		return hyperlane.RouteGatewayDesktopOpen, 0.72, "gateway_browser_open"
	case wantsGitStatus(s):
		return hyperlane.RouteGatewayGitStatus, 0.74, "gateway_git_status"
	default:
		return "", 0, ""
	}
}

func riskClassForRoute(route string) string {
	switch route {
	case hyperlane.RouteGatewayProcRun, hyperlane.RouteGatewayWebSearch, hyperlane.RouteGatewayNetFetch, hyperlane.RouteGatewayDesktopOpen:
		return "high"
	case hyperlane.RouteGatewayFSWrite, hyperlane.RouteGatewayFSMkdir:
		return "medium"
	case hyperlane.RouteGatewayFSRead, hyperlane.RouteGatewayFSList, hyperlane.RouteGatewayGitStatus:
		return "low"
	default:
		return "none"
	}
}

func rejectedReasonForUnknown(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if lower == "" {
		return "empty input"
	}
	if strings.Contains(lower, "../") || strings.Contains(lower, "/..") {
		return "unsafe path traversal rejected"
	}
	if strings.ContainsAny(raw, "\x00\r\n;&|<>`$") && (wantsFilesystemMkdir(raw) || wantsReadFile(raw) || wantsListDirectory(raw) || wantsWriteFile(raw)) {
		return "unsafe path metacharacter rejected"
	}
	if strings.Contains(lower, " /") && (wantsFilesystemMkdir(raw) || wantsReadFile(raw) || wantsListDirectory(raw) || wantsWriteFile(raw)) {
		return "absolute path rejected"
	}
	return "no deterministic intent matched"
}
