package runtime

import (
	"context"
	"sort"
	"sync"
)

type Registry struct {
	mu      sync.RWMutex
	drivers map[string]RuntimeDriver
}

func NewRegistry() *Registry {
	return &Registry{drivers: make(map[string]RuntimeDriver)}
}

func (r *Registry) RegisterDriver(driver RuntimeDriver) (RuntimeDriverManifest, error) {
	manifest, err := NewRuntimeDriverManifest(driver.Manifest())
	if err != nil {
		return RuntimeDriverManifest{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.drivers[manifest.DriverID]; exists {
		return RuntimeDriverManifest{}, ErrDriverAlreadyRegistered
	}
	r.drivers[manifest.DriverID] = driver
	return manifest, nil
}

func (r *Registry) GetDriver(driverID string) (RuntimeDriver, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	driver, ok := r.drivers[driverID]
	return driver, ok
}

func (r *Registry) GetManifest(driverID string) (RuntimeDriverManifest, bool) {
	driver, ok := r.GetDriver(driverID)
	if !ok {
		return RuntimeDriverManifest{}, false
	}
	manifest, err := NewRuntimeDriverManifest(driver.Manifest())
	if err != nil {
		return RuntimeDriverManifest{}, false
	}
	return manifest, true
}

func (r *Registry) ListManifests() []RuntimeDriverManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]RuntimeDriverManifest, 0, len(r.drivers))
	for _, driver := range r.drivers {
		manifest, err := NewRuntimeDriverManifest(driver.Manifest())
		if err != nil {
			continue
		}
		out = append(out, manifest)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DriverID < out[j].DriverID })
	return out
}

func (r *Registry) Capabilities(ctx context.Context, driverID, modelID string) (RuntimeCapabilityManifest, error) {
	driver, ok := r.GetDriver(driverID)
	if !ok {
		return RuntimeCapabilityManifest{}, ErrDriverNotFound
	}
	capability, err := driver.Capabilities(ctx, modelID)
	if err != nil {
		return RuntimeCapabilityManifest{}, err
	}
	if err := ValidateCapabilityManifest(capability); err != nil {
		return RuntimeCapabilityManifest{}, err
	}
	return NormalizeCapabilityManifest(capability), nil
}

func (r *Registry) Health(ctx context.Context, driverID string) (RuntimeHealth, error) {
	driver, ok := r.GetDriver(driverID)
	if !ok {
		return RuntimeHealth{}, ErrDriverNotFound
	}
	return driver.Health(ctx)
}
