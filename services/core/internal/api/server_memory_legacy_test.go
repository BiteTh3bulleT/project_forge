package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func TestLegacyMemoryMutationEndpointsAreBlockedByDefaultAndAudited(t *testing.T) {
	prev, hadPrev := os.LookupEnv("FORGE_ALLOW_LEGACY_MEMORY_MUTATIONS")
	_ = os.Unsetenv("FORGE_ALLOW_LEGACY_MEMORY_MUTATIONS")
	t.Cleanup(func() {
		if !hadPrev {
			_ = os.Unsetenv("FORGE_ALLOW_LEGACY_MEMORY_MUTATIONS")
			return
		}
		_ = os.Setenv("FORGE_ALLOW_LEGACY_MEMORY_MUTATIONS", prev)
	})

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
	}{
		{
			name:          "create observation",
			method:        http.MethodPost,
			path:          "/api/memory/observations",
			expectAction:  "legacy.memory.observation.create.blocked",
			expectSubject: "new",
		},
		{
			name:          "patch observation",
			method:        http.MethodPatch,
			path:          "/api/memory/observations/42",
			expectAction:  "legacy.memory.observation.patch.blocked",
			expectSubject: "42",
		},
		{
			name:          "mark observation usefulness",
			method:        http.MethodPost,
			path:          "/api/memory/observations/42/usefulness",
			expectAction:  "legacy.memory.observation.usefulness.blocked",
			expectSubject: "42",
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
			req := httptest.NewRequest(tc.method, url, bytes.NewReader([]byte(`{}`)))
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Fatalf("expected forbidden, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
			}
			if got := rr.Body.String(); !strings.Contains(got, "use authoritative syscall path via /api/gateway/invoke") {
				t.Fatalf("expected syscall guidance message, got %q", got)
			}

			var blockedCount int
			if err := st.DB.QueryRowContext(context.Background(), `
SELECT COUNT(1) FROM audit_records
WHERE category = 'memory' AND action = ? AND subject_id = ? AND outcome = 'denied'`, tc.expectAction, tc.expectSubject,
			).Scan(&blockedCount); err != nil {
				t.Fatalf("query blocked legacy memory mutation audit: %v", err)
			}
			if blockedCount == 0 {
				t.Fatalf("expected blocked legacy memory mutation audit record for action=%s subject=%s", tc.expectAction, tc.expectSubject)
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
		})
	}
}
