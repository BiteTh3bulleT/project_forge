package retrieval

import "strings"

type TrustTier string

const (
	TrustTierLocalLive    TrustTier = "local_live"
	TrustTierOfficial     TrustTier = "official"
	TrustTierCurated      TrustTier = "curated"
	TrustTierWeb          TrustTier = "web"
	TrustTierVectorRecall TrustTier = "vector_recall"
	TrustTierLowTrust     TrustTier = "low_trust"
)

type RoutingMode string

const (
	RoutingModeAnswerContext RoutingMode = "answer_context"
	RoutingModeCodeContext   RoutingMode = "code_context"
	RoutingModeAuditReview   RoutingMode = "audit_review"
	RoutingModeOperatorBrief RoutingMode = "operator_brief"
)

type SourceKind string

const (
	SourceKindLocalLive      SourceKind = "local_live_workspace"
	SourceKindCanonicalState SourceKind = "canonical_forge_state"
	SourceKindOfficial       SourceKind = "official_source"
	SourceKindCuratedDocs    SourceKind = "curated_project_docs"
	SourceKindWebSearch      SourceKind = "web_search"
	SourceKindVectorRecall   SourceKind = "vector_recall"
	SourceKindLowTrust       SourceKind = "low_trust"
)

type RejectionReason string

const (
	RejectionReasonStale             RejectionReason = "stale"
	RejectionReasonOutOfScope        RejectionReason = "out_of_scope"
	RejectionReasonLowerTrustDup     RejectionReason = "lower_trust_duplicate"
	RejectionReasonContradicted      RejectionReason = "contradicted_by_fresher_source"
	RejectionReasonWeakMatch         RejectionReason = "weak_semantic_match"
	RejectionReasonUnsupportedSource RejectionReason = "unsupported_source"
	RejectionReasonBudget            RejectionReason = "budget_contraction"
)

type EvidenceScope struct {
	WorkspaceID string `json:"workspaceId,omitempty"`
	LaneID      string `json:"laneId,omitempty"`
	Path        string `json:"path,omitempty"`
}

type Citation struct {
	Ref   string `json:"ref"`
	Start int    `json:"start,omitempty"`
	End   int    `json:"end,omitempty"`
}

type SearchEvidenceCandidate struct {
	ID            string        `json:"id"`
	SourceRef     string        `json:"sourceRef"`
	SourceKind    SourceKind    `json:"sourceKind"`
	TrustTier     TrustTier     `json:"trustTier"`
	FreshnessMs   int64         `json:"freshnessMs,omitempty"`
	Scope         EvidenceScope `json:"scope,omitempty"`
	Summary       string        `json:"summary,omitempty"`
	Citation      Citation      `json:"citation,omitempty"`
	Relevance     float64       `json:"relevance,omitempty"`
	Selected      bool          `json:"selected,omitempty"`
	SelectionNote string        `json:"selectionNote,omitempty"`
}

type RejectedSearchCandidate struct {
	CandidateID   string          `json:"candidateId"`
	SourceRef     string          `json:"sourceRef"`
	TrustTier     TrustTier       `json:"trustTier,omitempty"`
	Reason        RejectionReason `json:"reason"`
	ReplacedByRef string          `json:"replacedByRef,omitempty"`
}

type SearchEvidencePacket struct {
	ID                 string                    `json:"id"`
	WorkspaceID        string                    `json:"workspaceId"`
	Query              string                    `json:"query"`
	RoutingMode        RoutingMode               `json:"routingMode"`
	Candidates         []SearchEvidenceCandidate `json:"candidates"`
	RejectedCandidates []RejectedSearchCandidate `json:"rejectedCandidates"`
	CreatedAtMs        int64                     `json:"createdAtMs"`
}

type ContextSourceRef struct {
	SourceRef   string    `json:"sourceRef"`
	CandidateID string    `json:"candidateId,omitempty"`
	TrustTier   TrustTier `json:"trustTier"`
	Citation    Citation  `json:"citation,omitempty"`
}

type ContextPacket struct {
	ID              string                    `json:"id"`
	WorkspaceID     string                    `json:"workspaceId"`
	Query           string                    `json:"query"`
	RoutingMode     RoutingMode               `json:"routingMode"`
	SourcePacketIDs []string                  `json:"sourcePacketIds"`
	SelectedRefs    []ContextSourceRef        `json:"selectedRefs"`
	RejectedRefs    []RejectedSearchCandidate `json:"rejectedRefs"`
	TokenBudget     int                       `json:"tokenBudget"`
	CreatedAtMs     int64                     `json:"createdAtMs"`
}

type PacketValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (p SearchEvidencePacket) Validate() []PacketValidationIssue {
	var issues []PacketValidationIssue
	if strings.TrimSpace(p.ID) == "" {
		issues = append(issues, PacketValidationIssue{Field: "id", Message: "id is required"})
	}
	if strings.TrimSpace(p.WorkspaceID) == "" {
		issues = append(issues, PacketValidationIssue{Field: "workspaceId", Message: "workspaceId is required"})
	}
	if strings.TrimSpace(p.Query) == "" {
		issues = append(issues, PacketValidationIssue{Field: "query", Message: "query is required"})
	}
	if strings.TrimSpace(string(p.RoutingMode)) == "" {
		issues = append(issues, PacketValidationIssue{Field: "routingMode", Message: "routingMode is required"})
	}
	return issues
}

func (p SearchEvidencePacket) CanAuthorizeExecution() bool { return false }

func (p SearchEvidencePacket) CanWriteCanonicalMemory() bool { return false }

func (p SearchEvidencePacket) CanBypassAuthorityPlane() bool { return false }

func (p ContextPacket) Validate() []PacketValidationIssue {
	var issues []PacketValidationIssue
	if strings.TrimSpace(p.ID) == "" {
		issues = append(issues, PacketValidationIssue{Field: "id", Message: "id is required"})
	}
	if strings.TrimSpace(p.WorkspaceID) == "" {
		issues = append(issues, PacketValidationIssue{Field: "workspaceId", Message: "workspaceId is required"})
	}
	if strings.TrimSpace(p.Query) == "" {
		issues = append(issues, PacketValidationIssue{Field: "query", Message: "query is required"})
	}
	if strings.TrimSpace(string(p.RoutingMode)) == "" {
		issues = append(issues, PacketValidationIssue{Field: "routingMode", Message: "routingMode is required"})
	}
	if p.TokenBudget <= 0 {
		issues = append(issues, PacketValidationIssue{Field: "tokenBudget", Message: "tokenBudget must be positive"})
	}
	return issues
}

func (p ContextPacket) CanAuthorizeExecution() bool { return false }

func (p ContextPacket) CanWriteCanonicalMemory() bool { return false }

func (p ContextPacket) CanBypassAuthorityPlane() bool { return false }
