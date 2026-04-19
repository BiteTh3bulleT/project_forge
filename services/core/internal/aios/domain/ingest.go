package domain

import (
	"fmt"
	"strings"
)

type IngestInputKind string

const (
	IngestUserMessage   IngestInputKind = "user_message"
	IngestSystemEvent   IngestInputKind = "system_event"
	IngestToolResult    IngestInputKind = "tool_result"
	IngestArtifactEvent IngestInputKind = "artifact_event"
	IngestAdapterEvent  IngestInputKind = "adapter_event"
	IngestTestEvent     IngestInputKind = "test_event"
)

type IngestCommitMode string

const (
	IngestValidateOnly    IngestCommitMode = "validate_only"
	IngestCommitValid     IngestCommitMode = "commit_valid"
	IngestCommitAllOrFail IngestCommitMode = "commit_all_or_fail"
)

type IngestRequest struct {
	ID             string           `json:"id"`
	InputKind      IngestInputKind  `json:"inputKind"`
	Content        string           `json:"content"`
	Payload        map[string]any   `json:"payload"`
	Actor          ActorIdentity    `json:"actor"`
	Source         ActionSource     `json:"source"`
	Scope          ForgeScope       `json:"scope"`
	Provenance     Provenance       `json:"provenance"`
	CorrelationID  string           `json:"correlationId,omitempty"`
	TraceID        string           `json:"traceId,omitempty"`
	IdempotencyKey string           `json:"idempotencyKey,omitempty"`
	DryRun         bool             `json:"dryRun"`
	CommitMode     IngestCommitMode `json:"commitMode"`
	Metadata       map[string]any   `json:"metadata,omitempty"`
	RequestedAt    int64            `json:"requestedAt"`
}

type IngestErrorCode string

const (
	IngestErrInvalidRequest IngestErrorCode = "INVALID_INGEST_REQUEST"
	IngestErrInvalidMode    IngestErrorCode = "INVALID_COMMIT_MODE"
	IngestErrUnsupported    IngestErrorCode = "UNSUPPORTED_INGEST_MODE"
	IngestErrEventAppend    IngestErrorCode = "EVENT_APPEND_FAILED"
	IngestErrCellRun        IngestErrorCode = "CELL_RUN_FAILED"
	IngestErrCellDependency IngestErrorCode = "CELL_DEPENDENCY_INVALID"
	IngestErrKernel         IngestErrorCode = "KERNEL_PROCESS_FAILED"
)

type IngestError struct {
	Code    IngestErrorCode `json:"code"`
	Field   string          `json:"field,omitempty"`
	Message string          `json:"message"`
}

type CandidateActionBatch struct {
	ID            string           `json:"id"`
	SourceEventID string           `json:"sourceEventId"`
	ProducedBy    string           `json:"producedBy"`
	WorkspaceID   string           `json:"workspaceId"`
	LaneID        string           `json:"laneId,omitempty"`
	CorrelationID string           `json:"correlationId,omitempty"`
	TraceID       string           `json:"traceId,omitempty"`
	Actions       []SyscallRequest `json:"actions"`
	Warnings      []string         `json:"warnings"`
	Confidence    float64          `json:"confidence,omitempty"`
	Priority      int              `json:"priority,omitempty"`
	Metadata      map[string]any   `json:"metadata,omitempty"`
}

type CellDiagnostic struct {
	CellName      string         `json:"cellName"`
	CellVersion   string         `json:"cellVersion"`
	ProposedCount int            `json:"proposedCount"`
	Warnings      []string       `json:"warnings"`
	Errors        []IngestError  `json:"errors"`
	DurationMs    int64          `json:"durationMs,omitempty"`
	Skipped       bool           `json:"skipped"`
	SkippedReason string         `json:"skippedReason,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type IngestActionOutcome struct {
	Action         SyscallRequest `json:"action"`
	Result         SyscallResult  `json:"result"`
	CellName       string         `json:"cellName"`
	CellVersion    string         `json:"cellVersion"`
	CandidateBatch string         `json:"candidateBatch"`
}

type IngestSummary struct {
	ProposedCount  int `json:"proposedCount"`
	AcceptedCount  int `json:"acceptedCount"`
	RejectedCount  int `json:"rejectedCount"`
	CommittedCount int `json:"committedCount"`
	CellCount      int `json:"cellCount"`
}

type IngestResult struct {
	Success            bool                   `json:"success"`
	EventID            string                 `json:"eventId"`
	Scope              ForgeScope             `json:"scope"`
	CorrelationID      string                 `json:"correlationId,omitempty"`
	TraceID            string                 `json:"traceId,omitempty"`
	CellRunID          string                 `json:"cellRunId,omitempty"`
	ProposedActions    []SyscallRequest       `json:"proposedActions"`
	AcceptedActions    []IngestActionOutcome  `json:"acceptedActions"`
	RejectedActions    []IngestActionOutcome  `json:"rejectedActions"`
	CommittedObjectIDs []string               `json:"committedObjectIds"`
	Warnings           []string               `json:"warnings"`
	Errors             []IngestError          `json:"errors"`
	AuditIDs           []string               `json:"auditIds"`
	DryRun             bool                   `json:"dryRun"`
	Summary            IngestSummary          `json:"summary"`
	Diagnostics        []CellDiagnostic       `json:"diagnostics"`
	Batches            []CandidateActionBatch `json:"batches"`
	AutonomyRuns       []AutonomyRunSummary   `json:"autonomyRuns,omitempty"`
	TruthDiagnostics   map[string]any         `json:"truthDiagnostics,omitempty"`
}

func (r IngestRequest) Validate() []IngestError {
	var out []IngestError
	if strings.TrimSpace(string(r.InputKind)) == "" {
		out = append(out, IngestError{Code: IngestErrInvalidRequest, Field: "inputKind", Message: "inputKind is required"})
	}
	if strings.TrimSpace(string(r.Source)) == "" {
		out = append(out, IngestError{Code: IngestErrInvalidRequest, Field: "source", Message: "source is required"})
	}
	if strings.TrimSpace(r.Scope.WorkspaceID) == "" {
		out = append(out, IngestError{Code: IngestErrInvalidRequest, Field: "scope.workspaceId", Message: "scope.workspaceId is required"})
	}
	if strings.TrimSpace(r.Actor.ID) == "" {
		out = append(out, IngestError{Code: IngestErrInvalidRequest, Field: "actor.id", Message: "actor.id is required"})
	}
	if strings.TrimSpace(r.Actor.Kind) == "" {
		out = append(out, IngestError{Code: IngestErrInvalidRequest, Field: "actor.kind", Message: "actor.kind is required"})
	}
	if strings.TrimSpace(r.Provenance.Actor) == "" {
		out = append(out, IngestError{Code: IngestErrInvalidRequest, Field: "provenance.actor", Message: "provenance.actor is required"})
	}
	if strings.TrimSpace(r.Provenance.ActorType) == "" {
		out = append(out, IngestError{Code: IngestErrInvalidRequest, Field: "provenance.actorType", Message: "provenance.actorType is required"})
	}
	switch r.CommitMode {
	case "", IngestValidateOnly, IngestCommitValid, IngestCommitAllOrFail:
	default:
		out = append(out, IngestError{
			Code:    IngestErrInvalidMode,
			Field:   "commitMode",
			Message: fmt.Sprintf("unsupported commit mode %q", r.CommitMode),
		})
	}
	return out
}
