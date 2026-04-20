package api

import (
	"context"
	"strings"
	"testing"
)

func TestDiscordGatewayExecuteIntentPing(t *testing.T) {
	t.Parallel()

	gateway := &DiscordGateway{
		cfg: discordGatewayConfig{},
	}
	envelope := discordEventEnvelope{
		CorrelationID: "discord:message_create:abc",
		TimestampMs:   1_900_000_000_000,
		Actor:         discordActorIdentity{ExternalID: "u1"},
	}
	intent := discordIntent{
		Command: "ping",
		Class:   discordIntentSystemQuery,
	}

	resp, err := gateway.executeIntent(context.Background(), envelope, intent)
	if err != nil {
		t.Fatalf("execute ping: %v", err)
	}
	if resp.Kind != discordResponseStatus {
		t.Fatalf("response kind = %q, want %q", resp.Kind, discordResponseStatus)
	}
	if !strings.Contains(resp.Content, "pong") {
		t.Fatalf("response content = %q", resp.Content)
	}
}

func TestDiscordGatewayExecuteIntentPermissionDenied(t *testing.T) {
	t.Parallel()

	gateway := &DiscordGateway{
		cfg: discordGatewayConfig{
			AdminUserIDs: map[string]struct{}{},
			AdminRoleIDs: map[string]struct{}{},
		},
	}
	envelope := discordEventEnvelope{
		Actor: discordActorIdentity{
			ExternalID: "u1",
			RoleIDs:    []string{"role_member"},
		},
	}
	intent := discordIntent{
		Command: "agents",
		Class:   discordIntentAgentControl,
	}

	resp, err := gateway.executeIntent(context.Background(), envelope, intent)
	if err != nil {
		t.Fatalf("execute intent: %v", err)
	}
	if resp.Kind != discordResponseError {
		t.Fatalf("response kind = %q, want %q", resp.Kind, discordResponseError)
	}
	if !strings.Contains(strings.ToLower(resp.Content), "permission denied") {
		t.Fatalf("response content = %q", resp.Content)
	}
}

func TestDiscordGatewayMemoryQueryRequiresArgument(t *testing.T) {
	t.Parallel()

	gateway := &DiscordGateway{}
	resp, err := gateway.memoryQueryIntentResponse(context.Background(), "   ")
	if err != nil {
		t.Fatalf("memory query response: %v", err)
	}
	if resp.ErrorCode != "MISSING_QUERY" {
		t.Fatalf("error code = %q, want MISSING_QUERY", resp.ErrorCode)
	}
}
