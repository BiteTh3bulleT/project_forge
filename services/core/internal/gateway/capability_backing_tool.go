package gateway

import (
	"context"
	"database/sql"

	"forge/projectforge/services/core/internal/aios/domain"
)

type capabilityBackingTool struct {
	capability domain.ToolCapability
	toolID     string
	workspace  string
	dataDir    string
	db         *sql.DB
}

func (t *capabilityBackingTool) ID() string     { return t.toolID }
func (t *capabilityBackingTool) Domain() string { return t.capability.Domain }
func (t *capabilityBackingTool) Action() string { return t.capability.Name }
func (t *capabilityBackingTool) RiskClass() string {
	return gatewayRiskClassFromToolRisk(t.capability.Risk)
}
func (t *capabilityBackingTool) Description() string { return t.capability.Description }
func (t *capabilityBackingTool) Executes() bool {
	return capabilityHasEffect(t.capability, domain.ToolEffectExecute)
}
func (t *capabilityBackingTool) UsesNetwork() bool {
	return capabilityHasEffect(t.capability, domain.ToolEffectNetwork) || capabilityHasEffect(t.capability, domain.ToolEffectExternal)
}
func (t *capabilityBackingTool) WriteIntent() bool {
	return capabilityHasEffect(t.capability, domain.ToolEffectWrite) ||
		capabilityHasEffect(t.capability, domain.ToolEffectPrivileged) ||
		capabilityHasEffect(t.capability, domain.ToolEffectDestructive)
}
func (t *capabilityBackingTool) ExecutionLevel() string {
	if capabilityHasEffect(t.capability, domain.ToolEffectDestructive) {
		return "L4"
	}
	if capabilityHasEffect(t.capability, domain.ToolEffectPrivileged) {
		return "L3"
	}
	return executionLevelFromRisk(gatewayRiskClassFromToolRisk(t.capability.Risk))
}

func (t *capabilityBackingTool) Execute(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Domain {
	case "filesystem":
		return t.executeFilesystem(ctx, req)
	case "network":
		return t.executeNetwork(ctx, req)
	case "process":
		return t.executeProcess(ctx, req)
	case "code":
		return t.executeCode(ctx, req)
	case "identity":
		return t.executeIdentity(ctx, req)
	case "config":
		return t.executeConfig(ctx, req)
	case "observability":
		return t.executeObservability(ctx, req)
	case "ui":
		return t.executeUI(ctx, req)
	case "device":
		return t.executeDevice(ctx, req)
	case "time":
		return t.executeTime(ctx, req)
	case "external":
		return t.executeExternal(ctx, req)
	case "memory":
		return t.executeMemory(ctx, req)
	case "agent":
		return t.executeAgent(ctx, req)
	case "backup":
		return t.executeBackup(ctx, req)
	default:
		return capabilityOK("capability executed", map[string]any{
			"capabilityId": t.capability.ID,
			"toolId":       t.toolID,
		}), nil
	}
}
