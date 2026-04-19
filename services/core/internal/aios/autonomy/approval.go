package autonomy

import (
	"context"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

type ApprovalGate interface {
	RequestApproval(ctx context.Context, intent domain.AutonomyIntent, decision domain.AutonomyDecision) (domain.ApprovalEscalationResult, error)
}

type StaticApprovalEscalator struct {
	DefaultStatus domain.ApprovalStatus
}

func NewStaticApprovalEscalator() StaticApprovalEscalator {
	return StaticApprovalEscalator{DefaultStatus: domain.ApprovalRequired}
}

func (s StaticApprovalEscalator) RequestApproval(_ context.Context, intent domain.AutonomyIntent, decision domain.AutonomyDecision) (domain.ApprovalEscalationResult, error) {
	status := s.DefaultStatus
	if status == "" {
		status = domain.ApprovalRequired
	}
	approvalID := "approval-" + shortHash(intent.ID, decision.ID, string(status))
	out := domain.ApprovalEscalationResult{
		Status:            status,
		ApprovalID:        approvalID,
		Reason:            strings.TrimSpace(decision.RequiredApprovalReason),
		OperatorMessage:   fmt.Sprintf("autonomy intent %s requires approval for %d action(s)", intent.ID, len(decision.AllowedActions)),
		RecommendedAction: "review and approve or deny the intent",
	}
	if status == domain.ApprovalAllowed {
		out.OperatorMessage = "autonomy approval gate auto-allowed in this environment"
	}
	if status == domain.ApprovalDenied {
		out.OperatorMessage = "autonomy approval gate denied"
	}
	return out, nil
}
