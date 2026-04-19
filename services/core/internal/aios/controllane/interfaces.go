package controllane

import (
	"context"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/permissions"
)

// ForgeKernel is the deterministic commit boundary for semantic actions.
// It validates candidate actions and applies only accepted state mutations.
type ForgeKernel interface {
	Process(ctx context.Context, req domain.SyscallRequest) (domain.SyscallResult, error)
}

// PermissionService gates scope/tool/risk dimensions for proposed actions.
type PermissionService interface {
	Check(ctx context.Context, req permissions.CheckRequest) (*permissions.Decision, *permissions.Profile, error)
}

// StateTransitionService enforces deterministic state transition rules.
type StateTransitionService interface {
	ValidateTransition(ctx context.Context, stateKey, fromStatus, toStatus string) (bool, string, error)
}

// AuditService records kernel decisions as immutable audit evidence.
type AuditService interface {
	Record(ctx context.Context, req audit.CreateRequest) (*audit.Record, error)
}
