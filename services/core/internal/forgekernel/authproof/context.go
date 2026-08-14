package authproof

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type trustedOriginContextKey struct{}

// WithTrustedOrigin attaches identity evidence produced by an authenticated
// ingress. Callers must never construct an origin from request JSON or actor
// headers; the production authorization port treats absence as fail-closed.
func WithTrustedOrigin(ctx context.Context, origin PrincipalRecord) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, trustedOriginContextKey{}, origin)
}

// TrustedOriginFromContext returns the ingress-authenticated origin, if any.
func TrustedOriginFromContext(ctx context.Context) (PrincipalRecord, bool) {
	if ctx == nil {
		return PrincipalRecord{}, false
	}
	origin, ok := ctx.Value(trustedOriginContextKey{}).(PrincipalRecord)
	return origin, ok
}

// CredentialFingerprint returns a non-secret stable identifier. It is safe to
// persist, but the credential used to derive it must never be stored in proof.
func CredentialFingerprint(credential string) string {
	sum := sha256.Sum256([]byte("forge_k.credential_fingerprint.v1\n" + strings.TrimSpace(credential)))
	return "sha256:" + hex.EncodeToString(sum[:])
}
