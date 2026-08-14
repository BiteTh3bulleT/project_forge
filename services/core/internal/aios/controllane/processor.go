package controllane

import (
	"context"
	"errors"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/aios/rulecells"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/commitproof"
	"forge/projectforge/services/core/internal/forgekshadow"
)

type ControlLaneValidationObserver interface {
	ObserveControlLaneValidationBestEffort(ctx context.Context, input forgekshadow.ControlLaneValidationInput)
}

type ProcessorOptions struct {
	Registry                      ActionRegistry
	Validator                     SemanticActionValidator
	Capabilities                  CapabilityService
	ApprovalGate                  ApprovalGate
	TxRunner                      TransactionRunner
	AuditSink                     AuditSink
	NowMillis                     func() int64
	RuleEngine                    RuleEngine
	KVIdentityMetrics             KVIdentityEnforcementMetrics
	ControlLaneValidationObserver ControlLaneValidationObserver
}

type Processor struct {
	registry                      ActionRegistry
	validator                     SemanticActionValidator
	capabilities                  CapabilityService
	approvalGate                  ApprovalGate
	txRunner                      TransactionRunner
	auditSink                     AuditSink
	nowMillis                     func() int64
	ruleEngine                    RuleEngine
	kvIdentityMetrics             KVIdentityEnforcementMetrics
	controlLaneValidationObserver ControlLaneValidationObserver
}

var (
	_ forgekernel.Processor   = (*Processor)(nil)
	_ forgekernel.DurablePort = (*Processor)(nil)
)

type RuleEngine interface {
	Run(ctx context.Context, in rulecells.RunInput, opts rulecells.RunOptions) (rulecells.RunResult, error)
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
		registry:                      reg,
		validator:                     validator,
		capabilities:                  caps,
		approvalGate:                  approval,
		txRunner:                      tx,
		auditSink:                     opts.AuditSink,
		nowMillis:                     nowFn,
		ruleEngine:                    opts.RuleEngine,
		kvIdentityMetrics:             opts.KVIdentityMetrics,
		controlLaneValidationObserver: opts.ControlLaneValidationObserver,
	}
}

// Process is the legacy_v1 rollback orchestration. Production FORGE-K calls
// Prepare, Commit, RecordResult, and ObserveResult itself through DurablePort.
func (p *Processor) Process(ctx context.Context, req domain.SyscallRequest) (domain.SyscallResult, error) {
	prepared, err := p.Prepare(ctx, req)
	if err != nil || prepared.Disposition == forgekernel.DispositionComplete {
		result := p.RecordResult(ctx, prepared.Request, prepared.Result)
		p.ObserveResult(ctx, prepared.Request, result)
		return result, err
	}
	if prepared.Disposition == forgekernel.DispositionReplay {
		result, replayErr := verifyLegacyReplay(prepared)
		result = p.RecordResult(ctx, prepared.Request, result)
		p.ObserveResult(ctx, prepared.Request, result)
		return result, replayErr
	}
	boundPlan, bindErr := commitproof.BindPreparedPlan(prepared.Request, prepared.Plan)
	if bindErr != nil {
		result := p.rejectResult(prepared.Request, prepared.Result, "legacy_commit_proof", []domain.SyscallError{{
			Code: domain.ErrInternal, Field: "kernel.commitPlan", Message: bindErr.Error(),
		}})
		result = p.RecordResult(ctx, prepared.Request, result)
		p.ObserveResult(ctx, prepared.Request, result)
		return result, bindErr
	}
	prepared.Plan = boundPlan
	seal, sealErr := commitproof.SealPreparedPlan(prepared.Request, prepared.Plan)
	if sealErr != nil {
		result := p.rejectResult(prepared.Request, prepared.Result, "legacy_commit_proof", []domain.SyscallError{{
			Code: domain.ErrInternal, Field: "kernel.commitProof", Message: sealErr.Error(),
		}})
		result = p.RecordResult(ctx, prepared.Request, result)
		p.ObserveResult(ctx, prepared.Request, result)
		return result, sealErr
	}
	outcome, err := p.Commit(ctx, prepared, seal)
	result := outcome.Result
	result = p.RecordResult(ctx, prepared.Request, result)
	p.ObserveResult(ctx, prepared.Request, result)
	return result, err
}

// Prepare performs deterministic normalization, registry, capability,
// approval, idempotency, payload, and validation-lane checks without opening a
// write transaction. It returns either a completed result or a commit request.
func (p *Processor) Prepare(ctx context.Context, req domain.SyscallRequest) (forgekernel.PreparedSyscall, error) {
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
	complete := func(result domain.SyscallResult, err error) (forgekernel.PreparedSyscall, error) {
		return forgekernel.PreparedSyscall{Request: req, Result: result, Disposition: forgekernel.DispositionComplete}, err
	}
	reject := func(layer string, issues []domain.SyscallError, err error) (forgekernel.PreparedSyscall, error) {
		return complete(p.rejectResult(req, result, layer, issues), err)
	}

	def, ok := p.registry.Get(req.Action)
	if !ok {
		return reject("registry", []domain.SyscallError{
			{Code: domain.ErrUnsupportedAction, Field: "action", Message: "unsupported action"},
		}, nil)
	}

	productionAuthorized := forgekernel.HasVerifiedAuthorizationContext(ctx) && hasKernelAuthorizationMetadata(req)
	envelopeIssues := p.validator.ValidateEnvelope(req, def)
	if isForgeKOnlyAction(req.Action) && !productionAuthorized {
		envelopeIssues = append(envelopeIssues, domain.SyscallError{
			Code: domain.ErrUnauthorized, Field: "kernel.authority",
			Message: "action requires production FORGE-K orchestration",
		})
	}
	result.ValidationDetails = append(result.ValidationDetails, detailFor("envelope_validation", envelopeIssues))
	if len(envelopeIssues) > 0 {
		return reject("envelope_validation", envelopeIssues, nil)
	}

	if productionAuthorized {
		// Production identity/capability/approval authority was already resolved
		// and verified by FORGE-K. Static services remain legacy_v1/test policy
		// only and cannot override the production proof.
		result.ApprovalStatus = domain.ApprovalAllowed
		result.ValidationDetails = append(result.ValidationDetails,
			detailFor("forge_k_capability_proof", nil), detailFor("forge_k_approval_proof", nil))
	} else {
		allowed, reason, capErr := p.capabilities.HasCapability(ctx, req, def.Capability)
		if capErr != nil {
			return reject("capability_validation", []domain.SyscallError{
				{Code: domain.ErrInternal, Field: "capability", Message: capErr.Error()},
			}, capErr)
		}
		if !allowed {
			return reject("capability_validation", []domain.SyscallError{
				{Code: domain.ErrCapabilityDenied, Field: "capability", Message: reason},
			}, nil)
		}
		result.ValidationDetails = append(result.ValidationDetails, detailFor("capability_validation", nil))

		approval, err := p.approvalGate.Evaluate(ctx, req, def)
		if err != nil {
			return reject("approval_gate", []domain.SyscallError{
				{Code: domain.ErrInternal, Field: "approval", Message: err.Error()},
			}, err)
		}
		result.ApprovalStatus = approval.Status
		if approval.Status == domain.ApprovalRequired {
			return reject("approval_gate", []domain.SyscallError{
				{Code: domain.ErrApprovalRequired, Field: "approval", Message: approval.Reason},
			}, nil)
		}
		if approval.Status == domain.ApprovalDenied {
			return reject("approval_gate", []domain.SyscallError{
				{Code: domain.ErrUnauthorized, Field: "approval", Message: approval.Reason},
			}, nil)
		}
		result.ValidationDetails = append(result.ValidationDetails, detailFor("approval_gate", nil))
	}

	if strings.TrimSpace(req.IdempotencyKey) != "" {
		if existing, ok := p.txRunner.ReadStore().GetIdempotency(req.IdempotencyKey); ok {
			result.ValidationDetails = append(result.ValidationDetails, detailFor("idempotency", nil))
			fingerprint, fingerprintErr := commitproof.IdempotencyFingerprint(req)
			if fingerprintErr != nil {
				return reject("idempotency", []domain.SyscallError{{
					Code: domain.ErrInternal, Field: "idempotencyKey", Message: fingerprintErr.Error(),
				}}, fingerprintErr)
			}
			if existing.Action != req.Action || existing.IdempotencyFingerprint != fingerprint {
				return reject("idempotency", []domain.SyscallError{
					{Code: domain.ErrDuplicate, Field: "idempotencyKey", Message: "idempotency key already binds a different semantic request"},
				}, nil)
			}
			return forgekernel.PreparedSyscall{
				Request:                  req,
				Result:                   existing.Result,
				Disposition:              forgekernel.DispositionReplay,
				ReplayRequest:            existing.Request,
				ReplayPlan:               existing.Plan,
				ReplaySeal:               existing.Seal,
				ReplayReceipt:            existing.Receipt,
				ReplayAuthorizationProof: existing.AuthorizationProof,
			}, nil
		}
	}
	result.ValidationDetails = append(result.ValidationDetails, detailFor("idempotency", nil))

	var kvIdentityDecision KVIdentityEnforcementDecision
	if req.Action == domain.ActionValidateKVIdentity {
		kvIdentityDecision = EnforceKVIdentity(req)
		if !kvIdentityDecision.Accepted {
			result.StateSummary = kvIdentityDecision.ToStateSummary()
			p.recordKVIdentityDecision(kvIdentityDecision)
			return reject("kv_identity_enforcement", []domain.SyscallError{kvIdentityDecision.ToSyscallError()}, nil)
		}
		result.ValidationDetails = append(result.ValidationDetails, detailFor("kv_identity_enforcement", nil))
	}

	payloadIssues := p.validator.ValidatePayload(req, def, p.txRunner.ReadStore())
	result.ValidationDetails = append(result.ValidationDetails, detailFor("payload_validation", payloadIssues))
	if len(payloadIssues) > 0 {
		return reject("payload_validation", payloadIssues, nil)
	}

	var semanticOperationDecision SemanticOperationValidationDecision
	var admissionCandidateDecision AdmissionValidationDecision
	var contextAttributionDecision ContextAttributionValidationDecision
	var refShapeDecision RefShapeValidationDecision
	if req.Action == domain.ActionValidateRefShape {
		refShapeDecision = EnforceRefShape(req)
		if !refShapeDecision.Accepted {
			result.StateSummary = refShapeDecision.ToStateSummary()
			return reject("ref_shape_validation", []domain.SyscallError{refShapeDecision.ToSyscallError()}, nil)
		}
		result.ValidationDetails = append(result.ValidationDetails, detailFor("ref_shape_validation", nil))
	}
	var refShapeComparisonDecision RefShapeComparisonDecision
	if req.Action == domain.ActionCompareRefShape {
		refShapeComparisonDecision = EnforceRefShapeComparison(req)
		if !refShapeComparisonDecision.Accepted {
			result.StateSummary = refShapeComparisonDecision.ToStateSummary()
			return reject("ref_shape_comparison", []domain.SyscallError{refShapeComparisonDecision.ToSyscallError()}, nil)
		}
		result.ValidationDetails = append(result.ValidationDetails, detailFor("ref_shape_comparison", nil))
	}
	var sourceObjectAuthorityDecision SourceObjectAuthorityDecision
	if req.Action == domain.ActionValidateSourceObject {
		sourceObjectAuthorityDecision = EnforceSourceObjectAuthority(req, p.txRunner.ReadStore())
		if !sourceObjectAuthorityDecision.Accepted {
			result.StateSummary = sourceObjectAuthorityDecision.ToStateSummary()
			return reject("source_object_authority_validation", []domain.SyscallError{sourceObjectAuthorityDecision.ToSyscallError()}, nil)
		}
		result.ValidationDetails = append(result.ValidationDetails, detailFor("source_object_authority_validation", nil))
	}
	if req.Action == domain.ActionValidateSemanticOperation {
		semanticOperationDecision = EnforceSemanticOperation(req)
		if !semanticOperationDecision.Accepted {
			result.StateSummary = semanticOperationDecision.ToStateSummary()
			return reject("semantic_operation_validation", []domain.SyscallError{semanticOperationDecision.ToSyscallError()}, nil)
		}
		result.ValidationDetails = append(result.ValidationDetails, detailFor("semantic_operation_validation", nil))
	}
	if req.Action == domain.ActionValidateAdmissionCandidate {
		admissionCandidateDecision = EnforceAdmissionCandidate(req)
		if !admissionCandidateDecision.Accepted {
			result.StateSummary = admissionCandidateDecision.ToStateSummary()
			return reject("admission_candidate_validation", []domain.SyscallError{admissionCandidateDecision.ToSyscallError()}, nil)
		}
		result.ValidationDetails = append(result.ValidationDetails, detailFor("admission_candidate_validation", nil))
	}
	if req.Action == domain.ActionValidateContextAttribution {
		contextAttributionDecision = EnforceContextAttribution(req)
		if !contextAttributionDecision.Accepted {
			result.StateSummary = contextAttributionDecision.ToStateSummary()
			return reject("context_attribution_validation", []domain.SyscallError{contextAttributionDecision.ToSyscallError()}, nil)
		}
		result.ValidationDetails = append(result.ValidationDetails, detailFor("context_attribution_validation", nil))
	}

	if req.DryRun {
		result.Success = true
		result.StateSummary["dryRun"] = true
		if req.Action == domain.ActionValidateKVIdentity {
			result.StateSummary = kvIdentityDecision.ToStateSummary()
			result.StateSummary["dryRun"] = true
			p.recordKVIdentityDecision(kvIdentityDecision)
		} else if req.Action == domain.ActionValidateRefShape {
			if refShapeDecision.Decision == "" {
				refShapeDecision = EnforceRefShape(req)
			}
			result.StateSummary = refShapeDecision.ToStateSummary()
			result.StateSummary["dryRun"] = true
		} else if req.Action == domain.ActionCompareRefShape {
			if refShapeComparisonDecision.Decision == "" {
				refShapeComparisonDecision = EnforceRefShapeComparison(req)
			}
			result.StateSummary = refShapeComparisonDecision.ToStateSummary()
			result.StateSummary["dryRun"] = true
		} else if req.Action == domain.ActionValidateSourceObject {
			if sourceObjectAuthorityDecision.Decision == "" {
				sourceObjectAuthorityDecision = EnforceSourceObjectAuthority(req, p.txRunner.ReadStore())
			}
			result.StateSummary = sourceObjectAuthorityDecision.ToStateSummary()
			result.StateSummary["dryRun"] = true
		} else if req.Action == domain.ActionValidateSemanticOperation {
			if semanticOperationDecision.Decision == "" {
				semanticOperationDecision = EnforceSemanticOperation(req)
			}
			result.StateSummary = semanticOperationDecision.ToStateSummary()
			result.StateSummary["dryRun"] = true
		} else if req.Action == domain.ActionValidateAdmissionCandidate {
			if admissionCandidateDecision.Decision == "" {
				admissionCandidateDecision = EnforceAdmissionCandidate(req)
			}
			result.StateSummary = admissionCandidateDecision.ToStateSummary()
			result.StateSummary["dryRun"] = true
		} else if req.Action == domain.ActionValidateContextAttribution {
			if contextAttributionDecision.Decision == "" {
				contextAttributionDecision = EnforceContextAttribution(req)
			}
			result.StateSummary = contextAttributionDecision.ToStateSummary()
			result.StateSummary["dryRun"] = true
		}
		result.Warnings = append(result.Warnings, "dry-run: request validated without commit")
		return complete(result, nil)
	}
	plan, planErr := buildPreparedCommitPlan(req, def, p.txRunner.ReadStore())
	if planErr != nil {
		return reject("commit_plan", []domain.SyscallError{{
			Code: domain.ErrInternal, Field: "kernel.commitPlan", Message: planErr.Error(),
		}}, planErr)
	}
	return forgekernel.PreparedSyscall{Request: req, Result: result, Disposition: forgekernel.DispositionCommit, Plan: plan}, nil
}

func isForgeKOnlyAction(action domain.SemanticActionType) bool {
	switch action {
	case domain.ActionAdmitEvidence,
		domain.ActionAppealRuling,
		domain.ActionRecordRetrievalEvidence,
		domain.ActionRebuildMemoryAcceleration,
		domain.ActionRecordRetrievalUsefulness,
		domain.ActionRecordRestoreOutcomeFeedback:
		return true
	default:
		return false
	}
}

func hasKernelAuthorizationMetadata(req domain.SyscallRequest) bool {
	owner, _ := req.Metadata["kernelAuthorityOwner"].(string)
	ingress, _ := req.Metadata["forgeKIngressAuthority"].(bool)
	fingerprint, _ := req.Metadata["forgeKAuthorizationProof"].(string)
	return owner == forgekernel.AuthorityOwnerForgeK && ingress && strings.HasPrefix(fingerprint, "sha256:")
}

// Commit performs one atomic apply + journal + idempotency transaction. It is
// invoked by production FORGE-K only after Prepare returns DispositionCommit.
func (p *Processor) Commit(ctx context.Context, prepared forgekernel.PreparedSyscall, seal commitproof.PreparedPlanSeal) (forgekernel.CommitOutcome, error) {
	req := prepared.Request
	result := prepared.Result
	rejectOutcome := func(layer string, issues []domain.SyscallError) forgekernel.CommitOutcome {
		return forgekernel.CommitOutcome{Result: p.rejectResult(req, result, layer, issues)}
	}
	if prepared.Disposition != forgekernel.DispositionCommit {
		return rejectOutcome("commit_disposition", []domain.SyscallError{{
			Code: domain.ErrInternal, Field: "kernel.disposition", Message: "durable commit called without commit disposition",
		}}), nil
	}
	def, ok := p.registry.Get(req.Action)
	if !ok {
		return rejectOutcome("commit_registry", []domain.SyscallError{{
			Code: domain.ErrUnsupportedAction, Field: "action", Message: "unsupported action at commit",
		}}), nil
	}
	if err := commitproof.VerifyPreparedPlan(req, prepared.Plan, seal); err != nil {
		return rejectOutcome("commit_proof", []domain.SyscallError{{
			Code: domain.ErrInternal, Field: "kernel.commitProof", Message: err.Error(),
		}}), errors.Join(forgekernel.ErrInvalidPreparedProof, err)
	}
	var commitIDs []string
	var commitWarnings []string
	var summary map[string]any
	var receipt commitproof.CommitReceipt
	err := p.txRunner.Run(ctx, func(uow UnitOfWork) error {
		store := uow.Store()
		if aware, ok := store.(CommitAwareStore); ok {
			aware.SetCommitMetadata(CommitMetadata{
				SyscallID:     req.ID,
				CorrelationID: req.CorrelationID,
				TraceID:       req.TraceID,
				Source:        req.Source,
				ActorID:       req.Actor.ID,
				ActorKind:     req.Actor.Kind,
				CommittedBy:   prepared.Plan.ExpectedJournalCommittedBy,
			})
		}
		var applyErrs []domain.SyscallError
		commitIDs, summary, commitWarnings, applyErrs = p.apply(ctx, store, req, def)
		if len(applyErrs) > 0 {
			return commitError{Issues: applyErrs}
		}
		journalStore, ok := store.(interface {
			AppendWithEvidence(context.Context, domain.JournalEvent) (JournalAppendEvidence, error)
		})
		if !ok {
			return commitError{Issues: []domain.SyscallError{{
				Code: domain.ErrPersistenceUnavailable, Field: "journal_events", Message: "durable journal evidence adapter is required",
			}}}
		}
		auditOutboxID := prepared.Plan.ExpectedAuditOutboxID
		transactionID := prepared.Plan.ExpectedTransactionID
		journalPayload, payloadErr := commitproof.BuildJournalPayload(req, prepared.Plan)
		if payloadErr != nil {
			return commitError{Issues: []domain.SyscallError{{
				Code: domain.ErrInternal, Field: "journal_events.payload", Message: payloadErr.Error(),
			}}}
		}
		evt := domain.JournalEvent{
			ID:            prepared.Plan.ExpectedJournalEventID,
			Type:          prepared.Plan.JournalEventType,
			Timestamp:     req.RequestedAt,
			Source:        prepared.Plan.ExpectedJournalSource,
			Scope:         req.Scope,
			Payload:       journalPayload,
			CorrelationID: req.CorrelationID,
			Provenance:    commitproof.BuildJournalProvenance(req),
		}
		journalEvidence, journalErr := journalStore.AppendWithEvidence(ctx, evt)
		if journalErr != nil {
			return commitError{Issues: []domain.SyscallError{{
				Code: domain.ErrPersistenceUnavailable, Field: "journal_events", Message: journalErr.Error(),
			}}}
		}
		if prepared.Plan.Mutating {
			commitIDs = append(commitIDs, evt.ID)
		}
		idempotencyFingerprint, fingerprintErr := commitproof.IdempotencyFingerprint(req)
		if fingerprintErr != nil {
			return commitError{Issues: []domain.SyscallError{{
				Code: domain.ErrInternal, Field: "idempotencyFingerprint", Message: fingerprintErr.Error(),
			}}}
		}
		receipt = commitproof.CommitReceipt{
			Version:                commitproof.CommitReceiptVersion,
			RequestFingerprint:     seal.RequestFingerprint,
			PreparedPlanSeal:       seal.SealDigest,
			TransactionID:          transactionID,
			JournalEventID:         journalEvidence.Entry.EventID,
			JournalEventHash:       journalEvidence.Entry.Hash,
			ObjectIDs:              append([]string{}, commitIDs...),
			ProvenanceIDs:          []string{journalEvidence.Entry.ProvenanceID},
			AuditOutboxID:          auditOutboxID,
			IdempotencyFingerprint: idempotencyFingerprint,
			JournalEntry:           journalEvidence.Entry,
		}
		committedResult := result
		committedResult.Success = true
		committedResult.CommittedObjectIDs = append([]string{}, commitIDs...)
		committedResult.Warnings = append(committedResult.Warnings, commitWarnings...)
		committedResult.StateSummary = cloneMap(summary)
		committedResult.ValidationDetails = append(committedResult.ValidationDetails, detailFor("commit", nil))
		if proofErr := commitproof.ValidateCommitReceipt(req, prepared.Plan, seal, receipt, committedResult); proofErr != nil {
			return commitError{Issues: []domain.SyscallError{{
				Code: domain.ErrPersistenceUnavailable, Field: "commit_receipt", Message: proofErr.Error(),
			}}}
		}
		if outboxErr := store.CreateAuditOutbox(AuditOutboxRecord{
			ID: auditOutboxID, SyscallID: req.ID, RequestFingerprint: seal.RequestFingerprint,
			Action: req.Action, WorkspaceID: req.Scope.WorkspaceID, LaneID: req.Scope.LaneID,
			CorrelationID: req.CorrelationID, TraceID: req.TraceID, Success: true,
			Result: committedResult, Request: req, Receipt: receipt, AuthorizationProof: prepared.AuthorizationProof,
			CreatedAt: req.RequestedAt, CommittedBy: prepared.Plan.ExpectedJournalCommittedBy,
		}); outboxErr != nil {
			return commitError{Issues: []domain.SyscallError{{
				Code: domain.ErrPersistenceUnavailable, Field: "audit_outbox", Message: outboxErr.Error(),
			}}}
		}
		if strings.TrimSpace(req.IdempotencyKey) != "" {
			if idempotencyErr := store.SetIdempotency(req.IdempotencyKey, IdempotencyRecord{
				Action: req.Action, RequestFingerprint: seal.RequestFingerprint,
				IdempotencyFingerprint: idempotencyFingerprint, Result: committedResult,
				Request: req, Plan: prepared.Plan, Seal: seal, Receipt: receipt,
				AuthorizationProof: prepared.AuthorizationProof,
				CreatedAt:          req.RequestedAt, CorrelationID: req.CorrelationID,
			}); idempotencyErr != nil {
				return commitError{Issues: []domain.SyscallError{{
					Code: domain.ErrPersistenceUnavailable, Field: "idempotency", Message: idempotencyErr.Error(),
				}}}
			}
		}
		result = committedResult
		return nil
	})
	if err != nil {
		if ce, ok := err.(commitError); ok {
			return rejectOutcome("commit", ce.Issues), nil
		}
		return rejectOutcome("commit", []domain.SyscallError{
			{Code: domain.ErrInternal, Field: "commit", Message: err.Error()},
		}), err
	}

	if req.Action == domain.ActionValidateKVIdentity {
		p.recordKVIdentityDecision(EnforceKVIdentity(req))
	}
	return forgekernel.CommitOutcome{Result: result, Receipt: receipt}, nil
}

// verifyLegacyReplay keeps the explicit rollback mode compatible without
// trusting an unproved cached result. Production requests use Kernel.replay;
// this path exists only until legacy_v1 is retired.
func verifyLegacyReplay(prepared forgekernel.PreparedSyscall) (domain.SyscallResult, error) {
	currentFingerprint, err := commitproof.IdempotencyFingerprint(prepared.Request)
	if err == nil {
		err = commitproof.VerifyPreparedPlan(prepared.ReplayRequest, prepared.ReplayPlan, prepared.ReplaySeal)
	}
	if err == nil {
		err = commitproof.ValidateCommitReceipt(
			prepared.ReplayRequest,
			prepared.ReplayPlan,
			prepared.ReplaySeal,
			prepared.ReplayReceipt,
			prepared.Result,
		)
	}
	if err == nil && currentFingerprint != prepared.ReplayReceipt.IdempotencyFingerprint {
		err = commitproof.ErrReceiptMismatch
	}
	if err != nil {
		result := prepared.Result
		result.Success = false
		result.RejectedReasons = append(result.RejectedReasons, domain.SyscallError{
			Code: domain.ErrPersistenceUnavailable, Field: "idempotencyKey", Message: "verified idempotent replay failed: " + err.Error(),
		})
		result.DeterministicErrCode = domain.ErrPersistenceUnavailable
		result.ValidationDetails = append(result.ValidationDetails, detailFor("legacy_replay_proof", result.RejectedReasons[len(result.RejectedReasons)-1:]))
		return result, errors.Join(forgekernel.ErrInvalidCommitReceipt, err)
	}
	result := prepared.Result
	result.RequestID = prepared.Request.ID
	result.CorrelationID = prepared.Request.CorrelationID
	result.TraceID = prepared.Request.TraceID
	result.IdempotencyKey = prepared.Request.IdempotencyKey
	if !containsExactWarning(result.Warnings, "idempotent replay") {
		result.Warnings = append(result.Warnings, "idempotent replay")
	}
	return result, nil
}

func containsExactWarning(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (p *Processor) recordKVIdentityDecision(decision KVIdentityEnforcementDecision) {
	if p == nil || p.kvIdentityMetrics == nil {
		return
	}
	p.kvIdentityMetrics.Record(decision)
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

func (p *Processor) rejectResult(req domain.SyscallRequest, result domain.SyscallResult, layer string, issues []domain.SyscallError) domain.SyscallResult {
	if layer != "" {
		result.ValidationDetails = append(result.ValidationDetails, detailFor(layer, issues))
	}
	result.Success = false
	result.RejectedReasons = append(result.RejectedReasons, issues...)
	if len(issues) > 0 {
		result.DeterministicErrCode = issues[0].Code
	}
	result.StateSummary["dryRun"] = req.DryRun
	return result
}

// RecordResult projects a terminal result into the external audit service.
// Successful commits already contain an immutable audit-outbox intent and
// typed receipt in their canonical transaction; this best-effort projection
// may subsequently backfill the external audit id onto the transaction lineage.
func (p *Processor) RecordResult(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) domain.SyscallResult {
	result.AuditID = p.writeAudit(ctx, req, result)
	if result.Success && !req.DryRun && result.AuditID != "" {
		if linker, ok := p.txRunner.(auditLinkingRunner); ok {
			if err := linker.LinkAudit(ctx, req.CorrelationID, req.ID, result.AuditID); err != nil {
				result.Warnings = append(result.Warnings, "audit linkage update failed: "+err.Error())
			}
		}
	}
	return result
}

// ObserveResult exposes bounded best-effort validation metadata after the
// authoritative result is final. Observers cannot change that result.
func (p *Processor) ObserveResult(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) {
	p.observeControlLaneValidation(ctx, req, result)
}

func (p *Processor) writeAudit(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) string {
	if p.auditSink == nil {
		return ""
	}
	var facade map[string]any
	if p.registry != nil {
		if def, ok := p.registry.Get(req.Action); ok {
			facade = BuildSemanticSyscallFacade(req, def).ToAuditFields()
		}
	}
	id, _ := p.auditSink.Record(ctx, SyscallAuditRecord{
		Timestamp:                    p.nowMillis(),
		Action:                       req.Action,
		Actor:                        req.Actor.ID,
		Source:                       req.Source,
		WorkspaceID:                  req.Scope.WorkspaceID,
		RequestID:                    req.ID,
		CorrelationID:                req.CorrelationID,
		TraceID:                      req.TraceID,
		DryRun:                       req.DryRun,
		Success:                      result.Success,
		ApprovalStatus:               result.ApprovalStatus,
		ValidationIssues:             result.RejectedReasons,
		CommittedIDs:                 result.CommittedObjectIDs,
		ErrorCode:                    result.DeterministicErrCode,
		KVIdentityEnforcement:        kvIdentityAuditFields(result.StateSummary),
		RefShapeValidation:           refShapeAuditFields(result.StateSummary),
		RefShapeComparison:           refShapeComparisonAuditFields(result.StateSummary),
		SourceObjectAuthority:        sourceObjectAuthorityAuditFields(result.StateSummary),
		SemanticOperationValidation:  semanticOperationAuditFields(result.StateSummary),
		AdmissionCandidateValidation: admissionCandidateAuditFields(result.StateSummary),
		ContextAttributionValidation: contextAttributionAuditFields(result.StateSummary),
		SemanticSyscallEnvelope:      facade,
	})
	return id
}

func kvIdentityAuditFields(summary map[string]any) map[string]any {
	if summary == nil {
		return nil
	}
	fields, _ := summary["kvIdentityEnforcement"].(map[string]any)
	if fields == nil {
		return nil
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		out[key] = value
	}
	return out
}

func refShapeAuditFields(summary map[string]any) map[string]any {
	if summary == nil {
		return nil
	}
	fields, _ := summary["refShapeValidation"].(map[string]any)
	if fields == nil {
		return nil
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		out[key] = value
	}
	return out
}

func refShapeComparisonAuditFields(summary map[string]any) map[string]any {
	if summary == nil {
		return nil
	}
	fields, _ := summary["refShapeComparison"].(map[string]any)
	if fields == nil {
		return nil
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		out[key] = value
	}
	return out
}

func semanticOperationAuditFields(summary map[string]any) map[string]any {
	if summary == nil {
		return nil
	}
	fields, _ := summary["semanticOperationValidation"].(map[string]any)
	if fields == nil {
		return nil
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		out[key] = value
	}
	return out
}

func sourceObjectAuthorityAuditFields(summary map[string]any) map[string]any {
	if summary == nil {
		return nil
	}
	fields, _ := summary["sourceObjectAuthorityValidation"].(map[string]any)
	if fields == nil {
		return nil
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		out[key] = value
	}
	return out
}

func admissionCandidateAuditFields(summary map[string]any) map[string]any {
	if summary == nil {
		return nil
	}
	fields, _ := summary["admissionCandidateValidation"].(map[string]any)
	if fields == nil {
		return nil
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		out[key] = value
	}
	return out
}

func contextAttributionAuditFields(summary map[string]any) map[string]any {
	if summary == nil {
		return nil
	}
	fields, _ := summary["contextAttributionValidation"].(map[string]any)
	if fields == nil {
		return nil
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		out[key] = value
	}
	return out
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
