package controllane

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/audit"
)

type SyscallAuditRecord struct {
	Timestamp                    int64
	Action                       domain.SemanticActionType
	Actor                        string
	Source                       domain.ActionSource
	WorkspaceID                  string
	RequestID                    string
	CorrelationID                string
	TraceID                      string
	DryRun                       bool
	Success                      bool
	ApprovalStatus               domain.ApprovalStatus
	ValidationIssues             []domain.SyscallError
	CommittedIDs                 []string
	ErrorCode                    domain.SyscallErrorCode
	KVIdentityEnforcement        map[string]any
	RefShapeValidation           map[string]any
	RefShapeComparison           map[string]any
	SourceObjectAuthority        map[string]any
	SemanticOperationValidation  map[string]any
	AdmissionCandidateValidation map[string]any
	ContextAttributionValidation map[string]any
	SemanticSyscallEnvelope      map[string]any
}

type AuditSink interface {
	Record(ctx context.Context, rec SyscallAuditRecord) (string, error)
}

// AuditOutboxSink is the idempotent delivery boundary used for successful
// FORGE-K commits. The immutable outbox ID is the sink idempotency key.
type AuditOutboxSink interface {
	DeliverOutbox(ctx context.Context, rec AuditOutboxRecord) (string, error)
}

type InMemoryAuditSink struct {
	Records         []SyscallAuditRecord
	DeliveredOutbox map[string]string
}

func NewInMemoryAuditSink() *InMemoryAuditSink {
	return &InMemoryAuditSink{Records: []SyscallAuditRecord{}, DeliveredOutbox: map[string]string{}}
}

func (s *InMemoryAuditSink) DeliverOutbox(_ context.Context, rec AuditOutboxRecord) (string, error) {
	if err := VerifyAuditOutboxRecord(rec); err != nil {
		return "", err
	}
	if id := s.DeliveredOutbox[rec.ID]; id != "" {
		return id, nil
	}
	s.Records = append(s.Records, syscallAuditRecordFromOutbox(rec))
	id := fmt.Sprintf("audit-%d", len(s.Records))
	s.DeliveredOutbox[rec.ID] = id
	return id, nil
}

func (s *InMemoryAuditSink) Record(_ context.Context, rec SyscallAuditRecord) (string, error) {
	if rec.Timestamp <= 0 {
		rec.Timestamp = time.Now().UnixMilli()
	}
	s.Records = append(s.Records, rec)
	return fmt.Sprintf("audit-%d", len(s.Records)), nil
}

type CoreAuditSink struct {
	Service *audit.Service
}

func NewCoreAuditSink(service *audit.Service) *CoreAuditSink {
	return &CoreAuditSink{Service: service}
}

func (s *CoreAuditSink) Record(ctx context.Context, rec SyscallAuditRecord) (string, error) {
	if s == nil || s.Service == nil {
		return "", nil
	}
	created, err := s.Service.Record(ctx, audit.CreateRequest{
		CorrelationID: rec.CorrelationID,
		Category:      "semantic_syscall",
		Action:        "processed",
		Actor:         rec.Actor,
		SubjectType:   "semantic_action",
		SubjectID:     string(rec.Action),
		Outcome: func() string {
			if rec.Success {
				return "ok"
			}
			return "rejected"
		}(),
		Summary: fmt.Sprintf("action=%s dryRun=%v source=%s", rec.Action, rec.DryRun, rec.Source),
		Payload: map[string]any{
			"workspaceId":                  rec.WorkspaceID,
			"requestId":                    rec.RequestID,
			"traceId":                      rec.TraceID,
			"approvalStatus":               rec.ApprovalStatus,
			"committedIds":                 rec.CommittedIDs,
			"errorCode":                    rec.ErrorCode,
			"validationIssues":             rec.ValidationIssues,
			"kvIdentityEnforcement":        rec.KVIdentityEnforcement,
			"refShapeValidation":           rec.RefShapeValidation,
			"refShapeComparison":           rec.RefShapeComparison,
			"sourceObjectAuthority":        rec.SourceObjectAuthority,
			"semanticOperationValidation":  rec.SemanticOperationValidation,
			"admissionCandidateValidation": rec.AdmissionCandidateValidation,
			"contextAttributionValidation": rec.ContextAttributionValidation,
			"semanticSyscallEnvelope":      rec.SemanticSyscallEnvelope,
		},
	})
	if err != nil {
		return "", err
	}
	if created == nil {
		return "", nil
	}
	return strconv.FormatInt(created.ID, 10), nil
}

func (s *CoreAuditSink) DeliverOutbox(ctx context.Context, rec AuditOutboxRecord) (string, error) {
	if s == nil || s.Service == nil {
		return "", fmt.Errorf("audit service unavailable")
	}
	if err := VerifyAuditOutboxRecord(rec); err != nil {
		return "", err
	}
	created, err := s.Service.RecordForgeKOutbox(ctx, rec.ID, audit.CreateRequest{
		CorrelationID: rec.CorrelationID,
		Category:      "semantic_syscall",
		Action:        "delivered",
		Actor:         rec.Request.Actor.ID,
		SubjectType:   "semantic_action",
		SubjectID:     string(rec.Action),
		Outcome:       "ok",
		Summary:       fmt.Sprintf("action=%s source=%s FORGE-K outbox=%s", rec.Action, rec.Request.Source, rec.ID),
		Payload: map[string]any{
			"forgeKOutboxId":     rec.ID,
			"requestFingerprint": rec.RequestFingerprint,
			"workspaceId":        rec.WorkspaceID,
			"laneId":             rec.LaneID,
			"requestId":          rec.SyscallID,
			"traceId":            rec.TraceID,
			"committedIds":       rec.Result.CommittedObjectIDs,
			"transactionId":      rec.Receipt.TransactionID,
			"journalEventId":     rec.Receipt.JournalEventID,
			"journalEventHash":   rec.Receipt.JournalEventHash,
			"authorizationProof": rec.AuthorizationProof,
			"request":            rec.Request,
			"result":             rec.Result,
			"receipt":            rec.Receipt,
		},
	})
	if err != nil {
		return "", err
	}
	if created == nil {
		return "", fmt.Errorf("audit service returned no record")
	}
	return strconv.FormatInt(created.ID, 10), nil
}

func syscallAuditRecordFromOutbox(rec AuditOutboxRecord) SyscallAuditRecord {
	out := SyscallAuditRecord{
		Timestamp:      rec.CreatedAt,
		Action:         rec.Action,
		Actor:          rec.Request.Actor.ID,
		Source:         rec.Request.Source,
		WorkspaceID:    rec.WorkspaceID,
		RequestID:      rec.SyscallID,
		CorrelationID:  rec.CorrelationID,
		TraceID:        rec.TraceID,
		Success:        rec.Success,
		ApprovalStatus: rec.Result.ApprovalStatus,
		CommittedIDs:   append([]string{}, rec.Result.CommittedObjectIDs...),
	}
	if def, ok := NewStaticActionRegistry().Get(rec.Action); ok {
		out.SemanticSyscallEnvelope = BuildSemanticSyscallFacade(rec.Request, def).ToAuditFields()
	}
	return out
}
