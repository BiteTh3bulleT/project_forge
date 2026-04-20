package gateway

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"forge/projectforge/services/core/internal/aios/domain"
)

type ToolCapabilityRegistry struct {
	mu           sync.RWMutex
	byID         map[string]domain.ToolCapability
	byLegacyTool map[string]string
}

func NewToolCapabilityRegistry() *ToolCapabilityRegistry {
	r := &ToolCapabilityRegistry{
		byID:         map[string]domain.ToolCapability{},
		byLegacyTool: map[string]string{},
	}
	for _, cap := range defaultToolCapabilities() {
		_ = r.Register(cap)
	}
	return r
}

func (r *ToolCapabilityRegistry) Register(capability domain.ToolCapability) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := strings.TrimSpace(strings.ToLower(capability.ID))
	if id == "" {
		return fmt.Errorf("capability id is required")
	}
	if _, exists := r.byID[id]; exists {
		return fmt.Errorf("duplicate capability id %q", id)
	}
	issues := capability.Validate()
	if len(issues) > 0 {
		return issues[0]
	}
	capability.ID = id
	r.byID[id] = capability
	if toolID := metadataString(capability.Metadata, "gatewayToolId"); toolID != "" {
		r.byLegacyTool[strings.ToLower(toolID)] = id
	}
	return nil
}

func (r *ToolCapabilityRegistry) Get(id string) (domain.ToolCapability, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	capability, ok := r.byID[strings.TrimSpace(strings.ToLower(id))]
	return capability, ok
}

func (r *ToolCapabilityRegistry) Resolve(input string) (domain.ToolCapability, bool) {
	key := strings.TrimSpace(strings.ToLower(input))
	if key == "" {
		return domain.ToolCapability{}, false
	}
	if capability, ok := r.Get(key); ok {
		return capability, true
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if capabilityID, ok := r.byLegacyTool[key]; ok {
		capability, exists := r.byID[capabilityID]
		return capability, exists
	}
	return domain.ToolCapability{}, false
}

func (r *ToolCapabilityRegistry) List() []domain.ToolCapability {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.ToolCapability, 0, len(r.byID))
	for _, row := range r.byID {
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r *ToolCapabilityRegistry) ListByDomain(domainID string) []domain.ToolCapability {
	domainID = strings.TrimSpace(strings.ToLower(domainID))
	rows := r.List()
	if domainID == "" {
		return rows
	}
	out := make([]domain.ToolCapability, 0)
	for _, row := range rows {
		if strings.EqualFold(row.Domain, domainID) {
			out = append(out, row)
		}
	}
	return out
}

func (r *ToolCapabilityRegistry) ListByStatus(status domain.ToolCapabilityStatus) []domain.ToolCapability {
	rows := r.List()
	out := make([]domain.ToolCapability, 0)
	for _, row := range rows {
		if row.Status == status {
			out = append(out, row)
		}
	}
	return out
}

func (r *ToolCapabilityRegistry) ListByRisk(risk domain.ToolRisk) []domain.ToolCapability {
	rows := r.List()
	out := make([]domain.ToolCapability, 0)
	for _, row := range rows {
		if row.Risk == risk {
			out = append(out, row)
		}
	}
	return out
}

func (r *ToolCapabilityRegistry) UpdateStatus(id string, status domain.ToolCapabilityStatus) (domain.ToolCapability, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.TrimSpace(strings.ToLower(id))
	row, ok := r.byID[key]
	if !ok {
		return domain.ToolCapability{}, false
	}
	row.Status = status
	r.byID[key] = row
	return row, true
}

func defaultToolCapabilities() []domain.ToolCapability {
	domainPrimitiveMap := map[string][]string{
		"process": {
			"spawn_process", "kill_process", "signal_process", "inspect_process", "set_resource_limits", "run_job", "fork_context", "checkpoint_process", "restore_process",
		},
		"filesystem": {
			"read_file", "write_file", "delete_file", "move_file", "list_dir", "glob", "watch_path", "mount", "unmount", "create_snapshot", "restore_snapshot", "set_permissions", "get_permissions", "query_semantic_fs", "archive", "extract", "sync_to_remote", "sync_from_remote",
		},
		"network": {
			"dns_resolve", "dns_register", "http_request", "open_socket", "close_socket", "proxy_request", "open_tunnel", "scan_network", "intercept_traffic", "set_firewall_rule", "delete_firewall_rule",
		},
		"memory": {
			"remember", "recall", "forget", "embed_content", "semantic_search", "upsert_fact", "retract_fact", "summarize_context", "cross_reference", "rank_relevance", "diff_knowledge",
		},
		"device": {
			"list_devices", "read_sensor", "write_gpio", "read_gpio", "capture_camera", "stream_camera", "capture_audio", "play_audio", "print_document", "set_display", "bluetooth_scan", "bluetooth_connect",
		},
		"identity": {
			"get_current_user", "switch_user", "sudo", "issue_token", "revoke_token", "verify_token", "encrypt", "decrypt", "sign", "verify_signature", "store_secret", "retrieve_secret", "audit_log_read", "set_policy", "check_policy",
		},
		"time": {
			"schedule_once", "schedule_recurring", "cancel_schedule", "set_alarm", "set_deadline", "get_system_time", "set_system_time", "measure_duration", "defer_until",
		},
		"agent": {
			"spawn_agent", "kill_agent", "send_message", "broadcast", "request_approval", "delegate_task", "observe_agent", "merge_results", "escalate",
		},
		"ui": {
			"render_ui", "show_notification", "dismiss_notification", "prompt_user", "read_clipboard", "write_clipboard", "screenshot", "screen_record", "synthesize_speech", "transcribe_audio", "open_url", "navigate", "inject_input",
		},
		"code": {
			"run_shell", "eval_code", "compile", "link", "run_tests", "parse_test_results", "lint", "format", "diff_code", "patch_code", "search_code", "refactor",
		},
		"observability": {
			"read_logs", "get_metrics", "get_traces", "create_alert", "silence_alert", "profile_process", "explain_anomaly", "tail_stream",
		},
		"config": {
			"get_config", "set_config", "watch_config", "get_env", "set_env", "feature_flag_read", "feature_flag_set", "migrate_schema", "backup", "restore", "diff_config",
		},
		"external": {
			"call_llm", "query_database", "call_api", "read_email", "send_email", "post_message", "create_issue", "update_issue", "read_calendar", "create_event", "search_web",
		},
	}

	activeMappings := map[string]map[string]any{
		"filesystem.read_file":  {"gatewayToolId": "fs.read", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskLow, "lane": domain.ToolLaneIO},
		"filesystem.list_dir":   {"gatewayToolId": "fs.list", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskLow, "lane": domain.ToolLaneIO},
		"filesystem.write_file": {"gatewayToolId": "fs.write", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskMedium, "lane": domain.ToolLaneIO},
		"filesystem.move_file":  {"gatewayToolId": "fs.rename", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskMedium, "lane": domain.ToolLaneIO},
		"filesystem.delete_file": {
			"gatewayToolId": "fs.delete", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskCritical, "lane": domain.ToolLaneIO,
		},
		"filesystem.set_permissions": {"gatewayToolId": "fs.chmod", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskHigh, "lane": domain.ToolLaneIO},
		"process.spawn_process":      {"gatewayToolId": "proc.run", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskHigh, "lane": domain.ToolLaneIO},
		"process.kill_process":       {"gatewayToolId": "proc.terminate", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskCritical, "lane": domain.ToolLaneIO},
		"network.dns_resolve":        {"gatewayToolId": "net.dns_lookup", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskLow, "lane": domain.ToolLaneIO},
		"network.http_request":       {"gatewayToolId": "net.fetch", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskHigh, "lane": domain.ToolLaneIO},
		"network.scan_network":       {"gatewayToolId": "net.connectivity", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskHigh, "lane": domain.ToolLaneIO},
		"code.diff_code":             {"gatewayToolId": "git.diff", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskLow, "lane": domain.ToolLaneCompute},
		"code.run_shell":             {"gatewayToolId": "proc.run", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskHigh, "lane": domain.ToolLaneCompute},
		"observability.read_logs":    {"gatewayToolId": "system.logs", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskMedium, "lane": domain.ToolLaneIO},
		"identity.retrieve_secret":   {"gatewayToolId": "secret.get", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskHigh, "lane": domain.ToolLaneControl},
		"ui.show_notification":       {"gatewayToolId": "desktop.notify", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskLow, "lane": domain.ToolLaneIO},
		"ui.open_url":                {"gatewayToolId": "desktop.open", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskHigh, "lane": domain.ToolLaneIO},
		"time.get_system_time":       {"gatewayToolId": "time.now", "status": domain.ToolCapabilityActive, "risk": domain.ToolRiskNone, "lane": domain.ToolLaneIO},
	}

	out := make([]domain.ToolCapability, 0, 160)
	for domainID, primitives := range domainPrimitiveMap {
		for _, primitive := range primitives {
			id := domainID + "." + primitive
			capability := defaultCapabilityDescriptor(domainID, primitive)
			if override, ok := activeMappings[id]; ok {
				capability.Metadata = mergeToolMeta(capability.Metadata, override)
				if v, ok := override["status"].(domain.ToolCapabilityStatus); ok {
					capability.Status = v
				}
				if v, ok := override["risk"].(domain.ToolRisk); ok {
					capability.Risk = v
				}
				if v, ok := override["lane"].(domain.ToolLane); ok {
					capability.Lane = v
				}
				if v, ok := override["gatewayToolId"].(string); ok {
					capability.AdapterID = "gateway." + strings.ReplaceAll(v, ".", "_")
				}
			}
			out = append(out, capability)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func defaultCapabilityDescriptor(domainID, primitive string) domain.ToolCapability {
	id := strings.ToLower(strings.TrimSpace(domainID + "." + primitive))
	effects := inferCapabilityEffects(domainID, primitive)
	risk := inferCapabilityRisk(domainID, primitive, effects)
	status := domain.ToolCapabilityStubbed
	if risk.Rank() >= domain.ToolRiskHigh.Rank() {
		status = domain.ToolCapabilityApprovalOnly
	}
	return domain.ToolCapability{
		ID:                        id,
		Domain:                    strings.ToLower(strings.TrimSpace(domainID)),
		Name:                      primitive,
		Description:               fmt.Sprintf("%s capability %s", domainID, primitive),
		Status:                    status,
		Lane:                      inferCapabilityLane(domainID, effects),
		Effect:                    effects,
		Risk:                      risk,
		RequiresWorkspace:         domainID != "external" && domainID != "time",
		RequiresIntent:            false,
		RequiresApprovalByDefault: risk.Rank() >= domain.ToolRiskHigh.Rank(),
		AutonomyEligible:          risk.Rank() <= domain.ToolRiskMedium.Rank(),
		AllowedInDryRun:           true,
		RequiredCapabilities:      []string{},
		PolicyTags:                []string{domainID},
		ResourceCost: domain.ToolResourceCost{
			CostUnits: inferCostUnits(risk),
		},
		ResourceLimits: defaultResourceLimits(domainID, primitive, risk),
		InputSchema: map[string]any{
			"type": "object",
		},
		OutputSchema: map[string]any{
			"type": "object",
		},
		AuditLevel:       domain.ToolAuditBasic,
		ArtifactBehavior: defaultArtifactBehavior(effects),
		RollbackSupport:  risk.Rank() <= domain.ToolRiskMedium.Rank(),
		AdapterID:        "stub." + strings.ReplaceAll(id, ".", "_"),
		Metadata: map[string]any{
			"taxonomyVersion": "phase_5_9",
		},
	}
}

func inferCapabilityEffects(domainID, primitive string) []domain.ToolEffect {
	name := strings.ToLower(strings.TrimSpace(primitive))
	effects := []domain.ToolEffect{}
	add := func(effect domain.ToolEffect) {
		for _, existing := range effects {
			if existing == effect {
				return
			}
		}
		effects = append(effects, effect)
	}
	if hasAnyPrefix(name, "read_", "list_", "get_", "query_", "inspect_", "recall", "search_", "verify_", "measure_", "diff_") {
		add(domain.ToolEffectRead)
	}
	if hasAnyPrefix(name, "write_", "set_", "create_", "update_", "store_", "remember", "upsert_", "archive", "extract", "restore", "retract_", "forget", "cancel_") {
		add(domain.ToolEffectWrite)
	}
	if hasAnyPrefix(name, "run_", "spawn_", "kill_", "signal_", "compile", "link", "lint", "format", "refactor", "eval_", "patch_", "delegate_", "open_", "close_") {
		add(domain.ToolEffectExecute)
	}
	if hasAnyPrefix(name, "dns_", "http_", "open_socket", "proxy_", "open_tunnel", "scan_", "intercept_", "send_", "post_", "call_", "query_", "read_email", "search_web", "sync_to_", "sync_from_") {
		add(domain.ToolEffectNetwork)
	}
	if domainID == "external" || strings.Contains(name, "email") || strings.Contains(name, "calendar") || strings.Contains(name, "issue") || strings.Contains(name, "message") {
		add(domain.ToolEffectExternal)
	}
	if hasAnyPrefix(name, "sudo", "switch_user", "issue_token", "revoke_token", "store_secret", "retrieve_secret", "set_policy", "set_firewall", "delete_firewall", "set_permissions", "delete_file") {
		add(domain.ToolEffectPrivileged)
	}
	if hasAnyPrefix(name, "delete_", "kill_", "retract_", "forget", "restore_snapshot", "set_system_time", "inject_input", "intercept_traffic") {
		add(domain.ToolEffectDestructive)
	}
	if len(effects) == 0 {
		effects = append(effects, domain.ToolEffectRead)
	}
	return effects
}

func inferCapabilityRisk(domainID, primitive string, effects []domain.ToolEffect) domain.ToolRisk {
	risk := domain.ToolRiskLow
	for _, effect := range effects {
		switch effect {
		case domain.ToolEffectRead:
			if risk.Rank() < domain.ToolRiskLow.Rank() {
				risk = domain.ToolRiskLow
			}
		case domain.ToolEffectWrite:
			if risk.Rank() < domain.ToolRiskMedium.Rank() {
				risk = domain.ToolRiskMedium
			}
		case domain.ToolEffectExecute:
			if risk.Rank() < domain.ToolRiskHigh.Rank() {
				risk = domain.ToolRiskHigh
			}
		case domain.ToolEffectNetwork, domain.ToolEffectExternal, domain.ToolEffectPrivileged:
			if risk.Rank() < domain.ToolRiskHigh.Rank() {
				risk = domain.ToolRiskHigh
			}
		case domain.ToolEffectDestructive:
			if risk.Rank() < domain.ToolRiskCritical.Rank() {
				risk = domain.ToolRiskCritical
			}
		}
	}
	name := strings.ToLower(strings.TrimSpace(primitive))
	if strings.Contains(name, "delete") || strings.Contains(name, "kill") || strings.Contains(name, "sudo") || strings.Contains(name, "credential") {
		risk = domain.ToolRiskCritical
	}
	if domainID == "external" && risk.Rank() < domain.ToolRiskHigh.Rank() {
		risk = domain.ToolRiskHigh
	}
	if domainID == "time" && name == "get_system_time" {
		risk = domain.ToolRiskNone
	}
	return risk
}

func inferCapabilityLane(domainID string, effects []domain.ToolEffect) domain.ToolLane {
	if domainID == "memory" || domainID == "code" || domainID == "agent" {
		return domain.ToolLaneCompute
	}
	for _, effect := range effects {
		if effect == domain.ToolEffectPrivileged {
			return domain.ToolLaneControl
		}
	}
	return domain.ToolLaneIO
}

func defaultArtifactBehavior(effects []domain.ToolEffect) domain.ToolArtifactBehavior {
	for _, effect := range effects {
		if effect == domain.ToolEffectExecute || effect == domain.ToolEffectNetwork || effect == domain.ToolEffectExternal {
			return domain.ToolArtifactOptional
		}
	}
	return domain.ToolArtifactNone
}

func defaultResourceLimits(domainID, primitive string, risk domain.ToolRisk) domain.ToolResourceLimits {
	limits := domain.ToolResourceLimits{
		MaxDurationMs:  15000,
		MaxOutputBytes: 2 * 1024 * 1024,
	}
	if domainID == "filesystem" {
		limits.MaxDurationMs = 5000
	}
	if domainID == "network" || domainID == "external" {
		limits.MaxDurationMs = 10000
		limits.MaxOutputBytes = 512 * 1024
	}
	if risk.Rank() >= domain.ToolRiskHigh.Rank() {
		limits.MaxDurationMs = 8000
	}
	if strings.EqualFold(primitive, "run_tests") || strings.EqualFold(primitive, "lint") || strings.EqualFold(primitive, "format") {
		limits.MaxDurationMs = 120000
	}
	return limits
}

func inferCostUnits(risk domain.ToolRisk) int {
	switch risk {
	case domain.ToolRiskNone:
		return 0
	case domain.ToolRiskLow:
		return 1
	case domain.ToolRiskMedium:
		return 2
	case domain.ToolRiskHigh:
		return 5
	case domain.ToolRiskCritical:
		return 8
	default:
		return 3
	}
}

func hasAnyPrefix(v string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(v, prefix) {
			return true
		}
	}
	return false
}

func metadataString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	value, ok := meta[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

func mergeToolMeta(existing map[string]any, incoming map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range existing {
		out[key] = value
	}
	for key, value := range incoming {
		out[key] = value
	}
	return out
}
