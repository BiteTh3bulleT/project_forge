package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/forgekernel/authproof"
)

func TestRequireAPIAuthRejectsMissingOrInvalidBearerToken(t *testing.T) {
	srv := &Server{cfg: config.Config{APIToken: "secret"}}
	handler := srv.requireAPIAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, header := range []string{"", "secret", "Bearer wrong"} {
		req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("header %q status=%d, want 401", header, rr.Code)
		}
		if !strings.Contains(rr.Body.String(), `"code":"unauthorized"`) {
			t.Fatalf("expected structured auth error, got %s", rr.Body.String())
		}
	}
}

func TestRequireAPIAuthAcceptsBearerTokenAndSetsActor(t *testing.T) {
	srv := &Server{cfg: config.Config{APIToken: "secret", APIActor: "alice"}}
	handler := srv.requireAPIAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := authenticatedActorName(r); got != "alice" {
			t.Fatalf("actor=%q, want alice", got)
		}
		origin, ok := authproof.TrustedOriginFromContext(r.Context())
		if !ok || origin.SubjectID != "alice" || origin.SubjectKind != "user" || origin.Source != domain.SourceUser || origin.Issuer != "forge.api.bearer" {
			t.Fatalf("bearer origin=%#v present=%v", origin, ok)
		}
		if origin.CredentialFingerprint == "" || strings.Contains(origin.RecordID, "secret") || strings.Contains(origin.CredentialFingerprint, "secret") {
			t.Fatalf("bearer proof exposed credential: %#v", origin)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want 204", rr.Code)
	}
}

func TestNoTokenAttestsOnlyVerifiedLoopback(t *testing.T) {
	srv := &Server{cfg: config.Config{APIActor: "local-operator"}}
	for _, tc := range []struct {
		name       string
		remoteAddr string
		wantOrigin bool
	}{
		{name: "ipv4 loopback", remoteAddr: "127.0.0.1:4123", wantOrigin: true},
		{name: "ipv6 loopback", remoteAddr: "[::1]:4123", wantOrigin: true},
		{name: "remote", remoteAddr: "192.0.2.20:4123", wantOrigin: false},
		{name: "missing", remoteAddr: "", wantOrigin: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := srv.requireAPIAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				origin, ok := authproof.TrustedOriginFromContext(r.Context())
				if ok != tc.wantOrigin {
					t.Fatalf("origin=%#v present=%v want=%v", origin, ok, tc.wantOrigin)
				}
				if ok && (origin.Issuer != "forge.api.local_loopback" || origin.SubjectID != "local-operator") {
					t.Fatalf("unexpected loopback origin: %#v", origin)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
			req.RemoteAddr = tc.remoteAddr
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusNoContent {
				t.Fatalf("status=%d", rr.Code)
			}
		})
	}
}

func TestCORSDefaultsOnlyAllowTauriOrigins(t *testing.T) {
	srv := &Server{}
	for _, origin := range []string{"", "tauri://localhost", "http://tauri.localhost", "https://tauri.localhost"} {
		if !srv.corsOriginAllowed(origin) {
			t.Fatalf("expected origin %q to be allowed", origin)
		}
	}
	for _, origin := range []string{"http://localhost:5173", "http://127.0.0.1:5173", "https://example.com"} {
		if srv.corsOriginAllowed(origin) {
			t.Fatalf("expected origin %q to be rejected by default", origin)
		}
	}
}

func TestCORSAllowsLocalhostOnlyWithDevFlag(t *testing.T) {
	srv := &Server{cfg: config.Config{CORSAllowDevLocalhost: true}}
	if !srv.corsOriginAllowed("http://localhost:5173") || !srv.corsOriginAllowed("http://127.0.0.1:5173") {
		t.Fatalf("expected localhost origins to be allowed when dev flag is enabled")
	}
}

func TestConstantTimeTokenMatchDoesNotEarlyReturnOnLengthMismatch(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	body, err := os.ReadFile(strings.TrimSuffix(file, "_test.go") + ".go")
	if err != nil {
		t.Fatalf("read auth source: %v", err)
	}
	if strings.Contains(string(body), "len(provided) != len(expected)") {
		t.Fatal("constantTimeTokenMatch must not early-return on token length mismatch")
	}
}
