package controllane

import (
	"fmt"
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

const (
	RetrievalUsefulnessUseful       = "useful"
	RetrievalUsefulnessNotUseful    = "not_useful"
	RetrievalUsefulnessNoisy        = "noisy"
	RetrievalUsefulnessInsufficient = "insufficient"
	RetrievalUsefulnessUnknown      = "unknown"
)

// RetrievalUsefulnessTarget is the immutable identity/scope binding read from
// the original FORGE-K retrieval evidence before utility feedback is admitted.
// Legacy rows without these bindings are deliberately not eligible.
type RetrievalUsefulnessTarget struct {
	ResultID       int64             `json:"resultId"`
	RunID          int64             `json:"runId"`
	EvidenceID     string            `json:"evidenceId"`
	Scope          domain.ForgeScope `json:"scope"`
	JobID          *string           `json:"jobId,omitempty"`
	PacketID       *int64            `json:"packetId,omitempty"`
	OriginalLabel  string            `json:"originalLabel"`
	OriginalNote   string            `json:"originalNote"`
	SourceSyscall  string            `json:"sourceSyscallId"`
	SourceProvID   string            `json:"sourceProvenanceId"`
	SourceProvJSON string            `json:"sourceProvenanceJson"`
}

// RetrievalUsefulnessEvent is append-only authority evidence. Projection
// columns on retrieval_results may be rebuilt from these rows; they are never
// the evidence authority themselves.
type RetrievalUsefulnessEvent struct {
	ID               string            `json:"id"`
	CreatedAt        int64             `json:"createdAt"`
	ResultID         int64             `json:"resultId"`
	RunID            int64             `json:"runId"`
	TargetEvidenceID string            `json:"targetEvidenceId"`
	Scope            domain.ForgeScope `json:"scope"`
	Label            string            `json:"label"`
	Note             string            `json:"note"`
	JobID            *string           `json:"jobId,omitempty"`
	PacketID         *int64            `json:"packetId,omitempty"`
	PriorProjection  map[string]any    `json:"priorProjection"`
	CorrelationID    string            `json:"correlationId"`
	TraceID          string            `json:"traceId"`
	SyscallID        string            `json:"syscallId"`
	Provenance       domain.Provenance `json:"provenance"`
	ProvenanceID     string            `json:"provenanceId"`
	ProposedBy       string            `json:"proposedBy"`
	CommittedBy      string            `json:"committedBy"`
	Metadata         map[string]any    `json:"metadata"`
	SourceProvenance map[string]any    `json:"sourceProvenance"`
}

type RestoreOutcomeFeedbackTarget struct {
	RestoreOutcomeID string            `json:"restoreOutcomeId"`
	Scope            domain.ForgeScope `json:"scope"`
	OriginalOutcome  RestoreOutcome    `json:"originalOutcome"`
	SourceSyscall    string            `json:"sourceSyscallId"`
	CommittedBy      string            `json:"committedBy"`
}

// RestoreOutcomeFeedbackEvent preserves the original restore outcome row and
// appends operator feedback separately. The projection is explicitly
// noncanonical and may be discarded/rebuilt from this event history.
type RestoreOutcomeFeedbackEvent struct {
	ID                 string                 `json:"id"`
	CreatedAt          int64                  `json:"createdAt"`
	RestoreOutcomeID   string                 `json:"restoreOutcomeId"`
	Scope              domain.ForgeScope      `json:"scope"`
	OriginalOutcome    RestoreOutcome         `json:"originalOutcome"`
	Outcome            RestoreOutcome         `json:"outcome"`
	OutcomeConfidence  float64                `json:"outcomeConfidence"`
	OperatorFeedback   string                 `json:"operatorFeedback"`
	CorrectionSummary  string                 `json:"correctionSummary"`
	CorrelationID      string                 `json:"correlationId"`
	TraceID            string                 `json:"traceId"`
	SyscallID          string                 `json:"syscallId"`
	Provenance         domain.Provenance      `json:"provenance"`
	ProvenanceID       string                 `json:"provenanceId"`
	ProposedBy         string                 `json:"proposedBy"`
	CommittedBy        string                 `json:"committedBy"`
	Metadata           map[string]any         `json:"metadata"`
	PriorProjection    map[string]any         `json:"priorProjection"`
	ProjectionSnapshot RestoreOutcomeFeedback `json:"projectionSnapshot"`
}

type RestoreOutcomeFeedbackProjection struct {
	RestoreOutcomeID  string            `json:"restoreOutcomeId"`
	LatestEventID     string            `json:"latestEventId"`
	Scope             domain.ForgeScope `json:"scope"`
	Outcome           RestoreOutcome    `json:"outcome"`
	OutcomeConfidence float64           `json:"outcomeConfidence"`
	OperatorFeedback  string            `json:"operatorFeedback"`
	CorrectionSummary string            `json:"correctionSummary"`
	UpdatedBy         string            `json:"updatedBy"`
	UpdatedAt         int64             `json:"updatedAt"`
	Metadata          map[string]any    `json:"metadata"`
	NonCanonical      bool              `json:"nonCanonical"`
}

type RetrievalUsefulnessProjection struct {
	ResultID      int64  `json:"resultId"`
	LatestEventID string `json:"latestEventId"`
	Label         string `json:"label"`
	Note          string `json:"note"`
	UpdatedAt     int64  `json:"updatedAt"`
	NonCanonical  bool   `json:"nonCanonical"`
}

func NormalizeRetrievalUsefulnessLabel(label string) string {
	return strings.ToLower(strings.TrimSpace(label))
}

func ValidateRetrievalUsefulnessLabel(label string) bool {
	switch NormalizeRetrievalUsefulnessLabel(label) {
	case RetrievalUsefulnessUseful, RetrievalUsefulnessNotUseful, RetrievalUsefulnessNoisy,
		RetrievalUsefulnessInsufficient, RetrievalUsefulnessUnknown:
		return true
	default:
		return false
	}
}

func exactUtilityScopeMatches(left, right domain.ForgeScope) bool {
	if strings.TrimSpace(left.WorkspaceID) == "" || strings.TrimSpace(left.LaneID) == "" {
		return false
	}
	if len(normalizedUtilityPaths(left.SelectedPaths)) == 0 || len(normalizedUtilityPaths(right.SelectedPaths)) == 0 {
		return false
	}
	if strings.TrimSpace(left.WorkspaceID) != strings.TrimSpace(right.WorkspaceID) ||
		strings.TrimSpace(left.LaneID) != strings.TrimSpace(right.LaneID) {
		return false
	}
	return stringSetsEqual(left.SelectedPaths, right.SelectedPaths)
}

func stringSetsEqual(left, right []string) bool {
	l, r := normalizedUtilityPaths(left), normalizedUtilityPaths(right)
	if len(l) != len(r) {
		return false
	}
	for index := range l {
		if l[index] != r[index] {
			return false
		}
	}
	return true
}

func normalizedUtilityPaths(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func validateRetrievalUsefulnessEvent(event RetrievalUsefulnessEvent, target RetrievalUsefulnessTarget) error {
	if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.SyscallID) == "" || event.CreatedAt <= 0 {
		return fmt.Errorf("retrieval usefulness event identity/syscall/timestamp required")
	}
	if event.ResultID <= 0 || event.ResultID != target.ResultID || event.RunID != target.RunID ||
		strings.TrimSpace(event.TargetEvidenceID) == "" || event.TargetEvidenceID != target.EvidenceID {
		return fmt.Errorf("retrieval usefulness target binding mismatch")
	}
	if !exactUtilityScopeMatches(event.Scope, target.Scope) {
		return fmt.Errorf("retrieval usefulness target scope mismatch")
	}
	if !ValidateRetrievalUsefulnessLabel(event.Label) {
		return fmt.Errorf("invalid retrieval usefulness label")
	}
	if strings.TrimSpace(event.Provenance.Actor) == "" || strings.TrimSpace(event.Provenance.ActorType) == "" {
		return fmt.Errorf("retrieval usefulness provenance required")
	}
	if event.CommittedBy != "forge_k.kernel" {
		return fmt.Errorf("retrieval usefulness must be committed by forge_k.kernel")
	}
	if strings.TrimSpace(target.SourceSyscall) == "" || strings.TrimSpace(target.SourceProvID) == "" || strings.TrimSpace(target.SourceProvJSON) == "" {
		return fmt.Errorf("retrieval usefulness target is legacy-unbound")
	}
	return nil
}

func validateRestoreOutcomeFeedbackEvent(event RestoreOutcomeFeedbackEvent, target RestoreOutcomeFeedbackTarget) error {
	if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.SyscallID) == "" || event.CreatedAt <= 0 {
		return fmt.Errorf("restore outcome feedback event identity/syscall/timestamp required")
	}
	if strings.TrimSpace(event.RestoreOutcomeID) == "" || event.RestoreOutcomeID != target.RestoreOutcomeID ||
		event.OriginalOutcome != target.OriginalOutcome {
		return fmt.Errorf("restore outcome feedback target binding mismatch")
	}
	if !exactUtilityScopeMatches(event.Scope, target.Scope) {
		return fmt.Errorf("restore outcome feedback target scope mismatch")
	}
	if !ValidateRestoreOutcome(event.Outcome) || event.Outcome == RestoreOutcomeUnknown || event.OutcomeConfidence < 0 || event.OutcomeConfidence > 1 {
		return fmt.Errorf("valid non-unknown restore outcome and confidence required")
	}
	if strings.TrimSpace(event.Provenance.Actor) == "" || strings.TrimSpace(event.Provenance.ActorType) == "" {
		return fmt.Errorf("restore outcome feedback provenance required")
	}
	if event.CommittedBy != "forge_k.kernel" {
		return fmt.Errorf("restore outcome feedback must be committed by forge_k.kernel")
	}
	if strings.TrimSpace(target.SourceSyscall) == "" || strings.TrimSpace(target.CommittedBy) == "" {
		return fmt.Errorf("restore outcome target is legacy-unbound")
	}
	return nil
}
