package forgek

import (
	"errors"
	"time"
)

const (
	ObjectTypeWorkspace         = "Workspace"
	ObjectTypeCasePacket        = "CasePacket"
	ObjectTypeCapability        = "Capability"
	ObjectTypeJournal           = "JournalEvent"
	ObjectTypeExhibit           = "Exhibit"
	ObjectTypeRuling            = "Ruling"
	ObjectTypeContradiction     = "Contradiction"
	ObjectTypeSupersession      = "Supersession"
	ObjectTypeMemoryRoom        = "MemoryRoom"
	ObjectTypeMemoryAnchor      = "MemoryAnchor"
	ObjectTypePalaceRoute       = "PalaceRoute"
	ObjectTypeCandidate         = "CandidateObject"
	ObjectTypeSemanticObject    = "SemanticObject"
	ObjectTypeSemanticOperation = "SemanticOperation"
	ObjectTypeSnapshot          = "Snapshot"
	ObjectTypeRestoreSeed       = "RestoreSeed"
	ObjectTypeContextBlock      = "ContextBlock"
	ObjectTypeContextBundle     = "ContextBundle"
	ObjectTypeKVCacheManifest   = "KVCacheManifest"
	ObjectTypeRuntimeDriver     = "RuntimeDriverManifest"
	ObjectTypeRuntimeResult     = "RuntimeGenerateResult"
	ObjectTypeMaintenanceReport = "MaintenanceReport"
	ObjectTypeCleanupProposal   = "CleanupProposal"
)

const (
	AuthorityProposal     = "PROPOSAL"
	AuthorityValidated    = "VALIDATED"
	AuthorityAdmitted     = "ADMITTED"
	AuthorityCompiled     = "COMPILED"
	AuthorityCommitted    = "COMMITTED"
	AuthorityShape        = "SHAPE"
	AuthorityAcceleration = "ACCELERATION"
	AuthorityDriver       = "DRIVER_BOUNDARY"
	AuthoritySuperseded   = "SUPERSEDED"
	AuthorityExpired      = "EXPIRED"
)

const (
	CaseStatusOpen    = "OPEN"
	CaseStatusUpdated = "UPDATED"
	CaseStatusClosed  = "CLOSED"
)

const (
	MutationScopeNone      = "none"
	MutationScopeCanonical = "canonical"
)

const (
	SyscallCaseOpen                      = "case.open"
	SyscallCaseUpdate                    = "case.update"
	SyscallCaseClose                     = "case.close"
	SyscallObjectGet                     = "object.get"
	SyscallObjectList                    = "object.list"
	SyscallCapabilityGrant               = "capability.grant"
	SyscallCourtSubmit                   = "court.submit"
	SyscallCourtAdmit                    = "court.admit"
	SyscallCourtReject                   = "court.reject"
	SyscallCourtRule                     = "court.rule"
	SyscallCourtRegisterContradiction    = "court.register_contradiction"
	SyscallCourtRegisterSupersession     = "court.register_supersession"
	SyscallCourtListExhibits             = "court.list_exhibits"
	SyscallCourtListRulings              = "court.list_rulings"
	SyscallCourtListContradictions       = "court.list_contradictions"
	SyscallPalaceCreateRoom              = "palace.create_room"
	SyscallPalaceUpdateRoom              = "palace.update_room"
	SyscallPalaceLinkRooms               = "palace.link_rooms"
	SyscallPalaceCreateAnchor            = "palace.create_anchor"
	SyscallPalaceUpdateAnchor            = "palace.update_anchor"
	SyscallPalaceLinkAnchor              = "palace.link_anchor"
	SyscallPalaceRoute                   = "palace.route"
	SyscallPalaceRecordRouteResult       = "palace.record_route_result"
	SyscallPalaceListRooms               = "palace.list_rooms"
	SyscallPalaceListAnchors             = "palace.list_anchors"
	SyscallPalaceListRoutes              = "palace.list_routes"
	SyscallPalaceGetRoom                 = "palace.get_room"
	SyscallPalaceGetAnchor               = "palace.get_anchor"
	SyscallPalaceGetRoute                = "palace.get_route"
	SyscallSemanticApply                 = "semantic.apply"
	SyscallSemanticMerge                 = "semantic.merge"
	SyscallSemanticDiff                  = "semantic.diff"
	SyscallSemanticIntersect             = "semantic.intersect"
	SyscallSemanticContradict            = "semantic.contradict"
	SyscallSemanticSupersede             = "semantic.supersede"
	SyscallSemanticCompress              = "semantic.compress"
	SyscallSemanticDerive                = "semantic.derive"
	SyscallSemanticPromote               = "semantic.promote"
	SyscallSemanticDemote                = "semantic.demote"
	SyscallSemanticExpire                = "semantic.expire"
	SyscallSemanticListOperations        = "semantic.list_operations"
	SyscallSemanticGetOperation          = "semantic.get_operation"
	SyscallSnapshotCreate                = "snapshot.create"
	SyscallSnapshotGet                   = "snapshot.get"
	SyscallSnapshotList                  = "snapshot.list"
	SyscallSnapshotSeal                  = "snapshot.seal"
	SyscallSnapshotSupersede             = "snapshot.supersede"
	SyscallSnapshotExpire                = "snapshot.expire"
	SyscallSnapshotDiff                  = "snapshot.diff"
	SyscallSnapshotRestoreSeed           = "snapshot.restore_seed"
	SyscallContextCompile                = "context.compile"
	SyscallContextCompileFromSnapshot    = "context.compile_from_snapshot"
	SyscallContextCompileFromRestoreSeed = "context.compile_from_restore_seed"
	SyscallContextGetBundle              = "context.get_bundle"
	SyscallContextListBundles            = "context.list_bundles"
	SyscallContextGetBlock               = "context.get_block"
	SyscallContextListBlocks             = "context.list_blocks"
	SyscallContextValidateLayout         = "context.validate_layout"
	SyscallContextHash                   = "context.hash"
	SyscallContextRead                   = "context.read"
	SyscallKVRegister                    = "kv.register"
	SyscallKVLookup                      = "kv.lookup"
	SyscallKVRecordHit                   = "kv.record_hit"
	SyscallKVRecordMiss                  = "kv.record_miss"
	SyscallKVInvalidate                  = "kv.invalidate"
	SyscallKVEvict                       = "kv.evict"
	SyscallKVPromote                     = "kv.promote"
	SyscallKVDemote                      = "kv.demote"
	SyscallKVGetManifest                 = "kv.get_manifest"
	SyscallKVListManifests               = "kv.list_manifests"
	SyscallKVValidateIdentity            = "kv.validate_identity"
	SyscallKVRead                        = "kv.read"
	SyscallRuntimeRegisterDriver         = "runtime.register_driver"
	SyscallRuntimeListDrivers            = "runtime.list_drivers"
	SyscallRuntimeGetDriver              = "runtime.get_driver"
	SyscallRuntimeCapabilities           = "runtime.capabilities"
	SyscallRuntimeGenerate               = "runtime.generate"
	SyscallRuntimeHealth                 = "runtime.health"
	SyscallRuntimeRead                   = "runtime.read"
	SyscallLymphRunSweep                 = "lymph.run_sweep"
	SyscallLymphGetReport                = "lymph.get_report"
	SyscallLymphListReports              = "lymph.list_reports"
	SyscallLymphGetProposal              = "lymph.get_proposal"
	SyscallLymphListProposals            = "lymph.list_proposals"
	SyscallLymphCreateProposal           = "lymph.create_proposal"
	SyscallLymphRead                     = "lymph.read"
)

const (
	JournalEventCaseOpened                     = "CASE_OPENED"
	JournalEventCaseUpdated                    = "CASE_UPDATED"
	JournalEventCaseClosed                     = "CASE_CLOSED"
	JournalEventCapabilityGranted              = "CAPABILITY_GRANTED"
	JournalEventExhibitSubmitted               = "EXHIBIT_SUBMITTED"
	JournalEventExhibitAdmitted                = "EXHIBIT_ADMITTED"
	JournalEventExhibitRejected                = "EXHIBIT_REJECTED"
	JournalEventRulingCreated                  = "RULING_CREATED"
	JournalEventContradictionRegistered        = "CONTRADICTION_REGISTERED"
	JournalEventSupersessionRegistered         = "SUPERSESSION_REGISTERED"
	JournalEventMemoryRoomCreated              = "MEMORY_ROOM_CREATED"
	JournalEventMemoryRoomUpdated              = "MEMORY_ROOM_UPDATED"
	JournalEventMemoryRoomsLinked              = "MEMORY_ROOMS_LINKED"
	JournalEventMemoryAnchorCreated            = "MEMORY_ANCHOR_CREATED"
	JournalEventMemoryAnchorUpdated            = "MEMORY_ANCHOR_UPDATED"
	JournalEventMemoryAnchorLinked             = "MEMORY_ANCHOR_LINKED"
	JournalEventPalaceRouteCreated             = "PALACE_ROUTE_CREATED"
	JournalEventPalaceRouteResultRecorded      = "PALACE_ROUTE_RESULT_RECORDED"
	JournalEventSemanticOperationApplied       = "SEMANTIC_OPERATION_APPLIED"
	JournalEventSemanticMergeApplied           = "SEMANTIC_MERGE_APPLIED"
	JournalEventSemanticDiffApplied            = "SEMANTIC_DIFF_APPLIED"
	JournalEventSemanticIntersectApplied       = "SEMANTIC_INTERSECT_APPLIED"
	JournalEventSemanticContradictionApplied   = "SEMANTIC_CONTRADICTION_APPLIED"
	JournalEventSemanticSupersessionApplied    = "SEMANTIC_SUPERSESSION_APPLIED"
	JournalEventSemanticCompressApplied        = "SEMANTIC_COMPRESS_APPLIED"
	JournalEventSemanticDeriveApplied          = "SEMANTIC_DERIVE_APPLIED"
	JournalEventSemanticPromoteApplied         = "SEMANTIC_PROMOTE_APPLIED"
	JournalEventSemanticDemoteApplied          = "SEMANTIC_DEMOTE_APPLIED"
	JournalEventSemanticExpireApplied          = "SEMANTIC_EXPIRE_APPLIED"
	JournalEventSnapshotCreated                = "SNAPSHOT_CREATED"
	JournalEventSnapshotSealed                 = "SNAPSHOT_SEALED"
	JournalEventSnapshotSuperseded             = "SNAPSHOT_SUPERSEDED"
	JournalEventSnapshotExpired                = "SNAPSHOT_EXPIRED"
	JournalEventSnapshotRestoreSeedCreated     = "SNAPSHOT_RESTORE_SEED_CREATED"
	JournalEventContextCompiled                = "CONTEXT_COMPILED"
	JournalEventContextCompiledFromSnapshot    = "CONTEXT_COMPILED_FROM_SNAPSHOT"
	JournalEventContextCompiledFromRestoreSeed = "CONTEXT_COMPILED_FROM_RESTORE_SEED"
	JournalEventKVCacheRegistered              = "KV_CACHE_REGISTERED"
	JournalEventKVCacheHit                     = "KV_CACHE_HIT"
	JournalEventKVCacheMiss                    = "KV_CACHE_MISS"
	JournalEventKVCacheInvalidated             = "KV_CACHE_INVALIDATED"
	JournalEventKVCacheEvicted                 = "KV_CACHE_EVICTED"
	JournalEventKVCachePromoted                = "KV_CACHE_PROMOTED"
	JournalEventKVCacheDemoted                 = "KV_CACHE_DEMOTED"
	JournalEventRuntimeDriverRegistered        = "RUNTIME_DRIVER_REGISTERED"
	JournalEventRuntimeGenerationRequested     = "RUNTIME_GENERATION_REQUESTED"
	JournalEventRuntimeGenerationCompleted     = "RUNTIME_GENERATION_COMPLETED"
	JournalEventRuntimeGenerationFailed        = "RUNTIME_GENERATION_FAILED"
	JournalEventLymphaticSweepCompleted        = "LYMPHATIC_SWEEP_COMPLETED"
	JournalEventLymphaticProposalCreated       = "LYMPHATIC_PROPOSAL_CREATED"
)

const (
	SyscallResultCommitted = "COMMITTED"
	SyscallResultRead      = "READ"
	SyscallResultRejected  = "REJECTED"
)

var (
	ErrUnknownSyscall         = errors.New("unknown semantic syscall")
	ErrCapabilityDenied       = errors.New("capability denied")
	ErrInvalidInput           = errors.New("invalid syscall input")
	ErrObjectNotFound         = errors.New("kernel object not found")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrProposalOnly           = errors.New("neural output is proposal-only")
	ErrJournalRequired        = errors.New("journal event required")
)

type KernelObject struct {
	ObjectID        string
	ObjectType      string
	WorkspaceID     string
	OwnerID         string
	AuthorityLevel  string
	State           map[string]any
	SourceRefs      []string
	CapabilityScope []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	JournalRefs     []string
}

type CasePacket struct {
	CaseID               string
	WorkspaceID          string
	UserIntent           string
	Summary              string
	OpenedAt             time.Time
	ClosedAt             *time.Time
	Status               string
	ObjectRefs           []string
	JournalRefs          []string
	SubmittedExhibitRefs []string
	AdmittedExhibitRefs  []string
	RejectedExhibitRefs  []string
	RulingRefs           []string
	ContradictionRefs    []string
	SupersessionRefs     []string
	PalaceRouteRefs      []string
	CandidateObjectRefs  []string
	RetrievalSummary     string
}

type Capability struct {
	CapabilityID      string
	SubjectID         string
	AllowedSyscalls   []string
	WorkspaceScope    []string
	MutationScope     string
	Expiration        *time.Time
	DelegationAllowed bool
	AuditRequired     bool
}

type JournalEvent struct {
	EventID        string
	EventType      string
	Timestamp      time.Time
	WorkspaceID    string
	CaseID         string
	ActorID        string
	SyscallName    string
	InputHash      string
	OutputHash     string
	ObjectRefs     []string
	CapabilityRefs []string
	PriorEventRefs []string
	Result         string
	Error          string
	PriorHash      string
	EventHash      string
}

type SyscallRequest struct {
	Name        string
	ActorID     string
	WorkspaceID string
	CaseID      string
	Input       map[string]any
}

type SyscallResult struct {
	Success      bool
	SyscallName  string
	ObjectID     string
	JournalEvent string
	Output       any
	Error        error
}

type NeuralProposal struct {
	ProposalID  string
	ActorID     string
	WorkspaceID string
	Summary     string
}
