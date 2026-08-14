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
	ErrInvalidAuthorityMode  = errors.New("invalid kernel authority mode")
	ErrMissingCommitAdapter  = errors.New("missing durable kernel commit adapter")
	ErrMissingRejectionAudit = errors.New("missing kernel rejection audit adapter")
)

type Processor interface {
	Process(ctx context.Context, req domain.SyscallRequest) (domain.SyscallResult, error)
}

type RejectionAuditor interface {
	RecordKernelRejection(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) domain.SyscallResult
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
	commit  Processor
	rejects RejectionAuditor
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
	rejects, ok := commit.(RejectionAuditor)
	if !ok {
		return Selection{}, ErrMissingRejectionAudit
	}
	return Selection{
		Processor:       &Kernel{commit: commit, rejects: rejects},
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
	if k == nil || k.commit == nil {
		return rejectedResult(req, domain.ErrPersistenceUnavailable, "kernel", ErrMissingCommitAdapter.Error()), ErrMissingCommitAdapter
	}
	originalMetadata := req.Metadata
	req.Metadata = cloneMetadata(req.Metadata)
	req.Metadata["forgeKIngressAuthority"] = true
	req.Metadata["kernelAuthorityOwner"] = AuthorityOwnerForgeK
	req.Metadata["durableCommitAdapter"] = DurableCommitAdapter
	if field, ok := forbiddenAuthorityClaim(originalMetadata); ok {
		result := rejectedResult(req, domain.ErrUnauthorized, "metadata."+field, "external workers cannot claim FORGE authority")
		return k.rejects.RecordKernelRejection(ctx, req, result), nil
	}
	result, err := k.commit.Process(ctx, req)
	if result.StateSummary == nil {
		result.StateSummary = map[string]any{}
	}
	result.StateSummary["kernelAuthorityOwner"] = AuthorityOwnerForgeK
	result.StateSummary["durableCommitAdapter"] = DurableCommitAdapter
	result.StateSummary["singleCommitAuthority"] = true
	result.StateSummary["modelToolSelectionAuthority"] = false
	result.StateSummary["modelExecutionAuthority"] = false
	result.StateSummary["modelCanonicalMutationAuthority"] = false
	return result, err
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
