package vectorstore

import (
	"errors"
	"testing"
)

func TestSafePayloadValidatesAllowedRefs(t *testing.T) {
	payload := validPayload()
	if err := payload.Validate(1024); err != nil {
		t.Fatalf("expected safe payload to validate: %v", err)
	}
	if payload.Normalized().SchemaVersion != PayloadSchemaV1 {
		t.Fatalf("expected schema version to default")
	}
}

func TestSafePayloadRequiresRefsAndProvenance(t *testing.T) {
	payload := validPayload()
	payload.ObjectRefID = ""
	payload.SourceRefID = ""
	if !errors.Is(payload.Validate(1024), ErrInvalidPayload) {
		t.Fatalf("expected missing object/source refs to fail")
	}
	payload = validPayload()
	payload.ProvenanceRefID = ""
	if !errors.Is(payload.Validate(1024), ErrMissingProvenance) {
		t.Fatalf("expected missing provenance to fail")
	}
}

func TestSafePayloadRejectsForbiddenContentFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		mut  func(*SafePayload)
	}{
		{"source text", func(p *SafePayload) { p.Metadata["source_text"] = "raw source" }},
		{"chunk text", func(p *SafePayload) { p.Metadata["chunk_text"] = "raw chunk" }},
		{"prompt", func(p *SafePayload) { p.Metadata["prompt_text"] = "prompt" }},
		{"completion", func(p *SafePayload) { p.Metadata["completion_text"] = "completion" }},
		{"message body", func(p *SafePayload) { p.Metadata["message_body"] = "hello" }},
		{"memory content", func(p *SafePayload) { p.Metadata["memory_content"] = "memory" }},
		{"raw query", func(p *SafePayload) { p.Metadata["raw_query"] = "query" }},
		{"auth", func(p *SafePayload) { p.Metadata["auth_header"] = "bearer" }},
		{"cookie", func(p *SafePayload) { p.Metadata["cookie"] = "session" }},
		{"secret", func(p *SafePayload) { p.Metadata["safe_key"] = "contains secret value" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := validPayload()
			tc.mut(&payload)
			if !errors.Is(payload.Validate(1024), ErrUnsafePayload) {
				t.Fatalf("expected unsafe payload rejection")
			}
		})
	}
}

func TestSafePayloadRejectsOversizedMetadata(t *testing.T) {
	payload := validPayload()
	payload.Metadata["safe_ref"] = "abcdefghijklmnopqrstuvwxyz"
	if !errors.Is(payload.Validate(8), ErrUnsafePayload) {
		t.Fatalf("expected oversized payload rejection")
	}
}

func TestVectorDimensionsValidated(t *testing.T) {
	if err := ValidateVector([]float64{0.1, 0.2}, 2); err != nil {
		t.Fatalf("valid vector rejected: %v", err)
	}
	if !errors.Is(ValidateVector(nil, 2), ErrInvalidVector) {
		t.Fatalf("expected nil vector rejection")
	}
	if !errors.Is(ValidateVector([]float64{0.1}, 2), ErrInvalidVectorDimensions) {
		t.Fatalf("expected dimension mismatch")
	}
}

func TestDeterministicPointIDStable(t *testing.T) {
	one := DeterministicPointID(validPayload())
	two := DeterministicPointID(validPayload())
	if one == "" || one != two {
		t.Fatalf("expected stable point id, got %q and %q", one, two)
	}
	changed := validPayload()
	changed.SourceHash = "sha256:def"
	if one == DeterministicPointID(changed) {
		t.Fatalf("expected changed semantic identity to alter point id")
	}
}

func validPayload() SafePayload {
	return SafePayload{
		ObjectRefID:       "chunk:1",
		SourceRefID:       "source:1",
		WorkspaceID:       "workspace-a",
		EmbeddingRecordID: "embedding:1",
		EmbeddingModelID:  "local-hash-128",
		EmbeddingDims:     3,
		SourceHash:        "sha256:abc",
		RetrievalStrategy: "shadow",
		IndexClass:        "retrieval_shadow",
		CreatedAtUnix:     1778000000,
		ProvenanceRefID:   "retrieval-metadata:1",
		Metadata: map[string]string{
			"safe_ref": "ref:1",
		},
	}
}
