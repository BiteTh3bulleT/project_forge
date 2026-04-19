package controllane

import (
	"context"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

type ForgeKernelProcessor interface {
	Process(ctx context.Context, req domain.SyscallRequest) (domain.SyscallResult, error)
}

type ProcessorOptions struct {
	Registry     ActionRegistry
	Validator    SemanticActionValidator
	Capabilities CapabilityService
	ApprovalGate ApprovalGate
	TxRunner     TransactionRunner
	AuditSink    AuditSink
	NowMillis    func() int64
}

type Processor struct {
	registry     ActionRegistry
	validator    SemanticActionValidator
	capabilities CapabilityService
	approvalGate ApprovalGate
	txRunner     TransactionRunner
	auditSink    AuditSink
	nowMillis    func() int64
}

type auditLinkingRunner interface {
	LinkAudit(ctx context.Context, correlationID, syscallID, auditID string) error
}

func NewProcessor(opts ProcessorOptions) *Processor {
	nowFn := opts.NowMillis
	if nowFn == nil {
		nowFn = domain.NowMillis
	}
	reg := opts.Registry
	if reg == nil {
		reg = NewStaticActionRegistry()
	}
	validator := opts.Validator
	if validator == nil {
		validator = NewDeterministicValidator()
	}
	caps := opts.Capabilities
	if caps == nil {
		caps = NewStaticCapabilityService()
	}
	approval := opts.ApprovalGate
	if approval == nil {
		approval = NewStaticApprovalGate()
	}
	store := NewInMemorySemanticStore()
	tx := opts.TxRunner
	if tx == nil {
		tx = NewInMemoryTransactionRunner(store)
	}
	return &Processor{
		registry:     reg,
		validator:    validator,
		capabilities: caps,
		approvalGate: approval,
		txRunner:     tx,
		auditSink:    opts.AuditSink,
		nowMillis:    nowFn,
	}
}

func (p *Processor) Process(ctx context.Context, req domain.SyscallRequest) (domain.SyscallResult, error) {
	req = p.normalize(req)
	result := domain.SyscallResult{
		Success:            false,
		Action:             req.Action,
		RequestID:          req.ID,
		CorrelationID:      req.CorrelationID,
		TraceID:            req.TraceID,
		IdempotencyKey:     req.IdempotencyKey,
		DryRun:             req.DryRun,
		ApprovalStatus:     domain.ApprovalAllowed,
		CommittedObjectIDs: []string{},
		RejectedReasons:    []domain.SyscallError{},
		Warnings:           []string{},
		ValidationDetails:  []domain.ValidationDetail{},
		StateSummary:       map[string]any{},
	}

	def, ok := p.registry.Get(req.Action)
	if !ok {
		return p.reject(ctx, req, result, "registry", []domain.SyscallError{
			{Code: domain.ErrUnsupportedAction, Field: "action", Message: "unsupported action"},
		}), nil
	}

	envelopeIssues := p.validator.ValidateEnvelope(req, def)
	result.ValidationDetails = append(result.ValidationDetails, detailFor("envelope_validation", envelopeIssues))
	if len(envelopeIssues) > 0 {
		return p.reject(ctx, req, result, "envelope_validation", envelopeIssues), nil
	}

	allowed, reason, capErr := p.capabilities.HasCapability(ctx, req, def.Capability)
	if capErr != nil {
		return p.reject(ctx, req, result, "capability_validation", []domain.SyscallError{
			{Code: domain.ErrInternal, Field: "capability", Message: capErr.Error()},
		}), capErr
	}
	if !allowed {
		return p.reject(ctx, req, result, "capability_validation", []domain.SyscallError{
			{Code: domain.ErrCapabilityDenied, Field: "capability", Message: reason},
		}), nil
	}
	result.ValidationDetails = append(result.ValidationDetails, detailFor("capability_validation", nil))

	approval, err := p.approvalGate.Evaluate(ctx, req, def)
	if err != nil {
		return p.reject(ctx, req, result, "approval_gate", []domain.SyscallError{
			{Code: domain.ErrInternal, Field: "approval", Message: err.Error()},
		}), err
	}
	result.ApprovalStatus = approval.Status
	if approval.Status == domain.ApprovalRequired {
		return p.reject(ctx, req, result, "approval_gate", []domain.SyscallError{
			{Code: domain.ErrApprovalRequired, Field: "approval", Message: approval.Reason},
		}), nil
	}
	if approval.Status == domain.ApprovalDenied {
		return p.reject(ctx, req, result, "approval_gate", []domain.SyscallError{
			{Code: domain.ErrUnauthorized, Field: "approval", Message: approval.Reason},
		}), nil
	}
	result.ValidationDetails = append(result.ValidationDetails, detailFor("approval_gate", nil))

	if strings.TrimSpace(req.IdempotencyKey) != "" {
		if existing, ok := p.txRunner.ReadStore().GetIdempotency(req.IdempotencyKey); ok {
			result.ValidationDetails = append(result.ValidationDetails, detailFor("idempotency", nil))
			if existing.Action != req.Action {
				return p.reject(ctx, req, result, "idempotency", []domain.SyscallError{
					{Code: domain.ErrDuplicate, Field: "idempotencyKey", Message: "idempotency key already used for a different action"},
				}), nil
			}
			replayed := existing.Result
			replayed.Warnings = append(replayed.Warnings, "idempotent replay")
			replayed.RequestID = req.ID
			replayed.CorrelationID = req.CorrelationID
			replayed.TraceID = req.TraceID
			replayed.IdempotencyKey = req.IdempotencyKey
			replayed.AuditID = p.writeAudit(ctx, req, replayed)
			return replayed, nil
		}
	}
	result.ValidationDetails = append(result.ValidationDetails, detailFor("idempotency", nil))

	payloadIssues := p.validator.ValidatePayload(req, def, p.txRunner.ReadStore())
	result.ValidationDetails = append(result.ValidationDetails, detailFor("payload_validation", payloadIssues))
	if len(payloadIssues) > 0 {
		return p.reject(ctx, req, result, "payload_validation", payloadIssues), nil
	}

	if req.DryRun {
		result.Success = true
		result.StateSummary["dryRun"] = true
		result.Warnings = append(result.Warnings, "dry-run: request validated without commit")
		result.AuditID = p.writeAudit(ctx, req, result)
		return result, nil
	}

	var commitIDs []string
	var commitWarnings []string
	var summary map[string]any
	err = p.txRunner.Run(ctx, func(uow UnitOfWork) error {
		store := uow.Store()
		if aware, ok := store.(CommitAwareStore); ok {
			aware.SetCommitMetadata(CommitMetadata{
				SyscallID:     req.ID,
				CorrelationID: req.CorrelationID,
				TraceID:       req.TraceID,
				Source:        req.Source,
				ActorID:       req.Actor.ID,
				ActorKind:     req.Actor.Kind,
				CommittedBy:   "forge_kernel",
			})
		}
		var applyErrs []domain.SyscallError
		commitIDs, summary, commitWarnings, applyErrs = p.apply(ctx, store, req, def)
		if len(applyErrs) > 0 {
			return commitError{Issues: applyErrs}
		}
		if journalStore, ok := store.(interface {
			Append(ctx context.Context, evt domain.JournalEvent) error
		}); ok {
			evt := domain.JournalEvent{
				ID:        req.ID + ":journal_event",
				Type:      "semantic_syscall." + strings.ToLower(string(req.Action)),
				Timestamp: req.RequestedAt,
				Source:    "forge_kernel",
				Scope:     req.Scope,
				Payload: map[string]any{
					"action":             req.Action,
					"committedObjectIds": append([]string{}, commitIDs...),
					"dryRun":             false,
				},
				CorrelationID: req.CorrelationID,
				Provenance:    req.Provenance,
			}
			if evt.Provenance.Source == "" {
				evt.Provenance.Source = string(req.Source)
			}
			if evt.Provenance.TraceID == "" {
				evt.Provenance.TraceID = req.TraceID
			}
			if err := journalStore.Append(ctx, evt); err != nil {
				return commitError{Issues: []domain.SyscallError{{
					Code:    domain.ErrPersistenceUnavailable,
					Field:   "journal_events",
					Message: err.Error(),
				}}}
			}
			commitIDs = append(commitIDs, evt.ID)
		}
		if strings.TrimSpace(req.IdempotencyKey) != "" {
			uow.Store().SetIdempotency(req.IdempotencyKey, IdempotencyRecord{
				Action: req.Action,
				Result: domain.SyscallResult{
					Success:            true,
					Action:             req.Action,
					RequestID:          req.ID,
					CorrelationID:      req.CorrelationID,
					TraceID:            req.TraceID,
					IdempotencyKey:     req.IdempotencyKey,
					DryRun:             false,
					ApprovalStatus:     domain.ApprovalAllowed,
					CommittedObjectIDs: append([]string{}, commitIDs...),
					Warnings:           append([]string{}, commitWarnings...),
					ValidationDetails:  []domain.ValidationDetail{},
					StateSummary:       cloneMap(summary),
				},
			})
		}
		return nil
	})
	if err != nil {
		if ce, ok := err.(commitError); ok {
			return p.reject(ctx, req, result, "commit", ce.Issues), nil
		}
		return p.reject(ctx, req, result, "commit", []domain.SyscallError{
			{Code: domain.ErrInternal, Field: "commit", Message: err.Error()},
		}), err
	}

	result.Success = true
	result.CommittedObjectIDs = commitIDs
	result.Warnings = append(result.Warnings, commitWarnings...)
	result.StateSummary = summary
	result.ValidationDetails = append(result.ValidationDetails, detailFor("commit", nil))
	result.AuditID = p.writeAudit(ctx, req, result)
	if result.AuditID != "" {
		if linker, ok := p.txRunner.(auditLinkingRunner); ok {
			if err := linker.LinkAudit(ctx, req.CorrelationID, req.ID, result.AuditID); err != nil {
				result.Warnings = append(result.Warnings, "audit linkage update failed: "+err.Error())
			}
		}
	}
	return result, nil
}

type commitError struct {
	Issues []domain.SyscallError
}

func (e commitError) Error() string {
	if len(e.Issues) == 0 {
		return "commit failed"
	}
	return e.Issues[0].Message
}

func (p *Processor) reject(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult, layer string, issues []domain.SyscallError) domain.SyscallResult {
	if layer != "" {
		result.ValidationDetails = append(result.ValidationDetails, detailFor(layer, issues))
	}
	result.Success = false
	result.RejectedReasons = append(result.RejectedReasons, issues...)
	if len(issues) > 0 {
		result.DeterministicErrCode = issues[0].Code
	}
	result.StateSummary["dryRun"] = req.DryRun
	result.AuditID = p.writeAudit(ctx, req, result)
	return result
}

func (p *Processor) writeAudit(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) string {
	if p.auditSink == nil {
		return ""
	}
	id, _ := p.auditSink.Record(ctx, SyscallAuditRecord{
		Timestamp:        p.nowMillis(),
		Action:           req.Action,
		Actor:            req.Actor.ID,
		Source:           req.Source,
		WorkspaceID:      req.Scope.WorkspaceID,
		RequestID:        req.ID,
		CorrelationID:    req.CorrelationID,
		TraceID:          req.TraceID,
		DryRun:           req.DryRun,
		Success:          result.Success,
		ApprovalStatus:   result.ApprovalStatus,
		ValidationIssues: result.RejectedReasons,
		CommittedIDs:     result.CommittedObjectIDs,
		ErrorCode:        result.DeterministicErrCode,
	})
	return id
}

func detailFor(layer string, issues []domain.SyscallError) domain.ValidationDetail {
	return domain.ValidationDetail{
		Layer:  layer,
		Passed: len(issues) == 0,
		Issues: append([]domain.SyscallError{}, issues...),
	}
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
