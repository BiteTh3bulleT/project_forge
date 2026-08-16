package controllane

import (
	"context"
	"database/sql"

	"forge/projectforge/services/core/internal/aios/domain"
)

type SQLiteJournalRepository struct{ store *SQLiteSemanticStore }
type SQLiteMemoryNoteRepository struct{ store *SQLiteSemanticStore }
type SQLiteSemanticLinkRepository struct{ store *SQLiteSemanticStore }
type SQLiteStateRepository struct{ store *SQLiteSemanticStore }
type SQLiteOpenLoopRepository struct{ store *SQLiteSemanticStore }
type SQLiteArtifactRefRepository struct{ store *SQLiteSemanticStore }
type SQLiteDerivedModelRepository struct{ store *SQLiteSemanticStore }
type SQLiteContradictionRepository struct{ store *SQLiteSemanticStore }
type SQLiteSupersessionRepository struct{ store *SQLiteSemanticStore }
type SQLiteContextPacketRepository struct{ store *SQLiteSemanticStore }

var (
	_ JournalRepository       = (*SQLiteJournalRepository)(nil)
	_ MemoryNoteRepository    = (*SQLiteMemoryNoteRepository)(nil)
	_ SemanticLinkRepository  = (*SQLiteSemanticLinkRepository)(nil)
	_ StateRepository         = (*SQLiteStateRepository)(nil)
	_ OpenLoopRepository      = (*SQLiteOpenLoopRepository)(nil)
	_ ArtifactRefRepository   = (*SQLiteArtifactRefRepository)(nil)
	_ DerivedModelRepository  = (*SQLiteDerivedModelRepository)(nil)
	_ ContradictionRepository = (*SQLiteContradictionRepository)(nil)
	_ SupersessionRepository  = (*SQLiteSupersessionRepository)(nil)
	_ ContextPacketRepository = (*SQLiteContextPacketRepository)(nil)
)

func NewSQLiteJournalRepository(db *sql.DB) *SQLiteJournalRepository {
	return &SQLiteJournalRepository{store: NewSQLiteSemanticStore(db)}
}

func NewSQLiteMemoryNoteRepository(db *sql.DB) *SQLiteMemoryNoteRepository {
	return &SQLiteMemoryNoteRepository{store: NewSQLiteSemanticStore(db)}
}

func NewSQLiteSemanticLinkRepository(db *sql.DB) *SQLiteSemanticLinkRepository {
	return &SQLiteSemanticLinkRepository{store: NewSQLiteSemanticStore(db)}
}

func NewSQLiteStateRepository(db *sql.DB) *SQLiteStateRepository {
	return &SQLiteStateRepository{store: NewSQLiteSemanticStore(db)}
}

func NewSQLiteOpenLoopRepository(db *sql.DB) *SQLiteOpenLoopRepository {
	return &SQLiteOpenLoopRepository{store: NewSQLiteSemanticStore(db)}
}

func NewSQLiteArtifactRefRepository(db *sql.DB) *SQLiteArtifactRefRepository {
	return &SQLiteArtifactRefRepository{store: NewSQLiteSemanticStore(db)}
}

func NewSQLiteDerivedModelRepository(db *sql.DB) *SQLiteDerivedModelRepository {
	return &SQLiteDerivedModelRepository{store: NewSQLiteSemanticStore(db)}
}

func NewSQLiteContradictionRepository(db *sql.DB) *SQLiteContradictionRepository {
	return &SQLiteContradictionRepository{store: NewSQLiteSemanticStore(db)}
}

func NewSQLiteSupersessionRepository(db *sql.DB) *SQLiteSupersessionRepository {
	return &SQLiteSupersessionRepository{store: NewSQLiteSemanticStore(db)}
}

func NewSQLiteContextPacketRepository(db *sql.DB) *SQLiteContextPacketRepository {
	return &SQLiteContextPacketRepository{store: NewSQLiteSemanticStore(db)}
}

func (r *SQLiteJournalRepository) Append(ctx context.Context, evt domain.JournalEvent) error {
	return r.store.Append(ctx, evt)
}
func (r *SQLiteJournalRepository) GetByID(ctx context.Context, id string) (domain.JournalEvent, bool, error) {
	return r.store.GetByID(ctx, id)
}
func (r *SQLiteJournalRepository) ListByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.JournalEvent, error) {
	return r.store.ListByScope(ctx, scope, limit)
}
func (r *SQLiteJournalRepository) ListByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.JournalEvent, error) {
	return r.store.ListByCorrelation(ctx, correlationID, limit)
}
func (r *SQLiteJournalRepository) ListRecent(ctx context.Context, filter RecentFilter) ([]domain.JournalEvent, error) {
	return r.store.ListRecent(ctx, filter)
}

func (r *SQLiteMemoryNoteRepository) Create(ctx context.Context, note domain.MemoryNote) error {
	return r.store.Create(ctx, note)
}
func (r *SQLiteMemoryNoteRepository) GetByID(ctx context.Context, id string) (domain.MemoryNote, bool, error) {
	return r.store.GetByIDNote(ctx, id)
}
func (r *SQLiteMemoryNoteRepository) UpdateStatus(ctx context.Context, id string, status domain.MemoryNoteStatus, metadata map[string]any, updatedAt int64) error {
	return r.store.UpdateStatus(ctx, id, status, metadata, updatedAt)
}
func (r *SQLiteMemoryNoteRepository) ListByScope(ctx context.Context, scope ScopeFilter) ([]domain.MemoryNote, error) {
	return r.store.ListByScopeNotes(ctx, scope)
}
func (r *SQLiteMemoryNoteRepository) ListByType(ctx context.Context, typ domain.MemoryNoteType, scope ScopeFilter) ([]domain.MemoryNote, error) {
	return r.store.ListByType(ctx, typ, scope)
}
func (r *SQLiteMemoryNoteRepository) ListActive(ctx context.Context, scope ScopeFilter) ([]domain.MemoryNote, error) {
	return r.store.ListActive(ctx, scope)
}
func (r *SQLiteMemoryNoteRepository) ListSuperseded(ctx context.Context, scope ScopeFilter) ([]domain.MemoryNote, error) {
	return r.store.ListSuperseded(ctx, scope)
}
func (r *SQLiteMemoryNoteRepository) Archive(ctx context.Context, id, reason string, provenance domain.Provenance, updatedAt int64) error {
	return r.store.Archive(ctx, id, reason, provenance, updatedAt)
}
func (r *SQLiteMemoryNoteRepository) FindByProvenance(ctx context.Context, actor string, scope ScopeFilter, limit int) ([]domain.MemoryNote, error) {
	return r.store.FindByProvenance(ctx, actor, scope, limit)
}

func (r *SQLiteSemanticLinkRepository) Create(ctx context.Context, link domain.SemanticLink, sourceKind, targetKind string) error {
	return r.store.CreateLinkWithKinds(ctx, link, sourceKind, targetKind)
}
func (r *SQLiteSemanticLinkRepository) GetByID(ctx context.Context, id string) (domain.SemanticLink, bool, error) {
	return r.store.GetByIDLink(ctx, id)
}
func (r *SQLiteSemanticLinkRepository) ListBySource(ctx context.Context, sourceID string, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	return r.store.ListBySource(ctx, sourceID, scope, limit)
}
func (r *SQLiteSemanticLinkRepository) ListByTarget(ctx context.Context, targetID string, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	return r.store.ListByTarget(ctx, targetID, scope, limit)
}
func (r *SQLiteSemanticLinkRepository) ListNeighborhood(ctx context.Context, objectID string, scope ScopeFilter, depth, limit int) ([]domain.SemanticLink, error) {
	return r.store.ListNeighborhood(ctx, objectID, scope, depth, limit)
}
func (r *SQLiteSemanticLinkRepository) ListByType(ctx context.Context, typ domain.SemanticLinkType, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	return r.store.ListByTypeLinks(ctx, typ, scope, limit)
}

func (r *SQLiteStateRepository) GetCurrent(ctx context.Context, key string, scope ScopeFilter) (domain.StateItem, bool, error) {
	return r.store.GetCurrent(ctx, key, scope)
}
func (r *SQLiteStateRepository) UpsertCurrent(ctx context.Context, state domain.StateItem, changedBy string, syscallID, correlationID, traceID, auditID string, metadata map[string]any) error {
	return r.store.UpsertCurrent(ctx, state, changedBy, syscallID, correlationID, traceID, auditID, metadata)
}
func (r *SQLiteStateRepository) AppendHistory(ctx context.Context, version StateVersionRecord) error {
	return r.store.AppendHistory(ctx, version)
}
func (r *SQLiteStateRepository) GetTimeline(ctx context.Context, key string, scope ScopeFilter, limit int) ([]StateVersionRecord, error) {
	return r.store.GetTimeline(ctx, key, scope, limit)
}
func (r *SQLiteStateRepository) ListCurrent(ctx context.Context, scope ScopeFilter, limit int) ([]domain.StateItem, error) {
	return r.store.ListCurrent(ctx, scope, limit)
}
func (r *SQLiteStateRepository) ListRecentlyChanged(ctx context.Context, scope ScopeFilter, limit int) ([]domain.StateItem, error) {
	return r.store.ListRecentlyChanged(ctx, scope, limit)
}
func (r *SQLiteStateRepository) ListHistoryKeys(ctx context.Context, scope ScopeFilter, limit int) ([]string, error) {
	return r.store.ListHistoryKeys(ctx, scope, limit)
}

func (r *SQLiteOpenLoopRepository) Create(ctx context.Context, loop domain.OpenLoop) error {
	prev := r.store.meta
	r.store.meta = CommitMetadata{CommittedBy: "forge_kernel"}
	defer func() { r.store.meta = prev }()
	return r.store.CreateLoop(loop)
}
func (r *SQLiteOpenLoopRepository) GetByID(ctx context.Context, id string) (domain.OpenLoop, bool, error) {
	return r.store.GetByIDLoop(ctx, id)
}
func (r *SQLiteOpenLoopRepository) UpdateState(ctx context.Context, id string, state domain.OpenLoopState, metadata map[string]any, updatedAt int64) error {
	return r.store.UpdateState(ctx, id, state, metadata, updatedAt)
}
func (r *SQLiteOpenLoopRepository) ListByState(ctx context.Context, state domain.OpenLoopState, scope ScopeFilter, limit int) ([]domain.OpenLoop, error) {
	return r.store.ListByState(ctx, state, scope, limit)
}
func (r *SQLiteOpenLoopRepository) ListByPriority(ctx context.Context, priority string, scope ScopeFilter, limit int) ([]domain.OpenLoop, error) {
	return r.store.ListByPriority(ctx, priority, scope, limit)
}
func (r *SQLiteOpenLoopRepository) ListActive(ctx context.Context, scope ScopeFilter, limit int) ([]domain.OpenLoop, error) {
	return r.store.ListActiveLoops(ctx, scope, limit)
}
func (r *SQLiteOpenLoopRepository) ListStale(ctx context.Context, cutoff int64, scope ScopeFilter, limit int) ([]domain.OpenLoop, error) {
	return r.store.ListStale(ctx, cutoff, scope, limit)
}

func (r *SQLiteArtifactRefRepository) Create(ctx context.Context, ref domain.ArtifactRef) error {
	return r.store.CreateArtifact(ctx, ref)
}
func (r *SQLiteArtifactRefRepository) GetByID(ctx context.Context, id string) (domain.ArtifactRef, bool, error) {
	return r.store.GetArtifactByID(ctx, id)
}
func (r *SQLiteArtifactRefRepository) FindByChecksum(ctx context.Context, checksum string, scope ScopeFilter, limit int) ([]domain.ArtifactRef, error) {
	return r.store.FindByChecksum(ctx, checksum, scope, limit)
}
func (r *SQLiteArtifactRefRepository) ListByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.ArtifactRef, error) {
	return r.store.ListByScopeArtifacts(ctx, scope, limit)
}
func (r *SQLiteArtifactRefRepository) ListByProvenance(ctx context.Context, actor string, scope ScopeFilter, limit int) ([]domain.ArtifactRef, error) {
	return r.store.ListByProvenanceArtifacts(ctx, actor, scope, limit)
}

func (r *SQLiteDerivedModelRepository) Create(ctx context.Context, model domain.AdaptivePolicyModel) error {
	prev := r.store.meta
	r.store.meta = CommitMetadata{CommittedBy: "forge_kernel"}
	defer func() { r.store.meta = prev }()
	return r.store.CreateModel(model)
}
func (r *SQLiteDerivedModelRepository) GetByID(ctx context.Context, id string) (domain.AdaptivePolicyModel, bool, error) {
	return r.store.GetByIDModel(ctx, id)
}
func (r *SQLiteDerivedModelRepository) UpdateStatus(ctx context.Context, id string, status domain.AdaptivePolicyModelStatus, updatedAt int64) error {
	return r.store.UpdateStatusModel(ctx, id, status, updatedAt)
}
func (r *SQLiteDerivedModelRepository) ListByStatus(ctx context.Context, status domain.AdaptivePolicyModelStatus, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error) {
	return r.store.ListByStatusModel(ctx, status, scope, limit)
}
func (r *SQLiteDerivedModelRepository) ListByType(ctx context.Context, typ string, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error) {
	return r.store.ListByTypeModel(ctx, typ, scope, limit)
}
func (r *SQLiteDerivedModelRepository) ListDerivedFrom(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error) {
	return r.store.ListDerivedFrom(ctx, objectID, scope, limit)
}

func (r *SQLiteContradictionRepository) Create(ctx context.Context, record ContradictionRecord, leftKind, rightKind string, scope ScopeFilter) error {
	return r.store.CreateContradictionWithKinds(ctx, record, leftKind, rightKind, scope)
}
func (r *SQLiteContradictionRepository) GetByID(ctx context.Context, id string) (ContradictionRecord, bool, error) {
	return r.store.GetByIDContradiction(ctx, id)
}
func (r *SQLiteContradictionRepository) ListByObject(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]ContradictionRecord, error) {
	return r.store.ListByObject(ctx, objectID, scope, limit)
}
func (r *SQLiteContradictionRepository) ListByScope(ctx context.Context, scope ScopeFilter, limit int) ([]ContradictionRecord, error) {
	return r.store.ListByScopeContradictions(ctx, scope, limit)
}

func (r *SQLiteSupersessionRepository) Create(ctx context.Context, record SupersessionRecord, oldKind, newKind string, scope ScopeFilter) error {
	return r.store.CreateSupersessionWithKinds(ctx, record, oldKind, newKind, scope)
}
func (r *SQLiteSupersessionRepository) GetByID(ctx context.Context, id string) (SupersessionRecord, bool, error) {
	return r.store.GetByIDSupersession(ctx, id)
}
func (r *SQLiteSupersessionRepository) ListByOldObject(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]SupersessionRecord, error) {
	return r.store.ListByOldObject(ctx, objectID, scope, limit)
}
func (r *SQLiteSupersessionRepository) ListByNewObject(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]SupersessionRecord, error) {
	return r.store.ListByNewObject(ctx, objectID, scope, limit)
}
func (r *SQLiteSupersessionRepository) ListByScope(ctx context.Context, scope ScopeFilter, limit int) ([]SupersessionRecord, error) {
	return r.store.ListByScopeSupersessions(ctx, scope, limit)
}
func (r *SQLiteSupersessionRepository) GetCurrentSuccessor(ctx context.Context, objectID string, scope ScopeFilter) (SupersessionRecord, bool, error) {
	return r.store.GetCurrentSuccessor(ctx, objectID, scope)
}

func (r *SQLiteContextPacketRepository) GetSnapshotByID(ctx context.Context, id string) (domain.ContextPacket, bool, error) {
	return r.store.GetSnapshotByID(ctx, id)
}
func (r *SQLiteContextPacketRepository) ListSnapshotsByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.ContextPacket, error) {
	return r.store.ListSnapshotsByScope(ctx, scope, limit)
}
func (r *SQLiteContextPacketRepository) ListSnapshotsByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.ContextPacket, error) {
	return r.store.ListSnapshotsByCorrelation(ctx, correlationID, limit)
}
func (r *SQLiteContextPacketRepository) FindLatestByQueryAndKind(ctx context.Context, scope ScopeFilter, query, snapshotKind string) (domain.ContextPacket, bool, error) {
	return r.store.FindLatestSnapshotByQueryAndKind(ctx, scope, query, snapshotKind)
}
