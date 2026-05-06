package storagebackend

import "errors"

var (
	ErrInvalidBackend      = errors.New("invalid storage backend")
	ErrPostgresDSNRequired = errors.New("postgres backend requires FORGE_POSTGRES_DSN")
	ErrInvalidInfraKind    = errors.New("invalid storage infrastructure kind")
	ErrCanonicalDisallowed = errors.New("infrastructure cannot be canonical truth")
)
