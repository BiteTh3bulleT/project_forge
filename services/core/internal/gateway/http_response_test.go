package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestReadGatewayHTTPResponseBodyRejectsOversize(t *testing.T) {
	t.Parallel()

	_, err := readGatewayHTTPResponseBody(strings.NewReader(strings.Repeat("x", 6)), "test response", 5)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too large error, got %v", err)
	}
}

func TestCapabilityHTTPResponseReaderRejectsOversizeResponse(t *testing.T) {
	t.Parallel()

	_, err := readGatewayHTTPResponseBody(
		strings.NewReader(strings.Repeat("x", int(gatewayCapabilityHTTPResponseBodyLimit)+1)),
		"capability http request",
		gatewayCapabilityHTTPResponseBodyLimit,
	)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too large error, got %v", err)
	}
}

func TestDesktopBridgeRejectsOversizeResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", int(gatewayDesktopBridgeResponseBodyLimit)+1)))
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	_, port, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("split server host: %v", err)
	}
	t.Setenv("FORGE_DESKTOP_BRIDGE_PORT", port)
	t.Setenv("FORGE_DESKTOP_BRIDGE_TOKEN", "test-token")

	tool := capabilityBackingTool{capability: domain.ToolCapability{ID: "test.capability"}}
	_, err = tool.callDesktopBridge(context.Background(), Request{})
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too large error, got %v", err)
	}
}
