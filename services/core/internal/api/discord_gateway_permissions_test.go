package api

import "testing"

func TestAuthorizeDiscordIntentAdminCommandDeniedWithoutAdmin(t *testing.T) {
	t.Parallel()

	decision := authorizeDiscordIntent(
		discordIntent{Command: "agents"},
		discordActorIdentity{ExternalID: "user_1", RoleIDs: []string{"role_member"}},
		discordGatewayConfig{
			AdminUserIDs: map[string]struct{}{},
			AdminRoleIDs: map[string]struct{}{},
		},
	)
	if decision.Allowed {
		t.Fatalf("expected deny for non-admin actor")
	}
}

func TestAuthorizeDiscordIntentAdminCommandAllowedByRole(t *testing.T) {
	t.Parallel()

	decision := authorizeDiscordIntent(
		discordIntent{Command: "agents"},
		discordActorIdentity{ExternalID: "user_1", RoleIDs: []string{"role_admin"}},
		discordGatewayConfig{
			AdminUserIDs: map[string]struct{}{},
			AdminRoleIDs: map[string]struct{}{
				"role_admin": {},
			},
		},
	)
	if !decision.Allowed {
		t.Fatalf("expected allow for admin role")
	}
}

func TestAuthorizeDiscordIntentUnknownDenied(t *testing.T) {
	t.Parallel()

	decision := authorizeDiscordIntent(
		discordIntent{Command: "unknown"},
		discordActorIdentity{ExternalID: "u1"},
		discordGatewayConfig{},
	)
	if decision.Allowed {
		t.Fatalf("expected unknown command deny")
	}
}

func TestFormatDiscordResponse(t *testing.T) {
	t.Parallel()

	status := formatDiscordResponse(discordResponse{
		Kind:    discordResponseStatus,
		Content: "ok=true",
	})
	if status != "FORGE status:\nok=true" {
		t.Fatalf("unexpected formatted status: %q", status)
	}
	errLine := formatDiscordResponse(discordResponse{
		Kind:    discordResponseError,
		Content: "permission denied",
	})
	if errLine != "FORGE error:\npermission denied" {
		t.Fatalf("unexpected formatted error: %q", errLine)
	}
}
