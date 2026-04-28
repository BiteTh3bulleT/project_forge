package gateway

import "testing"

func TestParseHyperlaneIntentStructuredRoutes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		in         string
		intentType string
		route      string
	}{
		{name: "status", in: "what is forge core status?", intentType: "status_query", route: "forge.status"},
		{name: "diagnostics", in: "show diagnostics summary", intentType: "diagnostics_query", route: "forge.diagnostics"},
		{name: "restore", in: "latest restore score", intentType: "restore_inspection", route: "context.restore.inspect"},
		{name: "dream", in: "latest Dream report", intentType: "dream_report_inspection", route: "dream.report.inspect"},
		{name: "modelruntime", in: "model runtime queue status", intentType: "modelruntime_status", route: "modelruntime.status"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ParseHyperlaneIntent(tc.in)
			if got.RequiresModel {
				t.Fatalf("expected no-model intent, got %+v", got)
			}
			if got.Type != tc.intentType || got.Route != tc.route {
				t.Fatalf("intent=%+v want type=%q route=%q", got, tc.intentType, tc.route)
			}
			if !SupportsNoModelHyperlaneRoute(got) {
				t.Fatalf("expected supported no-model route: %+v", got)
			}
		})
	}
}

func TestParseHyperlaneIntentUnknownAndAmbiguousFallThrough(t *testing.T) {
	t.Parallel()
	for _, input := range []string{
		"explain the architecture",
		"forge status and latest Dream report",
	} {
		got := ParseHyperlaneIntent(input)
		if !got.RequiresModel {
			t.Fatalf("expected model-required fallthrough for %q, got %+v", input, got)
		}
	}
}
