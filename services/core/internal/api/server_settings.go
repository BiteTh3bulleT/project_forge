package api

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/forgekshadow"
	"forge/projectforge/services/core/internal/gpu"
	"forge/projectforge/services/core/internal/ingest"
)

const (
	redactedSettingSecret = "[redacted]"
	localOllamaBaseURL    = "http://127.0.0.1:11434"
)

func redactedSettingSecretValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return redactedSettingSecret
}

func shouldPersistSettingSecret(value string) bool {
	return strings.TrimSpace(value) != redactedSettingSecret
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	ext := loadSetting(s.st.DB, "extensions_csv", ingest.DefaultExtensionsCSV())
	theme := loadSetting(s.st.DB, "theme", "dark")
	ollamaBase := effectiveOllamaBaseURL(s.st.DB)
	ollamaModel := normalizeOllamaModel(loadSetting(s.st.DB, "ollama_model", ""))
	embeddingProvider := loadSetting(s.st.DB, "embedding_provider", "local_hash")
	embeddingModel := loadSetting(s.st.DB, "embedding_model", "")
	embeddingDims := loadSetting(s.st.DB, "embedding_dims", "128")
	embeddingTEIEndpoint := loadSetting(s.st.DB, "embedding_tei_endpoint", "")
	embeddingTEITimeoutMs := loadSetting(s.st.DB, "embedding_tei_timeout_ms", "30000")
	retrievalWeightKeyword := loadSetting(s.st.DB, "retrieval_weight_keyword", "0.45")
	retrievalWeightSemantic := loadSetting(s.st.DB, "retrieval_weight_semantic", "0.55")
	retrievalVSAMode := loadSetting(s.st.DB, "retrieval_vsa_mode", "off")
	retrievalVSADims := loadSetting(s.st.DB, "retrieval_vsa_dims", "128")
	retrievalVSASeed := loadSetting(s.st.DB, "retrieval_vsa_seed", "17")
	retrievalVSAWeightAssociative := loadSetting(s.st.DB, "retrieval_vsa_weight_associative", "0.06")
	retrievalVSAWeightRoleMatch := loadSetting(s.st.DB, "retrieval_vsa_weight_role_match", "0.04")
	retrievalVSAWeightRelational := loadSetting(s.st.DB, "retrieval_vsa_weight_relational", "0.03")
	retrievalVSAWeightFeedback := loadSetting(s.st.DB, "retrieval_vsa_weight_feedback", "0.03")
	retrievalVSAMaxAdditive := loadSetting(s.st.DB, "retrieval_vsa_max_additive", "0.12")
	chatPersonalityPrompt := loadSetting(s.st.DB, "chat_personality_prompt", defaultChatOperatorSystemPrompt())
	remoteAccessEnabled := parseRemoteBool(loadSetting(s.st.DB, remoteAccessEnabledKey, "false"))
	remoteAccessToken := strings.TrimSpace(loadSecretSetting(r.Context(), s.st.DB, s.cfg.DataDir, remoteAccessTokenKey, ""))
	remoteCrossChatContext := parseRemoteBool(loadSetting(s.st.DB, remoteCrossChatContextKey, "false"))
	telegramBotToken := strings.TrimSpace(loadSecretSetting(r.Context(), s.st.DB, s.cfg.DataDir, telegramBotTokenKey, ""))
	telegramDefaultChatID := strings.TrimSpace(loadSetting(s.st.DB, telegramDefaultChatIDKey, ""))
	discordBotToken := strings.TrimSpace(loadSecretSetting(r.Context(), s.st.DB, s.cfg.DataDir, discordBotTokenKey, ""))
	discordDefaultChannelID := strings.TrimSpace(loadSetting(s.st.DB, discordDefaultChannelIDKey, ""))
	discordWebhookURL := strings.TrimSpace(loadSecretSetting(r.Context(), s.st.DB, s.cfg.DataDir, discordWebhookURLKey, ""))
	discordCrossChatContext := parseRemoteBool(loadSetting(s.st.DB, discordGatewayCrossChatContextKey, "false"))
	remoteDefaultThreadID := strings.TrimSpace(loadSetting(s.st.DB, remoteDefaultThreadIDKey, ""))
	dreamModeEnabled := parseRemoteBool(loadSetting(s.st.DB, "dream_mode_enabled", "true"))
	dreamModeDefaultDryRun := parseRemoteBool(loadSetting(s.st.DB, "dream_mode_default_dry_run", "true"))
	dreamModeMode := strings.TrimSpace(loadSetting(s.st.DB, "dream_mode_mode", "microdream"))
	dreamModeWindowHours := loadSetting(s.st.DB, "dream_mode_window_hours", "6")
	dreamModeMaxCandidates := loadSetting(s.st.DB, "dream_mode_max_candidates", "8")
	dreamModeAllowLongTermPromotion := parseRemoteBool(loadSetting(s.st.DB, "dream_mode_allow_long_term_promotion", "false"))
	dreamModeRequireOperatorReviewForLongTerm := parseRemoteBool(loadSetting(s.st.DB, "dream_mode_require_operator_review_for_long_term", "true"))
	dreamModeAllowCommits := parseRemoteBool(loadSetting(s.st.DB, "dream_mode_allow_commits", "false"))
	runtimeControls := runtimeControlsFromSettings(s.st.DB, s.cfg)
	shadowMode := shadowModeFromSettings(s.st.DB, s.cfg)
	writeJSON(w, http.StatusOK, map[string]any{
		"extensionsCsv":                 ext,
		"theme":                         theme,
		"ollamaBaseUrl":                 ollamaBase,
		"ollamaModel":                   ollamaModel,
		"embeddingProvider":             embeddingProvider,
		"embeddingModel":                embeddingModel,
		"embeddingDims":                 embeddingDims,
		"embeddingTeiEndpoint":          embeddingTEIEndpoint,
		"embeddingTeiTimeoutMs":         embeddingTEITimeoutMs,
		"retrievalWeightKeyword":        retrievalWeightKeyword,
		"retrievalWeightSemantic":       retrievalWeightSemantic,
		"retrievalVSAMode":              retrievalVSAMode,
		"retrievalVSADims":              retrievalVSADims,
		"retrievalVSASeed":              retrievalVSASeed,
		"retrievalVSAWeightAssociative": retrievalVSAWeightAssociative,
		"retrievalVSAWeightRoleMatch":   retrievalVSAWeightRoleMatch,
		"retrievalVSAWeightRelational":  retrievalVSAWeightRelational,
		"retrievalVSAWeightFeedback":    retrievalVSAWeightFeedback,
		"retrievalVSAMaxAdditive":       retrievalVSAMaxAdditive,
		"retrievalVsaMode":              retrievalVSAMode,
		"retrievalVsaDims":              retrievalVSADims,
		"retrievalVsaSeed":              retrievalVSASeed,
		"retrievalVsaWeightAssociative": retrievalVSAWeightAssociative,
		"retrievalVsaWeightRoleMatch":   retrievalVSAWeightRoleMatch,
		"retrievalVsaWeightRelational":  retrievalVSAWeightRelational,
		"retrievalVsaWeightFeedback":    retrievalVSAWeightFeedback,
		"retrievalVsaMaxAdditive":       retrievalVSAMaxAdditive,
		"chatPersonalityPrompt":         chatPersonalityPrompt,
		"chatPromptDefault":             defaultChatOperatorSystemPrompt(),
		"remoteAccessEnabled":           remoteAccessEnabled,
		"remoteAccessToken":             redactedSettingSecretValue(remoteAccessToken),
		"remoteAccessTokenConfigured":   remoteAccessToken != "",
		"remoteCrossChatContext":        remoteCrossChatContext,
		"remoteDefaultThreadId":         remoteDefaultThreadID,
		"telegramBotToken":              redactedSettingSecretValue(telegramBotToken),
		"telegramBotTokenConfigured":    telegramBotToken != "",
		"telegramDefaultChatId":         telegramDefaultChatID,
		"discordBotToken":               redactedSettingSecretValue(discordBotToken),
		"discordBotTokenConfigured":     discordBotToken != "",
		"discordDefaultChannelId":       discordDefaultChannelID,
		"discordWebhookUrl":             redactedSettingSecretValue(discordWebhookURL),
		"discordWebhookUrlConfigured":   discordWebhookURL != "",
		"discordCrossChatContext":       discordCrossChatContext,
		"dreamMode": map[string]any{
			"enabled":                          dreamModeEnabled,
			"defaultDryRun":                    dreamModeDefaultDryRun,
			"mode":                             dreamModeMode,
			"windowHours":                      dreamModeWindowHours,
			"maxCandidates":                    dreamModeMaxCandidates,
			"allowLongTermPromotion":           dreamModeAllowLongTermPromotion,
			"requireOperatorReviewForLongTerm": dreamModeRequireOperatorReviewForLongTerm,
			"allowCommits":                     dreamModeAllowCommits,
		},
		"runtimeControls": runtimeControls,
		"shadowMode":      shadowMode,
	})
}

func (s *Server) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body map[string]any
	if err := decodeServerJSONBody(r, &body); err != nil {
		writeServerDecodeError(w, err)
		return
	}
	discordConfigChanged := false
	telegramConfigChanged := false
	if v, ok := body["extensionsCsv"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "extensions_csv", v); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
		s.ingest.SetExtensionsCSV(v)
	}
	if v, ok := body["theme"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "theme", v); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if v, ok := body["ollamaBaseUrl"].(string); ok {
		baseURL, err := validatePersistedOllamaBaseURL(v)
		if err != nil {
			writeAPIRequestError(w, http.StatusBadRequest, fmt.Errorf("invalid ollamaBaseUrl: %w", err))
			return
		}
		if err := upsertSetting(ctx, s.st.DB, "ollama_base_url", baseURL); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if v, ok := body["ollamaModel"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "ollama_model", normalizeOllamaModel(v)); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if v, ok := body["embeddingProvider"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "embedding_provider", v); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if v, ok := body["embeddingModel"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "embedding_model", v); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	switch v := body["embeddingDims"].(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "embedding_dims", v); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "embedding_dims", strconv.Itoa(int(v))); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if v, ok := body["embeddingTeiEndpoint"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "embedding_tei_endpoint", v); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if v, ok := body["embeddingTeiApiKey"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "embedding_tei_api_key", v); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	switch v := body["embeddingTeiTimeoutMs"].(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "embedding_tei_timeout_ms", v); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "embedding_tei_timeout_ms", strconv.Itoa(int(v))); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	switch v := body["retrievalWeightKeyword"].(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_weight_keyword", v); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_weight_keyword", strconv.FormatFloat(v, 'f', 4, 64)); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	switch v := body["retrievalWeightSemantic"].(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_weight_semantic", v); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_weight_semantic", strconv.FormatFloat(v, 'f', 4, 64)); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	modeRaw, hasMode := body["retrievalVSAMode"]
	if !hasMode {
		modeRaw = body["retrievalVsaMode"]
	}
	if v, ok := modeRaw.(string); ok {
		mode := strings.ToLower(strings.TrimSpace(v))
		if mode != "off" && mode != "shadow" && mode != "active" {
			mode = "off"
		}
		if err := upsertSetting(ctx, s.st.DB, "retrieval_vsa_mode", mode); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	dimsRaw, hasDims := body["retrievalVSADims"]
	if !hasDims {
		dimsRaw = body["retrievalVsaDims"]
	}
	switch v := dimsRaw.(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_vsa_dims", strings.TrimSpace(v)); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_vsa_dims", strconv.Itoa(int(v))); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	seedRaw, hasSeed := body["retrievalVSASeed"]
	if !hasSeed {
		seedRaw = body["retrievalVsaSeed"]
	}
	switch v := seedRaw.(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_vsa_seed", strings.TrimSpace(v)); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_vsa_seed", strconv.Itoa(int(v))); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	for _, item := range []struct {
		BodyKeys   []string
		SettingKey string
	}{
		{BodyKeys: []string{"retrievalVSAWeightAssociative", "retrievalVsaWeightAssociative"}, SettingKey: "retrieval_vsa_weight_associative"},
		{BodyKeys: []string{"retrievalVSAWeightRoleMatch", "retrievalVsaWeightRoleMatch"}, SettingKey: "retrieval_vsa_weight_role_match"},
		{BodyKeys: []string{"retrievalVSAWeightRelational", "retrievalVsaWeightRelational"}, SettingKey: "retrieval_vsa_weight_relational"},
		{BodyKeys: []string{"retrievalVSAWeightFeedback", "retrievalVsaWeightFeedback"}, SettingKey: "retrieval_vsa_weight_feedback"},
		{BodyKeys: []string{"retrievalVSAMaxAdditive", "retrievalVsaMaxAdditive"}, SettingKey: "retrieval_vsa_max_additive"},
	} {
		var (
			raw any
			ok  bool
		)
		for _, key := range item.BodyKeys {
			if raw, ok = body[key]; ok {
				break
			}
		}
		if ok {
			switch v := raw.(type) {
			case string:
				if err := upsertSetting(ctx, s.st.DB, item.SettingKey, strings.TrimSpace(v)); err != nil {
					writeAPIRequestError(w, http.StatusInternalServerError, err)
					return
				}
			case float64:
				if err := upsertSetting(ctx, s.st.DB, item.SettingKey, strconv.FormatFloat(v, 'f', 4, 64)); err != nil {
					writeAPIRequestError(w, http.StatusInternalServerError, err)
					return
				}
			}
		}
	}
	if v, ok := body["remoteAccessEnabled"]; ok {
		if err := upsertSetting(ctx, s.st.DB, remoteAccessEnabledKey, parseRemoteBoolValue(v)); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
		discordConfigChanged = true
		telegramConfigChanged = true
	}
	if v, ok := body["remoteAccessToken"].(string); ok {
		if shouldPersistSettingSecret(v) {
			if err := upsertSecretSetting(ctx, s.st.DB, s.cfg.DataDir, remoteAccessTokenKey, v); err != nil {
				writeAPIRequestError(w, http.StatusInternalServerError, err)
				return
			}
		}
	}
	if v, ok := body["remoteCrossChatContext"]; ok {
		if err := upsertSetting(ctx, s.st.DB, remoteCrossChatContextKey, parseRemoteBoolValue(v)); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if raw, ok := body["remoteDefaultThreadId"]; ok {
		if v := parseAnyInt64(raw); v > 0 {
			if err := upsertSetting(ctx, s.st.DB, remoteDefaultThreadIDKey, strconv.FormatInt(v, 10)); err != nil {
				writeAPIRequestError(w, http.StatusInternalServerError, err)
				return
			}
		} else if threadIDRaw, ok := raw.(string); ok && strings.TrimSpace(threadIDRaw) == "" {
			if err := upsertSetting(ctx, s.st.DB, remoteDefaultThreadIDKey, ""); err != nil {
				writeAPIRequestError(w, http.StatusInternalServerError, err)
				return
			}
		}
	}
	if v, ok := body["telegramBotToken"].(string); ok {
		if shouldPersistSettingSecret(v) {
			if err := upsertSecretSetting(ctx, s.st.DB, s.cfg.DataDir, telegramBotTokenKey, v); err != nil {
				writeAPIRequestError(w, http.StatusInternalServerError, err)
				return
			}
			telegramConfigChanged = true
		}
	}
	if v, ok := body["telegramDefaultChatId"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, telegramDefaultChatIDKey, strings.TrimSpace(v)); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if v, ok := body["discordBotToken"].(string); ok {
		if shouldPersistSettingSecret(v) {
			if err := upsertSecretSetting(ctx, s.st.DB, s.cfg.DataDir, discordBotTokenKey, v); err != nil {
				writeAPIRequestError(w, http.StatusInternalServerError, err)
				return
			}
			discordConfigChanged = true
		}
	}
	if v, ok := body["discordDefaultChannelId"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, discordDefaultChannelIDKey, strings.TrimSpace(v)); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
		discordConfigChanged = true
	}
	if v, ok := body["discordWebhookUrl"].(string); ok {
		if shouldPersistSettingSecret(v) {
			if err := upsertSecretSetting(ctx, s.st.DB, s.cfg.DataDir, discordWebhookURLKey, v); err != nil {
				writeAPIRequestError(w, http.StatusInternalServerError, err)
				return
			}
			discordConfigChanged = true
		}
	}
	if v, ok := body["discordCrossChatContext"]; ok {
		if err := upsertSetting(ctx, s.st.DB, discordGatewayCrossChatContextKey, parseRemoteBoolValue(v)); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
		discordConfigChanged = true
	}
	if v, ok := body["chatPersonalityPrompt"].(string); ok {
		next := strings.TrimSpace(v)
		if next == "" {
			next = defaultChatOperatorSystemPrompt()
		}
		if err := upsertSetting(ctx, s.st.DB, "chat_personality_prompt", next); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if rawDreamMode, ok := body["dreamMode"]; ok {
		dreamMode, ok := rawDreamMode.(map[string]any)
		if !ok {
			http.Error(w, "dreamMode must be an object", http.StatusBadRequest)
			return
		}
		for _, item := range []struct {
			bodyKey    string
			settingKey string
		}{
			{bodyKey: "enabled", settingKey: "dream_mode_enabled"},
			{bodyKey: "defaultDryRun", settingKey: "dream_mode_default_dry_run"},
			{bodyKey: "allowLongTermPromotion", settingKey: "dream_mode_allow_long_term_promotion"},
			{bodyKey: "requireOperatorReviewForLongTerm", settingKey: "dream_mode_require_operator_review_for_long_term"},
			{bodyKey: "allowCommits", settingKey: "dream_mode_allow_commits"},
		} {
			if v, ok := dreamMode[item.bodyKey]; ok {
				if err := upsertSetting(ctx, s.st.DB, item.settingKey, parseRemoteBoolValue(v)); err != nil {
					writeAPIRequestError(w, http.StatusInternalServerError, err)
					return
				}
			}
		}
		if v, ok := dreamMode["mode"].(string); ok {
			mode := strings.TrimSpace(v)
			switch mode {
			case "microdream", "nap", "deep_dream":
			default:
				mode = "microdream"
			}
			if err := upsertSetting(ctx, s.st.DB, "dream_mode_mode", mode); err != nil {
				writeAPIRequestError(w, http.StatusInternalServerError, err)
				return
			}
		}
		for _, item := range []struct {
			bodyKey    string
			settingKey string
		}{
			{bodyKey: "windowHours", settingKey: "dream_mode_window_hours"},
			{bodyKey: "maxCandidates", settingKey: "dream_mode_max_candidates"},
		} {
			if raw, ok := dreamMode[item.bodyKey]; ok {
				if value := parseAnyInt64(raw); value > 0 {
					if err := upsertSetting(ctx, s.st.DB, item.settingKey, strconv.FormatInt(value, 10)); err != nil {
						writeAPIRequestError(w, http.StatusInternalServerError, err)
						return
					}
				}
			}
		}
	}
	if err := s.patchShadowMode(ctx, body); err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	if discordConfigChanged {
		s.reloadDiscordGateway(ctx)
	}
	if telegramConfigChanged {
		s.reloadTelegramGateway(ctx)
	}
	if err := s.patchRuntimeControls(ctx, body); err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	_ = s.log.Emit(ctx, "command.executed", map[string]any{"command": "settings.patch"})
	s.handleGetSettings(w, r)
}

func (s *Server) handleGetOllamaModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	configuredBaseURL := strings.TrimSpace(effectiveOllamaBaseURL(s.st.DB))
	baseURLOverride := strings.TrimSpace(r.URL.Query().Get("baseUrl"))
	baseURL := baseURLOverride
	if baseURLOverride == "" {
		baseURL = configuredBaseURL
	} else if !sameOllamaModelsBaseURL(baseURLOverride, configuredBaseURL) {
		if err := validateOllamaModelsBaseURLOverride(ctx, baseURLOverride); err != nil {
			writeAPIRequestError(w, http.StatusBadRequest, fmt.Errorf("invalid baseUrl: %w", err))
			return
		}
	} else if configuredBaseURL != "" {
		baseURL = configuredBaseURL
	}
	if baseURL == "" {
		writeAPIRequestError(w, http.StatusBadRequest, fmt.Errorf("invalid baseUrl: must not be empty"))
		return
	}
	// Always allow best-effort discovery so settings UX can render even if Ollama is offline.
	resp := map[string]any{
		"baseUrl": baseURL,
	}

	ollamaAdapter, err := s.adapters.Get("ollama")
	if err != nil {
		resp["status"] = "unavailable"
		resp["error"] = err.Error()
		resp["models"] = []string{}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	ollama, ok := ollamaAdapter.(adapters.Ollama)
	if !ok {
		resp["status"] = "unavailable"
		resp["error"] = "ollama adapter has unexpected type"
		resp["models"] = []string{}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	models, err := ollama.FetchModels(ctx, baseURL, 1800*time.Millisecond)
	if err != nil {
		resp["status"] = "unavailable"
		resp["error"] = err.Error()
		resp["models"] = []string{}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	sort.Strings(models)
	resp["status"] = "ready"
	resp["models"] = models
	writeJSON(w, http.StatusOK, resp)
}

const maxOllamaModelsBaseURLOverrideLength = 2048

func validateOllamaModelsBaseURLOverride(ctx context.Context, raw string) error {
	baseURL := strings.TrimSpace(raw)
	if baseURL == "" {
		return fmt.Errorf("must not be empty")
	}
	if len(baseURL) > maxOllamaModelsBaseURLOverrideLength {
		return fmt.Errorf("is too long")
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return err
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	if parsed.User != nil {
		return fmt.Errorf("must not include userinfo")
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return fmt.Errorf("host is required")
	}
	if strings.Contains(host, "%") {
		return fmt.Errorf("host zone identifiers are not allowed")
	}
	if isLocalhostName(host) {
		return fmt.Errorf("localhost hosts are not allowed")
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		if !isAllowedOllamaModelsBaseURLAddr(addr) {
			return fmt.Errorf("host address is not allowed")
		}
		return nil
	}

	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("host resolved no addresses")
	}
	for _, addr := range addrs {
		if !isAllowedOllamaModelsBaseURLAddr(addr) {
			return fmt.Errorf("host resolves to an address that is not allowed")
		}
	}
	return nil
}

func isLocalhostName(host string) bool {
	normalized := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	return normalized == "localhost" || strings.HasSuffix(normalized, ".localhost")
}

func isAllowedOllamaModelsBaseURLAddr(addr netip.Addr) bool {
	return addr.IsValid() &&
		addr.IsGlobalUnicast() &&
		!addr.IsLoopback() &&
		!addr.IsPrivate() &&
		!addr.IsLinkLocalUnicast() &&
		!addr.IsMulticast() &&
		!addr.IsUnspecified()
}

func validatePersistedOllamaBaseURL(raw string) (string, error) {
	baseURL := strings.TrimSpace(raw)
	if baseURL == "" {
		return "", nil
	}
	if len(baseURL) > maxOllamaModelsBaseURLOverrideLength {
		return "", fmt.Errorf("is too long")
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("scheme must be http or https")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("must not include userinfo")
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	if strings.Contains(host, "%") {
		return "", fmt.Errorf("host zone identifiers are not allowed")
	}
	if isLocalhostName(host) || strings.EqualFold(strings.TrimSuffix(host, "."), "host.docker.internal") {
		return baseURL, nil
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		if addr.IsLoopback() {
			return baseURL, nil
		}
		return "", fmt.Errorf("host address must be loopback or host.docker.internal")
	}
	return "", fmt.Errorf("host must be localhost, loopback, or host.docker.internal")
}

func sameOllamaModelsBaseURL(a, b string) bool {
	left := strings.TrimRight(strings.TrimSpace(a), "/")
	right := strings.TrimRight(strings.TrimSpace(b), "/")
	if left == "" || right == "" {
		return false
	}
	return strings.EqualFold(left, right)
}

func defaultOllamaBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")); value != "" {
		return value
	}
	return localOllamaBaseURL
}

func effectiveOllamaBaseURL(db *sql.DB) string {
	envDefault := defaultOllamaBaseURL()
	stored := strings.TrimSpace(loadSetting(db, "ollama_base_url", ""))
	if stored == "" {
		return envDefault
	}
	if !sameOllamaModelsBaseURL(envDefault, localOllamaBaseURL) && sameOllamaModelsBaseURL(stored, localOllamaBaseURL) {
		return envDefault
	}
	return stored
}

func normalizeOllamaModel(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if s == "qwen-coder:30b" {
		return "qwen3-coder:30b"
	}
	return s
}

const (
	runtimeGPUEnabledKey              = "runtime_gpu_enabled"
	runtimeNVIDIADCGMEnabledKey       = "runtime_nvidia_dcgm_enabled"
	runtimeIntelLevelZeroEnabledKey   = "runtime_intel_level_zero_enabled"
	runtimeAllowOllamaCloudModelsKey  = "modelruntime_allow_ollama_cloud_models"
	shadowModeEnabledKey              = "forge_k_shadow_mode_enabled"
	shadowChatMetadataEnabledKey      = "forge_k_shadow_chat_metadata_enabled"
	shadowRetrievalMetadataEnabledKey = "forge_k_shadow_retrieval_metadata_enabled"
)

func runtimeConfigFromSettings(db *sql.DB, cfg config.Config) config.Config {
	cfg.GPUEnabled = parseRemoteBool(loadSetting(db, runtimeGPUEnabledKey, strconv.FormatBool(cfg.GPUEnabled)))
	cfg.NVIDIADCGMEnabled = parseRemoteBool(loadSetting(db, runtimeNVIDIADCGMEnabledKey, strconv.FormatBool(cfg.NVIDIADCGMEnabled)))
	cfg.IntelLevelZeroEnabled = parseRemoteBool(loadSetting(db, runtimeIntelLevelZeroEnabledKey, strconv.FormatBool(cfg.IntelLevelZeroEnabled)))
	cfg.ModelRuntimeAllowOllamaCloudModels = parseRemoteBool(loadSetting(db, runtimeAllowOllamaCloudModelsKey, strconv.FormatBool(cfg.ModelRuntimeAllowOllamaCloudModels)))
	cfg.ForgeKShadowModeEnabled = parseRemoteBool(loadSetting(db, shadowModeEnabledKey, strconv.FormatBool(cfg.ForgeKShadowModeEnabled)))
	cfg.ForgeKShadowChatMetadataEnabled = parseRemoteBool(loadSetting(db, shadowChatMetadataEnabledKey, strconv.FormatBool(cfg.ForgeKShadowChatMetadataEnabled)))
	cfg.ForgeKShadowRetrievalMetadataEnabled = parseRemoteBool(loadSetting(db, shadowRetrievalMetadataEnabledKey, strconv.FormatBool(cfg.ForgeKShadowRetrievalMetadataEnabled)))
	return cfg
}

func runtimeControlsFromSettings(db *sql.DB, cfg config.Config) map[string]any {
	effective := runtimeConfigFromSettings(db, cfg)
	return map[string]any{
		"gpuEnabled":              effective.GPUEnabled,
		"nvidiaDcgmEnabled":       effective.NVIDIADCGMEnabled,
		"intelLevelZeroEnabled":   effective.IntelLevelZeroEnabled,
		"allowOllamaCloudModels":  effective.ModelRuntimeAllowOllamaCloudModels,
		"safeModeForceCpuOnly":    effective.SafeModeForceCPUOnly,
		"effectiveGpuEnabled":     effective.GPUEnabled && !effective.SafeModeForceCPUOnly,
		"cloudModelsDefaultState": map[bool]string{true: "enabled", false: "disabled"}[effective.ModelRuntimeAllowOllamaCloudModels],
	}
}

func shadowModeFromSettings(db *sql.DB, cfg config.Config) map[string]any {
	effective := runtimeConfigFromSettings(db, cfg)
	return map[string]any{
		"enabled":                  effective.ForgeKShadowModeEnabled,
		"chatMetadataEnabled":      effective.ForgeKShadowChatMetadataEnabled,
		"retrievalMetadataEnabled": effective.ForgeKShadowRetrievalMetadataEnabled,
	}
}

func (s *Server) patchShadowMode(ctx context.Context, body map[string]any) error {
	raw, ok := body["shadowMode"]
	if !ok {
		return nil
	}
	shadowMode, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("shadowMode must be an object")
	}
	changed := false
	for _, item := range []struct {
		bodyKey    string
		settingKey string
	}{
		{bodyKey: "enabled", settingKey: shadowModeEnabledKey},
		{bodyKey: "chatMetadataEnabled", settingKey: shadowChatMetadataEnabledKey},
		{bodyKey: "retrievalMetadataEnabled", settingKey: shadowRetrievalMetadataEnabledKey},
	} {
		if v, exists := shadowMode[item.bodyKey]; exists {
			if err := upsertSetting(ctx, s.st.DB, item.settingKey, parseRemoteBoolValue(v)); err != nil {
				return err
			}
			changed = true
		}
	}
	if changed {
		s.reloadShadowMode(ctx)
	}
	return nil
}

func (s *Server) reloadShadowMode(ctx context.Context) {
	s.cfg = runtimeConfigFromSettings(s.st.DB, s.cfg)
	if s.cfg.ForgeKShadowModeEnabled {
		s.forgeKShadow = forgekshadow.NewObserver(forgekshadow.Config{
			Enabled:                      true,
			ChatMetadataEnabled:          s.cfg.ForgeKShadowChatMetadataEnabled,
			RetrievalMetadataEnabled:     s.cfg.ForgeKShadowRetrievalMetadataEnabled,
			AdvisoryEnabled:              s.cfg.ForgeKShadowAdvisoryEnabled,
			ControlLaneValidationEnabled: s.cfg.ForgeKShadowControlLaneValidationEnabled,
		})
	} else {
		s.forgeKShadow = nil
	}
	_ = s.log.Emit(ctx, "shadow.controls.reloaded", map[string]any{
		"enabled":                  s.cfg.ForgeKShadowModeEnabled,
		"chatMetadataEnabled":      s.cfg.ForgeKShadowChatMetadataEnabled,
		"retrievalMetadataEnabled": s.cfg.ForgeKShadowRetrievalMetadataEnabled,
		"advisoryEnabled":          s.cfg.ForgeKShadowAdvisoryEnabled,
	})
}

func (s *Server) patchRuntimeControls(ctx context.Context, body map[string]any) error {
	raw, ok := body["runtimeControls"]
	if !ok {
		return nil
	}
	controls, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("runtimeControls must be an object")
	}
	changed := false
	for _, item := range []struct {
		bodyKey    string
		settingKey string
	}{
		{bodyKey: "gpuEnabled", settingKey: runtimeGPUEnabledKey},
		{bodyKey: "nvidiaDcgmEnabled", settingKey: runtimeNVIDIADCGMEnabledKey},
		{bodyKey: "intelLevelZeroEnabled", settingKey: runtimeIntelLevelZeroEnabledKey},
		{bodyKey: "allowOllamaCloudModels", settingKey: runtimeAllowOllamaCloudModelsKey},
	} {
		if v, exists := controls[item.bodyKey]; exists {
			if err := upsertSetting(ctx, s.st.DB, item.settingKey, parseRemoteBoolValue(v)); err != nil {
				return err
			}
			changed = true
		}
	}
	if changed {
		s.reloadRuntimeControls(ctx)
	}
	return nil
}

func (s *Server) reloadRuntimeControls(ctx context.Context) {
	s.cfg = runtimeConfigFromSettings(s.st.DB, s.cfg)
	s.gpuTelemetry = gpu.New(gpu.Options{
		Enabled:                 s.cfg.NVIDIADCGMEnabled && !s.cfg.SafeModeForceCPUOnly,
		Endpoint:                s.cfg.NVIDIADCGMEndpoint,
		Timeout:                 time.Duration(s.cfg.NVIDIADCGMTimeoutMs) * time.Millisecond,
		MemoryPressureThreshold: s.cfg.GPUBackgroundMemoryPressureBlockThreshold,
	})
	s.intelTelemetry = gpu.NewIntel(gpu.IntelOptions{
		Enabled:     s.cfg.IntelLevelZeroEnabled && !s.cfg.SafeModeForceCPUOnly,
		ZEInfoPath:  s.cfg.IntelLevelZeroZEInfoPath,
		IntelGPUTop: s.cfg.IntelGPUTopPath,
		Timeout:     time.Duration(s.cfg.IntelGPUTelemetryTimeoutMs) * time.Millisecond,
	})
	s.modelRuntime = initModelRuntimeService(s.cfg, s.auditSvc, s.gpuTelemetry, s.intelTelemetry)
	_ = s.log.Emit(ctx, "runtime.controls.reloaded", map[string]any{
		"gpuEnabled":             s.cfg.GPUEnabled,
		"nvidiaDcgmEnabled":      s.cfg.NVIDIADCGMEnabled,
		"intelLevelZeroEnabled":  s.cfg.IntelLevelZeroEnabled,
		"allowOllamaCloudModels": s.cfg.ModelRuntimeAllowOllamaCloudModels,
	})
}

func loadSetting(db *sql.DB, key, def string) string {
	var v string
	err := db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil {
		return def
	}
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func upsertSetting(ctx context.Context, db *sql.DB, key, val string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, val,
	)
	return err
}
