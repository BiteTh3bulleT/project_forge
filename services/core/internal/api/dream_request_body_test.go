package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func TestDreamRunRejectsOversizeRequestBody(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	srv := NewServer(st, config.Config{DataDir: dataDir, WorkspaceDir: t.TempDir()})
	t.Cleanup(func() { srv.ShutdownWatch() })

	body := `{"workspaceId":"ws-dream","metadata":{"oversize":"` + strings.Repeat("a", dreamRunRequestBodyLimit+1) + `"}}`
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/dream/run", strings.NewReader(body)))

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusRequestEntityTooLarge, rr.Body.String())
	}
	if !strings.Contains(strings.ToLower(rr.Body.String()), "too large") {
		t.Fatalf("expected too-large response, got %q", rr.Body.String())
	}
}
