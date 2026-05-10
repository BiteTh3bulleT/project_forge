package api

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestRouteDiscordTextIntentCommand(t *testing.T) {
	t.Parallel()

	envelope := discordEventEnvelope{
		Source:     "discord",
		EventType:  discordEventMessageCreate,
		RawContent: "!forge memory query packet alignment",
	}

	intent, ok := routeDiscordTextIntent(envelope, "!forge", "")
	if !ok {
		t.Fatalf("expected command intent")
	}
	if intent.Command != "memory_query" {
		t.Fatalf("command = %q, want memory_query", intent.Command)
	}
	if intent.Class != discordIntentMemoryQuery {
		t.Fatalf("class = %q, want %q", intent.Class, discordIntentMemoryQuery)
	}
	if intent.ArgumentText != "packet alignment" {
		t.Fatalf("argument text = %q", intent.ArgumentText)
	}
}

func TestRouteDiscordTextIntentMentionConversation(t *testing.T) {
	t.Parallel()

	envelope := discordEventEnvelope{
		Source:     "discord",
		EventType:  discordEventMessageCreate,
		RawContent: "<@12345> summarize current open loops",
	}
	intent, ok := routeDiscordTextIntent(envelope, "!forge", "12345")
	if !ok {
		t.Fatalf("expected mention route")
	}
	if intent.Command != "conversation" {
		t.Fatalf("command = %q, want conversation", intent.Command)
	}
	if intent.Class != discordIntentConversational {
		t.Fatalf("class = %q, want %q", intent.Class, discordIntentConversational)
	}
}

func TestRouteDiscordInteractionIntentSlashMemoryQuery(t *testing.T) {
	t.Parallel()

	envelope := discordEventEnvelope{Source: "discord", EventType: discordEventInteractionCreate}
	ic := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:   "175928847299117063",
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "forge",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "memory",
						Type: discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandInteractionDataOption{
							{
								Name:  "query",
								Type:  discordgo.ApplicationCommandOptionString,
								Value: "context compiler timeline",
							},
						},
					},
				},
			},
		},
	}

	intent, err := routeDiscordInteractionIntent(envelope, ic)
	if err != nil {
		t.Fatalf("route interaction: %v", err)
	}
	if intent.Command != "memory_query" {
		t.Fatalf("command = %q, want memory_query", intent.Command)
	}
	if intent.ArgumentText != "context compiler timeline" {
		t.Fatalf("argument text = %q", intent.ArgumentText)
	}
}

func TestRouteDiscordInteractionIntentRejectsOversizeQuery(t *testing.T) {
	t.Parallel()

	envelope := discordEventEnvelope{Source: "discord", EventType: discordEventInteractionCreate}
	ic := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:   "175928847299117063",
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "forge",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "memory",
						Type: discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandInteractionDataOption{
							{
								Name:  "query",
								Type:  discordgo.ApplicationCommandOptionString,
								Value: strings.Repeat("a", discordIngressTextLimit+1),
							},
						},
					},
				},
			},
		},
	}

	if _, err := routeDiscordInteractionIntent(envelope, ic); err == nil {
		t.Fatalf("expected oversize Discord slash query to be rejected")
	} else if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too-large error, got %v", err)
	}
}

func TestRouteDiscordInteractionIntentMalformed(t *testing.T) {
	t.Parallel()

	if _, err := routeDiscordInteractionIntent(discordEventEnvelope{}, nil); err == nil {
		t.Fatalf("expected malformed interaction error")
	}
}
