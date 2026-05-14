package hostbridge

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testSnapshotSampler struct {
	snapshots []*Snapshot
	errs      []error
	calls     int
}

func (s *testSnapshotSampler) CaptureSnapshot(context.Context) (*Snapshot, error) {
	idx := s.calls
	s.calls++
	if idx < len(s.errs) && s.errs[idx] != nil {
		return nil, s.errs[idx]
	}
	if idx < len(s.snapshots) {
		return s.snapshots[idx], nil
	}
	return nil, nil
}

func TestSnapshotCacheMissRefreshesSource(t *testing.T) {
	now := fixedNow()
	sampler := &testSnapshotSampler{snapshots: []*Snapshot{cacheSnapshot("hostdiag_first", now)}}
	cache := NewSnapshotCache(sampler, SnapshotCacheOptions{TTL: time.Minute, Now: func() time.Time { return now }})

	read, err := cache.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sampler.calls != 1 {
		t.Fatalf("source calls = %d, want 1", sampler.calls)
	}
	if !read.Available || read.CacheHit || read.Stale || read.Snapshot == nil || read.Snapshot.SnapshotID != "hostdiag_first" {
		t.Fatalf("unexpected cache read on miss: %#v", read)
	}
	if read.Age != 0 {
		t.Fatalf("age = %s, want 0", read.Age)
	}
}

func TestSnapshotCacheHitAvoidsSource(t *testing.T) {
	now := fixedNow()
	sampler := &testSnapshotSampler{snapshots: []*Snapshot{cacheSnapshot("hostdiag_first", now)}}
	cache := NewSnapshotCache(sampler, SnapshotCacheOptions{TTL: time.Minute, Now: func() time.Time { return now }})
	if _, err := cache.Get(context.Background()); err != nil {
		t.Fatal(err)
	}

	now = now.Add(30 * time.Second)
	read, err := cache.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sampler.calls != 1 {
		t.Fatalf("source calls = %d, want 1", sampler.calls)
	}
	if !read.CacheHit || read.Stale || read.Age != 30*time.Second || read.Snapshot.SnapshotID != "hostdiag_first" {
		t.Fatalf("unexpected cache hit read: %#v", read)
	}
}

func TestSnapshotCacheStaleTTLRefreshes(t *testing.T) {
	now := fixedNow()
	sampler := &testSnapshotSampler{snapshots: []*Snapshot{
		cacheSnapshot("hostdiag_first", now),
		cacheSnapshot("hostdiag_second", now.Add(2*time.Minute)),
	}}
	cache := NewSnapshotCache(sampler, SnapshotCacheOptions{TTL: time.Minute, Now: func() time.Time { return now }})
	if _, err := cache.Get(context.Background()); err != nil {
		t.Fatal(err)
	}

	now = now.Add(2 * time.Minute)
	read, err := cache.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sampler.calls != 2 {
		t.Fatalf("source calls = %d, want 2", sampler.calls)
	}
	if read.CacheHit || read.Stale || read.Snapshot.SnapshotID != "hostdiag_second" {
		t.Fatalf("unexpected stale refresh read: %#v", read)
	}
}

func TestSnapshotCachePreservesSourceErrors(t *testing.T) {
	now := fixedNow()
	sourceErr := errors.New("host sampler failed")
	sampler := &testSnapshotSampler{
		snapshots: []*Snapshot{cacheSnapshot("hostdiag_first", now), nil},
		errs:      []error{nil, sourceErr},
	}
	cache := NewSnapshotCache(sampler, SnapshotCacheOptions{TTL: time.Minute, Now: func() time.Time { return now }})
	if _, err := cache.Get(context.Background()); err != nil {
		t.Fatal(err)
	}

	now = now.Add(2 * time.Minute)
	read, err := cache.Get(context.Background())
	if !errors.Is(err, sourceErr) {
		t.Fatalf("error = %v, want source error", err)
	}
	if !read.Available || !read.Stale || read.SourceError != sourceErr.Error() || read.Snapshot.SnapshotID != "hostdiag_first" {
		t.Fatalf("expected stale cached read with preserved source error, got %#v", read)
	}

	snapshotErr := SourceError{Source: "proc.meminfo", Error: "missing"}
	sampler = &testSnapshotSampler{snapshots: []*Snapshot{{SnapshotID: "hostdiag_with_source_error", CapturedAt: now, SourceErrors: []SourceError{snapshotErr}}}}
	cache = NewSnapshotCache(sampler, SnapshotCacheOptions{TTL: time.Minute, Now: func() time.Time { return now }})
	read, err = cache.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Snapshot.SourceErrors) != 1 || read.Snapshot.SourceErrors[0] != snapshotErr {
		t.Fatalf("snapshot source errors not preserved: %#v", read.Snapshot.SourceErrors)
	}
}

func TestSnapshotCacheNilSnapshotDoesNotPanic(t *testing.T) {
	now := fixedNow()
	cache := NewSnapshotCache(&testSnapshotSampler{}, SnapshotCacheOptions{TTL: time.Minute, Now: func() time.Time { return now }})

	read, err := cache.Get(context.Background())
	if !errors.Is(err, ErrSnapshotNil) {
		t.Fatalf("error = %v, want ErrSnapshotNil", err)
	}
	if read.Available || read.Snapshot != nil || read.SourceError != ErrSnapshotNil.Error() {
		t.Fatalf("unexpected nil snapshot read: %#v", read)
	}
}

func TestSnapshotCacheBoundaryFlagsStayReadOnly(t *testing.T) {
	now := fixedNow()
	cache := NewSnapshotCache(&testSnapshotSampler{snapshots: []*Snapshot{cacheSnapshot("hostdiag_first", now)}}, SnapshotCacheOptions{TTL: time.Minute, Now: func() time.Time { return now }})

	read, err := cache.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !read.ReadOnly || !read.AdvisoryOnly || read.HostMutation || read.SemanticMemoryWrite || read.FORGEKAuthority {
		t.Fatalf("cache boundary flags expanded authority: %#v", read)
	}
}

func cacheSnapshot(id string, capturedAt time.Time) *Snapshot {
	return &Snapshot{SnapshotID: id, CapturedAt: capturedAt}
}
