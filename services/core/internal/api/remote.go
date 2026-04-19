package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/chat"
)

const (
	remoteAccessEnabledKey     = "remote_access_enabled"
	remoteAccessTokenKey       = "remote_access_token"
	remoteDefaultThreadIDKey   = "remote_default_thread_id"
	telegramBotTokenKey        = "telegram_bot_token"
	telegramDefaultChatIDKey   = "telegram_default_chat_id"
	discordBotTokenKey         = "discord_bot_token"
	discordDefaultChannelIDKey = "discord_default_channel_id"
	discordWebhookURLKey       = "discord_webhook_url"
	remoteThreadMapKey         = "remote_thread_map"

	remoteAssistantTimeout = 185 * time.Second
	remoteOutboundTimeout  = 25 * time.Second
)

type remoteConfig struct {
	enabled             bool
	token               string
	defaultThreadID     int64
	telegramBotToken    string
	telegramDefaultChat string
	discordBotToken     string
	discordDefaultChat  string
	discordWebhookURL   string
	threadMap           map[string]int64
}

type remoteReplyTarget struct {
	platform  string
	chatID    string
	channelID string
	replyToID string
}

type telegramUpdate struct {
	Message     *telegramMessage `json:"message"`
	Edited      *telegramMessage `json:"edited_message"`
	ChannelPost *telegramMessage `json:"channel_post"`
}

type telegramMessage struct {
	MessageID int64         `json:"message_id"`
	Text      string        `json:"text"`
	Caption   string        `json:"caption"`
	Chat      telegramChat  `json:"chat"`
	From      *telegramUser `json:"from"`
}

type telegramChat struct {
	ID int64 `json:"id"`
}

type telegramUser struct {
	ID int64 `json:"id"`
}

type discordMessage struct {
	ID        string       `json:"id"`
	ChannelID string       `json:"channel_id"`
	Content   string       `json:"content"`
	Author    *discordUser `json:"author"`
}

type discordUser struct {
	ID string `json:"id"`
}

func (s *Server) handleRemoteTelegram(w http.ResponseWriter, r *http.Request) {
	conf, err := s.loadRemoteConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if !s.remoteTokenOk(r, conf.token) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload telegramUpdate
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	msg := remoteFirstTelegramMessage(&payload)
	if msg == nil {
		http.Error(w, "no telegram message", http.StatusBadRequest)
		return
	}

	text := remoteTrimmedText(msg.Text, msg.Caption)
	if text == "" {
		http.Error(w, "content required", http.StatusBadRequest)
		return
	}

	chatID := remoteStringID(msg.Chat.ID)
	if chatID == "" {
		chatID = conf.telegramDefaultChat
	}
	if chatID == "" {
		http.Error(w, "telegram chat id not configured", http.StatusBadRequest)
		return
	}

	fromID := remoteStringID(msg.FromID())
	sourceKey := remoteSourceKey("telegram", chatID, fromID)
	sendTarget := remoteReplyTarget{
		platform:  "telegram",
		chatID:    chatID,
		replyToID: remoteStringID(msg.MessageID),
	}

	if err := s.processRemoteMessage(r.Context(), conf, sourceKey, text, sendTarget, func(ctx context.Context, reply string) error {
		return s.sendTelegramReply(ctx, conf.telegramBotToken, sendTarget.chatID, sendTarget.replyToID, reply)
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleRemoteDiscord(w http.ResponseWriter, r *http.Request) {
	conf, err := s.loadRemoteConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if !s.remoteTokenOk(r, conf.token) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload discordMessage
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	text := remoteTrimmedText(payload.Content)
	if text == "" {
		http.Error(w, "content required", http.StatusBadRequest)
		return
	}

	channelID := strings.TrimSpace(payload.ChannelID)
	if channelID == "" {
		channelID = conf.discordDefaultChat
	}

	if channelID == "" && strings.TrimSpace(conf.discordWebhookURL) == "" {
		http.Error(w, "discord channel id not configured", http.StatusBadRequest)
		return
	}

	authorID := ""
	if payload.Author != nil {
		authorID = strings.TrimSpace(payload.Author.ID)
	}

	lookupChannelID := channelID
	if lookupChannelID == "" {
		lookupChannelID = "webhook"
	}

	sourceKey := remoteSourceKey("discord", lookupChannelID, authorID)
	sendTarget := remoteReplyTarget{
		platform:  "discord",
		channelID: channelID,
	}

	if err := s.processRemoteMessage(r.Context(), conf, sourceKey, text, sendTarget, func(ctx context.Context, reply string) error {
		return s.sendDiscordReply(ctx, conf.discordWebhookURL, conf.discordBotToken, sendTarget.channelID, reply)
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) processRemoteMessage(
	ctx context.Context,
	conf remoteConfig,
	sourceKey, content string,
	target remoteReplyTarget,
	sendReply func(context.Context, string) error,
) error {
	threadID, changed, err := s.resolveRemoteThread(ctx, conf, sourceKey)
	if err != nil {
		return err
	}
	if changed {
		if err := s.saveRemoteThreadMap(ctx, conf.threadMap); err != nil {
			return err
		}
	}

	um, err := s.chat.AppendMessage(ctx, threadID, "user", content, map[string]any{
		"source":       target.platform,
		"sourceKey":    sourceKey,
		"target":       remoteTargetLabel(target),
		"threadSource": true,
	})
	if err != nil {
		return err
	}

	ollamaAdapter, err := s.adapters.Get("ollama")
	if err != nil {
		return err
	}

	th, err := s.chat.GetThread(ctx, threadID)
	if err != nil {
		return err
	}

	_ = s.log.Emit(ctx, "chat.message.user", map[string]any{
		"threadId":       threadID,
		"messageId":      um.ID,
		"remoteSource":   sourceKey,
		"remotePlatform": target.platform,
	})

	key := chatInflightKey(threadID, um.ID)
	if _, loaded := s.chatAssistInflight.LoadOrStore(key, true); loaded {
		return fmt.Errorf("assistant generation already in progress for this message")
	}

	go s.runRemoteAssistantAsync(
		key,
		threadID,
		um.ID,
		th,
		content,
		ollamaAdapter,
		sendReply,
	)

	return nil
}

func (s *Server) runRemoteAssistantAsync(
	key string,
	threadID int64,
	userMessageID int64,
	th *chat.ThreadDetail,
	content string,
	ollamaAdapter adapters.Adapter,
	sendReply func(context.Context, string) error,
) {
	defer s.chatAssistInflight.Delete(key)

	asyncCtx := context.Background()
	asyncCtx, cancel := context.WithTimeout(asyncCtx, remoteAssistantTimeout)
	defer cancel()

	am := s.completeAssistantSync(asyncCtx, threadID, userMessageID, th, content, ollamaAdapter, false)
	if am == nil {
		_ = sendReply(asyncCtx, "Assistant reply could not be generated.")
		return
	}

	if err := sendReply(asyncCtx, am.Content); err != nil {
		_ = s.log.Emit(asyncCtx, "remote.reply_send_failed", map[string]any{
			"threadId":  threadID,
			"messageId": userMessageID,
			"error":     err.Error(),
		})
	}
}

func (s *Server) loadRemoteConfig(ctx context.Context) (remoteConfig, error) {
	conf := remoteConfig{
		enabled:             parseRemoteBool(loadSetting(s.st.DB, remoteAccessEnabledKey, "false")),
		token:               strings.TrimSpace(loadSetting(s.st.DB, remoteAccessTokenKey, "")),
		defaultThreadID:     parseRemoteInt64(loadSetting(s.st.DB, remoteDefaultThreadIDKey, "")),
		telegramBotToken:    strings.TrimSpace(loadSetting(s.st.DB, telegramBotTokenKey, "")),
		telegramDefaultChat: strings.TrimSpace(loadSetting(s.st.DB, telegramDefaultChatIDKey, "")),
		discordBotToken:     strings.TrimSpace(loadSetting(s.st.DB, discordBotTokenKey, "")),
		discordDefaultChat:  strings.TrimSpace(loadSetting(s.st.DB, discordDefaultChannelIDKey, "")),
		discordWebhookURL:   strings.TrimSpace(loadSetting(s.st.DB, discordWebhookURLKey, "")),
		threadMap:           remoteParseThreadMap(loadSetting(s.st.DB, remoteThreadMapKey, "{}")),
	}

	if !conf.enabled {
		return conf, fmt.Errorf("remote access disabled")
	}
	if conf.token == "" {
		return conf, fmt.Errorf("remote token not configured")
	}
	if conf.telegramBotToken == "" && conf.discordBotToken == "" && conf.discordWebhookURL == "" {
		return conf, fmt.Errorf("remote platform credentials not configured")
	}

	if conf.defaultThreadID > 0 {
		if _, err := s.chat.GetThread(ctx, conf.defaultThreadID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				conf.defaultThreadID = 0
			} else {
				return conf, err
			}
		}
	}

	return conf, nil
}

func (s *Server) remoteTokenOk(r *http.Request, expected string) bool {
	token := strings.TrimSpace(r.Header.Get("X-Forge-Remote-Token"))
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("X-Telegram-Bot-Api-Secret-Token"))
	}
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	return token != "" && token == expected
}

func remoteFirstTelegramMessage(u *telegramUpdate) *telegramMessage {
	if u == nil {
		return nil
	}
	if u.Message != nil {
		return u.Message
	}
	if u.Edited != nil {
		return u.Edited
	}
	if u.ChannelPost != nil {
		return u.ChannelPost
	}
	return nil
}

func (m *telegramMessage) FromID() int64 {
	if m == nil || m.From == nil {
		return 0
	}
	return m.From.ID
}

func remoteTargetLabel(t remoteReplyTarget) string {
	if t.platform == "telegram" {
		if strings.TrimSpace(t.replyToID) != "" {
			return "telegram:" + t.chatID + "#" + t.replyToID
		}
		return "telegram:" + t.chatID
	}
	if t.channelID != "" {
		return "discord:" + t.channelID
	}
	return "discord"
}

func (s *Server) resolveRemoteThread(ctx context.Context, conf remoteConfig, sourceKey string) (int64, bool, error) {
	mappedID := int64(0)
	if sourceKey != "" {
		mappedID = conf.threadMap[sourceKey]
		if mappedID > 0 {
			if _, err := s.chat.GetThread(ctx, mappedID); err == nil {
				return mappedID, false, nil
			}
			delete(conf.threadMap, sourceKey)
		}
	}

	if conf.defaultThreadID > 0 {
		if _, err := s.chat.GetThread(ctx, conf.defaultThreadID); err == nil {
			if sourceKey != "" && conf.threadMap[sourceKey] != conf.defaultThreadID {
				conf.threadMap[sourceKey] = conf.defaultThreadID
				return conf.defaultThreadID, true, nil
			}
			return conf.defaultThreadID, false, nil
		}
	}

	threadTitle := "Remote chat"
	if sourceKey != "" {
		threadTitle = "Remote " + sourceKey
	}
	t, err := s.chat.CreateThread(ctx, threadTitle, nil)
	if err != nil {
		return 0, false, err
	}
	if sourceKey != "" {
		conf.threadMap[sourceKey] = t.ID
		return t.ID, true, nil
	}
	return t.ID, false, nil
}

func (s *Server) saveRemoteThreadMap(ctx context.Context, m map[string]int64) error {
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return upsertSetting(ctx, s.st.DB, remoteThreadMapKey, string(raw))
}

func remoteParseThreadMap(raw string) map[string]int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]int64{}
	}

	out := map[string]int64{}
	if err := json.Unmarshal([]byte(raw), &out); err == nil {
		return out
	}

	var fallback map[string]any
	if err := json.Unmarshal([]byte(raw), &fallback); err != nil {
		return out
	}
	for k, v := range fallback {
		switch t := v.(type) {
		case float64:
			if t > 0 {
				out[k] = int64(t)
			}
		case int64:
			if t > 0 {
				out[k] = t
			}
		case string:
			if id, e := strconv.ParseInt(strings.TrimSpace(t), 10, 64); e == nil && id > 0 {
				out[k] = id
			}
		}
	}
	return out
}

func remoteSourceKey(platform, locationID, senderID string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if strings.TrimSpace(locationID) == "" {
		if strings.TrimSpace(senderID) == "" {
			return platform
		}
		return platform + "::" + strings.TrimSpace(senderID)
	}
	if senderID == "" {
		return platform + ":" + strings.TrimSpace(locationID)
	}
	return platform + ":" + strings.TrimSpace(locationID) + ":" + strings.TrimSpace(senderID)
}

func remoteTrimmedText(values ...string) string {
	for _, raw := range values {
		text := strings.TrimSpace(raw)
		if text != "" {
			return text
		}
	}
	return ""
}

func remoteStringID(id any) string {
	switch v := id.(type) {
	case int64:
		if v <= 0 {
			return ""
		}
		return strconv.FormatInt(v, 10)
	case int:
		if v <= 0 {
			return ""
		}
		return strconv.Itoa(v)
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func parseRemoteBool(raw string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(strings.ToLower(raw)))
	if err != nil {
		return false
	}
	return b
}

func parseRemoteInt64(raw string) int64 {
	if v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil {
		return v
	}
	return 0
}

func parseRemoteBoolValue(v any) string {
	switch raw := v.(type) {
	case bool:
		return strconv.FormatBool(raw)
	case string:
		if parseRemoteBool(raw) {
			return "true"
		}
		return "false"
	case float64:
		if raw != 0 {
			return "true"
		}
		return "false"
	case int:
		if raw != 0 {
			return "true"
		}
		return "false"
	case int64:
		if raw != 0 {
			return "true"
		}
		return "false"
	case json.Number:
		if raw == "0" {
			return "false"
		}
		return "true"
	default:
		return "false"
	}
}

func (s *Server) sendTelegramReply(ctx context.Context, botToken, chatID, replyToID, text string) error {
	if strings.TrimSpace(botToken) == "" {
		return fmt.Errorf("telegram bot token not configured")
	}
	if strings.TrimSpace(chatID) == "" {
		return fmt.Errorf("telegram chat id missing")
	}

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	client := &http.Client{Timeout: remoteOutboundTimeout}
	payloadChunks := remoteSplitMessage(text, 3800)
	for i, part := range payloadChunks {
		payload := map[string]any{
			"chat_id": chatID,
			"text":    part,
		}
		if i == 0 && strings.TrimSpace(replyToID) != "" {
			if replyTo, err := strconv.ParseInt(replyToID, 10, 64); err == nil && replyTo > 0 {
				payload["reply_to_message_id"] = replyTo
			}
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		if res.StatusCode >= 300 {
			return fmt.Errorf("telegram api returned %s", res.Status)
		}
	}
	return nil
}

func (s *Server) sendDiscordReply(ctx context.Context, webhookURL, botToken, channelID, text string) error {
	if strings.TrimSpace(webhookURL) != "" {
		if err := s.sendDiscordWebhookReply(ctx, webhookURL, text); err == nil {
			return nil
		} else if strings.TrimSpace(botToken) == "" {
			return err
		}
	}

	if strings.TrimSpace(botToken) == "" {
		return fmt.Errorf("discord bot token not configured")
	}
	if strings.TrimSpace(channelID) == "" {
		return fmt.Errorf("discord channel id missing")
	}
	return s.sendDiscordBotReply(ctx, botToken, channelID, text)
}

func (s *Server) sendDiscordWebhookReply(ctx context.Context, webhookURL, text string) error {
	return s.sendDiscordPayload(ctx, "", webhookURL, text)
}

func (s *Server) sendDiscordBotReply(ctx context.Context, botToken, channelID, text string) error {
	endpoint := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", channelID)
	return s.sendDiscordPayload(ctx, botToken, endpoint, text)
}

func (s *Server) sendDiscordPayload(ctx context.Context, botToken, endpoint, text string) error {
	client := &http.Client{Timeout: remoteOutboundTimeout}
	for _, part := range remoteSplitMessage(text, 1900) {
		payload := map[string]any{"content": part}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if botToken != "" {
			req.Header.Set("Authorization", "Bot "+botToken)
		}
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		if res.StatusCode >= 300 {
			return fmt.Errorf("discord api returned %s", res.Status)
		}
	}
	return nil
}

func remoteSplitMessage(text string, limit int) []string {
	text = strings.TrimSpace(text)
	if limit <= 0 {
		return []string{text}
	}
	if len(text) <= limit {
		return []string{text}
	}

	runes := []rune(text)
	out := []string{}
	for len(runes) > 0 {
		end := limit
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, strings.TrimSpace(string(runes[:end])))
		runes = runes[end:]
	}
	return out
}
