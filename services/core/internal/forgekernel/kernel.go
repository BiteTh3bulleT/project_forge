// Package forgekernel owns the production FORGE-K semantic syscall boundary.
//
// The simulator under internal/forgek remains isolated. During the staged live
// cutover this package owns ingress and delegates durable application to the
// existing Control Lane transaction adapter. Exactly one processor is selected
// at boot; shadow execution and dual commits are forbidden.
package forgekernel

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/authproof"
	"forge/projectforge/services/core/internal/forgekernel/commitproof"
	"forge/projectforge/services/core/internal/forgekernel/court"
)

type AuthorityMode string

const (
	ModeForgeK   AuthorityMode = "forge_k"
	ModeLegacyV1 AuthorityMode = "legacy_v1"

	AuthorityOwnerForgeK   = "forge_k.kernel"
	AuthorityOwnerLegacyV1 = "aios.controllane"
	DurableCommitAdapter   = "aios.controllane.sqlite"
)

var (
	ErrInvalidAuthorityMode = errors.New("invalid kernel authority mode")
	ErrMissingCommitAdapter = errors.New("missing durable kernel commit adapter")
	ErrMissingDurablePort   = errors.New("commit adapter does not implement the FORGE-K durable port")
	ErrMissingAuthorization = errors.New("missing production FORGE-K authorization port")
	ErrInvalidDisposition   = errors.New("invalid durable port disposition")
	ErrInvalidAuthorization = errors.New("invalid FORGE-K authorization proof")
	ErrInvalidPreparedProof = errors.New("invalid FORGE-K prepared commit proof")
	ErrInvalidCommitReceipt = errors.New("invalid FORGE-K commit receipt")
)

type Processor interface {
	Process(ctx context.Context, req domain.SyscallRequest) (domain.SyscallResult, error)
}

type Disposition string

const (
	DispositionComplete Disposition = "complete"
	DispositionCommit   Disposition = "commit"
	DispositionReplay   Disposition = "replay"
)

type PreparedSyscall struct {
	Request                  domain.SyscallRequest
	Result                   domain.SyscallResult
	Disposition              Disposition
	Plan                     commitproof.PreparedPlan
	AuthorizationProof       authproof.Proof
	ReplayRequest            domain.SyscallRequest
	ReplayPlan               commitproof.PreparedPlan
	ReplaySeal               commitproof.PreparedPlanSeal
	ReplayReceipt            commitproof.CommitReceipt
	ReplayAuthorizationProof authproof.Proof
}

type CommitOutcome struct {
	Result  domain.SyscallResult
	Receipt commitproof.CommitReceipt
}

// DurablePort is the temporary compatibility boundary implemented by the
// Control Lane adapter. FORGE-K owns the stage order; the port supplies
// non-mutating preflight work, one atomic commit/journal operation, audit
// persistence, and best-effort observation.
type DurablePort interface {
	Prepare(ctx context.Context, req domain.SyscallRequest) (PreparedSyscall, error)
	Commit(ctx context.Context, prepared PreparedSyscall, seal commitproof.PreparedPlanSeal) (CommitOutcome, error)
	RecordResult(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) domain.SyscallResult
	ObserveResult(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult)
}

// AuthorizationPort resolves trusted identity, registry, capability, and
// approval records. It must derive the active principal from trusted process
// construction or request context, never from caller-filled Actor/Source
// fields alone. Kernel validation remains independent of this resolver.
type AuthorizationPort interface {
	ResolveAuthorization(ctx context.Context, req domain.SyscallRequest) (authproof.Proof, error)
}

type Selection struct {
	Processor       Processor
	Mode            AuthorityMode
	AuthorityOwner  string
	CommitAdapter   string
	RollbackMode    AuthorityMode
	SingleAuthority bool
}

type Kernel struct {
	port          DurablePort
	authorization AuthorizationPort
}

type verifiedAuthorizationContextKey struct{}

func withVerifiedAuthorizationContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, verifiedAuthorizationContextKey{}, true)
}

// HasVerifiedAuthorizationContext lets the temporary durable adapter detect
// Kernel-owned production preparation without trusting request metadata.
// Only Kernel.Process can attach the private context key.
func HasVerifiedAuthorizationContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	verified, _ := ctx.Value(verifiedAuthorizationContextKey{}).(bool)
	return verified
}

// SelectAuthority accepts the production authorization port as a variadic
// parameter only to keep the explicit legacy_v1 rollback call source-compatible.
// forge_k requires exactly one non-nil port and otherwise fails closed.
func SelectAuthority(rawMode string, commit Processor, authorization ...AuthorizationPort) (Selection, error) {
	mode, err := ParseAuthorityMode(rawMode)
	if err != nil {
		return Selection{}, err
	}
	if commit == nil {
		return Selection{}, ErrMissingCommitAdapter
	}
	if mode == ModeLegacyV1 {
		return Selection{
			Processor:       commit,
			Mode:            mode,
			AuthorityOwner:  AuthorityOwnerLegacyV1,
			CommitAdapter:   DurableCommitAdapter,
			RollbackMode:    ModeLegacyV1,
			SingleAuthority: true,
		}, nil
	}
	port, ok := commit.(DurablePort)
	if !ok {
		return Selection{}, ErrMissingDurablePort
	}
	if len(authorization) != 1 || authorization[0] == nil {
		return Selection{}, ErrMissingAuthorization
	}
	return Selection{
		Processor:       &Kernel{port: port, authorization: authorization[0]},
		Mode:            ModeForgeK,
		AuthorityOwner:  AuthorityOwnerForgeK,
		CommitAdapter:   DurableCommitAdapter,
		RollbackMode:    ModeLegacyV1,
		SingleAuthority: true,
	}, nil
}

func ParseAuthorityMode(raw string) (AuthorityMode, error) {
	switch AuthorityMode(strings.ToLower(strings.TrimSpace(raw))) {
	case "", ModeForgeK:
		return ModeForgeK, nil
	case ModeLegacyV1:
		return ModeLegacyV1, nil
	default:
		return "", ErrInvalidAuthorityMode
	}
}

func (k *Kernel) Process(ctx context.Context, req domain.SyscallRequest) (domain.SyscallResult, error) {
	if k == nil || k.port == nil {
		return rejectedResult(req, domain.ErrPersistenceUnavailable, "kernel", ErrMissingCommitAdapter.Error()), ErrMissingCommitAdapter
	}
	if k.authorization == nil {
		return rejectedResult(req, domain.ErrPersistenceUnavailable, "kernel.authorization", ErrMissingAuthorization.Error()), ErrMissingAuthorization
	}
	originalMetadata := req.Metadata
	req.Metadata = cloneMetadata(req.Metadata)
	req.Metadata["forgeKIngressAuthority"] = true
	req.Metadata["kernelAuthorityOwner"] = AuthorityOwnerForgeK
	req.Metadata["durableCommitAdapter"] = DurableCommitAdapter
	if field, ok := forbiddenAuthorityClaim(originalMetadata); ok {
		result := rejectedResult(req, domain.ErrUnauthorized, "metadata."+field, "external workers cannot claim FORGE authority")
		result = annotateAuthority(result)
		result = k.port.RecordResult(ctx, req, result)
		k.port.ObserveResult(ctx, req, result)
		return result, nil
	}
	authorizationProof, authErr := k.authorization.ResolveAuthorization(ctx, req)
	if authErr == nil {
		authErr = authproof.VerifyProof(req, authorizationProof)
	}
	if authErr != nil {
		return k.rejectAuthorizationProof(ctx, req, authErr)
	}
	req.Metadata["forgeKAuthorizationProof"] = authorizationProof.AuthorizationFingerprint
	prepared, err := k.port.Prepare(withVerifiedAuthorizationContext(ctx), req)
	if err != nil || prepared.Disposition == DispositionComplete {
		result := annotateAuthority(prepared.Result)
		result = k.port.RecordResult(ctx, prepared.Request, result)
		k.port.ObserveResult(ctx, prepared.Request, result)
		return result, err
	}
	if prepared.Disposition != DispositionCommit && prepared.Disposition != DispositionReplay {
		result := rejectedResult(prepared.Request, domain.ErrInternal, "kernel.disposition", ErrInvalidDisposition.Error())
		result = annotateAuthority(result)
		result = k.port.RecordResult(ctx, prepared.Request, result)
		k.port.ObserveResult(ctx, prepared.Request, result)
		return result, ErrInvalidDisposition
	}
	if authErr = authproof.VerifyProof(prepared.Request, authorizationProof); authErr != nil {
		return k.rejectAuthorizationProof(ctx, prepared.Request, authErr)
	}
	if !preparedAuthorityMetadataValid(prepared.Request, authorizationProof) {
		return k.rejectAuthorizationProof(ctx, prepared.Request, authproof.ErrProofMismatch)
	}
	if supplied := prepared.AuthorizationProof; !reflect.DeepEqual(supplied, authproof.Proof{}) {
		if authErr = authproof.VerifyProof(prepared.Request, supplied); authErr == nil {
			authErr = authproof.SameAuthorization(supplied, authorizationProof)
		}
		if authErr != nil || supplied.AuthorizationFingerprint != authorizationProof.AuthorizationFingerprint {
			if authErr == nil {
				authErr = authproof.ErrProofMismatch
			}
			return k.rejectAuthorizationProof(ctx, prepared.Request, authErr)
		}
	}
	// The Kernel owns this field. The durable adapter persists it atomically
	// with idempotency/outbox evidence so replay never has to trust metadata as
	// a substitute for the typed proof.
	prepared.AuthorizationProof = authorizationProof
	if prepared.Disposition == DispositionReplay {
		return k.replay(ctx, prepared, authorizationProof)
	}
	if proofErr := authproof.VerifyPlanBinding(authorizationProof, authproof.PlanBinding{
		Action: prepared.Plan.Action, Capability: prepared.Plan.Capability,
		TargetObjectType: prepared.Plan.TargetObjectType, Mutating: prepared.Plan.Mutating,
		JournalEventType: prepared.Plan.JournalEventType,
	}); proofErr != nil {
		return k.rejectAuthorizationProof(ctx, prepared.Request, proofErr)
	}
	if court.IsAction(prepared.Request.Action) {
		decision, issues := court.Decide(prepared.Request)
		if len(issues) > 0 {
			result := rejectedResult(prepared.Request, issues[0].Code, issues[0].Field, issues[0].Message)
			result.ValidationDetails = append(result.ValidationDetails, domain.ValidationDetail{Layer: "forge_k_courthouse", Passed: false, Issues: issues})
			result = annotateAuthority(result)
			result = k.port.RecordResult(ctx, prepared.Request, result)
			k.port.ObserveResult(ctx, prepared.Request, result)
			return result, nil
		}
		prepared.Request.Metadata = cloneMetadata(prepared.Request.Metadata)
		prepared.Request.Metadata[court.MetadataDecisionKey] = decision
		prepared.Result.ValidationDetails = append(prepared.Result.ValidationDetails, domain.ValidationDetail{Layer: "forge_k_courthouse", Passed: true, Issues: []domain.SyscallError{}})
	}
	boundPlan, proofErr := commitproof.BindPreparedPlan(prepared.Request, prepared.Plan)
	prepared.Plan = boundPlan
	var seal commitproof.PreparedPlanSeal
	if proofErr == nil {
		seal, proofErr = commitproof.SealPreparedPlan(prepared.Request, prepared.Plan)
	}
	if proofErr == nil {
		proofErr = commitproof.VerifyPreparedPlan(prepared.Request, prepared.Plan, seal)
	}
	if proofErr != nil {
		return k.rejectCommitProof(ctx, prepared.Request, domain.ErrInternal, "kernel.commitProof.preparedPlan", ErrInvalidPreparedProof, proofErr)
	}
	outcome, err := k.port.Commit(ctx, prepared, seal)
	result := outcome.Result
	if err == nil && result.Success {
		proofErr = commitproof.ValidateCommitReceipt(prepared.Request, prepared.Plan, seal, outcome.Receipt, result)
		if proofErr != nil {
			return k.rejectCommitProof(ctx, prepared.Request, domain.ErrPersistenceUnavailable, "kernel.commitProof.receipt", ErrInvalidCommitReceipt, proofErr)
		}
		result = annotateCommitProof(result, seal, outcome.Receipt)
		result = annotateAuthorizationProof(result, authorizationProof)
	} else if result.Success {
		return k.rejectCommitProof(ctx, prepared.Request, domain.ErrPersistenceUnavailable, "kernel.commitProof.commit", ErrInvalidCommitReceipt, err)
	}
	result = annotateAuthority(result)
	result = k.port.RecordResult(ctx, prepared.Request, result)
	k.port.ObserveResult(ctx, prepared.Request, result)
	return result, err
}

func preparedAuthorityMetadataValid(req domain.SyscallRequest, proof authproof.Proof) bool {
	fingerprint, _ := req.Metadata["forgeKAuthorizationProof"].(string)
	owner, _ := req.Metadata["kernelAuthorityOwner"].(string)
	adapter, _ := req.Metadata["durableCommitAdapter"].(string)
	ingress, _ := req.Metadata["forgeKIngressAuthority"].(bool)
	return fingerprint == proof.AuthorizationFingerprint && owner == AuthorityOwnerForgeK && adapter == DurableCommitAdapter && ingress
}

func (k *Kernel) replay(ctx context.Context, prepared PreparedSyscall, currentAuthorization authproof.Proof) (domain.SyscallResult, error) {
	currentIdempotency, proofErr := commitproof.IdempotencyFingerprint(prepared.Request)
	if proofErr != nil {
		return k.rejectCommitProof(ctx, prepared.Request, domain.ErrInternal, "kernel.commitProof.replayRequest", ErrInvalidPreparedProof, proofErr)
	}

	proofRequest := prepared.ReplayRequest
	if proofErr = authproof.VerifyProof(proofRequest, prepared.ReplayAuthorizationProof); proofErr != nil {
		return k.rejectAuthorizationProof(ctx, prepared.Request, proofErr)
	}
	if proofErr = authproof.SameAuthorization(prepared.ReplayAuthorizationProof, currentAuthorization); proofErr != nil {
		return k.rejectAuthorizationProof(ctx, prepared.Request, proofErr)
	}
	if fingerprint, _ := proofRequest.Metadata["forgeKAuthorizationProof"].(string); fingerprint != prepared.ReplayAuthorizationProof.AuthorizationFingerprint {
		return k.rejectAuthorizationProof(ctx, prepared.Request, authproof.ErrProofMismatch)
	}
	if proofErr = authproof.VerifyPlanBinding(prepared.ReplayAuthorizationProof, authproof.PlanBinding{
		Action: prepared.ReplayPlan.Action, Capability: prepared.ReplayPlan.Capability,
		TargetObjectType: prepared.ReplayPlan.TargetObjectType, Mutating: prepared.ReplayPlan.Mutating,
		JournalEventType: prepared.ReplayPlan.JournalEventType,
	}); proofErr != nil {
		return k.rejectAuthorizationProof(ctx, prepared.Request, proofErr)
	}
	if court.IsAction(proofRequest.Action) {
		proofRequest.Metadata = cloneMetadata(proofRequest.Metadata)
		delete(proofRequest.Metadata, court.MetadataDecisionKey)
		decision, issues := court.Decide(proofRequest)
		if len(issues) > 0 {
			return k.rejectCommitProof(ctx, prepared.Request, domain.ErrPersistenceUnavailable, "kernel.commitProof.replayCourt", ErrInvalidPreparedProof, errors.New(issues[0].Message))
		}
		proofRequest.Metadata[court.MetadataDecisionKey] = decision
	}
	prepared.ReplayPlan, proofErr = commitproof.BindPreparedPlan(proofRequest, prepared.ReplayPlan)
	var computedSeal commitproof.PreparedPlanSeal
	if proofErr == nil {
		computedSeal, proofErr = commitproof.SealPreparedPlan(proofRequest, prepared.ReplayPlan)
	}
	if proofErr == nil {
		proofErr = commitproof.VerifyPreparedPlan(proofRequest, prepared.ReplayPlan, prepared.ReplaySeal)
	}
	if proofErr == nil && computedSeal != prepared.ReplaySeal {
		proofErr = commitproof.ErrSealMismatch
	}
	if proofErr != nil {
		return k.rejectCommitProof(ctx, prepared.Request, domain.ErrPersistenceUnavailable, "kernel.commitProof.replaySeal", ErrInvalidPreparedProof, proofErr)
	}
	if !prepared.Result.Success || prepared.Result.Action != proofRequest.Action {
		return k.rejectCommitProof(ctx, prepared.Request, domain.ErrPersistenceUnavailable, "kernel.commitProof.replayResult", ErrInvalidCommitReceipt, commitproof.ErrReceiptMismatch)
	}
	proofErr = commitproof.ValidateCommitReceipt(proofRequest, prepared.ReplayPlan, prepared.ReplaySeal, prepared.ReplayReceipt, prepared.Result)
	if proofErr != nil {
		return k.rejectCommitProof(ctx, prepared.Request, domain.ErrPersistenceUnavailable, "kernel.commitProof.replayReceipt", ErrInvalidCommitReceipt, proofErr)
	}
	if currentIdempotency != prepared.ReplayReceipt.IdempotencyFingerprint {
		return k.rejectCommitProof(ctx, prepared.Request, domain.ErrDuplicate, "idempotencyKey", ErrInvalidCommitReceipt, commitproof.ErrReceiptMismatch)
	}

	result := prepared.Result
	result.Action = prepared.Request.Action
	result.RequestID = prepared.Request.ID
	result.CorrelationID = prepared.Request.CorrelationID
	result.TraceID = prepared.Request.TraceID
	result.IdempotencyKey = prepared.Request.IdempotencyKey
	if !containsString(result.Warnings, "idempotent replay") {
		result.Warnings = append(result.Warnings, "idempotent replay")
	}
	result = annotateCommitProof(result, prepared.ReplaySeal, prepared.ReplayReceipt)
	result = annotateAuthorizationProof(result, prepared.ReplayAuthorizationProof)
	result = annotateAuthority(result)
	result = k.port.RecordResult(ctx, prepared.Request, result)
	k.port.ObserveResult(ctx, prepared.Request, result)
	return result, nil
}

func (k *Kernel) rejectAuthorizationProof(ctx context.Context, req domain.SyscallRequest, cause error) (domain.SyscallResult, error) {
	message := ErrInvalidAuthorization.Error()
	if cause != nil {
		message += ": " + cause.Error()
	}
	result := rejectedResult(req, domain.ErrUnauthorized, "kernel.authorizationProof", message)
	result.ValidationDetails = append(result.ValidationDetails, domain.ValidationDetail{
		Layer: "forge_k_authorization_proof", Passed: false,
		Issues: append([]domain.SyscallError(nil), result.RejectedReasons...),
	})
	result = annotateAuthority(result)
	result.StateSummary["authorizationProofVerified"] = false
	result = k.port.RecordResult(ctx, req, result)
	k.port.ObserveResult(ctx, req, result)
	if cause == nil {
		return result, ErrInvalidAuthorization
	}
	return result, errors.Join(ErrInvalidAuthorization, cause)
}

func (k *Kernel) rejectCommitProof(ctx context.Context, req domain.SyscallRequest, code domain.SyscallErrorCode, field string, sentinel, cause error) (domain.SyscallResult, error) {
	message := sentinel.Error()
	if cause != nil {
		message += ": " + cause.Error()
	}
	result := rejectedResult(req, code, field, message)
	result.ValidationDetails = append(result.ValidationDetails, domain.ValidationDetail{
		Layer:  "forge_k_commit_integrity",
		Passed: false,
		Issues: append([]domain.SyscallError(nil), result.RejectedReasons...),
	})
	result = annotateAuthority(result)
	result.StateSummary["commitProofVerified"] = false
	result = k.port.RecordResult(ctx, req, result)
	k.port.ObserveResult(ctx, req, result)
	if cause == nil {
		return result, sentinel
	}
	return result, errors.Join(sentinel, cause)
}

func annotateCommitProof(result domain.SyscallResult, seal commitproof.PreparedPlanSeal, receipt commitproof.CommitReceipt) domain.SyscallResult {
	if result.StateSummary == nil {
		result.StateSummary = map[string]any{}
	}
	result.StateSummary["commitProofVerified"] = true
	result.StateSummary["requestFingerprint"] = seal.RequestFingerprint
	result.StateSummary["preparedPlanSeal"] = seal.SealDigest
	result.StateSummary["transactionId"] = receipt.TransactionID
	result.StateSummary["journalEventId"] = receipt.JournalEventID
	result.StateSummary["journalEventHash"] = receipt.JournalEventHash
	result.StateSummary["auditOutboxId"] = receipt.AuditOutboxID
	result.StateSummary["idempotencyFingerprint"] = receipt.IdempotencyFingerprint
	result.ValidationDetails = append(result.ValidationDetails, domain.ValidationDetail{
		Layer:  "forge_k_commit_integrity",
		Passed: true,
		Issues: []domain.SyscallError{},
	})
	return result
}

func annotateAuthorizationProof(result domain.SyscallResult, proof authproof.Proof) domain.SyscallResult {
	if result.StateSummary == nil {
		result.StateSummary = map[string]any{}
	}
	result.StateSummary["authorizationProofVerified"] = true
	result.StateSummary["authorizationFingerprint"] = proof.AuthorizationFingerprint
	result.StateSummary["authorizationEvidenceSnapshotId"] = proof.EvidenceSnapshotID
	result.ValidationDetails = append(result.ValidationDetails, domain.ValidationDetail{
		Layer: "forge_k_authorization_proof", Passed: true, Issues: []domain.SyscallError{},
	})
	return result
}

func annotateAuthority(result domain.SyscallResult) domain.SyscallResult {
	if result.StateSummary == nil {
		result.StateSummary = map[string]any{}
	}
	result.StateSummary["kernelAuthorityOwner"] = AuthorityOwnerForgeK
	result.StateSummary["durableCommitAdapter"] = DurableCommitAdapter
	result.StateSummary["singleCommitAuthority"] = true
	result.StateSummary["modelToolSelectionAuthority"] = false
	result.StateSummary["modelExecutionAuthority"] = false
	result.StateSummary["modelCanonicalMutationAuthority"] = false
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func cloneMetadata(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+3)
	for key, value := range in {
		out[key] = value
	}
	return out
}

func forbiddenAuthorityClaim(metadata map[string]any) (string, bool) {
	for _, key := range []string{
		"modelToolSelectionAuthority",
		"modelExecutionAuthority",
		"modelCanonicalMutationAuthority",
		"directModelMutation",
	} {
		if value, ok := metadata[key]; ok && authorityClaimIsTruthy(value) {
			return key, true
		}
	}
	return "", false
}

func authorityClaimIsTruthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "", "0", "false", "no", "off":
			return false
		default:
			return true
		}
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case float64:
		return typed != 0
	case nil:
		return false
	default:
		return true
	}
}

func rejectedResult(req domain.SyscallRequest, code domain.SyscallErrorCode, field, message string) domain.SyscallResult {
	return domain.SyscallResult{
		Success:            false,
		Action:             req.Action,
		RequestID:          req.ID,
		CorrelationID:      req.CorrelationID,
		TraceID:            req.TraceID,
		IdempotencyKey:     req.IdempotencyKey,
		DryRun:             req.DryRun,
		ApprovalStatus:     domain.ApprovalDenied,
		CommittedObjectIDs: []string{},
		RejectedReasons: []domain.SyscallError{{
			Code: code, Field: field, Message: message,
		}},
		Warnings: []string{},
		ValidationDetails: []domain.ValidationDetail{{
			Layer: "forge_k_authority", Passed: false,
			Issues: []domain.SyscallError{{Code: code, Field: field, Message: message}},
		}},
		StateSummary: map[string]any{
			"kernelAuthorityOwner":            AuthorityOwnerForgeK,
			"durableCommitAdapter":            DurableCommitAdapter,
			"singleCommitAuthority":           true,
			"modelToolSelectionAuthority":     false,
			"modelExecutionAuthority":         false,
			"modelCanonicalMutationAuthority": false,
		},
		DeterministicErrCode: code,
	}
}
