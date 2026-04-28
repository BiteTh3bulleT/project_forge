package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/jobs"
)

const (
	chatTranscriptTurns               = 12
	chatAttachmentContextMaxMessages  = 6
	chatAttachmentContextMaxArtifacts = 8
	chatAttachmentExcerptRunes        = 800
)

const chatAssistantVisibilityGuard = `

Visibility boundary:
- Do not reveal hidden reasoning, scratchpad analysis, chain-of-thought, "Thinking Process" sections, draft options, or internal planning.
- Return only the final user-facing answer.
- If you need to think, do it privately and keep the response concise.
- Your visible name is FORGE. Never identify as Phi, ChatGPT, Claude, an Ollama model, or the underlying model family.
- Do not continue the transcript, invent USER/YOU turns, or append synthetic prompts.`

func defaultChatOperatorSystemPrompt() string {
	return `You are FORGE, a practical software/workflow assistant.
Your visible name is FORGE. If asked who you are or what your name is, answer as FORGE.

Tone:
- concise, direct, technically grounded
- candid about weaknesses and tradeoffs
- no roleplay, no fluff

Behavior:
- prioritize structural fixes over cosmetic tweaks
- prefer durable, testable solutions
- keep momentum and propose concrete next steps
- preserve operator control and explicit approvals

Response quality bar:
- identify what is strong, weak, and the main bottleneck
- distinguish temporary workaround vs durable fix
- be specific and actionable

Operational constraints:
- Ground responses in transcript facts only; do not invent IDs, outputs, or completed actions.
- Chat may provide analysis, plans, and code examples.
- Do not claim files/commands executed unless verified by tool results in this thread.
- For machine actions, route through governed jobs/tool gateway and report only real outcomes.
- For live/current information, use governed tools when available; if required details like location are missing, ask for that detail directly.
- Do not continue the user transcript or write fake USER/YOU turns.
- Be concise and operational.`
}

func (s *Server) chatOperatorSystemPrompt() string {
	override := strings.TrimSpace(loadSetting(s.st.DB, "chat_personality_prompt", ""))
	if override != "" {
		return override + chatAssistantVisibilityGuard
	}
	return defaultChatOperatorSystemPrompt() + chatAssistantVisibilityGuard
}

func (s *Server) buildChatPrompt(ctx context.Context, th *chat.ThreadDetail) string {
	transcript := s.chat.BuildTranscript(th.Messages, chatTranscriptTurns)
	sys := s.chatOperatorSystemPrompt()
	attachments := s.buildThreadAttachmentContext(ctx, th)
	if attachments != "" {
		return sys + "\n\n---\nTHREAD TITLE: " + th.Title + "\n---\n" + transcript + "\n\n---\nATTACHMENTS CONTEXT\n" + attachments
	}
	return sys + "\n\n---\nTHREAD TITLE: " + th.Title + "\n---\n" + transcript
}

func messageAttachmentIDs(metadata map[string]any) []int64 {
	if metadata == nil {
		return nil
	}
	raw, ok := metadata["attachments"]
	if !ok || raw == nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]int64, 0, len(items))
	for _, item := range items {
		rec, ok := item.(map[string]any)
		if !ok {
			continue
		}
		idRaw, ok := rec["artifactId"]
		if !ok {
			continue
		}
		switch v := idRaw.(type) {
		case float64:
			if v > 0 {
				out = append(out, int64(v))
			}
		case int64:
			if v > 0 {
				out = append(out, v)
			}
		case int:
			if v > 0 {
				out = append(out, int64(v))
			}
		case string:
			if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && n > 0 {
				out = append(out, n)
			}
		}
	}
	return out
}

func readRequestedModelID(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	for _, key := range []string{"requestedModelId", "modelId"} {
		raw, ok := metadata[key]
		if !ok || raw == nil {
			continue
		}
		if value, ok := raw.(string); ok {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func (s *Server) buildThreadAttachmentContext(ctx context.Context, th *chat.ThreadDetail) string {
	type attachmentRef struct {
		messageID   int64
		messageRole string
		artifactID  int64
	}
	refs := make([]attachmentRef, 0, chatAttachmentContextMaxArtifacts)
	seenArtifacts := map[int64]struct{}{}
	seenMessages := map[int64]struct{}{}
	for i := len(th.Messages) - 1; i >= 0 && len(refs) < chatAttachmentContextMaxArtifacts; i-- {
		m := th.Messages[i]
		ids := messageAttachmentIDs(m.Metadata)
		if len(ids) == 0 {
			continue
		}
		if _, seen := seenMessages[m.ID]; !seen && len(seenMessages) >= chatAttachmentContextMaxMessages {
			continue
		}
		seenMessages[m.ID] = struct{}{}
		for _, id := range ids {
			if id <= 0 {
				continue
			}
			if _, seen := seenArtifacts[id]; seen {
				continue
			}
			seenArtifacts[id] = struct{}{}
			refs = append(refs, attachmentRef{
				messageID:   m.ID,
				messageRole: strings.ToUpper(m.Role),
				artifactID:  id,
			})
			if len(refs) >= chatAttachmentContextMaxArtifacts {
				break
			}
		}
	}

	var b strings.Builder
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]
		art, err := s.artifacts.GetByID(ctx, ref.artifactID)
		if err != nil {
			continue
		}
		b.WriteString(fmt.Sprintf("Message #%d (%s):\n", ref.messageID, ref.messageRole))
		b.WriteString(fmt.Sprintf("- attachment %d: %s (%s)\n", art.ID, art.Title, art.MimeType))
		content, _, textual, err := s.artifacts.ReadArtifactText(ctx, art.ID)
		if err == nil && textual {
			runes := []rune(strings.TrimSpace(content))
			if len(runes) > chatAttachmentExcerptRunes {
				b.WriteString("  excerpt:\n" + string(runes[:chatAttachmentExcerptRunes]) + "\n  ...\n")
			} else if len(runes) > 0 {
				b.WriteString("  excerpt:\n" + string(runes) + "\n")
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func (s *Server) resolveAttachmentMetadata(ctx context.Context, threadID int64, ids []int64) ([]map[string]any, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	seen := map[int64]struct{}{}
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		art, err := s.artifacts.GetByID(ctx, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("attachment %d not found", id)
			}
			return nil, err
		}
		threadScoped := false
		if len(art.Metadata) > 0 {
			var meta map[string]any
			if json.Unmarshal(art.Metadata, &meta) == nil {
				if rawThread, ok := meta["threadId"]; ok {
					switch v := rawThread.(type) {
					case float64:
						threadScoped = int64(v) == threadID
					case int64:
						threadScoped = v == threadID
					case string:
						if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
							threadScoped = n == threadID
						}
					}
				}
			}
		}
		if !threadScoped {
			return nil, fmt.Errorf("attachment %d is not linked to this thread", id)
		}
		entry := map[string]any{
			"artifactId": id,
			"title":      art.Title,
			"mimeType":   art.MimeType,
			"fileName":   filepath.Base(art.FilePath),
		}
		if content, _, textual, err := s.artifacts.ReadArtifactText(ctx, id); err == nil && textual {
			runes := []rune(strings.TrimSpace(content))
			if len(runes) > 1200 {
				entry["textPreview"] = string(runes[:1200]) + "\n…"
			} else if len(runes) > 0 {
				entry["textPreview"] = string(runes)
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

// handleChatMessagePost saves the user message, then either completes assistant work synchronously,
// schedules async completion, or tells the client to open an SSE stream (see handleChatAssistantStream).
func (s *Server) handleChatMessagePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	threadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad thread id", http.StatusBadRequest)
		return
	}
	var body struct {
		Content               string  `json:"content"`
		ModelID               string  `json:"modelId"`
		RequestAssistant      bool    `json:"requestAssistant"`
		AssistantDryRun       bool    `json:"assistantDryRun"`
		Stream                bool    `json:"stream"`
		AsyncAssistant        *bool   `json:"asyncAssistant"`
		SyncAssistant         bool    `json:"syncAssistant"` // when true, block until assistant finishes (legacy)
		AttachmentArtifactIDs []int64 `json:"attachmentArtifactIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	body.Content = strings.TrimSpace(body.Content)
	body.ModelID = strings.TrimSpace(body.ModelID)
	if body.Content == "" {
		http.Error(w, "content required", http.StatusBadRequest)
		return
	}
	userMeta := map[string]any{"source": "operator"}
	if body.ModelID != "" {
		userMeta["requestedModelId"] = body.ModelID
	}
	if len(body.AttachmentArtifactIDs) > 0 {
		attachments, err := s.resolveAttachmentMetadata(ctx, threadID, body.AttachmentArtifactIDs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(attachments) > 0 {
			userMeta["attachments"] = attachments
		}
	}
	um, err := s.chat.AppendMessage(ctx, threadID, "user", body.Content, userMeta)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = s.log.Emit(ctx, "chat.message.user", map[string]any{"threadId": threadID, "messageId": um.ID})

	out := map[string]any{
		"userMessage":      um,
		"assistantMessage": nil,
		"assistantPending": false,
		"userMessageId":    um.ID,
		"stream":           false,
		"asyncAssistant":   false,
	}

	if decision, ok := parseChatApprovalDirective(body.Content); ok {
		am := s.handleChatApprovalDirective(ctx, threadID, um.ID, decision)
		out["assistantMessage"] = am
		writeJSON(w, http.StatusOK, out)
		return
	}

	if cmd, parseErr := parseChatGatewayCommand(body.Content); parseErr != nil {
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", "Tool command parse failed: "+parseErr.Error(), map[string]any{
			"command":              "tool",
			"commandError":         true,
			"replyToUserMessageId": um.ID,
		})
		out["assistantMessage"] = am
		writeJSON(w, http.StatusOK, out)
		return
	} else if cmd != nil {
		if s.jobs == nil {
			am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", "Tool gateway is not available in this runtime.", map[string]any{
				"command":              "tool",
				"replyToUserMessageId": um.ID,
			})
			out["assistantMessage"] = am
			writeJSON(w, http.StatusOK, out)
			return
		}
		corr := strings.TrimSpace(cmd.CorrelationID)
		if corr == "" {
			corr = fmt.Sprintf("chat-%d-%d", threadID, um.ID)
		}
		payload := map[string]any{
			"toolId":         cmd.ToolID,
			"laneId":         nonEmpty(cmd.LaneID, cmd.ToolID),
			"domain":         cmd.Domain,
			"action":         nonEmpty(cmd.Action, "invoke"),
			"riskClass":      cmd.RiskClass,
			"executionLevel": cmd.ExecutionLevel,
			"correlationId":  corr,
			"paths":          cmd.Paths,
			"input":          nonNilMap(cmd.Input),
			"dryRun":         cmd.DryRun,
		}
		j, err := s.jobs.Create(ctx, jobs.CreateRequest{
			TemplateID:       "gateway_action",
			Title:            fmt.Sprintf("Chat tool action: %s", cmd.ToolID),
			UserRequest:      body.Content,
			Objective:        fmt.Sprintf("Run governed gateway action %s", cmd.ToolID),
			InitiatingSource: "chat",
			Query:            cmd.ToolID,
			Scope:            jobs.ScopeInput{SelectedPaths: cmd.Paths},
			RequestPayload:   payload,
			ExpectedOutput: map[string]any{
				"deliverableType": "gateway_result",
			},
		})
		if err != nil {
			am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", "Unable to queue tool action job: "+err.Error(), map[string]any{
				"command":              "tool",
				"commandError":         true,
				"replyToUserMessageId": um.ID,
			})
			out["assistantMessage"] = am
			writeJSON(w, http.StatusOK, out)
			return
		}
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", fmt.Sprintf("Queued governed tool action `%s` as job `%s` via lane `%s`.", cmd.ToolID, j.ID, nonEmpty(cmd.LaneID, cmd.ToolID)), map[string]any{
			"command":              "tool",
			"jobId":                j.ID,
			"toolId":               cmd.ToolID,
			"laneId":               nonEmpty(cmd.LaneID, cmd.ToolID),
			"correlationId":        corr,
			"replyToUserMessageId": um.ID,
		})
		out["assistantMessage"] = am
		writeJSON(w, http.StatusOK, out)
		return
	}

	if !body.RequestAssistant {
		writeJSON(w, http.StatusOK, out)
		return
	}

	th, err := s.chat.GetThread(ctx, threadID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if am, handled := s.maybeRespondHyperlaneNoModel(ctx, threadID, um.ID, um.Content); handled {
		out["assistantMessage"] = am
		writeJSON(w, http.StatusOK, out)
		return
	}

	var ollamaAdapter adapters.Adapter
	if adapter, getErr := s.adapters.Get("ollama"); getErr == nil {
		ollamaAdapter = adapter
	}

	// Dry-run is always synchronous and fast.
	if body.AssistantDryRun {
		am := s.completeAssistantSync(ctx, threadID, um.ID, th, um.Content, ollamaAdapter, body.AssistantDryRun, body.ModelID)
		out["assistantMessage"] = am
		writeJSON(w, http.StatusOK, out)
		return
	}

	async := true
	if body.AsyncAssistant != nil {
		async = *body.AsyncAssistant
	}
	if body.SyncAssistant {
		async = false
	}

	if body.Stream && !body.SyncAssistant {
		out["assistantPending"] = true
		out["stream"] = true
		writeJSON(w, http.StatusOK, out)
		return
	}

	if async {
		key := chatInflightKey(threadID, um.ID)
		if _, loaded := s.chatAssistInflight.LoadOrStore(key, true); loaded {
			http.Error(w, "assistant generation already in progress for this message", http.StatusConflict)
			return
		}
		go s.runChatAssistantAsync(key, threadID, um.ID, th, um.Content, ollamaAdapter, body.ModelID)
		out["assistantPending"] = true
		out["asyncAssistant"] = true
		writeJSON(w, http.StatusOK, out)
		return
	}

	am := s.completeAssistantSync(ctx, threadID, um.ID, th, um.Content, ollamaAdapter, false, body.ModelID)
	out["assistantMessage"] = am
	writeJSON(w, http.StatusOK, out)
}

type chatGatewayCommand struct {
	ToolID         string         `json:"toolId"`
	LaneID         string         `json:"laneId"`
	Domain         string         `json:"domain"`
	Action         string         `json:"action"`
	RiskClass      string         `json:"riskClass"`
	ExecutionLevel string         `json:"executionLevel"`
	CorrelationID  string         `json:"correlationId"`
	Paths          []string       `json:"paths"`
	Input          map[string]any `json:"input"`
	DryRun         bool           `json:"dryRun"`
}

func parseChatGatewayCommand(content string) (*chatGatewayCommand, error) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "/tool") {
		return nil, nil
	}
	raw := strings.TrimSpace(strings.TrimPrefix(trimmed, "/tool"))
	if raw == "" {
		return nil, fmt.Errorf("expected JSON payload after /tool")
	}
	var cmd chatGatewayCommand
	if err := json.Unmarshal([]byte(raw), &cmd); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}
	cmd.ToolID = strings.TrimSpace(cmd.ToolID)
	cmd.LaneID = strings.TrimSpace(cmd.LaneID)
	cmd.Action = strings.TrimSpace(cmd.Action)
	cmd.Domain = strings.TrimSpace(cmd.Domain)
	cmd.RiskClass = strings.TrimSpace(cmd.RiskClass)
	cmd.ExecutionLevel = strings.TrimSpace(cmd.ExecutionLevel)
	cmd.CorrelationID = strings.TrimSpace(cmd.CorrelationID)
	if cmd.ToolID == "" {
		return nil, fmt.Errorf("toolId is required")
	}
	if len(cmd.Paths) > 0 {
		out := make([]string, 0, len(cmd.Paths))
		for _, p := range cmd.Paths {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		cmd.Paths = out
	}
	if cmd.Input == nil {
		cmd.Input = map[string]any{}
	}
	return &cmd, nil
}

func nonNilMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func parseChatApprovalDirective(content string) (decision string, ok bool) {
	normalized := strings.ToLower(strings.TrimSpace(content))
	normalized = strings.Trim(normalized, ".!?")
	switch normalized {
	case "approve", "approved", "yes", "yes approve", "approve it", "approve now", "go ahead", "run it", "please approve", "approve this", "i approve":
		return "approved", true
	case "deny", "denied", "reject", "rejected", "cancel", "cancel approval", "do not run", "no", "deny it", "cancel it":
		return "denied", true
	default:
		return "", false
	}
}

func (s *Server) handleChatApprovalDirective(ctx context.Context, threadID, replyToUserMessageID int64, decision string) *chat.Message {
	th, err := s.chat.GetThread(ctx, threadID)
	if err != nil {
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", "Could not inspect pending approvals for this thread.", map[string]any{
			"command":              "approval",
			"commandError":         true,
			"replyToUserMessageId": replyToUserMessageID,
		})
		return am
	}
	requestID, jobID := latestGatewayApprovalReferenceForReply(th, replyToUserMessageID)
	if requestID == 0 {
		requestID, jobID = s.latestPendingGatewayApprovalReferenceForMessages(ctx, th, replyToUserMessageID)
	}
	if requestID == 0 {
		requestID, jobID = latestGatewayApprovalReference(th)
	}
	if requestID == 0 {
		requestID, jobID = s.latestPendingGatewayApprovalReference(ctx, th)
	}
	if requestID <= 0 {
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", "No pending gateway approval request was found in this thread. Use Approve/Deny on the latest Tool Gateway card or open `#/approvals`.", map[string]any{
			"command":              "approval",
			"decision":             decision,
			"replyToUserMessageId": replyToUserMessageID,
		})
		return am
	}
	req, err := s.approvals.GetRequest(ctx, requestID)
	if err != nil {
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", fmt.Sprintf("Could not load approval request #%d: %v", requestID, err), map[string]any{
			"command":              "approval",
			"commandError":         true,
			"approvalRequestId":    requestID,
			"replyToUserMessageId": replyToUserMessageID,
		})
		return am
	}
	if req.Status != "pending" {
		altID, altJobID := s.latestPendingGatewayApprovalReferenceForMessages(ctx, th, replyToUserMessageID)
		if altID > 0 && altID != requestID {
			requestID = altID
			jobID = altJobID
			req, err = s.approvals.GetRequest(ctx, requestID)
			if err != nil {
				am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", fmt.Sprintf("Could not load fallback approval request #%d: %v", requestID, err), map[string]any{
					"command":              "approval",
					"commandError":         true,
					"approvalRequestId":    requestID,
					"replyToUserMessageId": replyToUserMessageID,
				})
				return am
			}
		}
	}
	if req.Status != "pending" {
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", fmt.Sprintf("Approval request #%d is %s already. No pending gateway approval was found to apply this decision.", requestID, req.Status), map[string]any{
			"command":              "approval",
			"decision":             decision,
			"approvalRequestId":    requestID,
			"replyToUserMessageId": replyToUserMessageID,
		})
		return am
	}
	dec, err := s.jobs.ApplyApprovalDecision(ctx, requestID, decision, "operator", "Decision from chat directive")
	if err != nil {
		am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", fmt.Sprintf("Approval decision failed for request #%d: %v", requestID, err), map[string]any{
			"command":              "approval",
			"commandError":         true,
			"decision":             decision,
			"approvalRequestId":    requestID,
			"replyToUserMessageId": replyToUserMessageID,
		})
		return am
	}
	verb := "approved"
	if decision == "denied" {
		verb = "denied"
	}
	msg := fmt.Sprintf("Approval request #%d %s.", requestID, verb)
	if strings.TrimSpace(jobID) != "" {
		if decision == "approved" {
			msg += fmt.Sprintf(" Job `%s` has been re-queued.", jobID)
		} else {
			msg += fmt.Sprintf(" Job `%s` was cancelled by decision.", jobID)
		}
	}
	am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", msg, map[string]any{
		"command":              "approval",
		"decision":             decision,
		"approvalRequestId":    requestID,
		"approvalDecisionId":   dec.ID,
		"jobId":                jobID,
		"replyToUserMessageId": replyToUserMessageID,
	})
	return am
}

func latestGatewayApprovalReference(th *chat.ThreadDetail) (requestID int64, jobID string) {
	if th == nil {
		return 0, ""
	}
	for i := len(th.Messages) - 1; i >= 0; i-- {
		msg := th.Messages[i]
		if !strings.EqualFold(strings.TrimSpace(msg.Role), "assistant") || msg.Metadata == nil {
			continue
		}
		rawActivity, ok := msg.Metadata["toolGatewayActivity"]
		if !ok {
			continue
		}
		activity, ok := rawActivity.(map[string]any)
		if !ok || activity == nil {
			continue
		}
		if state, _ := activity["executionState"].(string); strings.TrimSpace(state) != "needs_approval" {
			continue
		}
		id := findApprovalRequestID(activity["executionResult"])
		if id <= 0 {
			id = findApprovalRequestID(activity)
		}
		if id <= 0 {
			continue
		}
		return id, findJobID(activity["executionResult"])
	}
	return 0, ""
}

func latestGatewayApprovalReferenceForReply(th *chat.ThreadDetail, replyToUserMessageID int64) (requestID int64, jobID string) {
	if th == nil {
		return 0, ""
	}
	beforeIdx := len(th.Messages)
	if replyToUserMessageID > 0 {
		for i := len(th.Messages) - 1; i >= 0; i-- {
			if th.Messages[i].ID == replyToUserMessageID {
				beforeIdx = i
				break
			}
		}
	}
	for i := beforeIdx - 1; i >= 0; i-- {
		msg := th.Messages[i]
		if !strings.EqualFold(strings.TrimSpace(msg.Role), "assistant") || msg.Metadata == nil {
			continue
		}
		rawActivity, ok := msg.Metadata["toolGatewayActivity"]
		if !ok {
			continue
		}
		activity, ok := rawActivity.(map[string]any)
		if !ok || activity == nil {
			continue
		}
		state := strings.TrimSpace(asString(activity["executionState"]))
		if state != "needs_approval" {
			continue
		}
		id := findApprovalRequestID(activity)
		if id <= 0 {
			continue
		}
		return id, firstJobIDFromApprovalActivity(activity)
	}
	return 0, ""
}

func firstJobIDFromApprovalActivity(v any) string {
	if id := findJobID(v); strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	return ""
}

func (s *Server) latestPendingGatewayApprovalReferenceForMessages(ctx context.Context, th *chat.ThreadDetail, replyToUserMessageID int64) (requestID int64, jobID string) {
	if th == nil {
		return 0, ""
	}
	beforeIdx := len(th.Messages)
	if replyToUserMessageID > 0 {
		for i := len(th.Messages) - 1; i >= 0; i-- {
			if th.Messages[i].ID == replyToUserMessageID {
				beforeIdx = i
				break
			}
		}
	}
	for i := beforeIdx - 1; i >= 0; i-- {
		msg := th.Messages[i]
		if !strings.EqualFold(strings.TrimSpace(msg.Role), "assistant") || msg.Metadata == nil {
			continue
		}
		rawActivity, ok := msg.Metadata["toolGatewayActivity"]
		if !ok {
			continue
		}
		activity, ok := rawActivity.(map[string]any)
		if !ok || activity == nil {
			continue
		}
		id := findApprovalRequestID(activity)
		if id <= 0 {
			continue
		}
		req, err := s.approvals.GetRequest(ctx, id)
		if err != nil || req == nil {
			continue
		}
		if req.Status == "pending" {
			return id, firstJobIDFromApprovalActivity(activity)
		}
	}
	return 0, ""
}

func (s *Server) latestPendingGatewayApprovalReference(ctx context.Context, th *chat.ThreadDetail) (requestID int64, jobID string) {
	if th == nil {
		return 0, ""
	}
	seen := map[int64]struct{}{}
	for i := len(th.Messages) - 1; i >= 0; i-- {
		msg := th.Messages[i]
		if !strings.EqualFold(strings.TrimSpace(msg.Role), "assistant") || msg.Metadata == nil {
			continue
		}
		rawActivity, ok := msg.Metadata["toolGatewayActivity"]
		if !ok {
			continue
		}
		activity, ok := rawActivity.(map[string]any)
		if !ok || activity == nil {
			continue
		}
		id := findApprovalRequestID(activity["executionResult"])
		if id <= 0 {
			id = findApprovalRequestID(activity)
		}
		if id <= 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		req, err := s.approvals.GetRequest(ctx, id)
		if err != nil || req == nil {
			continue
		}
		if req.Status == "pending" {
			return id, findJobID(activity["executionResult"])
		}
	}
	return 0, ""
}

func findApprovalRequestID(v any) int64 {
	switch x := v.(type) {
	case map[string]any:
		if raw, ok := x["approvalRequestId"]; ok {
			if id := parseAnyInt64(raw); id > 0 {
				return id
			}
		}
		for _, child := range x {
			if id := findApprovalRequestID(child); id > 0 {
				return id
			}
		}
	case []any:
		for _, child := range x {
			if id := findApprovalRequestID(child); id > 0 {
				return id
			}
		}
	}
	return 0
}

func findJobID(v any) string {
	switch x := v.(type) {
	case map[string]any:
		if raw, ok := x["jobId"]; ok {
			if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
		for _, child := range x {
			if id := findJobID(child); strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id)
			}
		}
	case []any:
		for _, child := range x {
			if id := findJobID(child); strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id)
			}
		}
	}
	return ""
}

func parseAnyInt64(v any) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	case json.Number:
		n, err := x.Int64()
		if err == nil {
			return n
		}
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err == nil {
			return n
		}
	}
	return 0
}

func chatInflightKey(threadID, userMessageID int64) string {
	return fmt.Sprintf("%d-%d", threadID, userMessageID)
}

func chatOllamaStreamCapable(ctx context.Context, ollamaAdapter adapters.Adapter) bool {
	ol, ok := ollamaAdapter.(adapters.Ollama)
	if !ok {
		return false
	}
	return strings.TrimSpace(ol.ModelForChat(ctx)) != ""
}

func (s *Server) completeAssistantSync(
	ctx context.Context,
	threadID, userMessageID int64,
	th *chat.ThreadDetail,
	lastUserContent string,
	ollamaAdapter adapters.Adapter,
	dryRun bool,
	requestedModelID string,
) *chat.Message {
	return s.completeAssistantWithGatewayTools(ctx, threadID, userMessageID, th, lastUserContent, ollamaAdapter, dryRun, nil, requestedModelID)
}

func (s *Server) runChatAssistantAsync(
	key string,
	threadID, userMessageID int64,
	th *chat.ThreadDetail,
	lastUserContent string,
	ollamaAdapter adapters.Adapter,
	requestedModelID string,
) {
	defer s.chatAssistInflight.Delete(key)
	ctx, cancel := context.WithTimeout(context.Background(), 185*time.Second)
	defer cancel()
	th2, err := s.chat.GetThread(ctx, threadID)
	if err != nil {
		_, _ = s.chat.AppendMessage(ctx, threadID, "assistant", "Could not load thread for assistant reply.", map[string]any{"failure": true, "replyToUserMessageId": userMessageID})
		return
	}
	am := s.completeAssistantSync(ctx, threadID, userMessageID, th2, lastUserContent, ollamaAdapter, false, requestedModelID)
	if am == nil {
		_, _ = s.chat.AppendMessage(ctx, threadID, "assistant", "Assistant reply could not be saved.", map[string]any{"failure": true, "replyToUserMessageId": userMessageID})
	}
}

// handleChatAssistantStream streams when the Ollama streaming path is available, then persists the assistant message.
func (s *Server) handleChatAssistantStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	threadID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad thread id", http.StatusBadRequest)
		return
	}
	userMessageID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("userMessageId")), 10, 64)
	if err != nil || userMessageID <= 0 {
		http.Error(w, "userMessageId required", http.StatusBadRequest)
		return
	}

	if existing, err := s.chat.FindAssistantReplyTo(ctx, threadID, userMessageID); err == nil && existing != nil {
		s.writeSSEAssistantDone(w, existing)
		return
	}

	key := chatInflightKey(threadID, userMessageID)
	if _, loaded := s.chatAssistInflight.LoadOrStore(key, true); loaded {
		http.Error(w, "assistant generation already in progress", http.StatusConflict)
		return
	}
	defer s.chatAssistInflight.Delete(key)

	th, err := s.chat.GetThread(ctx, threadID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	found := false
	for _, m := range th.Messages {
		if m.ID == userMessageID && m.Role == "user" {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "user message not in thread", http.StatusBadRequest)
		return
	}

	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	var umContent string
	var requestedModelID string
	for _, m := range th.Messages {
		if m.ID == userMessageID && m.Role == "user" {
			umContent = m.Content
			requestedModelID = readRequestedModelID(m.Metadata)
			break
		}
	}

	var ollamaAdapter adapters.Adapter
	if adapter, getErr := s.adapters.Get("ollama"); getErr == nil {
		ollamaAdapter = adapter
	}
	if am, handled := s.maybeRespondHyperlaneNoModel(ctx, threadID, userMessageID, umContent); handled {
		s.initSSE(w)
		if am == nil {
			b, _ := json.Marshal(map[string]any{"message": "assistant reply could not be saved"})
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(b))
			w.(http.Flusher).Flush()
			return
		}
		s.writeSSEEvent(w, map[string]any{"assistantMessage": am})
		return
	}
	if !chatOllamaStreamCapable(ctx, ollamaAdapter) {
		s.initSSE(w)
		emitStage := func(stage string, data map[string]any) {
			row := map[string]any{"stage": stage, "atMs": time.Now().UnixMilli()}
			for k, v := range data {
				row[k] = v
			}
			s.writeNamedSSEEvent(w, "agent_stage", row)
		}
		emitStage("request_received", map[string]any{"userChars": len(umContent)})
		emitStage("stream_downgrade", map[string]any{"reason": "ollama streaming unavailable; using synchronous assistant completion"})
		emitStage("sync_completion_start", map[string]any{"threadId": threadID, "userMessageId": userMessageID})
		am := s.completeAssistantSync(ctx, threadID, userMessageID, th, umContent, ollamaAdapter, false, requestedModelID)
		if am == nil {
			b, _ := json.Marshal(map[string]any{"message": "assistant reply could not be saved"})
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(b))
			w.(http.Flusher).Flush()
			return
		}
		emitStage("sync_completion_done", map[string]any{"messageId": am.ID})
		s.writeSSEEvent(w, map[string]any{"assistantMessage": am})
		return
	}

	s.initSSE(w)
	emit := func(event string, payload map[string]any) {
		s.writeNamedSSEEvent(w, event, payload)
	}
	am := s.completeAssistantWithGatewayTools(ctx, threadID, userMessageID, th, umContent, ollamaAdapter, false, emit, requestedModelID)
	if am == nil {
		b, _ := json.Marshal(map[string]any{"message": "assistant reply could not be saved"})
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(b))
		w.(http.Flusher).Flush()
		return
	}
	_ = s.log.Emit(ctx, "chat.message.assistant", map[string]any{"threadId": threadID, "messageId": am.ID, "ok": true, "stream": true, "tools": true})
	s.writeSSEEvent(w, map[string]any{"assistantMessage": am})
}

func (s *Server) initSSE(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
}

func (s *Server) writeSSEEvent(w http.ResponseWriter, payload map[string]any) {
	s.writeNamedSSEEvent(w, "done", payload)
}

func (s *Server) writeNamedSSEEvent(w http.ResponseWriter, event string, payload map[string]any) {
	if strings.TrimSpace(event) == "" {
		event = "done"
	}
	b, err := json.Marshal(payload)
	if err != nil {
		b = []byte(`{"error":"encode"}`)
	}
	flusher := w.(http.Flusher)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(b))
	flusher.Flush()
}

func (s *Server) writeSSEAssistantDone(w http.ResponseWriter, msg *chat.Message) {
	s.initSSE(w)
	s.writeSSEEvent(w, map[string]any{"assistantMessage": msg, "cached": true})
}
