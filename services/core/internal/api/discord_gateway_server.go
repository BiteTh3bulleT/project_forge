package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"forge/projectforge/services/core/internal/config"
)

func (s *Server) loadDiscordGatewayConfig(_ context.Context, cfg config.Config) discordGatewayConfig {
	out := defaultDiscordGatewayConfig(cfg)

	// Settings fallback keeps the desktop settings page useful when env values
	// are not supplied. Environment variables remain the primary source.
	if strings.TrimSpace(out.BotToken) == "" {
		out.BotToken = strings.TrimSpace(loadSetting(s.st.DB, discordBotTokenKey, ""))
	}
	if strings.TrimSpace(out.DefaultChannelID) == "" {
		out.DefaultChannelID = strings.TrimSpace(loadSetting(s.st.DB, discordDefaultChannelIDKey, ""))
	}
	if setting := strings.TrimSpace(loadSetting(s.st.DB, discordGatewayCrossChatContextKey, "")); setting != "" {
		out.CrossChatContext = parseRemoteBool(setting)
	}
	if !out.Enabled {
		// Enabling via settings keeps backward compatibility for operators who
		// already configured remote Discord ingress in Forge settings.
		out.Enabled = parseRemoteBool(loadSetting(s.st.DB, remoteAccessEnabledKey, "false"))
	}
	return out
}

func (s *Server) tryStartDiscordGateway(ctx context.Context, cfg config.Config) *DiscordGateway {
	dcfg := s.loadDiscordGatewayConfig(ctx, cfg)
	gateway, err := newDiscordGateway(s, dcfg)
	if err != nil {
		logLocal("discord gateway initialization skipped: %v", err)
		s.discordMu.Lock()
		s.discordErr = err.Error()
		s.discordMu.Unlock()
		return nil
	}
	if gateway == nil {
		s.discordMu.Lock()
		s.discordErr = "discord gateway disabled by configuration"
		s.discordMu.Unlock()
		return nil
	}
	if err := gateway.Start(ctx); err != nil {
		logLocal("discord gateway startup failed: %v", err)
		s.discordMu.Lock()
		s.discordErr = err.Error()
		s.discordMu.Unlock()
		return nil
	}
	s.discordMu.Lock()
	s.discordErr = ""
	s.discordMu.Unlock()
	logLocal("discord gateway started (slash=%t text=%t passive=%t outbound=%t guild=%q)",
		dcfg.EnableSlashCommands, dcfg.EnableTextCommands, dcfg.EnablePassiveListen, dcfg.EnableOutbound, dcfg.GuildID)
	return gateway
}

func (s *Server) handleDiscordGatewayStatus(w http.ResponseWriter, r *http.Request) {
	s.discordMu.RLock()
	gateway := s.discordGateway
	reason := strings.TrimSpace(s.discordErr)
	s.discordMu.RUnlock()
	if gateway == nil {
		if reason == "" {
			reason = "discord gateway not started"
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false,
			"status":  "disabled",
			"reason":  reason,
		})
		return
	}
	status := gateway.Status()
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": true,
		"status":  status,
	})
}

func (s *Server) reloadDiscordGateway(ctx context.Context) {
	s.discordMu.Lock()
	current := s.discordGateway
	s.discordGateway = nil
	s.discordMu.Unlock()

	if current != nil {
		current.Stop()
	}

	next := s.tryStartDiscordGateway(ctx, s.cfg)
	s.discordMu.Lock()
	s.discordGateway = next
	s.discordMu.Unlock()
}

func (s *Server) enqueueDiscordConversation(
	ctx context.Context,
	envelope discordEventEnvelope,
	intent discordIntent,
	sendReply func(context.Context, string) error,
) (int64, error) {
	conf := remoteConfig{
		defaultThreadID: parseRemoteInt64(loadSetting(s.st.DB, remoteDefaultThreadIDKey, "")),
		threadMap:       remoteParseThreadMap(loadSetting(s.st.DB, discordGatewayThreadMapKey, "{}")),
	}
	sourceKey := s.discordGatewaySourceKey(envelope)
	threadID, changed, err := s.resolveRemoteThread(ctx, conf, sourceKey)
	if err != nil {
		return 0, err
	}
	if changed {
		if err := s.saveDiscordGatewayThreadMap(ctx, conf.threadMap); err != nil {
			return 0, err
		}
	}

	um, err := s.chat.AppendMessage(ctx, threadID, "user", strings.TrimSpace(intent.Content), map[string]any{
		"source":             "discord.gateway",
		"sourceKey":          sourceKey,
		"discordEventType":   envelope.EventType,
		"discordGuildId":     envelope.GuildID,
		"discordChannelId":   envelope.ChannelID,
		"discordMessageId":   envelope.MessageID,
		"discordUserId":      envelope.UserID,
		"discordDisplayName": firstNonEmpty(envelope.DisplayName, envelope.Username),
		"correlationId":      envelope.CorrelationID,
		"traceId":            envelope.TraceID,
		"threadSource":       true,
	})
	if err != nil {
		return 0, err
	}

	ollamaAdapter, err := s.adapters.Get("ollama")
	if err != nil {
		return 0, err
	}
	th, err := s.chat.GetThread(ctx, threadID)
	if err != nil {
		return 0, err
	}

	_ = s.log.Emit(ctx, "discord.gateway.intent.enqueued", map[string]any{
		"threadId":      threadID,
		"messageId":     um.ID,
		"correlationId": envelope.CorrelationID,
		"sourceKey":     sourceKey,
		"intentClass":   intent.Class,
		"intentCommand": intent.Command,
	})

	key := chatInflightKey(threadID, um.ID)
	if _, loaded := s.chatAssistInflight.LoadOrStore(key, true); loaded {
		return 0, fmt.Errorf("assistant generation already in progress for message %d", um.ID)
	}
	go s.runRemoteAssistantAsync(
		key,
		threadID,
		um.ID,
		th,
		intent.Content,
		ollamaAdapter,
		sendReply,
	)
	return threadID, nil
}

func (s *Server) discordGatewaySourceKey(envelope discordEventEnvelope) string {
	crossChat := parseEnvBoolWithDefault("FORGE_DISCORD_CROSS_CHAT_CONTEXT", false)
	if setting := strings.TrimSpace(loadSetting(s.st.DB, discordGatewayCrossChatContextKey, "")); setting != "" {
		crossChat = parseRemoteBool(setting)
	}
	return discordGatewaySourceKeyFromEnvelope(envelope, crossChat)
}

func discordGatewaySourceKeyFromEnvelope(envelope discordEventEnvelope, crossChat bool) string {
	if crossChat {
		scope := strings.TrimSpace(envelope.GuildID)
		if scope == "" {
			scope = "global"
		}
		return remoteSourceKey("discord_gateway_shared", scope, "shared")
	}
	return remoteSourceKey("discord_gateway", envelope.ChannelID, envelope.Actor.ExternalID)
}

func (s *Server) saveDiscordGatewayThreadMap(ctx context.Context, m map[string]int64) error {
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return upsertSetting(ctx, s.st.DB, discordGatewayThreadMapKey, string(raw))
}

func logLocal(format string, args ...any) {
	// Keep startup/shutdown transport diagnostics visible even before event
	// pipelines are initialized.
	fmt.Printf("[forge-core] "+format+"\n", args...)
}
