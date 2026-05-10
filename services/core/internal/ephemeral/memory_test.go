package ephemeral

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMemoryStoreCacheSetGetRequiresTTL(t *testing.T) {
	ctx := context.Background()
	policy := DefaultKeyPolicy("forge")
	store := NewMemoryStore(policy)
	key, err := policy.Key(KeyKindCache, "workspace-1", "job-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetCache(ctx, key, []byte("value"), 0); !errors.Is(err, ErrTTLRequired) {
		t.Fatalf("expected ttl required, got %v", err)
	}
	if err := store.SetCache(ctx, key, []byte("value"), time.Minute); err != nil {
		t.Fatalf("set cache failed: %v", err)
	}
	value, ok, err := store.GetCache(ctx, key)
	if err != nil {
		t.Fatalf("get cache failed: %v", err)
	}
	if !ok || string(value) != "value" {
		t.Fatalf("unexpected cache result ok=%v value=%q", ok, value)
	}
}

func TestMemoryStoreQueuePushPop(t *testing.T) {
	ctx := context.Background()
	policy := DefaultKeyPolicy("forge")
	store := NewMemoryStore(policy)
	key, err := policy.Key(KeyKindQueue, "workspace-1", "job-events")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PushQueue(ctx, key, []byte("first")); err != nil {
		t.Fatalf("push first failed: %v", err)
	}
	if err := store.PushQueue(ctx, key, []byte("second")); err != nil {
		t.Fatalf("push second failed: %v", err)
	}
	value, ok, err := store.PopQueue(ctx, key)
	if err != nil {
		t.Fatalf("pop failed: %v", err)
	}
	if !ok || string(value) != "first" {
		t.Fatalf("unexpected first pop ok=%v value=%q", ok, value)
	}
	value, ok, err = store.PopQueue(ctx, key)
	if err != nil {
		t.Fatalf("pop failed: %v", err)
	}
	if !ok || string(value) != "second" {
		t.Fatalf("unexpected second pop ok=%v value=%q", ok, value)
	}
	_, ok, err = store.PopQueue(ctx, key)
	if err != nil {
		t.Fatalf("empty pop failed: %v", err)
	}
	if ok {
		t.Fatalf("expected empty queue")
	}
}

func TestMemoryStoreLockAcquireRelease(t *testing.T) {
	ctx := context.Background()
	policy := DefaultKeyPolicy("forge")
	store := NewMemoryStore(policy)
	key, err := policy.Key(KeyKindLock, "workspace-1", "job-1")
	if err != nil {
		t.Fatal(err)
	}
	held, err := store.AcquireLock(ctx, key, "owner-a", time.Minute)
	if err != nil || !held {
		t.Fatalf("expected lock acquired held=%v err=%v", held, err)
	}
	held, err = store.AcquireLock(ctx, key, "owner-b", time.Minute)
	if err != nil {
		t.Fatalf("second acquire failed: %v", err)
	}
	if held {
		t.Fatalf("expected second owner not to acquire held lock")
	}
	if err := store.ReleaseLock(ctx, key, "owner-b"); !errors.Is(err, ErrLockHeld) {
		t.Fatalf("expected wrong owner release to fail, got %v", err)
	}
	if err := store.ReleaseLock(ctx, key, "owner-a"); err != nil {
		t.Fatalf("release failed: %v", err)
	}
}

func TestMemoryStoreProgressAppendRead(t *testing.T) {
	ctx := context.Background()
	policy := DefaultKeyPolicy("forge")
	store := NewMemoryStore(policy)
	key, err := policy.Key(KeyKindProgress, "workspace-1", "job-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AppendProgress(ctx, key, ProgressEntry{ID: "1", Message: "started"}, 0); !errors.Is(err, ErrTTLRequired) {
		t.Fatalf("expected ttl required for progress, got %v", err)
	}
	if err := store.AppendProgress(ctx, key, ProgressEntry{ID: "1", Message: "started"}, time.Minute); err != nil {
		t.Fatalf("append progress failed: %v", err)
	}
	if err := store.AppendProgress(ctx, key, ProgressEntry{ID: "2", Message: "done"}, time.Minute); err != nil {
		t.Fatalf("append progress failed: %v", err)
	}
	entries, err := store.ReadProgress(ctx, key, 1)
	if err != nil {
		t.Fatalf("read progress failed: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "2" {
		t.Fatalf("expected latest entry only, got %+v", entries)
	}
}

func TestMemoryStoreReadProgressDefaultsToBoundedLimit(t *testing.T) {
	ctx := context.Background()
	policy := DefaultKeyPolicy("forge")
	store := NewMemoryStore(policy)
	key, err := policy.Key(KeyKindProgress, "workspace-1", "job-1")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxEphemeralProgressReadEntries+1; i++ {
		entry := ProgressEntry{ID: strconv.Itoa(i), Message: "step"}
		if err := store.AppendProgress(ctx, key, entry, time.Minute); err != nil {
			t.Fatalf("append progress %d failed: %v", i, err)
		}
	}
	entries, err := store.ReadProgress(ctx, key, 0)
	if err != nil {
		t.Fatalf("read progress failed: %v", err)
	}
	if len(entries) != maxEphemeralProgressReadEntries {
		t.Fatalf("expected bounded progress read length %d, got %d", maxEphemeralProgressReadEntries, len(entries))
	}
	if entries[0].ID != "1" {
		t.Fatalf("expected oldest retained read entry to be 1, got %q", entries[0].ID)
	}
}

func TestMemoryStoreRejectsUnsafeKey(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(DefaultKeyPolicy("forge"))
	err := store.SetCache(ctx, "forge:cache:raw-prompt", []byte("value"), time.Minute)
	if !errors.Is(err, ErrUnsafeKey) {
		t.Fatalf("expected unsafe key rejection, got %v", err)
	}
}

func TestMemoryStoreRejectsOversizeValues(t *testing.T) {
	ctx := context.Background()
	policy := DefaultKeyPolicy("forge")
	store := NewMemoryStore(policy)
	oversize := make([]byte, maxEphemeralValueBytes+1)

	cacheKey, err := policy.Key(KeyKindCache, "workspace-1", "job-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetCache(ctx, cacheKey, oversize, time.Minute); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("expected cache value size rejection, got %v", err)
	}

	queueKey, err := policy.Key(KeyKindQueue, "workspace-1", "job-events")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PushQueue(ctx, queueKey, oversize); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("expected queue value size rejection, got %v", err)
	}

	pubsubKey, err := policy.Key(KeyKindPubSub, "workspace-1", "events")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Publish(ctx, pubsubKey, oversize); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("expected pubsub value size rejection, got %v", err)
	}

	lockKey, err := policy.Key(KeyKindLock, "workspace-1", "job-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcquireLock(ctx, lockKey, strings.Repeat("o", maxEphemeralValueBytes+1), time.Minute); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("expected lock owner size rejection, got %v", err)
	}

	progressKey, err := policy.Key(KeyKindProgress, "workspace-1", "job-1")
	if err != nil {
		t.Fatal(err)
	}
	entry := ProgressEntry{ID: "1", Message: strings.Repeat("m", maxEphemeralValueBytes+1)}
	if err := store.AppendProgress(ctx, progressKey, entry, time.Minute); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("expected progress value size rejection, got %v", err)
	}
}

func TestMemoryStoreHealth(t *testing.T) {
	status := NewMemoryStore(DefaultKeyPolicy("forge")).Health(context.Background())
	if !status.Enabled || !status.OK {
		t.Fatalf("unexpected health status %+v", status)
	}
}
