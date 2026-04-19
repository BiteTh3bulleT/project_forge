package autonomy

import (
	"fmt"
	"strconv"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

type CharterActionDecision struct {
	Allowed          bool
	ApprovalRequired bool
	Reasons          []string
	Warnings         []string
}

type CharterConditionContext struct {
	Action                   domain.SyscallRequest
	Intent                   domain.AutonomyIntent
	NowMillis                int64
	HasActiveStateDependency func(objectID string) bool
}

type CharterService struct{}

func NewCharterService() CharterService {
	return CharterService{}
}

func (CharterService) EvaluateAction(charter domain.AutonomyCharter, risk domain.AutonomyRisk, ctx CharterConditionContext) CharterActionDecision {
	decision := CharterActionDecision{Allowed: false, ApprovalRequired: false, Reasons: []string{}, Warnings: []string{}}
	if !charter.IsActive(ctx.NowMillis) {
		decision.Reasons = append(decision.Reasons, "charter is not active")
		return decision
	}
	if !scopeMatch(charter.Scope, ctx.Intent.Scope) || !scopeMatch(charter.Scope, ctx.Action.Scope) {
		decision.Reasons = append(decision.Reasons, "charter scope mismatch")
		return decision
	}
	if len(charter.AllowedSources) > 0 && !containsSource(charter.AllowedSources, ctx.Intent.Source) {
		decision.Reasons = append(decision.Reasons, "intent source is not allowed by charter")
		return decision
	}
	if charter.DeniesAction(ctx.Action.Action) {
		decision.Reasons = append(decision.Reasons, "action denied by charter")
		return decision
	}
	if charter.RequiresApproval(ctx.Action.Action) {
		decision.Allowed = true
		decision.ApprovalRequired = true
		decision.Reasons = append(decision.Reasons, "action requires approval by charter")
		return decision
	}
	if !charter.AllowsAction(ctx.Action.Action) {
		decision.Reasons = append(decision.Reasons, "action not listed in charter allowed actions")
		return decision
	}

	rule, hasRule := conditionalRuleForAction(charter, ctx.Action.Action)
	if hasRule {
		if rule.MaxRisk != "" && risk.Rank() > rule.MaxRisk.Rank() {
			decision.Reasons = append(decision.Reasons, fmt.Sprintf("action risk %s exceeds conditional maxRisk %s", risk, rule.MaxRisk))
			return decision
		}
		for _, cond := range rule.Conditions {
			ok, warn := evaluateCondition(cond, ctx)
			if warn != "" {
				decision.Warnings = append(decision.Warnings, warn)
			}
			if !ok {
				decision.Reasons = append(decision.Reasons, "conditional action requirement failed: "+cond)
				return decision
			}
		}
		if len(rule.ApprovalRequiredWhen) > 0 {
			for _, when := range rule.ApprovalRequiredWhen {
				ok, _ := evaluateCondition(when, ctx)
				if ok {
					decision.ApprovalRequired = true
					decision.Reasons = append(decision.Reasons, "conditional approval requirement matched: "+when)
					break
				}
			}
		}
	}
	decision.Allowed = true
	if len(decision.Reasons) == 0 {
		decision.Reasons = append(decision.Reasons, "action allowed by active charter")
	}
	return decision
}

func conditionalRuleForAction(charter domain.AutonomyCharter, action domain.SemanticActionType) (domain.ConditionRule, bool) {
	for _, rule := range charter.ConditionalActions {
		if rule.Action == action {
			return rule, true
		}
	}
	return domain.ConditionRule{}, false
}

func evaluateCondition(cond string, ctx CharterConditionContext) (bool, string) {
	cond = strings.TrimSpace(strings.ToLower(cond))
	if cond == "" {
		return true, ""
	}
	switch {
	case cond == "note.status == superseded":
		status := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", ctx.Action.Payload["noteStatus"])))
		if status == "" {
			status = strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", ctx.Action.Payload["status"])))
		}
		return status == "superseded", ""
	case strings.HasPrefix(cond, "age_days >"):
		thresholdRaw := strings.TrimSpace(strings.TrimPrefix(cond, "age_days >"))
		threshold, err := strconv.Atoi(thresholdRaw)
		if err != nil {
			return false, "unsupported conditional expression: " + cond
		}
		ageDays := parseInt(ctx.Action.Payload["ageDays"])
		if ageDays == 0 {
			createdAt := parseInt64(ctx.Action.Payload["createdAt"])
			if createdAt > 0 {
				ageDays = int((ctx.NowMillis - createdAt) / (24 * 60 * 60 * 1000))
			}
		}
		return ageDays > threshold, ""
	case cond == "no_active_state_depends_on_note":
		noteID := strings.TrimSpace(fmt.Sprintf("%v", ctx.Action.Payload["noteId"]))
		if noteID == "" {
			noteID = strings.TrimSpace(fmt.Sprintf("%v", ctx.Action.Payload["id"]))
		}
		if noteID == "" {
			return false, "note id missing for state dependency check"
		}
		if ctx.HasActiveStateDependency == nil {
			return false, "state dependency evaluator is not configured"
		}
		return !ctx.HasActiveStateDependency(noteID), ""
	case cond == "risk == high":
		return false, ""
	default:
		return false, "unsupported conditional expression: " + cond
	}
}

func parseInt(raw any) int {
	switch x := raw.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float64:
		return int(x)
	case string:
		v, err := strconv.Atoi(strings.TrimSpace(x))
		if err == nil {
			return v
		}
	}
	return 0
}

func parseInt64(raw any) int64 {
	switch x := raw.(type) {
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	case string:
		v, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err == nil {
			return v
		}
	}
	return 0
}
