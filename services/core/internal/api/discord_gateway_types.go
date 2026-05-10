package api

import (
	"fmt"
	"os"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/config"
)

const (
	discordGatewayThreadMapKey        = "discord_gateway_thread_map"
	discordGatewayCrossChatContextKey = "discord_gateway_cross_chat_context"
)

type discordIntentClass string

const discordIngressTextLimit = 16 << 10

const (
	discordIntentDirectCommand   discordIntentClass = "direct_command"
	discordIntentConversational  discordIntentClass = "conversational_input"
	discordIntentSystemQuery     discordIntentClass = "system_query"
	discordIntentAgentControl    discordIntentClass = "agent_control_request"
	discordIntentMemoryQuery     discordIntentClass = "memory_query"
	discordIntentAutomationEvent discordIntentClass = "automation_event"
)

type discordEventType string

const (
	discordEventMessageCreate     discordEventType = "message_create"
	discordEventInteractionCreate discordEventType = "interaction_create"
)

type discordResponseKind string

const (
	discordResponsePlain      discordResponseKind = "plain"
	discordResponseStatus     discordResponseKind = "status"
	discordResponseError      discordResponseKind = "error"
	discordResponseDiagnostic discordResponseKind = "diagnostic"
)

type discordGatewayConfig struct {
	Enabled bool

	BotToken         string
	ApplicationID    string
	GuildID          string
	DefaultChannelID string

	CommandPrefix string

	EnableSlashCommands bool
	EnableTextCommands  bool
	EnablePassiveListen bool
	EnableOutbound      bool
	CrossChatContext    bool

	AdminUserIDs map[string]struct{}
	AdminRoleIDs map[string]struct{}
}

type discordGatewayStatus struct {
	Enabled            bool     `json:"enabled"`
	Connected          bool     `json:"connected"`
	ApplicationID      string   `json:"applicationId"`
	GuildID            string   `json:"guildId,omitempty"`
	CommandPrefix      string   `json:"commandPrefix"`
	EnableSlash        bool     `json:"enableSlash"`
	EnableText         bool     `json:"enableText"`
	EnablePassive      bool     `json:"enablePassive"`
	EnableOutbound     bool     `json:"enableOutbound"`
	CrossChatContext   bool     `json:"crossChatContext"`
	RegisteredCommands []string `json:"registeredCommands"`
	LastError          string   `json:"lastError,omitempty"`
	StartedAtMs        int64    `json:"startedAtMs,omitempty"`
	LastInboundAtMs    int64    `json:"lastInboundAtMs,omitempty"`
	LastOutboundAtMs   int64    `json:"lastOutboundAtMs,omitempty"`
	InboundCount       int64    `json:"inboundCount"`
	OutboundCount      int64    `json:"outboundCount"`
}

type discordActorIdentity struct {
	ExternalID  string   `json:"externalId"`
	Username    string   `json:"username,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
	RoleIDs     []string `json:"roleIds,omitempty"`
	IsBot       bool     `json:"isBot"`
	IsAdmin     bool     `json:"isAdmin"`
}

type discordEventEnvelope struct {
	Source         string               `json:"source"`
	EventType      discordEventType     `json:"eventType"`
	GuildID        string               `json:"guildId,omitempty"`
	ChannelID      string               `json:"channelId,omitempty"`
	UserID         string               `json:"userId,omitempty"`
	Username       string               `json:"username,omitempty"`
	DisplayName    string               `json:"displayName,omitempty"`
	MessageID      string               `json:"messageId,omitempty"`
	InteractionID  string               `json:"interactionId,omitempty"`
	TimestampMs    int64                `json:"timestampMs"`
	RawContent     string               `json:"rawContent"`
	Metadata       map[string]any       `json:"metadata"`
	CorrelationID  string               `json:"correlationId"`
	TraceID        string               `json:"traceId"`
	PermissionInfo map[string]any       `json:"permissionContext"`
	Actor          discordActorIdentity `json:"actor"`
}

type discordIntent struct {
	ID           string             `json:"id"`
	Class        discordIntentClass `json:"class"`
	Command      string             `json:"command"`
	Args         []string           `json:"args"`
	ArgumentText string             `json:"argumentText,omitempty"`
	Content      string             `json:"content,omitempty"`
	Source       string             `json:"source"`
	Metadata     map[string]any     `json:"metadata"`
}

type discordResponse struct {
	Kind      discordResponseKind `json:"kind"`
	Content   string              `json:"content"`
	Warnings  []string            `json:"warnings,omitempty"`
	ErrorCode string              `json:"errorCode,omitempty"`
	Ephemeral bool                `json:"ephemeral"`
	Metadata  map[string]any      `json:"metadata,omitempty"`
}

type discordPermissionDecision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

func (c discordGatewayConfig) normalizedPrefix() string {
	p := strings.TrimSpace(c.CommandPrefix)
	if p == "" {
		return "!forge"
	}
	return p
}

func defaultDiscordGatewayConfig(cfg config.Config) discordGatewayConfig {
	prefix := strings.TrimSpace(os.Getenv("FORGE_DISCORD_COMMAND_PREFIX"))
	if prefix == "" {
		prefix = "!forge"
	}
	return discordGatewayConfig{
		Enabled:             parseEnvBoolWithDefault("FORGE_DISCORD_ENABLED", false),
		BotToken:            strings.TrimSpace(os.Getenv("FORGE_DISCORD_BOT_TOKEN")),
		ApplicationID:       strings.TrimSpace(os.Getenv("FORGE_DISCORD_APP_ID")),
		GuildID:             strings.TrimSpace(os.Getenv("FORGE_DISCORD_GUILD_ID")),
		DefaultChannelID:    strings.TrimSpace(os.Getenv("FORGE_DISCORD_DEFAULT_CHANNEL_ID")),
		CommandPrefix:       prefix,
		EnableSlashCommands: parseEnvBoolWithDefault("FORGE_DISCORD_ENABLE_SLASH_COMMANDS", true),
		EnableTextCommands:  parseEnvBoolWithDefault("FORGE_DISCORD_ENABLE_TEXT_COMMANDS", true),
		EnablePassiveListen: parseEnvBoolWithDefault("FORGE_DISCORD_ENABLE_PASSIVE_LISTENING", false),
		EnableOutbound:      parseEnvBoolWithDefault("FORGE_DISCORD_ENABLE_OUTBOUND_POSTING", true),
		CrossChatContext:    parseEnvBoolWithDefault("FORGE_DISCORD_CROSS_CHAT_CONTEXT", false),
		AdminUserIDs:        parseCSVSet(os.Getenv("FORGE_DISCORD_ADMIN_USER_IDS")),
		AdminRoleIDs:        parseCSVSet(os.Getenv("FORGE_DISCORD_ADMIN_ROLE_IDS")),
	}
}

func parseEnvBoolWithDefault(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	return parseRemoteBool(raw)
}

func parseCSVSet(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, token := range strings.Split(raw, ",") {
		item := strings.TrimSpace(token)
		if item == "" {
			continue
		}
		out[item] = struct{}{}
	}
	return out
}

func formatDiscordResponse(resp discordResponse) string {
	content := strings.TrimSpace(resp.Content)
	if content == "" {
		content = "(empty response)"
	}
	switch resp.Kind {
	case discordResponseStatus:
		return "FORGE status:\n" + content
	case discordResponseDiagnostic:
		return "FORGE diagnostic:\n" + content
	case discordResponseError:
		return "FORGE error:\n" + content
	default:
		return content
	}
}

func newDiscordCorrelationID(kind discordEventType, id string) string {
	clean := strings.TrimSpace(id)
	if clean == "" {
		return fmt.Sprintf("discord:%s:%d", kind, time.Now().UnixNano())
	}
	return fmt.Sprintf("discord:%s:%s", kind, clean)
}
