package forgekshadow

import (
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/refvalidation"
)

const ContextBundleShadowSchemaVersion = "context-bundle-shadow-v1"

type ContextBundleShadowReport struct {
	SchemaVersion               string                     `json:"schema_version"`
	BundleID                    string                     `json:"bundle_id"`
	BundleHash                  string                     `json:"bundle_hash"`
	WorkspaceID                 string                     `json:"workspace_id"`
	RequestID                   string                     `json:"request_id,omitempty"`
	CorrelationID               string                     `json:"correlation_id,omitempty"`
	SourceObservationID         string                     `json:"source_observation_id"`
	SourceAction                string                     `json:"source_action"`
	AdmissionStatus             string                     `json:"admission_status"`
	LayoutVersion               string                     `json:"layout_version"`
	EvidenceRefs                []refvalidation.ObjectRef  `json:"evidence_refs"`
	Blocks                      []ContextBundleShadowBlock `json:"blocks"`
	Warnings                    []string                   `json:"warnings,omitempty"`
	DiagnosticOnly              bool                       `json:"diagnostic_only"`
	ShadowContextShapeGenerated bool                       `json:"shadow_context_shape_generated"`
	LiveContextCompilation      bool                       `json:"live_context_compilation"`
	PromptAuthority             bool                       `json:"prompt_authority"`
	ModelRuntimeCall            bool                       `json:"model_runtime_call"`
	RetrievalExecution          bool                       `json:"retrieval_execution"`
	SearchExecution             bool                       `json:"search_execution"`
	EmbeddingExecution          bool                       `json:"embedding_execution"`
	MemoryMutation              bool                       `json:"memory_mutation"`
	EvidenceAdmission           bool                       `json:"evidence_admission"`
	UserVisibleOutput           bool                       `json:"user_visible_output"`
	LiveAuthorityMigration      bool                       `json:"live_authority_migration"`
	SimulatorAuthority          bool                       `json:"simulator_authority"`
}

type ContextBundleShadowBlock struct {
	BlockID          string                    `json:"block_id"`
	Label            string                    `json:"label"`
	Source           string                    `json:"source"`
	IncludedRefs     []refvalidation.ObjectRef `json:"included_refs,omitempty"`
	ExcludedRefCount int                       `json:"excluded_ref_count"`
	CacheEligibility string                    `json:"cache_eligibility"`
}

func BuildContextBundleShadowFromControlLaneValidation(observation ControlLaneValidationObservation) (ContextBundleShadowReport, bool, error) {
	if !isContextBundleShadowCandidate(observation) {
		return ContextBundleShadowReport{}, false, nil
	}
	workspaceID := strings.TrimSpace(observation.WorkspaceID)
	if workspaceID == "" {
		return ContextBundleShadowReport{}, false, fmt.Errorf("%w: context bundle shadow requires workspace_id", ErrUnsafeMetadata)
	}
	if len(observation.NormalizedRefs) == 0 {
		return ContextBundleShadowReport{}, false, nil
	}
	bundleID := strings.TrimSpace(observation.ObservationID)
	if bundleID == "" {
		bundleID = "context-bundle-shadow"
	} else {
		bundleID += ":context_bundle_shadow"
	}
	result := refvalidation.ValidateRefs(refvalidation.ValidationRequest{
		ResultID:    bundleID + ":evidence_refs",
		WorkspaceID: workspaceID,
		Refs:        observation.NormalizedRefs,
	})
	if !result.Passed {
		return ContextBundleShadowReport{}, false, fmt.Errorf("%w: context bundle shadow evidence refs rejected", ErrUnsafeMetadata)
	}
	refs := append([]refvalidation.ObjectRef{}, result.NormalizedRefs...)
	blocks := []ContextBundleShadowBlock{
		{
			BlockID:          bundleID + ":block:evidence_refs",
			Label:            "admission_candidate_evidence_refs",
			Source:           "control_lane_validation_metadata",
			IncludedRefs:     refs,
			ExcludedRefCount: 0,
			CacheEligibility: "shape_only_cacheable",
		},
	}
	report := ContextBundleShadowReport{
		SchemaVersion:               ContextBundleShadowSchemaVersion,
		BundleID:                    bundleID,
		WorkspaceID:                 workspaceID,
		RequestID:                   strings.TrimSpace(observation.RequestID),
		CorrelationID:               strings.TrimSpace(observation.CorrelationID),
		SourceObservationID:         strings.TrimSpace(observation.ObservationID),
		SourceAction:                observation.Action,
		AdmissionStatus:             "candidate_accepted_not_live_admitted",
		LayoutVersion:               "shadow-context-layout-v1",
		EvidenceRefs:                refs,
		Blocks:                      blocks,
		DiagnosticOnly:              true,
		ShadowContextShapeGenerated: true,
		LiveContextCompilation:      false,
		PromptAuthority:             false,
		ModelRuntimeCall:            false,
		RetrievalExecution:          false,
		SearchExecution:             false,
		EmbeddingExecution:          false,
		MemoryMutation:              false,
		EvidenceAdmission:           false,
		UserVisibleOutput:           false,
		LiveAuthorityMigration:      false,
		SimulatorAuthority:          false,
	}
	report.BundleHash = advisoryHash(map[string]any{
		"schema_version":        report.SchemaVersion,
		"workspace_id":          report.WorkspaceID,
		"request_id":            report.RequestID,
		"source_observation_id": report.SourceObservationID,
		"source_action":         report.SourceAction,
		"admission_status":      report.AdmissionStatus,
		"layout_version":        report.LayoutVersion,
		"evidence_refs":         refs,
		"blocks":                blocks,
	})
	return report, true, nil
}

func isContextBundleShadowCandidate(observation ControlLaneValidationObservation) bool {
	return observation.Passed &&
		strings.EqualFold(strings.TrimSpace(observation.Action), "VALIDATE_ADMISSION_CANDIDATE") &&
		strings.EqualFold(strings.TrimSpace(observation.ValidationKind), "admission_candidate") &&
		strings.EqualFold(strings.TrimSpace(observation.Decision), "accepted")
}
