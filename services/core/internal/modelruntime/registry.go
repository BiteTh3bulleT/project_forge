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

// RegisteredModel is a manifest entry with runtime registry metadata.
type RegisteredModel struct {
	Manifest      ModelManifest `json:"manifest"`
	State         ModelState    `json:"state,omitempty"`
	ModelFilePath string        `json:"modelFilePath"`
	ManifestPath  string        `json:"manifestPath"`
	Status        ModelStatus   `json:"status"`
	Archived      bool          `json:"archived,omitempty"`
	Warnings      []string      `json:"warnings,omitempty"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

// ModelRegistry tracks discovered model manifests and runtime status metadata.
type ModelRegistry struct {
	store  *ModelStore
	mu     sync.RWMutex
	models map[string]RegisteredModel
}

func NewModelRegistry(store *ModelStore) *ModelRegistry {
	return &ModelRegistry{
		store:  store,
		models: map[string]RegisteredModel{},
	}
}

// Scan refreshes the registry from model store manifests.
func (r *ModelRegistry) Scan(ctx context.Context) ([]RegisteredModel, error) {
	if r.store == nil {
		return nil, fmt.Errorf("model store is nil")
	}
	records, err := r.store.Scan(ctx)
	if err != nil {
		if !errors.Is(err, ErrModelHomeMissing) && !errors.Is(err, ErrModelsDirMissing) {
			return nil, err
		}
		records = nil
	}

	now := time.Now().UTC()
	next := make(map[string]RegisteredModel, len(records))

	r.mu.Lock()
	defer r.mu.Unlock()
	preserved := make(map[string]RegisteredModel, len(r.models))
	for id, model := range r.models {
		preserved[id] = model
	}

	for _, record := range records {
		rm := RegisteredModel{
			Manifest:      record.Manifest,
			State:         record.State,
			ModelFilePath: record.ModelFilePath,
			ManifestPath:  record.ManifestPath,
			Status:        stateStatusOrDefault(record.State, ModelStatusAvailable),
			Archived:      record.Archived,
			Warnings:      append([]string(nil), record.Warnings...),
			UpdatedAt:     now,
		}
		if prev, ok := r.models[record.Manifest.ID]; ok {
			rm.Status = prev.Status
			if rm.Status == "" {
				rm.Status = ModelStatusAvailable
			}
		}
		next[record.Manifest.ID] = rm
	}

	for id, model := range preserved {
		if _, ok := next[id]; ok {
			continue
		}
		if !isDiscoveredModel(model) {
			continue
		}
		model.UpdatedAt = now
		next[id] = model
	}
	r.models = next
	return sortedRegisteredModels(next), nil
}

func isDiscoveredModel(model RegisteredModel) bool {
	if strings.TrimSpace(model.ManifestPath) != "" || strings.TrimSpace(model.ModelFilePath) != "" {
		return false
	}
	v := model.Manifest.Metadata
	if v == nil {
		return false
	}
	discovered := v["discovered"]
	switch value := discovered.(type) {
	case bool:
		return value
	case string:
		return strings.TrimSpace(value) == "true"
	case int, int64, float64, float32:
		return true
	default:
		source, ok := model.Manifest.Metadata["source"].(string)
		if !ok {
			return false
		}
		return strings.TrimSpace(source) != ""
	}
}

// List returns all registered models sorted by model id.
func (r *ModelRegistry) List() []RegisteredModel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return sortedRegisteredModels(r.models)
}

// Get returns a single model by id.
func (r *ModelRegistry) Get(modelID string) (RegisteredModel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[strings.TrimSpace(modelID)]
	return m, ok
}

// Inspect returns detailed record for one model id.
func (r *ModelRegistry) Inspect(modelID string) (RegisteredModel, error) {
	id := strings.TrimSpace(modelID)
	if id == "" {
		return RegisteredModel{}, fmt.Errorf("%w: empty id", ErrModelNotFound)
	}
	if model, ok := r.Get(id); ok {
		return model, nil
	}
	return RegisteredModel{}, fmt.Errorf("%w: %s", ErrModelNotFound, id)
}

// Register validates and inserts/updates a manifest record in-memory.
func (r *ModelRegistry) Register(manifest ModelManifest) error {
	if err := r.ValidateManifest(manifest); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.models[manifest.ID]
	if !ok {
		entry = RegisteredModel{
			Manifest:  manifest,
			Status:    ModelStatusAvailable,
			UpdatedAt: time.Now().UTC(),
		}
	} else {
		entry.Manifest = manifest
		entry.UpdatedAt = time.Now().UTC()
		if entry.Status == "" {
			entry.Status = ModelStatusAvailable
		}
	}
	r.models[manifest.ID] = entry
	return nil
}

// UpdateStatus mutates only runtime status for a known model.
func (r *ModelRegistry) UpdateStatus(modelID string, status ModelStatus) error {
	if _, ok := validStatuses[status]; !ok {
		return fmt.Errorf("invalid model status %q", status)
	}

	id := strings.TrimSpace(modelID)
	if id == "" {
		return fmt.Errorf("%w: empty id", ErrModelNotFound)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.models[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrModelNotFound, id)
	}
	entry.Status = status
	entry.UpdatedAt = time.Now().UTC()
	r.models[id] = entry
	return nil
}

func (r *ModelRegistry) ValidateManifest(manifest ModelManifest) error {
	return ValidateManifest(manifest)
}

func (r *ModelRegistry) Reconcile(ctx context.Context) ([]RegisteredModel, error) {
	return r.Scan(ctx)
}

func (r *ModelRegistry) Import(ctx context.Context, path string, opts ImportModelOptions) (RegisteredModel, bool, error) {
	if r.store == nil {
		return RegisteredModel{}, false, fmt.Errorf("model store is nil")
	}
	result, err := r.store.Import(ctx, path, opts)
	if err != nil {
		return RegisteredModel{}, false, err
	}
	rm := registeredModelFromStored(result.Model)
	r.mu.Lock()
	r.models[rm.Manifest.ID] = rm
	r.mu.Unlock()
	return rm, result.Duplicate, nil
}

func (r *ModelRegistry) Verify(ctx context.Context, modelID string) (RegisteredModel, error) {
	if r.store == nil {
		return RegisteredModel{}, fmt.Errorf("model store is nil")
	}
	rec, err := r.store.Verify(ctx, modelID)
	if err != nil {
		return RegisteredModel{}, err
	}
	rm := registeredModelFromStored(rec)
	r.mu.Lock()
	r.models[rm.Manifest.ID] = rm
	r.mu.Unlock()
	return rm, nil
}

func (r *ModelRegistry) SetDisabled(ctx context.Context, modelID string, disabled bool) (RegisteredModel, error) {
	if r.store == nil {
		return RegisteredModel{}, fmt.Errorf("model store is nil")
	}
	rec, err := r.store.SetDisabled(ctx, modelID, disabled)
	if err != nil {
		return RegisteredModel{}, err
	}
	rm := registeredModelFromStored(rec)
	r.mu.Lock()
	r.models[rm.Manifest.ID] = rm
	r.mu.Unlock()
	return rm, nil
}

func (r *ModelRegistry) SetPreferred(ctx context.Context, modelID string, preferred bool) (RegisteredModel, error) {
	if r.store == nil {
		return RegisteredModel{}, fmt.Errorf("model store is nil")
	}
	rec, err := r.store.SetPreferred(ctx, modelID, preferred)
	if err != nil {
		return RegisteredModel{}, err
	}
	updated, err := r.Scan(ctx)
	if err != nil {
		return RegisteredModel{}, err
	}
	for _, model := range updated {
		if model.Manifest.ID == rec.Manifest.ID {
			return model, nil
		}
	}
	return registeredModelFromStored(rec), nil
}

func (r *ModelRegistry) Archive(ctx context.Context, modelID string) (RegisteredModel, error) {
	if r.store == nil {
		return RegisteredModel{}, fmt.Errorf("model store is nil")
	}
	rec, err := r.store.Archive(ctx, modelID)
	if err != nil {
		return RegisteredModel{}, err
	}
	rm := registeredModelFromStored(rec)
	r.mu.Lock()
	r.models[rm.Manifest.ID] = rm
	r.mu.Unlock()
	return rm, nil
}

func (r *ModelRegistry) RemoveRegistration(ctx context.Context, modelID string) (string, error) {
	if r.store == nil {
		return "", fmt.Errorf("model store is nil")
	}
	removedPath, err := r.store.RemoveRegistration(ctx, modelID)
	if err != nil {
		return "", err
	}
	r.mu.Lock()
	delete(r.models, strings.TrimSpace(modelID))
	r.mu.Unlock()
	return removedPath, nil
}

func (r *ModelRegistry) DeleteFiles(ctx context.Context, modelID string) (string, error) {
	if r.store == nil {
		return "", fmt.Errorf("model store is nil")
	}
	deletedPath, err := r.store.DeleteFiles(ctx, modelID)
	if err != nil {
		return "", err
	}
	r.mu.Lock()
	delete(r.models, strings.TrimSpace(modelID))
	r.mu.Unlock()
	return deletedPath, nil
}

func (r *ModelRegistry) PreferredModelID() (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.models))
	for id := range r.models {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		model := r.models[id]
		if model.State.Preferred && model.Status != StatusArchived && model.Status != StatusDisabled {
			return id, true
		}
	}
	return "", false
}

func registeredModelFromStored(rec StoredModel) RegisteredModel {
	rec.State.Normalize()
	return RegisteredModel{
		Manifest:      rec.Manifest,
		State:         rec.State,
		ModelFilePath: rec.ModelFilePath,
		ManifestPath:  rec.ManifestPath,
		Status:        stateStatusOrDefault(rec.State, StatusAvailable),
		Archived:      rec.Archived,
		Warnings:      append([]string(nil), rec.Warnings...),
		UpdatedAt:     time.Now().UTC(),
	}
}

func sortedRegisteredModels(input map[string]RegisteredModel) []RegisteredModel {
	out := make([]RegisteredModel, 0, len(input))
	for _, model := range input {
		out = append(out, model)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Manifest.ID < out[j].Manifest.ID
	})
	return out
}
