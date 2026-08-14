package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func TestLegacyMemoryMutationEndpointsAreRetiredAndAudited(t *testing.T) {
	dataDir := t.TempDir()
	wsDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{
		DataDir:      dataDir,
		WorkspaceDir: wsDir,
	})
	t.Cleanup(func() { srv.ShutdownWatch() })

	h := srv.Handler()
	tests := []struct {
		name          string
		method        string
		path          string
		expectAction  string
		expectSubject string
		body          string
	}{
		{
			name:          "create observation",
			method:        http.MethodPost,
			path:          "/api/memory/observations",
			expectAction:  "legacy.memory.observation.create.retired",
			expectSubject: "new",
			body:          `{not-json`,
		},
		{
			name:          "patch observation",
			method:        http.MethodPatch,
			path:          "/api/memory/observations/42",
			expectAction:  "legacy.memory.observation.patch.retired",
			expectSubject: "42",
			body:          strings.Repeat("x", memoryMutationRequestBodyLimit+1),
		},
		{
			name:          "mark observation usefulness",
			method:        http.MethodPost,
			path:          "/api/memory/observations/42/usefulness",
			expectAction:  "legacy.memory.observation.usefulness.retired",
			expectSubject: "42",
			body:          `{`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			correlation := "corr-" + strings.ReplaceAll(tc.expectAction, ".", "-")
			trace := "trace-" + strings.ReplaceAll(tc.expectAction, ".", "-")
			workspace := "workspace-legacy-memory"
			separator := "?"
			if strings.Contains(tc.path, "?") {
				separator = "&"
			}
			url := tc.path + separator + "correlationId=" + correlation + "&traceId=" + trace + "&workspaceId=" + workspace
			req := httptest.NewRequest(tc.method, url, bytes.NewReader([]byte(tc.body)))
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusGone {
				t.Fatalf("expected gone, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
			}
			var body map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode retired memory response: %v body=%s", err, rr.Body.String())
			}
			if body["historyPreserved"] != true {
				t.Fatalf("expected historyPreserved response, got %#v", body)
			}
			if got := rr.Body.String(); !strings.Contains(got, "MATERIALIZE_ADMITTED_EVIDENCE") || !strings.Contains(got, "REVISE_MEMORY_EVIDENCE") {
				t.Fatalf("expected production FORGE-K migration guidance message, got %q", got)
			}

			var observationCount int
			if err := st.DB.QueryRowContext(context.Background(), `SELECT COUNT(1) FROM memory_observations`).Scan(&observationCount); err != nil {
				t.Fatalf("query memory observation count: %v", err)
			}
			if observationCount != 0 {
				t.Fatalf("retired endpoint must not write memory_observations, got count=%d", observationCount)
			}

			var retiredCount int
			if err := st.DB.QueryRowContext(context.Background(), `
SELECT COUNT(1) FROM audit_records
WHERE category = 'memory' AND action = ? AND subject_id = ? AND outcome = 'denied'`, tc.expectAction, tc.expectSubject,
			).Scan(&retiredCount); err != nil {
				t.Fatalf("query retired legacy memory mutation audit: %v", err)
			}
			if retiredCount == 0 {
				t.Fatalf("expected retired legacy memory mutation audit record for action=%s subject=%s", tc.expectAction, tc.expectSubject)
			}

			auditCorrelation, outcome, payload := mustAuditRecordByActionAndCorrelation(t, st, tc.expectAction, correlation)
			if auditCorrelation != correlation {
				t.Fatalf("audit correlation = %q want %q", auditCorrelation, correlation)
			}
			if outcome != "denied" {
				t.Fatalf("audit outcome = %q want denied", outcome)
			}
			if !strings.Contains(payload, `"traceId":"`+trace+`"`) {
				t.Fatalf("expected traceId in legacy memory audit payload, got %s", payload)
			}
			if !strings.Contains(payload, `"workspaceId":"`+workspace+`"`) {
				t.Fatalf("expected workspaceId in legacy memory audit payload, got %s", payload)
			}
			if !strings.Contains(payload, `"historyPreserved":true`) || !strings.Contains(payload, `"MATERIALIZE_ADMITTED_EVIDENCE"`) || !strings.Contains(payload, `"REVISE_MEMORY_EVIDENCE"`) {
				t.Fatalf("expected migration review path in legacy memory audit payload, got %s", payload)
			}
		})
	}
}
