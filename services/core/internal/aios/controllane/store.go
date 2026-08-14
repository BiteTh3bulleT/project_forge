package controllane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/commitproof"
	"forge/projectforge/services/core/internal/forgekernel/court"
	forgejournal "forge/projectforge/services/core/internal/forgekernel/journal"
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
	Action                 domain.SemanticActionType    `json:"action"`
	RequestFingerprint     string                       `json:"requestFingerprint"`
	IdempotencyFingerprint string                       `json:"idempotencyFingerprint"`
	Result                 domain.SyscallResult         `json:"result"`
	Request                domain.SyscallRequest        `json:"request"`
	Plan                   commitproof.PreparedPlan     `json:"plan"`
	Seal                   commitproof.PreparedPlanSeal `json:"seal"`
	Receipt                commitproof.CommitReceipt    `json:"receipt"`
	CreatedAt              int64                        `json:"createdAt"`
	CorrelationID          string                       `json:"correlationId"`
}

type AuditOutboxRecord struct {
	ID                 string                    `json:"id"`
	SyscallID          string                    `json:"syscallId"`
	RequestFingerprint string                    `json:"requestFingerprint"`
	Action             domain.SemanticActionType `json:"action"`
	WorkspaceID        string                    `json:"workspaceId"`
	LaneID             string                    `json:"laneId"`
	CorrelationID      string                    `json:"correlationId"`
	TraceID            string                    `json:"traceId"`
	Success            bool                      `json:"success"`
	Result             domain.SyscallResult      `json:"result"`
	Receipt            commitproof.CommitReceipt `json:"receipt"`
	CreatedAt          int64                     `json:"createdAt"`
	CommittedBy        string                    `json:"committedBy"`
}

var (
	ErrInvalidIdempotencyRecord = errors.New("invalid idempotency record")
	ErrIdempotencyConflict      = errors.New("idempotency key fingerprint conflict")
	ErrInvalidAuditOutboxRecord = errors.New("invalid audit outbox record")
	ErrAuditOutboxConflict      = errors.New("audit outbox record conflict")
)

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
	GetAuditOutbox(id string) (AuditOutboxRecord, bool)
	ListAuditOutbox(limit int) []AuditOutboxRecord
	FindLatestContextSnapshot(scope domain.ForgeScope, query, snapshotKind string) (domain.ContextPacket, bool)
	ListContextSnapshots(scope domain.ForgeScope, query, snapshotKind string, limit int) []domain.ContextPacket
	FindCourtExhibit(id string, scope domain.ForgeScope) (court.Exhibit, bool)
	FindCourtRuling(id string, scope domain.ForgeScope) (court.Ruling, bool)
	ListCourtRulings(scope domain.ForgeScope, caseID, exhibitID string) []court.Ruling
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
	CreateCourtDecision(exhibit court.Exhibit, ruling court.Ruling, appeal *court.Appeal) error
	SetIdempotency(key string, rec IdempotencyRecord) error
	CreateAuditOutbox(rec AuditOutboxRecord) error
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
	courtExhibits    map[string]court.Exhibit
	courtRulings     map[string]court.Ruling
	courtAppeals     map[string]court.Appeal
	idempotency      map[string]IdempotencyRecord
	auditOutbox      map[string]AuditOutboxRecord
	journalEntries   []forgejournal.Entry
	journalHead      forgejournal.Head
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
		courtExhibits:    map[string]court.Exhibit{},
		courtRulings:     map[string]court.Ruling{},
		courtAppeals:     map[string]court.Appeal{},
		idempotency:      map[string]IdempotencyRecord{},
		auditOutbox:      map[string]AuditOutboxRecord{},
		journalEntries:   []forgejournal.Entry{},
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
	for k, v := range in.courtExhibits {
		out.courtExhibits[k] = v
	}
	for k, v := range in.courtRulings {
		out.courtRulings[k] = v
	}
	for k, v := range in.courtAppeals {
		out.courtAppeals[k] = v
	}
	for k, v := range in.idempotency {
		out.idempotency[k] = cloneIdempotencyRecord(v)
	}
	for k, v := range in.auditOutbox {
		out.auditOutbox[k] = cloneAuditOutboxRecord(v)
	}
	out.journalEntries = make([]forgejournal.Entry, len(in.journalEntries))
	for i, entry := range in.journalEntries {
		entry.SelectedPaths = append([]string{}, entry.SelectedPaths...)
		out.journalEntries[i] = entry
	}
	out.journalHead = in.journalHead
	return out
}

type InMemorySemanticStore struct {
	mu    sync.RWMutex
	state memoryState
	meta  CommitMetadata
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

func (s *InMemorySemanticStore) SetCommitMetadata(meta CommitMetadata) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.meta = meta
}

func (s *InMemorySemanticStore) Append(ctx context.Context, evt domain.JournalEvent) error {
	_, err := s.AppendWithEvidence(ctx, evt)
	return err
}

func (s *InMemorySemanticStore) AppendWithEvidence(_ context.Context, evt domain.JournalEvent) (JournalAppendEvidence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return appendMemoryJournal(&s.state, s.meta, evt)
}

func (s *InMemorySemanticStore) JournalChainHead(_ context.Context) (forgejournal.Head, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.journalHead, nil
}

func (s *InMemorySemanticStore) VerifyJournalChain(_ context.Context) (forgejournal.VerificationReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := cloneJournalEntries(s.state.journalEntries)
	head := s.state.journalHead
	return forgejournal.Verify(entries, &head), nil
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
	return cloneIdempotencyRecord(v), ok
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

func (s *InMemorySemanticStore) FindCourtExhibit(id string, scope domain.ForgeScope) (court.Exhibit, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exhibit, ok := s.state.courtExhibits[id]
	return exhibit, ok && scopeMatchesBuildContext(scope, exhibit.Scope)
}

func (s *InMemorySemanticStore) FindCourtRuling(id string, scope domain.ForgeScope) (court.Ruling, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ruling, ok := s.state.courtRulings[id]
	return ruling, ok && scopeMatchesBuildContext(scope, ruling.Scope)
}

func (s *InMemorySemanticStore) ListCourtRulings(scope domain.ForgeScope, caseID, exhibitID string) []court.Ruling {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return filterCourtRulings(s.state.courtRulings, scope, caseID, exhibitID)
}

func (s *InMemorySemanticStore) SetIdempotency(key string, rec IdempotencyRecord) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return ErrInvalidIdempotencyRecord
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.state.idempotency[key]; ok {
		if idempotencyRecordsMatch(existing, rec) {
			return nil
		}
		return fmt.Errorf("%w: key %q", ErrIdempotencyConflict, key)
	}
	if err := validateIdempotencyRecord(key, rec); err != nil {
		return err
	}
	s.state.idempotency[key] = cloneIdempotencyRecord(rec)
	return nil
}

func (s *InMemorySemanticStore) GetAuditOutbox(id string) (AuditOutboxRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.state.auditOutbox[strings.TrimSpace(id)]
	return cloneAuditOutboxRecord(rec), ok
}

func (s *InMemorySemanticStore) ListAuditOutbox(limit int) []AuditOutboxRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return listAuditOutboxRecords(s.state.auditOutbox, limit)
}

func (s *InMemorySemanticStore) CreateAuditOutbox(rec AuditOutboxRecord) error {
	rec = normalizeAuditOutboxRecord(rec)
	if rec.ID == "" || rec.SyscallID == "" {
		return ErrInvalidAuditOutboxRecord
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.state.auditOutbox[rec.ID]; ok {
		if auditOutboxRecordsEqual(existing, rec) {
			return nil
		}
		return fmt.Errorf("%w: id %q", ErrAuditOutboxConflict, rec.ID)
	}
	for _, existing := range s.state.auditOutbox {
		if existing.SyscallID == rec.SyscallID {
			return fmt.Errorf("%w: syscall %q", ErrAuditOutboxConflict, rec.SyscallID)
		}
	}
	if err := validateAuditOutboxRecord(rec); err != nil {
		return err
	}
	s.state.auditOutbox[rec.ID] = cloneAuditOutboxRecord(rec)
	return nil
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

func (s *InMemorySemanticStore) CreateCourtDecision(exhibit court.Exhibit, ruling court.Ruling, appeal *court.Appeal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.state.courtRulings[ruling.ID]; exists {
		return fmt.Errorf("court ruling %q already exists", ruling.ID)
	}
	if appeal == nil {
		if _, exists := s.state.courtExhibits[exhibit.ID]; exists {
			return fmt.Errorf("court exhibit %q already exists", exhibit.ID)
		}
	} else {
		current, exists := s.state.courtExhibits[exhibit.ID]
		if !exists || current.CurrentRulingID != appeal.PriorRulingID {
			return fmt.Errorf("court appeal prior ruling is not current")
		}
		if _, exists := s.state.courtAppeals[appeal.ID]; exists {
			return fmt.Errorf("court appeal %q already exists", appeal.ID)
		}
		s.state.courtAppeals[appeal.ID] = *appeal
	}
	s.state.courtExhibits[exhibit.ID] = exhibit
	s.state.courtRulings[ruling.ID] = ruling
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
	txMu sync.Mutex
}

func NewInMemoryTransactionRunner(store *InMemorySemanticStore) *InMemoryTransactionRunner {
	return &InMemoryTransactionRunner{base: store}
}

func (r *InMemoryTransactionRunner) ReadStore() SemanticReadStore {
	return r.base
}

func (r *InMemoryTransactionRunner) Run(_ context.Context, fn func(uow UnitOfWork) error) error {
	r.txMu.Lock()
	defer r.txMu.Unlock()
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
	meta  CommitMetadata
}

func (s *TransactionalSemanticStore) SetCommitMetadata(meta CommitMetadata) {
	s.meta = meta
}

func (s *TransactionalSemanticStore) Append(ctx context.Context, evt domain.JournalEvent) error {
	_, err := s.AppendWithEvidence(ctx, evt)
	return err
}

func (s *TransactionalSemanticStore) AppendWithEvidence(_ context.Context, evt domain.JournalEvent) (JournalAppendEvidence, error) {
	return appendMemoryJournal(s.state, s.meta, evt)
}

func (s *TransactionalSemanticStore) JournalChainHead(_ context.Context) (forgejournal.Head, error) {
	return s.state.journalHead, nil
}

func (s *TransactionalSemanticStore) VerifyJournalChain(_ context.Context) (forgejournal.VerificationReport, error) {
	entries := cloneJournalEntries(s.state.journalEntries)
	head := s.state.journalHead
	return forgejournal.Verify(entries, &head), nil
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
	return cloneIdempotencyRecord(v), ok
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

func (s *TransactionalSemanticStore) FindCourtExhibit(id string, scope domain.ForgeScope) (court.Exhibit, bool) {
	exhibit, ok := s.state.courtExhibits[id]
	return exhibit, ok && scopeMatchesBuildContext(scope, exhibit.Scope)
}

func (s *TransactionalSemanticStore) FindCourtRuling(id string, scope domain.ForgeScope) (court.Ruling, bool) {
	ruling, ok := s.state.courtRulings[id]
	return ruling, ok && scopeMatchesBuildContext(scope, ruling.Scope)
}

func (s *TransactionalSemanticStore) ListCourtRulings(scope domain.ForgeScope, caseID, exhibitID string) []court.Ruling {
	return filterCourtRulings(s.state.courtRulings, scope, caseID, exhibitID)
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

func (s *TransactionalSemanticStore) SetIdempotency(key string, rec IdempotencyRecord) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return ErrInvalidIdempotencyRecord
	}
	if existing, ok := s.state.idempotency[key]; ok {
		if idempotencyRecordsMatch(existing, rec) {
			return nil
		}
		return fmt.Errorf("%w: key %q", ErrIdempotencyConflict, key)
	}
	if err := validateIdempotencyRecord(key, rec); err != nil {
		return err
	}
	s.state.idempotency[key] = cloneIdempotencyRecord(rec)
	return nil
}

func (s *TransactionalSemanticStore) GetAuditOutbox(id string) (AuditOutboxRecord, bool) {
	rec, ok := s.state.auditOutbox[strings.TrimSpace(id)]
	return cloneAuditOutboxRecord(rec), ok
}

func (s *TransactionalSemanticStore) ListAuditOutbox(limit int) []AuditOutboxRecord {
	return listAuditOutboxRecords(s.state.auditOutbox, limit)
}

func (s *TransactionalSemanticStore) CreateAuditOutbox(rec AuditOutboxRecord) error {
	rec = normalizeAuditOutboxRecord(rec)
	if rec.ID == "" || rec.SyscallID == "" {
		return ErrInvalidAuditOutboxRecord
	}
	if existing, ok := s.state.auditOutbox[rec.ID]; ok {
		if auditOutboxRecordsEqual(existing, rec) {
			return nil
		}
		return fmt.Errorf("%w: id %q", ErrAuditOutboxConflict, rec.ID)
	}
	for _, existing := range s.state.auditOutbox {
		if existing.SyscallID == rec.SyscallID {
			return fmt.Errorf("%w: syscall %q", ErrAuditOutboxConflict, rec.SyscallID)
		}
	}
	if err := validateAuditOutboxRecord(rec); err != nil {
		return err
	}
	s.state.auditOutbox[rec.ID] = cloneAuditOutboxRecord(rec)
	return nil
}

func validateIdempotencyRecord(key string, rec IdempotencyRecord) error {
	if strings.TrimSpace(key) == "" || strings.TrimSpace(string(rec.Action)) == "" ||
		strings.TrimSpace(rec.RequestFingerprint) == "" || strings.TrimSpace(rec.IdempotencyFingerprint) == "" || rec.CreatedAt <= 0 {
		return ErrInvalidIdempotencyRecord
	}
	if rec.Receipt.Version != commitproof.CommitReceiptVersion || rec.Receipt.RequestFingerprint != rec.RequestFingerprint ||
		rec.Receipt.IdempotencyFingerprint != rec.IdempotencyFingerprint {
		return ErrInvalidIdempotencyRecord
	}
	if rec.Request.ID == "" || rec.Request.Action != rec.Action || rec.Plan.Action != rec.Action ||
		rec.Seal.Version != commitproof.PreparedPlanVersion || rec.Seal.RequestFingerprint != rec.RequestFingerprint {
		return ErrInvalidIdempotencyRecord
	}
	return nil
}

func idempotencyRecordsMatch(left, right IdempotencyRecord) bool {
	return left.Action == right.Action && left.RequestFingerprint == right.RequestFingerprint &&
		left.IdempotencyFingerprint == right.IdempotencyFingerprint
}

func validateAuditOutboxRecord(rec AuditOutboxRecord) error {
	if strings.TrimSpace(rec.ID) == "" || strings.TrimSpace(rec.SyscallID) == "" ||
		strings.TrimSpace(rec.RequestFingerprint) == "" || strings.TrimSpace(string(rec.Action)) == "" ||
		strings.TrimSpace(rec.WorkspaceID) == "" || strings.TrimSpace(rec.CorrelationID) == "" || rec.CreatedAt <= 0 {
		return ErrInvalidAuditOutboxRecord
	}
	if rec.Receipt.Version != commitproof.CommitReceiptVersion || rec.Receipt.RequestFingerprint != rec.RequestFingerprint ||
		rec.Receipt.AuditOutboxID != rec.ID {
		return ErrInvalidAuditOutboxRecord
	}
	return nil
}

func normalizeAuditOutboxRecord(rec AuditOutboxRecord) AuditOutboxRecord {
	rec.ID = strings.TrimSpace(rec.ID)
	rec.SyscallID = strings.TrimSpace(rec.SyscallID)
	rec.RequestFingerprint = strings.TrimSpace(rec.RequestFingerprint)
	rec.WorkspaceID = strings.TrimSpace(rec.WorkspaceID)
	rec.LaneID = strings.TrimSpace(rec.LaneID)
	rec.CorrelationID = strings.TrimSpace(rec.CorrelationID)
	rec.TraceID = strings.TrimSpace(rec.TraceID)
	rec.CommittedBy = nonEmpty(strings.TrimSpace(rec.CommittedBy), "forge_k.kernel")
	return rec
}

func auditOutboxSameIdentity(left, right AuditOutboxRecord) bool {
	return left.ID == right.ID && left.SyscallID == right.SyscallID &&
		left.RequestFingerprint == right.RequestFingerprint && left.Action == right.Action
}

func listAuditOutboxRecords(all map[string]AuditOutboxRecord, limit int) []AuditOutboxRecord {
	if limit <= 0 {
		limit = 100
	}
	out := make([]AuditOutboxRecord, 0, min(limit, len(all)))
	for _, rec := range all {
		out = append(out, cloneAuditOutboxRecord(rec))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt == out[j].CreatedAt {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt < out[j].CreatedAt
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func cloneIdempotencyRecord(rec IdempotencyRecord) IdempotencyRecord {
	return cloneIntegrityValue(rec)
}

func cloneAuditOutboxRecord(rec AuditOutboxRecord) AuditOutboxRecord {
	return cloneIntegrityValue(rec)
}

func cloneIntegrityValue[T any](value T) T {
	raw, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return value
	}
	return out
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

func (s *TransactionalSemanticStore) CreateCourtDecision(exhibit court.Exhibit, ruling court.Ruling, appeal *court.Appeal) error {
	if _, exists := s.state.courtRulings[ruling.ID]; exists {
		return fmt.Errorf("court ruling %q already exists", ruling.ID)
	}
	if appeal == nil {
		if _, exists := s.state.courtExhibits[exhibit.ID]; exists {
			return fmt.Errorf("court exhibit %q already exists", exhibit.ID)
		}
	} else {
		current, exists := s.state.courtExhibits[exhibit.ID]
		if !exists || current.CurrentRulingID != appeal.PriorRulingID {
			return fmt.Errorf("court appeal prior ruling is not current")
		}
		if _, exists := s.state.courtAppeals[appeal.ID]; exists {
			return fmt.Errorf("court appeal %q already exists", appeal.ID)
		}
		s.state.courtAppeals[appeal.ID] = *appeal
	}
	s.state.courtExhibits[exhibit.ID] = exhibit
	s.state.courtRulings[ruling.ID] = ruling
	return nil
}

func filterCourtRulings(all map[string]court.Ruling, scope domain.ForgeScope, caseID, exhibitID string) []court.Ruling {
	out := make([]court.Ruling, 0)
	for _, ruling := range all {
		if !scopeMatchesBuildContext(scope, ruling.Scope) {
			continue
		}
		if strings.TrimSpace(caseID) != "" && ruling.CaseID != strings.TrimSpace(caseID) {
			continue
		}
		if strings.TrimSpace(exhibitID) != "" && ruling.ExhibitID != strings.TrimSpace(exhibitID) {
			continue
		}
		out = append(out, ruling)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt == out[j].CreatedAt {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt < out[j].CreatedAt
	})
	return out
}

func appendMemoryJournal(state *memoryState, meta CommitMetadata, evt domain.JournalEvent) (JournalAppendEvidence, error) {
	if state == nil {
		return JournalAppendEvidence{}, fmt.Errorf("memory journal state is unavailable")
	}
	for _, existing := range state.journalEntries {
		if existing.EventID == evt.ID {
			return JournalAppendEvidence{}, fmt.Errorf("duplicate journal event id %q", evt.ID)
		}
	}
	payloadHash, err := forgejournal.HashJSON([]byte(encodeJSON(evt.Payload)))
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	provenanceHash, err := forgejournal.HashJSON([]byte(encodeJSON(evt.Provenance)))
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	metadataHash, err := forgejournal.HashJSON([]byte(encodeJSON(map[string]any{})))
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	previous := state.journalHead
	entry, err := forgejournal.PlanAppend(previous, forgejournal.AppendInput{
		EventID: evt.ID, EventType: evt.Type, Source: evt.Source, Actor: evt.Provenance.Actor,
		WorkspaceID: evt.Scope.WorkspaceID, LaneID: evt.Scope.LaneID,
		SelectedPaths: append([]string{}, evt.Scope.SelectedPaths...), CorrelationID: evt.CorrelationID,
		TraceID: nonEmpty(evt.Provenance.TraceID, meta.TraceID), ProvenanceID: provenanceID(evt.Scope, evt.Provenance),
		ProvenanceHash: provenanceHash, PayloadHash: payloadHash, MetadataHash: metadataHash,
		ProposedBy: string(meta.Source), CommittedBy: nonEmpty(meta.CommittedBy, "forge_kernel"),
		SyscallID: nonEmpty(meta.SyscallID, "legacy:"+evt.ID), CreatedAt: evt.Timestamp,
	})
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	head := forgejournal.Head{Sequence: entry.Sequence, EventID: entry.EventID, Hash: entry.Hash}
	state.journalEntries = append(state.journalEntries, entry)
	state.journalHead = head
	return JournalAppendEvidence{PreviousHead: previous, Entry: entry, Head: head}, nil
}

func cloneJournalEntries(entries []forgejournal.Entry) []forgejournal.Entry {
	out := make([]forgejournal.Entry, len(entries))
	for i, entry := range entries {
		entry.SelectedPaths = append([]string{}, entry.SelectedPaths...)
		out[i] = entry
	}
	return out
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
