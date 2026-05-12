package controllane

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"forge/projectforge/services/core/internal/aios/domain"
)

type ContradictionRecord struct {
	ID            string            `json:"id"`
	LeftID        string            `json:"leftId"`
	LeftKind      string            `json:"leftKind"`
	RightID       string            `json:"rightId"`
	RightKind     string            `json:"rightKind"`
	Reason        string            `json:"reason"`
	Severity      string            `json:"severity"`
	Confidence    float64           `json:"confidence"`
	WorkspaceID   string            `json:"workspaceId"`
	LaneID        string            `json:"laneId"`
	CorrelationID string            `json:"correlationId"`
	TraceID       string            `json:"traceId"`
	SyscallID     string            `json:"syscallId"`
	AuditID       string            `json:"auditId"`
	ProposedBy    string            `json:"proposedBy"`
	CommittedBy   string            `json:"committedBy"`
	Metadata      map[string]any    `json:"metadata"`
	CreatedAt     int64             `json:"createdAt"`
	Provenance    domain.Provenance `json:"provenance"`
}

type SupersessionRecord struct {
	ID            string            `json:"id"`
	OldID         string            `json:"oldId"`
	OldKind       string            `json:"oldKind"`
	NewID         string            `json:"newId"`
	NewKind       string            `json:"newKind"`
	Reason        string            `json:"reason"`
	WorkspaceID   string            `json:"workspaceId"`
	LaneID        string            `json:"laneId"`
	CorrelationID string            `json:"correlationId"`
	TraceID       string            `json:"traceId"`
	SyscallID     string            `json:"syscallId"`
	AuditID       string            `json:"auditId"`
	ProposedBy    string            `json:"proposedBy"`
	CommittedBy   string            `json:"committedBy"`
	Metadata      map[string]any    `json:"metadata"`
	CreatedAt     int64             `json:"createdAt"`
	Provenance    domain.Provenance `json:"provenance"`
}

type IdempotencyRecord struct {
	Action domain.SemanticActionType `json:"action"`
	Result domain.SyscallResult      `json:"result"`
}

type CommitMetadata struct {
	SyscallID     string              `json:"syscallId"`
	CorrelationID string              `json:"correlationId"`
	TraceID       string              `json:"traceId"`
	Source        domain.ActionSource `json:"source"`
	ActorID       string              `json:"actorId"`
	ActorKind     string              `json:"actorKind"`
	CommittedBy   string              `json:"committedBy"`
}

type SemanticReadStore interface {
	FindNote(id string) (domain.MemoryNote, bool)
	FindLink(id string) (domain.SemanticLink, bool)
	FindState(id string) (domain.StateItem, bool)
	FindLoop(id string) (domain.OpenLoop, bool)
	FindModel(id string) (domain.AdaptivePolicyModel, bool)
	ExistsObject(id string) bool
	FindStateByKey(key string) (domain.StateItem, bool)
	FindStateByScopeKey(scope domain.ForgeScope, key string) (domain.StateItem, bool)
	GetIdempotency(key string) (IdempotencyRecord, bool)
	FindLatestContextSnapshot(scope domain.ForgeScope, query, snapshotKind string) (domain.ContextPacket, bool)
	ListContextSnapshots(scope domain.ForgeScope, query, snapshotKind string, limit int) []domain.ContextPacket
	BuildContext(query string, scope domain.ForgeScope, budget domain.ContextBudget, now int64) domain.ContextPacket
}

type SemanticStore interface {
	SemanticReadStore
	CreateNote(note domain.MemoryNote) error
	UpdateNote(note domain.MemoryNote) error
	CreateLink(link domain.SemanticLink) error
	CreateState(state domain.StateItem) error
	CreateLoop(loop domain.OpenLoop) error
	UpdateLoop(loop domain.OpenLoop) error
	CreateModel(model domain.AdaptivePolicyModel) error
	UpdateModel(model domain.AdaptivePolicyModel) error
	CreateContradiction(record ContradictionRecord) error
	CreateSupersession(record SupersessionRecord) error
	CreateArtifactRef(ref domain.ArtifactRef) error
	CreateContextSnapshot(pkt domain.ContextPacket) error
	SetIdempotency(key string, rec IdempotencyRecord)
}

type CommitAwareStore interface {
	SetCommitMetadata(meta CommitMetadata)
}

type memoryState struct {
	notes            map[string]domain.MemoryNote
	links            map[string]domain.SemanticLink
	states           map[string]domain.StateItem
	stateByScopeKey  map[string][]string
	loops            map[string]domain.OpenLoop
	artifacts        map[string]domain.ArtifactRef
	models           map[string]domain.AdaptivePolicyModel
	contextSnapshots map[string]domain.ContextPacket
	restoreOutcomes  map[string]RestoreOutcomeEvent
	contradictions   map[string]ContradictionRecord
	supersessions    map[string]SupersessionRecord
	idempotency      map[string]IdempotencyRecord
}

func newMemoryState() memoryState {
	return memoryState{
		notes:            map[string]domain.MemoryNote{},
		links:            map[string]domain.SemanticLink{},
		states:           map[string]domain.StateItem{},
		stateByScopeKey:  map[string][]string{},
		loops:            map[string]domain.OpenLoop{},
		artifacts:        map[string]domain.ArtifactRef{},
		models:           map[string]domain.AdaptivePolicyModel{},
		contextSnapshots: map[string]domain.ContextPacket{},
		restoreOutcomes:  map[string]RestoreOutcomeEvent{},
		contradictions:   map[string]ContradictionRecord{},
		supersessions:    map[string]SupersessionRecord{},
		idempotency:      map[string]IdempotencyRecord{},
	}
}

func cloneState(in memoryState) memoryState {
	out := newMemoryState()
	for k, v := range in.notes {
		out.notes[k] = v
	}
	for k, v := range in.links {
		out.links[k] = v
	}
	for k, v := range in.states {
		out.states[k] = v
	}
	for k, v := range in.stateByScopeKey {
		cp := make([]string, len(v))
		copy(cp, v)
		out.stateByScopeKey[k] = cp
	}
	for k, v := range in.loops {
		out.loops[k] = v
	}
	for k, v := range in.artifacts {
		out.artifacts[k] = v
	}
	for k, v := range in.models {
		out.models[k] = v
	}
	for k, v := range in.contextSnapshots {
		out.contextSnapshots[k] = v
	}
	for k, v := range in.restoreOutcomes {
		out.restoreOutcomes[k] = v
	}
	for k, v := range in.contradictions {
		out.contradictions[k] = v
	}
	for k, v := range in.supersessions {
		out.supersessions[k] = v
	}
	for k, v := range in.idempotency {
		out.idempotency[k] = v
	}
	return out
}

type InMemorySemanticStore struct {
	mu    sync.RWMutex
	state memoryState
}

func NewInMemorySemanticStore() *InMemorySemanticStore {
	return &InMemorySemanticStore{state: newMemoryState()}
}

func (s *InMemorySemanticStore) snapshot() memoryState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneState(s.state)
}

func (s *InMemorySemanticStore) replace(next memoryState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = next
}

func (s *InMemorySemanticStore) FindNote(id string) (domain.MemoryNote, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	note, ok := s.state.notes[id]
	return note, ok
}

func (s *InMemorySemanticStore) FindLink(id string) (domain.SemanticLink, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	link, ok := s.state.links[id]
	return link, ok
}

func (s *InMemorySemanticStore) FindState(id string) (domain.StateItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.state.states[id]
	return state, ok
}

func (s *InMemorySemanticStore) FindLoop(id string) (domain.OpenLoop, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	loop, ok := s.state.loops[id]
	return loop, ok
}

func (s *InMemorySemanticStore) FindModel(id string) (domain.AdaptivePolicyModel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	model, ok := s.state.models[id]
	return model, ok
}

func (s *InMemorySemanticStore) ExistsObject(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.state.notes[id]; ok {
		return true
	}
	if _, ok := s.state.states[id]; ok {
		return true
	}
	if _, ok := s.state.loops[id]; ok {
		return true
	}
	if _, ok := s.state.artifacts[id]; ok {
		return true
	}
	if _, ok := s.state.models[id]; ok {
		return true
	}
	if _, ok := s.state.links[id]; ok {
		return true
	}
	if _, ok := s.state.supersessions[id]; ok {
		return true
	}
	if _, ok := s.state.contradictions[id]; ok {
		return true
	}
	return false
}

func (s *InMemorySemanticStore) FindStateByKey(key string) (domain.StateItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest domain.StateItem
	var ok bool
	for _, item := range s.state.states {
		if item.Key != key {
			continue
		}
		if !ok || item.UpdatedAt > latest.UpdatedAt {
			latest = item
			ok = true
		}
	}
	return latest, ok
}

func (s *InMemorySemanticStore) FindStateByScopeKey(scope domain.ForgeScope, key string) (domain.StateItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.state.stateByScopeKey[stateScopeKey(scope, key)]
	if len(ids) == 0 {
		return domain.StateItem{}, false
	}
	lastID := ids[len(ids)-1]
	item, ok := s.state.states[lastID]
	return item, ok
}

func (s *InMemorySemanticStore) GetIdempotency(key string) (IdempotencyRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.state.idempotency[key]
	return v, ok
}

func (s *InMemorySemanticStore) FindLatestContextSnapshot(scope domain.ForgeScope, query, snapshotKind string) (domain.ContextPacket, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	matches := filterContextSnapshots(s.state.contextSnapshots, scope, query, snapshotKind, 1)
	if len(matches) == 0 {
		return domain.ContextPacket{}, false
	}
	return matches[0], true
}

func (s *InMemorySemanticStore) ListContextSnapshots(scope domain.ForgeScope, query, snapshotKind string, limit int) []domain.ContextPacket {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return filterContextSnapshots(s.state.contextSnapshots, scope, query, snapshotKind, limit)
}

func (s *InMemorySemanticStore) SetIdempotency(key string, rec IdempotencyRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.idempotency[key] = rec
}

func (s *InMemorySemanticStore) CreateNote(note domain.MemoryNote) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.notes[note.ID]; ok {
		return fmt.Errorf("note %q already exists", note.ID)
	}
	s.state.notes[note.ID] = note
	return nil
}

func (s *InMemorySemanticStore) UpdateNote(note domain.MemoryNote) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.notes[note.ID]; !ok {
		return fmt.Errorf("note %q not found", note.ID)
	}
	s.state.notes[note.ID] = note
	return nil
}

func (s *InMemorySemanticStore) CreateLink(link domain.SemanticLink) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.links[link.ID]; ok {
		return fmt.Errorf("link %q already exists", link.ID)
	}
	s.state.links[link.ID] = link
	return nil
}

func (s *InMemorySemanticStore) CreateState(state domain.StateItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.states[state.ID]; ok {
		return fmt.Errorf("state %q already exists", state.ID)
	}
	s.state.states[state.ID] = state
	scopedKey := stateScopeKey(state.Scope, state.Key)
	s.state.stateByScopeKey[scopedKey] = append(s.state.stateByScopeKey[scopedKey], state.ID)
	return nil
}

func (s *InMemorySemanticStore) CreateLoop(loop domain.OpenLoop) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.loops[loop.ID]; ok {
		return fmt.Errorf("loop %q already exists", loop.ID)
	}
	s.state.loops[loop.ID] = loop
	return nil
}

func (s *InMemorySemanticStore) UpdateLoop(loop domain.OpenLoop) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.loops[loop.ID]; !ok {
		return fmt.Errorf("loop %q not found", loop.ID)
	}
	s.state.loops[loop.ID] = loop
	return nil
}

func (s *InMemorySemanticStore) CreateModel(model domain.AdaptivePolicyModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.models[model.ID]; ok {
		return fmt.Errorf("model %q already exists", model.ID)
	}
	s.state.models[model.ID] = model
	return nil
}

func (s *InMemorySemanticStore) UpdateModel(model domain.AdaptivePolicyModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.models[model.ID]; !ok {
		return fmt.Errorf("model %q not found", model.ID)
	}
	s.state.models[model.ID] = model
	return nil
}

func (s *InMemorySemanticStore) CreateContradiction(record ContradictionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.contradictions[record.ID]; ok {
		return fmt.Errorf("contradiction %q already exists", record.ID)
	}
	s.state.contradictions[record.ID] = record
	return nil
}

func (s *InMemorySemanticStore) CreateSupersession(record SupersessionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.supersessions[record.ID]; ok {
		return fmt.Errorf("supersession %q already exists", record.ID)
	}
	s.state.supersessions[record.ID] = record
	return nil
}

func (s *InMemorySemanticStore) CreateArtifactRef(ref domain.ArtifactRef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.artifacts[ref.ID]; ok {
		return fmt.Errorf("artifact %q already exists", ref.ID)
	}
	s.state.artifacts[ref.ID] = ref
	return nil
}

func (s *InMemorySemanticStore) CreateContextSnapshot(pkt domain.ContextPacket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state.contextSnapshots[pkt.ID]; ok {
		return fmt.Errorf("context snapshot %q already exists", pkt.ID)
	}
	s.state.contextSnapshots[pkt.ID] = pkt
	return nil
}

func (s *InMemorySemanticStore) CreateRestoreOutcome(ctx context.Context, event RestoreOutcomeEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	event = normalizeRestoreOutcomeEvent(event)
	if event.ID == "" {
		return fmt.Errorf("restore outcome id required")
	}
	if _, ok := s.state.restoreOutcomes[event.ID]; ok {
		return fmt.Errorf("restore outcome %q already exists", event.ID)
	}
	s.state.restoreOutcomes[event.ID] = event
	return nil
}

func (s *InMemorySemanticStore) GetRestoreOutcome(ctx context.Context, id string) (RestoreOutcomeEvent, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.state.restoreOutcomes[strings.TrimSpace(id)]
	return event, ok, nil
}

func (s *InMemorySemanticStore) ListRestoreOutcomes(ctx context.Context, filter RestoreOutcomeFilter) ([]RestoreOutcomeEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return filterRestoreOutcomes(s.state.restoreOutcomes, filter), nil
}

func (s *InMemorySemanticStore) UpdateRestoreOutcomeFeedback(ctx context.Context, id string, scope domain.ForgeScope, feedback RestoreOutcomeFeedback) (RestoreOutcomeEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.state.restoreOutcomes[strings.TrimSpace(id)]
	if !ok {
		return RestoreOutcomeEvent{}, restoreOutcomeNotFound(id)
	}
	if !restoreOutcomeMatchesScope(event, scope) {
		return RestoreOutcomeEvent{}, fmt.Errorf("restore outcome %q outside requested scope", strings.TrimSpace(id))
	}
	event = applyRestoreOutcomeFeedback(event, feedback)
	s.state.restoreOutcomes[event.ID] = event
	return event, nil
}

func (s *InMemorySemanticStore) BuildContext(query string, scope domain.ForgeScope, budget domain.ContextBudget, now int64) domain.ContextPacket {
	s.mu.RLock()
	defer s.mu.RUnlock()

	noteIDs := keys(s.state.notes)
	sort.Strings(noteIDs)
	notes := make([]domain.MemoryNote, 0, len(noteIDs))
	for _, id := range noteIDs {
		note := s.state.notes[id]
		if !scopeMatchesBuildContext(scope, note.Scope) || note.Status != domain.NoteActive {
			continue
		}
		notes = append(notes, note)
		if len(notes) >= budget.MaxNotes {
			break
		}
	}

	loopIDs := keys(s.state.loops)
	sort.Strings(loopIDs)
	loops := make([]domain.OpenLoop, 0, len(loopIDs))
	for _, id := range loopIDs {
		loop := s.state.loops[id]
		if !scopeMatchesBuildContext(scope, loop.Scope) || !isActiveContextLoop(loop.State) {
			continue
		}
		loops = append(loops, loop)
		if len(loops) >= budget.MaxNotes {
			break
		}
	}

	stateIDs := keys(s.state.states)
	sort.Strings(stateIDs)
	activeState := make([]domain.StateItem, 0, len(stateIDs))
	for _, id := range stateIDs {
		item := s.state.states[id]
		if !scopeMatchesBuildContext(scope, item.Scope) || item.Status == domain.StateArchived {
			continue
		}
		activeState = append(activeState, item)
		if len(activeState) >= budget.MaxNotes {
			break
		}
	}

	linkIDs := keys(s.state.links)
	sort.Strings(linkIDs)
	links := make([]domain.SemanticLink, 0, len(linkIDs))
	for _, id := range linkIDs {
		link := s.state.links[id]
		if !scopeMatchesBuildContext(scope, link.Scope) {
			continue
		}
		links = append(links, link)
		if len(links) >= budget.MaxNotes {
			break
		}
	}

	modelIDs := keys(s.state.models)
	sort.Strings(modelIDs)
	models := make([]domain.AdaptivePolicyModel, 0, len(modelIDs))
	for _, id := range modelIDs {
		model := s.state.models[id]
		if !scopeMatchesBuildContext(scope, model.Scope) {
			continue
		}
		models = append(models, model)
		if len(models) >= budget.MaxNotes {
			break
		}
	}

	artifactIDs := keys(s.state.artifacts)
	sort.Strings(artifactIDs)
	artifacts := make([]domain.ArtifactRef, 0, len(artifactIDs))
	for _, id := range artifactIDs {
		artifact := s.state.artifacts[id]
		if !scopeMatchesBuildContext(scope, artifact.Scope) || isSnapshotCardArtifact(artifact) {
			continue
		}
		artifacts = append(artifacts, artifact)
		if len(artifacts) >= budget.MaxNotes {
			break
		}
	}

	return domain.ContextPacket{
		ID:          "ctx-" + strings.ReplaceAll(query, " ", "_") + "-" + fmt.Sprintf("%d", now),
		Query:       query,
		Scope:       scope,
		ActiveState: activeState,
		OpenLoops:   loops,
		Notes:       notes,
		LinkedNotes: links,
		Models:      models,
		Artifacts:   artifacts,
		RawEvents:   []domain.JournalEvent{},
		Budget:      budget,
		InclusionReasons: map[string]string{
			"mode": "deterministic_stub",
		},
		CreatedAt: now,
	}
}

func keys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

type UnitOfWork interface {
	Store() SemanticStore
}

type txUnitOfWork struct {
	store SemanticStore
}

func (u *txUnitOfWork) Store() SemanticStore { return u.store }

type TransactionRunner interface {
	Run(ctx context.Context, fn func(uow UnitOfWork) error) error
	ReadStore() SemanticReadStore
}

type InMemoryTransactionRunner struct {
	base *InMemorySemanticStore
}

func NewInMemoryTransactionRunner(store *InMemorySemanticStore) *InMemoryTransactionRunner {
	return &InMemoryTransactionRunner{base: store}
}

func (r *InMemoryTransactionRunner) ReadStore() SemanticReadStore {
	return r.base
}

func (r *InMemoryTransactionRunner) Run(_ context.Context, fn func(uow UnitOfWork) error) error {
	next := r.base.snapshot()
	txStore := &TransactionalSemanticStore{state: &next}
	uow := &txUnitOfWork{store: txStore}
	if err := fn(uow); err != nil {
		return err
	}
	r.base.replace(next)
	return nil
}

type TransactionalSemanticStore struct {
	state *memoryState
}

func (s *TransactionalSemanticStore) FindNote(id string) (domain.MemoryNote, bool) {
	v, ok := s.state.notes[id]
	return v, ok
}

func (s *TransactionalSemanticStore) FindLink(id string) (domain.SemanticLink, bool) {
	v, ok := s.state.links[id]
	return v, ok
}

func (s *TransactionalSemanticStore) FindState(id string) (domain.StateItem, bool) {
	v, ok := s.state.states[id]
	return v, ok
}

func (s *TransactionalSemanticStore) FindLoop(id string) (domain.OpenLoop, bool) {
	v, ok := s.state.loops[id]
	return v, ok
}

func (s *TransactionalSemanticStore) FindModel(id string) (domain.AdaptivePolicyModel, bool) {
	v, ok := s.state.models[id]
	return v, ok
}

func (s *TransactionalSemanticStore) ExistsObject(id string) bool {
	if _, ok := s.state.notes[id]; ok {
		return true
	}
	if _, ok := s.state.states[id]; ok {
		return true
	}
	if _, ok := s.state.loops[id]; ok {
		return true
	}
	if _, ok := s.state.artifacts[id]; ok {
		return true
	}
	if _, ok := s.state.models[id]; ok {
		return true
	}
	if _, ok := s.state.links[id]; ok {
		return true
	}
	if _, ok := s.state.supersessions[id]; ok {
		return true
	}
	if _, ok := s.state.contradictions[id]; ok {
		return true
	}
	return false
}

func (s *TransactionalSemanticStore) FindStateByKey(key string) (domain.StateItem, bool) {
	var latest domain.StateItem
	var ok bool
	for _, item := range s.state.states {
		if item.Key != key {
			continue
		}
		if !ok || item.UpdatedAt > latest.UpdatedAt {
			latest = item
			ok = true
		}
	}
	return latest, ok
}

func (s *TransactionalSemanticStore) FindStateByScopeKey(scope domain.ForgeScope, key string) (domain.StateItem, bool) {
	ids := s.state.stateByScopeKey[stateScopeKey(scope, key)]
	if len(ids) == 0 {
		return domain.StateItem{}, false
	}
	last := ids[len(ids)-1]
	v, ok := s.state.states[last]
	return v, ok
}

func (s *TransactionalSemanticStore) GetIdempotency(key string) (IdempotencyRecord, bool) {
	v, ok := s.state.idempotency[key]
	return v, ok
}

func (s *TransactionalSemanticStore) FindLatestContextSnapshot(scope domain.ForgeScope, query, snapshotKind string) (domain.ContextPacket, bool) {
	matches := filterContextSnapshots(s.state.contextSnapshots, scope, query, snapshotKind, 1)
	if len(matches) == 0 {
		return domain.ContextPacket{}, false
	}
	return matches[0], true
}

func (s *TransactionalSemanticStore) ListContextSnapshots(scope domain.ForgeScope, query, snapshotKind string, limit int) []domain.ContextPacket {
	return filterContextSnapshots(s.state.contextSnapshots, scope, query, snapshotKind, limit)
}

func (s *TransactionalSemanticStore) CreateRestoreOutcome(ctx context.Context, event RestoreOutcomeEvent) error {
	event = normalizeRestoreOutcomeEvent(event)
	if event.ID == "" {
		return fmt.Errorf("restore outcome id required")
	}
	if _, ok := s.state.restoreOutcomes[event.ID]; ok {
		return fmt.Errorf("restore outcome %q already exists", event.ID)
	}
	s.state.restoreOutcomes[event.ID] = event
	return nil
}

func (s *TransactionalSemanticStore) GetRestoreOutcome(ctx context.Context, id string) (RestoreOutcomeEvent, bool, error) {
	event, ok := s.state.restoreOutcomes[strings.TrimSpace(id)]
	return event, ok, nil
}

func (s *TransactionalSemanticStore) ListRestoreOutcomes(ctx context.Context, filter RestoreOutcomeFilter) ([]RestoreOutcomeEvent, error) {
	return filterRestoreOutcomes(s.state.restoreOutcomes, filter), nil
}

func (s *TransactionalSemanticStore) UpdateRestoreOutcomeFeedback(ctx context.Context, id string, scope domain.ForgeScope, feedback RestoreOutcomeFeedback) (RestoreOutcomeEvent, error) {
	event, ok := s.state.restoreOutcomes[strings.TrimSpace(id)]
	if !ok {
		return RestoreOutcomeEvent{}, restoreOutcomeNotFound(id)
	}
	if !restoreOutcomeMatchesScope(event, scope) {
		return RestoreOutcomeEvent{}, fmt.Errorf("restore outcome %q outside requested scope", strings.TrimSpace(id))
	}
	event = applyRestoreOutcomeFeedback(event, feedback)
	s.state.restoreOutcomes[event.ID] = event
	return event, nil
}

func (s *TransactionalSemanticStore) SetIdempotency(key string, rec IdempotencyRecord) {
	s.state.idempotency[key] = rec
}

func filterRestoreOutcomes(all map[string]RestoreOutcomeEvent, filter RestoreOutcomeFilter) []RestoreOutcomeEvent {
	filter = normalizeRestoreOutcomeFilter(filter)
	matches := make([]RestoreOutcomeEvent, 0, filter.Limit)
	for _, event := range all {
		if filter.WorkspaceID != "" && strings.TrimSpace(event.WorkspaceID) != filter.WorkspaceID {
			continue
		}
		if filter.LaneID != "" && strings.TrimSpace(event.LaneID) != "" && strings.TrimSpace(event.LaneID) != filter.LaneID {
			continue
		}
		if filter.Query != "" && strings.TrimSpace(event.Query) != filter.Query {
			continue
		}
		if filter.SnapshotID != "" && strings.TrimSpace(event.SnapshotID) != filter.SnapshotID {
			continue
		}
		if filter.Outcome != "" && event.Outcome != filter.Outcome {
			continue
		}
		if filter.Since > 0 && event.CreatedAt < filter.Since {
			continue
		}
		matches = append(matches, event)
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].CreatedAt == matches[j].CreatedAt {
			return matches[i].ID > matches[j].ID
		}
		return matches[i].CreatedAt > matches[j].CreatedAt
	})
	if len(matches) > filter.Limit {
		matches = matches[:filter.Limit]
	}
	return matches
}

func filterContextSnapshots(all map[string]domain.ContextPacket, scope domain.ForgeScope, query, snapshotKind string, limit int) []domain.ContextPacket {
	workspaceID := strings.TrimSpace(scope.WorkspaceID)
	laneID := strings.TrimSpace(scope.LaneID)
	trimmedQuery := strings.TrimSpace(query)
	trimmedKind := strings.TrimSpace(snapshotKind)
	if limit <= 0 {
		limit = 50
	}
	matches := make([]domain.ContextPacket, 0, limit)
	for _, pkt := range all {
		if strings.TrimSpace(pkt.Scope.WorkspaceID) != workspaceID {
			continue
		}
		if laneID != "" && strings.TrimSpace(pkt.Scope.LaneID) != laneID {
			continue
		}
		if trimmedQuery != "" && strings.TrimSpace(pkt.Query) != trimmedQuery {
			continue
		}
		if trimmedKind != "" && contextSnapshotKind(pkt) != trimmedKind {
			continue
		}
		matches = append(matches, pkt)
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].CreatedAt == matches[j].CreatedAt {
			return matches[i].ID > matches[j].ID
		}
		return matches[i].CreatedAt > matches[j].CreatedAt
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}

func (s *TransactionalSemanticStore) BuildContext(query string, scope domain.ForgeScope, budget domain.ContextBudget, now int64) domain.ContextPacket {
	tmp := &InMemorySemanticStore{state: cloneState(*s.state)}
	return tmp.BuildContext(query, scope, budget, now)
}

func (s *TransactionalSemanticStore) CreateNote(note domain.MemoryNote) error {
	if _, ok := s.state.notes[note.ID]; ok {
		return fmt.Errorf("note %q already exists", note.ID)
	}
	s.state.notes[note.ID] = note
	return nil
}

func (s *TransactionalSemanticStore) UpdateNote(note domain.MemoryNote) error {
	if _, ok := s.state.notes[note.ID]; !ok {
		return fmt.Errorf("note %q not found", note.ID)
	}
	s.state.notes[note.ID] = note
	return nil
}

func (s *TransactionalSemanticStore) CreateLink(link domain.SemanticLink) error {
	if _, ok := s.state.links[link.ID]; ok {
		return fmt.Errorf("link %q already exists", link.ID)
	}
	s.state.links[link.ID] = link
	return nil
}

func (s *TransactionalSemanticStore) CreateState(state domain.StateItem) error {
	if _, ok := s.state.states[state.ID]; ok {
		return fmt.Errorf("state %q already exists", state.ID)
	}
	s.state.states[state.ID] = state
	scopedKey := stateScopeKey(state.Scope, state.Key)
	s.state.stateByScopeKey[scopedKey] = append(s.state.stateByScopeKey[scopedKey], state.ID)
	return nil
}

func (s *TransactionalSemanticStore) CreateLoop(loop domain.OpenLoop) error {
	if _, ok := s.state.loops[loop.ID]; ok {
		return fmt.Errorf("loop %q already exists", loop.ID)
	}
	s.state.loops[loop.ID] = loop
	return nil
}

func (s *TransactionalSemanticStore) UpdateLoop(loop domain.OpenLoop) error {
	if _, ok := s.state.loops[loop.ID]; !ok {
		return fmt.Errorf("loop %q not found", loop.ID)
	}
	s.state.loops[loop.ID] = loop
	return nil
}

func (s *TransactionalSemanticStore) CreateModel(model domain.AdaptivePolicyModel) error {
	if _, ok := s.state.models[model.ID]; ok {
		return fmt.Errorf("model %q already exists", model.ID)
	}
	s.state.models[model.ID] = model
	return nil
}

func (s *TransactionalSemanticStore) UpdateModel(model domain.AdaptivePolicyModel) error {
	if _, ok := s.state.models[model.ID]; !ok {
		return fmt.Errorf("model %q not found", model.ID)
	}
	s.state.models[model.ID] = model
	return nil
}

func (s *TransactionalSemanticStore) CreateContradiction(record ContradictionRecord) error {
	if _, ok := s.state.contradictions[record.ID]; ok {
		return fmt.Errorf("contradiction %q already exists", record.ID)
	}
	s.state.contradictions[record.ID] = record
	return nil
}

func (s *TransactionalSemanticStore) CreateSupersession(record SupersessionRecord) error {
	if _, ok := s.state.supersessions[record.ID]; ok {
		return fmt.Errorf("supersession %q already exists", record.ID)
	}
	s.state.supersessions[record.ID] = record
	return nil
}

func (s *TransactionalSemanticStore) CreateArtifactRef(ref domain.ArtifactRef) error {
	if _, ok := s.state.artifacts[ref.ID]; ok {
		return fmt.Errorf("artifact %q already exists", ref.ID)
	}
	s.state.artifacts[ref.ID] = ref
	return nil
}

func (s *TransactionalSemanticStore) CreateContextSnapshot(pkt domain.ContextPacket) error {
	if _, ok := s.state.contextSnapshots[pkt.ID]; ok {
		return fmt.Errorf("context snapshot %q already exists", pkt.ID)
	}
	s.state.contextSnapshots[pkt.ID] = pkt
	return nil
}

func stateScopeKey(scope domain.ForgeScope, key string) string {
	return strings.TrimSpace(scope.WorkspaceID) + "|" + strings.TrimSpace(scope.LaneID) + "|" + strings.TrimSpace(key)
}

func contextSnapshotKind(pkt domain.ContextPacket) string {
	if pkt.RestoreSnapshot != nil && strings.TrimSpace(pkt.RestoreSnapshot.SnapshotKind) != "" {
		return strings.TrimSpace(pkt.RestoreSnapshot.SnapshotKind)
	}
	if pkt.CompileOptions != nil {
		return strings.TrimSpace(pkt.CompileOptions.SnapshotKind)
	}
	return ""
}

func scopeMatchesBuildContext(target, object domain.ForgeScope) bool {
	if strings.TrimSpace(target.WorkspaceID) == "" {
		return false
	}
	if strings.TrimSpace(target.WorkspaceID) != strings.TrimSpace(object.WorkspaceID) {
		return false
	}
	targetLane := strings.TrimSpace(target.LaneID)
	if targetLane == "" {
		return true
	}
	return targetLane == strings.TrimSpace(object.LaneID)
}

func isActiveContextLoop(state domain.OpenLoopState) bool {
	switch state {
	case domain.LoopOpen, domain.LoopInProgress, domain.LoopBlocked:
		return true
	default:
		return false
	}
}
