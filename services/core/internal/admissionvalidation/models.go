package admissionvalidation

import (
	"strings"

	"forge/projectforge/services/core/internal/refvalidation"
)

const (
	GateWorkspace        = "workspace_present"
	GateCase             = "case_present"
	GateAdmissionMode    = "admission_mode_allowed"
	GateEvidenceRefs     = "evidence_refs_present"
	GateRefValidation    = "ref_shape_valid"
	GateNoAuthorityClaim = "no_authority_claim"
)

var allowedAdmissionModes = []string{
	"admission_candidate",
	"admission_shadow",
	"admission_only",
	"rejection_candidate",
}

var allowedAdmissionModeSet = buildAllowedAdmissionModeSet(allowedAdmissionModes)

var forbiddenAuthorityClaims = []string{
	"commit",
	"canonical_commit",
	"write_memory",
	"memory_mutation",
	"admit_evidence",
	"reject_evidence",
	"admission_decision",
	"call_model",
	"call_modelruntime",
	"execute_tool",
	"gateway_execution",
	"run_retrieval",
	"run_search",
	"run_embeddings",
	"compile_context",
	"context_compilation",
	"live_kv_reuse",
	"simulator_authority",
	"live_kernel_authority",
	"forge_k_live_authority",
	"live_authority_migration",
}

var forbiddenAuthorityClaimSet = buildForbiddenAuthorityClaimSet(forbiddenAuthorityClaims)

type AdmissionRequest struct {
	ResultID       string                    `json:"result_id"`
	WorkspaceID    string                    `json:"workspace_id"`
	CaseID         string                    `json:"case_id"`
	AdmissionMode  string                    `json:"admission_mode"`
	EvidenceRefs   []refvalidation.ObjectRef `json:"evidence_refs"`
	SourceRefs     []refvalidation.ObjectRef `json:"source_refs,omitempty"`
	PolicyRefs     []refvalidation.ObjectRef `json:"policy_refs,omitempty"`
	ProvenanceRefs []refvalidation.ObjectRef `json:"provenance_refs,omitempty"`
	Claims         map[string]bool           `json:"claims,omitempty"`
}

type ValidationFailure struct {
	Gate    string `json:"gate"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type AdmissionResult struct {
	ResultID                 string                    `json:"result_id"`
	WorkspaceID              string                    `json:"workspace_id"`
	CaseID                   string                    `json:"case_id"`
	AdmissionMode            string                    `json:"admission_mode"`
	Passed                   bool                      `json:"passed"`
	NormalizedEvidenceRefs   []refvalidation.ObjectRef `json:"normalized_evidence_refs"`
	NormalizedSourceRefs     []refvalidation.ObjectRef `json:"normalized_source_refs,omitempty"`
	NormalizedPolicyRefs     []refvalidation.ObjectRef `json:"normalized_policy_refs,omitempty"`
	NormalizedProvenanceRefs []refvalidation.ObjectRef `json:"normalized_provenance_refs,omitempty"`
	Failures                 []ValidationFailure       `json:"failures,omitempty"`
	Warnings                 []string                  `json:"warnings,omitempty"`
	CanonicalCommit          bool                      `json:"canonical_commit"`
	MemoryMutation           bool                      `json:"memory_mutation"`
	ModelRuntimeCall         bool                      `json:"model_runtime_call"`
	GatewayExecution         bool                      `json:"gateway_execution"`
	ContextCompilation       bool                      `json:"context_compilation"`
	LiveAuthorityMigration   bool                      `json:"live_authority_migration"`
}

func AllowedAdmissionModes() []string {
	out := make([]string, len(allowedAdmissionModes))
	copy(out, allowedAdmissionModes)
	return out
}

func ForbiddenAuthorityClaims() []string {
	out := make([]string, len(forbiddenAuthorityClaims))
	copy(out, forbiddenAuthorityClaims)
	return out
}

func ValidateAdmission(req AdmissionRequest) AdmissionResult {
	resultID := strings.TrimSpace(req.ResultID)
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	caseID := strings.TrimSpace(req.CaseID)
	admissionMode := strings.ToLower(strings.TrimSpace(req.AdmissionMode))
	failures := make([]ValidationFailure, 0)

	if workspaceID == "" {
		failures = append(failures, ValidationFailure{Gate: GateWorkspace, Field: "workspace_id", Message: "workspace_id is required"})
	}
	if caseID == "" {
		failures = append(failures, ValidationFailure{Gate: GateCase, Field: "case_id", Message: "case_id is required"})
	}
	if admissionMode == "" || !allowedAdmissionMode(admissionMode) {
		failures = append(failures, ValidationFailure{Gate: GateAdmissionMode, Field: "admission_mode", Message: "admission_mode is not allowed"})
	}
	if len(req.EvidenceRefs) == 0 {
		failures = append(failures, ValidationFailure{Gate: GateEvidenceRefs, Field: "evidence_refs", Message: "at least one evidence ref is required"})
	}

	evidenceResult := refvalidation.ValidateRefs(refvalidation.ValidationRequest{
		ResultID:    resultID + ":evidence_refs",
		WorkspaceID: workspaceID,
		Refs:        req.EvidenceRefs,
	})
	if len(req.EvidenceRefs) > 0 && !evidenceResult.Passed {
		failures = appendRefFailures(failures, "evidence_refs", evidenceResult.Failures)
	}

	sourceResult := validateOptionalRefs(resultID+":source_refs", workspaceID, req.SourceRefs)
	if !sourceResult.Passed {
		failures = appendRefFailures(failures, "source_refs", sourceResult.Failures)
	}

	policyResult := validateOptionalRefs(resultID+":policy_refs", workspaceID, req.PolicyRefs)
	if !policyResult.Passed {
		failures = appendRefFailures(failures, "policy_refs", policyResult.Failures)
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
			failures = append(failures, ValidationFailure{Gate: GateNoAuthorityClaim, Field: "claims." + claim, Message: "admission validation cannot claim live authority"})
		}
	}

	return AdmissionResult{
		ResultID:                 resultID,
		WorkspaceID:              workspaceID,
		CaseID:                   caseID,
		AdmissionMode:            admissionMode,
		Passed:                   len(failures) == 0,
		NormalizedEvidenceRefs:   evidenceResult.NormalizedRefs,
		NormalizedSourceRefs:     sourceResult.NormalizedRefs,
		NormalizedPolicyRefs:     policyResult.NormalizedRefs,
		NormalizedProvenanceRefs: provenanceResult.NormalizedRefs,
		Failures:                 failures,
		CanonicalCommit:          false,
		MemoryMutation:           false,
		ModelRuntimeCall:         false,
		GatewayExecution:         false,
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

func allowedAdmissionMode(mode string) bool {
	_, ok := allowedAdmissionModeSet[strings.ToLower(strings.TrimSpace(mode))]
	return ok
}

func buildAllowedAdmissionModeSet(modes []string) map[string]struct{} {
	out := make(map[string]struct{}, len(modes))
	for _, mode := range modes {
		normalized := strings.ToLower(strings.TrimSpace(mode))
		if normalized == "" {
			continue
		}
		out[normalized] = struct{}{}
	}
	return out
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
