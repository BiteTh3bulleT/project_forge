package ephemeral

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
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

func TestReadRESPRejectsOversizeBulkString(t *testing.T) {
	_, err := readRESP(bufio.NewReader(strings.NewReader(fmt.Sprintf("$%d\r\n", maxRedisRESPBulkBytes+1))))
	if err == nil || !strings.Contains(err.Error(), "redis bulk response too large") {
		t.Fatalf("readRESP error = %v, want bulk size error", err)
	}
}

func TestReadRESPRejectsOversizeArray(t *testing.T) {
	_, err := readRESP(bufio.NewReader(strings.NewReader(fmt.Sprintf("*%d\r\n", maxRedisRESPArrayItems+1))))
	if err == nil || !strings.Contains(err.Error(), "redis array response too large") {
		t.Fatalf("readRESP error = %v, want array size error", err)
	}
}

func TestReadRESPRejectsOversizeLine(t *testing.T) {
	_, err := readRESP(bufio.NewReader(strings.NewReader("+" + strings.Repeat("x", maxRedisRESPLineBytes+1) + "\r\n")))
	if err == nil || !strings.Contains(err.Error(), "redis line response too large") {
		t.Fatalf("readRESP error = %v, want line size error", err)
	}
}

func TestWriteRESPRejectsOversizeArgumentBeforeWrite(t *testing.T) {
	var buf bytes.Buffer
	err := writeRESP(&buf, "SET", strings.Repeat("x", maxEphemeralValueBytes+1))
	if !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("expected value size rejection, got %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no partial RESP write, got %q", buf.String())
	}
}

func TestRedisStoreRejectsOversizeValuesBeforeDial(t *testing.T) {
	ctx := context.Background()
	policy := DefaultKeyPolicy("forge")
	store, err := NewRedisStore(RedisConfig{
		Enabled:   true,
		Addr:      "127.0.0.1:1",
		KeyPolicy: policy,
		Timeout:   time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
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

func TestNormalizeProgressReadLimitBoundsDefaultAndMaximum(t *testing.T) {
	if got := normalizeProgressReadLimit(0); got != maxEphemeralProgressReadEntries {
		t.Fatalf("default progress read limit = %d, want %d", got, maxEphemeralProgressReadEntries)
	}
	if got := normalizeProgressReadLimit(maxEphemeralProgressReadEntries + 1); got != maxEphemeralProgressReadEntries {
		t.Fatalf("oversize progress read limit = %d, want %d", got, maxEphemeralProgressReadEntries)
	}
	if got := normalizeProgressReadLimit(7); got != 7 {
		t.Fatalf("explicit progress read limit = %d, want 7", got)
	}
}

func TestRedisIntegrationOptional(t *testing.T) {
	addr := requireIntegrationEnvOrSkip(t, "FORGE_REDIS_TEST_ADDR")
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

func requireIntegrationEnvOrSkip(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value != "" {
		return value
	}
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Fatalf("%s must be set in CI for integration coverage", name)
	}
	t.Skipf("%s not set", name)
	return ""
}
