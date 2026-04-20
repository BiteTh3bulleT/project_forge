package api

import "testing"

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
