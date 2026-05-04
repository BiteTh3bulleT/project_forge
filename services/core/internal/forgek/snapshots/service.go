package snapshots

import (
	"sort"
	"sync"
	"time"
)

type ListFilter struct {
	WorkspaceID  string
	CaseID       string
	SnapshotType SnapshotType
	Status       SnapshotStatus
}

type Service struct {
	mu           sync.RWMutex
	snapshots    map[string]Snapshot
	restoreSeeds map[string]RestoreSeed
}

func NewService() *Service {
	return &Service{
		snapshots:    make(map[string]Snapshot),
		restoreSeeds: make(map[string]RestoreSeed),
	}
}

func (s *Service) CreateSnapshot(input SnapshotInput) (Snapshot, error) {
	snapshot, err := NewSnapshot(input)
	if err != nil {
		return Snapshot{}, err
	}
	s.StoreSnapshot(snapshot)
	return snapshot.Clone(), nil
}

func (s *Service) StoreSnapshot(snapshot Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots[snapshot.SnapshotID] = snapshot.Clone()
}

func (s *Service) GetSnapshot(snapshotID string) (Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot, ok := s.snapshots[snapshotID]
	if !ok {
		return Snapshot{}, false
	}
	return snapshot.Clone(), true
}

func (s *Service) ListSnapshots(filter ListFilter) []Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Snapshot, 0)
	for _, snapshot := range s.snapshots {
		if filter.WorkspaceID != "" && snapshot.WorkspaceID != filter.WorkspaceID {
			continue
		}
		if filter.CaseID != "" && snapshot.CaseID != filter.CaseID {
			continue
		}
		if filter.SnapshotType != "" && snapshot.SnapshotType != filter.SnapshotType {
			continue
		}
		if filter.Status != "" && snapshot.Status != filter.Status {
			continue
		}
		out = append(out, snapshot.Clone())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].SnapshotID < out[j].SnapshotID
	})
	return out
}

func (s *Service) SealSnapshot(snapshotID string, now time.Time, journalRef string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.snapshots[snapshotID]
	if !ok {
		return Snapshot{}, ErrSnapshotNotFound
	}
	if snapshot.Status != StatusDraft {
		return Snapshot{}, ErrInvalidStateTransition
	}
	snapshot.Status = StatusSealed
	snapshot.SealedAt = &now
	snapshot.JournalRefs = append(snapshot.JournalRefs, journalRef)
	s.snapshots[snapshotID] = snapshot.Clone()
	return snapshot.Clone(), nil
}

func (s *Service) SupersedeSnapshot(oldSnapshotID, newSnapshotID string, now time.Time, journalRef string) (SnapshotSupersessionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldSnapshot, ok := s.snapshots[oldSnapshotID]
	if !ok {
		return SnapshotSupersessionResult{}, ErrSnapshotNotFound
	}
	newSnapshot, ok := s.snapshots[newSnapshotID]
	if !ok {
		return SnapshotSupersessionResult{}, ErrSnapshotNotFound
	}
	if oldSnapshot.WorkspaceID != newSnapshot.WorkspaceID {
		return SnapshotSupersessionResult{}, ErrWorkspaceMismatch
	}
	if oldSnapshot.Status == StatusExpired || oldSnapshot.Status == StatusSuperseded {
		return SnapshotSupersessionResult{}, ErrInvalidStateTransition
	}
	oldSnapshot.Status = StatusSuperseded
	oldSnapshot.SupersededBy = newSnapshotID
	oldSnapshot.JournalRefs = append(oldSnapshot.JournalRefs, journalRef)
	newSnapshot.Supersedes = appendUnique(newSnapshot.Supersedes, oldSnapshotID)
	newSnapshot.JournalRefs = append(newSnapshot.JournalRefs, journalRef)
	_ = now
	s.snapshots[oldSnapshotID] = oldSnapshot.Clone()
	s.snapshots[newSnapshotID] = newSnapshot.Clone()
	return SnapshotSupersessionResult{Superseded: oldSnapshot.Clone(), Superseding: newSnapshot.Clone()}, nil
}

func (s *Service) ExpireSnapshot(snapshotID string, now time.Time, journalRef string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.snapshots[snapshotID]
	if !ok {
		return Snapshot{}, ErrSnapshotNotFound
	}
	if snapshot.Status == StatusExpired || snapshot.Status == StatusSuperseded {
		return Snapshot{}, ErrInvalidStateTransition
	}
	snapshot.Status = StatusExpired
	snapshot.ExpiredAt = &now
	snapshot.JournalRefs = append(snapshot.JournalRefs, journalRef)
	s.snapshots[snapshotID] = snapshot.Clone()
	return snapshot.Clone(), nil
}

func (s *Service) DiffSnapshots(leftSnapshotID, rightSnapshotID, diffID string, createdAt time.Time, metadata map[string]any) (SnapshotDiff, error) {
	left, ok := s.GetSnapshot(leftSnapshotID)
	if !ok {
		return SnapshotDiff{}, ErrSnapshotNotFound
	}
	right, ok := s.GetSnapshot(rightSnapshotID)
	if !ok {
		return SnapshotDiff{}, ErrSnapshotNotFound
	}
	return DiffSnapshots(diffID, left, right, createdAt, metadata)
}

func (s *Service) CreateRestoreSeed(snapshotID, seedID string, createdAt time.Time, metadata map[string]any, journalRef string) (RestoreSeed, Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.snapshots[snapshotID]
	if !ok {
		return RestoreSeed{}, Snapshot{}, ErrSnapshotNotFound
	}
	if snapshot.Status == StatusExpired || snapshot.Status == StatusSuperseded {
		return RestoreSeed{}, Snapshot{}, ErrInvalidStateTransition
	}
	seed, err := NewRestoreSeed(seedID, snapshot, createdAt, metadata)
	if err != nil {
		return RestoreSeed{}, Snapshot{}, err
	}
	seed.JournalRefs = append(seed.JournalRefs, journalRef)
	snapshot.Status = StatusRestoreSeedCreated
	snapshot.JournalRefs = append(snapshot.JournalRefs, journalRef)
	s.restoreSeeds[seed.RestoreSeedID] = seed.Clone()
	s.snapshots[snapshotID] = snapshot.Clone()
	return seed.Clone(), snapshot.Clone(), nil
}

func (s *Service) GetRestoreSeed(seedID string) (RestoreSeed, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seed, ok := s.restoreSeeds[seedID]
	if !ok {
		return RestoreSeed{}, false
	}
	return seed.Clone(), true
}

func (s *Service) StoreRestoreSeed(seed RestoreSeed) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restoreSeeds[seed.RestoreSeedID] = seed.Clone()
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
