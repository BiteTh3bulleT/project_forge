package api

import "strings"

func authorizeDiscordIntent(intent discordIntent, actor discordActorIdentity, cfg discordGatewayConfig) discordPermissionDecision {
	adminOnly := map[string]struct{}{
		"agents":        {},
		"agent_control": {},
	}
	if _, ok := adminOnly[intent.Command]; ok {
		if actorIsDiscordAdmin(actor, cfg) {
			return discordPermissionDecision{Allowed: true}
		}
		return discordPermissionDecision{Allowed: false, Reason: "admin command requires configured admin user or role"}
	}
	if intent.Command == "unknown" {
		return discordPermissionDecision{Allowed: false, Reason: "unknown command"}
	}
	return discordPermissionDecision{Allowed: true}
}

func actorIsDiscordAdmin(actor discordActorIdentity, cfg discordGatewayConfig) bool {
	if actor.IsAdmin {
		return true
	}
	id := strings.TrimSpace(actor.ExternalID)
	if id != "" {
		if _, ok := cfg.AdminUserIDs[id]; ok {
			return true
		}
	}
	for _, roleID := range actor.RoleIDs {
		if _, ok := cfg.AdminRoleIDs[strings.TrimSpace(roleID)]; ok {
			return true
		}
	}
	return false
}
