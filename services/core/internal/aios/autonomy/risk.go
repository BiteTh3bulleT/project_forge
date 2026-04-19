package autonomy

import (
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

type RiskClassification struct {
	Action           domain.SyscallRequest `json:"action"`
	Risk             domain.AutonomyRisk   `json:"risk"`
	AutonomyLevel    domain.AutonomyLevel  `json:"autonomyLevel"`
	RequiresApproval bool                  `json:"requiresApproval"`
	Reasons          []string              `json:"reasons"`
}

type RiskSummary struct {
	MaxRisk             domain.AutonomyRisk  `json:"maxRisk"`
	Classifications     []RiskClassification `json:"classifications"`
	ApprovalRequired    bool                 `json:"approvalRequired"`
	ContainsGuardrail   bool                 `json:"containsGuardrailBlock"`
	GuardrailViolations []string             `json:"guardrailViolations,omitempty"`
}

type RiskClassifier interface {
	ClassifyIntent(intent domain.AutonomyIntent, actions []domain.SyscallRequest) RiskSummary
	ClassifyAction(intent domain.AutonomyIntent, action domain.SyscallRequest) RiskClassification
}

type DeterministicRiskClassifier struct{}

func NewDeterministicRiskClassifier() DeterministicRiskClassifier {
	return DeterministicRiskClassifier{}
}

func (DeterministicRiskClassifier) ClassifyIntent(intent domain.AutonomyIntent, actions []domain.SyscallRequest) RiskSummary {
	if len(actions) == 0 {
		return RiskSummary{MaxRisk: domain.AutonomyRiskNone, Classifications: nil, ApprovalRequired: false}
	}
	out := RiskSummary{MaxRisk: domain.AutonomyRiskNone, Classifications: make([]RiskClassification, 0, len(actions))}
	for _, action := range actions {
		classification := classifyAutonomyAction(intent, action)
		out.Classifications = append(out.Classifications, classification)
		if classification.Risk.Rank() > out.MaxRisk.Rank() {
			out.MaxRisk = classification.Risk
		}
		if classification.RequiresApproval {
			out.ApprovalRequired = true
		}
		for _, reason := range classification.Reasons {
			if strings.HasPrefix(reason, "guardrail:") {
				out.ContainsGuardrail = true
				out.GuardrailViolations = append(out.GuardrailViolations, reason)
			}
		}
	}
	return out
}

func (DeterministicRiskClassifier) ClassifyAction(intent domain.AutonomyIntent, action domain.SyscallRequest) RiskClassification {
	return classifyAutonomyAction(intent, action)
}

func classifyAutonomyAction(intent domain.AutonomyIntent, action domain.SyscallRequest) RiskClassification {
	risk := domain.AutonomyRiskLow
	reasons := []string{}
	actionName := strings.TrimSpace(string(action.Action))

	switch action.Action {
	case domain.ActionCreateLink:
		risk = domain.AutonomyRiskLow
		reasons = append(reasons, "semantic link creation within same workspace")
	case domain.ActionCreateNote:
		risk = domain.AutonomyRiskLow
		noteType := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", action.Payload["type"])))
		if noteType != "" && noteType != "system" && noteType != "episode" && noteType != "policy" {
			risk = domain.AutonomyRiskMedium
			reasons = append(reasons, "non-diagnostic note write")
		} else {
			reasons = append(reasons, "diagnostic/maintenance note")
		}
	case domain.ActionCompileContext:
		risk = domain.AutonomyRiskLow
		reasons = append(reasons, "context compilation snapshot")
	case domain.ActionRegisterContradict:
		risk = domain.AutonomyRiskLow
		reasons = append(reasons, "contradiction preserves both sides")
	case domain.ActionArchiveNote:
		risk = domain.AutonomyRiskMedium
		reasons = append(reasons, "archival mutates lifecycle status")
		if affectsActiveStateEvidence(action) {
			risk = domain.AutonomyRiskHigh
			reasons = append(reasons, "guardrail: archive appears to affect active-state evidence")
		}
	case domain.ActionMarkSuperseded:
		risk = domain.AutonomyRiskMedium
		reasons = append(reasons, "supersession changes current retrieval status")
	case domain.ActionDeriveModel:
		risk = domain.AutonomyRiskMedium
		reasons = append(reasons, "model derivation is inferential")
		if supportCount(action) < 2 {
			risk = domain.AutonomyRiskHigh
			reasons = append(reasons, "guardrail: model promotion/derivation with weak evidence")
		}
	case domain.ActionUpdateState:
		risk = domain.AutonomyRiskMedium
		key := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", action.Payload["key"])))
		if strings.Contains(key, "architecture") || strings.Contains(key, "policy") || strings.Contains(key, "permission") || strings.Contains(key, "system") {
			risk = domain.AutonomyRiskHigh
			reasons = append(reasons, "state key is architecture/policy/system-impact")
		} else {
			reasons = append(reasons, "low-impact internal state update")
		}
	case domain.ActionCloseLoop:
		risk = domain.AutonomyRiskHigh
		reasons = append(reasons, "loop closure can hide unresolved work")
		if isHighPriorityLoop(action) {
			reasons = append(reasons, "guardrail: high-priority loop close")
		}
	case domain.ActionOpenLoop:
		risk = domain.AutonomyRiskLow
		reasons = append(reasons, "opening loop preserves unresolved state")
	default:
		if isCriticalExternalAction(actionName) {
			risk = domain.AutonomyRiskCritical
			reasons = append(reasons, "guardrail: destructive/external action category")
		} else {
			risk = domain.AutonomyRiskHigh
			reasons = append(reasons, "unknown action defaults high risk")
		}
	}

	if !scopeMatch(intent.Scope, action.Scope) {
		risk = raiseRisk(risk)
		reasons = append(reasons, "scope mismatch between intent and action")
		if strings.TrimSpace(intent.Scope.WorkspaceID) != strings.TrimSpace(action.Scope.WorkspaceID) {
			risk = domain.AutonomyRiskCritical
			reasons = append(reasons, "guardrail: cross-workspace mutation")
		}
	}
	if weakProvenance(action.Provenance) {
		risk = raiseRisk(risk)
		reasons = append(reasons, "weak provenance")
	}

	requiresApproval := risk.Rank() >= domain.AutonomyRiskHigh.Rank()
	for _, reason := range reasons {
		if strings.HasPrefix(reason, "guardrail:") {
			requiresApproval = true
		}
	}

	return RiskClassification{
		Action:           action,
		Risk:             risk,
		AutonomyLevel:    recommendedLevelForRisk(risk),
		RequiresApproval: requiresApproval,
		Reasons:          reasons,
	}
}

func recommendedLevelForRisk(risk domain.AutonomyRisk) domain.AutonomyLevel {
	switch risk {
	case domain.AutonomyRiskNone:
		return domain.AutonomyLevelObserveOnly
	case domain.AutonomyRiskLow:
		return domain.AutonomyLevelAutoCommitSafe
	case domain.AutonomyRiskMedium:
		return domain.AutonomyLevelProposeOnly
	case domain.AutonomyRiskHigh, domain.AutonomyRiskCritical:
		return domain.AutonomyLevelApprovalRequired
	default:
		return domain.AutonomyLevelApprovalRequired
	}
}

func raiseRisk(risk domain.AutonomyRisk) domain.AutonomyRisk {
	switch risk {
	case domain.AutonomyRiskNone:
		return domain.AutonomyRiskLow
	case domain.AutonomyRiskLow:
		return domain.AutonomyRiskMedium
	case domain.AutonomyRiskMedium:
		return domain.AutonomyRiskHigh
	case domain.AutonomyRiskHigh:
		return domain.AutonomyRiskCritical
	default:
		return domain.AutonomyRiskCritical
	}
}

func weakProvenance(prov domain.Provenance) bool {
	return strings.TrimSpace(prov.Actor) == "" || strings.TrimSpace(prov.ActorType) == "" || strings.TrimSpace(prov.Source) == ""
}

func isHighPriorityLoop(action domain.SyscallRequest) bool {
	priority := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", action.Payload["priority"])))
	if priority == "high" || priority == "critical" {
		return true
	}
	if loopPriority, ok := action.Metadata["loopPriority"]; ok {
		v := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", loopPriority)))
		return v == "high" || v == "critical"
	}
	return false
}

func affectsActiveStateEvidence(action domain.SyscallRequest) bool {
	for _, key := range []string{"active_state_dependency", "activeStateDependency", "protectFromArchive"} {
		if v, ok := action.Payload[key]; ok {
			text := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", v)))
			if text == "true" || text == "1" || text == "yes" {
				return true
			}
		}
	}
	return false
}

func supportCount(action domain.SyscallRequest) int {
	if v, ok := action.Payload["supportCount"]; ok {
		switch x := v.(type) {
		case int:
			return x
		case int32:
			return int(x)
		case int64:
			return int(x)
		case float64:
			return int(x)
		}
	}
	if v, ok := action.Payload["derivedFrom"]; ok {
		switch x := v.(type) {
		case []string:
			return len(x)
		case []any:
			return len(x)
		}
	}
	return 0
}

func isCriticalExternalAction(actionName string) bool {
	upper := strings.ToUpper(strings.TrimSpace(actionName))
	if upper == "" {
		return false
	}
	critical := []string{
		"DELETE_ARTIFACT", "DELETE_FILE", "DELETE_WORKSPACE", "EXTERNAL_SEND", "PUBLISH_RELEASE",
		"CHANGE_PERMISSION", "CHANGE_CREDENTIAL", "CROSS_WORKSPACE_MUTATION", "EXPORT_DATA",
	}
	for _, item := range critical {
		if upper == item {
			return true
		}
	}
	if strings.Contains(upper, "DELETE") || strings.Contains(upper, "EXTERNAL") || strings.Contains(upper, "CREDENTIAL") {
		return true
	}
	return false
}
