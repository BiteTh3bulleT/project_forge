package controllane

import (
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/semanticvalidation"
)

const (
	SemanticOperationDecisionAccepted  = "accepted"
	SemanticOperationDecisionRejected  = "rejected"
	SemanticOperationDecisionMalformed = "malformed"

	SemanticOperationPolicyVersion    = "phase-14c-v1"
	SemanticOperationValidatorVersion = "semanticvalidation-v1"
)

type SemanticOperationValidationDecision struct {
	Accepted               bool                                   `json:"accepted"`
	Decision               string                                 `json:"decision"`
	Reason                 string                                 `json:"reason"`
	Source                 string                                 `json:"source"`
	OperationType          string                                 `json:"operationType"`
	Failures               []semanticvalidation.ValidationFailure `json:"failures,omitempty"`
	Warnings               []string                               `json:"warnings,omitempty"`
	MemoryMutation         bool                                   `json:"memoryMutation"`
	ModelRuntimeCall       bool                                   `json:"modelRuntimeCall"`
	EvidenceAdmission      bool                                   `json:"evidenceAdmission"`
	ContextCompilation     bool                                   `json:"contextCompilation"`
	LiveAuthorityMigration bool                                   `json:"liveAuthorityMigration"`
	ValidatorVersion       string                                 `json:"validatorVersion"`
	PolicyVersion          string                                 `json:"policyVersion"`
	ValidationResult       semanticvalidation.OperationResult     `json:"validationResult"`
}

func EnforceSemanticOperation(req domain.SyscallRequest) SemanticOperationValidationDecision {
	base := SemanticOperationValidationDecision{
		Source:                 string(req.Source),
		MemoryMutation:         false,
		ModelRuntimeCall:       false,
		EvidenceAdmission:      false,
		ContextCompilation:     false,
		LiveAuthorityMigration: false,
		ValidatorVersion:       SemanticOperationValidatorVersion,
		PolicyVersion:          SemanticOperationPolicyVersion,
	}
	if issues := validateSemanticOperation(req); len(issues) > 0 {
		base.Accepted = false
		base.Decision = SemanticOperationDecisionMalformed
		base.Reason = issues[0].Message
		return base
	}
	result := semanticvalidation.ValidateOperation(semanticOperationRequestFromPayload(req))
	base.ValidationResult = result
	base.OperationType = result.OperationType
	base.Failures = append([]semanticvalidation.ValidationFailure{}, result.Failures...)
	base.Warnings = append([]string{}, result.Warnings...)
	if result.Passed {
		base.Accepted = true
		base.Decision = SemanticOperationDecisionAccepted
		base.Reason = "semantic operation validation accepted"
		return base
	}
	base.Accepted = false
	base.Decision = SemanticOperationDecisionRejected
	base.Reason = "semantic operation validation rejected"
	if len(result.Failures) > 0 {
		base.Reason = result.Failures[0].Message
	}
	return base
}

func validateSemanticOperation(req domain.SyscallRequest) []domain.SyscallError {
	if strings.TrimSpace(readString(req.Payload, "workspace_id")) == "" {
		return []domain.SyscallError{errField(domain.ErrMissingRequiredField, "payload.workspace_id", "workspace_id is required")}
	}
	if strings.TrimSpace(readString(req.Payload, "operation_type")) == "" {
		return []domain.SyscallError{errField(domain.ErrMissingRequiredField, "payload.operation_type", "operation_type is required")}
	}
	if _, ok := req.Payload["source_refs"]; !ok {
		return []domain.SyscallError{errField(domain.ErrMissingRequiredField, "payload.source_refs", "source_refs are required")}
	}
	return nil
}

func semanticOperationRequestFromPayload(req domain.SyscallRequest) semanticvalidation.OperationRequest {
	return semanticvalidation.OperationRequest{
		ResultID:       req.ID + ":semantic_operation_validation",
		WorkspaceID:    readString(req.Payload, "workspace_id"),
		OperationType:  readString(req.Payload, "operation_type"),
		SourceRefs:     refObjectsFromPayload(req.Payload["source_refs"]),
		DerivedRefs:    refObjectsFromPayload(req.Payload["derived_refs"]),
		ProvenanceRefs: refObjectsFromPayload(req.Payload["provenance_refs"]),
		Claims:         boolClaimsFromPayload(req.Payload["claims"]),
	}
}

func boolClaimsFromPayload(raw any) map[string]bool {
	claims := map[string]bool{}
	m, ok := raw.(map[string]any)
	if !ok {
		return claims
	}
	for key, value := range m {
		switch v := value.(type) {
		case bool:
			claims[strings.ToLower(strings.TrimSpace(key))] = v
		case string:
			claims[strings.ToLower(strings.TrimSpace(key))] = strings.EqualFold(strings.TrimSpace(v), "true") || strings.TrimSpace(v) == "1"
		}
	}
	return claims
}

func (d SemanticOperationValidationDecision) ToStateSummary() map[string]any {
	return map[string]any{
		"semanticOperationValidation": d.ToAuditFields(),
		"semanticOperationResult":     d.ValidationResult,
		"passed":                      d.Accepted,
		"operationType":               d.OperationType,
		"failures":                    append([]semanticvalidation.ValidationFailure{}, d.Failures...),
		"warnings":                    append([]string{}, d.Warnings...),
		"memoryMutation":              d.MemoryMutation,
		"modelRuntimeCall":            d.ModelRuntimeCall,
		"evidenceAdmission":           d.EvidenceAdmission,
		"contextCompilation":          d.ContextCompilation,
		"liveAuthorityMigration":      d.LiveAuthorityMigration,
		"forgeKActivation":            forgeKActivationSummary(string(domain.ActionValidateSemanticOperation)),
		"forgeKNoEffect":              forgeKNoEffectSummary(),
	}
}

func (d SemanticOperationValidationDecision) ToAuditFields() map[string]any {
	return map[string]any{
		"accepted":               d.Accepted,
		"decision":               d.Decision,
		"reason":                 d.Reason,
		"source":                 d.Source,
		"operationType":          d.OperationType,
		"failures":               append([]semanticvalidation.ValidationFailure{}, d.Failures...),
		"warnings":               append([]string{}, d.Warnings...),
		"failureCount":           len(d.Failures),
		"warningCount":           len(d.Warnings),
		"memoryMutation":         d.MemoryMutation,
		"modelRuntimeCall":       d.ModelRuntimeCall,
		"evidenceAdmission":      d.EvidenceAdmission,
		"contextCompilation":     d.ContextCompilation,
		"liveAuthorityMigration": d.LiveAuthorityMigration,
		"validatorVersion":       d.ValidatorVersion,
		"policyVersion":          d.PolicyVersion,
		"forgeKActivation":       forgeKActivationSummary(string(domain.ActionValidateSemanticOperation)),
		"forgeKNoEffect":         forgeKNoEffectSummary(),
	}
}

func (d SemanticOperationValidationDecision) ToSyscallError() domain.SyscallError {
	return domain.SyscallError{Code: domain.ErrInvalidPayload, Field: "payload.operation_type", Message: d.Reason}
}
