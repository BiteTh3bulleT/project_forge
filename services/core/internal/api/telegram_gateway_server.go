package api

import (
	"context"
	"strings"

	"forge/projectforge/services/core/internal/config"
)

func (s *Server) loadTelegramGatewayConfig(_ context.Context, cfg config.Config) telegramGatewayConfig {
	out := defaultTelegramGatewayConfig(cfg)

	if strings.TrimSpace(out.BotToken) == "" {
		out.BotToken = strings.TrimSpace(loadSetting(s.st.DB, telegramBotTokenKey, ""))
	}

	remoteEnabled := parseRemoteBool(loadSetting(s.st.DB, remoteAccessEnabledKey, "false"))
	if !remoteEnabled {
		out.Enabled = false
	}
	return out
}

func (s *Server) tryStartTelegramGateway(ctx context.Context, cfg config.Config) *TelegramGateway {
	tcfg := s.loadTelegramGatewayConfig(ctx, cfg)
	gateway, err := newTelegramGateway(s, tcfg)
	if err != nil {
		logLocal("telegram gateway initialization skipped: %v", err)
		s.telegramMu.Lock()
		s.telegramErr = err.Error()
		s.telegramMu.Unlock()
		return nil
	}
	if gateway == nil {
		s.telegramMu.Lock()
		s.telegramErr = "telegram gateway disabled by configuration"
		s.telegramMu.Unlock()
		return nil
	}
	if err := gateway.Start(ctx); err != nil {
		logLocal("telegram gateway startup failed: %v", err)
		s.telegramMu.Lock()
		s.telegramErr = err.Error()
		s.telegramMu.Unlock()
		return nil
	}
	s.telegramMu.Lock()
	s.telegramErr = ""
	s.telegramMu.Unlock()
	logLocal("telegram gateway started (mode=%s poll_timeout=%ds)", gateway.Status().Mode, tcfg.PollTimeoutS)
	return gateway
}

func (s *Server) reloadTelegramGateway(ctx context.Context) {
	s.telegramMu.Lock()
	current := s.telegramGateway
	s.telegramGateway = nil
	s.telegramMu.Unlock()

	if current != nil {
		current.Stop()
	}

	next := s.tryStartTelegramGateway(ctx, s.cfg)
	s.telegramMu.Lock()
	s.telegramGateway = next
	s.telegramMu.Unlock()
}
