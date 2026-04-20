package api

import (
	"os"
	"strconv"
	"strings"

	"forge/projectforge/services/core/internal/config"
)

const (
	telegramGatewayOffsetKey = "telegram_gateway_update_offset"
)

type telegramGatewayConfig struct {
	Enabled       bool
	BotToken      string
	PollTimeoutS  int
	IdleDelayMs   int
	ErrorDelayMs  int
	MaxBatchItems int

	EnableWakeCommands       bool
	EnableComputerWake       bool
	ComputerWakeMAC          string
	ComputerWakeHost         string
	ComputerWakePort         int
	AllowedWakeTelegramUsers map[string]struct{}
}

type telegramGatewayStatus struct {
	Enabled          bool   `json:"enabled"`
	Connected        bool   `json:"connected"`
	Mode             string `json:"mode"`
	StartedAtMs      int64  `json:"startedAtMs,omitempty"`
	LastInboundAtMs  int64  `json:"lastInboundAtMs,omitempty"`
	InboundCount     int64  `json:"inboundCount"`
	LastUpdateID     int64  `json:"lastUpdateId,omitempty"`
	LastError        string `json:"lastError,omitempty"`
	LastErrorAtMs    int64  `json:"lastErrorAtMs,omitempty"`
	RemoteEnabled    bool   `json:"remoteEnabled"`
	CrossChatContext bool   `json:"crossChatContext"`
	WakeCommands     bool   `json:"wakeCommands"`
	ComputerWake     bool   `json:"computerWake"`
	ComputerWakeMAC  bool   `json:"computerWakeMacConfigured"`
}

func defaultTelegramGatewayConfig(_ config.Config) telegramGatewayConfig {
	return telegramGatewayConfig{
		Enabled:                  parseEnvBoolWithDefault("FORGE_TELEGRAM_GATEWAY_ENABLED", true),
		BotToken:                 strings.TrimSpace(os.Getenv("FORGE_TELEGRAM_BOT_TOKEN")),
		PollTimeoutS:             parseEnvIntWithDefault("FORGE_TELEGRAM_POLL_TIMEOUT_SECONDS", 25),
		IdleDelayMs:              parseEnvIntWithDefault("FORGE_TELEGRAM_POLL_IDLE_DELAY_MS", 350),
		ErrorDelayMs:             parseEnvIntWithDefault("FORGE_TELEGRAM_POLL_ERROR_DELAY_MS", 2000),
		MaxBatchItems:            parseEnvIntWithDefault("FORGE_TELEGRAM_POLL_MAX_BATCH", 20),
		EnableWakeCommands:       parseEnvBoolWithDefault("FORGE_TELEGRAM_ENABLE_WAKE_COMMANDS", true),
		EnableComputerWake:       parseEnvBoolWithDefault("FORGE_TELEGRAM_ENABLE_COMPUTER_WAKE", false),
		ComputerWakeMAC:          strings.TrimSpace(os.Getenv("FORGE_TELEGRAM_WAKE_MAC")),
		ComputerWakeHost:         strings.TrimSpace(os.Getenv("FORGE_TELEGRAM_WAKE_HOST")),
		ComputerWakePort:         parseEnvIntWithDefault("FORGE_TELEGRAM_WAKE_PORT", 9),
		AllowedWakeTelegramUsers: parseCSVSet(os.Getenv("FORGE_TELEGRAM_WAKE_ALLOW_USER_IDS")),
	}
}

func parseEnvIntWithDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
