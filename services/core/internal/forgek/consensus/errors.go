package consensus

import "errors"

var (
	ErrInvalidClaim           = errors.New("invalid consensus claim")
	ErrInvalidEvidenceRef     = errors.New("invalid consensus evidence ref")
	ErrInvalidAgentRun        = errors.New("invalid consensus agent run")
	ErrInvalidPolicy          = errors.New("invalid consensus policy")
	ErrInvalidRequest         = errors.New("invalid consensus request")
	ErrObjectNotFound         = errors.New("consensus object not found")
	ErrUnsupportedClaim       = errors.New("claim lacks eligible support")
	ErrComposerInputRejected  = errors.New("composer input rejected")
	ErrRejectedClaimReference = errors.New("rejected claim cannot be used by composer")
)
