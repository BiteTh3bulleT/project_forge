package controllane

import (
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/refvalidation"
)

const (
	RefShapeDecisionAccepted  = "accepted"
	RefShapeDecisionRejected  = "rejected"
	RefShapeDecisionMalformed = "malformed"

	RefShapePolicyVersion    = "phase-14b-v1"
	RefShapeValidatorVersion = "refvalidation-v1"
)

type RefShapeValidationDecision struct {
	Accepted               bool                              `json:"accepted"`
	Decision               string                            `json:"decision"`
	Reason                 string                            `json:"reason"`
	Source                 string                            `json:"source"`
	Failures               []refvalidation.ValidationFailure `json:"failures,omitempty"`
	Warnings               []string                          `json:"warnings,omitempty"`
	NormalizedRefs         []refvalidation.ObjectRef         `json:"normalizedRefs,omitempty"`
	MemoryMutation         bool                              `json:"memoryMutation"`
	RuntimeMutation        bool                              `json:"runtimeMutation"`
	LiveAuthorityMigration bool                              `json:"liveAuthorityMigration"`
	ValidatorVersion       string                            `json:"validatorVersion"`
	PolicyVersion          string                            `json:"policyVersion"`
	ValidationResult       refvalidation.ValidationResult    `json:"validationResult"`
}

func EnforceRefShape(req domain.SyscallRequest) RefShapeValidationDecision {
	base := RefShapeValidationDecision{
		Source:                 string(req.Source),
		MemoryMutation:         false,
		RuntimeMutation:        false,
		LiveAuthorityMigration: false,
		ValidatorVersion:       RefShapeValidatorVersion,
		PolicyVersion:          RefShapePolicyVersion,
	}
	if issues := validateRefShape(req); len(issues) > 0 {
		base.Accepted = false
		base.Decision = RefShapeDecisionMalformed
		base.Reason = issues[0].Message
		return base
	}
	result := refvalidation.ValidateRefs(refValidationRequestFromPayload(req))
	base.ValidationResult = result
	base.Failures = append([]refvalidation.ValidationFailure{}, result.Failures...)
	base.Warnings = append([]string{}, result.Warnings...)
	base.NormalizedRefs = append([]refvalidation.ObjectRef{}, result.NormalizedRefs...)
	if result.Passed {
		base.Accepted = true
		base.Decision = RefShapeDecisionAccepted
		base.Reason = "ref shape validation accepted"
		return base
	}
	base.Accepted = false
	base.Decision = RefShapeDecisionRejected
	base.Reason = "ref shape validation rejected"
	if len(result.Failures) > 0 {
		base.Reason = result.Failures[0].Message
	}
	return base
}

func validateRefShape(req domain.SyscallRequest) []domain.SyscallError {
	if strings.TrimSpace(readString(req.Payload, "workspace_id")) == "" {
		return []domain.SyscallError{errField(domain.ErrMissingRequiredField, "payload.workspace_id", "workspace_id is required")}
	}
	if _, ok := req.Payload["refs"]; !ok {
		return []domain.SyscallError{errField(domain.ErrMissingRequiredField, "payload.refs", "refs are required")}
	}
	return nil
}

func refValidationRequestFromPayload(req domain.SyscallRequest) refvalidation.ValidationRequest {
	return refvalidation.ValidationRequest{
		ResultID:    req.ID + ":ref_shape_validation",
		WorkspaceID: readString(req.Payload, "workspace_id"),
		Refs:        refObjectsFromPayload(req.Payload["refs"]),
	}
}

func refObjectsFromPayload(raw any) []refvalidation.ObjectRef {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]refvalidation.ObjectRef, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, refvalidation.ObjectRef{
			RefType:     readString(m, "ref_type"),
			RefID:       readString(m, "ref_id"),
			WorkspaceID: readString(m, "workspace_id"),
			SourceRef:   readString(m, "source_ref"),
		})
	}
	return out
}

func (d RefShapeValidationDecision) ToStateSummary() map[string]any {
	return map[string]any{
		"refShapeValidation":     d.ToAuditFields(),
		"refShapeResult":         d.ValidationResult,
		"passed":                 d.Accepted,
		"normalizedRefs":         refsForSummary(d.NormalizedRefs),
		"failures":               append([]refvalidation.ValidationFailure{}, d.Failures...),
		"warnings":               append([]string{}, d.Warnings...),
		"memoryMutation":         d.MemoryMutation,
		"runtimeMutation":        d.RuntimeMutation,
		"liveAuthorityMigration": d.LiveAuthorityMigration,
	}
}

func (d RefShapeValidationDecision) ToAuditFields() map[string]any {
	return map[string]any{
		"accepted":               d.Accepted,
		"decision":               d.Decision,
		"reason":                 d.Reason,
		"source":                 d.Source,
		"normalizedRefCount":     len(d.NormalizedRefs),
		"failures":               append([]refvalidation.ValidationFailure{}, d.Failures...),
		"warnings":               append([]string{}, d.Warnings...),
		"memoryMutation":         d.MemoryMutation,
		"runtimeMutation":        d.RuntimeMutation,
		"liveAuthorityMigration": d.LiveAuthorityMigration,
		"validatorVersion":       d.ValidatorVersion,
		"policyVersion":          d.PolicyVersion,
	}
}

func (d RefShapeValidationDecision) ToSyscallError() domain.SyscallError {
	return domain.SyscallError{Code: domain.ErrInvalidPayload, Field: "payload.refs", Message: d.Reason}
}

func refsForSummary(refs []refvalidation.ObjectRef) []map[string]string {
	out := make([]map[string]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, map[string]string{
			"ref_type":     ref.RefType,
			"ref_id":       ref.RefID,
			"workspace_id": ref.WorkspaceID,
		})
	}
	return out
}
