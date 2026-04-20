package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleDiscordGatewayStatusDisabled(t *testing.T) {
	t.Parallel()

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/discord/status", nil)
	rr := httptest.NewRecorder()
	s.handleDiscordGatewayStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["enabled"] != false {
		t.Fatalf("expected disabled payload, got %+v", payload)
	}
}

func TestHandleDiscordGatewayStatusEnabled(t *testing.T) {
	t.Parallel()

	s := &Server{
		discordGateway: &DiscordGateway{
			status: discordGatewayStatus{
				Enabled:        true,
				Connected:      true,
				CommandPrefix:  "!forge",
				EnableSlash:    true,
				EnableText:     true,
				EnablePassive:  false,
				EnableOutbound: true,
				InboundCount:   2,
				OutboundCount:  1,
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/discord/status", nil)
	rr := httptest.NewRecorder()
	s.handleDiscordGatewayStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["enabled"] != true {
		t.Fatalf("expected enabled payload, got %+v", payload)
	}
	statusRaw, ok := payload["status"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested status payload: %+v", payload)
	}
	if statusRaw["connected"] != true {
		t.Fatalf("expected connected status payload: %+v", statusRaw)
	}
}
