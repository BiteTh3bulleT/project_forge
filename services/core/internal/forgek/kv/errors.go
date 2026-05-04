package kv

import "errors"

var (
	ErrInvalidManifest        = errors.New("invalid kv cache manifest")
	ErrInvalidCacheMode       = errors.New("invalid kv cache mode")
	ErrInvalidMemoryTier      = errors.New("invalid kv memory tier")
	ErrInvalidStatus          = errors.New("invalid kv status")
	ErrManifestNotFound       = errors.New("kv cache manifest not found")
	ErrWorkspaceMismatch      = errors.New("kv workspace mismatch")
	ErrInvalidLookupRequest   = errors.New("invalid kv lookup request")
	ErrInvalidStateTransition = errors.New("invalid kv state transition")
)
