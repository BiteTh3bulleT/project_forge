package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/forgekernel/runtimeproposal"
)

const (
	governedContextDecisionMetadata = "forgeKContextDecisionDigest"
	governedContextBundleMetadata   = "forgeKContextBundleHash"
)

type governedPromptBinding struct {
	DecisionDigest string
	BundleHash     string
	PacketID       string
}

func (s *Server) compileGovernedChatContext(ctx context.Context, th *chat.ThreadDetail) (string, governedPromptBinding, error) {
	if s == nil || s.kernelAuthority.Processor == nil || !s.kernelAuthority.SingleAuthority || !s.kernelAuthorizationReady {
		return "", governedPromptBinding{}, fmt.Errorf("production FORGE-K Context Compiler is unavailable")
	}
	latest, ok := latestUserChatMessage(th)
	if !ok {
		return "", governedPromptBinding{}, fmt.Errorf("current user turn is required for context compilation")
	}
	workspace := strings.TrimSpace(s.cfg.WorkspaceDir)
	if workspace == "" {
		return "", governedPromptBinding{}, fmt.Errorf("workspace scope is unavailable")
	}
	requestID := fmt.Sprintf("chat-context:%d:%d", th.ID, latest.ID)
	result, err := s.kernelAuthority.Processor.Process(ctx, domain.SyscallRequest{
		ID: requestID, Action: domain.ActionCompileContext,
		Actor: domain.ActorIdentity{ID: "forge.core", Kind: "service"}, Source: domain.SourceInternal,
		Scope: domain.ForgeScope{WorkspaceID: workspace, LaneID: "control.semantic", SelectedPaths: []string{workspace}},
		Payload: map[string]any{
			"query":           strings.TrimSpace(latest.Content),
			"budget":          map[string]any{"maxTokens": 4000, "maxEvents": 32, "maxNotes": 32},
			"persistSnapshot": true, "snapshotKind": "chat",
		},
		Provenance:    domain.Provenance{Actor: "forge.core", ActorType: "service", Source: "chat_context_compiler", TraceID: requestID + ":trace"},
		CorrelationID: requestID + ":correlation", TraceID: requestID + ":trace",
		IdempotencyKey: requestID, RequestedAt: time.Now().UnixMilli(), RequiredCapability: "context.compile",
		Metadata: map[string]any{"chatThreadId": th.ID, "chatUserMessageId": latest.ID},
	})
	if err != nil || !result.Success {
		if err != nil {
			return "", governedPromptBinding{}, err
		}
		return "", governedPromptBinding{}, fmt.Errorf("context compilation rejected: %v", result.RejectedReasons)
	}
	binding := governedPromptBinding{
		DecisionDigest: strings.TrimSpace(asString(result.StateSummary["contextDecisionDigest"])),
		BundleHash:     strings.TrimSpace(asString(result.StateSummary["contextPacketCommitment"])),
		PacketID:       strings.TrimSpace(asString(result.StateSummary["contextPacketId"])),
	}
	if binding.DecisionDigest == "" || binding.BundleHash == "" || binding.PacketID == "" {
		return "", governedPromptBinding{}, fmt.Errorf("context compiler returned incomplete binding")
	}
	lines := []string{"FORGE-K governed context bundle " + binding.PacketID + ":"}
	rawSources, _ := json.Marshal(result.StateSummary["governedSources"])
	var sourceRows []struct {
		EvidenceID     string `json:"evidenceId"`
		ContentSummary string `json:"contentSummary"`
	}
	if json.Unmarshal(rawSources, &sourceRows) == nil {
		for _, row := range sourceRows {
			id, summary := strings.TrimSpace(row.EvidenceID), strings.TrimSpace(row.ContentSummary)
			if id != "" && summary != "" {
				lines = append(lines, "- ["+id+"] "+summary)
			}
		}
	}
	if len(lines) == 1 {
		lines = append(lines, "- No admitted memory evidence matched this exact scope.")
	}
	return strings.Join(lines, "\n"), binding, nil
}

func latestUserChatMessage(th *chat.ThreadDetail) (chat.Message, bool) {
	if th == nil {
		return chat.Message{}, false
	}
	for i := len(th.Messages) - 1; i >= 0; i-- {
		if strings.EqualFold(strings.TrimSpace(th.Messages[i].Role), "user") && strings.TrimSpace(th.Messages[i].Content) != "" {
			return th.Messages[i], true
		}
	}
	return chat.Message{}, false
}

func (s *Server) prepareGovernedOllamaPrompt(ctx context.Context, th *chat.ThreadDetail) (string, string, governedPromptBinding, error) {
	contextText, binding, err := s.compileGovernedChatContext(ctx, th)
	if err != nil {
		return "", "", governedPromptBinding{}, err
	}
	latest, _ := latestUserChatMessage(th)
	system := s.chatOperatorSystemPrompt()
	if s.gateway != nil {
		system += "\n\n" + s.gateway.ChatSystemSupplement()
	}
	user := contextText + "\n\nCURRENT OPERATOR TURN\n" + strings.TrimSpace(latest.Content)
	return system, user, binding, nil
}

func (s *Server) prepareGovernedModelRuntimePrompt(ctx context.Context, th *chat.ThreadDetail) ([]ModelRuntimeChatMessage, modelRuntimePromptBudget, governedPromptBinding, error) {
	contextText, binding, err := s.compileGovernedChatContext(ctx, th)
	if err != nil {
		return nil, modelRuntimePromptBudget{}, governedPromptBinding{}, err
	}
	latest, _ := latestUserChatMessage(th)
	system := s.chatOperatorSystemPrompt() + "\n\n" + contextText
	messages := []ModelRuntimeChatMessage{{Role: "system", Content: trimSummary(system, modelRuntimePlainChatSystemMax)}, {Role: "user", Content: trimSummary(strings.TrimSpace(latest.Content), modelRuntimePlainChatUserMax)}}
	budget := modelRuntimePromptBudget{ThreadMessages: len(th.Messages), IncludedMessages: 1, UserChars: len(messages[1].Content), SystemChars: len(messages[0].Content), TotalChars: len(messages[0].Content) + len(messages[1].Content)}
	return messages, budget, binding, nil
}

func (s *Server) prepareGovernedDirectModelRequest(ctx context.Context, req ModelRuntimeChatRequest) (ModelRuntimeChatRequest, error) {
	if s == nil || s.kernelAuthority.Processor == nil || !s.kernelAuthority.SingleAuthority || !s.kernelAuthorizationReady {
		return req, fmt.Errorf("production FORGE-K Context Compiler is unavailable")
	}
	query := strings.TrimSpace(req.Prompt)
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if strings.EqualFold(strings.TrimSpace(req.Messages[i].Role), "user") && strings.TrimSpace(req.Messages[i].Content) != "" {
			query = strings.TrimSpace(req.Messages[i].Content)
			break
		}
	}
	if query == "" {
		return req, fmt.Errorf("current user turn is required for context compilation")
	}
	workspace := strings.TrimSpace(req.Meta.WorkspaceID)
	if workspace == "" {
		workspace = strings.TrimSpace(s.cfg.WorkspaceDir)
	}
	promptHash := runtimeproposal.HashText(query + "\n" + strings.TrimSpace(req.Meta.CorrelationID))
	requestID := "direct-model-context:" + strings.TrimPrefix(promptHash, "sha256:")[:24]
	result, err := s.kernelAuthority.Processor.Process(ctx, domain.SyscallRequest{
		ID: requestID, Action: domain.ActionCompileContext,
		Actor: domain.ActorIdentity{ID: "forge.core", Kind: "service"}, Source: domain.SourceInternal,
		Scope:         domain.ForgeScope{WorkspaceID: workspace, LaneID: "control.semantic", SelectedPaths: []string{workspace}},
		Payload:       map[string]any{"query": query, "budget": map[string]any{"maxTokens": 4000, "maxEvents": 32, "maxNotes": 32}, "persistSnapshot": true, "snapshotKind": "direct_model"},
		Provenance:    domain.Provenance{Actor: "forge.core", ActorType: "service", Source: "direct_model_context_compiler", TraceID: requestID + ":trace"},
		CorrelationID: requestID + ":correlation", TraceID: requestID + ":trace", IdempotencyKey: requestID, RequestedAt: time.Now().UnixMilli(), RequiredCapability: "context.compile",
	})
	if err != nil || !result.Success {
		if err != nil {
			return req, err
		}
		return req, fmt.Errorf("context compilation rejected: %v", result.RejectedReasons)
	}
	binding := governedPromptBinding{DecisionDigest: strings.TrimSpace(asString(result.StateSummary["contextDecisionDigest"])), BundleHash: strings.TrimSpace(asString(result.StateSummary["contextPacketCommitment"])), PacketID: strings.TrimSpace(asString(result.StateSummary["contextPacketId"]))}
	if binding.DecisionDigest == "" || binding.BundleHash == "" {
		return req, fmt.Errorf("context compiler returned incomplete binding")
	}
	contextText := "FORGE-K governed context bundle " + binding.PacketID + ". Only current admitted memory evidence is eligible; no legacy chat memory is authoritative."
	req.Messages = append([]ModelRuntimeChatMessage{{Role: "system", Content: contextText}}, req.Messages...)
	if len(req.Messages) == 1 && strings.TrimSpace(req.Prompt) != "" {
		req.Prompt = contextText + "\n\nCURRENT OPERATOR TURN\n" + strings.TrimSpace(req.Prompt)
	}
	req.Metadata = cloneAnyMap(req.Metadata)
	req.Metadata[governedContextDecisionMetadata] = binding.DecisionDigest
	req.Metadata[governedContextBundleMetadata] = binding.BundleHash
	return req, nil
}

func governedBindingFromModelRequest(req ModelRuntimeChatRequest) governedPromptBinding {
	return governedPromptBinding{DecisionDigest: strings.TrimSpace(asString(req.Metadata[governedContextDecisionMetadata])), BundleHash: strings.TrimSpace(asString(req.Metadata[governedContextBundleMetadata]))}
}
