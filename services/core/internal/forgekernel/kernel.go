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
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
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
	ErrInvalidDisposition   = errors.New("invalid durable port disposition")
)

type Processor interface {
	Process(ctx context.Context, req domain.SyscallRequest) (domain.SyscallResult, error)
}

type Disposition string

const (
	DispositionComplete Disposition = "complete"
	DispositionCommit   Disposition = "commit"
)

type PreparedSyscall struct {
	Request     domain.SyscallRequest
	Result      domain.SyscallResult
	Disposition Disposition
}

// DurablePort is the temporary compatibility boundary implemented by the
// Control Lane adapter. FORGE-K owns the stage order; the port supplies
// non-mutating preflight work, one atomic commit/journal operation, audit
// persistence, and best-effort observation.
type DurablePort interface {
	Prepare(ctx context.Context, req domain.SyscallRequest) (PreparedSyscall, error)
	Commit(ctx context.Context, prepared PreparedSyscall) (domain.SyscallResult, error)
	RecordResult(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) domain.SyscallResult
	ObserveResult(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult)
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
	port DurablePort
}

func SelectAuthority(rawMode string, commit Processor) (Selection, error) {
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
	return Selection{
		Processor:       &Kernel{port: port},
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
	prepared, err := k.port.Prepare(ctx, req)
	if err != nil || prepared.Disposition == DispositionComplete {
		result := annotateAuthority(prepared.Result)
		result = k.port.RecordResult(ctx, prepared.Request, result)
		k.port.ObserveResult(ctx, prepared.Request, result)
		return result, err
	}
	if prepared.Disposition != DispositionCommit {
		result := rejectedResult(prepared.Request, domain.ErrInternal, "kernel.disposition", ErrInvalidDisposition.Error())
		result = annotateAuthority(result)
		result = k.port.RecordResult(ctx, prepared.Request, result)
		k.port.ObserveResult(ctx, prepared.Request, result)
		return result, ErrInvalidDisposition
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
	result, err := k.port.Commit(ctx, prepared)
	result = annotateAuthority(result)
	result = k.port.RecordResult(ctx, prepared.Request, result)
	k.port.ObserveResult(ctx, prepared.Request, result)
	return result, err
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
