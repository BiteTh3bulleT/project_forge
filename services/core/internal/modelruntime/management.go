package modelruntime

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type ManagementRequestMeta struct {
	WorkspaceID   string
	Actor         string
	Source        string
	CorrelationID string
	TraceID       string
	Metadata      map[string]any
}

type ImportRequest struct {
	Path string
	ImportModelOptions
	ManagementRequestMeta
}

type ImportResult struct {
	Model       ModelInfo
	Duplicate   bool
	ManagedPath string
	SourcePath  string
	Warnings    []string
}

type RemoveRegistrationResult struct {
	ModelID       string
	RemovedPath   string
	CorrelationID string
}

type DeleteFilesResult struct {
	ModelID       string
	DeletedPath   string
	Deleted       bool
	CorrelationID string
}

type CompatibilityReport struct {
	ModelID            string           `json:"modelId"`
	Backend            ModelBackendKind `json:"backend"`
	Status             ModelStatus      `json:"status"`
	Loaded             bool             `json:"loaded"`
	BackendConfigured  bool             `json:"backendConfigured"`
	BackendHealthy     bool             `json:"backendHealthy"`
	BackendDetail      string           `json:"backendDetail,omitempty"`
	SupportedByBackend bool             `json:"supportedByBackend"`
	CanGenerate        bool             `json:"canGenerate"`
	Preferred          bool             `json:"preferred,omitempty"`
	Warnings           []string         `json:"warnings,omitempty"`
	Metadata           map[string]any   `json:"metadata,omitempty"`
}

type RuntimeBackendStatus struct {
	Kind        ModelBackendKind           `json:"kind"`
	Name        string                     `json:"name"`
	Healthy     bool                       `json:"healthy"`
	Detail      string                     `json:"detail,omitempty"`
	LoadedModel string                     `json:"loadedModel,omitempty"`
	Meta        map[string]any             `json:"meta,omitempty"`
	Supervision BackendSupervisionSnapshot `json:"supervision,omitempty"`
}

type RuntimeUsageSummary struct {
	Registered     int                                 `json:"registered"`
	Imported       int                                 `json:"imported"`
	Verified       int                                 `json:"verified"`
	Available      int                                 `json:"available"`
	Disabled       int                                 `json:"disabled"`
	Archived       int                                 `json:"archived"`
	Loaded         int                                 `json:"loaded"`
	QueueDepth     int                                 `json:"queueDepth"`
	Running        int                                 `json:"running"`
	Completed      int                                 `json:"completed"`
	ResourceLimits RuntimeResourceLimits               `json:"resourceLimits"`
	Backends       map[ModelBackendKind]map[string]any `json:"backends,omitempty"`
}

func (s *Service) ImportModel(ctx context.Context, req ImportRequest) (ImportResult, error) {
	if s.registry == nil {
		return ImportResult{}, ErrManagementUnavailable
	}
	registered, duplicate, err := s.registry.Import(ctx, req.Path, req.ImportModelOptions)
	if err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "import", ModelID: registered.Manifest.ID, WorkspaceID: strings.TrimSpace(req.WorkspaceID), Actor: strings.TrimSpace(req.Actor), Source: strings.TrimSpace(req.Source), CorrelationID: strings.TrimSpace(req.CorrelationID), TraceID: strings.TrimSpace(req.TraceID), Outcome: "error", Error: err.Error(), Metadata: req.ManagementRequestMeta.Metadata})
		return ImportResult{}, err
	}
	s.refreshFromRegistry()
	info, err := s.getModelInfo(registered.Manifest.ID)
	if err != nil {
		return ImportResult{}, err
	}
	managedPath := registered.ManifestPath
	if registered.ModelFilePath != "" {
		managedPath = registered.ModelFilePath
	}
	s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "import", ModelID: registered.Manifest.ID, Backend: registered.Manifest.Backend, WorkspaceID: strings.TrimSpace(req.WorkspaceID), Actor: strings.TrimSpace(req.Actor), Source: strings.TrimSpace(req.Source), CorrelationID: strings.TrimSpace(req.CorrelationID), TraceID: strings.TrimSpace(req.TraceID), Outcome: "ok", Metadata: map[string]any{"duplicate": duplicate, "managedPath": managedPath}})
	return ImportResult{Model: info, Duplicate: duplicate, ManagedPath: managedPath, SourcePath: registered.State.SourcePath, Warnings: append([]string(nil), registered.Warnings...)}, nil
}

func (s *Service) ScanModels(ctx context.Context, meta ManagementRequestMeta) ([]ModelInfo, error) {
	if s.registry == nil {
		return nil, ErrManagementUnavailable
	}
	if _, err := s.registry.Reconcile(ctx); err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "scan", WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "error", Error: err.Error(), Metadata: meta.Metadata})
		return nil, err
	}
	s.refreshFromRegistry()
	infos, err := s.List(ctx)
	if err == nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "scan", WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "ok", Metadata: map[string]any{"count": len(infos)}})
	}
	return infos, err
}

func (s *Service) RegisterDiscoveredModels(ctx context.Context, manifests []ModelManifest, meta ManagementRequestMeta) (int, error) {
	if s.registry == nil {
		return 0, ErrManagementUnavailable
	}
	registered := 0
	for _, manifest := range manifests {
		id := strings.TrimSpace(manifest.ID)
		if id == "" {
			continue
		}
		if _, exists := s.registry.Get(id); exists {
			continue
		}
		if err := s.registry.Register(manifest); err != nil {
			s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "discover", ModelID: id, Backend: manifest.Backend, WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "error", Error: err.Error(), Metadata: meta.Metadata})
			return registered, err
		}
		registered++
	}
	if registered > 0 {
		s.refreshFromRegistry()
		s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "discover", WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "ok", Metadata: map[string]any{"count": registered}})
	}
	return registered, nil
}

func (s *Service) VerifyModel(ctx context.Context, modelID string, meta ManagementRequestMeta) (ModelInfo, error) {
	if s.registry == nil {
		return ModelInfo{}, ErrManagementUnavailable
	}
	registered, err := s.registry.Verify(ctx, modelID)
	if err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "verify", ModelID: strings.TrimSpace(modelID), WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "error", Error: err.Error(), Metadata: meta.Metadata})
		return ModelInfo{}, err
	}
	s.refreshFromRegistry()
	info, err := s.getModelInfo(registered.Manifest.ID)
	if err != nil {
		return ModelInfo{}, err
	}
	s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "verify", ModelID: registered.Manifest.ID, Backend: registered.Manifest.Backend, WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "ok", Metadata: meta.Metadata})
	return info, nil
}

func (s *Service) EnableModel(ctx context.Context, modelID string, meta ManagementRequestMeta) (ModelInfo, error) {
	return s.setModelDisabled(ctx, modelID, false, meta)
}

func (s *Service) DisableModel(ctx context.Context, modelID string, meta ManagementRequestMeta) (ModelInfo, error) {
	return s.setModelDisabled(ctx, modelID, true, meta)
}

func (s *Service) setModelDisabled(ctx context.Context, modelID string, disabled bool, meta ManagementRequestMeta) (ModelInfo, error) {
	if s.registry == nil {
		return ModelInfo{}, ErrManagementUnavailable
	}
	if disabled {
		if err := s.Unload(ctx, modelID); err != nil && err != ErrModelNotLoaded && err != ErrModelNotFound {
			return ModelInfo{}, err
		}
	}
	registered, err := s.registry.SetDisabled(ctx, modelID, disabled)
	if err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: ternary(disabled, "disable", "enable"), ModelID: strings.TrimSpace(modelID), WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "error", Error: err.Error(), Metadata: meta.Metadata})
		return ModelInfo{}, err
	}
	s.refreshFromRegistry()
	info, err := s.getModelInfo(registered.Manifest.ID)
	if err != nil {
		return ModelInfo{}, err
	}
	s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: ternary(disabled, "disable", "enable"), ModelID: registered.Manifest.ID, Backend: registered.Manifest.Backend, WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "ok", Metadata: meta.Metadata})
	return info, nil
}

func (s *Service) ArchiveModel(ctx context.Context, modelID string, meta ManagementRequestMeta) (ModelInfo, error) {
	if s.registry == nil {
		return ModelInfo{}, ErrManagementUnavailable
	}
	if err := s.Unload(ctx, modelID); err != nil && err != ErrModelNotLoaded && err != ErrModelNotFound {
		return ModelInfo{}, err
	}
	registered, err := s.registry.Archive(ctx, modelID)
	if err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "archive", ModelID: strings.TrimSpace(modelID), WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "error", Error: err.Error(), Metadata: meta.Metadata})
		return ModelInfo{}, err
	}
	s.refreshFromRegistry()
	info, err := s.getModelInfo(registered.Manifest.ID)
	if err != nil {
		return ModelInfo{}, err
	}
	s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "archive", ModelID: registered.Manifest.ID, Backend: registered.Manifest.Backend, WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "ok", Metadata: meta.Metadata})
	return info, nil
}

func (s *Service) RemoveModelRegistration(ctx context.Context, modelID string, meta ManagementRequestMeta) (RemoveRegistrationResult, error) {
	if s.registry == nil {
		return RemoveRegistrationResult{}, ErrManagementUnavailable
	}
	if err := s.Unload(ctx, modelID); err != nil && err != ErrModelNotLoaded && err != ErrModelNotFound {
		return RemoveRegistrationResult{}, err
	}
	removedPath, err := s.registry.RemoveRegistration(ctx, modelID)
	if err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "remove_registration", ModelID: strings.TrimSpace(modelID), WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "error", Error: err.Error(), Metadata: meta.Metadata})
		return RemoveRegistrationResult{}, err
	}
	s.refreshFromRegistry()
	s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "remove_registration", ModelID: strings.TrimSpace(modelID), WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "ok", Metadata: map[string]any{"removedPath": removedPath}})
	return RemoveRegistrationResult{ModelID: strings.TrimSpace(modelID), RemovedPath: removedPath, CorrelationID: strings.TrimSpace(meta.CorrelationID)}, nil
}

func (s *Service) DeleteModelFiles(ctx context.Context, modelID string, meta ManagementRequestMeta) (DeleteFilesResult, error) {
	if s.registry == nil {
		return DeleteFilesResult{}, ErrManagementUnavailable
	}
	if err := s.Unload(ctx, modelID); err != nil && err != ErrModelNotLoaded && err != ErrModelNotFound {
		return DeleteFilesResult{}, err
	}
	deletedPath, err := s.registry.DeleteFiles(ctx, modelID)
	if err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "delete_file", ModelID: strings.TrimSpace(modelID), WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "error", Error: err.Error(), Metadata: meta.Metadata})
		return DeleteFilesResult{}, err
	}
	s.refreshFromRegistry()
	s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "delete_file", ModelID: strings.TrimSpace(modelID), WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "ok", Metadata: map[string]any{"deletedPath": deletedPath}})
	return DeleteFilesResult{ModelID: strings.TrimSpace(modelID), DeletedPath: deletedPath, Deleted: true, CorrelationID: strings.TrimSpace(meta.CorrelationID)}, nil
}

func (s *Service) PreferModel(ctx context.Context, modelID string, meta ManagementRequestMeta) (ModelInfo, error) {
	if s.registry == nil {
		return ModelInfo{}, ErrManagementUnavailable
	}
	registered, err := s.registry.SetPreferred(ctx, modelID, true)
	if err != nil {
		return ModelInfo{}, err
	}
	s.defaultModelID = registered.Manifest.ID
	s.refreshFromRegistry()
	info, err := s.getModelInfo(registered.Manifest.ID)
	if err != nil {
		return ModelInfo{}, err
	}
	s.recordAudit(ctx, ModelRuntimeAuditRecord{Operation: "prefer", ModelID: registered.Manifest.ID, Backend: registered.Manifest.Backend, WorkspaceID: strings.TrimSpace(meta.WorkspaceID), Actor: strings.TrimSpace(meta.Actor), Source: strings.TrimSpace(meta.Source), CorrelationID: strings.TrimSpace(meta.CorrelationID), TraceID: strings.TrimSpace(meta.TraceID), Outcome: "ok", Metadata: meta.Metadata})
	return info, nil
}

func (s *Service) Compatibility(ctx context.Context, modelID string) (CompatibilityReport, error) {
	modelID = strings.TrimSpace(modelID)
	s.mu.RLock()
	manifest, ok := s.models[modelID]
	status := s.status[modelID]
	loaded := false
	if _, loadedState := s.loaded[modelID]; loadedState {
		loaded = true
	}
	backend, backendOK := s.backends[manifest.Backend]
	s.mu.RUnlock()
	if !ok {
		return CompatibilityReport{}, ErrModelNotFound
	}
	report := CompatibilityReport{ModelID: modelID, Backend: manifest.Backend, Status: status, Loaded: loaded, BackendConfigured: backendOK, SupportedByBackend: backendOK, CanGenerate: backendOK && status != StatusArchived && status != StatusDisabled, Metadata: cloneStateMetadata(manifest.Metadata)}
	if s.registry != nil {
		if registered, found := s.registry.Get(modelID); found {
			report.Preferred = registered.State.Preferred
		}
	}
	if backendOK {
		report.SupportedByBackend = backend.Supports(manifest.Format, manifest.Capabilities)
		health, err := backend.Health(ctx)
		if err == nil {
			report.BackendHealthy = health.Healthy
			report.BackendDetail = health.Detail
		} else {
			report.BackendDetail = err.Error()
		}
	}
	if status == StatusArchived {
		report.Warnings = append(report.Warnings, ErrModelArchived.Error())
		report.CanGenerate = false
	}
	if status == StatusDisabled {
		report.Warnings = append(report.Warnings, ErrModelUnavailable.Error())
		report.CanGenerate = false
	}
	if !report.SupportedByBackend {
		report.Warnings = append(report.Warnings, fmt.Sprintf("backend %s does not support format %s", manifest.Backend, manifest.Format))
		report.CanGenerate = false
	}
	return report, nil
}

func (s *Service) Backends(ctx context.Context) ([]RuntimeBackendStatus, error) {
	health, err := s.Health(ctx)
	statuses := make([]RuntimeBackendStatus, 0, len(health.Backends))
	kinds := make([]string, 0, len(health.Backends))
	for kind := range health.Backends {
		kinds = append(kinds, string(kind))
	}
	sort.Strings(kinds)
	for _, rawKind := range kinds {
		kind := ModelBackendKind(rawKind)
		backendHealth := health.Backends[kind]
		statuses = append(statuses, RuntimeBackendStatus{Kind: kind, Name: backendHealth.Name, Healthy: backendHealth.Healthy, Detail: backendHealth.Detail, LoadedModel: health.Loaded[kind], Meta: cloneStateMetadata(backendHealth.Meta), Supervision: backendHealth.Supervision})
	}
	return statuses, err
}

func (s *Service) Usage(ctx context.Context) (RuntimeUsageSummary, error) {
	infos, err := s.List(ctx)
	if err != nil {
		return RuntimeUsageSummary{}, err
	}
	snapshot := s.SchedulerSnapshot()
	usage := RuntimeUsageSummary{Registered: len(infos), QueueDepth: len(snapshot.Queued), Running: len(snapshot.Running), Completed: len(snapshot.Completed), Loaded: len(s.LoadedModels()), ResourceLimits: s.resourceLimitsSnapshot(), Backends: map[ModelBackendKind]map[string]any{}}
	for _, info := range infos {
		switch info.Status {
		case StatusImported:
			usage.Imported++
		case StatusVerified:
			usage.Verified++
		case StatusDisabled:
			usage.Disabled++
		case StatusArchived:
			usage.Archived++
		case StatusLoaded:
			usage.Loaded++
		default:
			usage.Available++
		}
	}
	backends, healthErr := s.Backends(ctx)
	for _, backend := range backends {
		usage.Backends[backend.Kind] = map[string]any{"healthy": backend.Healthy, "loadedModel": backend.LoadedModel, "name": backend.Name}
	}
	if err == nil {
		err = healthErr
	}
	return usage, err
}

func (s *Service) refreshFromRegistry() {
	if s.registry == nil {
		return
	}
	registered := s.registry.List()
	modelMap := make(map[string]ModelManifest, len(registered))
	statusMap := make(map[string]ModelStatus, len(registered))
	for _, rec := range registered {
		modelMap[rec.Manifest.ID] = rec.Manifest
		statusMap[rec.Manifest.ID] = rec.Status
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models = modelMap
	for id, loaded := range s.loaded {
		if _, ok := modelMap[id]; ok {
			statusMap[id] = StatusLoaded
			s.loaded[id] = loaded
		}
	}
	s.status = statusMap
	for kind, modelID := range s.loadedBy {
		if _, ok := modelMap[modelID]; !ok {
			delete(s.loadedBy, kind)
		}
	}
}

func (s *Service) getModelInfo(modelID string) (ModelInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.models[modelID]; !ok {
		return ModelInfo{}, ErrModelNotFound
	}
	return s.inspectLocked(modelID), nil
}

func (s *Service) resolveModelID(req GenerateRequest) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.defaultModelID != "" {
		if manifest, ok := s.models[s.defaultModelID]; ok && manifestCanServeRequest(manifest, req) && s.status[s.defaultModelID] != StatusDisabled && s.status[s.defaultModelID] != StatusArchived {
			if req.Backend == "" || req.Backend == manifest.Backend {
				return s.defaultModelID, nil
			}
		}
	}
	if s.registry != nil {
		if preferredID, ok := s.registry.PreferredModelID(); ok {
			if manifest, exists := s.models[preferredID]; exists && manifestCanServeRequest(manifest, req) && (req.Backend == "" || req.Backend == manifest.Backend) {
				return preferredID, nil
			}
		}
	}
	candidates := make([]string, 0, len(s.models))
	for id, manifest := range s.models {
		if s.status[id] == StatusDisabled || s.status[id] == StatusArchived {
			continue
		}
		if req.Backend != "" && manifest.Backend != req.Backend {
			continue
		}
		if !manifestCanServeRequest(manifest, req) {
			continue
		}
		candidates = append(candidates, id)
	}
	sort.Strings(candidates)
	switch len(candidates) {
	case 0:
		return "", ErrModelNotFound
	case 1:
		return candidates[0], nil
	default:
		return "", fmt.Errorf("%w: %s", ErrModelSelectionAmbiguous, strings.Join(candidates, ", "))
	}
}

func manifestCanServeRequest(manifest ModelManifest, req GenerateRequest) bool {
	if len(req.Messages) > 0 {
		return manifestHasCapability(manifest, CapabilityChat)
	}
	return manifestHasCapability(manifest, CapabilityCompletion) || manifestHasCapability(manifest, CapabilityChat)
}

func (s *Service) registeredStatus(modelID string) ModelStatus {
	if s.registry != nil {
		if registered, ok := s.registry.Get(modelID); ok {
			return registered.Status
		}
	}
	if status, ok := s.status[modelID]; ok && status != StatusLoaded && status != StatusLoading && status != StatusUnloading {
		return status
	}
	return StatusAvailable
}

func ternary[T any](cond bool, whenTrue, whenFalse T) T {
	if cond {
		return whenTrue
	}
	return whenFalse
}
