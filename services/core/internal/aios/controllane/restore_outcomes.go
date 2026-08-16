package controllane

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

type RestoreOutcome string

const (
	RestoreOutcomeUnknown              RestoreOutcome = "unknown"
	RestoreOutcomeHelpful              RestoreOutcome = "helpful"
	RestoreOutcomeNotHelpful           RestoreOutcome = "not_helpful"
	RestoreOutcomeHarmful              RestoreOutcome = "harmful"
	RestoreOutcomeStale                RestoreOutcome = "stale"
	RestoreOutcomeContradictory        RestoreOutcome = "contradictory"
	RestoreOutcomeFreshCompileRequired RestoreOutcome = "fresh_compile_required"
	RestoreOutcomeOperatorCorrected    RestoreOutcome = "operator_corrected"
	RestoreOutcomeNoCandidate          RestoreOutcome = "no_candidate"
	RestoreOutcomeFailedExecution      RestoreOutcome = "failed_execution"
)

type RestoreOutcomeEvent struct {
	ID                   string         `json:"id"`
	CreatedAt            int64          `json:"createdAt"`
	UpdatedAt            int64          `json:"updatedAt"`
	WorkspaceID          string         `json:"workspaceId"`
	LaneID               string         `json:"laneId"`
	Query                string         `json:"query"`
	ContextPacketID      string         `json:"contextPacketId"`
	SnapshotID           string         `json:"snapshotId"`
	SnapshotKind         string         `json:"snapshotKind"`
	RestoreScore         float64        `json:"restoreScore"`
	RequiresFreshCompile bool           `json:"requiresFreshCompile"`
	SelectedEvidence     []string       `json:"selectedEvidence"`
	SelectedStateKeys    []string       `json:"selectedStateKeys"`
	SelectedLoopIDs      []string       `json:"selectedLoopIds"`
	SelectedArtifactIDs  []string       `json:"selectedArtifactIds"`
	Outcome              RestoreOutcome `json:"outcome"`
	OutcomeConfidence    float64        `json:"outcomeConfidence"`
	OperatorFeedback     string         `json:"operatorFeedback"`
	FailureReason        string         `json:"failureReason"`
	CorrectionSummary    string         `json:"correctionSummary"`
	DownstreamActionType string         `json:"downstreamActionType"`
	DownstreamObjectID   string         `json:"downstreamObjectId"`
	CorrelationID        string         `json:"correlationId"`
	TraceID              string         `json:"traceId"`
	SyscallID            string         `json:"syscallId"`
	AuditID              string         `json:"auditId"`
	ProposedBy           string         `json:"proposedBy"`
	CommittedBy          string         `json:"committedBy"`
	Metadata             map[string]any `json:"metadata"`
}

type RestoreOutcomeFilter struct {
	WorkspaceID string
	LaneID      string
	Query       string
	SnapshotID  string
	Outcome     RestoreOutcome
	Since       int64
	Limit       int
}

type RestoreOutcomeFeedback struct {
	Outcome           RestoreOutcome `json:"outcome"`
	OutcomeConfidence float64        `json:"outcomeConfidence"`
	OperatorFeedback  string         `json:"operatorFeedback"`
	CorrectionSummary string         `json:"correctionSummary"`
	Metadata          map[string]any `json:"metadata"`
	CorrelationID     string         `json:"correlationId"`
	TraceID           string         `json:"traceId"`
	UpdatedBy         string         `json:"updatedBy"`
	UpdatedAt         int64          `json:"updatedAt"`
}

type RestoreOutcomeStore interface {
	GetRestoreOutcome(ctx context.Context, id string) (RestoreOutcomeEvent, bool, error)
	ListRestoreOutcomes(ctx context.Context, filter RestoreOutcomeFilter) ([]RestoreOutcomeEvent, error)
}

func NewRestoreOutcomeID(parts ...string) string {
	h := sha1.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(strings.TrimSpace(part)))
		_, _ = h.Write([]byte{0})
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if len(sum) > 16 {
		sum = sum[:16]
	}
	return "restore-outcome-" + sum
}

func ValidateRestoreOutcome(outcome RestoreOutcome) bool {
	switch outcome {
	case RestoreOutcomeUnknown, RestoreOutcomeHelpful, RestoreOutcomeNotHelpful, RestoreOutcomeHarmful,
		RestoreOutcomeStale, RestoreOutcomeContradictory, RestoreOutcomeFreshCompileRequired,
		RestoreOutcomeOperatorCorrected, RestoreOutcomeNoCandidate, RestoreOutcomeFailedExecution:
		return true
	default:
		return false
	}
}

func normalizeRestoreOutcomeEvent(event RestoreOutcomeEvent) RestoreOutcomeEvent {
	event.ID = strings.TrimSpace(event.ID)
	event.WorkspaceID = strings.TrimSpace(event.WorkspaceID)
	event.LaneID = strings.TrimSpace(event.LaneID)
	event.Query = strings.TrimSpace(event.Query)
	event.ContextPacketID = strings.TrimSpace(event.ContextPacketID)
	event.SnapshotID = strings.TrimSpace(event.SnapshotID)
	event.SnapshotKind = strings.TrimSpace(event.SnapshotKind)
	event.CorrelationID = strings.TrimSpace(event.CorrelationID)
	event.TraceID = strings.TrimSpace(event.TraceID)
	event.SyscallID = strings.TrimSpace(event.SyscallID)
	event.AuditID = strings.TrimSpace(event.AuditID)
	event.ProposedBy = strings.TrimSpace(event.ProposedBy)
	event.CommittedBy = strings.TrimSpace(event.CommittedBy)
	if event.CommittedBy == "" {
		event.CommittedBy = "forge_kernel"
	}
	if !ValidateRestoreOutcome(event.Outcome) {
		event.Outcome = RestoreOutcomeUnknown
	}
	event.OutcomeConfidence = clamp01(event.OutcomeConfidence)
	event.SelectedEvidence = normalizeStringSet(event.SelectedEvidence)
	event.SelectedStateKeys = normalizeStringSet(event.SelectedStateKeys)
	event.SelectedLoopIDs = normalizeStringSet(event.SelectedLoopIDs)
	event.SelectedArtifactIDs = normalizeStringSet(event.SelectedArtifactIDs)
	if event.Metadata == nil {
		event.Metadata = map[string]any{}
	}
	return event
}

func normalizeRestoreOutcomeFilter(filter RestoreOutcomeFilter) RestoreOutcomeFilter {
	filter.WorkspaceID = strings.TrimSpace(filter.WorkspaceID)
	filter.LaneID = strings.TrimSpace(filter.LaneID)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.SnapshotID = strings.TrimSpace(filter.SnapshotID)
	if !ValidateRestoreOutcome(filter.Outcome) {
		filter.Outcome = ""
	}
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	return filter
}

func normalizeRestoreOutcomeFeedback(feedback RestoreOutcomeFeedback) RestoreOutcomeFeedback {
	if !ValidateRestoreOutcome(feedback.Outcome) {
		feedback.Outcome = RestoreOutcomeUnknown
	}
	feedback.OutcomeConfidence = clamp01(feedback.OutcomeConfidence)
	feedback.OperatorFeedback = strings.TrimSpace(feedback.OperatorFeedback)
	feedback.CorrectionSummary = strings.TrimSpace(feedback.CorrectionSummary)
	feedback.CorrelationID = strings.TrimSpace(feedback.CorrelationID)
	feedback.TraceID = strings.TrimSpace(feedback.TraceID)
	feedback.UpdatedBy = strings.TrimSpace(feedback.UpdatedBy)
	if feedback.UpdatedBy == "" {
		feedback.UpdatedBy = "operator"
	}
	if feedback.Metadata == nil {
		feedback.Metadata = map[string]any{}
	}
	return feedback
}

func normalizeStringSet(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func restoreOutcomeNotFound(id string) error {
	return fmt.Errorf("restore outcome %q not found", strings.TrimSpace(id))
}
