package runtime

import (
	"context"
	"sort"
	"sync"
	"time"
)

type Service struct {
	registry *Registry
	mu       sync.RWMutex
	results  map[string]RuntimeGenerateResult
}

func NewService() *Service {
	return &Service{
		registry: NewRegistry(),
		results:  make(map[string]RuntimeGenerateResult),
	}
}

func (s *Service) RegisterDriver(driver RuntimeDriver) (RuntimeDriverManifest, error) {
	return s.registry.RegisterDriver(driver)
}

func (s *Service) GetDriverManifest(driverID string) (RuntimeDriverManifest, bool) {
	return s.registry.GetManifest(driverID)
}

func (s *Service) ListDrivers() []RuntimeDriverManifest {
	return s.registry.ListManifests()
}

func (s *Service) Capabilities(ctx context.Context, driverID, modelID string) (RuntimeCapabilityManifest, error) {
	return s.registry.Capabilities(ctx, driverID, modelID)
}

func (s *Service) Health(ctx context.Context, driverID string) (RuntimeHealth, error) {
	return s.registry.Health(ctx, driverID)
}

func (s *Service) Generate(ctx context.Context, request RuntimeGenerateRequest) (RuntimeGenerateResult, error) {
	request = NormalizeGenerateRequest(request)
	if err := ValidateGenerateRequest(request); err != nil {
		return RuntimeGenerateResult{}, err
	}
	driver, ok := s.registry.GetDriver(request.DriverID)
	if !ok {
		return RuntimeGenerateResult{}, ErrDriverNotFound
	}
	result, err := driver.Generate(ctx, request)
	if err != nil {
		return RuntimeGenerateResult{}, err
	}
	result = NormalizeGenerateResult(result, request)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[result.ResultID] = result.Clone()
	return result.Clone(), nil
}

func (s *Service) StoreResult(result RuntimeGenerateResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[result.ResultID] = result.Clone()
}

func (s *Service) GetResult(resultID string) (RuntimeGenerateResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result, ok := s.results[resultID]
	if !ok {
		return RuntimeGenerateResult{}, false
	}
	return result.Clone(), true
}

func (s *Service) ListResults(workspaceID string) []RuntimeGenerateResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]RuntimeGenerateResult, 0)
	for _, result := range s.results {
		if workspaceID != "" && result.WorkspaceID != workspaceID {
			continue
		}
		out = append(out, result.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ResultID < out[j].ResultID })
	return out
}

func NormalizeGenerateResult(result RuntimeGenerateResult, request RuntimeGenerateRequest) RuntimeGenerateResult {
	result.RequestID = firstNonEmpty(result.RequestID, request.RequestID)
	result.DriverID = firstNonEmpty(result.DriverID, request.DriverID)
	result.WorkspaceID = firstNonEmpty(result.WorkspaceID, request.WorkspaceID)
	result.CaseID = firstNonEmpty(result.CaseID, request.CaseID)
	result.BundleID = firstNonEmpty(result.BundleID, request.BundleID)
	result.KVLookupID = firstNonEmpty(result.KVLookupID, request.KVLookupID)
	result.KVCacheID = firstNonEmpty(result.KVCacheID, request.KVCacheID)
	result.ModelID = firstNonEmpty(result.ModelID, request.ModelID)
	if result.ResultID == "" {
		result.ResultID = "runtime-result-" + SHA256Text(StableJSON(request))[:12]
	}
	if result.FinishReason == "" {
		result.FinishReason = FinishStop
	}
	if result.CreatedAt.IsZero() {
		result.CreatedAt = request.CreatedAt
	}
	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now().UTC()
	}
	result.RuntimeMetadata = CloneMap(result.RuntimeMetadata)
	result.OutputJSON = CloneMap(result.OutputJSON)
	result.Warnings = NormalizeRefs(result.Warnings)
	result.ProvenanceRefs = NormalizeRefs(append(result.ProvenanceRefs, request.RequestID, request.BundleID, request.KVLookupID, request.KVCacheID))
	result.AuthorityLevel = RuntimeAuthorityProposalOnly
	result.IsCanonicalTruth = false
	result.IsAdmittedEvidence = false
	result.IsModelDriverProposal = true
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
