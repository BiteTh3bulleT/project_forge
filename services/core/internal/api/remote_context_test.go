package api

import (
	"context"
	"testing"
)

func TestRemoteSourceKeyForPlatformDefault(t *testing.T) {
	t.Parallel()

	conf := remoteConfig{crossChatContext: false}
	got := remoteSourceKeyForPlatform(conf, "telegram", "123", "456")
	want := "telegram:123:456"
	if got != want {
		t.Fatalf("source key = %q, want %q", got, want)
	}
}

func TestRemoteSourceKeyForPlatformCrossChat(t *testing.T) {
	t.Parallel()

	conf := remoteConfig{crossChatContext: true}
	got := remoteSourceKeyForPlatform(conf, "telegram", "123", "456")
	want := "telegram_shared:global:shared"
	if got != want {
		t.Fatalf("source key = %q, want %q", got, want)
	}
}

func TestResolveRemoteThreadCrossChatReusesSharedThread(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	conf := remoteConfig{
		crossChatContext: true,
		threadMap:        map[string]int64{},
	}
	ctx := context.Background()

	firstKey := remoteSourceKeyForPlatform(conf, "telegram", "chat-a", "sender-a")
	firstThreadID, changed, err := srv.resolveRemoteThread(ctx, conf, firstKey)
	if err != nil {
		t.Fatalf("resolve first thread: %v", err)
	}
	if !changed {
		t.Fatalf("expected first shared thread resolution to update map")
	}

	secondKey := remoteSourceKeyForPlatform(conf, "telegram", "chat-b", "sender-b")
	secondThreadID, changed, err := srv.resolveRemoteThread(ctx, conf, secondKey)
	if err != nil {
		t.Fatalf("resolve second thread: %v", err)
	}
	if firstKey != secondKey {
		t.Fatalf("expected cross-chat source keys to match, got %q and %q", firstKey, secondKey)
	}
	if changed {
		t.Fatalf("expected second shared thread resolution to reuse existing map")
	}
	if secondThreadID != firstThreadID {
		t.Fatalf("expected shared thread id %d, got %d", firstThreadID, secondThreadID)
	}
}
