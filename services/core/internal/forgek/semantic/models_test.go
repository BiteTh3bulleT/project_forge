package semantic

import (
	"testing"
	"time"
)

func TestSemanticObjectPreservesRefsAndProvenance(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)

	obj, err := NewSemanticObject(SemanticObjectInput{
		SemanticObjectID:  "semantic-1",
		WorkspaceID:       "workspace-a",
		ObjectType:        ObjectTypeEvidence,
		SourceObjectRefs:  []string{"exhibit-1"},
		SourceRefs:        []string{"doc:source"},
		ContentSummary:    "admitted evidence summary",
		NormalizedContent: "admitted evidence summary",
		Confidence:        0.9,
		AuthorityLevel:    AuthorityAdmitted,
		ProvenanceRefs:    []string{"event-1"},
		CreatedAt:         now,
		Metadata:          map[string]any{"domain": "architecture"},
	})
	if err != nil {
		t.Fatalf("NewSemanticObject failed: %v", err)
	}

	if obj.WorkspaceID != "workspace-a" || obj.ObjectType != ObjectTypeEvidence {
		t.Fatalf("object lost identity: %#v", obj)
	}
	if obj.SourceObjectRefs[0] != "exhibit-1" || obj.ProvenanceRefs[0] != "event-1" {
		t.Fatalf("object lost refs or provenance: %#v", obj)
	}
	if obj.IsCanonicalTruth() {
		t.Fatal("semantic object should not be canonical truth by construction")
	}

	clone := obj.Clone()
	clone.SourceRefs[0] = "tampered"
	if obj.SourceRefs[0] == "tampered" {
		t.Fatal("semantic object clone exposed mutable source refs")
	}
}

func TestSemanticOperationAndTransformResultPreserveProvenance(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)

	operation, err := NewSemanticOperation(SemanticOperationInput{
		OperationID:      "operation-1",
		OperationType:    OperationMerge,
		WorkspaceID:      "workspace-a",
		CaseID:           "case-1",
		InputObjectRefs:  []string{"semantic-1", "semantic-2"},
		OutputObjectRefs: []string{"semantic-3"},
		OperatorVersion:  "v1",
		ReasoningSummary: "merged compatible admitted evidence",
		ProvenanceRefs:   []string{"event-1"},
		CreatedBy:        "operator",
		CreatedAt:        now,
	})
	if err != nil {
		t.Fatalf("NewSemanticOperation failed: %v", err)
	}
	if operation.OperationType != OperationMerge || operation.InputObjectRefs[1] != "semantic-2" {
		t.Fatalf("operation lost operation data: %#v", operation)
	}

	result := SemanticTransformResult{
		ResultID:       "result-1",
		OperationID:    operation.OperationID,
		OperationType:  operation.OperationType,
		WorkspaceID:    operation.WorkspaceID,
		CaseID:         operation.CaseID,
		OutputRefs:     []string{"semantic-3"},
		ProvenanceRefs: []string{"event-1", "semantic-1"},
		CreatedAt:      now,
	}
	clone := result.Clone()
	clone.OutputRefs[0] = "tampered"
	if result.OutputRefs[0] == "tampered" {
		t.Fatal("transform result clone exposed mutable output refs")
	}
}

func TestOperatorRegistryDispatchesKnownOperators(t *testing.T) {
	registry := NewOperatorRegistry()
	if err := registry.Register(OperatorDefinition{
		OperationType: OperationMerge,
		Version:       "test",
		Deterministic: true,
		Handler: func(ctx OperatorContext) (SemanticTransformResult, error) {
			return SemanticTransformResult{OperationType: ctx.OperationType, WorkspaceID: ctx.WorkspaceID}, nil
		},
	}); err != nil {
		t.Fatalf("register operator: %v", err)
	}

	if _, ok := registry.Get(OperationMerge); !ok {
		t.Fatal("registered operator missing")
	}
	result, err := registry.Dispatch(OperatorContext{OperationType: OperationMerge, WorkspaceID: "workspace-a"})
	if err != nil {
		t.Fatalf("dispatch known operator: %v", err)
	}
	if result.OperationType != OperationMerge {
		t.Fatalf("unexpected result: %#v", result)
	}
	if _, err := registry.Dispatch(OperatorContext{OperationType: "UNKNOWN"}); err == nil {
		t.Fatal("unknown operator dispatched")
	}
}
