package controllane

import (
	"context"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekshadow"
	"forge/projectforge/services/core/internal/refvalidation"
)

func (p *Processor) observeControlLaneValidation(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) {
	if p == nil || p.controlLaneValidationObserver == nil {
		return
	}
	input, ok := controlLaneValidationShadowInput(req, result)
	if !ok {
		return
	}
	defer func() {
		_ = recover()
	}()
	p.controlLaneValidationObserver.ObserveControlLaneValidationBestEffort(ctx, input)
}

func controlLaneValidationShadowInput(req domain.SyscallRequest, result domain.SyscallResult) (forgekshadow.ControlLaneValidationInput, bool) {
	validationKind, summaryKey, ok := controlLaneValidationShadowKind(req.Action)
	if !ok {
		return forgekshadow.ControlLaneValidationInput{}, false
	}
	summary := controlLaneValidationSummary(result.StateSummary, summaryKey)
	return forgekshadow.ControlLaneValidationInput{
		WorkspaceID:            req.Scope.WorkspaceID,
		RequestID:              req.ID,
		CorrelationID:          req.CorrelationID,
		Action:                 string(req.Action),
		ValidationKind:         validationKind,
		Decision:               controlLaneValidationShadowDecision(result, summary),
		Passed:                 result.Success,
		Match:                  controlLaneValidationSummaryBool(result.StateSummary, summary, "match"),
		OperationType:          firstNonEmpty(readString(result.StateSummary, "operationType"), readString(summary, "operationType")),
		NormalizedRefCount:     controlLaneValidationCount(result.StateSummary, summary, "normalizedRefs", "normalizedRefCount"),
		AddedRefCount:          controlLaneValidationCount(result.StateSummary, summary, "addedRefs", "addedRefCount"),
		RemovedRefCount:        controlLaneValidationCount(result.StateSummary, summary, "removedRefs", "removedRefCount"),
		UnchangedRefCount:      controlLaneValidationCount(result.StateSummary, summary, "unchangedRefs", "unchangedRefCount"),
		FailureCount:           len(result.RejectedReasons),
		WarningCount:           len(result.Warnings),
		NormalizedRefs:         controlLaneValidationSummaryRefs(result.StateSummary, summary),
		MemoryMutation:         false,
		RuntimeMutation:        false,
		ModelRuntimeCall:       false,
		EvidenceAdmission:      false,
		ContextCompilation:     false,
		UserVisibleOutput:      false,
		LiveAuthorityMigration: false,
		Metadata: map[string]any{
			"dry_run": result.DryRun,
		},
	}, true
}

func controlLaneValidationSummaryRefs(state, summary map[string]any) []refvalidation.ObjectRef {
	refs := objectRefsFromSummary(summary["normalizedRefs"])
	if len(refs) > 0 {
		return refs
	}
	return objectRefsFromSummary(state["normalizedRefs"])
}

func objectRefsFromSummary(raw any) []refvalidation.ObjectRef {
	items, ok := raw.([]map[string]string)
	if ok {
		out := make([]refvalidation.ObjectRef, 0, len(items))
		for _, item := range items {
			ref := refvalidation.ObjectRef{
				RefType:     strings.TrimSpace(item["ref_type"]),
				RefID:       strings.TrimSpace(item["ref_id"]),
				WorkspaceID: strings.TrimSpace(item["workspace_id"]),
			}
			if ref.RefType != "" || ref.RefID != "" || ref.WorkspaceID != "" {
				out = append(out, ref)
			}
		}
		return out
	}
	anyItems, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]refvalidation.ObjectRef, 0, len(anyItems))
	for _, item := range anyItems {
		values, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ref := refvalidation.ObjectRef{
			RefType:     readString(values, "ref_type"),
			RefID:       readString(values, "ref_id"),
			WorkspaceID: readString(values, "workspace_id"),
		}
		if ref.RefType != "" || ref.RefID != "" || ref.WorkspaceID != "" {
			out = append(out, ref)
		}
	}
	return out
}

func controlLaneValidationShadowKind(action domain.SemanticActionType) (string, string, bool) {
	switch action {
	case domain.ActionValidateKVIdentity:
		return "kv_identity", "kvIdentityEnforcement", true
	case domain.ActionValidateRefShape:
		return "ref_shape", "refShapeValidation", true
	case domain.ActionCompareRefShape:
		return "ref_shape_comparison", "refShapeComparison", true
	case domain.ActionValidateSourceObject:
		return "source_object_authority", "sourceObjectAuthorityValidation", true
	case domain.ActionValidateSemanticOperation:
		return "semantic_operation", "semanticOperationValidation", true
	case domain.ActionValidateAdmissionCandidate:
		return "admission_candidate", "admissionCandidateValidation", true
	default:
		return "", "", false
	}
}

func controlLaneValidationSummary(state map[string]any, key string) map[string]any {
	if state == nil {
		return nil
	}
	summary, _ := state[key].(map[string]any)
	return summary
}

func controlLaneValidationShadowDecision(result domain.SyscallResult, summary map[string]any) string {
	if decision := strings.TrimSpace(readString(summary, "decision")); decision != "" {
		return decision
	}
	if controlLaneValidationBool(summary, "accepted") {
		return "accepted"
	}
	if controlLaneValidationBool(summary, "rejected") {
		return "rejected"
	}
	if controlLaneValidationBool(summary, "mismatch") {
		return "mismatch"
	}
	if controlLaneValidationBool(summary, "validated") {
		return "validated"
	}
	if result.Success {
		return "validated"
	}
	return "rejected"
}

func controlLaneValidationSummaryBool(state, summary map[string]any, key string) bool {
	if controlLaneValidationBool(summary, key) {
		return true
	}
	return controlLaneValidationBool(state, key)
}

func controlLaneValidationBool(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	raw, ok := values[key]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true") || strings.TrimSpace(v) == "1"
	default:
		return false
	}
}

func controlLaneValidationCount(state, summary map[string]any, stateKey, summaryKey string) int {
	if count := controlLaneValidationInt(summary, summaryKey); count > 0 {
		return count
	}
	return controlLaneValidationLen(state[stateKey])
}

func controlLaneValidationInt(values map[string]any, key string) int {
	if values == nil {
		return 0
	}
	switch v := values[key].(type) {
	case int:
		return controlLaneValidationNonNegative(v)
	case int8:
		return controlLaneValidationNonNegative(int(v))
	case int16:
		return controlLaneValidationNonNegative(int(v))
	case int32:
		return controlLaneValidationNonNegative(int(v))
	case int64:
		return controlLaneValidationNonNegative(int(v))
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		if v > uint64(^uint(0)>>1) {
			return int(^uint(0) >> 1)
		}
		return int(v)
	case float64:
		return controlLaneValidationNonNegative(int(v))
	case float32:
		return controlLaneValidationNonNegative(int(v))
	default:
		return 0
	}
}

func controlLaneValidationNonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func controlLaneValidationLen(raw any) int {
	switch v := raw.(type) {
	case []map[string]string:
		return len(v)
	case []map[string]any:
		return len(v)
	case []any:
		return len(v)
	case []string:
		return len(v)
	default:
		return 0
	}
}
