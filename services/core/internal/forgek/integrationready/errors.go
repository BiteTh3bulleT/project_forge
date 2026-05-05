package integrationready

import "errors"

var (
	ErrMissingRequiredField = errors.New("missing required field")
	ErrInvalidStatus        = errors.New("invalid readiness status")
	ErrInvalidAdapterType   = errors.New("invalid adapter type")
	ErrInvalidShadowPolicy  = errors.New("invalid shadow mode policy")
	ErrLiveMutationAllowed  = errors.New("live mutation is not allowed in phase 11f")
	ErrLiveSideEffect       = errors.New("live side effect is not allowed in phase 11f")
)
