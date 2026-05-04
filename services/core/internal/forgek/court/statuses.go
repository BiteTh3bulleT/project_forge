package court

const (
	StatusSubmitted    = "SUBMITTED"
	StatusAdmitted     = "ADMITTED"
	StatusRejected     = "REJECTED"
	StatusContradicted = "CONTRADICTED"
	StatusSuperseded   = "SUPERSEDED"
	StatusExpired      = "EXPIRED"
)

const (
	SourceTypeNeuronEnvelope = "NeuronEnvelope"
	SourceTypeKernelObject   = "KernelObject"
	SourceTypeArtifactRef    = "ArtifactRef"
	SourceTypeManual         = "Manual"
)

const (
	RulingAdmission     = "ADMISSION"
	RulingRejection     = "REJECTION"
	RulingContradiction = "CONTRADICTION"
	RulingSupersession  = "SUPERSESSION"
	RulingCaseSummary   = "CASE_SUMMARY"
)

const (
	ContradictionOpen       = "OPEN"
	ContradictionResolved   = "RESOLVED"
	ContradictionSuperseded = "SUPERSEDED"
)
