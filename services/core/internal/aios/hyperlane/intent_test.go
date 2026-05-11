package hyperlane

import "testing"

func TestUnknownIntentCopiesWarningsAndRequiresModelFallback(t *testing.T) {
	t.Parallel()

	warnings := []string{"unsafe path rejected", "operator review required"}
	got := UnknownIntent("intent-1", "no deterministic intent matched", warnings)
	warnings[0] = "mutated input"

	if got.ID != "intent-1" {
		t.Fatalf("ID = %q, want intent-1", got.ID)
	}
	if got.Type != IntentUnknown || got.Route != RouteUnknown || got.Lane != "operator" {
		t.Fatalf("unknown intent route/type/lane = %q/%q/%q", got.Type, got.Route, got.Lane)
	}
	if got.RequiresGateway {
		t.Fatalf("unknown fallback must not require gateway execution: %+v", got)
	}
	if !got.RequiresModel {
		t.Fatalf("unknown fallback must require model handling: %+v", got)
	}
	if got.RequiresApprovalHint {
		t.Fatalf("unknown fallback must not imply approval: %+v", got)
	}
	if got.RiskClass != "none" {
		t.Fatalf("RiskClass = %q, want none", got.RiskClass)
	}
	if got.MatchedRule != "unknown" || got.Trace.MatchedRule != "unknown" {
		t.Fatalf("matched rules = %q/%q, want unknown", got.MatchedRule, got.Trace.MatchedRule)
	}
	if got.Trace.ParserVersion != ParserVersion {
		t.Fatalf("trace parser version = %q, want %q", got.Trace.ParserVersion, ParserVersion)
	}
	if got.Trace.RejectedReason != "no deterministic intent matched" {
		t.Fatalf("rejected reason = %q", got.Trace.RejectedReason)
	}
	if got.Warnings[0] != "unsafe path rejected" || got.Trace.Warnings[0] != "unsafe path rejected" {
		t.Fatalf("warnings should be copied from input: %+v / %+v", got.Warnings, got.Trace.Warnings)
	}

	got.Warnings[0] = "mutated top-level"
	if got.Trace.Warnings[0] != "unsafe path rejected" {
		t.Fatalf("trace warnings should not alias top-level warnings: %+v", got.Trace.Warnings)
	}
}

func TestSupportsNoModelRouteAllowsOnlyStructuredNoAuthorityRoutes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		route string
		want  bool
	}{
		{name: "status", route: RouteStructuredStatus, want: true},
		{name: "diagnostics", route: RouteStructuredDiagnostics, want: true},
		{name: "chat memory", route: RouteChatMemoryInspector, want: true},
		{name: "chat history", route: RouteChatHistoryLookup, want: true},
		{name: "restore inspector", route: RouteRestoreInspector, want: true},
		{name: "dream reports", route: RouteDreamReportInspector, want: true},
		{name: "modelruntime status", route: RouteModelruntimeStatus, want: true},
		{name: "filesystem mkdir", route: RouteGatewayFSMkdir, want: false},
		{name: "filesystem write", route: RouteGatewayFSWrite, want: false},
		{name: "filesystem read", route: RouteGatewayFSRead, want: false},
		{name: "directory list", route: RouteGatewayFSList, want: false},
		{name: "process run", route: RouteGatewayProcRun, want: false},
		{name: "web search", route: RouteGatewayWebSearch, want: false},
		{name: "network fetch", route: RouteGatewayNetFetch, want: false},
		{name: "desktop open", route: RouteGatewayDesktopOpen, want: false},
		{name: "git status", route: RouteGatewayGitStatus, want: false},
		{name: "model runtime", route: RouteModelRuntime, want: false},
		{name: "unknown", route: RouteUnknown, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SupportsNoModelRoute(Intent{Route: tc.route})
			if got != tc.want {
				t.Fatalf("SupportsNoModelRoute(%q) = %v, want %v", tc.route, got, tc.want)
			}
		})
	}
}

func TestSupportsNoModelRouteRejectsGatewayOrModelRequiredIntent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		intent Intent
	}{
		{
			name: "model required",
			intent: Intent{
				Route:         RouteStructuredStatus,
				RequiresModel: true,
			},
		},
		{
			name: "gateway required",
			intent: Intent{
				Route:           RouteStructuredStatus,
				RequiresGateway: true,
			},
		},
		{
			name: "model and gateway required",
			intent: Intent{
				Route:           RouteStructuredStatus,
				RequiresGateway: true,
				RequiresModel:   true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if SupportsNoModelRoute(tc.intent) {
				t.Fatalf("SupportsNoModelRoute(%+v) = true, want false", tc.intent)
			}
		})
	}
}

func TestIntentAndRouteCompatibilityAliases(t *testing.T) {
	t.Parallel()

	if IntentModelRuntimeStatus != IntentModelruntimeStatus {
		t.Fatalf("IntentModelRuntimeStatus = %q, want %q", IntentModelRuntimeStatus, IntentModelruntimeStatus)
	}
	if RouteModelRuntimeStatus != RouteModelruntimeStatus {
		t.Fatalf("RouteModelRuntimeStatus = %q, want %q", RouteModelRuntimeStatus, RouteModelruntimeStatus)
	}
	if RouteStatusQuery != RouteStructuredStatus {
		t.Fatalf("RouteStatusQuery = %q, want %q", RouteStatusQuery, RouteStructuredStatus)
	}
	if RouteDiagnosticsQuery != RouteStructuredDiagnostics {
		t.Fatalf("RouteDiagnosticsQuery = %q, want %q", RouteDiagnosticsQuery, RouteStructuredDiagnostics)
	}
	if RouteChatMemoryInspection != RouteChatMemoryInspector {
		t.Fatalf("RouteChatMemoryInspection = %q, want %q", RouteChatMemoryInspection, RouteChatMemoryInspector)
	}
	if RouteChatHistoryLookupHint != RouteChatHistoryLookup {
		t.Fatalf("RouteChatHistoryLookupHint = %q, want %q", RouteChatHistoryLookupHint, RouteChatHistoryLookup)
	}
	if RouteRestoreInspection != RouteRestoreInspector {
		t.Fatalf("RouteRestoreInspection = %q, want %q", RouteRestoreInspection, RouteRestoreInspector)
	}
	if RouteDreamReportInspection != RouteDreamReportInspector {
		t.Fatalf("RouteDreamReportInspection = %q, want %q", RouteDreamReportInspection, RouteDreamReportInspector)
	}
}
