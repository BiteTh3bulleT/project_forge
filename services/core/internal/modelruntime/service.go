package modelruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type ModelRuntimeAuditRecord struct {
	Operation     string
	RequestID     string
	ModelID       string
	Backend       ModelBackendKind
	WorkspaceID   string
	Actor         string
	Source        string
	CorrelationID string
	TraceID       string
	TimeoutMs     int
	MaxTokens     int
	QueueWaitMs   int64
	QueueDepth    int
	RunningCount  int
	DurationMs    int64
	OutputBytes   int
	Outcome       string
	Error         string
	PromptTokens  int
	OutputTokens  int
	Metadata      map[string]any
}

type AuditRecorder interface {
	RecordModelRuntime(ctx context.Context, record ModelRuntimeAuditRecord) (string, error)
}

type GenerateRequestValidator func(ctx context.Context, req GenerateRequest) error

type GPUTelemetrySnapshot struct {
	Enabled                 bool             `json:"enabled"`
	Available               bool             `json:"available"`
	Healthy                 bool             `json:"healthy"`
	State                   string           `json:"state"`
	Detail                  string           `json:"detail,omitempty"`
	MemoryPressure          float64          `json:"memoryPressure,omitempty"`
	MemoryPressureThreshold float64          `json:"memoryPressureThreshold,omitempty"`
	BackgroundAdmissionOK   bool             `json:"backgroundAdmissionOk"`
	Devices                 []map[string]any `json:"devices,omitempty"`
	Warnings                []string         `json:"warnings,omitempty"`
}

type GPUTelemetryFunc func(ctx context.Context) (GPUTelemetrySnapshot, error)

type ServiceOptions struct {
	Backends                                    []ModelBackend
	Models                                      []ModelManifest
	Registry                                    *ModelRegistry
	DefaultModelID                              string
	DefaultTimeout                              time.Duration
	MaxPromptTokens                             int
	MaxOutputTokens                             int
	MaxOutputBytes                              int
	MaxLoadedModels                             int
	LoadTimeout                                 time.Duration
	UnloadTimeout                               time.Duration
	AutoLoad                                    bool
	Audit                                       AuditRecorder
	Clock                                       func() time.Time
	MaxQueueDepth                               int
	MaxConcurrentRequests                       int
	CompletedHistoryLimit                       int
	RequestValidator                            GenerateRequestValidator
	ChatMaxAttempts                             int
	ChatModelCooldown                           time.Duration
	ChatRetryBackoff                            time.Duration
	ChatProviderCooldown                        time.Duration
	ChatCheckpointLimit                         int
	GPUEnabled                                  bool
	GPURequiredForInteractiveInference          bool
	GPUVRAMHeadroomFraction                     float64
	GPUBkgJobsEnabled                           bool
	GPUBkgIdleThreshold                         time.Duration
	GPUMaxBackgroundJobs                        int
	DegradeOnUnavailableGPU                     bool
	SchedulingInteractivePriorityOverBackground bool
	DreamModeAllowGPUClassify                   bool
	GPUTelemetry                                GPUTelemetryFunc
}

type ModelInfo struct {
	Manifest ModelManifest `json:"manifest"`
	Status   ModelStatus   `json:"status"`
	Loaded   *LoadedModel  `json:"loaded,omitempty"`
}

type ModelInspection struct {
	ModelInfo
	BackendInspect BackendInspectResult `json:"backendInspect"`
}

type RequestExecutionState string

const (
	RequestStateQueued    RequestExecutionState = "queued"
	RequestStateRunning   RequestExecutionState = "running"
	RequestStateCompleted RequestExecutionState = "completed"
	RequestStateRejected  RequestExecutionState = "rejected"
	RequestStateCanceled  RequestExecutionState = "canceled"
)

type RequestExecutionRecord struct {
	RequestID           string                `json:"requestId"`
	ModelID             string                `json:"modelId"`
	Backend             ModelBackendKind      `json:"backend"`
	WorkspaceID         string                `json:"workspaceId,omitempty"`
	Actor               string                `json:"actor,omitempty"`
	Source              string                `json:"source,omitempty"`
	CorrelationID       string                `json:"correlationId,omitempty"`
	TraceID             string                `json:"traceId,omitempty"`
	WorkloadClass       GPUWorkloadClass      `json:"workloadClass,omitempty"`
	State               RequestExecutionState `json:"state"`
	Outcome             string                `json:"outcome,omitempty"`
	Error               string                `json:"error,omitempty"`
	EnqueuedAt          time.Time             `json:"enqueuedAt"`
	StartedAt           time.Time             `json:"startedAt,omitempty"`
	FinishedAt          time.Time             `json:"finishedAt,omitempty"`
	QueueWaitMs         int64                 `json:"queueWaitMs,omitempty"`
	DurationMs          int64                 `json:"durationMs,omitempty"`
	QueueDepthAtEnqueue int                   `json:"queueDepthAtEnqueue,omitempty"`
	RunningAtAdmission  int                   `json:"runningAtAdmission,omitempty"`
	PromptTokens        int                   `json:"promptTokens,omitempty"`
	CompletionTokens    int                   `json:"completionTokens,omitempty"`
	OutputBytes         int                   `json:"outputBytes,omitempty"`
}

type SchedulerSnapshot struct {
	MaxQueueDepth         int                      `json:"maxQueueDepth"`
	MaxConcurrentRequests int                      `json:"maxConcurrentRequests"`
	InteractiveQueued     int                      `json:"interactiveQueued"`
	BackgroundQueued      int                      `json:"backgroundQueued"`
	InteractiveRunning    int                      `json:"interactiveRunning"`
	BackgroundRunning     int                      `json:"backgroundRunning"`
	CooldownJobs          int                      `json:"cooldownJobs"`
	Queued                []RequestExecutionRecord `json:"queued"`
	Running               []RequestExecutionRecord `json:"running"`
	Completed             []RequestExecutionRecord `json:"completed"`
}

type RuntimeHealthState string

const (
	RuntimeHealthAvailable   RuntimeHealthState = "available"
	RuntimeHealthDegraded    RuntimeHealthState = "degraded"
	RuntimeHealthUnavailable RuntimeHealthState = "unavailable"
	RuntimeHealthCooldown    RuntimeHealthState = "cooldown"
	RuntimeHealthOverloaded  RuntimeHealthState = "overloaded"
)

type RuntimeHealth struct {
	Healthy         bool                               `json:"healthy"`
	State           RuntimeHealthState                 `json:"state"`
	DegradedReasons []string                           `json:"degradedReasons,omitempty"`
	PolicyWarnings  []string                           `json:"policyWarnings,omitempty"`
	RuntimeEnabled  bool                               `json:"runtimeEnabled"`
	GPUAware        bool                               `json:"gpuAware"`
	GPUTelemetry    *GPUTelemetrySnapshot              `json:"gpuTelemetry,omitempty"`
	Backends        map[ModelBackendKind]BackendHealth `json:"backends"`
	Loaded          map[ModelBackendKind]string        `json:"loaded"`
	Scheduler       SchedulerSnapshot                  `json:"scheduler"`
}

type Service struct {
	mu sync.RWMutex

	backends           map[ModelBackendKind]ModelBackend
	registry           *ModelRegistry
	models             map[string]ModelManifest
	status             map[string]ModelStatus
	loaded             map[string]LoadedModel
	loadedBy           map[ModelBackendKind]string
	backendSupervision map[ModelBackendKind]BackendSupervisionSnapshot

	defaultModelID string

	defaultTimeout   time.Duration
	maxPromptTokens  int
	maxOutputTokens  int
	maxOutputBytes   int
	maxLoadedModels  int
	loadTimeout      time.Duration
	unloadTimeout    time.Duration
	autoLoad         bool
	audit            AuditRecorder
	clock            func() time.Time
	requestValidator GenerateRequestValidator

	schedulerMu           sync.Mutex
	schedulerCond         *sync.Cond
	nextRequestSeq        int64
	queued                []*RequestExecutionRecord
	running               map[string]*RequestExecutionRecord
	completed             []RequestExecutionRecord
	maxQueueDepth         int
	maxConcurrentRequests int
	completedHistoryLimit int

	chatMu                                      sync.Mutex
	nextChatExecutionSeq                        int64
	chatMaxAttempts                             int
	chatModelCooldown                           time.Duration
	chatRetryBackoff                            time.Duration
	chatProviderCooldown                        time.Duration
	chatCheckpointLimit                         int
	chatCheckpoints                             map[string]ChatExecutionCheckpoint
	chatCheckpointOrder                         []string
	chatProviderCooldowns                       map[ModelBackendKind]time.Time
	chatModelCooldowns                          map[string]time.Time
	gpuEnabled                                  bool
	gpuRequiredForInteractiveInference          bool
	gpuVRAMHeadroomFraction                     float64
	gpuBackgroundJobsEnabled                    bool
	gpuBackgroundIdleThreshold                  time.Duration
	gpuMaxBackgroundJobs                        int
	gpuDegradeOnUnavailable                     bool
	schedulingInteractivePriorityOverBackground bool
	backgroundCoolDownUntil                     time.Time
	lastInteractiveCompleteAt                   time.Time
	underCooldown                               bool
	dreamModeAllowGPUClassify                   bool
	gpuTelemetry                                GPUTelemetryFunc
}

func NewService(opts ServiceOptions) (*Service, error) {
	backendMap := map[ModelBackendKind]ModelBackend{}
	for _, backend := range opts.Backends {
		if backend == nil {
			continue
		}
		backendMap[backend.Kind()] = backend
	}
	if len(backendMap) == 0 {
		return nil, fmt.Errorf("at least one model backend is required")
	}

	modelMap := make(map[string]ModelManifest, len(opts.Models))
	statusMap := make(map[string]ModelStatus, len(opts.Models))
	for _, model := range opts.Models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			return nil, fmt.Errorf("model manifest id required")
		}
		model.ID = id
		modelMap[id] = model
		statusMap[id] = StatusAvailable
	}
	if opts.Registry != nil {
		for _, registered := range opts.Registry.List() {
			modelMap[registered.Manifest.ID] = registered.Manifest
			statusMap[registered.Manifest.ID] = registered.Status
		}
	}

	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}

	maxQueueDepth := opts.MaxQueueDepth
	if maxQueueDepth <= 0 {
		maxQueueDepth = 64
	}
	maxConcurrent := opts.MaxConcurrentRequests
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	completedLimit := opts.CompletedHistoryLimit
	if completedLimit <= 0 {
		completedLimit = 128
	}
	chatMaxAttempts := positiveOrDefault(opts.ChatMaxAttempts, 3)
	chatCheckpointLimit := positiveOrDefault(opts.ChatCheckpointLimit, 128)
	chatRetryBackoff := opts.ChatRetryBackoff
	if chatRetryBackoff < 0 {
		chatRetryBackoff = 0
	}
	chatProviderCooldown := opts.ChatProviderCooldown
	if chatProviderCooldown < 0 {
		chatProviderCooldown = 0
	}
	chatModelCooldown := opts.ChatModelCooldown
	if chatModelCooldown < 0 {
		chatModelCooldown = 0
	}
	if chatModelCooldown == 0 {
		chatModelCooldown = chatProviderCooldown
	}
	gpuVRAMHeadroomFraction := opts.GPUVRAMHeadroomFraction
	if gpuVRAMHeadroomFraction < 0 {
		gpuVRAMHeadroomFraction = 0
	} else if gpuVRAMHeadroomFraction > 1 {
		gpuVRAMHeadroomFraction = 1
	}
	gpuMaxBackgroundJobs := opts.GPUMaxBackgroundJobs
	if gpuMaxBackgroundJobs < 0 {
		gpuMaxBackgroundJobs = 0
	}
	gpuBackgroundIdleThreshold := opts.GPUBkgIdleThreshold
	if gpuBackgroundIdleThreshold < 0 {
		gpuBackgroundIdleThreshold = 0
	}

	svc := &Service{
		backends:                           backendMap,
		registry:                           opts.Registry,
		models:                             modelMap,
		status:                             statusMap,
		loaded:                             map[string]LoadedModel{},
		loadedBy:                           map[ModelBackendKind]string{},
		backendSupervision:                 map[ModelBackendKind]BackendSupervisionSnapshot{},
		defaultModelID:                     strings.TrimSpace(opts.DefaultModelID),
		defaultTimeout:                     defaultIfZero(opts.DefaultTimeout, 30*time.Second),
		maxPromptTokens:                    opts.MaxPromptTokens,
		maxOutputTokens:                    opts.MaxOutputTokens,
		maxOutputBytes:                     opts.MaxOutputBytes,
		maxLoadedModels:                    positiveOrDefault(opts.MaxLoadedModels, 1),
		loadTimeout:                        opts.LoadTimeout,
		unloadTimeout:                      opts.UnloadTimeout,
		autoLoad:                           opts.AutoLoad,
		audit:                              opts.Audit,
		clock:                              clock,
		requestValidator:                   opts.RequestValidator,
		queued:                             []*RequestExecutionRecord{},
		running:                            map[string]*RequestExecutionRecord{},
		completed:                          []RequestExecutionRecord{},
		maxQueueDepth:                      maxQueueDepth,
		maxConcurrentRequests:              maxConcurrent,
		completedHistoryLimit:              completedLimit,
		chatMaxAttempts:                    chatMaxAttempts,
		chatRetryBackoff:                   chatRetryBackoff,
		chatProviderCooldown:               chatProviderCooldown,
		chatModelCooldown:                  chatModelCooldown,
		chatCheckpointLimit:                chatCheckpointLimit,
		gpuEnabled:                         opts.GPUEnabled,
		gpuRequiredForInteractiveInference: opts.GPURequiredForInteractiveInference,
		gpuVRAMHeadroomFraction:            gpuVRAMHeadroomFraction,
		gpuBackgroundJobsEnabled:           opts.GPUBkgJobsEnabled,
		gpuBackgroundIdleThreshold:         gpuBackgroundIdleThreshold,
		gpuMaxBackgroundJobs:               gpuMaxBackgroundJobs,
		gpuDegradeOnUnavailable:            opts.DegradeOnUnavailableGPU,
		schedulingInteractivePriorityOverBackground: opts.SchedulingInteractivePriorityOverBackground,
		dreamModeAllowGPUClassify:                   opts.DreamModeAllowGPUClassify,
		gpuTelemetry:                                opts.GPUTelemetry,
		chatCheckpoints:                             map[string]ChatExecutionCheckpoint{},
		chatCheckpointOrder:                         []string{},
		chatProviderCooldowns:                       map[ModelBackendKind]time.Time{},
		chatModelCooldowns:                          map[string]time.Time{},
	}
	svc.schedulerCond = sync.NewCond(&svc.schedulerMu)
	return svc, nil
}

func (s *Service) List(_ context.Context) ([]ModelInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.models))
	for id := range s.models {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]ModelInfo, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.inspectLocked(id))
	}
	return out, nil
}

func (s *Service) Inspect(ctx context.Context, modelID string) (ModelInspection, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ModelInspection{}, fmt.Errorf("modelID is required")
	}

	s.mu.RLock()
	manifest, ok := s.models[modelID]
	if !ok {
		s.mu.RUnlock()
		return ModelInspection{}, ErrModelNotFound
	}
	info := s.inspectLocked(modelID)
	backend, backendOK := s.backends[manifest.Backend]
	s.mu.RUnlock()
	if !backendOK {
		return ModelInspection{}, fmt.Errorf("%w: %s", ErrBackendUnavailable, manifest.Backend)
	}

	inspect, err := backend.Inspect(ctx, modelID)
	if err != nil {
		return ModelInspection{}, err
	}
	return ModelInspection{ModelInfo: info, BackendInspect: inspect}, nil
}

func (s *Service) Load(ctx context.Context, modelID string) (LoadedModel, error) {
	return s.load(ctx, modelID, false)
}

func (s *Service) load(ctx context.Context, modelID string, allowWhenRunning bool) (LoadedModel, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return LoadedModel{}, fmt.Errorf("modelID is required")
	}

	s.mu.RLock()
	manifest, ok := s.models[modelID]
	if !ok {
		s.mu.RUnlock()
		return LoadedModel{}, ErrModelNotFound
	}
	backend, backendOK := s.backends[manifest.Backend]
	status := s.status[modelID]
	if loaded, loadedOK := s.loaded[modelID]; loadedOK && status == StatusLoaded {
		s.mu.RUnlock()
		return loaded, nil
	}
	if status == StatusLoading || status == StatusUnloading {
		s.mu.RUnlock()
		return LoadedModel{}, ErrModelLifecycleBusy
	}
	if status == StatusDisabled || status == StatusArchived {
		s.mu.RUnlock()
		return LoadedModel{}, ErrModelUnavailable
	}
	previousLoaded := s.loadedBy[manifest.Backend]
	s.mu.RUnlock()

	if !backendOK {
		s.setStatus(modelID, StatusError)
		return LoadedModel{}, fmt.Errorf("%w: %s", ErrBackendUnavailable, manifest.Backend)
	}
	if previousLoaded == "" && s.loadedModelCount() >= s.maxLoadedModels {
		return LoadedModel{}, ErrLoadedModelLimit
	}
	if !allowWhenRunning && s.hasRunningForBackend(manifest.Backend) {
		return LoadedModel{}, ErrModelLifecycleBusy
	}

	s.setStatus(modelID, StatusLoading)
	loadCtx, cancel := withOptionalTimeout(ctx, s.loadTimeout)
	defer cancel()

	if previousLoaded != "" && previousLoaded != modelID {
		s.setStatus(previousLoaded, StatusUnloading)
		if err := backend.Unload(loadCtx, previousLoaded); err != nil {
			s.setStatus(previousLoaded, StatusError)
			s.setStatus(modelID, StatusError)
			return LoadedModel{}, err
		}
		s.mu.Lock()
		delete(s.loaded, previousLoaded)
		s.status[previousLoaded] = s.registeredStatus(previousLoaded)
		s.loadedBy[manifest.Backend] = ""
		s.mu.Unlock()
	}

	loaded, err := backend.Load(loadCtx, manifest)
	if err != nil {
		s.setStatus(modelID, StatusError)
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation: "load",
			ModelID:   modelID,
			Backend:   manifest.Backend,
			Outcome:   "error",
			Error:     err.Error(),
		})
		return LoadedModel{}, err
	}
	if loaded.Status == "" {
		loaded.Status = StatusLoaded
	}
	if loaded.LoadedAt.IsZero() {
		loaded.LoadedAt = s.clock()
	}
	loaded.ModelID = modelID
	loaded.Backend = manifest.Backend

	s.mu.Lock()
	s.loaded[modelID] = loaded
	s.loadedBy[manifest.Backend] = modelID
	s.status[modelID] = StatusLoaded
	s.mu.Unlock()

	s.recordAudit(ctx, ModelRuntimeAuditRecord{
		Operation: "load",
		ModelID:   modelID,
		Backend:   manifest.Backend,
		Outcome:   "ok",
	})
	return loaded, nil
}

func (s *Service) Unload(ctx context.Context, modelID string) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return fmt.Errorf("modelID is required")
	}

	s.mu.RLock()
	manifest, ok := s.models[modelID]
	if !ok {
		s.mu.RUnlock()
		return ErrModelNotFound
	}
	backend, backendOK := s.backends[manifest.Backend]
	status := s.status[modelID]
	_, loaded := s.loaded[modelID]
	s.mu.RUnlock()
	if !backendOK {
		return fmt.Errorf("%w: %s", ErrBackendUnavailable, manifest.Backend)
	}
	if status == StatusLoading || status == StatusUnloading {
		return ErrModelLifecycleBusy
	}
	if !loaded {
		return ErrModelNotLoaded
	}
	if s.hasRunningForBackend(manifest.Backend) {
		return ErrModelLifecycleBusy
	}

	s.setStatus(modelID, StatusUnloading)
	unloadCtx, cancel := withOptionalTimeout(ctx, s.unloadTimeout)
	defer cancel()
	if err := backend.Unload(unloadCtx, modelID); err != nil {
		s.setStatus(modelID, StatusError)
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation: "unload",
			ModelID:   modelID,
			Backend:   manifest.Backend,
			Outcome:   "error",
			Error:     err.Error(),
		})
		return err
	}

	s.mu.Lock()
	delete(s.loaded, modelID)
	if current := s.loadedBy[manifest.Backend]; current == modelID {
		s.loadedBy[manifest.Backend] = ""
	}
	s.status[modelID] = s.registeredStatus(modelID)
	s.mu.Unlock()

	s.recordAudit(ctx, ModelRuntimeAuditRecord{
		Operation: "unload",
		ModelID:   modelID,
		Backend:   manifest.Backend,
		Outcome:   "ok",
	})
	return nil
}

func (s *Service) Generate(ctx context.Context, req GenerateRequest) (result GenerateResult, err error) {
	req.ModelID = strings.TrimSpace(req.ModelID)
	req.Backend = ParseModelBackendKind(string(req.Backend))
	req.Actor = strings.TrimSpace(req.Actor)
	req.Source = strings.TrimSpace(req.Source)
	req.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	req.WorkloadClass = ParseGPUWorkloadClass(string(req.WorkloadClass))
	if req.WorkloadClass == GPUWorkloadUnknown {
		req.WorkloadClass = s.defaultWorkloadClassFromRequest(req)
	}
	if req.ModelID == "" {
		if req.ModelID, err = s.resolveModelID(req); err != nil {
			s.recordAudit(ctx, ModelRuntimeAuditRecord{
				Operation:     "generate",
				WorkspaceID:   req.WorkspaceID,
				Actor:         req.Actor,
				Source:        req.Source,
				CorrelationID: req.CorrelationID,
				TraceID:       req.TraceID,
				TimeoutMs:     req.TimeoutMs,
				MaxTokens:     req.MaxTokens,
				Outcome:       "error",
				Error:         err.Error(),
				Metadata:      req.Metadata,
			})
			return GenerateResult{}, err
		}
	}

	started := s.clock()
	if err = ValidateGenerateRequest(req); err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation:     "generate",
			ModelID:       req.ModelID,
			WorkspaceID:   req.WorkspaceID,
			Actor:         req.Actor,
			Source:        req.Source,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			TimeoutMs:     req.TimeoutMs,
			MaxTokens:     req.MaxTokens,
			Outcome:       "error",
			Error:         err.Error(),
			Metadata:      req.Metadata,
		})
		return GenerateResult{}, err
	}
	if err = s.validateGPUWorkloadPolicy(ctx, req.WorkloadClass); err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation:     "generate",
			ModelID:       req.ModelID,
			WorkspaceID:   req.WorkspaceID,
			Actor:         req.Actor,
			Source:        req.Source,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			TimeoutMs:     req.TimeoutMs,
			MaxTokens:     req.MaxTokens,
			Outcome:       "error",
			Error:         err.Error(),
			Metadata:      req.Metadata,
		})
		return GenerateResult{}, err
	}
	if s.maxPromptTokens > 0 {
		promptTokens := approxGeneratePromptTokens(req)
		if promptTokens > s.maxPromptTokens {
			err = fmt.Errorf("prompt token estimate %d exceeds max %d", promptTokens, s.maxPromptTokens)
			s.recordAudit(ctx, ModelRuntimeAuditRecord{
				Operation:     "generate",
				ModelID:       req.ModelID,
				WorkspaceID:   req.WorkspaceID,
				Actor:         req.Actor,
				Source:        req.Source,
				CorrelationID: req.CorrelationID,
				TraceID:       req.TraceID,
				TimeoutMs:     req.TimeoutMs,
				MaxTokens:     req.MaxTokens,
				Outcome:       "error",
				Error:         err.Error(),
				PromptTokens:  promptTokens,
				Metadata:      req.Metadata,
			})
			return GenerateResult{}, err
		}
	}

	if err = s.validateRequestContext(ctx, req); err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation:     "generate",
			ModelID:       req.ModelID,
			WorkspaceID:   req.WorkspaceID,
			Actor:         req.Actor,
			Source:        req.Source,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			TimeoutMs:     req.TimeoutMs,
			MaxTokens:     req.MaxTokens,
			Outcome:       "error",
			Error:         err.Error(),
			Metadata:      req.Metadata,
		})
		return GenerateResult{}, err
	}

	s.mu.RLock()
	manifest, ok := s.models[req.ModelID]
	if !ok {
		s.mu.RUnlock()
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation:     "generate",
			ModelID:       req.ModelID,
			WorkspaceID:   req.WorkspaceID,
			Actor:         req.Actor,
			Source:        req.Source,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			TimeoutMs:     req.TimeoutMs,
			MaxTokens:     req.MaxTokens,
			DurationMs:    s.clock().Sub(started).Milliseconds(),
			Outcome:       "error",
			Error:         ErrModelNotFound.Error(),
			Metadata:      req.Metadata,
		})
		return GenerateResult{}, ErrModelNotFound
	}
	backend, backendOK := s.backends[manifest.Backend]
	status := s.status[req.ModelID]
	s.mu.RUnlock()
	if req.Backend != "" && manifest.Backend != req.Backend {
		err = fmt.Errorf("%w: model %s uses backend %s not %s", ErrUnsupportedBackendOverride, req.ModelID, manifest.Backend, req.Backend)
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation:     "generate",
			ModelID:       req.ModelID,
			Backend:       manifest.Backend,
			WorkspaceID:   req.WorkspaceID,
			Actor:         req.Actor,
			Source:        req.Source,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			TimeoutMs:     req.TimeoutMs,
			MaxTokens:     req.MaxTokens,
			DurationMs:    s.clock().Sub(started).Milliseconds(),
			Outcome:       "error",
			Error:         err.Error(),
			Metadata:      req.Metadata,
		})
		return GenerateResult{}, err
	}
	if status == StatusDisabled || status == StatusArchived {
		err = ErrModelUnavailable
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation:     "generate",
			ModelID:       req.ModelID,
			Backend:       manifest.Backend,
			WorkspaceID:   req.WorkspaceID,
			Actor:         req.Actor,
			Source:        req.Source,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			TimeoutMs:     req.TimeoutMs,
			MaxTokens:     req.MaxTokens,
			DurationMs:    s.clock().Sub(started).Milliseconds(),
			Outcome:       "error",
			Error:         err.Error(),
			Metadata:      req.Metadata,
		})
		return GenerateResult{}, err
	}

	if !backendOK {
		err = fmt.Errorf("%w: %s", ErrBackendUnavailable, manifest.Backend)
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation:     "generate",
			ModelID:       req.ModelID,
			Backend:       manifest.Backend,
			WorkspaceID:   req.WorkspaceID,
			Actor:         req.Actor,
			Source:        req.Source,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			TimeoutMs:     req.TimeoutMs,
			MaxTokens:     req.MaxTokens,
			DurationMs:    s.clock().Sub(started).Milliseconds(),
			Outcome:       "error",
			Error:         err.Error(),
			Metadata:      req.Metadata,
		})
		return GenerateResult{}, err
	}

	if err = ensureModelCapability(manifest, req); err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation:     "generate",
			ModelID:       req.ModelID,
			Backend:       manifest.Backend,
			WorkspaceID:   req.WorkspaceID,
			Actor:         req.Actor,
			Source:        req.Source,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			TimeoutMs:     req.TimeoutMs,
			MaxTokens:     req.MaxTokens,
			DurationMs:    s.clock().Sub(started).Milliseconds(),
			Outcome:       "error",
			Error:         err.Error(),
			Metadata:      req.Metadata,
		})
		return GenerateResult{}, err
	}

	s.mu.RLock()
	_, loaded := s.loaded[req.ModelID]
	s.mu.RUnlock()
	if !loaded {
		if !s.autoLoad && manifest.Backend != BackendOllamaCompat {
			err = ErrModelNotLoaded
			s.recordAudit(ctx, ModelRuntimeAuditRecord{
				Operation:     "generate",
				ModelID:       req.ModelID,
				Backend:       manifest.Backend,
				WorkspaceID:   req.WorkspaceID,
				Actor:         req.Actor,
				Source:        req.Source,
				CorrelationID: req.CorrelationID,
				TraceID:       req.TraceID,
				TimeoutMs:     req.TimeoutMs,
				MaxTokens:     req.MaxTokens,
				DurationMs:    s.clock().Sub(started).Milliseconds(),
				Outcome:       "error",
				Error:         err.Error(),
				Metadata:      req.Metadata,
			})
			return GenerateResult{}, err
		}
		if _, err = s.load(ctx, req.ModelID, true); err != nil {
			s.recordAudit(ctx, ModelRuntimeAuditRecord{
				Operation:     "generate",
				ModelID:       req.ModelID,
				Backend:       manifest.Backend,
				WorkspaceID:   req.WorkspaceID,
				Actor:         req.Actor,
				Source:        req.Source,
				CorrelationID: req.CorrelationID,
				TraceID:       req.TraceID,
				TimeoutMs:     req.TimeoutMs,
				MaxTokens:     req.MaxTokens,
				DurationMs:    s.clock().Sub(started).Milliseconds(),
				Outcome:       "error",
				Error:         err.Error(),
				Metadata:      req.Metadata,
			})
			return GenerateResult{}, err
		}
	}

	reqRecord, admissionErr := s.acquireExecutionSlot(ctx, req, manifest.Backend)
	if admissionErr != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation:     "generate",
			RequestID:     reqRecord.RequestID,
			ModelID:       req.ModelID,
			Backend:       manifest.Backend,
			WorkspaceID:   req.WorkspaceID,
			Actor:         req.Actor,
			Source:        req.Source,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			TimeoutMs:     req.TimeoutMs,
			MaxTokens:     req.MaxTokens,
			QueueWaitMs:   reqRecord.QueueWaitMs,
			QueueDepth:    reqRecord.QueueDepthAtEnqueue,
			DurationMs:    s.clock().Sub(started).Milliseconds(),
			Outcome:       "error",
			Error:         admissionErr.Error(),
			Metadata:      req.Metadata,
		})
		return GenerateResult{}, admissionErr
	}
	defer s.finishExecutionSlot(reqRecord, &result, &err)

	if req.MaxTokens <= 0 && manifest.DefaultRuntime.MaxTokens > 0 {
		req.MaxTokens = manifest.DefaultRuntime.MaxTokens
	}
	if req.TimeoutMs <= 0 && manifest.DefaultRuntime.TimeoutMs > 0 {
		req.TimeoutMs = manifest.DefaultRuntime.TimeoutMs
	}
	if req.MaxTokens <= 0 && s.maxOutputTokens > 0 {
		req.MaxTokens = s.maxOutputTokens
	}
	if req.MaxTokens > 0 && s.maxOutputTokens > 0 && req.MaxTokens > s.maxOutputTokens {
		req.MaxTokens = s.maxOutputTokens
	}

	s.mu.RLock()
	_, loaded = s.loaded[req.ModelID]
	s.mu.RUnlock()
	if !loaded {
		err = ErrModelNotLoaded
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation:     "generate",
			RequestID:     reqRecord.RequestID,
			ModelID:       req.ModelID,
			Backend:       manifest.Backend,
			WorkspaceID:   req.WorkspaceID,
			Actor:         req.Actor,
			Source:        req.Source,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			TimeoutMs:     req.TimeoutMs,
			MaxTokens:     req.MaxTokens,
			QueueWaitMs:   reqRecord.QueueWaitMs,
			QueueDepth:    reqRecord.QueueDepthAtEnqueue,
			RunningCount:  reqRecord.RunningAtAdmission,
			DurationMs:    s.clock().Sub(started).Milliseconds(),
			Outcome:       "error",
			Error:         err.Error(),
			Metadata:      req.Metadata,
		})
		return GenerateResult{}, err
	}

	timeout := EffectiveTimeout(req, s.defaultTimeout)
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if req.Stream {
		streamingBackend, ok := backend.(StreamingModelBackend)
		if !ok {
			err = ErrStreamingUnsupported
			durationMs := s.clock().Sub(started).Milliseconds()
			s.recordAudit(ctx, ModelRuntimeAuditRecord{
				Operation:     "generate",
				RequestID:     reqRecord.RequestID,
				ModelID:       req.ModelID,
				Backend:       manifest.Backend,
				WorkspaceID:   req.WorkspaceID,
				Actor:         req.Actor,
				Source:        req.Source,
				CorrelationID: req.CorrelationID,
				TraceID:       req.TraceID,
				TimeoutMs:     req.TimeoutMs,
				MaxTokens:     req.MaxTokens,
				QueueWaitMs:   reqRecord.QueueWaitMs,
				QueueDepth:    reqRecord.QueueDepthAtEnqueue,
				RunningCount:  reqRecord.RunningAtAdmission,
				DurationMs:    durationMs,
				Outcome:       "error",
				Error:         err.Error(),
				Metadata:      req.Metadata,
			})
			return GenerateResult{}, err
		}
		result, err = streamingBackend.GenerateStream(callCtx, req, req.StreamHandler)
	} else {
		result, err = backend.Generate(callCtx, req)
	}
	durationMs := s.clock().Sub(started).Milliseconds()
	if err != nil {
		s.recordAudit(ctx, ModelRuntimeAuditRecord{
			Operation:     "generate",
			RequestID:     reqRecord.RequestID,
			ModelID:       req.ModelID,
			Backend:       manifest.Backend,
			WorkspaceID:   req.WorkspaceID,
			Actor:         req.Actor,
			Source:        req.Source,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			TimeoutMs:     req.TimeoutMs,
			MaxTokens:     req.MaxTokens,
			QueueWaitMs:   reqRecord.QueueWaitMs,
			QueueDepth:    reqRecord.QueueDepthAtEnqueue,
			RunningCount:  reqRecord.RunningAtAdmission,
			DurationMs:    durationMs,
			Outcome:       "error",
			Error:         err.Error(),
			Metadata:      req.Metadata,
		})
		return GenerateResult{}, err
	}

	result.ModelID = req.ModelID
	result.Backend = manifest.Backend
	if result.DurationMs == 0 {
		result.DurationMs = durationMs
	}
	if result.FinishReason == "" {
		result.FinishReason = "stop"
	}

	if bounded, truncated := BoundTextApproxTokens(result.Content, EffectiveMaxOutputTokens(req, s.maxOutputTokens)); truncated {
		result.Content = bounded
		result.FinishReason = "length"
		result.Warnings = append(result.Warnings, ErrOutputBound.Error())
	}
	if bounded, truncated := BoundTextBytes(result.Content, s.maxOutputBytes); truncated {
		result.Content = bounded
		result.FinishReason = "length"
		result.Warnings = append(result.Warnings, ErrOutputBytesBound.Error())
	}

	outputBytes := len([]byte(result.Content))
	auditID := s.recordAudit(ctx, ModelRuntimeAuditRecord{
		Operation:     "generate",
		RequestID:     reqRecord.RequestID,
		ModelID:       req.ModelID,
		Backend:       manifest.Backend,
		WorkspaceID:   req.WorkspaceID,
		Actor:         req.Actor,
		Source:        req.Source,
		CorrelationID: req.CorrelationID,
		TraceID:       req.TraceID,
		TimeoutMs:     req.TimeoutMs,
		MaxTokens:     req.MaxTokens,
		QueueWaitMs:   reqRecord.QueueWaitMs,
		QueueDepth:    reqRecord.QueueDepthAtEnqueue,
		RunningCount:  reqRecord.RunningAtAdmission,
		DurationMs:    result.DurationMs,
		PromptTokens:  result.PromptTokens,
		OutputTokens:  result.CompletionTokens,
		OutputBytes:   outputBytes,
		Outcome:       "ok",
		Metadata:      req.Metadata,
	})
	result.AuditID = auditID
	result.Proposal = buildProposalEnvelope(req, reqRecord.RequestID, result, auditID, outputBytes)
	return result, nil
}

func buildProposalEnvelope(req GenerateRequest, requestID string, result GenerateResult, auditID string, outputBytes int) *ProposalEnvelope {
	sum := sha256.Sum256([]byte(result.Content))
	totalTokens := result.PromptTokens + result.CompletionTokens
	return &ProposalEnvelope{
		SchemaVersion:          "modelruntime.proposal/v1",
		ProposalID:             firstNonEmptyProposalValue(requestID, result.AuditID, auditID, req.CorrelationID, req.ModelID) + ":proposal",
		ProposalKind:           "model_runtime_output",
		ModelID:                strings.TrimSpace(result.ModelID),
		Backend:                result.Backend,
		WorkspaceID:            strings.TrimSpace(req.WorkspaceID),
		Actor:                  strings.TrimSpace(req.Actor),
		Source:                 strings.TrimSpace(req.Source),
		CorrelationID:          strings.TrimSpace(req.CorrelationID),
		TraceID:                strings.TrimSpace(req.TraceID),
		RequestID:              strings.TrimSpace(requestID),
		AuditID:                firstNonEmptyProposalValue(result.AuditID, auditID),
		OutputHash:             "sha256:" + hex.EncodeToString(sum[:]),
		OutputBytes:            outputBytes,
		FinishReason:           strings.TrimSpace(result.FinishReason),
		PromptTokens:           result.PromptTokens,
		CompletionTokens:       result.CompletionTokens,
		TotalTokens:            totalTokens,
		ProposalOnly:           true,
		CanonicalCommit:        false,
		TruthMutation:          false,
		MemoryMutation:         false,
		EvidenceAdmission:      false,
		GatewayExecution:       false,
		ModelOutputAuthority:   false,
		RequiresKernelCommit:   true,
		RequiresValidation:     true,
		LiveAuthorityMigration: false,
	}
}

func firstNonEmptyProposalValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return "modelruntime-output"
}

func (s *Service) SchedulerSnapshot() SchedulerSnapshot {
	s.schedulerMu.Lock()
	defer s.schedulerMu.Unlock()
	return s.schedulerSnapshotLocked(s.clock().UTC())
}

func (s *Service) LoadedModels() []LoadedModel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.loaded))
	for id := range s.loaded {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]LoadedModel, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.loaded[id])
	}
	return out
}

func (s *Service) setStatus(modelID string, status ModelStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status[modelID] = status
}

func (s *Service) inspectLocked(modelID string) ModelInfo {
	manifest := s.models[modelID]
	status := s.status[modelID]
	if status == "" {
		status = StatusAvailable
	}
	var loaded *LoadedModel
	if loadedState, ok := s.loaded[modelID]; ok {
		clone := loadedState
		loaded = &clone
	}
	return ModelInfo{
		Manifest: manifest,
		Status:   status,
		Loaded:   loaded,
	}
}

func (s *Service) validateRequestContext(ctx context.Context, req GenerateRequest) error {
	if req.Stream && req.StreamHandler == nil {
		return ErrStreamingUnsupported
	}
	if strings.TrimSpace(req.Actor) == "" {
		return ErrActorRequired
	}
	if strings.TrimSpace(req.Source) == "" {
		return ErrSourceRequired
	}
	if s.requestValidator != nil {
		if err := s.requestValidator(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

func ensureModelCapability(manifest ModelManifest, req GenerateRequest) error {
	if len(req.Messages) > 0 {
		if manifestHasCapability(manifest, CapabilityChat) {
			return nil
		}
		return fmt.Errorf("%w: model %s missing capability %s", ErrModelCapabilityUnsupported, manifest.ID, CapabilityChat)
	}
	if manifestHasCapability(manifest, CapabilityCompletion) || manifestHasCapability(manifest, CapabilityChat) {
		return nil
	}
	return fmt.Errorf("%w: model %s missing capability completion/chat", ErrModelCapabilityUnsupported, manifest.ID)
}

func manifestHasCapability(manifest ModelManifest, capability ModelCapability) bool {
	for _, cap := range manifest.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

func (s *Service) acquireExecutionSlot(ctx context.Context, req GenerateRequest, backend ModelBackendKind) (*RequestExecutionRecord, error) {
	s.schedulerMu.Lock()
	defer s.schedulerMu.Unlock()
	s.updateCooldownStateLocked(s.clock().UTC())

	now := s.clock().UTC()
	s.nextRequestSeq++
	requestID := fmt.Sprintf("mr-%08d", s.nextRequestSeq)

	record := &RequestExecutionRecord{
		RequestID:     requestID,
		ModelID:       req.ModelID,
		Backend:       backend,
		WorkspaceID:   req.WorkspaceID,
		Actor:         req.Actor,
		Source:        req.Source,
		CorrelationID: req.CorrelationID,
		TraceID:       req.TraceID,
		WorkloadClass: req.WorkloadClass,
		State:         RequestStateQueued,
		EnqueuedAt:    now,
	}

	if len(s.queued) >= s.maxQueueDepth {
		record.State = RequestStateRejected
		record.Outcome = "rejected"
		record.Error = ErrRequestQueueFull.Error()
		record.FinishedAt = now
		record.QueueDepthAtEnqueue = len(s.queued)
		s.addCompletedLocked(*record)
		return record, ErrRequestQueueFull
	}

	s.enqueueRecordLocked(record)
	record.QueueDepthAtEnqueue = len(s.queued)
	s.schedulerCond.Broadcast()

	waitDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			s.schedulerMu.Lock()
			s.schedulerCond.Broadcast()
			s.schedulerMu.Unlock()
		case <-waitDone:
		}
	}()
	defer close(waitDone)

	for {
		if err := ctx.Err(); err != nil {
			if s.removeQueuedByIDLocked(record.RequestID) {
				record.State = RequestStateCanceled
				record.Outcome = "canceled"
				record.Error = err.Error()
				record.FinishedAt = s.clock().UTC()
				record.QueueWaitMs = record.FinishedAt.Sub(record.EnqueuedAt).Milliseconds()
				s.addCompletedLocked(*record)
				s.schedulerCond.Broadcast()
			}
			return record, err
		}

		if s.canAdmitLocked(record.RequestID) {
			s.removeQueuedByIDLocked(record.RequestID)
			record.State = RequestStateRunning
			record.StartedAt = s.clock().UTC()
			record.QueueWaitMs = record.StartedAt.Sub(record.EnqueuedAt).Milliseconds()
			record.RunningAtAdmission = len(s.running) + 1
			s.running[record.RequestID] = record
			s.schedulerCond.Broadcast()
			return record, nil
		}

		s.schedulerCond.Wait()
	}
}

func (s *Service) finishExecutionSlot(record *RequestExecutionRecord, result *GenerateResult, runErr *error) {
	s.schedulerMu.Lock()
	defer s.schedulerMu.Unlock()
	s.updateCooldownStateLocked(s.clock().UTC())

	delete(s.running, record.RequestID)
	now := s.clock().UTC()
	record.FinishedAt = now
	if !record.StartedAt.IsZero() {
		record.DurationMs = now.Sub(record.StartedAt).Milliseconds()
	}

	if runErr != nil && *runErr != nil {
		record.State = RequestStateCompleted
		record.Outcome = "error"
		record.Error = (*runErr).Error()
	} else {
		record.State = RequestStateCompleted
		record.Outcome = "ok"
	}
	if result != nil {
		record.PromptTokens = result.PromptTokens
		record.CompletionTokens = result.CompletionTokens
		record.OutputBytes = len([]byte(result.Content))
	}
	if record.WorkloadClass.IsInteractive() && !record.StartedAt.IsZero() {
		s.lastInteractiveCompleteAt = now
		if s.gpuBackgroundIdleThreshold > 0 {
			s.underCooldown = true
			s.backgroundCoolDownUntil = now.Add(s.gpuBackgroundIdleThreshold)
		}
	}

	s.addCompletedLocked(*record)
	s.schedulerCond.Broadcast()
}

func (s *Service) schedulerSnapshotLocked(now time.Time) SchedulerSnapshot {
	out := SchedulerSnapshot{
		MaxQueueDepth:         s.maxQueueDepth,
		MaxConcurrentRequests: s.maxConcurrentRequests,
		Queued:                make([]RequestExecutionRecord, 0, len(s.queued)),
		Running:               make([]RequestExecutionRecord, 0, len(s.running)),
		Completed:             make([]RequestExecutionRecord, len(s.completed)),
	}
	for _, q := range s.queued {
		out.Queued = append(out.Queued, *q)
		if q.WorkloadClass.IsInteractive() {
			out.InteractiveQueued++
		}
		if q.WorkloadClass.IsBackground() {
			out.BackgroundQueued++
			if s.isBackgroundBlockedLocked(q, now) {
				out.CooldownJobs++
			}
		}
	}
	runningIDs := make([]string, 0, len(s.running))
	for id := range s.running {
		runningIDs = append(runningIDs, id)
	}
	sort.Strings(runningIDs)
	for _, id := range runningIDs {
		out.Running = append(out.Running, *s.running[id])
		if s.running[id].WorkloadClass.IsInteractive() {
			out.InteractiveRunning++
		}
		if s.running[id].WorkloadClass.IsBackground() {
			out.BackgroundRunning++
		}
	}
	copy(out.Completed, s.completed)
	if out.CooldownJobs > out.BackgroundQueued {
		out.CooldownJobs = out.BackgroundQueued
	}
	return out
}

func (s *Service) canAdmitLocked(requestID string) bool {
	if len(s.queued) == 0 {
		return false
	}
	if s.queued[0].RequestID != requestID {
		return false
	}
	if err := s.validateWorkloadAdmissionLocked(s.queued[0]); err != nil {
		s.queued[0].Error = err.Error()
		return false
	}
	if len(s.running) >= s.maxConcurrentRequests {
		return false
	}
	next := s.queued[0]
	if next.WorkloadClass.IsBackground() {
		if s.schedulingInteractivePriorityOverBackground && s.hasInteractiveRunningLocked() {
			return false
		}
		if runningBackgroundCount := s.backgroundRunningCountLocked(); runningBackgroundCount >= maxInt(1, s.gpuMaxBackgroundJobs) {
			return false
		}
	}

	for _, running := range s.running {
		if running.Backend == next.Backend {
			return false
		}
	}
	if next.WorkloadClass.IsBackground() && s.isBackgroundBlockedLocked(next, s.clock().UTC()) {
		return false
	}
	return true
}

func (s *Service) validateWorkloadAdmissionLocked(req *RequestExecutionRecord) error {
	if req.WorkloadClass.IsBackground() && !s.gpuBackgroundJobsEnabled {
		return ErrBackgroundJobsDisabled
	}
	return nil
}

func (s *Service) updateCooldownStateLocked(now time.Time) {
	if s.backgroundCoolDownUntil.IsZero() {
		s.underCooldown = false
		return
	}
	if !s.underCooldown {
		return
	}
	if now.After(s.backgroundCoolDownUntil) {
		s.underCooldown = false
	}
}

func (s *Service) isBackgroundBlockedLocked(record *RequestExecutionRecord, now time.Time) bool {
	if !record.WorkloadClass.IsBackground() {
		return false
	}
	if !s.gpuBackgroundJobsEnabled {
		return true
	}
	if s.schedulingInteractivePriorityOverBackground && s.hasInteractiveRunningLocked() {
		return true
	}
	if s.schedulingInteractivePriorityOverBackground {
		for _, q := range s.queued {
			if q.RequestID == record.RequestID {
				break
			}
			if q.WorkloadClass.IsInteractive() {
				return true
			}
		}
	}
	if s.underCooldown && now.Before(s.backgroundCoolDownUntil) {
		return true
	}
	return false
}

func (s *Service) hasInteractiveRunningLocked() bool {
	for _, running := range s.running {
		if running.WorkloadClass.IsInteractive() {
			return true
		}
	}
	return false
}

func (s *Service) backgroundRunningCountLocked() int {
	count := 0
	for _, running := range s.running {
		if running.WorkloadClass.IsBackground() {
			count++
		}
	}
	return count
}

func (s *Service) healthNeedsOverloadedSignalLocked() bool {
	return len(s.queued) >= maxInt(1, s.maxQueueDepth) || len(s.running) > s.maxConcurrentRequests
}

func (s *Service) gpuCurrentlyAvailableLocked() bool {
	for _, backend := range s.backends {
		h, err := backend.Health(context.Background())
		if err != nil {
			continue
		}
		if h.Healthy {
			return true
		}
	}
	return false
}

func (s *Service) validateGPUWorkloadPolicy(ctx context.Context, workload GPUWorkloadClass) error {
	if !workload.IsInteractive() && !workload.IsBackground() {
		return nil
	}
	if workload.IsBackground() && !s.gpuBackgroundJobsEnabled {
		return ErrBackgroundJobsDisabled
	}
	if workload.IsInteractive() && s.gpuRequiredForInteractiveInference && !s.gpuEnabled {
		return ErrGPUNotAllowedForInteractive
	}
	if workload.IsBackground() && s.gpuTelemetry != nil {
		telemetry, err := s.gpuTelemetry(ctx)
		if err == nil && telemetry.Enabled && !telemetry.BackgroundAdmissionOK {
			return fmt.Errorf("%w: gpu memory pressure %.3f >= %.3f", ErrBackgroundWorkloadDeferred, telemetry.MemoryPressure, telemetry.MemoryPressureThreshold)
		}
	}
	return nil
}

func (s *Service) defaultWorkloadClassFromRequest(req GenerateRequest) GPUWorkloadClass {
	if s.dreamModeAllowGPUClassify && strings.EqualFold(req.WorkspaceID, "dream") {
		return GPUWorkloadDreamDistillation
	}
	if len(req.Messages) > 0 {
		return GPUWorkloadInteractiveInference
	}
	if strings.EqualFold(strings.TrimSpace(req.ModelID), "embed") {
		return GPUWorkloadInteractiveEmbedding
	}
	if req.WorkloadClass != GPUWorkloadUnknown {
		return req.WorkloadClass
	}
	return GPUWorkloadInteractiveInference
}

func (s *Service) enqueueRecordLocked(record *RequestExecutionRecord) {
	if !s.schedulingInteractivePriorityOverBackground || !record.WorkloadClass.IsInteractive() {
		s.queued = append(s.queued, record)
		return
	}
	insertAt := len(s.queued)
	for i, queued := range s.queued {
		if !queued.WorkloadClass.IsInteractive() {
			insertAt = i
			break
		}
	}
	s.queued = append(s.queued, nil)
	copy(s.queued[insertAt+1:], s.queued[insertAt:])
	s.queued[insertAt] = record
}

func maxInt(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

func (s *Service) removeQueuedByIDLocked(requestID string) bool {
	for i := range s.queued {
		if s.queued[i].RequestID != requestID {
			continue
		}
		s.queued = append(s.queued[:i], s.queued[i+1:]...)
		return true
	}
	return false
}

func (s *Service) addCompletedLocked(record RequestExecutionRecord) {
	s.completed = append(s.completed, record)
	if len(s.completed) <= s.completedHistoryLimit {
		return
	}
	excess := len(s.completed) - s.completedHistoryLimit
	s.completed = append([]RequestExecutionRecord(nil), s.completed[excess:]...)
}

func (s *Service) recordAudit(ctx context.Context, record ModelRuntimeAuditRecord) string {
	if s.audit == nil {
		return ""
	}
	auditID, err := s.audit.RecordModelRuntime(ctx, record)
	if err != nil {
		return ""
	}
	return auditID
}

func (s *Service) hasRunningForBackend(backend ModelBackendKind) bool {
	s.schedulerMu.Lock()
	defer s.schedulerMu.Unlock()
	for _, running := range s.running {
		if running.Backend == backend {
			return true
		}
	}
	return false
}

func withOptionalTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func approxGeneratePromptTokens(req GenerateRequest) int {
	if len(req.Messages) > 0 {
		total := 0
		for _, msg := range req.Messages {
			total += len(strings.Fields(msg.Role))
			total += len(strings.Fields(msg.Content))
			total += len(strings.Fields(msg.Name))
		}
		return total
	}
	return len(strings.Fields(req.Prompt))
}

func positiveOrDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func (s *Service) loadedModelCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.loaded)
}
