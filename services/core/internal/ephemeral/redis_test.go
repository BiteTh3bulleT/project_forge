package ephemeral

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRedisStoreDisabledDoesNotConnect(t *testing.T) {
	store, err := NewRedisStore(RedisConfig{
		Enabled:   false,
		Addr:      "",
		KeyPolicy: DefaultKeyPolicy("forge"),
		Timeout:   time.Millisecond,
	})
	if err != nil {
		t.Fatalf("disabled redis should construct without addr: %v", err)
	}
	status := store.Health(context.Background())
	if status.Enabled || !status.OK {
		t.Fatalf("unexpected disabled health %+v", status)
	}
}

func TestRedisStoreEnabledRequiresConfig(t *testing.T) {
	if _, err := NewRedisStore(RedisConfig{
		Enabled:   true,
		KeyPolicy: DefaultKeyPolicy("forge"),
		Timeout:   time.Second,
	}); err == nil {
		t.Fatalf("expected enabled redis without addr to fail")
	}
	if _, err := NewRedisStore(RedisConfig{
		Enabled:   true,
		Addr:      "127.0.0.1:6379",
		KeyPolicy: DefaultKeyPolicy("forge"),
		Timeout:   0,
	}); err == nil {
		t.Fatalf("expected enabled redis without timeout to fail")
	}
}

func TestRedisIntegrationOptional(t *testing.T) {
	addr := os.Getenv("FORGE_REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("FORGE_REDIS_TEST_ADDR not set")
	}
	ctx := context.Background()
	policy := DefaultKeyPolicy("forge-test")
	store, err := NewRedisStore(RedisConfig{
		Enabled:   true,
		Addr:      addr,
		KeyPolicy: policy,
		Timeout:   2 * time.Second,
	})
	if err != nil {
		t.Fatalf("new redis store: %v", err)
	}
	if status := store.Health(ctx); !status.Enabled || !status.OK {
		t.Fatalf("redis health failed: %+v", status)
	}

	cacheKey, err := policy.Key(KeyKindCache, "phase13h", StableOpaqueSegment(t.Name(), "cache"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetCache(ctx, cacheKey, []byte("value"), time.Minute); err != nil {
		t.Fatalf("set cache: %v", err)
	}
	value, ok, err := store.GetCache(ctx, cacheKey)
	if err != nil || !ok || string(value) != "value" {
		t.Fatalf("get cache ok=%v value=%q err=%v", ok, value, err)
	}

	queueKey, err := policy.Key(KeyKindQueue, "phase13h", StableOpaqueSegment(t.Name(), "queue"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PushQueue(ctx, queueKey, []byte("queued")); err != nil {
		t.Fatalf("push queue: %v", err)
	}
	value, ok, err = store.PopQueue(ctx, queueKey)
	if err != nil || !ok || string(value) != "queued" {
		t.Fatalf("pop queue ok=%v value=%q err=%v", ok, value, err)
	}

	lockKey, err := policy.Key(KeyKindLock, "phase13h", StableOpaqueSegment(t.Name(), "lock"))
	if err != nil {
		t.Fatal(err)
	}
	held, err := store.AcquireLock(ctx, lockKey, "owner", time.Minute)
	if err != nil || !held {
		t.Fatalf("acquire lock held=%v err=%v", held, err)
	}
	if err := store.ReleaseLock(ctx, lockKey, "owner"); err != nil {
		t.Fatalf("release lock: %v", err)
	}
}
