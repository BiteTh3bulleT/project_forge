package controllane

import (
	"context"

	"forge/projectforge/services/core/internal/aios/domain"
)

type ScopeFilter struct {
	WorkspaceID string
	LaneID      string
}

type RecentFilter struct {
	Scope ScopeFilter
	Limit int
}

type JournalRepository interface {
	Append(ctx context.Context, evt domain.JournalEvent) error
	GetByID(ctx context.Context, id string) (domain.JournalEvent, bool, error)
	ListByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.JournalEvent, error)
	ListByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.JournalEvent, error)
	ListRecent(ctx context.Context, filter RecentFilter) ([]domain.JournalEvent, error)
}

type MemoryNoteRepository interface {
	Create(ctx context.Context, note domain.MemoryNote) error
	GetByID(ctx context.Context, id string) (domain.MemoryNote, bool, error)
	UpdateStatus(ctx context.Context, id string, status domain.MemoryNoteStatus, metadata map[string]any, updatedAt int64) error
	ListByScope(ctx context.Context, scope ScopeFilter) ([]domain.MemoryNote, error)
	ListByType(ctx context.Context, typ domain.MemoryNoteType, scope ScopeFilter) ([]domain.MemoryNote, error)
	ListActive(ctx context.Context, scope ScopeFilter) ([]domain.MemoryNote, error)
	ListSuperseded(ctx context.Context, scope ScopeFilter) ([]domain.MemoryNote, error)
	Archive(ctx context.Context, id, reason string, provenance domain.Provenance, updatedAt int64) error
	FindByProvenance(ctx context.Context, actor string, scope ScopeFilter, limit int) ([]domain.MemoryNote, error)
}

type SemanticLinkRepository interface {
	Create(ctx context.Context, link domain.SemanticLink, sourceKind, targetKind string) error
	GetByID(ctx context.Context, id string) (domain.SemanticLink, bool, error)
	ListBySource(ctx context.Context, sourceID string, scope ScopeFilter, limit int) ([]domain.SemanticLink, error)
	ListByTarget(ctx context.Context, targetID string, scope ScopeFilter, limit int) ([]domain.SemanticLink, error)
	ListNeighborhood(ctx context.Context, objectID string, scope ScopeFilter, depth, limit int) ([]domain.SemanticLink, error)
	ListByType(ctx context.Context, typ domain.SemanticLinkType, scope ScopeFilter, limit int) ([]domain.SemanticLink, error)
}

type StateRepository interface {
	GetCurrent(ctx context.Context, key string, scope ScopeFilter) (domain.StateItem, bool, error)
	UpsertCurrent(ctx context.Context, state domain.StateItem, changedBy string, syscallID, correlationID, traceID, auditID string, metadata map[string]any) error
	AppendHistory(ctx context.Context, version StateVersionRecord) error
	GetTimeline(ctx context.Context, key string, scope ScopeFilter, limit int) ([]StateVersionRecord, error)
	ListCurrent(ctx context.Context, scope ScopeFilter, limit int) ([]domain.StateItem, error)
	ListRecentlyChanged(ctx context.Context, scope ScopeFilter, limit int) ([]domain.StateItem, error)
	ListHistoryKeys(ctx context.Context, scope ScopeFilter, limit int) ([]string, error)
}

type OpenLoopRepository interface {
	Create(ctx context.Context, loop domain.OpenLoop) error
	GetByID(ctx context.Context, id string) (domain.OpenLoop, bool, error)
	UpdateState(ctx context.Context, id string, state domain.OpenLoopState, metadata map[string]any, updatedAt int64) error
	ListByState(ctx context.Context, state domain.OpenLoopState, scope ScopeFilter, limit int) ([]domain.OpenLoop, error)
	ListByPriority(ctx context.Context, priority string, scope ScopeFilter, limit int) ([]domain.OpenLoop, error)
	ListActive(ctx context.Context, scope ScopeFilter, limit int) ([]domain.OpenLoop, error)
	ListStale(ctx context.Context, cutoffMillis int64, scope ScopeFilter, limit int) ([]domain.OpenLoop, error)
}

type ArtifactRefRepository interface {
	Create(ctx context.Context, ref domain.ArtifactRef) error
	GetByID(ctx context.Context, id string) (domain.ArtifactRef, bool, error)
	FindByChecksum(ctx context.Context, checksum string, scope ScopeFilter, limit int) ([]domain.ArtifactRef, error)
	ListByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.ArtifactRef, error)
	ListByProvenance(ctx context.Context, actor string, scope ScopeFilter, limit int) ([]domain.ArtifactRef, error)
}

type DerivedModelRepository interface {
	Create(ctx context.Context, model domain.AdaptivePolicyModel) error
	GetByID(ctx context.Context, id string) (domain.AdaptivePolicyModel, bool, error)
	UpdateStatus(ctx context.Context, id string, status domain.AdaptivePolicyModelStatus, updatedAt int64) error
	ListByStatus(ctx context.Context, status domain.AdaptivePolicyModelStatus, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error)
	ListByType(ctx context.Context, typ string, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error)
	ListDerivedFrom(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error)
}

type ContradictionRepository interface {
	Create(ctx context.Context, record ContradictionRecord, leftKind, rightKind string, scope ScopeFilter) error
	GetByID(ctx context.Context, id string) (ContradictionRecord, bool, error)
	ListByObject(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]ContradictionRecord, error)
	ListByScope(ctx context.Context, scope ScopeFilter, limit int) ([]ContradictionRecord, error)
}

type SupersessionRepository interface {
	Create(ctx context.Context, record SupersessionRecord, oldKind, newKind string, scope ScopeFilter) error
	GetByID(ctx context.Context, id string) (SupersessionRecord, bool, error)
	ListByOldObject(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]SupersessionRecord, error)
	ListByNewObject(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]SupersessionRecord, error)
	ListByScope(ctx context.Context, scope ScopeFilter, limit int) ([]SupersessionRecord, error)
	GetCurrentSuccessor(ctx context.Context, objectID string, scope ScopeFilter) (SupersessionRecord, bool, error)
}

type ContextPacketRepository interface {
	CreateSnapshot(ctx context.Context, pkt domain.ContextPacket, syscallID, correlationID, traceID string, metadata map[string]any) error
	GetSnapshotByID(ctx context.Context, id string) (domain.ContextPacket, bool, error)
	ListSnapshotsByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.ContextPacket, error)
	ListSnapshotsByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.ContextPacket, error)
}

type StateVersionRecord struct {
	ID            int64          `json:"id"`
	StateItemID   string         `json:"stateItemId"`
	StateKey      string         `json:"stateKey"`
	WorkspaceID   string         `json:"workspaceId"`
	LaneID        string         `json:"laneId"`
	PreviousValue map[string]any `json:"previousValue"`
	NewValue      map[string]any `json:"newValue"`
	ChangedBy     string         `json:"changedBy"`
	DerivedFrom   []string       `json:"derivedFrom"`
	SyscallID     string         `json:"syscallId"`
	AuditID       string         `json:"auditId"`
	CorrelationID string         `json:"correlationId"`
	TraceID       string         `json:"traceId"`
	ProposedBy    string         `json:"proposedBy"`
	CommittedBy   string         `json:"committedBy"`
	CreatedAt     int64          `json:"createdAt"`
	Metadata      map[string]any `json:"metadata"`
}
