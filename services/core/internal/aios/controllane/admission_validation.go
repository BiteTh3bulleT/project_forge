package controllane

import (
	"strings"

	"forge/projectforge/services/core/internal/admissionvalidation"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/refvalidation"
)

const (
	AdmissionDecisionAccepted  = "accepted"
	AdmissionDecisionRejected  = "rejected"
	AdmissionDecisionMalformed = "malformed"

	AdmissionPolicyVersion    = "phase-06-admission-only-v1"
	AdmissionValidatorVersion = "admissionvalidation-v1"
)

type AdmissionValidationDecision struct {
	Accepted                 bool                                    `json:"accepted"`
	Decision                 string                                  `json:"decision"`
	Reason                   string                                  `json:"reason"`
	Source                   string                                  `json:"source"`
	AdmissionMode            string                                  `json:"admissionMode"`
	CaseID                   string                                  `json:"caseId"`
	Failures                 []admissionvalidation.ValidationFailure `json:"failures,omitempty"`
	Warnings                 []string                                `json:"warnings,omitempty"`
	NormalizedEvidenceRefs   []refvalidation.ObjectRef               `json:"normalizedEvidenceRefs,omitempty"`
	NormalizedSourceRefs     []refvalidation.ObjectRef               `json:"normalizedSourceRefs,omitempty"`
	NormalizedPolicyRefs     []refvalidation.ObjectRef               `json:"normalizedPolicyRefs,omitempty"`
	NormalizedProvenanceRefs []refvalidation.ObjectRef               `json:"normalizedProvenanceRefs,omitempty"`
	CanonicalCommit          bool                                    `json:"canonicalCommit"`
	MemoryMutation           bool                                    `json:"memoryMutation"`
	RuntimeMutation          bool                                    `json:"runtimeMutation"`
	ModelRuntimeCall         bool                                    `json:"modelRuntimeCall"`
	GatewayExecution         bool                                    `json:"gatewayExecution"`
	EvidenceAdmission        bool                                    `json:"evidenceAdmission"`
	ContextCompilation       bool                                    `json:"contextCompilation"`
	LiveAuthorityMigration   bool                                    `json:"liveAuthorityMigration"`
	ValidatorVersion         string                                  `json:"validatorVersion"`
	PolicyVersion            string                                  `json:"policyVersion"`
	ValidationResult         admissionvalidation.AdmissionResult     `json:"validationResult"`
}

func EnforceAdmissionCandidate(req domain.SyscallRequest) AdmissionValidationDecision {
	base := AdmissionValidationDecision{
		Source:                 string(req.Source),
		CanonicalCommit:        false,
		MemoryMutation:         false,
		RuntimeMutation:        false,
		ModelRuntimeCall:       false,
		GatewayExecution:       false,
		EvidenceAdmission:      false,
		ContextCompilation:     false,
		LiveAuthorityMigration: false,
		ValidatorVersion:       AdmissionValidatorVersion,
		PolicyVersion:          AdmissionPolicyVersion,
	}
	if issues := validateAdmissionCandidate(req); len(issues) > 0 {
		base.Accepted = false
		base.Decision = AdmissionDecisionMalformed
		base.Reason = issues[0].Message
		return base
	}
	result := admissionvalidation.ValidateAdmission(admissionRequestFromPayload(req))
	base.ValidationResult = result
	base.AdmissionMode = result.AdmissionMode
	base.CaseID = result.CaseID
	base.Failures = append([]admissionvalidation.ValidationFailure{}, result.Failures...)
	base.Warnings = append([]string{}, result.Warnings...)
	base.NormalizedEvidenceRefs = append([]refvalidation.ObjectRef{}, result.NormalizedEvidenceRefs...)
	base.NormalizedSourceRefs = append([]refvalidation.ObjectRef{}, result.NormalizedSourceRefs...)
	base.NormalizedPolicyRefs = append([]refvalidation.ObjectRef{}, result.NormalizedPolicyRefs...)
	base.NormalizedProvenanceRefs = append([]refvalidation.ObjectRef{}, result.NormalizedProvenanceRefs...)
	if result.Passed {
		base.Accepted = true
		base.Decision = AdmissionDecisionAccepted
		base.Reason = "admission candidate validation accepted"
		return base
	}
	base.Accepted = false
	base.Decision = AdmissionDecisionRejected
	base.Reason = "admission candidate validation rejected"
	if len(result.Failures) > 0 {
		base.Reason = result.Failures[0].Message
	}
	return base
}

func admissionRequestFromPayload(req domain.SyscallRequest) admissionvalidation.AdmissionRequest {
	return admissionvalidation.AdmissionRequest{
		ResultID:       req.ID + ":admission_candidate_validation",
		WorkspaceID:    firstNonEmpty(readString(req.Payload, "workspace_id"), req.Scope.WorkspaceID),
		CaseID:         readString(req.Payload, "case_id"),
		AdmissionMode:  readString(req.Payload, "admission_mode"),
		EvidenceRefs:   refObjectsFromPayload(req.Payload["evidence_refs"]),
		SourceRefs:     refObjectsFromPayload(req.Payload["source_refs"]),
		PolicyRefs:     refObjectsFromPayload(req.Payload["policy_refs"]),
		ProvenanceRefs: refObjectsFromPayload(req.Payload["provenance_refs"]),
		Claims:         admissionBoolClaimsFromPayload(req.Payload["claims"]),
	}
}

func admissionBoolClaimsFromPayload(raw any) map[string]bool {
	items, ok := raw.(map[string]any)
	if !ok || len(items) == 0 {
		return nil
	}
	out := make(map[string]bool, len(items))
	for key, value := range items {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		switch v := value.(type) {
		case bool:
			out[key] = v
		case string:
			out[key] = strings.EqualFold(strings.TrimSpace(v), "true") || strings.TrimSpace(v) == "1"
		}
	}
	return out
}

func (d AdmissionValidationDecision) ToStateSummary() map[string]any {
	return map[string]any{
		"admissionCandidateValidation": d.ToAuditFields(),
		"admissionCandidateResult":     d.ValidationResult,
		"passed":                       d.Accepted,
		"caseId":                       d.CaseID,
		"admissionMode":                d.AdmissionMode,
		"normalizedRefs":               refsForSummary(d.NormalizedEvidenceRefs),
		"normalizedEvidenceRefs":       refsForSummary(d.NormalizedEvidenceRefs),
		"normalizedSourceRefs":         refsForSummary(d.NormalizedSourceRefs),
		"normalizedPolicyRefs":         refsForSummary(d.NormalizedPolicyRefs),
		"normalizedProvenanceRefs":     refsForSummary(d.NormalizedProvenanceRefs),
		"failures":                     append([]admissionvalidation.ValidationFailure{}, d.Failures...),
		"warnings":                     append([]string{}, d.Warnings...),
		"canonicalCommit":              d.CanonicalCommit,
		"memoryMutation":               d.MemoryMutation,
		"runtimeMutation":              d.RuntimeMutation,
		"modelRuntimeCall":             d.ModelRuntimeCall,
		"gatewayExecution":             d.GatewayExecution,
		"evidenceAdmission":            d.EvidenceAdmission,
		"contextCompilation":           d.ContextCompilation,
		"liveAuthorityMigration":       d.LiveAuthorityMigration,
		"forgeKActivation":             forgeKActivationSummary(string(domain.ActionValidateAdmissionCandidate)),
		"forgeKNoEffect":               forgeKNoEffectSummary(),
	}
}

func (d AdmissionValidationDecision) ToAuditFields() map[string]any {
	return map[string]any{
		"accepted":                     d.Accepted,
		"decision":                     d.Decision,
		"reason":                       d.Reason,
		"source":                       d.Source,
		"caseId":                       d.CaseID,
		"admissionMode":                d.AdmissionMode,
		"normalizedEvidenceRefCount":   len(d.NormalizedEvidenceRefs),
		"normalizedSourceRefCount":     len(d.NormalizedSourceRefs),
		"normalizedPolicyRefCount":     len(d.NormalizedPolicyRefs),
		"normalizedProvenanceRefCount": len(d.NormalizedProvenanceRefs),
		"failures":                     append([]admissionvalidation.ValidationFailure{}, d.Failures...),
		"warnings":                     append([]string{}, d.Warnings...),
		"failureCount":                 len(d.Failures),
		"warningCount":                 len(d.Warnings),
		"canonicalCommit":              d.CanonicalCommit,
		"memoryMutation":               d.MemoryMutation,
		"runtimeMutation":              d.RuntimeMutation,
		"modelRuntimeCall":             d.ModelRuntimeCall,
		"gatewayExecution":             d.GatewayExecution,
		"evidenceAdmission":            d.EvidenceAdmission,
		"contextCompilation":           d.ContextCompilation,
		"liveAuthorityMigration":       d.LiveAuthorityMigration,
		"validatorVersion":             d.ValidatorVersion,
		"policyVersion":                d.PolicyVersion,
		"forgeKActivation":             forgeKActivationSummary(string(domain.ActionValidateAdmissionCandidate)),
		"forgeKNoEffect":               forgeKNoEffectSummary(),
	}
}

func (d AdmissionValidationDecision) ToSyscallError() domain.SyscallError {
	return domain.SyscallError{Code: domain.ErrInvalidPayload, Field: "payload.admission_candidate", Message: d.Reason}
}
