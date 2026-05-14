package forgeh

import (
	"context"
	"errors"
	"sync"
	"time"
)

const defaultPolicySnapshotCacheTTL = 5 * time.Second

var (
	ErrPolicySnapshotNil       = errors.New("forge-h policy snapshot is nil")
	ErrPolicySnapshotSourceNil = errors.New("forge-h policy snapshot source is nil")
)

type PolicySnapshotSampler interface {
	CapturePolicySnapshot(context.Context) (*ResourcePolicySnapshot, error)
}

type PolicySnapshotCacheOptions struct {
	TTL time.Duration
	Now func() time.Time
}

type PolicySnapshotCache struct {
	mu          sync.Mutex
	source      PolicySnapshotSampler
	ttl         time.Duration
	now         func() time.Time
	policy      *ResourcePolicySnapshot
	refreshedAt time.Time
}

type CachedPolicySnapshot struct {
	Policy              *ResourcePolicySnapshot `json:"policy,omitempty"`
	RefreshedAt         time.Time               `json:"refreshed_at,omitempty"`
	Age                 time.Duration           `json:"age"`
	Available           bool                    `json:"available"`
	CacheHit            bool                    `json:"cache_hit"`
	Stale               bool                    `json:"stale"`
	SourceError         string                  `json:"source_error,omitempty"`
	ReadOnly            bool                    `json:"read_only"`
	AdvisoryOnly        bool                    `json:"advisory_only"`
	HostMutation        bool                    `json:"host_mutation"`
	SemanticMemoryWrite bool                    `json:"semantic_memory_write"`
	FORGEKAuthority     bool                    `json:"forge_k_authority"`
	SourceErrors        []ResourcePolicyError   `json:"source_errors,omitempty"`
	Warnings            []string                `json:"warnings,omitempty"`
}

func NewPolicySnapshotCache(source PolicySnapshotSampler, opts PolicySnapshotCacheOptions) *PolicySnapshotCache {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = defaultPolicySnapshotCacheTTL
	}
	return &PolicySnapshotCache{source: source, ttl: ttl, now: now}
}

func (c *PolicySnapshotCache) Get(ctx context.Context) (CachedPolicySnapshot, error) {
	if c == nil {
		c = NewPolicySnapshotCache(nil, PolicySnapshotCacheOptions{})
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now().UTC()
	if c.policy != nil && !c.isStaleLocked(now) {
		return c.readLocked(now, true, false, nil), nil
	}
	if c.source == nil {
		return c.readLocked(now, false, c.policy != nil, ErrPolicySnapshotSourceNil), ErrPolicySnapshotSourceNil
	}
	policy, err := c.source.CapturePolicySnapshot(ctx)
	if err != nil {
		return c.readLocked(now, false, c.policy != nil, err), err
	}
	if policy == nil {
		return c.readLocked(now, false, c.policy != nil, ErrPolicySnapshotNil), ErrPolicySnapshotNil
	}
	c.policy = clonePolicySnapshot(policy)
	c.refreshedAt = now
	return c.readLocked(now, false, false, nil), nil
}

func (c *PolicySnapshotCache) Current() (CachedPolicySnapshot, error) {
	if c == nil {
		c = NewPolicySnapshotCache(nil, PolicySnapshotCacheOptions{})
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now().UTC()
	if c.policy == nil {
		return c.readLocked(now, false, false, ErrPolicySnapshotNil), ErrPolicySnapshotNil
	}
	return c.readLocked(now, true, c.isStaleLocked(now), nil), nil
}

func (c *PolicySnapshotCache) isStaleLocked(now time.Time) bool {
	if c.policy == nil {
		return true
	}
	return now.Sub(c.refreshedAt) > c.ttl
}

func (c *PolicySnapshotCache) readLocked(now time.Time, cacheHit bool, stale bool, sourceErr error) CachedPolicySnapshot {
	read := CachedPolicySnapshot{
		Available:           c.policy != nil,
		CacheHit:            cacheHit,
		Stale:               stale,
		ReadOnly:            true,
		AdvisoryOnly:        true,
		HostMutation:        false,
		SemanticMemoryWrite: false,
		FORGEKAuthority:     false,
	}
	if c.policy != nil {
		read.Policy = clonePolicySnapshot(c.policy)
		read.RefreshedAt = c.refreshedAt
		read.Age = now.Sub(c.refreshedAt)
		read.SourceErrors = append([]ResourcePolicyError{}, c.policy.SourceErrors...)
		read.Warnings = append([]string{}, c.policy.Warnings...)
	}
	if sourceErr != nil {
		read.SourceError = sourceErr.Error()
	}
	return read
}

func clonePolicySnapshot(policy *ResourcePolicySnapshot) *ResourcePolicySnapshot {
	if policy == nil {
		return nil
	}
	clone := *policy
	clone.LaneDecisions = make(map[string]LaneDecision, len(policy.LaneDecisions))
	for lane, decision := range policy.LaneDecisions {
		decision.Reasons = append([]string{}, decision.Reasons...)
		clone.LaneDecisions[lane] = decision
	}
	clone.Warnings = append([]string{}, policy.Warnings...)
	clone.OperatorActions = append([]string{}, policy.OperatorActions...)
	clone.SourceErrors = append([]ResourcePolicyError{}, policy.SourceErrors...)
	return &clone
}
