package api

import "testing"

func TestDiscordGatewaySourceKeyPerChannelUser(t *testing.T) {
	t.Parallel()

	envelope := discordEventEnvelope{
		GuildID:   "guild_1",
		ChannelID: "channel_9",
		Actor: discordActorIdentity{
			ExternalID: "user_3",
		},
	}
	got := discordGatewaySourceKeyFromEnvelope(envelope, false)
	want := "discord_gateway:channel_9:user_3"
	if got != want {
		t.Fatalf("source key = %q, want %q", got, want)
	}
}

func TestDiscordGatewaySourceKeyCrossChat(t *testing.T) {
	t.Parallel()

	envelope := discordEventEnvelope{
		GuildID:   "guild_1",
		ChannelID: "channel_9",
		Actor: discordActorIdentity{
			ExternalID: "user_3",
		},
	}
	got := discordGatewaySourceKeyFromEnvelope(envelope, true)
	want := "discord_gateway_shared:guild_1:shared"
	if got != want {
		t.Fatalf("source key = %q, want %q", got, want)
	}
}

func TestDiscordGatewaySourceKeyCrossChatGlobalFallback(t *testing.T) {
	t.Parallel()

	envelope := discordEventEnvelope{
		ChannelID: "dm-channel",
		Actor: discordActorIdentity{
			ExternalID: "user_3",
		},
	}
	got := discordGatewaySourceKeyFromEnvelope(envelope, true)
	want := "discord_gateway_shared:global:shared"
	if got != want {
		t.Fatalf("source key = %q, want %q", got, want)
	}
}
