package memory

import "encoding/json"

type Observation struct {
	ID                int64           `json:"id"`
	CreatedAtMs       int64           `json:"createdAtMs"`
	UpdatedAtMs       int64           `json:"updatedAtMs"`
	ObservedAtMs      int64           `json:"observedAtMs"`
	Type              string          `json:"type"`
	RawContent        string          `json:"rawContent"`
	Summary           string          `json:"summary"`
	EmbeddingRef      string          `json:"embeddingRef"`
	DossierID         *int64          `json:"dossierId"`
	ProjectKey        string          `json:"projectKey"`
	SourcePath        string          `json:"sourcePath"`
	Entities          json.RawMessage `json:"entities"`
	Tags              json.RawMessage `json:"tags"`
	RelatedFiles      json.RawMessage `json:"relatedFiles"`
	TaskType          string          `json:"taskType"`
	Confidence        float64         `json:"confidence"`
	VerificationState string          `json:"verificationState"`
	Lineage           json.RawMessage `json:"lineage"`
	OriginKind        string          `json:"originKind"`
	OriginID          string          `json:"originId"`
	Stale             bool            `json:"stale"`
	LastVerifiedAtMs  *int64          `json:"lastVerifiedAtMs"`
	UsefulnessScore   float64         `json:"usefulnessScore"`
	UsefulnessCount   int             `json:"usefulnessCount"`
	NoiseCount        int             `json:"noiseCount"`
}

type ObservationLink struct {
	ID                int64  `json:"id"`
	CreatedAtMs       int64  `json:"createdAtMs"`
	FromObservationID int64  `json:"fromObservationId"`
	ToObservationID   int64  `json:"toObservationId"`
	RelationType      string `json:"relationType"`
	Note              string `json:"note"`
}

type UsefulnessEvent struct {
	ID                int64   `json:"id"`
	CreatedAtMs       int64   `json:"createdAtMs"`
	ObservationID     int64   `json:"observationId"`
	RetrievalResultID *int64  `json:"retrievalResultId"`
	RetrievalRunID    *int64  `json:"retrievalRunId"`
	PacketID          *int64  `json:"packetId"`
	JobID             *string `json:"jobId"`
	Signal            string  `json:"signal"`
	Weight            float64 `json:"weight"`
	Note              string  `json:"note"`
}

type ObservationDetail struct {
	Observation
	IncomingLinks []ObservationLink     `json:"incomingLinks"`
	OutgoingLinks []ObservationLink     `json:"outgoingLinks"`
	Signals       []UsefulnessEvent     `json:"signals"`
	VSA           *ObservationVSADetail `json:"vsa,omitempty"`
}

type ObservationVSADetail struct {
	ObservationID int64             `json:"observationId"`
	Pointer       *VSAPointerRecord `json:"pointer,omitempty"`
	RoleBindings  []VSARoleBinding  `json:"roleBindings"`
	Associations  []VSAAssociation  `json:"associations"`
}

type VSAPointerRecord struct {
	ID                int64           `json:"id"`
	ObservationID     int64           `json:"observationId"`
	Dims              int             `json:"dims"`
	Pointer           json.RawMessage `json:"pointer"`
	Norm              float64         `json:"norm"`
	SourceFingerprint string          `json:"sourceFingerprint"`
	Stale             bool            `json:"stale"`
	Metadata          json.RawMessage `json:"metadata"`
	CreatedAtMs       int64           `json:"createdAtMs"`
	UpdatedAtMs       int64           `json:"updatedAtMs"`
}

type VSARoleBinding struct {
	ID            int64           `json:"id"`
	ObservationID int64           `json:"observationId"`
	Role          string          `json:"role"`
	Filler        string          `json:"filler"`
	Weight        float64         `json:"weight"`
	SupportCount  int             `json:"supportCount"`
	NoiseCount    int             `json:"noiseCount"`
	Binding       json.RawMessage `json:"binding"`
	CreatedAtMs   int64           `json:"createdAtMs"`
	UpdatedAtMs   int64           `json:"updatedAtMs"`
}

type VSAAssociation struct {
	ID                int64           `json:"id"`
	FromObservationID int64           `json:"fromObservationId"`
	ToObservationID   int64           `json:"toObservationId"`
	AssociationType   string          `json:"associationType"`
	Strength          float64         `json:"strength"`
	SupportCount      int             `json:"supportCount"`
	NoiseCount        int             `json:"noiseCount"`
	Evidence          json.RawMessage `json:"evidence"`
	CreatedAtMs       int64           `json:"createdAtMs"`
	UpdatedAtMs       int64           `json:"updatedAtMs"`
}

type VSARoleBindingSeed struct {
	Role   string
	Filler string
	Weight float64
}

type RetrievalSelection struct {
	RetrievalResultID int64           `json:"retrievalResultId"`
	Reason            json.RawMessage `json:"reason"`
	CreatedAtMs       int64           `json:"createdAtMs"`
}

type PacketAlignmentNote struct {
	ID                int64  `json:"id"`
	PacketID          int64  `json:"packetId"`
	ObservationID     *int64 `json:"observationId"`
	RetrievalResultID *int64 `json:"retrievalResultId"`
	Note              string `json:"note"`
	CreatedAtMs       int64  `json:"createdAtMs"`
}

type DossierMemoryView struct {
	DossierID             int64                 `json:"dossierId"`
	ObservationCount      int                   `json:"observationCount"`
	StaleObservationCount int                   `json:"staleObservationCount"`
	RecentObservations    []Observation         `json:"recentObservations"`
	RecentSignals         []UsefulnessEvent     `json:"recentSignals"`
	RecentAlignmentNotes  []PacketAlignmentNote `json:"recentAlignmentNotes"`
	VSASummary            *DossierVSASummary    `json:"vsaSummary,omitempty"`
}

type RepairRun struct {
	ID            int64  `json:"id"`
	CreatedAtMs   int64  `json:"createdAtMs"`
	StartedAtMs   int64  `json:"startedAtMs"`
	CompletedAtMs *int64 `json:"completedAtMs"`
	DossierID     *int64 `json:"dossierId"`
	Mode          string `json:"mode"`
	MaxAgeDays    int    `json:"maxAgeDays"`
	Candidates    int    `json:"candidates"`
	Repaired      int    `json:"repaired"`
	Skipped       int    `json:"skipped"`
	Failed        int    `json:"failed"`
	Note          string `json:"note"`
}

type RepairItem struct {
	ID            int64           `json:"id"`
	RepairRunID   int64           `json:"repairRunId"`
	ObservationID int64           `json:"observationId"`
	Status        string          `json:"status"`
	Issue         string          `json:"issue"`
	Before        json.RawMessage `json:"before"`
	After         json.RawMessage `json:"after"`
	Note          string          `json:"note"`
	CreatedAtMs   int64           `json:"createdAtMs"`
}

type RepairRunDetail struct {
	Run   RepairRun    `json:"run"`
	Items []RepairItem `json:"items"`
}

type VSAReindexRun struct {
	ID            int64           `json:"id"`
	CreatedAtMs   int64           `json:"createdAtMs"`
	StartedAtMs   int64           `json:"startedAtMs"`
	CompletedAtMs *int64          `json:"completedAtMs"`
	DossierID     *int64          `json:"dossierId"`
	Mode          string          `json:"mode"`
	Status        string          `json:"status"`
	Candidates    int             `json:"candidates"`
	Indexed       int             `json:"indexed"`
	Skipped       int             `json:"skipped"`
	Failed        int             `json:"failed"`
	TriggeredBy   string          `json:"triggeredBy"`
	Note          string          `json:"note"`
	Settings      json.RawMessage `json:"settings"`
}

type VSAReindexItem struct {
	ID                int64           `json:"id"`
	ReindexRunID      int64           `json:"reindexRunId"`
	ObservationID     int64           `json:"observationId"`
	Status            string          `json:"status"`
	Reason            string          `json:"reason"`
	BeforeFingerprint string          `json:"beforeFingerprint"`
	AfterFingerprint  string          `json:"afterFingerprint"`
	Before            json.RawMessage `json:"before"`
	After             json.RawMessage `json:"after"`
	Note              string          `json:"note"`
	CreatedAtMs       int64           `json:"createdAtMs"`
}

type VSAReindexRunDetail struct {
	Run   VSAReindexRun    `json:"run"`
	Items []VSAReindexItem `json:"items"`
}

type DossierVSASummary struct {
	DossierID        int64   `json:"dossierId"`
	PointerCount     int     `json:"pointerCount"`
	BindingCount     int     `json:"bindingCount"`
	AssociationCount int     `json:"associationCount"`
	LastReindexRunID *int64  `json:"lastReindexRunId"`
	LastReindexAtMs  *int64  `json:"lastReindexAtMs"`
	CoverageScore    float64 `json:"coverageScore"`
	Health           string  `json:"health"`
}

type RetrievalResultVSASignal struct {
	ID                int64           `json:"id"`
	RetrievalResultID int64           `json:"retrievalResultId"`
	RetrievalRunID    int64           `json:"retrievalRunId"`
	ChunkID           int64           `json:"chunkId"`
	ObservationID     *int64          `json:"observationId"`
	Mode              string          `json:"mode"`
	AssociativeScore  float64         `json:"associativeScore"`
	RoleMatchScore    float64         `json:"roleMatchScore"`
	RelationalScore   float64         `json:"relationalScore"`
	FeedbackScore     float64         `json:"feedbackScore"`
	AdditiveScore     float64         `json:"additiveScore"`
	AppliedScore      float64         `json:"appliedScore"`
	Explain           json.RawMessage `json:"explain"`
	CreatedAtMs       int64           `json:"createdAtMs"`
}

type VSAQueryCandidate struct {
	ChunkID int64  `json:"chunkId"`
	AbsPath string `json:"absPath"`
	RelPath string `json:"relPath"`
}

type VSAQuerySignalsRequest struct {
	Query      string              `json:"query"`
	DossierID  *int64              `json:"dossierId,omitempty"`
	Candidates []VSAQueryCandidate `json:"candidates"`
}

type ListObservationsRequest struct {
	Limit      int
	DossierID  *int64
	Type       string
	OriginKind string
	StaleOnly  bool
}

type RecordObservationRequest struct {
	Type              string
	RawContent        string
	Summary           string
	EmbeddingRef      string
	DossierID         *int64
	ProjectKey        string
	SourcePath        string
	Entities          []string
	Tags              []string
	RelatedFiles      []string
	TaskType          string
	Confidence        float64
	VerificationState string
	Lineage           []string
	OriginKind        string
	OriginID          string
	ObservedAtMs      int64
}

type UpdateObservationRequest struct {
	Summary           *string
	VerificationState *string
	Stale             *bool
	LastVerifiedAtMs  *int64
	Tags              []string
	RelatedFiles      []string
}

type MarkUsefulnessRequest struct {
	ObservationID     int64
	RetrievalResultID *int64
	RetrievalRunID    *int64
	PacketID          *int64
	JobID             *string
	Signal            string
	Weight            float64
	Note              string
}

type SaveSelectionReasonRequest struct {
	RetrievalResultID int64
	Reason            map[string]any
}

type AddAlignmentNoteRequest struct {
	PacketID          int64
	ObservationID     *int64
	RetrievalResultID *int64
	Note              string
}

type RunRepairRequest struct {
	DossierID  *int64
	Mode       string
	MaxAgeDays int
	Limit      int
	Note       string
}

type RunVSAReindexRequest struct {
	DossierID   *int64
	Mode        string
	Limit       int
	TriggeredBy string
	Reason      string
	Note        string
	StaleOnly   bool
	Force       bool
}

// MaintenancePreview is read-only evidence describing a maintenance action
// that would require a future governed commit path. It never authorizes or
// records a repair/reindex run and it never mutates evidence or projections.
type MaintenancePreview struct {
	Kind          string   `json:"kind"`
	DryRun        bool     `json:"dryRun"`
	ProposalOnly  bool     `json:"proposalOnly"`
	DossierID     *int64   `json:"dossierId,omitempty"`
	CandidateIDs  []int64  `json:"candidateIds"`
	Candidates    int      `json:"candidates"`
	WouldWrite    []string `json:"wouldWrite"`
	RequiresOwner string   `json:"requiresOwner"`
	Note          string   `json:"note,omitempty"`
}
