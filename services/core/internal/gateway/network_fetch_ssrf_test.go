package gateway

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type staticOutboundResolver map[string][]net.IPAddr

func (r staticOutboundResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	if addrs, ok := r[host]; ok {
		return addrs, nil
	}
	return nil, errors.New("unexpected host lookup: " + host)
}

func ipAddr(raw string) net.IPAddr {
	return net.IPAddr{IP: net.ParseIP(raw)}
}

func TestValidateOutboundHTTPURLRejectsInternalTargets(t *testing.T) {
	t.Parallel()

	cases := []string{
		"http://127.0.0.1/",
		"http://localhost/",
		"http://[::1]/",
		"http://10.0.0.1/",
		"http://192.168.0.1/",
		"http://172.16.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://[fd00::1]/",
	}
	resolver := staticOutboundResolver{
		"localhost": []net.IPAddr{ipAddr("127.0.0.1"), ipAddr("::1")},
	}
	for _, raw := range cases {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			_, err := validateOutboundHTTPURL(context.Background(), raw, resolver)
			if err == nil {
				t.Fatalf("expected %s to be rejected", raw)
			}
			if !strings.Contains(err.Error(), "blocked outbound target") {
				t.Fatalf("expected blocked outbound target error, got %v", err)
			}
		})
	}
}

func TestValidateOutboundHTTPURLRejectsPrivateDNSResults(t *testing.T) {
	t.Parallel()

	resolver := staticOutboundResolver{
		"internal.example": []net.IPAddr{ipAddr("203.0.113.10"), ipAddr("10.1.2.3")},
	}

	_, err := validateOutboundHTTPURL(context.Background(), "https://internal.example/path", resolver)
	if err == nil {
		t.Fatal("expected private DNS result to be rejected")
	}
	if !strings.Contains(err.Error(), "blocked outbound target") {
		t.Fatalf("expected blocked outbound target error, got %v", err)
	}
}

func TestValidateOutboundHTTPURLAllowsPublicTargets(t *testing.T) {
	t.Parallel()

	resolver := staticOutboundResolver{
		"example.com": []net.IPAddr{ipAddr("93.184.216.34")},
	}

	parsed, err := validateOutboundHTTPURL(context.Background(), "https://example.com/path", resolver)
	if err != nil {
		t.Fatalf("expected public target to be allowed: %v", err)
	}
	if parsed.String() != "https://example.com/path" {
		t.Fatalf("unexpected parsed URL: %s", parsed.String())
	}
}

func TestValidateOutboundHTTPURLRejectsUserInfo(t *testing.T) {
	t.Parallel()

	_, err := validateOutboundHTTPURL(context.Background(), "https://user:pass@example.com/path", staticOutboundResolver{})
	if err == nil {
		t.Fatal("expected userinfo URL to be rejected")
	}
	if !strings.Contains(err.Error(), "userinfo") {
		t.Fatalf("expected userinfo error, got %v", err)
	}
}

func TestValidateOutboundHTTPURLRejectsOversizeURL(t *testing.T) {
	t.Parallel()

	raw := "https://example.com/" + strings.Repeat("x", maxOutboundHTTPURLBytes)
	_, err := validateOutboundHTTPURL(context.Background(), raw, staticOutboundResolver{
		"example.com": []net.IPAddr{ipAddr("93.184.216.34")},
	})
	if !errors.Is(err, errOutboundHTTPURLTooLarge) {
		t.Fatalf("expected outbound URL size rejection, got %v", err)
	}
}

func TestNetworkFetchRedirectPolicyRejectsInternalTarget(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1/admin", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	err = validateOutboundHTTPRedirect(context.Background(), req, staticOutboundResolver{})
	if err == nil {
		t.Fatal("expected redirect to internal target to be rejected")
	}
	if !strings.Contains(err.Error(), "blocked outbound target") {
		t.Fatalf("expected blocked outbound target error, got %v", err)
	}
}

func TestGuardedOutboundHTTPClientDoesNotUseEnvironmentProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:8080")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:8443")

	client := newGuardedOutboundHTTPClient(context.Background(), staticOutboundResolver{})
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected http.Transport, got %T", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("net.fetch transport must not inherit environment proxy settings")
	}
}

func TestNetworkFetchToolRejectsInternalTargetBeforeDial(t *testing.T) {
	t.Parallel()

	_, err := (&networkFetchTool{}).Execute(context.Background(), Request{
		Input: map[string]any{"url": "http://127.0.0.1:1/admin"},
	})
	if err == nil {
		t.Fatal("expected internal target to be rejected")
	}
	if !strings.Contains(err.Error(), "blocked outbound target") {
		t.Fatalf("expected blocked outbound target error, got %v", err)
	}
}

func TestCapabilityFetchURLRejectsInternalTargetBeforeDial(t *testing.T) {
	t.Parallel()

	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := fetchURL(context.Background(), server.URL, http.MethodGet)
	if err == nil {
		t.Fatal("expected internal target to be rejected")
	}
	if called {
		t.Fatal("capability fetch dialed internal test server before rejecting target")
	}
	if !strings.Contains(err.Error(), "blocked outbound target") {
		t.Fatalf("expected blocked outbound target error, got %v", err)
	}
}

func TestGatewayApprovedNetworkFetchStillRejectsInternalTarget(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, _, _ := newToolSurfaceGatewayHarness(t)

	req := Request{
		ToolID:        "net.fetch",
		LaneID:        "net.fetch",
		CorrelationID: "corr-net-fetch-ssrf-approved",
		TraceID:       "trace-net-fetch-ssrf-approved",
		Input:         map[string]any{"url": "http://127.0.0.1:1/admin"},
		Initiator:     "tester",
	}
	approvalID := approveGatewayRequestForTest(t, ctx, gw, req, "approved network fetch still must respect outbound target policy")
	req.ApprovalID = strconv.FormatInt(approvalID, 10)

	res, err := gw.Execute(ctx, req)
	if err != nil {
		t.Fatalf("execute approved net.fetch: %v", err)
	}
	if res.Status != StatusError {
		t.Fatalf("expected approved internal net.fetch to fail at tool boundary, got %s", res.Status)
	}
	if !strings.Contains(res.Message, "blocked outbound target") {
		t.Fatalf("expected blocked outbound target message, got %q", res.Message)
	}
}
