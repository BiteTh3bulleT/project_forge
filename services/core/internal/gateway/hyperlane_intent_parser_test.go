package gateway

import (
	"os"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/hyperlane"
)

func TestParseHyperlaneIntentGatewayFilesystem(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		user       string
		wantType   hyperlane.IntentType
		wantRoute  string
		wantArgKey string
		wantArg    string
	}{
		{name: "mkdir", user: "mkdir scratch/hyperlane", wantType: hyperlane.IntentMkdir, wantRoute: hyperlane.RouteGatewayFSMkdir, wantArgKey: "path", wantArg: "scratch/hyperlane"},
		{name: "write file", user: `Create a directory called scratch/hyperlane and a file labeled "note.txt" and the words "hi"`, wantType: hyperlane.IntentWriteFile, wantRoute: hyperlane.RouteGatewayFSWrite, wantArgKey: "path", wantArg: "scratch/hyperlane/note.txt"},
		{name: "combined mkdir write", user: `create a file labeled "test.txt" inside scratch/not_another_test/ and inside said file the words "This is a test file"`, wantType: hyperlane.IntentWriteFile, wantRoute: hyperlane.RouteGatewayFSWrite, wantArgKey: "path", wantArg: "scratch/not_another_test/test.txt"},
		{name: "read file", user: `read file "README.md"`, wantType: hyperlane.IntentReadFile, wantRoute: hyperlane.RouteGatewayFSRead, wantArgKey: "path", wantArg: "README.md"},
		{name: "list directory", user: "list files in docs", wantType: hyperlane.IntentListDirectory, wantRoute: hyperlane.RouteGatewayFSList, wantArgKey: "path", wantArg: "docs"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ParseHyperlaneIntent(tc.user)
			assertHyperlaneTrace(t, got)
			if got.Type != tc.wantType || got.Route != tc.wantRoute {
				t.Fatalf("intent type/route = %q/%q, want %q/%q", got.Type, got.Route, tc.wantType, tc.wantRoute)
			}
			if !got.RequiresGateway || got.RequiresModel {
				t.Fatalf("gateway filesystem intents should require gateway and not model: %+v", got)
			}
			if got.Arguments[tc.wantArgKey] != tc.wantArg {
				t.Fatalf("arg %q = %#v, want %q", tc.wantArgKey, got.Arguments[tc.wantArgKey], tc.wantArg)
			}
		})
	}
}

func TestParseHyperlaneIntentRunCommandApproval(t *testing.T) {
	t.Parallel()
	got := ParseHyperlaneIntent("run go test ./...")
	assertHyperlaneTrace(t, got)
	if got.Type != hyperlane.IntentRunCommand || got.Route != hyperlane.RouteGatewayProcRun {
		t.Fatalf("intent type/route = %q/%q", got.Type, got.Route)
	}
	if !got.RequiresGateway || got.RequiresModel || !got.RequiresApprovalHint || got.RiskClass != "high" {
		t.Fatalf("run command should be gateway-only high-risk approval hint: %+v", got)
	}
	if got.Arguments["command"] != "go test ./..." {
		t.Fatalf("command arg = %#v", got.Arguments["command"])
	}
	if len(got.Warnings) == 0 {
		t.Fatalf("expected warning for command route")
	}
}

func TestParseHyperlaneIntentNoModelQueries(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		user      string
		wantType  hyperlane.IntentType
		wantRoute string
	}{
		{name: "status", user: "what is the status", wantType: hyperlane.IntentStatusQuery, wantRoute: hyperlane.RouteStructuredStatus},
		{name: "chat memory", user: "do we have cross chat context and memory?", wantType: hyperlane.IntentChatMemoryInspection, wantRoute: hyperlane.RouteChatMemoryInspector},
		{name: "chat history lookup", user: "On 4/16/2026 at 3:54:49 PM, what did I ask you to do?", wantType: hyperlane.IntentChatHistoryLookup, wantRoute: hyperlane.RouteChatHistoryLookup},
		{name: "diagnostics", user: "show diagnostics and what is degraded", wantType: hyperlane.IntentDiagnosticsQuery, wantRoute: hyperlane.RouteStructuredDiagnostics},
		{name: "restore inspection", user: "show recent restore decisions", wantType: hyperlane.IntentRestoreInspection, wantRoute: hyperlane.RouteRestoreInspector},
		{name: "dream report", user: "show latest Dream report", wantType: hyperlane.IntentDreamReportInspection, wantRoute: hyperlane.RouteDreamReportInspector},
		{name: "modelruntime status", user: "modelruntime status and registry availability", wantType: hyperlane.IntentModelruntimeStatus, wantRoute: hyperlane.RouteModelruntimeStatus},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ParseHyperlaneIntent(tc.user)
			assertHyperlaneTrace(t, got)
			if got.Type != tc.wantType || got.Route != tc.wantRoute {
				t.Fatalf("intent type/route = %q/%q, want %q/%q", got.Type, got.Route, tc.wantType, tc.wantRoute)
			}
			if got.RequiresGateway || got.RequiresModel || got.RequiresApprovalHint {
				t.Fatalf("structured no-model query should not require gateway/model/approval: %+v", got)
			}
		})
	}
}

func TestParseHyperlaneIntentUnknownAndUnsafe(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		user         string
		wantRejected string
	}{
		{name: "unrelated", user: "what is the best sandwich", wantRejected: "no deterministic intent matched"},
		{name: "traversal", user: "mkdir ../escape", wantRejected: "unsafe path traversal rejected"},
		{name: "absolute", user: "mkdir /tmp/escape", wantRejected: "absolute path rejected"},
		{name: "metacharacter", user: "cat docs/README.md;rm -rf .", wantRejected: "unsafe path metacharacter rejected"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ParseHyperlaneIntent(tc.user)
			if got.Type != hyperlane.IntentUnknown {
				t.Fatalf("type = %q, want unknown", got.Type)
			}
			if got.Trace.RejectedReason != tc.wantRejected {
				t.Fatalf("rejected reason = %q, want %q", got.Trace.RejectedReason, tc.wantRejected)
			}
		})
	}
}

func TestParseHyperlaneIntentAllowsExplicitGatewayDirectoryHint(t *testing.T) {
	t.Parallel()
	got := ParseHyperlaneIntentWithDirectoryHint(
		`In the same directory, create a test webpage. I would like it to look like it belongs to a video game journal site.`,
		`/home/rshort/Downloads/PeanutButterJellyTime/flower.svg`,
	)
	assertHyperlaneTrace(t, got)
	if got.Type != hyperlane.IntentGenerateTemplate || got.Route != hyperlane.RouteGatewayFSWrite {
		t.Fatalf("type/route = %q/%q", got.Type, got.Route)
	}
	if got.Arguments["path"] != "/home/rshort/Downloads/PeanutButterJellyTime/test-webpage.html" {
		t.Fatalf("path = %#v", got.Arguments["path"])
	}
}

func TestParseHyperlaneIntentTemplateDeterministic(t *testing.T) {
	t.Parallel()
	user := `Create a folder in the Downloads directory labled Python_Scripts/. Inside the folder create a python script that will make anything I download get sorted into a folder in the downloads folder.`
	first := ParseHyperlaneIntent(user)
	second := ParseHyperlaneIntent(user)
	if first.Type != hyperlane.IntentGenerateTemplate || second.Type != hyperlane.IntentGenerateTemplate {
		t.Fatalf("expected generated template intents: %q/%q", first.Type, second.Type)
	}
	if first.Arguments["path"] != "~/Downloads/Python_Scripts/sort_downloads.py" {
		t.Fatalf("path = %#v", first.Arguments["path"])
	}
	if first.Arguments["contents"] == "" || first.Arguments["contents"] != second.Arguments["contents"] {
		t.Fatalf("template content should be deterministic")
	}
}

func TestParseHyperlaneIntentDoesNotExecuteOrMutate(t *testing.T) {
	t.Parallel()
	marker := "hyperlane_noexec_marker_unit_test"
	_ = os.Remove(marker)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("test marker already exists or cannot stat: %v", err)
	}
	got := ParseHyperlaneIntent("mkdir " + marker)
	if got.Type != hyperlane.IntentMkdir {
		t.Fatalf("type = %q", got.Type)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("parser created or mutated filesystem marker: %v", err)
	}
}

func assertHyperlaneTrace(t *testing.T, got hyperlane.Intent) {
	t.Helper()
	if got.ID == "" {
		t.Fatalf("missing intent id")
	}
	if got.Trace.ParserVersion != hyperlane.ParserVersion {
		t.Fatalf("parser version = %q", got.Trace.ParserVersion)
	}
	if got.Trace.MatchedRule == "" || got.MatchedRule == "" {
		t.Fatalf("missing matched rule: %+v", got)
	}
	if got.Trace.Route != got.Route {
		t.Fatalf("trace route = %q, route = %q", got.Trace.Route, got.Route)
	}
	if got.Trace.Confidence != got.Confidence {
		t.Fatalf("trace confidence = %v, confidence = %v", got.Trace.Confidence, got.Confidence)
	}
	if strings.TrimSpace(string(got.Type)) == "" {
		t.Fatalf("missing type")
	}
}
