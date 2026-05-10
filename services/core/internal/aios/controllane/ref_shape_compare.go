package controllane

import (
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/refvalidation"
)

const (
	RefShapeCompareDecisionMatch     = "match"
	RefShapeCompareDecisionDrift     = "drift"
	RefShapeCompareDecisionMalformed = "malformed"
	RefShapeCompareDecisionRejected  = "rejected"

	RefShapeComparePolicyVersion    = "phase-14c-v1"
	RefShapeCompareValidatorVersion = "refvalidation-compare-v1"
)

type RefShapeComparisonDecision struct {
	Accepted               bool                              `json:"accepted"`
	Decision               string                            `json:"decision"`
	Reason                 string                            `json:"reason"`
	Source                 string                            `json:"source"`
	Match                  bool                              `json:"match"`
	AddedRefs              []refvalidation.ObjectRef         `json:"addedRefs,omitempty"`
	RemovedRefs            []refvalidation.ObjectRef         `json:"removedRefs,omitempty"`
	UnchangedRefs          []refvalidation.ObjectRef         `json:"unchangedRefs,omitempty"`
	Failures               []refvalidation.ValidationFailure `json:"failures,omitempty"`
	Warnings               []string                          `json:"warnings,omitempty"`
	MemoryMutation         bool                              `json:"memoryMutation"`
	RuntimeMutation        bool                              `json:"runtimeMutation"`
	LiveAuthorityMigration bool                              `json:"liveAuthorityMigration"`
	ValidatorVersion       string                            `json:"validatorVersion"`
	PolicyVersion          string                            `json:"policyVersion"`
	ComparisonResult       refvalidation.ComparisonResult    `json:"comparisonResult"`
}

func EnforceRefShapeComparison(req domain.SyscallRequest) RefShapeComparisonDecision {
	base := RefShapeComparisonDecision{
		Source:                 string(req.Source),
		MemoryMutation:         false,
		RuntimeMutation:        false,
		LiveAuthorityMigration: false,
		ValidatorVersion:       RefShapeCompareValidatorVersion,
		PolicyVersion:          RefShapeComparePolicyVersion,
	}
	if issues := validateRefShapeCompare(req); len(issues) > 0 {
		base.Accepted = false
		base.Decision = RefShapeCompareDecisionMalformed
		base.Reason = issues[0].Message
		return base
	}
	result := refvalidation.CompareRefShapes(refShapeComparisonRequestFromPayload(req))
	base.ComparisonResult = result
	base.Failures = append([]refvalidation.ValidationFailure{}, result.Failures...)
	base.Warnings = append([]string{}, result.Warnings...)
	base.Match = result.Match
	base.AddedRefs = append([]refvalidation.ObjectRef{}, result.AddedRefs...)
	base.RemovedRefs = append([]refvalidation.ObjectRef{}, result.RemovedRefs...)
	base.UnchangedRefs = append([]refvalidation.ObjectRef{}, result.UnchangedRefs...)
	if !result.Passed {
		base.Accepted = false
		base.Decision = RefShapeCompareDecisionRejected
		base.Reason = "ref shape comparison rejected"
		if len(result.Failures) > 0 {
			base.Reason = result.Failures[0].Message
		}
		return base
	}
	base.Accepted = true
	if result.Match {
		base.Decision = RefShapeCompareDecisionMatch
		base.Reason = "ref shape comparison matched"
	} else {
		base.Decision = RefShapeCompareDecisionDrift
		base.Reason = "ref shape comparison drift detected"
	}
	return base
}

func validateRefShapeCompare(req domain.SyscallRequest) []domain.SyscallError {
	if strings.TrimSpace(readString(req.Payload, "workspace_id")) == "" {
		return []domain.SyscallError{errField(domain.ErrMissingRequiredField, "payload.workspace_id", "workspace_id is required")}
	}
	if _, ok := req.Payload["candidate_refs"]; !ok {
		return []domain.SyscallError{errField(domain.ErrMissingRequiredField, "payload.candidate_refs", "candidate_refs are required")}
	}
	if _, ok := req.Payload["observed_refs"]; !ok {
		return []domain.SyscallError{errField(domain.ErrMissingRequiredField, "payload.observed_refs", "observed_refs are required")}
	}
	return nil
}

func refShapeComparisonRequestFromPayload(req domain.SyscallRequest) refvalidation.ComparisonRequest {
	return refvalidation.ComparisonRequest{
		ResultID:      req.ID + ":ref_shape_comparison",
		WorkspaceID:   readString(req.Payload, "workspace_id"),
		CandidateRefs: refObjectsFromPayload(req.Payload["candidate_refs"]),
		ObservedRefs:  refObjectsFromPayload(req.Payload["observed_refs"]),
	}
}

func (d RefShapeComparisonDecision) ToStateSummary() map[string]any {
	return map[string]any{
		"refShapeComparison":     d.ToAuditFields(),
		"refShapeResult":         d.ComparisonResult,
		"passed":                 d.Accepted,
		"match":                  d.Match,
		"addedRefs":              refsForSummary(d.AddedRefs),
		"removedRefs":            refsForSummary(d.RemovedRefs),
		"unchangedRefs":          refsForSummary(d.UnchangedRefs),
		"failures":               append([]refvalidation.ValidationFailure{}, d.Failures...),
		"warnings":               append([]string{}, d.Warnings...),
		"memoryMutation":         d.MemoryMutation,
		"runtimeMutation":        d.RuntimeMutation,
		"liveAuthorityMigration": d.LiveAuthorityMigration,
	}
}

func (d RefShapeComparisonDecision) ToAuditFields() map[string]any {
	return map[string]any{
		"accepted":               d.Accepted,
		"decision":               d.Decision,
		"reason":                 d.Reason,
		"source":                 d.Source,
		"match":                  d.Match,
		"addedRefCount":          len(d.AddedRefs),
		"removedRefCount":        len(d.RemovedRefs),
		"unchangedRefCount":      len(d.UnchangedRefs),
		"failures":               append([]refvalidation.ValidationFailure{}, d.Failures...),
		"warnings":               append([]string{}, d.Warnings...),
		"memoryMutation":         d.MemoryMutation,
		"runtimeMutation":        d.RuntimeMutation,
		"liveAuthorityMigration": d.LiveAuthorityMigration,
		"validatorVersion":       d.ValidatorVersion,
		"policyVersion":          d.PolicyVersion,
	}
}

func (d RefShapeComparisonDecision) ToSyscallError() domain.SyscallError {
	return domain.SyscallError{Code: domain.ErrInvalidPayload, Field: "payload.observed_refs", Message: d.Reason}
}
