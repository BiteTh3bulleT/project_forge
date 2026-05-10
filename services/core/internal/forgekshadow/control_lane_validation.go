package forgekshadow

import (
	"fmt"
	"strings"
	"time"
)

const (
	controlLaneValidationObservationType = "control_lane_validation"
	controlLaneValidationRouteClass      = "control_lane"
)

func normalizeControlLaneValidationInput(input ControlLaneValidationInput, now time.Time, observationID string) (ControlLaneValidationObservation, map[string]any, error) {
	action := normalizeControlLaneAction(input.Action)
	if action == "" {
		action = "UNKNOWN_VALIDATION"
	}
	kind := normalizeControlLaneValidationToken(input.ValidationKind)
	if kind == "" {
		kind = "validation"
	}
	decision := normalizeControlLaneValidationToken(input.Decision)
	if decision == "" {
		decision = "unknown"
	}
	operationType := normalizeControlLaneValidationToken(input.OperationType)
	warnings, err := safeControlLaneValidationWarnings(input.Warnings)
	if err != nil {
		return ControlLaneValidationObservation{}, nil, err
	}
	if input.MemoryMutation || input.RuntimeMutation || input.ModelRuntimeCall || input.EvidenceAdmission || input.ContextCompilation || input.UserVisibleOutput || input.LiveAuthorityMigration {
		return ControlLaneValidationObservation{}, nil, fmt.Errorf("%w: control lane validation observation claims forbidden effects", ErrPolicyRejected)
	}
	durationMS := input.Duration.Milliseconds()
	if durationMS < 0 {
		durationMS = 0
	}
	metadata := map[string]any{
		"observation_type":    controlLaneValidationObservationType,
		"route_class":         controlLaneValidationRouteClass,
		"action":              action,
		"validation_kind":     kind,
		"decision":            decision,
		"passed":              input.Passed,
		"duration_ms":         durationMS,
		"memory_mutation":     false,
		"runtime_mutation":    false,
		"modelruntime_call":   false,
		"evidence_admission":  false,
		"context_compilation": false,
		"user_visible_output": false,
	}
	if requestID := strings.TrimSpace(input.RequestID); requestID != "" {
		metadata["request_id"] = requestID
	}
	if workspaceID := strings.TrimSpace(input.WorkspaceID); workspaceID != "" {
		metadata["workspace_id"] = workspaceID
	}
	if correlationID := strings.TrimSpace(input.CorrelationID); correlationID != "" {
		metadata["correlation_id"] = correlationID
	}
	if operationType != "" {
		metadata["operation_type"] = operationType
	}
	addCount(metadata, "normalized_ref_count", input.NormalizedRefCount)
	addCount(metadata, "added_ref_count", input.AddedRefCount)
	addCount(metadata, "removed_ref_count", input.RemovedRefCount)
	addCount(metadata, "unchanged_ref_count", input.UnchangedRefCount)
	addCount(metadata, "failure_count", input.FailureCount)
	addCount(metadata, "warning_count", input.WarningCount+len(warnings))
	for key, value := range input.Metadata {
		trimmedKey := strings.TrimSpace(key)
		if isReservedControlLaneValidationMetadataKey(trimmedKey) {
			continue
		}
		if _, exists := metadata[trimmedKey]; exists {
			continue
		}
		metadata[trimmedKey] = value
	}
	safe, err := safeMetadata(metadata)
	if err != nil {
		return ControlLaneValidationObservation{}, nil, err
	}
	return ControlLaneValidationObservation{
		ObservationID:          observationID,
		ObservedAt:             now,
		WorkspaceID:            strings.TrimSpace(input.WorkspaceID),
		RequestID:              strings.TrimSpace(input.RequestID),
		CorrelationID:          strings.TrimSpace(input.CorrelationID),
		Action:                 action,
		ValidationKind:         kind,
		Decision:               decision,
		Passed:                 input.Passed,
		Match:                  input.Match,
		OperationType:          operationType,
		NormalizedRefCount:     nonNegative(input.NormalizedRefCount),
		AddedRefCount:          nonNegative(input.AddedRefCount),
		RemovedRefCount:        nonNegative(input.RemovedRefCount),
		UnchangedRefCount:      nonNegative(input.UnchangedRefCount),
		FailureCount:           nonNegative(input.FailureCount),
		WarningCount:           nonNegative(input.WarningCount + len(warnings)),
		MemoryMutation:         false,
		RuntimeMutation:        false,
		ModelRuntimeCall:       false,
		EvidenceAdmission:      false,
		ContextCompilation:     false,
		UserVisibleOutput:      false,
		LiveAuthorityMigration: false,
		DurationMS:             durationMS,
		Warnings:               warnings,
		Metadata:               safe,
	}, safe, nil
}

func addCount(metadata map[string]any, key string, value int) {
	if value > 0 {
		metadata[key] = value
	}
}

func nonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func normalizeControlLaneValidationToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	if containsUnsafeTerm(value) || containsRawContentTerm(value) || len(value) > 96 {
		return ""
	}
	return value
}

func normalizeControlLaneAction(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	if containsUnsafeTerm(value) || containsRawContentTerm(value) || len(value) > 96 {
		return ""
	}
	return value
}

func safeControlLaneValidationWarnings(in []string) ([]string, error) {
	out := make([]string, 0, len(in))
	for _, warning := range in {
		warning = normalizeControlLaneValidationToken(warning)
		if warning == "" {
			continue
		}
		out = append(out, warning)
	}
	return normalizeAdvisoryStrings(out), nil
}

func isReservedControlLaneValidationMetadataKey(key string) bool {
	switch normalizeMetadataToken(key) {
	case "observation_type", "route_class", "action", "validation_kind", "decision",
		"passed", "duration_ms", "request_id", "workspace_id", "correlation_id",
		"operation_type", "normalized_ref_count", "added_ref_count", "removed_ref_count",
		"unchanged_ref_count", "failure_count", "warning_count", "memory_mutation",
		"runtime_mutation", "modelruntime_call", "evidence_admission", "context_compilation",
		"user_visible_output":
		return true
	default:
		return false
	}
}
