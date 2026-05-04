package lymphatic

import "errors"

var (
	ErrInvalidPolicy       = errors.New("invalid lymphatic policy")
	ErrInvalidSweepKind    = errors.New("invalid lymphatic sweep kind")
	ErrInvalidSweepRequest = errors.New("invalid lymphatic sweep request")
	ErrInvalidProposal     = errors.New("invalid lymphatic cleanup proposal")
	ErrReportNotFound      = errors.New("lymphatic maintenance report not found")
	ErrProposalNotFound    = errors.New("lymphatic cleanup proposal not found")
)
