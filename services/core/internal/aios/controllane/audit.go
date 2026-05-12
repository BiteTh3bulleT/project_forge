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
	Timestamp                   int64
	Action                      domain.SemanticActionType
	Actor                       string
	Source                      domain.ActionSource
	WorkspaceID                 string
	RequestID                   string
	CorrelationID               string
	TraceID                     string
	DryRun                      bool
	Success                     bool
	ApprovalStatus              domain.ApprovalStatus
	ValidationIssues            []domain.SyscallError
	CommittedIDs                []string
	ErrorCode                   domain.SyscallErrorCode
	KVIdentityEnforcement       map[string]any
	RefShapeValidation          map[string]any
	RefShapeComparison          map[string]any
	SourceObjectAuthority       map[string]any
	SemanticOperationValidation map[string]any
}

type AuditSink interface {
	Record(ctx context.Context, rec SyscallAuditRecord) (string, error)
}

type InMemoryAuditSink struct {
	Records []SyscallAuditRecord
}

func NewInMemoryAuditSink() *InMemoryAuditSink {
	return &InMemoryAuditSink{Records: []SyscallAuditRecord{}}
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
			"workspaceId":                 rec.WorkspaceID,
			"requestId":                   rec.RequestID,
			"traceId":                     rec.TraceID,
			"approvalStatus":              rec.ApprovalStatus,
			"committedIds":                rec.CommittedIDs,
			"errorCode":                   rec.ErrorCode,
			"validationIssues":            rec.ValidationIssues,
			"kvIdentityEnforcement":       rec.KVIdentityEnforcement,
			"refShapeValidation":          rec.RefShapeValidation,
			"refShapeComparison":          rec.RefShapeComparison,
			"sourceObjectAuthority":       rec.SourceObjectAuthority,
			"semanticOperationValidation": rec.SemanticOperationValidation,
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
