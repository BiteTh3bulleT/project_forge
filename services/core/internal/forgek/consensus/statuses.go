package consensus

type ClaimType string

const (
	ClaimTypeFact                 ClaimType = "fact"
	ClaimTypePreference           ClaimType = "preference"
	ClaimTypeDecision             ClaimType = "decision"
	ClaimTypeTask                 ClaimType = "task"
	ClaimTypeEvent                ClaimType = "event"
	ClaimTypeConstraint           ClaimType = "constraint"
	ClaimTypeRecommendation       ClaimType = "recommendation"
	ClaimTypeInference            ClaimType = "inference"
	ClaimTypeUncertainty          ClaimType = "uncertainty"
	ClaimTypeActionProposal       ClaimType = "action_proposal"
	ClaimTypeMemoryUpdateProposal ClaimType = "memory_update_proposal"
)

type ClaimStatus string

const (
	StatusProposed          ClaimStatus = "proposed"
	StatusAccepted          ClaimStatus = "accepted"
	StatusRejected          ClaimStatus = "rejected"
	StatusUncertain         ClaimStatus = "uncertain"
	StatusConflicted        ClaimStatus = "conflicted"
	StatusNeedsMoreEvidence ClaimStatus = "needs_more_evidence"
	StatusDeferred          ClaimStatus = "deferred"
)

type EvidenceType string

const (
	EvidenceDBRow          EvidenceType = "db_row"
	EvidenceFileChunk      EvidenceType = "file_chunk"
	EvidenceSourceCode     EvidenceType = "source_code"
	EvidenceEmail          EvidenceType = "email"
	EvidenceCalendarEvent  EvidenceType = "calendar_event"
	EvidenceAPIResponse    EvidenceType = "api_response"
	EvidenceMemoryRecord   EvidenceType = "memory_record"
	EvidenceUserMessage    EvidenceType = "user_message"
	EvidenceSystemConfig   EvidenceType = "system_config"
	EvidenceModelInference EvidenceType = "model_inference"
)

type EvidenceTier string

const (
	EvidenceTier1 EvidenceTier = "tier_1_primary"
	EvidenceTier2 EvidenceTier = "tier_2_derived"
	EvidenceTier3 EvidenceTier = "tier_3_model_inference"
)

type AgentRunStatus string

const (
	AgentRunStarted   AgentRunStatus = "started"
	AgentRunCompleted AgentRunStatus = "completed"
	AgentRunFailed    AgentRunStatus = "failed"
)

type Criticality string

const (
	CriticalityLow      Criticality = "low"
	CriticalityMedium   Criticality = "medium"
	CriticalityHigh     Criticality = "high"
	CriticalityCritical Criticality = "critical"
)

func ValidClaimType(value ClaimType) bool {
	switch value {
	case ClaimTypeFact, ClaimTypePreference, ClaimTypeDecision, ClaimTypeTask, ClaimTypeEvent,
		ClaimTypeConstraint, ClaimTypeRecommendation, ClaimTypeInference, ClaimTypeUncertainty,
		ClaimTypeActionProposal, ClaimTypeMemoryUpdateProposal:
		return true
	default:
		return false
	}
}

func ValidClaimStatus(value ClaimStatus) bool {
	switch value {
	case StatusProposed, StatusAccepted, StatusRejected, StatusUncertain, StatusConflicted,
		StatusNeedsMoreEvidence, StatusDeferred:
		return true
	default:
		return false
	}
}

func ValidEvidenceTier(value EvidenceTier) bool {
	switch value {
	case EvidenceTier1, EvidenceTier2, EvidenceTier3:
		return true
	default:
		return false
	}
}

func ValidEvidenceType(value EvidenceType) bool {
	switch value {
	case EvidenceDBRow, EvidenceFileChunk, EvidenceSourceCode, EvidenceEmail, EvidenceCalendarEvent,
		EvidenceAPIResponse, EvidenceMemoryRecord, EvidenceUserMessage, EvidenceSystemConfig, EvidenceModelInference:
		return true
	default:
		return false
	}
}

func ValidCriticality(value Criticality) bool {
	switch value {
	case CriticalityLow, CriticalityMedium, CriticalityHigh, CriticalityCritical:
		return true
	default:
		return false
	}
}
