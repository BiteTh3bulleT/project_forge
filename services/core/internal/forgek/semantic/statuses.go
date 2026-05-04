package semantic

const (
	ObjectTypeClaim             = "CLAIM"
	ObjectTypeEvidence          = "EVIDENCE"
	ObjectTypeMemoryNode        = "MEMORY_NODE"
	ObjectTypeDecision          = "DECISION"
	ObjectTypeContradiction     = "CONTRADICTION"
	ObjectTypeOpenLoop          = "OPEN_LOOP"
	ObjectTypeGoal              = "GOAL"
	ObjectTypeConstraint        = "CONSTRAINT"
	ObjectTypeArtifact          = "ARTIFACT"
	ObjectTypeSnapshotRef       = "SNAPSHOT_REF"
	ObjectTypeContextBlockRef   = "CONTEXT_BLOCK_REF"
	ObjectTypeCasePacket        = "CASE_PACKET"
	ObjectTypeRuling            = "RULING"
	ObjectTypePrecedent         = "PRECEDENT"
	ObjectTypeExhibitRef        = "EXHIBIT_REF"
	ObjectTypeCandidateRef      = "CANDIDATE_REF"
	ObjectTypeNeuronEnvelopeRef = "NEURON_ENVELOPE_REF"
	ObjectTypeDerived           = "DERIVED"
)

const (
	AuthorityProposal   = "PROPOSAL"
	AuthorityValidated  = "VALIDATED"
	AuthorityAdmitted   = "ADMITTED"
	AuthorityCommitted  = "COMMITTED"
	AuthoritySuperseded = "SUPERSEDED"
	AuthorityExpired    = "EXPIRED"
)

const (
	AdmissibilityAdmitted = "ADMITTED"
	AdmissibilityRejected = "REJECTED"
)

const (
	OperationRetrieve   = "RETRIEVE"
	OperationSubmit     = "SUBMIT"
	OperationAdmit      = "ADMIT"
	OperationReject     = "REJECT"
	OperationMerge      = "MERGE"
	OperationDiff       = "DIFF"
	OperationIntersect  = "INTERSECT"
	OperationContradict = "CONTRADICT"
	OperationSupersede  = "SUPERSEDE"
	OperationCompress   = "COMPRESS"
	OperationDerive     = "DERIVE"
	OperationPromote    = "PROMOTE"
	OperationDemote     = "DEMOTE"
	OperationExpire     = "EXPIRE"
)

const (
	WarningCompressionCannotCreateTruth = "compression cannot create new truth"
)
