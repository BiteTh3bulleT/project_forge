package forgeh

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testPolicySampler struct {
	policies []*ResourcePolicySnapshot
	errs     []error
	calls    int
}

func (s *testPolicySampler) CapturePolicySnapshot(context.Context) (*ResourcePolicySnapshot, error) {
	idx := s.calls
	s.calls++
	if idx < len(s.errs) && s.errs[idx] != nil {
		return nil, s.errs[idx]
	}
	if idx < len(s.policies) {
		return s.policies[idx], nil
	}
	return nil, nil
}

func TestPolicySnapshotCacheHitMissAndStaleRefresh(t *testing.T) {
	now := fixedNow()
	sampler := &testPolicySampler{policies: []*ResourcePolicySnapshot{
		cachePolicy("forgeh_policy_first", now),
		cachePolicy("forgeh_policy_second", now.Add(2*time.Minute)),
	}}
	cache := NewPolicySnapshotCache(sampler, PolicySnapshotCacheOptions{TTL: time.Minute, Now: func() time.Time { return now }})

	first, err := cache.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sampler.calls != 1 || first.CacheHit || first.Stale || first.Policy.PolicyID != "forgeh_policy_first" {
		t.Fatalf("unexpected miss read: calls=%d read=%#v", sampler.calls, first)
	}

	now = now.Add(30 * time.Second)
	hit, err := cache.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sampler.calls != 1 || !hit.CacheHit || hit.Stale || hit.Age != 30*time.Second {
		t.Fatalf("unexpected hit read: calls=%d read=%#v", sampler.calls, hit)
	}

	now = now.Add(90 * time.Second)
	stale, err := cache.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sampler.calls != 2 || stale.CacheHit || stale.Stale || stale.Policy.PolicyID != "forgeh_policy_second" {
		t.Fatalf("unexpected stale refresh read: calls=%d read=%#v", sampler.calls, stale)
	}
}

func TestPolicySnapshotCachePreservesErrorsAndAdvisoryPosture(t *testing.T) {
	now := fixedNow()
	sourceErr := errors.New("policy sampler failed")
	sampler := &testPolicySampler{
		policies: []*ResourcePolicySnapshot{cachePolicy("forgeh_policy_first", now), nil},
		errs:     []error{nil, sourceErr},
	}
	cache := NewPolicySnapshotCache(sampler, PolicySnapshotCacheOptions{TTL: time.Minute, Now: func() time.Time { return now }})
	if _, err := cache.Get(context.Background()); err != nil {
		t.Fatal(err)
	}

	now = now.Add(2 * time.Minute)
	read, err := cache.Get(context.Background())
	if !errors.Is(err, sourceErr) {
		t.Fatalf("error = %v, want source error", err)
	}
	if !read.Available || !read.Stale || read.SourceError != sourceErr.Error() || read.Policy.PolicyID != "forgeh_policy_first" {
		t.Fatalf("expected stale cached read with source error, got %#v", read)
	}
	if !read.ReadOnly || !read.AdvisoryOnly || read.HostMutation || read.SemanticMemoryWrite || read.FORGEKAuthority {
		t.Fatalf("policy cache boundary flags expanded authority: %#v", read)
	}
}

func TestPolicySnapshotCacheNilSnapshotDoesNotPanic(t *testing.T) {
	now := fixedNow()
	cache := NewPolicySnapshotCache(&testPolicySampler{}, PolicySnapshotCacheOptions{TTL: time.Minute, Now: func() time.Time { return now }})

	read, err := cache.Get(context.Background())
	if !errors.Is(err, ErrPolicySnapshotNil) {
		t.Fatalf("error = %v, want ErrPolicySnapshotNil", err)
	}
	if read.Available || read.Policy != nil || read.SourceError != ErrPolicySnapshotNil.Error() {
		t.Fatalf("unexpected nil policy read: %#v", read)
	}
}

func cachePolicy(id string, capturedAt time.Time) *ResourcePolicySnapshot {
	return &ResourcePolicySnapshot{
		PolicyID:       id,
		CapturedAt:     capturedAt,
		AdvisoryOnly:   true,
		OverallPosture: ResourcePostureNormal,
	}
}
