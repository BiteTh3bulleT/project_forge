package contextattribution

import (
	"strings"

	"forge/projectforge/services/core/internal/refvalidation"
)

const (
	GateWorkspace        = "workspace_present"
	GateQuery            = "query_present"
	GatePurpose          = "context_purpose_allowed"
	GateSourceRefs       = "source_refs_present"
	GateRefValidation    = "ref_shape_valid"
	GateSelectionReason  = "selection_reason_present"
	GateNoAuthorityClaim = "no_authority_claim"
)

var allowedContextPurposes = map[string]struct{}{
	"chat_turn":              {},
	"restore_preview":        {},
	"operator_inspection":    {},
	"admission_candidate":    {},
	"maintenance_proposal":   {},
	"diagnostic_attribution": {},
}

var forbiddenAuthorityClaims = map[string]struct{}{
	"compile_context":            {},
	"context_compilation":        {},
	"commit":                     {},
	"write_memory":               {},
	"semantic_memory_write":      {},
	"memory_mutation":            {},
	"admit_evidence":             {},
	"reject_evidence":            {},
	"call_model":                 {},
	"call_modelruntime":          {},
	"modelruntime_call":          {},
	"runtime_mutation":           {},
	"execute_tool":               {},
	"gateway_execution":          {},
	"run_retrieval":              {},
	"retrieval_execution":        {},
	"run_search":                 {},
	"search_execution":           {},
	"run_embeddings":             {},
	"embedding_execution":        {},
	"live_kv_reuse":              {},
	"simulator_authority":        {},
	"live_kernel_authority":      {},
	"forge_k_live_authority":     {},
	"live_authority_migration":   {},
	"context_compiler_authority": {},
}

type AttributionRequest struct {
	ResultID         string                    `json:"result_id"`
	WorkspaceID      string                    `json:"workspace_id"`
	Query            string                    `json:"query"`
	ContextPurpose   string                    `json:"context_purpose"`
	SourceRefs       []refvalidation.ObjectRef `json:"source_refs"`
	SelectionReasons map[string]string         `json:"selection_reasons"`
	Claims           map[string]bool           `json:"claims,omitempty"`
}

type ValidationFailure struct {
	Gate    string `json:"gate"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type AttributionResult struct {
	ResultID               string                    `json:"result_id"`
	WorkspaceID            string                    `json:"workspace_id"`
	Query                  string                    `json:"query"`
	ContextPurpose         string                    `json:"context_purpose"`
	Passed                 bool                      `json:"passed"`
	NormalizedSourceRefs   []refvalidation.ObjectRef `json:"normalized_source_refs"`
	NormalizedReasonKeys   []string                  `json:"normalized_reason_keys"`
	Failures               []ValidationFailure       `json:"failures,omitempty"`
	Warnings               []string                  `json:"warnings,omitempty"`
	ContextCompilation     bool                      `json:"context_compilation"`
	MemoryMutation         bool                      `json:"memory_mutation"`
	ModelRuntimeCall       bool                      `json:"model_runtime_call"`
	GatewayExecution       bool                      `json:"gateway_execution"`
	LiveAuthorityMigration bool                      `json:"live_authority_migration"`
}

func ValidateAttribution(req AttributionRequest) AttributionResult {
	resultID := strings.TrimSpace(req.ResultID)
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	query := strings.TrimSpace(req.Query)
	purpose := strings.ToLower(strings.TrimSpace(req.ContextPurpose))
	failures := make([]ValidationFailure, 0)

	if workspaceID == "" {
		failures = append(failures, ValidationFailure{Gate: GateWorkspace, Field: "workspace_id", Message: "workspace_id is required"})
	}
	if query == "" {
		failures = append(failures, ValidationFailure{Gate: GateQuery, Field: "query", Message: "query is required"})
	}
	if purpose == "" || !allowedContextPurpose(purpose) {
		failures = append(failures, ValidationFailure{Gate: GatePurpose, Field: "context_purpose", Message: "context_purpose is not allowed"})
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

	normalizedReasonKeys := make([]string, 0, len(sourceResult.NormalizedRefs))
	for _, ref := range sourceResult.NormalizedRefs {
		key := reasonKey(ref)
		reason := strings.TrimSpace(req.SelectionReasons[key])
		if reason == "" {
			failures = append(failures, ValidationFailure{Gate: GateSelectionReason, Field: "selection_reasons." + key, Message: "selection reason is required for each source ref"})
			continue
		}
		if reasonUnsafe(reason) {
			failures = append(failures, ValidationFailure{Gate: GateSelectionReason, Field: "selection_reasons." + key, Message: "selection reason contains unsafe content"})
			continue
		}
		normalizedReasonKeys = append(normalizedReasonKeys, key)
	}

	for claim, value := range req.Claims {
		if !value {
			continue
		}
		if forbiddenAuthorityClaim(claim) {
			failures = append(failures, ValidationFailure{Gate: GateNoAuthorityClaim, Field: "claims." + claim, Message: "context attribution validation cannot claim live authority or execute compilation"})
		}
	}

	return AttributionResult{
		ResultID:               resultID,
		WorkspaceID:            workspaceID,
		Query:                  query,
		ContextPurpose:         purpose,
		Passed:                 len(failures) == 0,
		NormalizedSourceRefs:   sourceResult.NormalizedRefs,
		NormalizedReasonKeys:   normalizedReasonKeys,
		Failures:               failures,
		ContextCompilation:     false,
		MemoryMutation:         false,
		ModelRuntimeCall:       false,
		GatewayExecution:       false,
		LiveAuthorityMigration: false,
	}
}

func allowedContextPurpose(purpose string) bool {
	_, ok := allowedContextPurposes[strings.ToLower(strings.TrimSpace(purpose))]
	return ok
}

func reasonKey(ref refvalidation.ObjectRef) string {
	return strings.ToLower(strings.TrimSpace(ref.RefType)) + ":" + strings.TrimSpace(ref.RefID)
}

func reasonUnsafe(reason string) bool {
	if len(reason) > 512 {
		return true
	}
	lower := strings.ToLower(reason)
	for _, term := range []string{"secret", "token", "password", "apikey", "api_key", "authorization", "cookie", "bearer"} {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}

func forbiddenAuthorityClaim(claim string) bool {
	_, ok := forbiddenAuthorityClaims[strings.ToLower(strings.TrimSpace(claim))]
	return ok
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
