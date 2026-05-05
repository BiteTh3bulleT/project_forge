package consensus

import "time"

type ClaimInput struct {
	ClaimID      string         `json:"claim_id"`
	RequestID    string         `json:"request_id"`
	ClaimType    ClaimType      `json:"claim_type"`
	Subject      string         `json:"subject"`
	Predicate    string         `json:"predicate"`
	ValueJSON    any            `json:"value_json"`
	Scope        string         `json:"scope,omitempty"`
	Temporal     string         `json:"temporal,omitempty"`
	EvidenceRefs []string       `json:"evidence_refs,omitempty"`
	Confidence   float64        `json:"confidence"`
	AgentID      string         `json:"agent_id"`
	AgentRunID   string         `json:"agent_run_id,omitempty"`
	RiskFlags    []string       `json:"risk_flags,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type Claim struct {
	ClaimID      string         `json:"claim_id"`
	RequestID    string         `json:"request_id"`
	ClaimKey     string         `json:"claim_key"`
	ClaimType    ClaimType      `json:"claim_type"`
	Subject      string         `json:"subject"`
	Predicate    string         `json:"predicate"`
	ValueJSON    any            `json:"value_json"`
	Scope        string         `json:"scope,omitempty"`
	Temporal     string         `json:"temporal,omitempty"`
	EvidenceRefs []string       `json:"evidence_refs,omitempty"`
	Confidence   float64        `json:"confidence"`
	AgentID      string         `json:"agent_id"`
	AgentRunID   string         `json:"agent_run_id,omitempty"`
	RiskFlags    []string       `json:"risk_flags,omitempty"`
	Status       ClaimStatus    `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type EvidenceRef struct {
	EvidenceID       string         `json:"evidence_id"`
	EvidenceType     EvidenceType   `json:"evidence_type"`
	Tier             EvidenceTier   `json:"tier"`
	Source           string         `json:"source"`
	Locator          string         `json:"locator"`
	RetrievedAt      time.Time      `json:"retrieved_at"`
	FreshnessScore   float64        `json:"freshness_score"`
	ReliabilityScore float64        `json:"reliability_score"`
	SourceHash       string         `json:"source_hash,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type AgentRun struct {
	AgentRunID  string         `json:"agent_run_id"`
	RequestID   string         `json:"request_id"`
	AgentID     string         `json:"agent_id"`
	AgentType   string         `json:"agent_type"`
	InputHash   string         `json:"input_hash"`
	OutputHash  string         `json:"output_hash"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Status      AgentRunStatus `json:"status"`
	Error       string         `json:"error,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type ConsensusRequest struct {
	RequestID   string         `json:"request_id"`
	WorkspaceID string         `json:"workspace_id"`
	CaseID      string         `json:"case_id,omitempty"`
	PolicyID    string         `json:"policy_id"`
	OpenedBy    string         `json:"opened_by"`
	OpenedAt    time.Time      `json:"opened_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type ConsensusPolicy struct {
	PolicyID                 string         `json:"policy_id"`
	WorkspaceID              string         `json:"workspace_id"`
	Criticality              Criticality    `json:"criticality"`
	RequiredAgents           int            `json:"required_agents"`
	RequiredTier1Count       int            `json:"required_tier1_count"`
	RequiredTier2Count       int            `json:"required_tier2_count"`
	MinSupportRatio          float64        `json:"min_support_ratio"`
	MaxConflictRatio         float64        `json:"max_conflict_ratio"`
	RequireHumanConfirmation bool           `json:"require_human_confirmation"`
	AllowTier3ForFacts       bool           `json:"allow_tier3_for_facts"`
	MaxAgentCount            int            `json:"max_agent_count"`
	MaxToolCalls             int            `json:"max_tool_calls"`
	MaxModelCalls            int            `json:"max_model_calls"`
	MaxWallTimeMS            int64          `json:"max_wall_time_ms"`
	CreatedAt                time.Time      `json:"created_at"`
	Metadata                 map[string]any `json:"metadata,omitempty"`
}

type ConsensusDecision struct {
	DecisionID           string         `json:"decision_id"`
	RequestID            string         `json:"request_id"`
	ClaimKey             string         `json:"claim_key"`
	Status               ClaimStatus    `json:"status"`
	AcceptedClaimIDs     []string       `json:"accepted_claim_ids,omitempty"`
	RejectedClaimIDs     []string       `json:"rejected_claim_ids,omitempty"`
	UncertainClaimIDs    []string       `json:"uncertain_claim_ids,omitempty"`
	ConflictedClaimIDs   []string       `json:"conflicted_claim_ids,omitempty"`
	SupportWeight        float64        `json:"support_weight"`
	OpposingWeight       float64        `json:"opposing_weight"`
	TotalEligibleWeight  float64        `json:"total_eligible_weight"`
	SupportRatio         float64        `json:"support_ratio"`
	ConflictRatio        float64        `json:"conflict_ratio"`
	QuorumMet            bool           `json:"quorum_met"`
	EvidencePolicyPassed bool           `json:"evidence_policy_passed"`
	RiskPolicyPassed     bool           `json:"risk_policy_passed"`
	DecisionReason       string         `json:"decision_reason"`
	CreatedAt            time.Time      `json:"created_at"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

type ConsensusReport struct {
	ReportID           string              `json:"report_id"`
	RequestID          string              `json:"request_id"`
	WorkspaceID        string              `json:"workspace_id"`
	CaseID             string              `json:"case_id,omitempty"`
	PolicyID           string              `json:"policy_id"`
	Decisions          []ConsensusDecision `json:"decisions"`
	AcceptedClaimIDs   []string            `json:"accepted_claim_ids,omitempty"`
	UncertainClaimIDs  []string            `json:"uncertain_claim_ids,omitempty"`
	RejectedClaimIDs   []string            `json:"rejected_claim_ids,omitempty"`
	ConflictedClaimIDs []string            `json:"conflicted_claim_ids,omitempty"`
	Escalations        []string            `json:"escalations,omitempty"`
	Summary            string              `json:"summary"`
	CreatedAt          time.Time           `json:"created_at"`
	JournalRefs        []string            `json:"journal_refs,omitempty"`
	Metadata           map[string]any      `json:"metadata,omitempty"`
}

type ResponseCompositionInput struct {
	InputID                 string         `json:"input_id"`
	ReportID                string         `json:"report_id"`
	RequestID               string         `json:"request_id"`
	WorkspaceID             string         `json:"workspace_id"`
	AcceptedClaims          []Claim        `json:"accepted_claims,omitempty"`
	UncertainClaims         []Claim        `json:"uncertain_claims,omitempty"`
	ApprovedActionProposals []Claim        `json:"approved_action_proposals,omitempty"`
	MemoryUpdateProposals   []Claim        `json:"memory_update_proposals,omitempty"`
	StyleConstraints        []string       `json:"style_constraints,omitempty"`
	UserCurrentTurnText     string         `json:"user_current_turn_text,omitempty"`
	ResponseTrace           []string       `json:"response_trace,omitempty"`
	CreatedAt               time.Time      `json:"created_at"`
	Metadata                map[string]any `json:"metadata,omitempty"`
}

type ReportListFilter struct {
	WorkspaceID string
	RequestID   string
	CaseID      string
}

type ClaimListFilter struct {
	WorkspaceID string
	RequestID   string
	Status      ClaimStatus
}
