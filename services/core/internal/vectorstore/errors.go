package vectorstore

import "errors"

var (
	ErrDisabled                = errors.New("qdrant shadow index disabled")
	ErrInvalidConfig           = errors.New("invalid vector store config")
	ErrInvalidCollection       = errors.New("invalid vector collection")
	ErrInvalidVector           = errors.New("invalid vector")
	ErrInvalidPayload          = errors.New("invalid vector payload")
	ErrUnsafePayload           = errors.New("unsafe vector payload")
	ErrMissingProvenance       = errors.New("missing provenance reference")
	ErrInvalidVectorDimensions = errors.New("invalid vector dimensions")
)
