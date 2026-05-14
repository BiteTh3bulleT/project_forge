package hostbridge

import (
	"context"
	"sync"
	"time"
)

const defaultSnapshotCacheTTL = 5 * time.Second

type SnapshotSampler interface {
	CaptureSnapshot(context.Context) (*Snapshot, error)
}

type SnapshotCacheOptions struct {
	TTL time.Duration
	Now func() time.Time
}

type SnapshotCache struct {
	mu          sync.Mutex
	source      SnapshotSampler
	ttl         time.Duration
	now         func() time.Time
	snapshot    *Snapshot
	refreshedAt time.Time
}

type CachedSnapshot struct {
	Snapshot            *Snapshot     `json:"snapshot,omitempty"`
	RefreshedAt         time.Time     `json:"refreshed_at,omitempty"`
	Age                 time.Duration `json:"age"`
	Available           bool          `json:"available"`
	CacheHit            bool          `json:"cache_hit"`
	Stale               bool          `json:"stale"`
	SourceError         string        `json:"source_error,omitempty"`
	ReadOnly            bool          `json:"read_only"`
	AdvisoryOnly        bool          `json:"advisory_only"`
	HostMutation        bool          `json:"host_mutation"`
	SemanticMemoryWrite bool          `json:"semantic_memory_write"`
	FORGEKAuthority     bool          `json:"forge_k_authority"`
	SourceErrors        []SourceError `json:"source_errors,omitempty"`
	Warnings            []string      `json:"warnings,omitempty"`
}

func NewSnapshotCache(source SnapshotSampler, opts SnapshotCacheOptions) *SnapshotCache {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = defaultSnapshotCacheTTL
	}
	return &SnapshotCache{source: source, ttl: ttl, now: now}
}

func (s *Service) CaptureSnapshot(ctx context.Context) (*Snapshot, error) {
	if s == nil {
		s = New(Options{})
	}
	snapshot := s.Snapshot(ctx)
	return &snapshot, nil
}

func (c *SnapshotCache) Get(ctx context.Context) (CachedSnapshot, error) {
	if c == nil {
		c = NewSnapshotCache(nil, SnapshotCacheOptions{})
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now().UTC()
	if c.snapshot != nil && !c.isStaleLocked(now) {
		return c.readLocked(now, true, false, nil), nil
	}

	if c.source == nil {
		return c.readLocked(now, false, c.snapshot != nil, ErrSnapshotSourceNil), ErrSnapshotSourceNil
	}
	snapshot, err := c.source.CaptureSnapshot(ctx)
	if err != nil {
		return c.readLocked(now, false, c.snapshot != nil, err), err
	}
	if snapshot == nil {
		return c.readLocked(now, false, c.snapshot != nil, ErrSnapshotNil), ErrSnapshotNil
	}
	c.snapshot = cloneSnapshot(snapshot)
	c.refreshedAt = now
	return c.readLocked(now, false, false, nil), nil
}

func (c *SnapshotCache) Current() (CachedSnapshot, error) {
	if c == nil {
		c = NewSnapshotCache(nil, SnapshotCacheOptions{})
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now().UTC()
	if c.snapshot == nil {
		return c.readLocked(now, false, false, ErrSnapshotNil), ErrSnapshotNil
	}
	return c.readLocked(now, true, c.isStaleLocked(now), nil), nil
}

func (c *SnapshotCache) isStaleLocked(now time.Time) bool {
	if c.snapshot == nil {
		return true
	}
	return now.Sub(c.refreshedAt) > c.ttl
}

func (c *SnapshotCache) readLocked(now time.Time, cacheHit bool, stale bool, sourceErr error) CachedSnapshot {
	read := CachedSnapshot{
		Available:           c.snapshot != nil,
		CacheHit:            cacheHit,
		Stale:               stale,
		ReadOnly:            true,
		AdvisoryOnly:        true,
		HostMutation:        false,
		SemanticMemoryWrite: false,
		FORGEKAuthority:     false,
	}
	if c.snapshot != nil {
		read.Snapshot = cloneSnapshot(c.snapshot)
		read.RefreshedAt = c.refreshedAt
		read.Age = now.Sub(c.refreshedAt)
		read.SourceErrors = append([]SourceError{}, c.snapshot.SourceErrors...)
		read.Warnings = append([]string{}, c.snapshot.Warnings...)
	}
	if sourceErr != nil {
		read.SourceError = sourceErr.Error()
	}
	return read
}

func cloneSnapshot(snapshot *Snapshot) *Snapshot {
	if snapshot == nil {
		return nil
	}
	clone := *snapshot
	clone.Kernel.Modules = append([]ModuleDiagnostics{}, snapshot.Kernel.Modules...)
	clone.Boot.Parameters = append([]string{}, snapshot.Boot.Parameters...)
	clone.CPU.LoadAverage = append([]float64{}, snapshot.CPU.LoadAverage...)
	clone.GPU.Devices = append([]GPUDeviceDiagnostics{}, snapshot.GPU.Devices...)
	clone.GPU.Warnings = append([]string{}, snapshot.GPU.Warnings...)
	clone.Thermal.Sensors = append([]ThermalSensor{}, snapshot.Thermal.Sensors...)
	clone.Thermal.Warnings = append([]string{}, snapshot.Thermal.Warnings...)
	clone.Services.Units = append([]UnitState{}, snapshot.Services.Units...)
	clone.Services.Failed = append([]UnitState{}, snapshot.Services.Failed...)
	clone.Services.Warnings = append([]string{}, snapshot.Services.Warnings...)
	clone.ModelRuntime.Warnings = append([]string{}, snapshot.ModelRuntime.Warnings...)
	clone.Warnings = append([]string{}, snapshot.Warnings...)
	clone.Redactions = append([]RedactionRecord{}, snapshot.Redactions...)
	clone.SourceErrors = append([]SourceError{}, snapshot.SourceErrors...)
	return &clone
}
