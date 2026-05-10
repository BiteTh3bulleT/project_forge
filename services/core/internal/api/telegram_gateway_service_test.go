package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNormalizeWakeCommand(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "/wake", want: "wake_forge"},
		{in: "wake forge", want: "wake_forge"},
		{in: "forge wake", want: "wake_forge"},
		{in: "/wake pc", want: "wake_computer"},
		{in: "wake computer", want: "wake_computer"},
		{in: "hello", want: ""},
	}
	for _, tc := range tests {
		if got := normalizeWakeCommand(tc.in); got != tc.want {
			t.Fatalf("normalizeWakeCommand(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseMAC(t *testing.T) {
	valid := []string{
		"AA:BB:CC:DD:EE:FF",
		"aa-bb-cc-dd-ee-ff",
		"aabb.ccdd.eeff",
		"aabbccddeeff",
	}
	for _, in := range valid {
		got, err := parseMAC(in)
		if err != nil {
			t.Fatalf("parseMAC(%q) unexpected err: %v", in, err)
		}
		if len(got) != 6 {
			t.Fatalf("parseMAC(%q) len=%d want 6", in, len(got))
		}
	}
	if _, err := parseMAC("xyz"); err == nil {
		t.Fatalf("parseMAC invalid input expected error")
	}
}

func TestProcessWakeCommandRejectsOversizeText(t *testing.T) {
	g := &TelegramGateway{cfg: telegramGatewayConfig{EnableWakeCommands: true}}
	msg := &telegramMessage{Text: strings.Repeat("a", remoteInboundTextLimit+1)}

	handled, err := g.processWakeCommandIfPresent(context.Background(), remoteConfig{}, msg)
	if !handled {
		t.Fatalf("expected oversize wake-command text to be handled as rejected")
	}
	if !errors.Is(err, errRemoteMessageTooLarge) {
		t.Fatalf("expected oversize wake-command text error, got %v", err)
	}
}

func TestTelegramGatewayFetchUpdatesRejectsOversizeResponse(t *testing.T) {
	g := &TelegramGateway{
		cfg: telegramGatewayConfig{BotToken: "token", PollTimeoutS: 1, MaxBatchItems: 1},
		client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body := `{"ok":true,"result":[]}` + strings.Repeat(" ", telegramPollResponseBodyLimit+1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		})},
	}

	_, err := g.fetchUpdates(context.Background(), 0)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected response too large error, got %v", err)
	}
}
