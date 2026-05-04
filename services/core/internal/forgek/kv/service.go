package kv

import (
	"sort"
	"sync"
	"time"
)

const (
	MissReasonNoCandidate         = "no_candidate_manifest"
	MissReasonIdentityGatesFailed = "identity_gates_failed"
	MissReasonManifestUnavailable = "manifest_unavailable"
)

type Service struct {
	mu        sync.RWMutex
	manifests map[string]KVCacheManifest
	misses    []KVLookupResult
}

func NewService() *Service {
	return &Service{
		manifests: make(map[string]KVCacheManifest),
		misses:    make([]KVLookupResult, 0),
	}
}

func (s *Service) RegisterManifest(input ManifestInput) (KVCacheManifest, error) {
	manifest, err := NewManifest(input)
	if err != nil {
		return KVCacheManifest{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manifests[manifest.CacheID] = manifest.Clone()
	return manifest.Clone(), nil
}

func (s *Service) StoreManifest(manifest KVCacheManifest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manifests[manifest.CacheID] = manifest.Clone()
}

func (s *Service) GetManifest(cacheID string) (KVCacheManifest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	manifest, ok := s.manifests[cacheID]
	if !ok {
		return KVCacheManifest{}, false
	}
	return manifest.Clone(), true
}

func (s *Service) ListManifests(filter ManifestListFilter) []KVCacheManifest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]KVCacheManifest, 0)
	for _, manifest := range s.manifests {
		if !manifestMatchesFilter(manifest, filter) {
			continue
		}
		out = append(out, manifest.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CacheID < out[j].CacheID })
	return out
}

func (s *Service) ValidateIdentity(cacheID, resultID string, request KVLookupRequest, createdAt time.Time) (ValidationResult, error) {
	if err := ValidateLookupRequest(request); err != nil {
		return ValidationResult{}, err
	}
	manifest, ok := s.GetManifest(cacheID)
	if !ok {
		return ValidationResult{}, ErrManifestNotFound
	}
	if manifest.WorkspaceID != request.WorkspaceID {
		return ValidationResult{}, ErrWorkspaceMismatch
	}
	return ValidateIdentity(resultID, manifest, request, createdAt), nil
}

func (s *Service) Lookup(request KVLookupRequest, resultID string, createdAt time.Time) (KVLookupResult, error) {
	request = NormalizeLookupRequest(request)
	if err := ValidateLookupRequest(request); err != nil {
		return KVLookupResult{}, err
	}
	candidates := s.ListManifests(ManifestListFilter{
		WorkspaceID: request.WorkspaceID,
		CaseID:      request.CaseID,
		BundleID:    request.BundleID,
		BlockID:     request.BlockID,
		CacheMode:   request.CacheMode,
	})
	if len(candidates) == 0 {
		return KVLookupResult{
			Hit:        false,
			MissReason: MissReasonNoCandidate,
			CreatedAt:  createdAt,
		}, nil
	}
	var firstFailure ValidationResult
	for i, manifest := range candidates {
		validation := ValidateIdentity(resultID, manifest, request, createdAt)
		if validation.Passed {
			cloned := manifest.Clone()
			return KVLookupResult{
				Hit:              true,
				CacheID:          manifest.CacheID,
				ValidationResult: validation,
				Manifest:         &cloned,
				CreatedAt:        createdAt,
			}, nil
		}
		if i == 0 {
			firstFailure = validation
		}
	}
	reason := MissReasonIdentityGatesFailed
	if len(firstFailure.FailedGates) == 1 && firstFailure.FailedGates[0] == GateManifestAvailable {
		reason = MissReasonManifestUnavailable
	}
	return KVLookupResult{
		Hit:              false,
		ValidationResult: firstFailure,
		MissReason:       reason,
		FailedGates:      NormalizeRefs(firstFailure.FailedGates),
		CreatedAt:        createdAt,
	}, nil
}

func (s *Service) RecordHit(cacheID string, usedAt time.Time, journalRef string) (KVCacheManifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	manifest, ok := s.manifests[cacheID]
	if !ok {
		return KVCacheManifest{}, ErrManifestNotFound
	}
	if !HitEligibleStatus(manifest.Status) {
		return KVCacheManifest{}, ErrInvalidStateTransition
	}
	manifest.Status = StatusHitRecorded
	manifest.ReuseCount++
	manifest.LastUsedAt = &usedAt
	manifest.JournalRefs = appendUnique(manifest.JournalRefs, journalRef)
	s.manifests[cacheID] = manifest.Clone()
	return manifest.Clone(), nil
}

func (s *Service) RecordMiss(result KVLookupResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result.FailedGates = NormalizeRefs(result.FailedGates)
	s.misses = append(s.misses, result)
}

func (s *Service) Invalidate(cacheID, reason string, invalidatedAt time.Time, journalRef string) (KVCacheManifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	manifest, ok := s.manifests[cacheID]
	if !ok {
		return KVCacheManifest{}, ErrManifestNotFound
	}
	manifest, err := InvalidateManifest(manifest, reason, invalidatedAt, journalRef)
	if err != nil {
		return KVCacheManifest{}, err
	}
	s.manifests[cacheID] = manifest.Clone()
	return manifest.Clone(), nil
}

func (s *Service) Evict(cacheID, reason string, evictedAt time.Time, journalRef string) (KVCacheManifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	manifest, ok := s.manifests[cacheID]
	if !ok {
		return KVCacheManifest{}, ErrManifestNotFound
	}
	manifest = EvictManifest(manifest, reason, evictedAt, journalRef)
	s.manifests[cacheID] = manifest.Clone()
	return manifest.Clone(), nil
}

func (s *Service) Promote(cacheID string, journalRef string) (KVCacheManifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	manifest, ok := s.manifests[cacheID]
	if !ok {
		return KVCacheManifest{}, ErrManifestNotFound
	}
	manifest.MemoryTier = PromoteTier(manifest.MemoryTier)
	manifest.JournalRefs = appendUnique(manifest.JournalRefs, journalRef)
	s.manifests[cacheID] = manifest.Clone()
	return manifest.Clone(), nil
}

func (s *Service) Demote(cacheID string, journalRef string) (KVCacheManifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	manifest, ok := s.manifests[cacheID]
	if !ok {
		return KVCacheManifest{}, ErrManifestNotFound
	}
	manifest.MemoryTier = DemoteTier(manifest.MemoryTier)
	manifest.JournalRefs = appendUnique(manifest.JournalRefs, journalRef)
	s.manifests[cacheID] = manifest.Clone()
	return manifest.Clone(), nil
}

func ValidateLookupRequest(request KVLookupRequest) error {
	request = NormalizeLookupRequest(request)
	if request.WorkspaceID == "" || request.BundleID == "" {
		return ErrInvalidLookupRequest
	}
	if !ValidCacheMode(request.CacheMode) {
		return ErrInvalidCacheMode
	}
	if request.ModelID == "" || request.ModelRevision == "" ||
		request.TokenizerID == "" || request.TokenizerRevision == "" ||
		request.ChatTemplateHash == "" || request.PromptLayoutHash == "" ||
		request.PolicySchemaHash == "" || request.SyscallSchemaHash == "" ||
		request.TokenInputHash == "" || request.RuntimeBackend == "" ||
		request.RuntimeVersion == "" || request.AttentionBackend == "" ||
		request.RopeConfigHash == "" || request.KVPrecision == "" ||
		request.CacheSalt == "" {
		return ErrInvalidLookupRequest
	}
	return nil
}

func manifestMatchesFilter(manifest KVCacheManifest, filter ManifestListFilter) bool {
	if filter.WorkspaceID != "" && manifest.WorkspaceID != filter.WorkspaceID {
		return false
	}
	if filter.CaseID != "" && manifest.CaseID != filter.CaseID {
		return false
	}
	if filter.BundleID != "" && manifest.BundleID != filter.BundleID {
		return false
	}
	if filter.BlockID != "" && manifest.BlockID != filter.BlockID {
		return false
	}
	if filter.CacheMode != "" && manifest.CacheMode != filter.CacheMode {
		return false
	}
	if filter.Status != "" && manifest.Status != filter.Status {
		return false
	}
	if filter.MemoryTier != "" && manifest.MemoryTier != filter.MemoryTier {
		return false
	}
	return true
}
