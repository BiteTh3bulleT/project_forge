package runtime

import "errors"

var (
	ErrInvalidDriverManifest     = errors.New("invalid runtime driver manifest")
	ErrInvalidDriverKind         = errors.New("invalid runtime driver kind")
	ErrDriverAlreadyRegistered   = errors.New("runtime driver already registered")
	ErrDriverNotFound            = errors.New("runtime driver not found")
	ErrInvalidCapabilityManifest = errors.New("invalid runtime capability manifest")
	ErrInvalidGenerateRequest    = errors.New("invalid runtime generate request")
	ErrRuntimeGenerationFailed   = errors.New("runtime generation failed")
	ErrSecretInManifest          = errors.New("runtime manifest must not contain secrets")
)
