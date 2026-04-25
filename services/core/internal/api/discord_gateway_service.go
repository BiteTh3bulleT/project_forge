package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/dashboard"
)

type DiscordGateway struct {
	server *Server
	cfg    discordGatewayConfig

	mu      sync.RWMutex
	session *discordgo.Session
	status  discordGatewayStatus
	botID   string
}

func newDiscordGateway(s *Server, cfg discordGatewayConfig) (*DiscordGateway, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if strings.TrimSpace(cfg.BotToken) == "" {
		return nil, fmt.Errorf("discord gateway enabled but bot token is missing")
	}
	if !cfg.EnableSlashCommands && !cfg.EnableTextCommands && !cfg.EnablePassiveListen {
		return nil, fmt.Errorf("discord gateway enabled but all input modes are disabled")
	}
	gateway := &DiscordGateway{
		server: s,
		cfg:    cfg,
		status: discordGatewayStatus{
			Enabled:          cfg.Enabled,
			ApplicationID:    cfg.ApplicationID,
			GuildID:          cfg.GuildID,
			CommandPrefix:    cfg.normalizedPrefix(),
			EnableSlash:      cfg.EnableSlashCommands,
			EnableText:       cfg.EnableTextCommands,
			EnablePassive:    cfg.EnablePassiveListen,
			EnableOutbound:   cfg.EnableOutbound,
			CrossChatContext: cfg.CrossChatContext,
		},
	}
	return gateway, nil
}

func (g *DiscordGateway) Start(ctx context.Context) error {
	session, err := discordgo.New("Bot " + g.cfg.BotToken)
	if err != nil {
		g.setLastError(err)
		return err
	}

	intents := discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages
	if g.cfg.EnableTextCommands || g.cfg.EnablePassiveListen {
		intents |= discordgo.IntentsMessageContent
	}
	session.Identify.Intents = intents

	session.AddHandler(func(_ *discordgo.Session, ready *discordgo.Ready) {
		g.onReady(ready)
	})
	session.AddHandler(func(_ *discordgo.Session, msg *discordgo.MessageCreate) {
		g.handleMessageCreate(msg)
	})
	session.AddHandler(func(_ *discordgo.Session, ic *discordgo.InteractionCreate) {
		g.handleInteractionCreate(ic)
	})

	if err := session.Open(); err != nil {
		g.setLastError(err)
		_ = session.Close()
		return err
	}
	g.mu.Lock()
	g.session = session
	g.status.Connected = true
	g.status.StartedAtMs = time.Now().UnixMilli()
	if strings.TrimSpace(g.status.ApplicationID) == "" && session.State != nil && session.State.User != nil {
		g.status.ApplicationID = strings.TrimSpace(session.State.User.ID)
	}
	g.mu.Unlock()

	if g.cfg.EnableSlashCommands {
		if err := g.registerSlashCommands(); err != nil {
			g.setLastError(err)
			log.Printf("discord gateway: slash registration failed: %v", err)
		}
	}
	_ = g.emitEvent(ctx, "discord.gateway.started", map[string]any{
		"guildId":       g.cfg.GuildID,
		"applicationId": g.cfg.ApplicationID,
		"slash":         g.cfg.EnableSlashCommands,
		"text":          g.cfg.EnableTextCommands,
		"passive":       g.cfg.EnablePassiveListen,
		"outbound":      g.cfg.EnableOutbound,
	})

	go func() {
		<-ctx.Done()
		g.Stop()
	}()
	return nil
}

func (g *DiscordGateway) Stop() {
	g.mu.Lock()
	session := g.session
	g.session = nil
	g.status.Connected = false
	g.mu.Unlock()
	if session != nil {
		_ = session.Close()
	}
}

func (g *DiscordGateway) Status() discordGatewayStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := g.status
	out.RegisteredCommands = append([]string{}, g.status.RegisteredCommands...)
	return out
}

func (g *DiscordGateway) onReady(ready *discordgo.Ready) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if ready != nil && ready.User != nil {
		g.botID = strings.TrimSpace(ready.User.ID)
	}
}

func (g *DiscordGateway) handleMessageCreate(msg *discordgo.MessageCreate) {
	if msg == nil || msg.Message == nil || msg.Author == nil || msg.Author.Bot {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 190*time.Second)
	defer cancel()

	envelope, err := normalizeDiscordMessageEvent(msg)
	if err != nil {
		g.recordGatewayError(ctx, "message_normalization", err, nil)
		return
	}
	if !g.allowedGuild(envelope.GuildID) {
		return
	}
	g.recordInbound(ctx, envelope)

	intent, ok := routeDiscordTextIntent(envelope, g.cfg.normalizedPrefix(), g.botID)
	if !ok && g.cfg.EnablePassiveListen && (envelope.GuildID == "" || messageMentionsBot(msg, g.botID)) {
		intent = newDiscordIntent(envelope, discordIntentConversational, "conversation")
		intent.ArgumentText = envelope.RawContent
		intent.Content = envelope.RawContent
		ok = true
	}
	if !ok {
		return
	}
	intent.Source = "discord.message"
	if err := g.processIntentAndRespond(ctx, envelope, intent, msg.ChannelID, msg.ID); err != nil {
		g.recordGatewayError(ctx, "message_process", err, &envelope)
	}
}

func (g *DiscordGateway) handleInteractionCreate(ic *discordgo.InteractionCreate) {
	if ic == nil || ic.Interaction == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	envelope, err := normalizeDiscordInteractionEvent(ic)
	if err != nil {
		g.recordGatewayError(ctx, "interaction_normalization", err, nil)
		_ = g.respondInteraction(ic, discordResponse{
			Kind:      discordResponseError,
			Content:   "Malformed interaction payload.",
			Ephemeral: true,
			ErrorCode: "MALFORMED_PAYLOAD",
		})
		return
	}
	if !g.allowedGuild(envelope.GuildID) {
		_ = g.respondInteraction(ic, discordResponse{
			Kind:      discordResponseError,
			Content:   "This Forge Discord gateway is not enabled for this guild.",
			Ephemeral: true,
			ErrorCode: "GUILD_DENIED",
		})
		return
	}
	g.recordInbound(ctx, envelope)

	intent, err := routeDiscordInteractionIntent(envelope, ic)
	if err != nil {
		g.recordGatewayError(ctx, "interaction_route", err, &envelope)
		_ = g.respondInteraction(ic, discordResponse{
			Kind:      discordResponseError,
			Content:   "Unable to parse interaction command.",
			Ephemeral: true,
			ErrorCode: "INVALID_COMMAND",
		})
		return
	}
	intent.Source = "discord.interaction"
	response, execErr := g.executeIntent(ctx, envelope, intent)
	if execErr != nil {
		g.recordGatewayError(ctx, "interaction_execute", execErr, &envelope)
		response = discordResponse{
			Kind:      discordResponseError,
			Content:   execErr.Error(),
			Ephemeral: true,
			ErrorCode: "EXECUTION_FAILED",
		}
	}
	_ = g.respondInteraction(ic, response)
	g.recordOutbound(ctx, envelope, intent, response)
}

func (g *DiscordGateway) processIntentAndRespond(ctx context.Context, envelope discordEventEnvelope, intent discordIntent, channelID, replyToID string) error {
	response, err := g.executeIntent(ctx, envelope, intent)
	if err != nil {
		response = discordResponse{
			Kind:      discordResponseError,
			Content:   err.Error(),
			Ephemeral: false,
			ErrorCode: "EXECUTION_FAILED",
		}
	}
	if sendErr := g.sendChannelResponse(ctx, channelID, replyToID, response); sendErr != nil {
		g.recordGatewayError(ctx, "channel_send", sendErr, &envelope)
		return sendErr
	}
	g.recordOutbound(ctx, envelope, intent, response)
	return nil
}

func (g *DiscordGateway) executeIntent(ctx context.Context, envelope discordEventEnvelope, intent discordIntent) (discordResponse, error) {
	decision := authorizeDiscordIntent(intent, envelope.Actor, g.cfg)
	if !decision.Allowed {
		_ = g.recordAudit(ctx, envelope, intent, "denied", decision.Reason, map[string]any{"reason": decision.Reason})
		return discordResponse{
			Kind:      discordResponseError,
			Content:   "Permission denied: " + decision.Reason,
			ErrorCode: "PERMISSION_DENIED",
			Ephemeral: true,
		}, nil
	}

	_ = g.recordAudit(ctx, envelope, intent, "accepted", "intent accepted", map[string]any{"class": intent.Class, "command": intent.Command})
	switch intent.Command {
	case "ping":
		latency := time.Now().UnixMilli() - envelope.TimestampMs
		return discordResponse{
			Kind:    discordResponseStatus,
			Content: fmt.Sprintf("pong | source=discord | latency=%dms | correlation=%s", latency, envelope.CorrelationID),
			Metadata: map[string]any{
				"latencyMs": latency,
			},
		}, nil
	case "status":
		return g.statusIntentResponse(ctx, envelope)
	case "help":
		return g.helpIntentResponse(), nil
	case "memory_query":
		return g.memoryQueryIntentResponse(ctx, intent.ArgumentText)
	case "agents":
		return g.agentsIntentResponse(ctx), nil
	case "conversation":
		return g.conversationIntentResponse(ctx, envelope, intent)
	case "unknown":
		return discordResponse{
			Kind:      discordResponseError,
			Content:   "Unknown command. Use /forge help or !forge help.",
			ErrorCode: "UNKNOWN_COMMAND",
			Ephemeral: true,
		}, nil
	default:
		return discordResponse{
			Kind:      discordResponseDiagnostic,
			Content:   "Command is recognized by the gateway but not yet implemented.",
			ErrorCode: "NOT_IMPLEMENTED",
			Ephemeral: true,
		}, nil
	}
}

func (g *DiscordGateway) statusIntentResponse(ctx context.Context, envelope discordEventEnvelope) (discordResponse, error) {
	summary, err := g.server.dashboard.Summary(ctx)
	if err != nil {
		return discordResponse{}, err
	}
	autonomyMode := "off"
	dreamState := "unknown"
	if g.server.autonomy != nil {
		autonomyMode = string(g.server.autonomy.Mode())
		if g.server.autonomy.Status().Active {
			dreamState = "active"
		} else {
			dreamState = "idle"
		}
	}
	lines := []string{
		fmt.Sprintf("workspace=%s", g.server.cfg.WorkspaceDir),
		fmt.Sprintf("active_jobs=%d", len(summary.ActiveJobs)),
		fmt.Sprintf("approvals_pending=%d", summary.ApprovalsPending),
		fmt.Sprintf("reviews_pending=%d", summary.ReviewsPending),
		fmt.Sprintf("autonomy_mode=%s", autonomyMode),
		fmt.Sprintf("dream_state=%s", dreamState),
		fmt.Sprintf("correlation=%s", envelope.CorrelationID),
	}
	return discordResponse{
		Kind:    discordResponseStatus,
		Content: strings.Join(lines, "\n"),
	}, nil
}

func (g *DiscordGateway) helpIntentResponse() discordResponse {
	help := []string{
		"Forge Discord interface",
		"- /forge ping",
		"- /forge status",
		"- /forge memory query <text>",
		"- /forge agents (admin)",
		"- /forge help",
		"",
		fmt.Sprintf("Text fallback: %s ping | status | memory query <text> | agents | help", g.cfg.normalizedPrefix()),
	}
	return discordResponse{Kind: discordResponsePlain, Content: strings.Join(help, "\n")}
}

func (g *DiscordGateway) memoryQueryIntentResponse(ctx context.Context, query string) (discordResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return discordResponse{
			Kind:      discordResponseError,
			Content:   "Memory query requires text. Example: /forge memory query packet alignment",
			ErrorCode: "MISSING_QUERY",
			Ephemeral: true,
		}, nil
	}
	hits, err := g.server.search.Search(ctx, query, 5)
	if err != nil {
		return discordResponse{}, err
	}
	if len(hits) == 0 {
		return discordResponse{
			Kind:    discordResponsePlain,
			Content: "No memory hits found for query: " + query,
		}, nil
	}
	lines := []string{fmt.Sprintf("Top memory hits for %q:", query)}
	for idx, hit := range hits {
		lines = append(lines, fmt.Sprintf("%d) %s (score=%.3f)", idx+1, truncateDiscordText(hit.RelPath, 72), hit.Score))
		snippet := strings.TrimSpace(hit.Snippet)
		if snippet != "" {
			lines = append(lines, "   "+truncateDiscordText(snippet, 132))
		}
	}
	return discordResponse{Kind: discordResponsePlain, Content: strings.Join(lines, "\n")}, nil
}

func (g *DiscordGateway) agentsIntentResponse(ctx context.Context) discordResponse {
	infos := g.server.adapters.List(ctx)
	if len(infos) == 0 {
		return discordResponse{Kind: discordResponseDiagnostic, Content: "No adapters registered."}
	}
	lines := []string{"Registered adapters:"}
	for _, info := range infos {
		lines = append(lines, fmt.Sprintf("- %s: %s (%s)", info.ID, info.Status, info.Detail))
	}
	return discordResponse{Kind: discordResponseDiagnostic, Content: strings.Join(lines, "\n")}
}

func (g *DiscordGateway) conversationIntentResponse(ctx context.Context, envelope discordEventEnvelope, intent discordIntent) (discordResponse, error) {
	if strings.TrimSpace(intent.Content) == "" {
		return discordResponse{
			Kind:      discordResponseError,
			Content:   "Conversation input cannot be empty.",
			ErrorCode: "EMPTY_INPUT",
			Ephemeral: true,
		}, nil
	}

	replyToID := strings.TrimSpace(envelope.MessageID)
	channelID := strings.TrimSpace(envelope.ChannelID)
	if channelID == "" {
		channelID = strings.TrimSpace(g.cfg.DefaultChannelID)
	}
	sendReply := func(innerCtx context.Context, reply string) error {
		if channelID == "" {
			return nil
		}
		return g.sendChannelResponse(innerCtx, channelID, replyToID, discordResponse{
			Kind:    discordResponsePlain,
			Content: reply,
		})
	}

	threadID, err := g.server.enqueueDiscordConversation(ctx, envelope, intent, sendReply)
	if err != nil {
		return discordResponse{}, err
	}
	return discordResponse{
		Kind: discordResponseStatus,
		Content: fmt.Sprintf(
			"Conversation queued for FORGE processing.\nthread_id=%d\ncorrelation=%s",
			threadID,
			envelope.CorrelationID,
		),
	}, nil
}

func (g *DiscordGateway) allowedGuild(guildID string) bool {
	if strings.TrimSpace(g.cfg.GuildID) == "" {
		return true
	}
	current := strings.TrimSpace(guildID)
	if current == "" {
		return true
	}
	return current == strings.TrimSpace(g.cfg.GuildID)
}

func messageMentionsBot(msg *discordgo.MessageCreate, botID string) bool {
	if msg == nil || msg.Message == nil {
		return false
	}
	botID = strings.TrimSpace(botID)
	if botID == "" {
		return false
	}
	for _, user := range msg.Mentions {
		if user == nil {
			continue
		}
		if strings.TrimSpace(user.ID) == botID {
			return true
		}
	}
	return false
}

func (g *DiscordGateway) registerSlashCommands() error {
	g.mu.RLock()
	session := g.session
	g.mu.RUnlock()
	if session == nil {
		return fmt.Errorf("discord session is not connected")
	}
	appID := strings.TrimSpace(g.cfg.ApplicationID)
	if appID == "" {
		appID = strings.TrimSpace(g.botID)
	}
	if appID == "" && session.State != nil && session.State.User != nil {
		appID = strings.TrimSpace(session.State.User.ID)
	}
	if appID == "" {
		return fmt.Errorf("discord application id unavailable: set FORGE_DISCORD_APP_ID or allow bot ready state before slash registration")
	}

	commands, err := session.ApplicationCommandBulkOverwrite(appID, strings.TrimSpace(g.cfg.GuildID), forgeSlashCommands())
	if err != nil {
		return err
	}
	registered := make([]string, 0, len(commands))
	for _, command := range commands {
		registered = append(registered, command.Name)
	}
	g.mu.Lock()
	g.status.RegisteredCommands = registered
	g.mu.Unlock()
	return nil
}

func forgeSlashCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "forge",
			Description: "Forge command surface",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "status",
					Description: "Show Forge status summary",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "ping",
					Description: "Ping Forge gateway",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "help",
					Description: "Show Forge Discord help",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "agents",
					Description: "List adapter and agent status (admin)",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "memory",
					Description: "Query Forge memory index",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "query",
							Description: "Memory query text",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (g *DiscordGateway) sendChannelResponse(ctx context.Context, channelID, replyToID string, resp discordResponse) error {
	if !g.cfg.EnableOutbound {
		return nil
	}
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return fmt.Errorf("discord outbound channel id is required")
	}
	content := formatDiscordResponse(resp)

	g.mu.RLock()
	session := g.session
	g.mu.RUnlock()
	if session == nil {
		return fmt.Errorf("discord session unavailable")
	}

	msg := &discordgo.MessageSend{
		Content: content,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{},
		},
	}
	if strings.TrimSpace(replyToID) != "" {
		msg.Reference = &discordgo.MessageReference{
			MessageID: replyToID,
			ChannelID: channelID,
		}
	}
	if _, err := session.ChannelMessageSendComplex(channelID, msg); err != nil {
		return err
	}

	g.mu.Lock()
	g.status.LastOutboundAtMs = time.Now().UnixMilli()
	g.status.OutboundCount++
	g.mu.Unlock()
	return nil
}

func (g *DiscordGateway) respondInteraction(ic *discordgo.InteractionCreate, resp discordResponse) error {
	if ic == nil || ic.Interaction == nil {
		return errDiscordMalformedPayload
	}
	g.mu.RLock()
	session := g.session
	g.mu.RUnlock()
	if session == nil {
		return fmt.Errorf("discord session unavailable")
	}
	content := formatDiscordResponse(resp)
	flags := discordgo.MessageFlags(0)
	if resp.Ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	return session.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   flags,
		},
	})
}

func (g *DiscordGateway) recordInbound(ctx context.Context, envelope discordEventEnvelope) {
	g.mu.Lock()
	g.status.LastInboundAtMs = time.Now().UnixMilli()
	g.status.InboundCount++
	g.mu.Unlock()
	_ = g.emitEvent(ctx, "discord.gateway.inbound", map[string]any{
		"correlationId": envelope.CorrelationID,
		"eventType":     envelope.EventType,
		"guildId":       envelope.GuildID,
		"channelId":     envelope.ChannelID,
		"userId":        envelope.UserID,
	})
}

func (g *DiscordGateway) recordOutbound(ctx context.Context, envelope discordEventEnvelope, intent discordIntent, resp discordResponse) {
	_ = g.emitEvent(ctx, "discord.gateway.outbound", map[string]any{
		"correlationId": envelope.CorrelationID,
		"intentId":      intent.ID,
		"command":       intent.Command,
		"kind":          resp.Kind,
		"errorCode":     resp.ErrorCode,
	})
}

func (g *DiscordGateway) recordGatewayError(ctx context.Context, stage string, err error, envelope *discordEventEnvelope) {
	g.setLastError(err)
	payload := map[string]any{
		"stage": stage,
		"error": err.Error(),
	}
	if envelope != nil {
		payload["correlationId"] = envelope.CorrelationID
		payload["channelId"] = envelope.ChannelID
		payload["userId"] = envelope.UserID
	}
	_ = g.emitEvent(ctx, "discord.gateway.error", payload)
}

func (g *DiscordGateway) emitEvent(ctx context.Context, typ string, payload map[string]any) error {
	if g.server == nil || g.server.log == nil {
		return nil
	}
	return g.server.log.Emit(ctx, typ, payload)
}

func (g *DiscordGateway) recordAudit(ctx context.Context, envelope discordEventEnvelope, intent discordIntent, outcome, summary string, payload map[string]any) error {
	if g.server == nil || g.server.auditSvc == nil {
		return nil
	}
	body := map[string]any{
		"event":  envelope,
		"intent": intent,
	}
	for key, value := range payload {
		body[key] = value
	}
	_, err := g.server.auditSvc.Record(ctx, audit.CreateRequest{
		CorrelationID: envelope.CorrelationID,
		Category:      "discord_gateway",
		Action:        intent.Command,
		Actor:         "discord:" + envelope.Actor.ExternalID,
		SubjectType:   "discord_event",
		SubjectID:     firstNonEmpty(envelope.MessageID, envelope.InteractionID, envelope.CorrelationID),
		RiskClass:     "low",
		Outcome:       outcome,
		Summary:       summary,
		Payload:       body,
	})
	return err
}

func (g *DiscordGateway) setLastError(err error) {
	if err == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.status.LastError = err.Error()
}

func truncateDiscordText(raw string, max int) string {
	raw = strings.TrimSpace(raw)
	if max <= 0 || len(raw) <= max {
		return raw
	}
	return strings.TrimSpace(raw[:max]) + "..."
}

func (g *DiscordGateway) marshalStatusJSON() string {
	st := g.Status()
	raw, _ := json.Marshal(st)
	return string(raw)
}

func dashboardSummaryLite(summary *dashboard.Summary) map[string]any {
	if summary == nil {
		return map[string]any{}
	}
	return map[string]any{
		"activeJobs":       len(summary.ActiveJobs),
		"approvalsPending": summary.ApprovalsPending,
		"reviewsPending":   summary.ReviewsPending,
	}
}
