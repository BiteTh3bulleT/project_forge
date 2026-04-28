package hyperlane

import (
	"regexp"
	"strings"
)

const (
	IntentUnknown               = "unknown"
	IntentStatusQuery           = "status_query"
	IntentDiagnosticsQuery      = "diagnostics_query"
	IntentRestoreInspection     = "restore_inspection"
	IntentDreamReportInspection = "dream_report_inspection"
	IntentModelRuntimeStatus    = "modelruntime_status"

	RouteStatusQuery           = "forge.status"
	RouteDiagnosticsQuery      = "forge.diagnostics"
	RouteRestoreInspection     = "context.restore.inspect"
	RouteDreamReportInspection = "dream.report.inspect"
	RouteModelRuntimeStatus    = "modelruntime.status"
)

type Intent struct {
	Type          string  `json:"type"`
	Route         string  `json:"route"`
	Confidence    float64 `json:"confidence"`
	MatchedRule   string  `json:"matchedRule"`
	RequiresModel bool    `json:"requiresModel"`
	Ambiguous     bool    `json:"ambiguous,omitempty"`
}

type rule struct {
	intentType string
	route      string
	name       string
	confidence float64
	patterns   []*regexp.Regexp
}

var rules = []rule{
	{
		intentType: IntentModelRuntimeStatus,
		route:      RouteModelRuntimeStatus,
		name:       "modelruntime_status_terms",
		confidence: 0.94,
		patterns: regexps(
			`\bmodel\s*runtime\b.*\b(status|health|queue|loaded|available|degraded|cooldown)\b`,
			`\b(status|health|queue|loaded|available|degraded|cooldown)\b.*\bmodel\s*runtime\b`,
			`\bmodelruntime\b.*\b(status|health|queue|loaded|available|degraded|cooldown)\b`,
			`\bloaded\s+models?\b`,
			`\bruntime\s+queue\b`,
		),
	},
	{
		intentType: IntentDreamReportInspection,
		route:      RouteDreamReportInspection,
		name:       "dream_report_terms",
		confidence: 0.93,
		patterns: regexps(
			`\bdream\s+reports?\b`,
			`\blatest\s+dream\b`,
			`\bdream\s+mode\b.*\b(report|summary|status)\b`,
		),
	},
	{
		intentType: IntentRestoreInspection,
		route:      RouteRestoreInspection,
		name:       "restore_inspection_terms",
		confidence: 0.93,
		patterns: regexps(
			`\bcontext\s+restore\b`,
			`\brestore\s+(inspection|summary|score|scores|snapshot|metadata|hints?)\b`,
			`\blatest\s+restore\b`,
		),
	},
	{
		intentType: IntentDiagnosticsQuery,
		route:      RouteDiagnosticsQuery,
		name:       "diagnostics_terms",
		confidence: 0.91,
		patterns: regexps(
			`\bdiagnostics?\b`,
			`\bdiagnostic\s+(summary|status|report)\b`,
			`\bhealth\s+diagnostics?\b`,
		),
	},
	{
		intentType: IntentStatusQuery,
		route:      RouteStatusQuery,
		name:       "operator_status_terms",
		confidence: 0.89,
		patterns: regexps(
			`\b(system|core|forge|runtime)\s+(status|health)\b`,
			`\b(status|health)\s+(of\s+)?(system|core|forge|runtime)\b`,
			`\bsafe\s*mode\s+(status|health)?\b`,
			`\bhow\s+(are\s+we|is\s+forge|is\s+the\s+core)\b`,
			`\bwhere\s+are\s+we\b`,
		),
	},
}

// ParseIntent classifies low-risk operator inspection requests for Hyperlane.
// Unknown or multi-route text intentionally requires the normal model path.
func ParseIntent(user string) Intent {
	text := normalize(user)
	if text == "" {
		return unknown()
	}
	matches := []Intent{}
	seen := map[string]struct{}{}
	for _, candidate := range rules {
		for _, pattern := range candidate.patterns {
			if !pattern.MatchString(text) {
				continue
			}
			if _, ok := seen[candidate.route]; ok {
				break
			}
			seen[candidate.route] = struct{}{}
			matches = append(matches, Intent{
				Type:          candidate.intentType,
				Route:         candidate.route,
				Confidence:    candidate.confidence,
				MatchedRule:   candidate.name,
				RequiresModel: false,
			})
			break
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	if len(matches) > 1 {
		out := unknown()
		out.Ambiguous = true
		out.MatchedRule = "multiple_structured_routes"
		return out
	}
	return unknown()
}

func SupportsNoModelRoute(intent Intent) bool {
	if intent.RequiresModel {
		return false
	}
	switch intent.Route {
	case RouteStatusQuery, RouteDiagnosticsQuery, RouteRestoreInspection, RouteDreamReportInspection, RouteModelRuntimeStatus:
		return true
	default:
		return false
	}
}

func unknown() Intent {
	return Intent{Type: IntentUnknown, RequiresModel: true}
}

func regexps(patterns ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		out = append(out, regexp.MustCompile(`(?i)`+pattern))
	}
	return out
}

func normalize(user string) string {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"\n", " ",
		"\t", " ",
		"?", " ",
		"!", " ",
		".", " ",
		",", " ",
		";", " ",
		":", " ",
	)
	s = replacer.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}
