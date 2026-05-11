package semanticvalidation

import (
	"strings"

	"forge/projectforge/services/core/internal/refvalidation"
)

const (
	GateWorkspace        = "workspace_present"
	GateOperationType    = "operation_type_allowed"
	GateSourceRefs       = "source_refs_present"
	GateRefValidation    = "ref_shape_valid"
	GateNoAuthorityClaim = "no_authority_claim"
)

var forbiddenAuthorityClaims = []string{
	"execute",
	"commit",
	"admit_evidence",
	"reject_evidence",
	"write_memory",
	"semantic_memory_write",
	"memory_mutation",
	"call_model",
	"call_modelruntime",
	"modelruntime_call",
	"modelruntime_mutation",
	"runtime_mutation",
	"load_model",
	"unload_model",
	"execute_tool",
	"gateway_execution",
	"run_retrieval",
	"retrieval_execution",
	"run_search",
	"search_execution",
	"run_embeddings",
	"embedding_execution",
	"compile_context",
	"context_compilation",
	"live_kv_reuse",
	"simulator_authority",
	"live_kernel_authority",
	"forge_k_live_authority",
	"live_authority_migration",
}

var forbiddenAuthorityClaimSet = buildForbiddenAuthorityClaimSet(forbiddenAuthorityClaims)

func ForbiddenAuthorityClaims() []string {
	out := make([]string, len(forbiddenAuthorityClaims))
	copy(out, forbiddenAuthorityClaims)
	return out
}

type OperationRequest struct {
	ResultID       string                    `json:"result_id"`
	WorkspaceID    string                    `json:"workspace_id"`
	OperationType  string                    `json:"operation_type"`
	SourceRefs     []refvalidation.ObjectRef `json:"source_refs"`
	DerivedRefs    []refvalidation.ObjectRef `json:"derived_refs,omitempty"`
	ProvenanceRefs []refvalidation.ObjectRef `json:"provenance_refs,omitempty"`
	Claims         map[string]bool           `json:"claims,omitempty"`
}

type ValidationFailure struct {
	Gate    string `json:"gate"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type OperationResult struct {
	ResultID                 string                    `json:"result_id"`
	WorkspaceID              string                    `json:"workspace_id"`
	OperationType            string                    `json:"operation_type"`
	Passed                   bool                      `json:"passed"`
	NormalizedSourceRefs     []refvalidation.ObjectRef `json:"normalized_source_refs"`
	NormalizedDerivedRefs    []refvalidation.ObjectRef `json:"normalized_derived_refs,omitempty"`
	NormalizedProvenanceRefs []refvalidation.ObjectRef `json:"normalized_provenance_refs,omitempty"`
	Failures                 []ValidationFailure       `json:"failures,omitempty"`
	Warnings                 []string                  `json:"warnings,omitempty"`
	MemoryMutation           bool                      `json:"memory_mutation"`
	ModelRuntimeCall         bool                      `json:"model_runtime_call"`
	EvidenceAdmission        bool                      `json:"evidence_admission"`
	ContextCompilation       bool                      `json:"context_compilation"`
	LiveAuthorityMigration   bool                      `json:"live_authority_migration"`
}

func ValidateOperation(req OperationRequest) OperationResult {
	resultID := strings.TrimSpace(req.ResultID)
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	operationType := strings.ToLower(strings.TrimSpace(req.OperationType))
	failures := make([]ValidationFailure, 0)

	if workspaceID == "" {
		failures = append(failures, ValidationFailure{Gate: GateWorkspace, Field: "workspace_id", Message: "workspace_id is required"})
	}
	if operationType == "" || !allowedOperationType(operationType) {
		failures = append(failures, ValidationFailure{Gate: GateOperationType, Field: "operation_type", Message: "operation_type is not allowed"})
	}
	if len(req.SourceRefs) == 0 {
		failures = append(failures, ValidationFailure{Gate: GateSourceRefs, Field: "source_refs", Message: "at least one source ref is required"})
	}

	sourceResult := refvalidation.ValidateRefs(refvalidation.ValidationRequest{
		ResultID:    resultID + ":source_refs",
		WorkspaceID: workspaceID,
		Refs:        req.SourceRefs,
	})
	if len(req.SourceRefs) > 0 && !sourceResult.Passed {
		failures = appendRefFailures(failures, "source_refs", sourceResult.Failures)
	}

	derivedResult := validateOptionalRefs(resultID+":derived_refs", workspaceID, req.DerivedRefs)
	if !derivedResult.Passed {
		failures = appendRefFailures(failures, "derived_refs", derivedResult.Failures)
	}

	provenanceResult := validateOptionalRefs(resultID+":provenance_refs", workspaceID, req.ProvenanceRefs)
	if !provenanceResult.Passed {
		failures = appendRefFailures(failures, "provenance_refs", provenanceResult.Failures)
	}

	for claim, value := range req.Claims {
		if !value {
			continue
		}
		if forbiddenAuthorityClaim(claim) {
			failures = append(failures, ValidationFailure{Gate: GateNoAuthorityClaim, Field: "claims." + claim, Message: "semantic operation validation cannot claim live authority"})
		}
	}

	return OperationResult{
		ResultID:                 resultID,
		WorkspaceID:              workspaceID,
		OperationType:            operationType,
		Passed:                   len(failures) == 0,
		NormalizedSourceRefs:     sourceResult.NormalizedRefs,
		NormalizedDerivedRefs:    derivedResult.NormalizedRefs,
		NormalizedProvenanceRefs: provenanceResult.NormalizedRefs,
		Failures:                 failures,
		MemoryMutation:           false,
		ModelRuntimeCall:         false,
		EvidenceAdmission:        false,
		ContextCompilation:       false,
		LiveAuthorityMigration:   false,
	}
}

func validateOptionalRefs(resultID, workspaceID string, refs []refvalidation.ObjectRef) refvalidation.ValidationResult {
	if len(refs) == 0 {
		return refvalidation.ValidationResult{ResultID: resultID, WorkspaceID: strings.TrimSpace(workspaceID), Passed: true}
	}
	return refvalidation.ValidateRefs(refvalidation.ValidationRequest{
		ResultID:    resultID,
		WorkspaceID: workspaceID,
		Refs:        refs,
	})
}

func appendRefFailures(out []ValidationFailure, fieldPrefix string, failures []refvalidation.ValidationFailure) []ValidationFailure {
	for _, failure := range failures {
		out = append(out, ValidationFailure{
			Gate:    GateRefValidation,
			Field:   fieldPrefix + "." + failure.Field,
			Message: failure.Message,
		})
	}
	return out
}

func allowedOperationType(operationType string) bool {
	switch strings.TrimSpace(operationType) {
	case "merge",
		"derive",
		"summarize",
		"classify",
		"link",
		"contradiction_check",
		"supersede",
		"route",
		"context_prepare":
		return true
	default:
		return false
	}
}

func forbiddenAuthorityClaim(claim string) bool {
	_, ok := forbiddenAuthorityClaimSet[normalizeClaimName(claim)]
	return ok
}

func buildForbiddenAuthorityClaimSet(claims []string) map[string]struct{} {
	out := make(map[string]struct{}, len(claims))
	for _, claim := range claims {
		out[normalizeClaimName(claim)] = struct{}{}
	}
	return out
}

func normalizeClaimName(claim string) string {
	return strings.ToLower(strings.TrimSpace(claim))
}
