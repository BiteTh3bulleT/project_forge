package forgekshadow

import (
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/refvalidation"
)

const MemoryPalaceMirrorSchemaVersion = "memory-palace-mirror-v1"

type MemoryPalaceMirrorReport struct {
	SchemaVersion          string                    `json:"schema_version"`
	MirrorID               string                    `json:"mirror_id"`
	WorkspaceID            string                    `json:"workspace_id,omitempty"`
	RequestID              string                    `json:"request_id,omitempty"`
	CorrelationID          string                    `json:"correlation_id,omitempty"`
	SourceObservationID    string                    `json:"source_observation_id"`
	EvidenceRefs           []refvalidation.ObjectRef `json:"evidence_refs"`
	Warnings               []string                  `json:"warnings,omitempty"`
	DiagnosticOnly         bool                      `json:"diagnostic_only"`
	RetrievalExecution     bool                      `json:"retrieval_execution"`
	SearchExecution        bool                      `json:"search_execution"`
	EmbeddingExecution     bool                      `json:"embedding_execution"`
	MemoryMutation         bool                      `json:"memory_mutation"`
	EvidenceAdmission      bool                      `json:"evidence_admission"`
	ContextCompilation     bool                      `json:"context_compilation"`
	ModelRuntimeCall       bool                      `json:"model_runtime_call"`
	LiveAuthorityMigration bool                      `json:"live_authority_migration"`
	SimulatorAuthority     bool                      `json:"simulator_authority"`
}

func BuildMemoryPalaceMirrorFromRetrievalMetadata(observation RetrievalMetadataObservation) (MemoryPalaceMirrorReport, error) {
	workspaceID := strings.TrimSpace(observation.WorkspaceID)
	mirrorID := strings.TrimSpace(observation.ObservationID)
	if mirrorID == "" {
		mirrorID = "memory-palace-mirror"
	} else {
		mirrorID += ":memory_palace_mirror"
	}
	report := MemoryPalaceMirrorReport{
		SchemaVersion:          MemoryPalaceMirrorSchemaVersion,
		MirrorID:               mirrorID,
		WorkspaceID:            workspaceID,
		RequestID:              strings.TrimSpace(observation.RequestID),
		CorrelationID:          strings.TrimSpace(observation.CorrelationID),
		SourceObservationID:    strings.TrimSpace(observation.ObservationID),
		DiagnosticOnly:         true,
		RetrievalExecution:     false,
		SearchExecution:        false,
		EmbeddingExecution:     false,
		MemoryMutation:         false,
		EvidenceAdmission:      false,
		ContextCompilation:     false,
		ModelRuntimeCall:       false,
		LiveAuthorityMigration: false,
		SimulatorAuthority:     false,
	}
	if workspaceID == "" {
		return report, fmt.Errorf("%w: memory palace mirror requires workspace_id", ErrUnsafeMetadata)
	}

	refs := []refvalidation.ObjectRef{
		{RefType: "diagnostic_report", RefID: strings.TrimSpace(observation.ObservationID), WorkspaceID: workspaceID},
		{RefType: "retrieval_run", RefID: strings.TrimSpace(observation.RetrievalRunID), WorkspaceID: workspaceID},
		{RefType: "retrieval_result", RefID: strings.TrimSpace(observation.RetrievalResultID), WorkspaceID: workspaceID},
	}
	if sourceRef := sourceEvidenceRefFromRetrievalMetadata(observation, workspaceID); sourceRef.RefID != "" {
		refs = append(refs, sourceRef)
	}
	refs = nonEmptyMirrorRefs(refs)
	if len(refs) == 0 {
		report.Warnings = append(report.Warnings, "no_mirrorable_refs")
		report.EvidenceRefs = []refvalidation.ObjectRef{}
		return report, nil
	}
	result := refvalidation.ValidateRefs(refvalidation.ValidationRequest{
		ResultID:    mirrorID + ":evidence_refs",
		WorkspaceID: workspaceID,
		Refs:        refs,
	})
	if !result.Passed {
		return report, fmt.Errorf("%w: memory palace mirror evidence refs rejected", ErrUnsafeMetadata)
	}
	report.EvidenceRefs = append([]refvalidation.ObjectRef{}, result.NormalizedRefs...)
	return report, nil
}

func sourceEvidenceRefFromRetrievalMetadata(observation RetrievalMetadataObservation, workspaceID string) refvalidation.ObjectRef {
	refID := strings.TrimSpace(observation.SourceRefID)
	if refID == "" {
		return refvalidation.ObjectRef{}
	}
	refType := "semantic_object"
	switch strings.TrimSpace(observation.SourceType) {
	case "memory_ref":
		refType = "memory_note"
	case "retrieval_result":
		refType = "retrieval_result"
	case "packet":
		refType = "context_bundle"
	case "dossier", "source", "file", "chunk":
		refType = "semantic_object"
	}
	return refvalidation.ObjectRef{RefType: refType, RefID: refID, WorkspaceID: workspaceID}
}

func nonEmptyMirrorRefs(refs []refvalidation.ObjectRef) []refvalidation.ObjectRef {
	out := make([]refvalidation.ObjectRef, 0, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref.RefID) == "" {
			continue
		}
		out = append(out, ref)
	}
	return out
}
