package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type telegramMeResult struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

type telegramWebhookInfo struct {
	URL                string `json:"url"`
	HasCustomCert      bool   `json:"has_custom_certificate"`
	PendingUpdateCount int64  `json:"pending_update_count"`
	LastErrorDate      int64  `json:"last_error_date"`
	LastErrorMessage   string `json:"last_error_message"`
	MaxConnections     int64  `json:"max_connections"`
	IPAddress          string `json:"ip_address"`
}

type telegramAPIEnvelope[T any] struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Result      T      `json:"result"`
}

func (s *Server) handleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := strings.TrimSpace(loadSetting(s.st.DB, telegramBotTokenKey, ""))
	defaultChatID := strings.TrimSpace(loadSetting(s.st.DB, telegramDefaultChatIDKey, ""))
	remoteEnabled := parseRemoteBool(loadSetting(s.st.DB, remoteAccessEnabledKey, "false"))
	crossChat := parseRemoteBool(loadSetting(s.st.DB, remoteCrossChatContextKey, "false"))

	payload := map[string]any{
		"remoteAccessEnabled": remoteEnabled,
		"tokenConfigured":     token != "",
		"defaultChatId":       defaultChatID,
		"crossChatContext":    crossChat,
	}
	s.telegramMu.RLock()
	gateway := s.telegramGateway
	gatewayErr := strings.TrimSpace(s.telegramErr)
	s.telegramMu.RUnlock()
	if gateway != nil {
		payload["gateway"] = gateway.Status()
	} else if gatewayErr != "" {
		payload["gatewayReason"] = gatewayErr
	}
	if token == "" {
		payload["ready"] = false
		payload["reason"] = "telegram bot token is not configured"
		writeJSON(w, http.StatusOK, payload)
		return
	}

	me, meErr := telegramGetMe(ctx, token)
	if meErr != nil {
		payload["ready"] = false
		payload["reason"] = meErr.Error()
		writeJSON(w, http.StatusOK, payload)
		return
	}
	payload["bot"] = map[string]any{
		"id":        me.ID,
		"username":  me.Username,
		"firstName": me.FirstName,
	}

	if webhookInfo, err := telegramGetWebhookInfo(ctx, token); err == nil {
		payload["webhook"] = webhookInfo
	} else {
		payload["webhookError"] = err.Error()
	}

	payload["ready"] = true
	writeJSON(w, http.StatusOK, payload)
}

func telegramGetMe(ctx context.Context, token string) (telegramMeResult, error) {
	var out telegramAPIEnvelope[telegramMeResult]
	if err := telegramAPIRequest(ctx, token, "getMe", &out); err != nil {
		return telegramMeResult{}, err
	}
	if !out.OK {
		return telegramMeResult{}, fmt.Errorf("telegram getMe failed: %s", strings.TrimSpace(out.Description))
	}
	return out.Result, nil
}

func telegramGetWebhookInfo(ctx context.Context, token string) (telegramWebhookInfo, error) {
	var out telegramAPIEnvelope[telegramWebhookInfo]
	if err := telegramAPIRequest(ctx, token, "getWebhookInfo", &out); err != nil {
		return telegramWebhookInfo{}, err
	}
	if !out.OK {
		return telegramWebhookInfo{}, fmt.Errorf("telegram getWebhookInfo failed: %s", strings.TrimSpace(out.Description))
	}
	return out.Result, nil
}

func telegramAPIRequest(ctx context.Context, token string, method string, out any) error {
	token = strings.TrimSpace(token)
	method = strings.TrimSpace(method)
	if token == "" || method == "" {
		return fmt.Errorf("telegram api request missing token or method")
	}
	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", token, method)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return fmt.Errorf("telegram api returned %s", res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}
