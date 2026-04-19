package controllane

import (
	"context"

	"forge/projectforge/services/core/internal/aios/domain"
)

type ApprovalDecision struct {
	Status domain.ApprovalStatus `json:"status"`
	Reason string                `json:"reason"`
}

type ApprovalGate interface {
	Evaluate(ctx context.Context, req domain.SyscallRequest, def ActionDefinition) (ApprovalDecision, error)
}

type StaticApprovalGate struct{}

func NewStaticApprovalGate() *StaticApprovalGate {
	return &StaticApprovalGate{}
}

func (g *StaticApprovalGate) Evaluate(_ context.Context, req domain.SyscallRequest, def ActionDefinition) (ApprovalDecision, error) {
	if !def.ApprovalPossible {
		return ApprovalDecision{Status: domain.ApprovalAllowed, Reason: "approval not required for action"}, nil
	}
	if !def.Mutating {
		return ApprovalDecision{Status: domain.ApprovalAllowed, Reason: "read-only action"}, nil
	}
	switch req.Source {
	case domain.SourceFutureIRIS, domain.SourceAdapter:
		return ApprovalDecision{
			Status: domain.ApprovalRequired,
			Reason: "mutating semantic actions from proposer sources require approval",
		}, nil
	default:
		return ApprovalDecision{Status: domain.ApprovalAllowed, Reason: "approval gate passed"}, nil
	}
}
