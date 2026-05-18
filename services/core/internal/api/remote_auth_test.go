package api

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
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

func TestRemoteDiscordRejectsMissingSignatureWhenPublicKeyConfigured(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	seedRemoteIngressSettings(t, st.DB)
	publicKey, _, _ := ed25519.GenerateKey(nil)
	seedRemoteDiscordPublicKey(t, st.DB, publicKey)

	req := httptest.NewRequest(http.MethodPost, "/api/remote/discord", strings.NewReader(`{"content":"","channel_id":"channel-1"}`))
	req.Header.Set("X-Forge-Remote-Token", "remote-secret")
	rr := httptest.NewRecorder()

	srv.handleRemoteDiscord(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing Discord signature to return 401, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

func TestRemoteDiscordRejectsInvalidSignatureWhenPublicKeyConfigured(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	seedRemoteIngressSettings(t, st.DB)
	publicKey, _, _ := ed25519.GenerateKey(nil)
	seedRemoteDiscordPublicKey(t, st.DB, publicKey)

	body := `{"content":"","channel_id":"channel-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/remote/discord", strings.NewReader(body))
	req.Header.Set("X-Forge-Remote-Token", "remote-secret")
	req.Header.Set("X-Signature-Timestamp", "1700000000")
	req.Header.Set("X-Signature-Ed25519", strings.Repeat("00", ed25519.SignatureSize))
	rr := httptest.NewRecorder()

	srv.handleRemoteDiscord(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid Discord signature to return 401, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

func TestRemoteDiscordAcceptsValidSignatureWhenPublicKeyConfigured(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	seedRemoteIngressSettings(t, st.DB)
	publicKey, privateKey, _ := ed25519.GenerateKey(nil)
	seedRemoteDiscordPublicKey(t, st.DB, publicKey)

	body := `{"content":"","channel_id":"channel-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/remote/discord", strings.NewReader(body))
	req.Header.Set("X-Forge-Remote-Token", "remote-secret")
	setDiscordSignatureHeaders(req, privateKey, "1700000000", []byte(body))
	rr := httptest.NewRecorder()

	srv.handleRemoteDiscord(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected valid Discord signature to continue to payload validation, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if !strings.Contains(rr.Body.String(), "content required") {
		t.Fatalf("expected payload validation response after valid signature, got %s", rr.Body.String())
	}
}

func TestRemoteDiscordKeepsTokenOnlyBehaviorWhenPublicKeyUnconfigured(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	seedRemoteIngressSettings(t, st.DB)

	req := httptest.NewRequest(http.MethodPost, "/api/remote/discord", strings.NewReader(`{"content":"","channel_id":"channel-1"}`))
	req.Header.Set("X-Forge-Remote-Token", "remote-secret")
	rr := httptest.NewRecorder()

	srv.handleRemoteDiscord(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected unconfigured Discord public key to preserve token-only validation, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

func TestRemoteTokenMatchDoesNotEarlyReturnOnLengthMismatch(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	body, err := os.ReadFile(strings.TrimSuffix(file, "_auth_test.go") + ".go")
	if err != nil {
		t.Fatalf("read remote source: %v", err)
	}
	if strings.Contains(string(body), "len(provided) != len(expected)") {
		t.Fatal("remoteTokenMatches must not early-return on token length mismatch")
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

func seedRemoteDiscordPublicKey(t *testing.T, db *sql.DB, publicKey ed25519.PublicKey) {
	t.Helper()
	if err := upsertSetting(context.Background(), db, "discord_public_key", hex.EncodeToString(publicKey)); err != nil {
		t.Fatalf("seed discord public key: %v", err)
	}
}

func setDiscordSignatureHeaders(req *http.Request, privateKey ed25519.PrivateKey, timestamp string, body []byte) {
	signature := ed25519.Sign(privateKey, append([]byte(timestamp), body...))
	req.Header.Set("X-Signature-Timestamp", timestamp)
	req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(signature))
}
