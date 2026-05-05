package shadowharness

import "errors"

var (
	ErrMissingRequiredField = errors.New("missing required field")
	ErrSecretMetadata       = errors.New("secret-looking metadata is not allowed")
	ErrSideEffectAllowed    = errors.New("shadow harness side effect is not allowed")
	ErrNoEffectNotVerified  = errors.New("no-effect guarantee is not verified")
)
