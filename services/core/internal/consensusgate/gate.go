package consensusgate

import (
	"sort"
	"strings"
)

const (
	SchemaVersion = "consensusgate/v1"

	StatusAcceptedMetadataOnly = "accepted_metadata_only"
	StatusUncertain            = "uncertain"
	StatusWithheld             = "withheld"

	SurfaceChatFinal      = "chat.final_response"
	SurfaceActionProposal = "action.proposal"
)

type Input struct {
	Content               string
	Surface               string
	WorkspaceID           string
	RequestID             string
	CorrelationID         string
	EvidenceRefs          []string
	GatewayExecutionState string
	ModelProposalOnly     bool
	RiskClass             string
}

type Decision struct {
	SchemaVersion          string   `json:"schemaVersion"`
	Status                 string   `json:"status"`
	Surface                string   `json:"surface"`
	WorkspaceID            string   `json:"workspaceId,omitempty"`
	RequestID              string   `json:"requestId,omitempty"`
	CorrelationID          string   `json:"correlationId,omitempty"`
	AcceptedClaimCount     int      `json:"acceptedClaimCount"`
	UncertainClaimCount    int      `json:"uncertainClaimCount"`
	WithheldClaimCount     int      `json:"withheldClaimCount"`
	ConflictDetected       bool     `json:"conflictDetected"`
	HighRiskActionClaim    bool     `json:"highRiskActionClaim"`
	EvidenceRefCount       int      `json:"evidenceRefCount"`
	GatewayEvidence        bool     `json:"gatewayEvidence"`
	ModelProposalOnly      bool     `json:"modelProposalOnly"`
	CanonicalTruth         bool     `json:"canonicalTruth"`
	MemoryMutation         bool     `json:"memoryMutation"`
	EvidenceAdmission      bool     `json:"evidenceAdmission"`
	GatewayExecution       bool     `json:"gatewayExecution"`
	ModelRuntimeCall       bool     `json:"modelRuntimeCall"`
	ContextCompilation     bool     `json:"contextCompilation"`
	LiveAuthorityMigration bool     `json:"liveAuthorityMigration"`
	Content                string   `json:"-"`
	Warnings               []string `json:"warnings,omitempty"`
	RiskFlags              []string `json:"riskFlags,omitempty"`
}

func Gate(input Input) Decision {
	content := strings.TrimSpace(input.Content)
	surface := normalizeSurface(input.Surface)
	refs := normalizeRefs(input.EvidenceRefs)
	gatewayEvidence := isGatewayEvidence(input.GatewayExecutionState)
	highRiskAction := containsHighRiskActionClaim(content)
	conflict := containsConflict(content)

	decision := Decision{
		SchemaVersion:          SchemaVersion,
		Status:                 StatusUncertain,
		Surface:                surface,
		WorkspaceID:            strings.TrimSpace(input.WorkspaceID),
		RequestID:              strings.TrimSpace(input.RequestID),
		CorrelationID:          strings.TrimSpace(input.CorrelationID),
		ConflictDetected:       conflict,
		HighRiskActionClaim:    highRiskAction,
		EvidenceRefCount:       len(refs),
		GatewayEvidence:        gatewayEvidence,
		ModelProposalOnly:      input.ModelProposalOnly,
		CanonicalTruth:         false,
		MemoryMutation:         false,
		EvidenceAdmission:      false,
		GatewayExecution:       false,
		ModelRuntimeCall:       false,
		ContextCompilation:     false,
		LiveAuthorityMigration: false,
		Content:                content,
	}

	if content == "" {
		decision.UncertainClaimCount = 1
		decision.Warnings = []string{"empty_response"}
		return decision
	}
	if highRiskAction && !gatewayEvidence {
		decision.Status = StatusWithheld
		decision.WithheldClaimCount = 1
		decision.Content = "I cannot claim that action completed without gateway, audit, or approval evidence."
		decision.Warnings = []string{"unsupported_high_risk_action_claim_withheld"}
		decision.RiskFlags = []string{"high_risk_action_claim", "insufficient_execution_evidence"}
		return decision
	}
	if gatewayEvidence || len(refs) > 0 {
		decision.Status = StatusAcceptedMetadataOnly
		decision.AcceptedClaimCount = 1
		if conflict {
			decision.Status = StatusUncertain
			decision.AcceptedClaimCount = 0
			decision.UncertainClaimCount = 1
			decision.Warnings = []string{"conflict_detected"}
			decision.RiskFlags = []string{"unresolved_conflict"}
		}
		return decision
	}

	decision.UncertainClaimCount = 1
	if input.ModelProposalOnly {
		decision.Warnings = []string{"model_proposal_without_external_evidence"}
	}
	if conflict {
		decision.Warnings = append(decision.Warnings, "conflict_detected")
		decision.RiskFlags = []string{"unresolved_conflict"}
	}
	return decision
}

func normalizeSurface(value string) string {
	switch strings.TrimSpace(value) {
	case SurfaceActionProposal:
		return SurfaceActionProposal
	default:
		return SurfaceChatFinal
	}
}

func normalizeRefs(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, ref := range in {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	sort.Strings(out)
	return out
}

func isGatewayEvidence(state string) bool {
	switch strings.TrimSpace(strings.ToLower(state)) {
	case "ok", "needs_approval", "denied", "error":
		return true
	default:
		return false
	}
}

func containsHighRiskActionClaim(content string) bool {
	normalized := " " + strings.ToLower(strings.Join(strings.Fields(content), " ")) + " "
	for _, phrase := range []string{
		" i deleted ",
		" i removed ",
		" i wrote ",
		" i created ",
		" i modified ",
		" i changed ",
		" i ran ",
		" i executed ",
		" i installed ",
		" i pushed ",
		" i committed ",
		" i shut down ",
		" i rebooted ",
		" file was deleted ",
		" file was written ",
		" command was executed ",
	} {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func containsConflict(content string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(content), " "))
	if strings.Contains(normalized, "conflict") || strings.Contains(normalized, "contradiction") {
		return true
	}
	for _, pair := range [][2]string{
		{" exists", " does not exist"},
		{" is enabled", " is not enabled"},
		{" is available", " is unavailable"},
		{" completed", " did not complete"},
	} {
		if strings.Contains(normalized, pair[0]) && strings.Contains(normalized, pair[1]) {
			return true
		}
	}
	return false
}
