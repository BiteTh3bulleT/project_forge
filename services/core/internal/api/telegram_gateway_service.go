package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TelegramGateway struct {
	server *Server
	cfg    telegramGatewayConfig

	mu      sync.RWMutex
	status  telegramGatewayStatus
	client  *http.Client
	stop    context.CancelFunc
	stopped chan struct{}
}

type telegramPollEnvelope struct {
	OK          bool                 `json:"ok"`
	Description string               `json:"description"`
	Result      []telegramPollUpdate `json:"result"`
}

type telegramPollUpdate struct {
	UpdateID    int64            `json:"update_id"`
	Message     *telegramMessage `json:"message"`
	Edited      *telegramMessage `json:"edited_message"`
	ChannelPost *telegramMessage `json:"channel_post"`
}

func (u telegramPollUpdate) firstMessage() *telegramMessage {
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

func newTelegramGateway(s *Server, cfg telegramGatewayConfig) (*TelegramGateway, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if strings.TrimSpace(cfg.BotToken) == "" {
		return nil, fmt.Errorf("telegram gateway enabled but bot token is missing")
	}
	if cfg.PollTimeoutS <= 0 {
		cfg.PollTimeoutS = 25
	}
	if cfg.IdleDelayMs <= 0 {
		cfg.IdleDelayMs = 350
	}
	if cfg.ErrorDelayMs <= 0 {
		cfg.ErrorDelayMs = 2000
	}
	if cfg.MaxBatchItems <= 0 {
		cfg.MaxBatchItems = 20
	}
	gateway := &TelegramGateway{
		server: s,
		cfg:    cfg,
		client: &http.Client{Timeout: time.Duration(cfg.PollTimeoutS+15) * time.Second},
		status: telegramGatewayStatus{
			Enabled:         true,
			Mode:            "private_long_poll",
			RemoteEnabled:   parseRemoteBool(loadSetting(s.st.DB, remoteAccessEnabledKey, "false")),
			WakeCommands:    cfg.EnableWakeCommands,
			ComputerWake:    cfg.EnableComputerWake,
			ComputerWakeMAC: strings.TrimSpace(cfg.ComputerWakeMAC) != "",
		},
	}
	gateway.status.CrossChatContext = parseRemoteBool(loadSetting(s.st.DB, remoteCrossChatContextKey, "false"))
	return gateway, nil
}

func (g *TelegramGateway) Start(ctx context.Context) error {
	g.mu.Lock()
	if g.stop != nil {
		g.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(context.Background())
	g.stop = cancel
	g.stopped = make(chan struct{})
	g.status.Connected = true
	g.status.StartedAtMs = time.Now().UnixMilli()
	g.status.LastError = ""
	g.status.LastErrorAtMs = 0
	g.mu.Unlock()

	go func() {
		defer close(g.stopped)
		g.run(runCtx)
	}()

	go func() {
		<-ctx.Done()
		g.Stop()
	}()
	return nil
}

func (g *TelegramGateway) Stop() {
	g.mu.Lock()
	stop := g.stop
	stopped := g.stopped
	g.stop = nil
	g.stopped = nil
	g.status.Connected = false
	g.mu.Unlock()
	if stop != nil {
		stop()
	}
	if stopped != nil {
		<-stopped
	}
}

func (g *TelegramGateway) Status() telegramGatewayStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.status
}

func (g *TelegramGateway) run(ctx context.Context) {
	offset := g.loadOffset()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, err := g.fetchUpdates(ctx, offset)
		if err != nil {
			g.recordError(err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(g.cfg.ErrorDelayMs) * time.Millisecond):
			}
			continue
		}

		if len(updates) == 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(g.cfg.IdleDelayMs) * time.Millisecond):
			}
			continue
		}

		maxUpdateID := offset - 1
		for _, update := range updates {
			if update.UpdateID > maxUpdateID {
				maxUpdateID = update.UpdateID
			}
			msg := update.firstMessage()
			if msg == nil {
				continue
			}
			if err := g.processUpdateMessage(ctx, msg); err != nil {
				g.recordError(err)
			}
		}
		if maxUpdateID >= 0 {
			offset = maxUpdateID + 1
			g.saveOffset(offset)
			g.mu.Lock()
			g.status.LastUpdateID = maxUpdateID
			g.mu.Unlock()
		}
	}
}

func (g *TelegramGateway) processUpdateMessage(ctx context.Context, msg *telegramMessage) error {
	conf, err := g.server.loadRemoteConfigForMode(ctx, false)
	if err != nil {
		return err
	}
	if strings.TrimSpace(conf.telegramBotToken) == "" {
		return fmt.Errorf("telegram bot token not configured for private poll ingress")
	}
	if handled, err := g.processWakeCommandIfPresent(ctx, conf, msg); handled || err != nil {
		if err != nil {
			return err
		}
		g.mu.Lock()
		g.status.InboundCount++
		g.status.LastInboundAtMs = time.Now().UnixMilli()
		g.status.RemoteEnabled = conf.enabled
		g.status.CrossChatContext = conf.crossChatContext
		g.mu.Unlock()
		return nil
	}
	if err := g.server.processTelegramRemoteMessage(ctx, conf, msg); err != nil {
		return err
	}

	g.mu.Lock()
	g.status.InboundCount++
	g.status.LastInboundAtMs = time.Now().UnixMilli()
	g.status.RemoteEnabled = conf.enabled
	g.status.CrossChatContext = conf.crossChatContext
	g.mu.Unlock()
	return nil
}

func (g *TelegramGateway) processWakeCommandIfPresent(ctx context.Context, conf remoteConfig, msg *telegramMessage) (bool, error) {
	if !g.cfg.EnableWakeCommands || msg == nil {
		return false, nil
	}
	raw := strings.TrimSpace(remoteTrimmedText(msg.Text, msg.Caption))
	if raw == "" {
		return false, nil
	}
	cmd := normalizeWakeCommand(raw)
	if cmd == "" {
		return false, nil
	}

	chatID := remoteStringID(msg.Chat.ID)
	if chatID == "" {
		chatID = conf.telegramDefaultChat
	}
	if chatID == "" {
		return true, fmt.Errorf("telegram chat id missing for wake command")
	}
	replyToID := remoteStringID(msg.MessageID)
	userID := remoteStringID(msg.FromID())

	if !g.isWakeUserAllowed(userID) {
		return true, g.server.sendTelegramReply(ctx, conf.telegramBotToken, chatID, replyToID, "Wake command denied: user is not authorized.")
	}

	switch cmd {
	case "wake_forge":
		return true, g.replyWakeForge(ctx, conf, chatID, replyToID)
	case "wake_computer":
		return true, g.replyWakeComputer(ctx, conf, chatID, replyToID)
	default:
		return false, nil
	}
}

func (g *TelegramGateway) replyWakeForge(ctx context.Context, conf remoteConfig, chatID, replyToID string) error {
	st := g.Status()
	text := fmt.Sprintf(
		"FORGE wake check:\ncore=online\ngateway=%s\nmode=%s\ninbound=%d\nlast_inbound=%s",
		boolWord(st.Connected),
		st.Mode,
		st.InboundCount,
		formatUnixMillis(st.LastInboundAtMs),
	)
	return g.server.sendTelegramReply(ctx, conf.telegramBotToken, chatID, replyToID, text)
}

func (g *TelegramGateway) replyWakeComputer(ctx context.Context, conf remoteConfig, chatID, replyToID string) error {
	if !g.cfg.EnableComputerWake {
		return g.server.sendTelegramReply(ctx, conf.telegramBotToken, chatID, replyToID, "Computer wake is disabled. Set FORGE_TELEGRAM_ENABLE_COMPUTER_WAKE=true and FORGE_TELEGRAM_WAKE_MAC.")
	}
	if strings.TrimSpace(g.cfg.ComputerWakeMAC) == "" {
		return g.server.sendTelegramReply(ctx, conf.telegramBotToken, chatID, replyToID, "Computer wake is not configured: FORGE_TELEGRAM_WAKE_MAC is missing.")
	}
	host := strings.TrimSpace(g.cfg.ComputerWakeHost)
	if host == "" {
		host = "255.255.255.255"
	}
	if err := sendWakeOnLAN(g.cfg.ComputerWakeMAC, host, g.cfg.ComputerWakePort); err != nil {
		return g.server.sendTelegramReply(ctx, conf.telegramBotToken, chatID, replyToID, "Computer wake failed: "+err.Error())
	}
	return g.server.sendTelegramReply(ctx, conf.telegramBotToken, chatID, replyToID, fmt.Sprintf("Wake packet sent to %s:%d for MAC %s.", host, g.cfg.ComputerWakePort, g.cfg.ComputerWakeMAC))
}

func normalizeWakeCommand(raw string) string {
	text := strings.ToLower(strings.TrimSpace(raw))
	switch text {
	case "/wake", "/wake forge", "wake", "wake forge", "forge wake":
		return "wake_forge"
	case "/wake computer", "/wake pc", "wake computer", "wake pc", "wake machine":
		return "wake_computer"
	default:
		return ""
	}
}

func (g *TelegramGateway) isWakeUserAllowed(userID string) bool {
	if len(g.cfg.AllowedWakeTelegramUsers) == 0 {
		return true
	}
	_, ok := g.cfg.AllowedWakeTelegramUsers[strings.TrimSpace(userID)]
	return ok
}

func sendWakeOnLAN(macAddress, host string, port int) error {
	hw, err := parseMAC(macAddress)
	if err != nil {
		return err
	}
	packet := make([]byte, 6+16*len(hw))
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}
	for i := 6; i < len(packet); i += len(hw) {
		copy(packet[i:i+len(hw)], hw)
	}

	if port <= 0 {
		port = 9
	}
	addr := &net.UDPAddr{
		IP:   net.ParseIP(host),
		Port: port,
	}
	if addr.IP == nil {
		return fmt.Errorf("invalid wake host %q", host)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	if _, err := conn.Write(packet); err != nil {
		return err
	}
	return nil
}

func parseMAC(value string) ([]byte, error) {
	raw := strings.ToLower(strings.TrimSpace(value))
	raw = strings.ReplaceAll(raw, ":", "")
	raw = strings.ReplaceAll(raw, "-", "")
	raw = strings.ReplaceAll(raw, ".", "")
	if len(raw) != 12 {
		return nil, fmt.Errorf("invalid MAC address %q", value)
	}
	mac, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid MAC address %q", value)
	}
	return mac, nil
}

func boolWord(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func formatUnixMillis(v int64) string {
	if v <= 0 {
		return "never"
	}
	return time.UnixMilli(v).Format(time.RFC3339)
}

func (g *TelegramGateway) fetchUpdates(ctx context.Context, offset int64) ([]telegramPollUpdate, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.telegram.org",
		Path:   "/bot" + g.cfg.BotToken + "/getUpdates",
	}
	query := u.Query()
	query.Set("timeout", strconv.Itoa(g.cfg.PollTimeoutS))
	query.Set("limit", strconv.Itoa(g.cfg.MaxBatchItems))
	if offset > 0 {
		query.Set("offset", strconv.FormatInt(offset, 10))
	}
	query.Set("allowed_updates", `["message","edited_message","channel_post"]`)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("telegram getUpdates returned %s", res.Status)
	}
	var parsed telegramPollEnvelope
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		return nil, fmt.Errorf("telegram getUpdates failed: %s", strings.TrimSpace(parsed.Description))
	}
	return parsed.Result, nil
}

func (g *TelegramGateway) loadOffset() int64 {
	raw := strings.TrimSpace(loadSetting(g.server.st.DB, telegramGatewayOffsetKey, "0"))
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func (g *TelegramGateway) saveOffset(offset int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = upsertSetting(ctx, g.server.st.DB, telegramGatewayOffsetKey, strconv.FormatInt(offset, 10))
}

func (g *TelegramGateway) recordError(err error) {
	if err == nil {
		return
	}
	g.mu.Lock()
	g.status.LastError = err.Error()
	g.status.LastErrorAtMs = time.Now().UnixMilli()
	g.mu.Unlock()
	_ = g.server.log.Emit(context.Background(), "telegram.gateway.error", map[string]any{
		"error": err.Error(),
	})
}
