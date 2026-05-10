package api

import (
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestNormalizeDiscordMessageEvent(t *testing.T) {
	t.Parallel()

	ts := time.UnixMilli(1_900_000_000_123)
	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg_1",
			GuildID:   "guild_1",
			ChannelID: "channel_1",
			Content:   " !forge ping ",
			Timestamp: ts,
			Mentions: []*discordgo.User{
				{ID: "u2"},
			},
			Author: &discordgo.User{
				ID:         "u1",
				Username:   "forge_operator",
				GlobalName: "Forge Operator",
			},
			Member: &discordgo.Member{
				Nick:  "Operator",
				Roles: []string{"role_admin", ""},
			},
		},
	}

	envelope, err := normalizeDiscordMessageEvent(msg)
	if err != nil {
		t.Fatalf("normalize message: %v", err)
	}

	if envelope.Source != "discord" {
		t.Fatalf("source = %q, want discord", envelope.Source)
	}
	if envelope.EventType != discordEventMessageCreate {
		t.Fatalf("event type = %q, want %q", envelope.EventType, discordEventMessageCreate)
	}
	if envelope.GuildID != "guild_1" || envelope.ChannelID != "channel_1" {
		t.Fatalf("unexpected scope: guild=%q channel=%q", envelope.GuildID, envelope.ChannelID)
	}
	if envelope.Actor.ExternalID != "u1" || envelope.Actor.DisplayName != "Operator" {
		t.Fatalf("unexpected actor: %+v", envelope.Actor)
	}
	if got := envelope.Metadata["mentionCount"]; got != 1 {
		t.Fatalf("mentionCount = %v, want 1", got)
	}
	if envelope.TimestampMs != ts.UnixMilli() {
		t.Fatalf("timestamp = %d, want %d", envelope.TimestampMs, ts.UnixMilli())
	}
	if !strings.HasPrefix(envelope.CorrelationID, "discord:message_create:") {
		t.Fatalf("correlation id = %q", envelope.CorrelationID)
	}
}

func TestNormalizeDiscordMessageEventRejectsOversizeContent(t *testing.T) {
	t.Parallel()

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg_oversize",
			ChannelID: "channel_1",
			Content:   strings.Repeat("a", discordIngressTextLimit+1),
			Author: &discordgo.User{
				ID:       "u1",
				Username: "forge_operator",
			},
		},
	}

	if _, err := normalizeDiscordMessageEvent(msg); err == nil {
		t.Fatalf("expected oversize Discord message content to be rejected")
	} else if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too-large error, got %v", err)
	}
}

func TestNormalizeDiscordInteractionEvent(t *testing.T) {
	t.Parallel()

	ic := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:        "175928847299117063",
			Type:      discordgo.InteractionApplicationCommand,
			GuildID:   "guild_2",
			ChannelID: "channel_2",
			Member: &discordgo.Member{
				Nick: "Commander",
				Roles: []string{
					"role_ops",
				},
				User: &discordgo.User{
					ID:         "u9",
					Username:   "ops_user",
					GlobalName: "Ops User",
				},
			},
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "forge",
			},
		},
	}

	envelope, err := normalizeDiscordInteractionEvent(ic)
	if err != nil {
		t.Fatalf("normalize interaction: %v", err)
	}

	if envelope.EventType != discordEventInteractionCreate {
		t.Fatalf("event type = %q, want %q", envelope.EventType, discordEventInteractionCreate)
	}
	if envelope.RawContent != "forge" {
		t.Fatalf("raw content = %q, want forge", envelope.RawContent)
	}
	if envelope.Actor.ExternalID != "u9" {
		t.Fatalf("actor id = %q, want u9", envelope.Actor.ExternalID)
	}
	if !strings.HasPrefix(envelope.CorrelationID, "discord:interaction_create:") {
		t.Fatalf("correlation id = %q", envelope.CorrelationID)
	}
}

func TestNormalizeDiscordEventMalformedPayload(t *testing.T) {
	t.Parallel()

	if _, err := normalizeDiscordMessageEvent(nil); err == nil {
		t.Fatalf("expected message malformed error")
	}
	if _, err := normalizeDiscordInteractionEvent(nil); err == nil {
		t.Fatalf("expected interaction malformed error")
	}
}
