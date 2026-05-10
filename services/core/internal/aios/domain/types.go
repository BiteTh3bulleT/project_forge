package domain

import "time"

// ForgeScope is the explicit workspace and path boundary for AI-OS objects.
type ForgeScope struct {
	WorkspaceID   string   `json:"workspaceId"`
	LaneID        string   `json:"laneId,omitempty"`
	SelectedPaths []string `json:"selectedPaths,omitempty"`
}

// Provenance tracks actor/source lineage for inspectability and auditability.
type Provenance struct {
	Actor     string `json:"actor"`
	ActorType string `json:"actorType"`
	Source    string `json:"source,omitempty"`
	TraceID   string `json:"traceId,omitempty"`
}

type ActorIdentity struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type ActionSource string

const (
	SourceUser       ActionSource = "user"
	SourceSystem     ActionSource = "system"
	SourceInternal   ActionSource = "internal_cell"
	SourceAdapter    ActionSource = "adapter"
	SourceFutureIRIS ActionSource = "future_iris"
	SourceTest       ActionSource = "test"
)

type JournalEvent struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Timestamp     int64          `json:"timestamp"`
	Source        string         `json:"source"`
	Scope         ForgeScope     `json:"scope"`
	Payload       map[string]any `json:"payload"`
	CorrelationID string         `json:"correlationId,omitempty"`
	Provenance    Provenance     `json:"provenance"`
}

type MemoryNoteType string

const (
	NoteFact       MemoryNoteType = "fact"
	NotePreference MemoryNoteType = "preference"
	NoteGoal       MemoryNoteType = "goal"
	NoteDecision   MemoryNoteType = "decision"
	NoteProcedure  MemoryNoteType = "procedure"
	NoteEpisode    MemoryNoteType = "episode"
	NoteOpenLoop   MemoryNoteType = "open_loop"
	NoteArtifact   MemoryNoteType = "artifact_ref"
	NotePolicy     MemoryNoteType = "policy"
	NoteSystem     MemoryNoteType = "system"
)

type MemoryNoteStatus string

const (
	NoteActive     MemoryNoteStatus = "active"
	NoteSuperseded MemoryNoteStatus = "superseded"
	NoteArchived   MemoryNoteStatus = "archived"
)

type MemoryNote struct {
	ID         string           `json:"id"`
	Type       MemoryNoteType   `json:"type"`
	Title      string           `json:"title"`
	Content    string           `json:"content"`
	Scope      ForgeScope       `json:"scope"`
	Confidence float64          `json:"confidence"`
	Status     MemoryNoteStatus `json:"status"`
	CreatedAt  int64            `json:"createdAt"`
	UpdatedAt  int64            `json:"updatedAt"`
	Provenance Provenance       `json:"provenance"`
}

type SemanticLinkType string

const (
	LinkRelatesTo   SemanticLinkType = "relates_to"
	LinkSupports    SemanticLinkType = "supports"
	LinkContradicts SemanticLinkType = "contradicts"
	LinkSupersedes  SemanticLinkType = "supersedes"
	LinkDependsOn   SemanticLinkType = "depends_on"
	LinkCauses      SemanticLinkType = "causes"
	LinkAbout       SemanticLinkType = "about"
	LinkDerivedFrom SemanticLinkType = "derived_from"
	LinkBlocks      SemanticLinkType = "blocks"
	LinkResolves    SemanticLinkType = "resolves"
)

type SemanticLink struct {
	ID         string           `json:"id"`
	Type       SemanticLinkType `json:"type"`
	SourceID   string           `json:"sourceId"`
	TargetID   string           `json:"targetId"`
	Scope      ForgeScope       `json:"scope"`
	Confidence float64          `json:"confidence"`
	Provenance Provenance       `json:"provenance"`
	CreatedAt  int64            `json:"createdAt"`
}

type StateItemStatus string

const (
	StateActive     StateItemStatus = "active"
	StateSuperseded StateItemStatus = "superseded"
	StateArchived   StateItemStatus = "archived"
)

type StateItem struct {
	ID          string          `json:"id"`
	Key         string          `json:"key"`
	Value       map[string]any  `json:"value"`
	Scope       ForgeScope      `json:"scope"`
	Status      StateItemStatus `json:"status"`
	DerivedFrom []string        `json:"derivedFrom"`
	UpdatedAt   int64           `json:"updatedAt"`
}

type OpenLoopState string

const (
	LoopOpen       OpenLoopState = "open"
	LoopInProgress OpenLoopState = "in_progress"
	LoopBlocked    OpenLoopState = "blocked"
	LoopResolved   OpenLoopState = "resolved"
	LoopArchived   OpenLoopState = "archived"
)

type OpenLoop struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	State        OpenLoopState `json:"state"`
	Scope        ForgeScope    `json:"scope"`
	Priority     string        `json:"priority"`
	Owner        string        `json:"owner"`
	Blocker      string        `json:"blocker"`
	NextAction   string        `json:"nextAction"`
	RelatedNotes []string      `json:"relatedNotes"`
	CreatedFrom  string        `json:"createdFrom"`
	CreatedAt    int64         `json:"createdAt"`
	UpdatedAt    int64         `json:"updatedAt"`
}

type ArtifactRef struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	URI         string         `json:"uri"`
	Scope       ForgeScope     `json:"scope"`
	ContentHash string         `json:"contentHash"`
	CreatedAt   int64          `json:"createdAt"`
	Provenance  Provenance     `json:"provenance"`
	Metadata    map[string]any `json:"metadata"`
}

type AdaptivePolicyModelStatus string

const (
	ModelProvisional AdaptivePolicyModelStatus = "provisional"
	ModelPromoted    AdaptivePolicyModelStatus = "promoted"
	ModelDeprecated  AdaptivePolicyModelStatus = "deprecated"
)

type AdaptivePolicyModel struct {
	ID              string                    `json:"id"`
	Type            string                    `json:"type"`
	Expression      any                       `json:"expression"`
	DerivedFrom     []string                  `json:"derivedFrom"`
	SupportCount    int                       `json:"supportCount"`
	Confidence      float64                   `json:"confidence"`
	Status          AdaptivePolicyModelStatus `json:"status"`
	Scope           ForgeScope                `json:"scope"`
	LastValidatedAt *int64                    `json:"lastValidatedAt"`
	CreatedAt       int64                     `json:"createdAt"`
}

type ContextBudget struct {
	MaxTokens int `json:"maxTokens"`
	MaxEvents int `json:"maxEvents"`
	MaxNotes  int `json:"maxNotes"`
}

type ContextCompileOptions struct {
	PersistSnapshot    bool   `json:"persistSnapshot,omitempty"`
	RenderSnapshotCard bool   `json:"renderSnapshotCard,omitempty"`
	SnapshotKind       string `json:"snapshotKind,omitempty"`
}

// ContextRestoreSnapshot is non-canonical evidence describing a restored snapshot.
type ContextRestoreSnapshot struct {
	SnapshotID   string         `json:"snapshotId,omitempty"`
	SnapshotKind string         `json:"snapshotKind"`
	Evidence     map[string]any `json:"evidence,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type ContextPacket struct {
	ID               string                  `json:"id"`
	Query            string                  `json:"query"`
	Scope            ForgeScope              `json:"scope"`
	CompileOptions   *ContextCompileOptions  `json:"compileOptions,omitempty"`
	RestoreSnapshot  *ContextRestoreSnapshot `json:"restoreSnapshot,omitempty"`
	ActiveState      []StateItem             `json:"activeState"`
	OpenLoops        []OpenLoop              `json:"openLoops"`
	Notes            []MemoryNote            `json:"notes"`
	LinkedNotes      []SemanticLink          `json:"linkedNotes"`
	Models           []AdaptivePolicyModel   `json:"models"`
	Artifacts        []ArtifactRef           `json:"artifacts"`
	RawEvents        []JournalEvent          `json:"rawEvents"`
	Budget           ContextBudget           `json:"budget"`
	InclusionReasons map[string]string       `json:"inclusionReasons"`
	CreatedAt        int64                   `json:"createdAt"`
}

type SemanticActionType string

const (
	ActionCreateNote                SemanticActionType = "CREATE_NOTE"
	ActionCreateLink                SemanticActionType = "CREATE_LINK"
	ActionUpdateState               SemanticActionType = "UPDATE_STATE"
	ActionOpenLoop                  SemanticActionType = "OPEN_LOOP"
	ActionCloseLoop                 SemanticActionType = "CLOSE_LOOP"
	ActionMarkSuperseded            SemanticActionType = "MARK_SUPERSEDED"
	ActionRegisterContradict        SemanticActionType = "REGISTER_CONTRADICTION"
	ActionDeriveModel               SemanticActionType = "DERIVE_MODEL"
	ActionArchiveNote               SemanticActionType = "ARCHIVE_NOTE"
	ActionCompileContext            SemanticActionType = "COMPILE_CONTEXT"
	ActionValidateKVIdentity        SemanticActionType = "VALIDATE_KV_IDENTITY"
	ActionValidateRefShape          SemanticActionType = "VALIDATE_REF_SHAPE"
	ActionCompareRefShape           SemanticActionType = "COMPARE_REF_SHAPE"
	ActionValidateSemanticOperation SemanticActionType = "VALIDATE_SEMANTIC_OPERATION"
)

type SyscallRequest struct {
	ID                 string             `json:"id"`
	Action             SemanticActionType `json:"action"`
	Actor              ActorIdentity      `json:"actor"`
	Source             ActionSource       `json:"source"`
	Scope              ForgeScope         `json:"scope"`
	Payload            map[string]any     `json:"payload"`
	Provenance         Provenance         `json:"provenance"`
	CorrelationID      string             `json:"correlationId,omitempty"`
	TraceID            string             `json:"traceId,omitempty"`
	IdempotencyKey     string             `json:"idempotencyKey,omitempty"`
	DryRun             bool               `json:"dryRun,omitempty"`
	RequestedAt        int64              `json:"requestedAt"`
	RequiredCapability string             `json:"requiredCapability,omitempty"`
	CapabilityHints    []string           `json:"capabilityHints,omitempty"`
	Metadata           map[string]any     `json:"metadata,omitempty"`
}

type SemanticAction = SyscallRequest

type SyscallErrorCode string

const (
	ErrInvalidAction          SyscallErrorCode = "INVALID_ACTION"
	ErrInvalidPayload         SyscallErrorCode = "INVALID_PAYLOAD"
	ErrMissingRequiredField   SyscallErrorCode = "MISSING_REQUIRED_FIELD"
	ErrInvalidScope           SyscallErrorCode = "INVALID_SCOPE"
	ErrInvalidProvenance      SyscallErrorCode = "INVALID_PROVENANCE"
	ErrInvalidStateTransition SyscallErrorCode = "INVALID_STATE_TRANSITION"
	ErrUnsupportedAction      SyscallErrorCode = "UNSUPPORTED_ACTION"
	ErrUnauthorized           SyscallErrorCode = "UNAUTHORIZED"
	ErrCapabilityDenied       SyscallErrorCode = "CAPABILITY_DENIED"
	ErrApprovalRequired       SyscallErrorCode = "APPROVAL_REQUIRED"
	ErrConflict               SyscallErrorCode = "CONFLICT"
	ErrDuplicate              SyscallErrorCode = "DUPLICATE"
	ErrNotFound               SyscallErrorCode = "NOT_FOUND"
	ErrPersistenceUnavailable SyscallErrorCode = "PERSISTENCE_UNAVAILABLE"
	ErrInternal               SyscallErrorCode = "INTERNAL_ERROR"
)

type SyscallError struct {
	Code    SyscallErrorCode `json:"code"`
	Field   string           `json:"field,omitempty"`
	Message string           `json:"message"`
}

type ValidationDetail struct {
	Layer  string         `json:"layer"`
	Passed bool           `json:"passed"`
	Issues []SyscallError `json:"issues"`
}

type ApprovalStatus string

const (
	ApprovalAllowed  ApprovalStatus = "allowed"
	ApprovalDenied   ApprovalStatus = "denied"
	ApprovalRequired ApprovalStatus = "approval_required"
)

type SyscallResult struct {
	Success              bool               `json:"success"`
	Action               SemanticActionType `json:"action"`
	RequestID            string             `json:"requestId"`
	CorrelationID        string             `json:"correlationId,omitempty"`
	TraceID              string             `json:"traceId,omitempty"`
	IdempotencyKey       string             `json:"idempotencyKey,omitempty"`
	DryRun               bool               `json:"dryRun"`
	ApprovalStatus       ApprovalStatus     `json:"approvalStatus"`
	CommittedObjectIDs   []string           `json:"committedObjectIds"`
	RejectedReasons      []SyscallError     `json:"rejectedReasons"`
	Warnings             []string           `json:"warnings"`
	AuditID              string             `json:"auditId,omitempty"`
	ValidationDetails    []ValidationDetail `json:"validationDetails"`
	StateSummary         map[string]any     `json:"stateSummary,omitempty"`
	DeterministicErrCode SyscallErrorCode   `json:"deterministicErrorCode,omitempty"`
}

type ActionResult = SyscallResult

func NowMillis() int64 {
	return time.Now().UnixMilli()
}
