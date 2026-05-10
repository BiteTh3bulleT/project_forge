package api

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRemoteTokenOkRejectsQueryToken(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest("POST", "/api/remote/telegram?token=remote-secret", nil)

	if srv.remoteTokenOk(req, "remote-secret") {
		t.Fatalf("remote token in URL query must not authenticate")
	}
}

func TestRemoteTokenOkAcceptsHeaderTokens(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{name: "forge remote token", header: "X-Forge-Remote-Token"},
		{name: "telegram secret token", header: "X-Telegram-Bot-Api-Secret-Token"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := &Server{}
			req := httptest.NewRequest("POST", "/api/remote/telegram?token=wrong-token", nil)
			req.Header.Set(tc.header, "remote-secret")

			if !srv.remoteTokenOk(req, "remote-secret") {
				t.Fatalf("%s must authenticate", tc.header)
			}
		})
	}
}

func TestRemoteTokenMatchesTrimsAndRejectsEmptyValues(t *testing.T) {
	if !remoteTokenMatches(" remote-secret ", "remote-secret") {
		t.Fatalf("expected trimmed matching remote token to authenticate")
	}
	for _, tc := range []struct {
		name     string
		provided string
		expected string
	}{
		{name: "empty provided", provided: "", expected: "remote-secret"},
		{name: "empty expected", provided: "remote-secret", expected: ""},
		{name: "mismatch", provided: "remote-secreu", expected: "remote-secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if remoteTokenMatches(tc.provided, tc.expected) {
				t.Fatalf("expected token match to reject %s", tc.name)
			}
		})
	}
}

func TestRemoteTokenOkRejectsMismatchedSameLengthToken(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest("POST", "/api/remote/telegram", nil)
	req.Header.Set("X-Forge-Remote-Token", "remote-secreu")

	if srv.remoteTokenOk(req, "remote-secret") {
		t.Fatalf("same-length mismatched remote token must not authenticate")
	}
}

func TestRemoteBoundedTextRejectsOversizeContent(t *testing.T) {
	_, err := remoteBoundedText(strings.Repeat("a", remoteInboundTextLimit+1))
	if err == nil {
		t.Fatalf("expected oversized remote text to be rejected")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too-large error, got %v", err)
	}
}

func TestRemoteBoundedTextTrimsAndAcceptsLimitSizedContent(t *testing.T) {
	raw := " \n" + strings.Repeat("a", remoteInboundTextLimit) + "\t "
	got, err := remoteBoundedText(raw)
	if err != nil {
		t.Fatalf("expected limit-sized remote text to be accepted: %v", err)
	}
	if len(got) != remoteInboundTextLimit {
		t.Fatalf("expected trimmed limit-sized text length %d, got %d", remoteInboundTextLimit, len(got))
	}
}

func TestRemoteDiscordRejectsOversizeTextBeforeProcessing(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	seedRemoteIngressSettings(t, st.DB)

	body := `{"content":"` + strings.Repeat("a", remoteInboundTextLimit+1) + `","channel_id":"channel-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/remote/discord", strings.NewReader(body))
	req.Header.Set("X-Forge-Remote-Token", "remote-secret")
	rr := httptest.NewRecorder()

	srv.handleRemoteDiscord(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected oversize remote text to return 413, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

func TestRemoteDiscordRejectsOversizeBodyAtDecoder(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	seedRemoteIngressSettings(t, st.DB)

	body := `{"content":"` + strings.Repeat("a", remoteInboundBodyLimit) + `","channel_id":"channel-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/remote/discord", strings.NewReader(body))
	req.Header.Set("X-Forge-Remote-Token", "remote-secret")
	rr := httptest.NewRecorder()

	srv.handleRemoteDiscord(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected oversize remote body to return 413, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

func seedRemoteIngressSettings(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	for key, value := range map[string]string{
		remoteAccessEnabledKey: "true",
		remoteAccessTokenKey:   "remote-secret",
		discordWebhookURLKey:   "https://discord.example/webhook",
	} {
		if err := upsertSetting(ctx, db, key, value); err != nil {
			t.Fatalf("seed setting %s: %v", key, err)
		}
	}
}
