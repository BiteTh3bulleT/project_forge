package semantic

import "errors"

var (
	ErrInvalidSemanticObject     = errors.New("invalid semantic object")
	ErrInvalidSemanticOperation  = errors.New("invalid semantic operation")
	ErrInvalidTransformResult    = errors.New("invalid semantic transform result")
	ErrUnknownOperator           = errors.New("unknown semantic operator")
	ErrInvalidOperation          = errors.New("invalid semantic operation request")
	ErrContradictionMerge        = errors.New("cannot silently merge contradicted semantic objects")
	ErrRejectedEvidenceInput     = errors.New("rejected evidence cannot be treated as admitted semantic input")
	ErrSemanticObjectNotFound    = errors.New("semantic object not found")
	ErrSemanticOperationNotFound = errors.New("semantic operation not found")
)
