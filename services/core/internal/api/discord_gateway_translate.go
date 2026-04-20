package api

import (
	"errors"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	errDiscordMalformedPayload = errors.New("discord payload malformed")
)

func normalizeDiscordMessageEvent(msg *discordgo.MessageCreate) (discordEventEnvelope, error) {
	if msg == nil || msg.Message == nil {
		return discordEventEnvelope{}, errDiscordMalformedPayload
	}
	user := msg.Author
	if user == nil {
		return discordEventEnvelope{}, errDiscordMalformedPayload
	}
	var memberNick string
	var memberRoles []string
	if msg.Member != nil {
		memberNick = strings.TrimSpace(msg.Member.Nick)
		memberRoles = normalizeStringSlice(msg.Member.Roles)
	}
	envelope := discordEventEnvelope{
		Source:      "discord",
		EventType:   discordEventMessageCreate,
		GuildID:     strings.TrimSpace(msg.GuildID),
		ChannelID:   strings.TrimSpace(msg.ChannelID),
		UserID:      strings.TrimSpace(user.ID),
		Username:    strings.TrimSpace(user.Username),
		DisplayName: strings.TrimSpace(user.GlobalName),
		MessageID:   strings.TrimSpace(msg.ID),
		TimestampMs: time.Now().UnixMilli(),
		RawContent:  strings.TrimSpace(msg.Content),
		Metadata: map[string]any{
			"isDM":         strings.TrimSpace(msg.GuildID) == "",
			"mentionCount": len(msg.Mentions),
		},
		CorrelationID: newDiscordCorrelationID(discordEventMessageCreate, msg.ID),
		TraceID:       strings.TrimSpace(msg.ID),
		PermissionInfo: map[string]any{
			"guildId":   strings.TrimSpace(msg.GuildID),
			"channelId": strings.TrimSpace(msg.ChannelID),
		},
		Actor: discordActorIdentity{
			ExternalID:  strings.TrimSpace(user.ID),
			Username:    strings.TrimSpace(user.Username),
			DisplayName: firstNonEmpty(memberNick, strings.TrimSpace(user.GlobalName)),
			RoleIDs:     memberRoles,
			IsBot:       user.Bot,
		},
	}
	if !msg.Timestamp.IsZero() {
		envelope.TimestampMs = msg.Timestamp.UnixMilli()
	}
	return envelope, nil
}

func normalizeDiscordInteractionEvent(ic *discordgo.InteractionCreate) (discordEventEnvelope, error) {
	if ic == nil || ic.Interaction == nil {
		return discordEventEnvelope{}, errDiscordMalformedPayload
	}
	user := ic.User
	if user == nil {
		user = ic.Member.User
	}
	if user == nil {
		return discordEventEnvelope{}, errDiscordMalformedPayload
	}
	var memberNick string
	var memberRoles []string
	if ic.Member != nil {
		memberNick = strings.TrimSpace(ic.Member.Nick)
		memberRoles = normalizeStringSlice(ic.Member.Roles)
	}
	envelope := discordEventEnvelope{
		Source:        "discord",
		EventType:     discordEventInteractionCreate,
		GuildID:       strings.TrimSpace(ic.GuildID),
		ChannelID:     strings.TrimSpace(ic.ChannelID),
		UserID:        strings.TrimSpace(user.ID),
		Username:      strings.TrimSpace(user.Username),
		DisplayName:   strings.TrimSpace(user.GlobalName),
		InteractionID: strings.TrimSpace(ic.ID),
		TimestampMs:   time.Now().UnixMilli(),
		RawContent:    strings.TrimSpace(ic.ApplicationCommandData().Name),
		Metadata: map[string]any{
			"interactionType": ic.Type.String(),
		},
		CorrelationID: newDiscordCorrelationID(discordEventInteractionCreate, ic.ID),
		TraceID:       strings.TrimSpace(ic.ID),
		PermissionInfo: map[string]any{
			"guildId":   strings.TrimSpace(ic.GuildID),
			"channelId": strings.TrimSpace(ic.ChannelID),
		},
		Actor: discordActorIdentity{
			ExternalID:  strings.TrimSpace(user.ID),
			Username:    strings.TrimSpace(user.Username),
			DisplayName: firstNonEmpty(memberNick, strings.TrimSpace(user.GlobalName)),
			RoleIDs:     memberRoles,
			IsBot:       user.Bot,
		},
	}
	if ts, err := discordgo.SnowflakeTimestamp(ic.ID); err == nil {
		envelope.TimestampMs = ts.UnixMilli()
	}
	return envelope, nil
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
