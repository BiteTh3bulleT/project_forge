package api

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

type authenticatedActor struct {
	Name   string
	Source string
}

type authContextKey struct{}

func (s *Server) requireAPIAuth(next http.Handler) http.Handler {
	token := strings.TrimSpace(s.cfg.APIToken)
	if token == "" {
		return next
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
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
