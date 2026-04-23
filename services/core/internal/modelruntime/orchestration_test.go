package modelruntime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceExecuteChatRoleRoutesByRole(t *testing.T) {
	t.Parallel()

	backend := NewFakeBackend(FakeBackendOptions{Healthy: true, Kind: BackendFake})
	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models: []ModelManifest{
			{
				ID:           "assistant-model",
				Backend:      BackendFake,
				Format:       ModelFormatGGUF,
				Capabilities: []ModelCapability{CapabilityChat},
				Metadata:     map[string]any{"chatRoles": []string{"assistant"}},
			},
			{
				ID:           "planner-model",
				Backend:      BackendFake,
				Format:       ModelFormatGGUF,
				Capabilities: []ModelCapability{CapabilityChat, CapabilityStructuredOutput},
				Metadata:     map[string]any{"chatRoles": []string{"planner"}},
			},
		},
		AutoLoad:            true,
		ChatMaxAttempts:     2,
		ChatCheckpointLimit: 8,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := svc.ExecuteChatRole(context.Background(), ChatExecutionRequest{
		Role: ChatExecutionRolePlanner,
		GenerateRequest: GenerateRequest{
			WorkspaceID: "ws-role",
			Actor:       "tester",
			Source:      "orchestration_test",
			Messages: []GenerateMessage{
				{Role: "user", Content: "plan this change"},
			},
			CorrelationID: "corr-role",
			TraceID:       "trace-role",
		},
	})
	if err != nil {
		t.Fatalf("execute chat role: %v", err)
	}
	if result.ModelID != "planner-model" {
		t.Fatalf("expected planner-model to be selected, got %+v", result)
	}
	if result.Role != ChatExecutionRolePlanner {
		t.Fatalf("expected role planner, got %s", result.Role)
	}
}

func TestServiceExecuteChatRoleHonorsRetryBound(t *testing.T) {
	t.Parallel()

	transientErr := errors.New("503 upstream unavailable")
	var fakeCalls, openAICalls, vllmCalls int
	backends := []ModelBackend{
		NewFakeBackend(FakeBackendOptions{
			Healthy: true,
			Kind:    BackendFake,
			Generate: func(req GenerateRequest) (GenerateResult, error) {
				fakeCalls++
				return GenerateResult{}, transientErr
			},
		}),
		NewFakeBackend(FakeBackendOptions{
			Healthy: true,
			Kind:    BackendOpenAICompat,
			Generate: func(req GenerateRequest) (GenerateResult, error) {
				openAICalls++
				return GenerateResult{}, transientErr
			},
		}),
		NewFakeBackend(FakeBackendOptions{
			Healthy: true,
			Kind:    BackendVLLM,
			Generate: func(req GenerateRequest) (GenerateResult, error) {
				vllmCalls++
				return GenerateResult{}, transientErr
			},
		}),
	}
	svc, err := NewService(ServiceOptions{
		Backends: backends,
		Models: []ModelManifest{
			completionManifest("alpha", BackendFake),
			completionManifest("beta", BackendOpenAICompat),
			completionManifest("gamma", BackendVLLM),
		},
		AutoLoad:             true,
		MaxLoadedModels:      4,
		ChatMaxAttempts:      2,
		ChatProviderCooldown: time.Hour,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := svc.ExecuteChatRole(context.Background(), ChatExecutionRequest{
		Role: ChatExecutionRoleAssistant,
		GenerateRequest: GenerateRequest{
			WorkspaceID: "ws-retry",
			Actor:       "tester",
			Source:      "orchestration_test",
			Prompt:      "hello",
		},
	})
	if !errors.Is(err, ErrChatRetryExhausted) {
		t.Fatalf("expected ErrChatRetryExhausted, got %v", err)
	}
	if result.AttemptCount != 2 {
		t.Fatalf("expected 2 attempts, got %+v", result.Checkpoint)
	}
	if fakeCalls+openAICalls+vllmCalls != 2 {
		t.Fatalf("expected exactly 2 backend generate calls, got fake=%d openai=%d vllm=%d", fakeCalls, openAICalls, vllmCalls)
	}
}

func TestServiceExecuteChatRoleRespectsProviderCooldownAcrossExecutions(t *testing.T) {
	t.Parallel()

	var backendACalls, backendBCalls int
	backendA := NewFakeBackend(FakeBackendOptions{
		Healthy: true,
		Kind:    BackendOpenAICompat,
		Generate: func(req GenerateRequest) (GenerateResult, error) {
			backendACalls++
			return GenerateResult{}, errors.New("503 provider unavailable")
		},
	})
	backendB := NewFakeBackend(FakeBackendOptions{
		Healthy: true,
		Kind:    BackendVLLM,
		Generate: func(req GenerateRequest) (GenerateResult, error) {
			backendBCalls++
			return GenerateResult{
				Content:          "ok from " + req.ModelID,
				FinishReason:     "stop",
				PromptTokens:     3,
				CompletionTokens: 4,
			}, nil
		},
	})
	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backendA, backendB},
		Models: []ModelManifest{
			completionManifest("a-model", BackendOpenAICompat),
			completionManifest("b-model", BackendVLLM),
		},
		AutoLoad:             true,
		MaxLoadedModels:      4,
		ChatMaxAttempts:      2,
		ChatProviderCooldown: time.Hour,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	first, err := svc.ExecuteChatRole(context.Background(), ChatExecutionRequest{
		Role: ChatExecutionRoleAssistant,
		GenerateRequest: GenerateRequest{
			WorkspaceID: "ws-cooldown",
			Actor:       "tester",
			Source:      "orchestration_test",
			Prompt:      "first",
		},
	})
	if err != nil {
		t.Fatalf("first execute chat role: %v", err)
	}
	if first.AttemptCount != 2 || first.ModelID != "b-model" {
		t.Fatalf("expected second backend to recover request, got %+v", first)
	}

	second, err := svc.ExecuteChatRole(context.Background(), ChatExecutionRequest{
		Role: ChatExecutionRoleAssistant,
		GenerateRequest: GenerateRequest{
			WorkspaceID: "ws-cooldown",
			Actor:       "tester",
			Source:      "orchestration_test",
			Prompt:      "second",
		},
	})
	if err != nil {
		t.Fatalf("second execute chat role: %v", err)
	}
	if backendACalls != 1 {
		t.Fatalf("expected backend A to stay in cooldown on second execution, got %d calls", backendACalls)
	}
	if backendBCalls != 2 {
		t.Fatalf("expected backend B to serve both executions after cooldown, got %d calls", backendBCalls)
	}
	if second.AttemptCount != 1 || second.ModelID != "b-model" {
		t.Fatalf("expected second execution to route directly to b-model, got %+v", second)
	}
}

func TestServiceExecuteChatRoleCheckpointTransitions(t *testing.T) {
	t.Parallel()

	var firstCall = true
	backendA := NewFakeBackend(FakeBackendOptions{
		Healthy: true,
		Kind:    BackendOpenAICompat,
		Generate: func(req GenerateRequest) (GenerateResult, error) {
			if firstCall {
				firstCall = false
				return GenerateResult{}, errors.New("503 transient provider failure")
			}
			return GenerateResult{
				Content:          "unexpected reuse",
				FinishReason:     "stop",
				PromptTokens:     1,
				CompletionTokens: 1,
			}, nil
		},
	})
	backendB := NewFakeBackend(FakeBackendOptions{
		Healthy: true,
		Kind:    BackendVLLM,
		Generate: func(req GenerateRequest) (GenerateResult, error) {
			return GenerateResult{
				Content:          "verified",
				FinishReason:     "stop",
				PromptTokens:     2,
				CompletionTokens: 2,
			}, nil
		},
	})
	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backendA, backendB},
		Models: []ModelManifest{
			completionManifest("alpha", BackendOpenAICompat),
			completionManifest("beta", BackendVLLM),
		},
		AutoLoad:             true,
		MaxLoadedModels:      4,
		ChatMaxAttempts:      2,
		ChatProviderCooldown: time.Hour,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := svc.ExecuteChatRole(context.Background(), ChatExecutionRequest{
		Role: ChatExecutionRoleAssistant,
		GenerateRequest: GenerateRequest{
			WorkspaceID:   "ws-checkpoint",
			Actor:         "tester",
			Source:        "orchestration_test",
			Prompt:        "checkpoint",
			CorrelationID: "corr-checkpoint",
			TraceID:       "trace-checkpoint",
		},
	})
	if err != nil {
		t.Fatalf("execute chat role: %v", err)
	}

	checkpoint, ok := svc.ChatExecutionCheckpoint(result.ExecutionID)
	if !ok {
		t.Fatalf("expected checkpoint for execution %s", result.ExecutionID)
	}
	if checkpoint.State != ChatExecutionStateCompleted {
		t.Fatalf("expected completed checkpoint, got %+v", checkpoint)
	}
	if len(checkpoint.Attempts) != 2 {
		t.Fatalf("expected 2 attempts in checkpoint, got %+v", checkpoint)
	}
	if got := transitionStates(checkpoint.Transitions); len(got) < 5 {
		t.Fatalf("expected transition history, got %+v", got)
	} else {
		want := []ChatExecutionState{
			ChatExecutionStateQueued,
			ChatExecutionStateRouted,
			ChatExecutionStateRunning,
			ChatExecutionStateBackingOff,
			ChatExecutionStateRunning,
			ChatExecutionStateCompleted,
		}
		for _, expected := range want {
			if !hasTransitionState(got, expected) {
				t.Fatalf("expected transition %s in %v", expected, got)
			}
		}
	}
	if checkpoint.CorrelationID != "corr-checkpoint" || checkpoint.TraceID != "trace-checkpoint" {
		t.Fatalf("expected correlation/trace to persist in checkpoint, got %+v", checkpoint)
	}
}

func transitionStates(transitions []ChatExecutionTransition) []ChatExecutionState {
	out := make([]ChatExecutionState, 0, len(transitions))
	for _, transition := range transitions {
		out = append(out, transition.State)
	}
	return out
}

func hasTransitionState(states []ChatExecutionState, want ChatExecutionState) bool {
	for _, state := range states {
		if state == want {
			return true
		}
	}
	return false
}
