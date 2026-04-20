package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"forge/projectforge/services/core/internal/store"
)

func TestHandleTelegramStatusWithoutToken(t *testing.T) {
	t.Parallel()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	s := &Server{st: st}
	req := httptest.NewRequest(http.MethodGet, "/api/telegram/status", nil)
	rr := httptest.NewRecorder()
	s.handleTelegramStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["ready"] != false {
		t.Fatalf("expected ready=false, payload=%+v", payload)
	}
}
