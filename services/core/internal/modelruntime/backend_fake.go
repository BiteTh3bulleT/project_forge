package modelruntime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type FakeGenerateFunc func(req GenerateRequest) (GenerateResult, error)

type FakeBackendOptions struct {
	Name            string
	Kind            ModelBackendKind
	Clock           func() time.Time
	Generate        FakeGenerateFunc
	MaxOutputTokens int
	Healthy         bool
	HealthDetail    string
	HealthErr       error
}

type FakeBackend struct {
	name            string
	kind            ModelBackendKind
	clock           func() time.Time
	generate        FakeGenerateFunc
	maxOutputTokens int

	mu         sync.RWMutex
	loaded     map[string]LoadedModel
	loadEvents []string
	healthy    bool
	detail     string
	healthErr  error
}

func NewFakeBackend(opts FakeBackendOptions) *FakeBackend {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = "fake"
	}
	kind := opts.Kind
	if strings.TrimSpace(string(kind)) == "" {
		kind = BackendFake
	}
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}
	healthy := true
	if !opts.Healthy {
		healthy = false
	}
	if opts.HealthDetail == "" {
		if healthy {
			opts.HealthDetail = "fake backend healthy"
		} else {
			opts.HealthDetail = "fake backend unhealthy"
		}
	}
	return &FakeBackend{
		name:            name,
		kind:            kind,
		clock:           clock,
		generate:        opts.Generate,
		maxOutputTokens: opts.MaxOutputTokens,
		loaded:          map[string]LoadedModel{},
		healthy:         healthy,
		detail:          opts.HealthDetail,
		healthErr:       opts.HealthErr,
	}
}

func (b *FakeBackend) Name() string { return b.name }

func (b *FakeBackend) Kind() ModelBackendKind { return b.kind }

func (b *FakeBackend) Supports(_ ModelFormat, _ []ModelCapability) bool { return true }

func (b *FakeBackend) Load(_ context.Context, manifest ModelManifest) (LoadedModel, error) {
	modelID := strings.TrimSpace(manifest.ID)
	if modelID == "" {
		return LoadedModel{}, errors.New("manifest.id is required")
	}
	loaded := LoadedModel{
		ModelID:  modelID,
		Backend:  b.kind,
		Status:   StatusLoaded,
		LoadedAt: b.clock(),
		Metadata: map[string]any{
			"backend": "fake",
		},
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.loaded[modelID] = loaded
	b.loadEvents = append(b.loadEvents, "load:"+modelID)
	return loaded, nil
}

func (b *FakeBackend) Unload(_ context.Context, modelID string) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return errors.New("modelID is required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.loaded[modelID]; !ok {
		return ErrModelNotLoaded
	}
	delete(b.loaded, modelID)
	b.loadEvents = append(b.loadEvents, "unload:"+modelID)
	return nil
}

func (b *FakeBackend) Generate(_ context.Context, req GenerateRequest) (GenerateResult, error) {
	if err := ValidateGenerateRequest(req); err != nil {
		return GenerateResult{}, err
	}
	b.mu.RLock()
	_, ok := b.loaded[req.ModelID]
	b.mu.RUnlock()
	if !ok {
		return GenerateResult{}, ErrModelNotLoaded
	}

	if b.generate != nil {
		out, err := b.generate(req)
		if err != nil {
			return GenerateResult{}, err
		}
		out.Backend = b.kind
		if out.ModelID == "" {
			out.ModelID = req.ModelID
		}
		bounded, truncated := BoundTextApproxTokens(out.Content, EffectiveMaxOutputTokens(req, b.maxOutputTokens))
		if truncated {
			out.Content = bounded
			out.FinishReason = "length"
			out.Warnings = append(out.Warnings, ErrOutputBound.Error())
		}
		return out, nil
	}

	content := req.Prompt
	if len(req.Messages) > 0 {
		parts := make([]string, 0, len(req.Messages))
		for _, msg := range req.Messages {
			parts = append(parts, fmt.Sprintf("%s:%s", strings.TrimSpace(msg.Role), strings.TrimSpace(msg.Content)))
		}
		content = strings.Join(parts, " | ")
	}
	text := fmt.Sprintf("fake[%s]: %s", req.ModelID, strings.TrimSpace(content))
	text, truncated := BoundTextApproxTokens(text, EffectiveMaxOutputTokens(req, b.maxOutputTokens))
	result := GenerateResult{
		Content:          text,
		FinishReason:     "stop",
		PromptTokens:     tokenCountApprox(req),
		CompletionTokens: len(strings.Fields(text)),
		Backend:          b.kind,
		ModelID:          req.ModelID,
	}
	if truncated {
		result.FinishReason = "length"
		result.Warnings = append(result.Warnings, ErrOutputBound.Error())
	}
	return result, nil
}

func (b *FakeBackend) Health(_ context.Context) (BackendHealth, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	health := BackendHealth{
		Name:    b.name,
		Kind:    b.kind,
		Healthy: b.healthy,
		Detail:  b.detail,
		Meta: map[string]any{
			"loaded": len(b.loaded),
		},
	}
	return health, b.healthErr
}

func (b *FakeBackend) Inspect(_ context.Context, modelID string) (BackendInspectResult, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return BackendInspectResult{}, errors.New("modelID is required")
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	loaded, ok := b.loaded[modelID]
	meta := map[string]any{"loaded": ok}
	if ok {
		meta["loadedAt"] = loaded.LoadedAt
	}
	return BackendInspectResult{
		ModelID: modelID,
		Backend: b.kind,
		Found:   ok,
		Meta:    meta,
	}, nil
}

func (b *FakeBackend) LoadedModelIDs() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	ids := make([]string, 0, len(b.loaded))
	for id := range b.loaded {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (b *FakeBackend) Events() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, len(b.loadEvents))
	copy(out, b.loadEvents)
	return out
}

func tokenCountApprox(req GenerateRequest) int {
	if len(req.Messages) > 0 {
		total := 0
		for _, msg := range req.Messages {
			total += len(strings.Fields(msg.Content))
		}
		return total
	}
	return len(strings.Fields(req.Prompt))
}
