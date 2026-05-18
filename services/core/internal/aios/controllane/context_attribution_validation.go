package controllane

import (
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/contextattribution"
)

const (
	ContextAttributionDecisionAccepted  = "accepted"
	ContextAttributionDecisionRejected  = "rejected"
	ContextAttributionDecisionMalformed = "malformed"

	ContextAttributionPolicyVersion    = "phase-19-context-attribution-v1"
	ContextAttributionValidatorVersion = "contextattribution-v1"
)

type ContextAttributionValidationDecision struct {
	Accepted               bool                                   `json:"accepted"`
	Decision               string                                 `json:"decision"`
	Reason                 string                                 `json:"reason"`
	Source                 string                                 `json:"source"`
	ContextPurpose         string                                 `json:"contextPurpose"`
	NormalizedSourceRefs   []map[string]string                    `json:"normalizedSourceRefs,omitempty"`
	NormalizedReasonKeys   []string                               `json:"normalizedReasonKeys,omitempty"`
	Failures               []contextattribution.ValidationFailure `json:"failures,omitempty"`
	Warnings               []string                               `json:"warnings,omitempty"`
	ContextCompilation     bool                                   `json:"contextCompilation"`
	MemoryMutation         bool                                   `json:"memoryMutation"`
	ModelRuntimeCall       bool                                   `json:"modelRuntimeCall"`
	GatewayExecution       bool                                   `json:"gatewayExecution"`
	LiveAuthorityMigration bool                                   `json:"liveAuthorityMigration"`
	ValidatorVersion       string                                 `json:"validatorVersion"`
	PolicyVersion          string                                 `json:"policyVersion"`
	ValidationResult       contextattribution.AttributionResult   `json:"validationResult"`
}

func EnforceContextAttribution(req domain.SyscallRequest) ContextAttributionValidationDecision {
	base := ContextAttributionValidationDecision{
		Source:                 string(req.Source),
		ContextCompilation:     false,
		MemoryMutation:         false,
		ModelRuntimeCall:       false,
		GatewayExecution:       false,
		LiveAuthorityMigration: false,
		ValidatorVersion:       ContextAttributionValidatorVersion,
		PolicyVersion:          ContextAttributionPolicyVersion,
	}
	if issues := validateContextAttribution(req); len(issues) > 0 {
		base.Accepted = false
		base.Decision = ContextAttributionDecisionMalformed
		base.Reason = issues[0].Message
		return base
	}
	result := contextattribution.ValidateAttribution(contextAttributionRequestFromPayload(req))
	base.ValidationResult = result
	base.ContextPurpose = result.ContextPurpose
	base.Failures = append([]contextattribution.ValidationFailure{}, result.Failures...)
	base.Warnings = append([]string{}, result.Warnings...)
	base.NormalizedSourceRefs = refsForSummary(result.NormalizedSourceRefs)
	base.NormalizedReasonKeys = append([]string{}, result.NormalizedReasonKeys...)
	if result.Passed {
		base.Accepted = true
		base.Decision = ContextAttributionDecisionAccepted
		base.Reason = "context attribution validation accepted"
		return base
	}
	base.Accepted = false
	base.Decision = ContextAttributionDecisionRejected
	base.Reason = "context attribution validation rejected"
	if len(result.Failures) > 0 {
		base.Reason = result.Failures[0].Message
	}
	return base
}

func contextAttributionRequestFromPayload(req domain.SyscallRequest) contextattribution.AttributionRequest {
	return contextattribution.AttributionRequest{
		ResultID:         req.ID + ":context_attribution_validation",
		WorkspaceID:      firstNonEmpty(readString(req.Payload, "workspace_id"), req.Scope.WorkspaceID),
		Query:            readString(req.Payload, "query"),
		ContextPurpose:   readString(req.Payload, "context_purpose"),
		SourceRefs:       refObjectsFromPayload(req.Payload["source_refs"]),
		SelectionReasons: selectionReasonsFromPayload(req.Payload["selection_reasons"]),
		Claims:           boolClaimsFromPayload(req.Payload["claims"]),
	}
}

func selectionReasonsFromPayload(raw any) map[string]string {
	items, ok := raw.(map[string]any)
	if !ok || len(items) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(items))
	for key, value := range items {
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(fmt.Sprintf("%v", value))
	}
	return out
}

func (d ContextAttributionValidationDecision) ToStateSummary() map[string]any {
	return map[string]any{
		"contextAttributionValidation": d.ToAuditFields(),
		"contextAttributionResult":     d.ValidationResult,
		"passed":                       d.Accepted,
		"contextPurpose":               d.ContextPurpose,
		"normalizedRefs":               d.NormalizedSourceRefs,
		"normalizedSourceRefs":         d.NormalizedSourceRefs,
		"normalizedReasonKeys":         append([]string{}, d.NormalizedReasonKeys...),
		"failures":                     append([]contextattribution.ValidationFailure{}, d.Failures...),
		"warnings":                     append([]string{}, d.Warnings...),
		"contextCompilation":           d.ContextCompilation,
		"memoryMutation":               d.MemoryMutation,
		"modelRuntimeCall":             d.ModelRuntimeCall,
		"gatewayExecution":             d.GatewayExecution,
		"liveAuthorityMigration":       d.LiveAuthorityMigration,
		"forgeKActivation":             forgeKActivationSummary(string(domain.ActionValidateContextAttribution)),
		"forgeKNoEffect":               forgeKNoEffectSummary(),
	}
}

func (d ContextAttributionValidationDecision) ToAuditFields() map[string]any {
	return map[string]any{
		"accepted":                 d.Accepted,
		"decision":                 d.Decision,
		"reason":                   d.Reason,
		"source":                   d.Source,
		"contextPurpose":           d.ContextPurpose,
		"normalizedSourceRefCount": len(d.NormalizedSourceRefs),
		"normalizedReasonKeyCount": len(d.NormalizedReasonKeys),
		"failures":                 append([]contextattribution.ValidationFailure{}, d.Failures...),
		"warnings":                 append([]string{}, d.Warnings...),
		"failureCount":             len(d.Failures),
		"warningCount":             len(d.Warnings),
		"contextCompilation":       d.ContextCompilation,
		"memoryMutation":           d.MemoryMutation,
		"modelRuntimeCall":         d.ModelRuntimeCall,
		"gatewayExecution":         d.GatewayExecution,
		"liveAuthorityMigration":   d.LiveAuthorityMigration,
		"validatorVersion":         d.ValidatorVersion,
		"policyVersion":            d.PolicyVersion,
		"forgeKActivation":         forgeKActivationSummary(string(domain.ActionValidateContextAttribution)),
		"forgeKNoEffect":           forgeKNoEffectSummary(),
	}
}

func (d ContextAttributionValidationDecision) ToSyscallError() domain.SyscallError {
	return domain.SyscallError{Code: domain.ErrInvalidPayload, Field: "payload.context_attribution", Message: d.Reason}
}
