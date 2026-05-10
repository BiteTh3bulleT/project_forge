package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestTelegramAPIRequestRejectsOversizeResponse(t *testing.T) {
	previousClient := http.DefaultClient
	t.Cleanup(func() {
		http.DefaultClient = previousClient
	})
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"ok":true,"result":{"id":1,"is_bot":true,"username":"forge_bot"}}` + strings.Repeat(" ", telegramAPIResponseBodyLimit+1)
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}

	var out telegramAPIEnvelope[telegramMeResult]
	err := telegramAPIRequest(context.Background(), "test-token", "getMe", &out)
	if err == nil {
		t.Fatalf("expected oversize Telegram API response to be rejected")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too-large error, got %v", err)
	}
}
