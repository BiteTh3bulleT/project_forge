package modelruntime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ChatExecutionRole string

const (
	ChatExecutionRoleAssistant     ChatExecutionRole = "assistant"
	ChatExecutionRolePlanner       ChatExecutionRole = "planner"
	ChatExecutionRoleExecutor      ChatExecutionRole = "executor"
	ChatExecutionRoleVerifier      ChatExecutionRole = "verifier"
	ChatExecutionRoleSummarizer    ChatExecutionRole = "summarizer"
	ChatExecutionRoleRepairAnalyst ChatExecutionRole = "repair_analyst"
)

type ChatExecutionState string

const (
	ChatExecutionStateQueued      ChatExecutionState = "queued"
	ChatExecutionStateRouted      ChatExecutionState = "routed"
	ChatExecutionStateRunning     ChatExecutionState = "running"
	ChatExecutionStateBackingOff  ChatExecutionState = "backing_off"
	ChatExecutionStateCoolingDown ChatExecutionState = "cooldown_blocked"
	ChatExecutionStateCompleted   ChatExecutionState = "completed"
	ChatExecutionStateFailed      ChatExecutionState = "failed"
)

type ChatExecutionRequest struct {
	Role ChatExecutionRole `json:"role,omitempty"`
	GenerateRequest
	MaxAttempts int `json:"maxAttempts,omitempty"`
}

type ChatExecutionAttempt struct {
	Attempt       int                `json:"attempt"`
	ModelID       string             `json:"modelId"`
	Backend       ModelBackendKind   `json:"backend"`
	State         ChatExecutionState `json:"state"`
	StartedAt     time.Time          `json:"startedAt"`
	FinishedAt    time.Time          `json:"finishedAt,omitempty"`
	Error         string             `json:"error,omitempty"`
	CooldownUntil time.Time          `json:"cooldownUntil,omitempty"`
}

type ChatExecutionTransition struct {
	State   ChatExecutionState `json:"state"`
	At      time.Time          `json:"at"`
	ModelID string             `json:"modelId,omitempty"`
	Backend ModelBackendKind   `json:"backend,omitempty"`
	Detail  string             `json:"detail,omitempty"`
}

type ChatExecutionCheckpoint struct {
	ExecutionID   string                    `json:"executionId"`
	CorrelationID string                    `json:"correlationId,omitempty"`
	TraceID       string                    `json:"traceId,omitempty"`
	WorkspaceID   string                    `json:"workspaceId,omitempty"`
	Role          ChatExecutionRole         `json:"role"`
	ModelID       string                    `json:"modelId,omitempty"`
	Backend       ModelBackendKind          `json:"backend,omitempty"`
	State         ChatExecutionState        `json:"state"`
	AttemptCount  int                       `json:"attemptCount"`
	MaxAttempts   int                       `json:"maxAttempts"`
	LastError     string                    `json:"lastError,omitempty"`
	CooldownUntil time.Time                 `json:"cooldownUntil,omitempty"`
	StartedAt     time.Time                 `json:"startedAt"`
	UpdatedAt     time.Time                 `json:"updatedAt"`
	FinishedAt    time.Time                 `json:"finishedAt,omitempty"`
	Attempts      []ChatExecutionAttempt    `json:"attempts,omitempty"`
	Transitions   []ChatExecutionTransition `json:"transitions,omitempty"`
}

type ChatExecutionResult struct {
	GenerateResult
	ExecutionID  string                  `json:"executionId"`
	Role         ChatExecutionRole       `json:"role"`
	AttemptCount int                     `json:"attemptCount"`
	Checkpoint   ChatExecutionCheckpoint `json:"checkpoint"`
}

type chatExecutionCandidate struct {
	manifest   ModelManifest
	status     ModelStatus
	preferred  bool
	roleWeight int
	statusRank int
}

func normalizeChatExecutionRole(role ChatExecutionRole) ChatExecutionRole {
	normalized := strings.ToLower(strings.TrimSpace(string(role)))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	switch ChatExecutionRole(normalized) {
	case ChatExecutionRolePlanner:
		return ChatExecutionRolePlanner
	case ChatExecutionRoleExecutor:
		return ChatExecutionRoleExecutor
	case ChatExecutionRoleVerifier:
		return ChatExecutionRoleVerifier
	case ChatExecutionRoleSummarizer:
		return ChatExecutionRoleSummarizer
	case ChatExecutionRoleRepairAnalyst:
		return ChatExecutionRoleRepairAnalyst
	default:
		return ChatExecutionRoleAssistant
	}
}

func (s *Service) ExecuteChatRole(ctx context.Context, req ChatExecutionRequest) (ChatExecutionResult, error) {
	req.Role = normalizeChatExecutionRole(req.Role)
	req.GenerateRequest.ModelID = strings.TrimSpace(req.GenerateRequest.ModelID)
	req.GenerateRequest.Backend = ParseModelBackendKind(string(req.GenerateRequest.Backend))
	req.GenerateRequest.Actor = strings.TrimSpace(req.GenerateRequest.Actor)
	req.GenerateRequest.Source = strings.TrimSpace(req.GenerateRequest.Source)
	req.GenerateRequest.WorkspaceID = strings.TrimSpace(req.GenerateRequest.WorkspaceID)
	req.GenerateRequest.Scope = strings.TrimSpace(req.GenerateRequest.Scope)
	req.GenerateRequest.CorrelationID = strings.TrimSpace(req.GenerateRequest.CorrelationID)
	req.GenerateRequest.TraceID = strings.TrimSpace(req.GenerateRequest.TraceID)

	executionID := s.nextChatExecutionID(req.GenerateRequest.CorrelationID)
	now := s.clock().UTC()
	maxAttempts := req.MaxAttempts
	if maxAttempts <= 0 || maxAttempts > s.chatMaxAttempts {
		maxAttempts = s.chatMaxAttempts
	}
	checkpoint := ChatExecutionCheckpoint{
		ExecutionID:   executionID,
		CorrelationID: req.GenerateRequest.CorrelationID,
		TraceID:       req.GenerateRequest.TraceID,
		WorkspaceID:   req.GenerateRequest.WorkspaceID,
		Role:          req.Role,
		State:         ChatExecutionStateQueued,
		MaxAttempts:   maxAttempts,
		StartedAt:     now,
		UpdatedAt:     now,
	}
	appendChatTransition(&checkpoint, "created")
	s.storeChatCheckpoint(checkpoint)

	candidates, err := s.chatExecutionCandidates(req)
	if err != nil {
		checkpoint.State = ChatExecutionStateFailed
		checkpoint.LastError = err.Error()
		checkpoint.FinishedAt = s.clock().UTC()
		checkpoint.UpdatedAt = checkpoint.FinishedAt
		appendChatTransition(&checkpoint, err.Error())
		s.storeChatCheckpoint(checkpoint)
		s.recordChatExecutionAudit(ctx, checkpoint, "error")
		return ChatExecutionResult{ExecutionID: executionID, Role: req.Role, Checkpoint: checkpoint}, err
	}
	if len(candidates) == 0 {
		err = fmt.Errorf("%w: no chat-capable model candidates for role %s", ErrModelUnavailable, req.Role)
		checkpoint.State = ChatExecutionStateFailed
		checkpoint.LastError = err.Error()
		checkpoint.FinishedAt = s.clock().UTC()
		checkpoint.UpdatedAt = checkpoint.FinishedAt
		appendChatTransition(&checkpoint, err.Error())
		s.storeChatCheckpoint(checkpoint)
		s.recordChatExecutionAudit(ctx, checkpoint, "error")
		return ChatExecutionResult{ExecutionID: executionID, Role: req.Role, Checkpoint: checkpoint}, err
	}

	checkpoint.State = ChatExecutionStateRouted
	checkpoint.ModelID = candidates[0].manifest.ID
	checkpoint.Backend = candidates[0].manifest.Backend
	checkpoint.UpdatedAt = s.clock().UTC()
	appendChatTransition(&checkpoint, "candidate_selected")
	s.storeChatCheckpoint(checkpoint)

	attemptedModels := map[string]struct{}{}
	var lastErr error
	var blockedCooldownUntil time.Time

	for len(checkpoint.Attempts) < maxAttempts {
		candidate, cooldownUntil, ok := s.nextChatCandidate(candidates, attemptedModels)
		if !ok {
			if lastErr == nil {
				lastErr = fmt.Errorf("%w: no eligible provider available", ErrProviderCooldownActive)
			} else {
				lastErr = fmt.Errorf("%w: %v", ErrChatRetryExhausted, lastErr)
			}
			checkpoint.State = ChatExecutionStateCoolingDown
			checkpoint.LastError = lastErr.Error()
			checkpoint.CooldownUntil = blockedCooldownUntil
			if cooldownUntil.After(blockedCooldownUntil) {
				checkpoint.CooldownUntil = cooldownUntil
			}
			checkpoint.FinishedAt = s.clock().UTC()
			checkpoint.UpdatedAt = checkpoint.FinishedAt
			appendChatTransition(&checkpoint, checkpoint.LastError)
			s.storeChatCheckpoint(checkpoint)
			s.recordChatExecutionAudit(ctx, checkpoint, "error")
			return ChatExecutionResult{
				ExecutionID:  executionID,
				Role:         req.Role,
				AttemptCount: checkpoint.AttemptCount,
				Checkpoint:   checkpoint,
			}, lastErr
		}

		attemptedModels[candidate.manifest.ID] = struct{}{}
		attempt := ChatExecutionAttempt{
			Attempt:   len(checkpoint.Attempts) + 1,
			ModelID:   candidate.manifest.ID,
			Backend:   candidate.manifest.Backend,
			State:     ChatExecutionStateRunning,
			StartedAt: s.clock().UTC(),
		}
		checkpoint.Attempts = append(checkpoint.Attempts, attempt)
		checkpoint.AttemptCount = len(checkpoint.Attempts)
		checkpoint.ModelID = candidate.manifest.ID
		checkpoint.Backend = candidate.manifest.Backend
		checkpoint.State = ChatExecutionStateRunning
		checkpoint.CooldownUntil = time.Time{}
		checkpoint.UpdatedAt = attempt.StartedAt
		appendChatTransition(&checkpoint, fmt.Sprintf("attempt_%d_started", checkpoint.AttemptCount))
		s.storeChatCheckpoint(checkpoint)

		attemptReq := req.GenerateRequest
		attemptReq.ModelID = candidate.manifest.ID
		attemptReq.Backend = candidate.manifest.Backend
		attemptReq.Metadata = attachChatExecutionMetadata(req.GenerateRequest.Metadata, executionID, req.Role, checkpoint.AttemptCount, maxAttempts)

		result, runErr := s.Generate(ctx, attemptReq)
		finishedAt := s.clock().UTC()
		attempt.FinishedAt = finishedAt
		if runErr == nil {
			attempt.State = ChatExecutionStateCompleted
			checkpoint.Attempts[len(checkpoint.Attempts)-1] = attempt
			checkpoint.State = ChatExecutionStateCompleted
			checkpoint.LastError = ""
			checkpoint.FinishedAt = finishedAt
			checkpoint.UpdatedAt = finishedAt
			appendChatTransition(&checkpoint, fmt.Sprintf("attempt_%d_succeeded", checkpoint.AttemptCount))
			s.storeChatCheckpoint(checkpoint)
			s.recordChatExecutionAudit(ctx, checkpoint, "ok")
			if checkpoint.AttemptCount > 1 {
				result.Warnings = append(result.Warnings, fmt.Sprintf("chat execution succeeded after %d attempts", checkpoint.AttemptCount))
			}
			return ChatExecutionResult{
				GenerateResult: result,
				ExecutionID:    executionID,
				Role:           req.Role,
				AttemptCount:   checkpoint.AttemptCount,
				Checkpoint:     checkpoint,
			}, nil
		}

		lastErr = runErr
		attempt.State = ChatExecutionStateFailed
		attempt.Error = runErr.Error()
		checkpoint.Attempts[len(checkpoint.Attempts)-1] = attempt
		checkpoint.LastError = runErr.Error()
		checkpoint.UpdatedAt = finishedAt

		if !shouldRetryChatExecution(runErr) || checkpoint.AttemptCount >= maxAttempts {
			checkpoint.State = ChatExecutionStateFailed
			checkpoint.FinishedAt = finishedAt
			appendChatTransition(&checkpoint, checkpoint.LastError)
			s.storeChatCheckpoint(checkpoint)
			s.recordChatExecutionAudit(ctx, checkpoint, "error")
			if checkpoint.AttemptCount >= maxAttempts && shouldRetryChatExecution(runErr) {
				return ChatExecutionResult{
					ExecutionID:  executionID,
					Role:         req.Role,
					AttemptCount: checkpoint.AttemptCount,
					Checkpoint:   checkpoint,
				}, fmt.Errorf("%w: %v", ErrChatRetryExhausted, runErr)
			}
			return ChatExecutionResult{
				ExecutionID:  executionID,
				Role:         req.Role,
				AttemptCount: checkpoint.AttemptCount,
				Checkpoint:   checkpoint,
			}, runErr
		}

		modelCooldownUntil := s.setChatModelCooldown(candidate.manifest.ID, checkpoint.AttemptCount)
		backendCooldownUntil := s.setChatProviderCooldown(candidate.manifest.Backend)
		cooldownUntil = maxTime(modelCooldownUntil, backendCooldownUntil)
		attempt.CooldownUntil = cooldownUntil
		checkpoint.Attempts[len(checkpoint.Attempts)-1] = attempt
		checkpoint.CooldownUntil = cooldownUntil
		checkpoint.State = ChatExecutionStateBackingOff
		checkpoint.UpdatedAt = s.clock().UTC()
		appendChatTransition(&checkpoint, checkpoint.LastError)
		s.storeChatCheckpoint(checkpoint)
		if cooldownUntil.After(blockedCooldownUntil) {
			blockedCooldownUntil = cooldownUntil
		}
		if err := s.waitChatRetryBackoff(ctx, checkpoint.AttemptCount); err != nil {
			checkpoint.State = ChatExecutionStateFailed
			checkpoint.LastError = err.Error()
			checkpoint.FinishedAt = s.clock().UTC()
			checkpoint.UpdatedAt = checkpoint.FinishedAt
			appendChatTransition(&checkpoint, checkpoint.LastError)
			s.storeChatCheckpoint(checkpoint)
			s.recordChatExecutionAudit(ctx, checkpoint, "error")
			return ChatExecutionResult{
				ExecutionID:  executionID,
				Role:         req.Role,
				AttemptCount: checkpoint.AttemptCount,
				Checkpoint:   checkpoint,
			}, err
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("%w: no attempts executed", ErrChatRetryExhausted)
	}
	checkpoint.State = ChatExecutionStateFailed
	checkpoint.LastError = lastErr.Error()
	checkpoint.FinishedAt = s.clock().UTC()
	checkpoint.UpdatedAt = checkpoint.FinishedAt
	appendChatTransition(&checkpoint, checkpoint.LastError)
	s.storeChatCheckpoint(checkpoint)
	s.recordChatExecutionAudit(ctx, checkpoint, "error")
	return ChatExecutionResult{
		ExecutionID:  executionID,
		Role:         req.Role,
		AttemptCount: checkpoint.AttemptCount,
		Checkpoint:   checkpoint,
	}, fmt.Errorf("%w: %v", ErrChatRetryExhausted, lastErr)
}

func (s *Service) ChatExecutionCheckpoint(executionID string) (ChatExecutionCheckpoint, bool) {
	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	checkpoint, ok := s.chatCheckpoints[strings.TrimSpace(executionID)]
	return checkpoint, ok
}

func (s *Service) ChatExecutionCheckpoints() []ChatExecutionCheckpoint {
	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	out := make([]ChatExecutionCheckpoint, 0, len(s.chatCheckpointOrder))
	for _, executionID := range s.chatCheckpointOrder {
		checkpoint, ok := s.chatCheckpoints[executionID]
		if !ok {
			continue
		}
		out = append(out, checkpoint)
	}
	return out
}

func (s *Service) nextChatExecutionID(correlationID string) string {
	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	s.nextChatExecutionSeq++
	if trimmed := strings.TrimSpace(correlationID); trimmed != "" {
		return fmt.Sprintf("%s-%04d", trimmed, s.nextChatExecutionSeq)
	}
	return fmt.Sprintf("chat-%08d", s.nextChatExecutionSeq)
}

func (s *Service) storeChatCheckpoint(checkpoint ChatExecutionCheckpoint) {
	checkpoint.ExecutionID = strings.TrimSpace(checkpoint.ExecutionID)
	if checkpoint.ExecutionID == "" {
		return
	}
	checkpoint.Role = normalizeChatExecutionRole(checkpoint.Role)
	checkpoint.Attempts = append([]ChatExecutionAttempt(nil), checkpoint.Attempts...)
	checkpoint.Transitions = append([]ChatExecutionTransition(nil), checkpoint.Transitions...)

	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	if _, exists := s.chatCheckpoints[checkpoint.ExecutionID]; !exists {
		s.chatCheckpointOrder = append(s.chatCheckpointOrder, checkpoint.ExecutionID)
	}
	s.chatCheckpoints[checkpoint.ExecutionID] = checkpoint
	if len(s.chatCheckpointOrder) <= s.chatCheckpointLimit {
		return
	}
	excess := len(s.chatCheckpointOrder) - s.chatCheckpointLimit
	for _, executionID := range s.chatCheckpointOrder[:excess] {
		delete(s.chatCheckpoints, executionID)
	}
	s.chatCheckpointOrder = append([]string(nil), s.chatCheckpointOrder[excess:]...)
}

func (s *Service) chatExecutionCandidates(req ChatExecutionRequest) ([]chatExecutionCandidate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requestedModelID := strings.TrimSpace(req.GenerateRequest.ModelID)
	if requestedModelID != "" {
		manifest, ok := s.models[requestedModelID]
		if !ok {
			return nil, ErrModelNotFound
		}
		if requestedBackend := req.GenerateRequest.Backend; requestedBackend != "" && requestedBackend != manifest.Backend {
			return nil, fmt.Errorf("%w: model %s uses backend %s not %s", ErrUnsupportedBackendOverride, manifest.ID, manifest.Backend, requestedBackend)
		}
		status := s.status[requestedModelID]
		if status == "" {
			status = StatusAvailable
		}
		if !chatCandidateUsable(status) {
			return nil, ErrModelUnavailable
		}
		if !manifestHasCapability(manifest, CapabilityChat) && !manifestHasCapability(manifest, CapabilityCompletion) {
			return nil, fmt.Errorf("%w: model %s missing capability chat/completion", ErrModelCapabilityUnsupported, manifest.ID)
		}
		return []chatExecutionCandidate{{
			manifest:   manifest,
			status:     status,
			preferred:  s.isPreferredModelLocked(manifest.ID),
			roleWeight: chatRoleWeight(manifest, req.Role),
			statusRank: chatCandidateStatusRank(status),
		}}, nil
	}

	candidates := make([]chatExecutionCandidate, 0, len(s.models))
	for modelID, manifest := range s.models {
		if req.GenerateRequest.Backend != "" && manifest.Backend != req.GenerateRequest.Backend {
			continue
		}
		status := s.status[modelID]
		if status == "" {
			status = StatusAvailable
		}
		if !chatCandidateUsable(status) {
			continue
		}
		roleWeight := chatRoleWeight(manifest, req.Role)
		if roleWeight <= 0 {
			continue
		}
		candidates = append(candidates, chatExecutionCandidate{
			manifest:   manifest,
			status:     status,
			preferred:  s.isPreferredModelLocked(modelID),
			roleWeight: roleWeight,
			statusRank: chatCandidateStatusRank(status),
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].preferred != candidates[j].preferred {
			return candidates[i].preferred
		}
		if candidates[i].roleWeight != candidates[j].roleWeight {
			return candidates[i].roleWeight > candidates[j].roleWeight
		}
		if candidates[i].statusRank != candidates[j].statusRank {
			return candidates[i].statusRank > candidates[j].statusRank
		}
		return strings.ToLower(strings.TrimSpace(candidates[i].manifest.ID)) < strings.ToLower(strings.TrimSpace(candidates[j].manifest.ID))
	})
	return candidates, nil
}

func (s *Service) nextChatCandidate(candidates []chatExecutionCandidate, attemptedModels map[string]struct{}) (chatExecutionCandidate, time.Time, bool) {
	now := s.clock().UTC()
	var latestCooldown time.Time
	for _, candidate := range candidates {
		if _, attempted := attemptedModels[candidate.manifest.ID]; attempted {
			continue
		}
		modelCooldownUntil := s.chatModelCooldownUntil(candidate.manifest.ID)
		backendCooldownUntil := s.chatProviderCooldownUntil(candidate.manifest.Backend)
		cooldownUntil := maxTime(modelCooldownUntil, backendCooldownUntil)
		if cooldownUntil.After(now) {
			if cooldownUntil.After(latestCooldown) {
				latestCooldown = cooldownUntil
			}
			continue
		}
		return candidate, time.Time{}, true
	}
	return chatExecutionCandidate{}, latestCooldown, false
}

func (s *Service) chatProviderCooldownUntil(backend ModelBackendKind) time.Time {
	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	return s.chatProviderCooldowns[backend]
}

func (s *Service) chatModelCooldownUntil(modelID string) time.Time {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return time.Time{}
	}
	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	return s.chatModelCooldowns[modelID]
}

func (s *Service) setChatProviderCooldown(backend ModelBackendKind) time.Time {
	if s.chatProviderCooldown <= 0 {
		return time.Time{}
	}
	until := s.clock().UTC().Add(s.chatProviderCooldown)
	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	s.chatProviderCooldowns[backend] = until
	return until
}

func (s *Service) setChatModelCooldown(modelID string, attempt int) time.Time {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" || s.chatModelCooldown <= 0 {
		return time.Time{}
	}
	if attempt < 1 {
		attempt = 1
	}
	multiplier := 1 << uint(min(attempt-1, 6))
	until := s.clock().UTC().Add(s.chatModelCooldown * time.Duration(multiplier))
	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	existing := s.chatModelCooldowns[modelID]
	if existing.After(until) {
		return existing
	}
	s.chatModelCooldowns[modelID] = until
	return until
}

func (s *Service) waitChatRetryBackoff(ctx context.Context, attempt int) error {
	if s.chatRetryBackoff <= 0 {
		return nil
	}
	if attempt < 1 {
		attempt = 1
	}
	delay := s.chatRetryBackoff * time.Duration(1<<(attempt-1))
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *Service) recordChatExecutionAudit(ctx context.Context, checkpoint ChatExecutionCheckpoint, outcome string) {
	metadata := map[string]any{
		"executionId":   checkpoint.ExecutionID,
		"role":          string(checkpoint.Role),
		"attemptCount":  checkpoint.AttemptCount,
		"maxAttempts":   checkpoint.MaxAttempts,
		"state":         string(checkpoint.State),
		"cooldownUntil": checkpoint.CooldownUntil,
		"attempts":      checkpoint.Attempts,
	}
	if checkpoint.LastError != "" {
		metadata["lastError"] = checkpoint.LastError
	}
	s.recordAudit(ctx, ModelRuntimeAuditRecord{
		Operation:     "chat_execute",
		ModelID:       checkpoint.ModelID,
		Backend:       checkpoint.Backend,
		WorkspaceID:   checkpoint.WorkspaceID,
		CorrelationID: checkpoint.CorrelationID,
		TraceID:       checkpoint.TraceID,
		Outcome:       outcome,
		Error:         checkpoint.LastError,
		Metadata:      metadata,
	})
}

func (s *Service) isPreferredModelLocked(modelID string) bool {
	if strings.TrimSpace(modelID) == strings.TrimSpace(s.defaultModelID) && strings.TrimSpace(modelID) != "" {
		return true
	}
	if s.registry == nil {
		return false
	}
	registered, ok := s.registry.Get(modelID)
	return ok && registered.State.Preferred
}

func attachChatExecutionMetadata(metadata map[string]any, executionID string, role ChatExecutionRole, attempt, maxAttempts int) map[string]any {
	out := cloneStateMetadata(metadata)
	out["chatExecution"] = map[string]any{
		"executionId": executionID,
		"role":        string(normalizeChatExecutionRole(role)),
		"attempt":     attempt,
		"maxAttempts": maxAttempts,
	}
	return out
}

func appendChatTransition(checkpoint *ChatExecutionCheckpoint, detail string) {
	if checkpoint == nil {
		return
	}
	checkpoint.Transitions = append(checkpoint.Transitions, ChatExecutionTransition{
		State:   checkpoint.State,
		At:      checkpoint.UpdatedAt,
		ModelID: checkpoint.ModelID,
		Backend: checkpoint.Backend,
		Detail:  strings.TrimSpace(detail),
	})
}

func maxTime(first, second time.Time) time.Time {
	if first.After(second) {
		return first
	}
	return second
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func chatRoleWeight(manifest ModelManifest, role ChatExecutionRole) int {
	role = normalizeChatExecutionRole(role)
	if explicit := explicitChatRoleWeight(manifest.Metadata, role); explicit > 0 {
		return explicit + 100
	}
	switch role {
	case ChatExecutionRolePlanner:
		if manifestHasCapability(manifest, CapabilityStructuredOutput) && manifestHasCapability(manifest, CapabilityToolCalling) {
			return 40
		}
		if manifestHasCapability(manifest, CapabilityStructuredOutput) {
			return 35
		}
		if manifestHasCapability(manifest, CapabilityToolCalling) {
			return 30
		}
	case ChatExecutionRoleSummarizer:
		if manifestHasCapability(manifest, CapabilityStructuredOutput) {
			return 35
		}
		if manifestHasCapability(manifest, CapabilityChat) {
			return 25
		}
	case ChatExecutionRoleExecutor:
		if manifestHasCapability(manifest, CapabilityCode) {
			return 40
		}
		if manifestHasCapability(manifest, CapabilityToolCalling) {
			return 30
		}
	case ChatExecutionRoleRepairAnalyst:
		if manifestHasCapability(manifest, CapabilityCode) && manifestHasCapability(manifest, CapabilityToolCalling) {
			return 45
		}
		if manifestHasCapability(manifest, CapabilityCode) {
			return 40
		}
		if manifestHasCapability(manifest, CapabilityStructuredOutput) {
			return 35
		}
	case ChatExecutionRoleVerifier:
		if manifestHasCapability(manifest, CapabilityStructuredOutput) {
			return 40
		}
		if manifestHasCapability(manifest, CapabilityRerank) {
			return 35
		}
	}
	if manifestHasCapability(manifest, CapabilityChat) {
		return 20
	}
	if manifestHasCapability(manifest, CapabilityCompletion) {
		return 10
	}
	return 0
}

func explicitChatRoleWeight(metadata map[string]any, role ChatExecutionRole) int {
	for _, raw := range explicitChatRoles(metadata) {
		if raw == role {
			return 1
		}
	}
	return 0
}

func explicitChatRoles(metadata map[string]any) []ChatExecutionRole {
	if len(metadata) == 0 {
		return nil
	}
	for _, key := range []string{"chatRoles", "chat_roles", "roles"} {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		return parseChatRolesValue(value)
	}
	return nil
}

func parseChatRolesValue(value any) []ChatExecutionRole {
	switch raw := value.(type) {
	case []string:
		out := make([]ChatExecutionRole, 0, len(raw))
		for _, item := range raw {
			role := normalizeChatExecutionRole(ChatExecutionRole(item))
			out = append(out, role)
		}
		return out
	case []any:
		out := make([]ChatExecutionRole, 0, len(raw))
		for _, item := range raw {
			text, ok := item.(string)
			if !ok {
				continue
			}
			out = append(out, normalizeChatExecutionRole(ChatExecutionRole(text)))
		}
		return out
	case string:
		parts := strings.Split(raw, ",")
		out := make([]ChatExecutionRole, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			out = append(out, normalizeChatExecutionRole(ChatExecutionRole(trimmed)))
		}
		return out
	default:
		return nil
	}
}

func chatCandidateUsable(status ModelStatus) bool {
	switch status {
	case StatusDisabled, StatusArchived, StatusUnavailable, StatusError:
		return false
	default:
		return true
	}
}

func chatCandidateStatusRank(status ModelStatus) int {
	switch status {
	case StatusLoaded:
		return 5
	case StatusAvailable:
		return 4
	case StatusVerified:
		return 3
	case StatusImported:
		return 2
	case StatusLoading, StatusUnloading:
		return 1
	default:
		return 0
	}
}

func shouldRetryChatExecution(err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrBackendUnavailable):
		return true
	case errors.Is(err, ErrRequestQueueFull):
		return true
	case errors.Is(err, ErrModelLifecycleBusy):
		return true
	case errors.Is(err, ErrModelNotLoaded):
		return true
	case errors.Is(err, context.DeadlineExceeded):
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	for _, token := range []string{
		"timeout",
		"tempor",
		"unavailable",
		"connection refused",
		"connection reset",
		"bad gateway",
		"gateway timeout",
		"too many requests",
		"503",
		"502",
		"429",
	} {
		if strings.Contains(message, token) {
			return true
		}
	}
	return false
}
