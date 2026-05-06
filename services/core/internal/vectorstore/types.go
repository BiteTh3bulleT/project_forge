package vectorstore

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

const (
	DefaultCollection = "forge_shadow_embeddings"
	PayloadSchemaV1   = "qdrant_shadow_vector_payload.v1"
)

type VectorStore interface {
	EnsureCollection(ctx context.Context, spec CollectionSpec) error
	UpsertVector(ctx context.Context, point VectorPoint) error
	SearchVector(ctx context.Context, req SearchRequest) (SearchResult, error)
	DeleteVector(ctx context.Context, collection, pointID string) error
	Health(ctx context.Context) error
}

type CollectionSpec struct {
	Name       string
	VectorSize int
	Distance   string
}

type VectorPoint struct {
	Collection string
	PointID    string
	Vector     []float64
	Payload    SafePayload
}

type SearchRequest struct {
	Collection string
	Vector     []float64
	Limit      int
}

type SearchResult struct {
	Matches []SearchMatch
}

type SearchMatch struct {
	PointID string
	Score   float64
	Payload SafePayload
}

type SafePayload struct {
	ObjectRefID       string            `json:"object_ref_id,omitempty"`
	SourceRefID       string            `json:"source_ref_id,omitempty"`
	WorkspaceID       string            `json:"workspace_id,omitempty"`
	EmbeddingRecordID string            `json:"embedding_record_id,omitempty"`
	EmbeddingModelID  string            `json:"embedding_model_id,omitempty"`
	EmbeddingDims     int               `json:"embedding_dims,omitempty"`
	SourceHash        string            `json:"source_hash,omitempty"`
	RetrievalStrategy string            `json:"retrieval_strategy,omitempty"`
	IndexClass        string            `json:"index_class,omitempty"`
	CreatedAtUnix     int64             `json:"created_at,omitempty"`
	SchemaVersion     string            `json:"schema_version,omitempty"`
	ProvenanceRefID   string            `json:"provenance_ref_id,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

func (p SafePayload) Validate(maxMetadataBytes int) error {
	if strings.TrimSpace(p.ObjectRefID) == "" && strings.TrimSpace(p.SourceRefID) == "" {
		return ErrInvalidPayload
	}
	if strings.TrimSpace(p.EmbeddingRecordID) == "" {
		return ErrInvalidPayload
	}
	if strings.TrimSpace(p.EmbeddingModelID) == "" || p.EmbeddingDims <= 0 {
		return ErrInvalidPayload
	}
	if strings.TrimSpace(p.ProvenanceRefID) == "" {
		return ErrMissingProvenance
	}
	if maxMetadataBytes <= 0 {
		maxMetadataBytes = 8192
	}
	total := 0
	for key, value := range p.Metadata {
		if unsafePayloadKey(key) || unsafePayloadKey(value) {
			return ErrUnsafePayload
		}
		total += len(key) + len(value)
		if total > maxMetadataBytes {
			return ErrUnsafePayload
		}
	}
	if unsafePayloadKey(p.ObjectRefID) ||
		unsafePayloadKey(p.SourceRefID) ||
		unsafePayloadKey(p.WorkspaceID) ||
		unsafePayloadKey(p.EmbeddingRecordID) ||
		unsafePayloadKey(p.EmbeddingModelID) ||
		unsafePayloadKey(p.SourceHash) ||
		unsafePayloadKey(p.RetrievalStrategy) ||
		unsafePayloadKey(p.IndexClass) ||
		unsafePayloadKey(p.ProvenanceRefID) {
		return ErrUnsafePayload
	}
	return nil
}

func (p SafePayload) Normalized() SafePayload {
	out := p
	out.ObjectRefID = strings.TrimSpace(out.ObjectRefID)
	out.SourceRefID = strings.TrimSpace(out.SourceRefID)
	out.WorkspaceID = strings.TrimSpace(out.WorkspaceID)
	out.EmbeddingRecordID = strings.TrimSpace(out.EmbeddingRecordID)
	out.EmbeddingModelID = strings.TrimSpace(out.EmbeddingModelID)
	out.SourceHash = strings.TrimSpace(out.SourceHash)
	out.RetrievalStrategy = strings.TrimSpace(out.RetrievalStrategy)
	out.IndexClass = strings.TrimSpace(out.IndexClass)
	out.ProvenanceRefID = strings.TrimSpace(out.ProvenanceRefID)
	if out.SchemaVersion == "" {
		out.SchemaVersion = PayloadSchemaV1
	}
	if len(out.Metadata) > 0 {
		keys := make([]string, 0, len(out.Metadata))
		for key := range out.Metadata {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		next := make(map[string]string, len(out.Metadata))
		for _, key := range keys {
			next[strings.TrimSpace(key)] = strings.TrimSpace(out.Metadata[key])
		}
		out.Metadata = next
	}
	return out
}

func ValidateVector(vector []float64, expectedDims int) error {
	if len(vector) == 0 {
		return ErrInvalidVector
	}
	if expectedDims > 0 && len(vector) != expectedDims {
		return ErrInvalidVectorDimensions
	}
	return nil
}

func DeterministicPointID(payload SafePayload) string {
	p := payload.Normalized()
	seed := strings.Join([]string{
		p.WorkspaceID,
		p.ObjectRefID,
		p.SourceRefID,
		p.EmbeddingRecordID,
		p.EmbeddingModelID,
		p.SourceHash,
		p.ProvenanceRefID,
	}, "\x00")
	sum := sha256.Sum256([]byte(seed))
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

func unsafePayloadKey(value string) bool {
	token := strings.ToLower(strings.TrimSpace(value))
	token = strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(token)
	for _, forbidden := range []string{
		"source_text",
		"chunk_text",
		"document_content",
		"content",
		"prompt",
		"completion",
		"message_body",
		"tool_payload",
		"tool_output",
		"memory_content",
		"raw_query",
		"auth",
		"cookie",
		"token",
		"secret",
		"api_key",
		"password",
	} {
		if strings.Contains(token, forbidden) {
			return true
		}
	}
	return false
}
