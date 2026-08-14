package api

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"net"
	"net/http"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/authproof"
)

type authenticatedActor struct {
	Name   string
	Source string
}

type authContextKey struct{}

func (s *Server) requireAPIAuth(next http.Handler) http.Handler {
	token := strings.TrimSpace(s.cfg.APIToken)
	if token == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !verifiedLoopbackRequest(r) {
				next.ServeHTTP(w, r)
				return
			}
			actor := strings.TrimSpace(s.cfg.APIActor)
			if actor == "" {
				actor = "operator"
			}
			ctx := context.WithValue(r.Context(), authContextKey{}, authenticatedActor{Name: actor, Source: "local_loopback"})
			ctx = authproof.WithTrustedOrigin(ctx, localLoopbackOrigin(actor))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := bearerToken(r.Header.Get("Authorization"))
		if !constantTimeTokenMatch(provided, token) {
			writeAuthError(w)
			return
		}
		actor := strings.TrimSpace(s.cfg.APIActor)
		if actor == "" {
			actor = "operator"
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, authenticatedActor{
			Name:   actor,
			Source: "api_bearer",
		})
		ctx = authproof.WithTrustedOrigin(ctx, bearerOrigin(actor, token))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerOrigin(actor, token string) authproof.PrincipalRecord {
	fingerprint := authproof.CredentialFingerprint(token)
	return authproof.PrincipalRecord{
		RecordID: "api_bearer:" + strings.TrimPrefix(fingerprint, "sha256:"), Version: "forge.api.bearer_identity.v1",
		SubjectID: strings.TrimSpace(actor), SubjectKind: "user", Source: domain.SourceUser,
		Issuer: "forge.api.bearer", CredentialFingerprint: fingerprint,
		Status: authproof.StatusActive, AuthenticatedAt: 1,
	}
}

func localLoopbackOrigin(actor string) authproof.PrincipalRecord {
	fingerprint := authproof.CredentialFingerprint("local_loopback:" + strings.TrimSpace(actor))
	return authproof.PrincipalRecord{
		RecordID: "local_loopback:" + strings.TrimPrefix(fingerprint, "sha256:"), Version: "forge.api.local_loopback_identity.v1",
		SubjectID: strings.TrimSpace(actor), SubjectKind: "user", Source: domain.SourceUser,
		Issuer: "forge.api.local_loopback", CredentialFingerprint: fingerprint,
		Status: authproof.StatusActive, AuthenticatedAt: 1,
	}
}

func verifiedLoopbackRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func authenticatedActorName(r *http.Request) string {
	if r == nil {
		return "operator"
	}
	if actor, ok := r.Context().Value(authContextKey{}).(authenticatedActor); ok {
		if name := strings.TrimSpace(actor.Name); name != "" {
			return name
		}
	}
	return "operator"
}

func authenticatedActorSource(r *http.Request) string {
	if r == nil {
		return "operator"
	}
	if actor, ok := r.Context().Value(authContextKey{}).(authenticatedActor); ok {
		if source := strings.TrimSpace(actor.Source); source != "" {
			return source
		}
	}
	return "operator"
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func constantTimeTokenMatch(provided, expected string) bool {
	provided = strings.TrimSpace(provided)
	expected = strings.TrimSpace(expected)
	if provided == "" || expected == "" {
		return false
	}
	providedHash := sha256.Sum256([]byte(provided))
	expectedHash := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) == 1
}

func writeAuthError(w http.ResponseWriter) {
	writeAPIError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid bearer token", nil)
}
