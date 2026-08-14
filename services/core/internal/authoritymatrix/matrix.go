package authoritymatrix

import "sort"

const (
	OwnerGateway       = "gateway"
	OwnerModelRuntime  = "modelruntime"
	OwnerControlLane   = "aios.controllane"
	OwnerMemory        = "memory"
	OwnerBackup        = "backup"
	OwnerHostBridge    = "hostbridge"
	OwnerForgeH        = "forgeh"
	OwnerSystemStatus  = "system_status"
	OwnerEmbedding     = "embeddings"
	OwnerRetrieval     = "retrieval"
	OwnerConfiguration = "configuration"

	GatewayStatusActive        = "active"
	GatewayStatusApprovalOnly  = "approval_only"
	GatewayStatusNotApplicable = "not_applicable"

	ApprovalNone                   = "none"
	ApprovalGatewayPolicy          = "gateway_policy"
	ApprovalControlLaneGate        = "control_lane_approval_gate"
	ApprovalModelRuntimeManagement = "modelruntime_management"
	ApprovalBackupRestore          = "backup_restore"
	ApprovalLegacyMemoryGate       = "legacy_memory_mutation_gate"

	VisibilityOperator    = "operator"
	VisibilitySystem      = "system"
	VisibilityDiagnostic  = "diagnostic"
	VisibilityEvidence    = "evidence"
	VisibilityUnavailable = "unavailable"

	StatusReal                  = "real"
	StatusPartialLiveValidation = "partial_live_validation"
	StatusReadOnly              = "read_only"
	StatusDiagnosticOnly        = "diagnostic_only"
	StatusCompatibility         = "compatibility"
	StatusLegacyGate            = "legacy_gate"
	StatusDeferred              = "deferred"
)

type Row struct {
	ID                      string `json:"id"`
	Surface                 string `json:"surface"`
	Method                  string `json:"method"`
	Route                   string `json:"route"`
	Action                  string `json:"action"`
	AuthorityOwner          string `json:"authorityOwner"`
	CapabilityID            string `json:"capabilityId"`
	GatewayCapabilityStatus string `json:"gatewayCapabilityStatus"`
	Mutating                bool   `json:"mutating"`
	Destructive             bool   `json:"destructive"`
	RequiresApproval        bool   `json:"requiresApproval"`
	ApprovalMechanism       string `json:"approvalMechanism"`
	AuditCategory           string `json:"auditCategory"`
	AuditAction             string `json:"auditAction"`
	ResponseVisibility      string `json:"responseVisibility"`
	LiveAuthority           bool   `json:"liveAuthority"`
	ForgeKAuthority         bool   `json:"forgeKAuthority"`
	HostMutation            bool   `json:"hostMutation"`
	ModelRuntimeMutation    bool   `json:"modelruntimeMutation"`
	SemanticMemoryWrite     bool   `json:"semanticMemoryWrite"`
	Status                  string `json:"status"`
	Notes                   string `json:"notes"`
}

func DefaultRows() []Row {
	rows := []Row{
		gatewayInvokeRow(),
		gatewayCapabilityStatusRow(),

		modelRow("model.list", "GET", "/forge/models", "model.runtime.list", false, false, false, ApprovalNone, "model.runtime.list", StatusReal, "Lists governed model registry records through modelruntime."),
		modelRow("model.import", "POST", "/forge/models/import", "model.runtime.import", true, false, true, ApprovalModelRuntimeManagement, "model.runtime.import", StatusReal, "Imports/registers model metadata through modelruntime management governance; high-risk imports require approval."),
		modelRow("model.scan", "POST", "/forge/models/scan", "model.runtime.scan", true, false, false, ApprovalModelRuntimeManagement, "model.runtime.scan", StatusReal, "Refreshes model registry view; actor, source, and model.management capability are required."),
		modelRow("model.get", "GET", "/forge/models/{id}", "model.runtime.get", false, false, false, ApprovalNone, "model.runtime.get", StatusReal, "Reads a single governed model record."),
		modelRow("model.compatibility", "GET", "/forge/models/{id}/compatibility", "model.runtime.compatibility", false, false, false, ApprovalNone, "model.runtime.compatibility", StatusReal, "Reports compatibility and backend readiness; no lifecycle mutation."),
		modelRow("model.verify", "POST", "/forge/models/{id}/verify", "model.runtime.verify", true, false, false, ApprovalModelRuntimeManagement, "model.runtime.verify", StatusReal, "Verifies model metadata through modelruntime governance."),
		modelRow("model.enable", "POST", "/forge/models/{id}/enable", "model.runtime.enable", true, false, false, ApprovalModelRuntimeManagement, "model.runtime.enable", StatusReal, "Enables model selection state; approval required when policy classifies the operation high-risk."),
		modelRow("model.disable", "POST", "/forge/models/{id}/disable", "model.runtime.disable", true, false, false, ApprovalModelRuntimeManagement, "model.runtime.disable", StatusReal, "Disables model selection state; actor/source/capability are required."),
		modelRow("model.archive", "POST", "/forge/models/{id}/archive", "model.runtime.archive", true, true, true, ApprovalModelRuntimeManagement, "model.runtime.archive", StatusReal, "Archives model registry state through high-risk modelruntime governance."),
		modelRow("model.remove_registration", "POST", "/forge/models/{id}/remove", "model.remove_registration", true, false, true, ApprovalModelRuntimeManagement, "model.runtime.remove_registration", StatusReal, "Removes modelruntime registration through approval-governed management; this is not destructive model file deletion."),
		modelDeleteFileRow(),
		modelRow("model.load", "POST", "/forge/models/{id}/load", "model.runtime.load", true, false, true, ApprovalModelRuntimeManagement, "model.runtime.load", StatusReal, "Loads a model through modelruntime lifecycle governance; high-risk operation requires approval."),
		modelRow("model.unload", "POST", "/forge/models/{id}/unload", "model.runtime.unload", true, false, true, ApprovalModelRuntimeManagement, "model.runtime.unload", StatusReal, "Unloads a model through modelruntime lifecycle governance; high-risk operation requires approval."),
		modelChatRow(),
		modelStatusRow("modelruntime.backends", "/forge/model-runtime/backends", "model.runtime.backends", "Backend health/configuration summary."),
		modelStatusRow("modelruntime.usage", "/forge/model-runtime/usage", "model.runtime.usage", "Model registry, loaded, queue, and backend usage summary."),
		modelStatusRow("modelruntime.health", "/forge/model-runtime/health", "model.runtime.health", "Modelruntime health and policy warnings."),
		modelStatusRow("modelruntime.queue", "/forge/model-runtime/queue", "model.runtime.queue", "Scheduler queue status."),
		modelStatusRow("modelruntime.loaded", "/forge/model-runtime/loaded", "model.runtime.loaded", "Loaded model status."),
		openAIModelsRow(),
		modelGenerateRow(),

		controlLaneValidationRow("controllane.validate_kv_identity", "VALIDATE_KV_IDENTITY", "kv_identity"),
		controlLaneValidationRow("controllane.validate_ref_shape", "VALIDATE_REF_SHAPE", "ref_shape"),
		controlLaneValidationRow("controllane.compare_ref_shape", "COMPARE_REF_SHAPE", "ref_shape_comparison"),
		controlLaneValidationRow("controllane.validate_source_object_authority", "VALIDATE_SOURCE_OBJECT_AUTHORITY", "source_object_authority"),
		controlLaneValidationRow("controllane.validate_semantic_operation", "VALIDATE_SEMANTIC_OPERATION", "semantic_operation"),

		memoryRow("memory.observations.read", "GET", "/api/memory/observations", "memory.observations.list", false, false, ApprovalNone, "Reads legacy memory observation surfaces; output is evidence, not automatic truth."),
		memoryRow("memory.observations.write", "POST/PATCH", "/api/memory/observations*", "legacy.memory.observation.mutate", true, true, ApprovalLegacyMemoryGate, "Legacy memory mutation endpoints are retired and audited; canonical semantic writes must use Courthouse review and Control Lane syscall paths."),
		memoryRow("memory.retrieval.read", "GET", "/api/retrieval/runs*", "memory.retrieval.read", false, false, ApprovalNone, "Retrieval surfaces read evidence/search results and do not own truth mutation."),
		memoryRow("memory.retrieval.write", "POST", "/api/retrieval/runs", "memory.retrieval.run", true, false, ApprovalNone, "Creates retrieval run evidence; Qdrant/search output is not canonical truth."),
		memoryRow("memory.embeddings.status", "GET", "/api/embeddings/status", "embeddings.status", false, false, ApprovalNone, "Embedding provider status is diagnostic and not truth authority."),
		embeddingReembedRow(),

		backupCreateRow(),
		backupRestoreRow(),
		backupDeleteRow(),
		hostBridgeDiagnosticsRow(),
		forgeHPostureRow(),
		forgeHProposalRow(),
		systemStatusRow(),
		healthRow(),
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

func ByID(rows []Row) map[string]Row {
	out := make(map[string]Row, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

func gatewayInvokeRow() Row {
	return Row{
		ID: "gateway.invoke", Surface: "api.gateway", Method: "POST", Route: "/api/gateway/invoke", Action: "gateway.tool.execute",
		AuthorityOwner: OwnerGateway, CapabilityID: "gateway.tool.execute", GatewayCapabilityStatus: GatewayStatusActive,
		Mutating: true, ApprovalMechanism: ApprovalGatewayPolicy, AuditCategory: "gateway", AuditAction: "gateway.tool.invoked",
		ResponseVisibility: VisibilityEvidence, LiveAuthority: true, Status: StatusReal,
		Notes: "Gateway is the live owner for tool execution; per-tool capabilities, lanes, approvals, permissions, and audit decide concrete effects.",
	}
}

func gatewayCapabilityStatusRow() Row {
	return Row{
		ID: "gateway.capability_status", Surface: "api.gateway", Method: "PATCH", Route: "/api/gateway/capabilities/{id}/status", Action: "gateway.capability.status.update",
		AuthorityOwner: OwnerGateway, CapabilityID: "gateway.capability.status.update", GatewayCapabilityStatus: GatewayStatusApprovalOnly,
		Mutating: true, RequiresApproval: true, ApprovalMechanism: ApprovalGatewayPolicy, AuditCategory: "gateway", AuditAction: "tool.capability.status.updated",
		ResponseVisibility: VisibilityOperator, LiveAuthority: true, Status: StatusReal,
		Notes: "Operator-visible capability status governance; high-risk transitions require approval fingerprint validation.",
	}
}

func modelRow(id, method, route, action string, mutating, destructive, approval bool, approvalMechanism, auditAction, status, notes string) Row {
	gatewayStatus := GatewayStatusNotApplicable
	return Row{
		ID: id, Surface: "api.modelruntime", Method: method, Route: route, Action: action,
		AuthorityOwner: OwnerModelRuntime, CapabilityID: capabilityForModelRow(id), GatewayCapabilityStatus: gatewayStatus,
		Mutating: mutating, Destructive: destructive, RequiresApproval: approval, ApprovalMechanism: approvalMechanism,
		AuditCategory: "model_runtime", AuditAction: auditAction, ResponseVisibility: VisibilityOperator,
		LiveAuthority: true, ModelRuntimeMutation: mutating, Status: status, Notes: notes,
	}
}

func capabilityForModelRow(id string) string {
	if id == "model.chat" || id == "model.generate" {
		return id
	}
	return "model.management"
}

func modelDeleteFileRow() Row {
	return Row{
		ID: "model.delete_file", Surface: "api.modelruntime", Method: "POST", Route: "/forge/models/{id}/delete-file", Action: "model.delete_file",
		AuthorityOwner: OwnerModelRuntime, CapabilityID: "model.delete_file", GatewayCapabilityStatus: GatewayStatusApprovalOnly,
		Mutating: true, Destructive: true, RequiresApproval: true, ApprovalMechanism: ApprovalModelRuntimeManagement,
		AuditCategory: "model_runtime", AuditAction: "model.runtime.delete_file", ResponseVisibility: VisibilityOperator,
		LiveAuthority: true, ModelRuntimeMutation: true, Status: StatusReal,
		Notes: "Deletes a managed model artifact through approval-governed modelruntime management while preserving the registration as unavailable evidence.",
	}
}

func modelChatRow() Row {
	return Row{
		ID: "model.chat", Surface: "api.modelruntime", Method: "POST", Route: "/forge/models/{id}/chat", Action: "model.chat",
		AuthorityOwner: OwnerModelRuntime, CapabilityID: "model.chat", GatewayCapabilityStatus: GatewayStatusNotApplicable,
		ApprovalMechanism: ApprovalNone, AuditCategory: "model_runtime", AuditAction: "model.runtime.chat",
		ResponseVisibility: VisibilityOperator, LiveAuthority: true, Status: StatusReal,
		Notes: "Modelruntime owns chat execution governance; Gateway tool capability status is not implied for inference.",
	}
}

func modelGenerateRow() Row {
	return Row{
		ID: "model.generate", Surface: "api.openai_compat", Method: "POST", Route: "/v1/chat/completions", Action: "model.generate",
		AuthorityOwner: OwnerModelRuntime, CapabilityID: "model.generate", GatewayCapabilityStatus: GatewayStatusNotApplicable,
		ApprovalMechanism: ApprovalNone, AuditCategory: "model_runtime", AuditAction: "v1.chat.completions",
		ResponseVisibility: VisibilityOperator, LiveAuthority: true, Status: StatusCompatibility,
		Notes: "OpenAI-compatible chat completions are governed by modelruntime; model output is evidence/proposal text, not canonical truth.",
	}
}

func modelStatusRow(id, route, action, notes string) Row {
	return Row{
		ID: id, Surface: "api.modelruntime", Method: "GET", Route: route, Action: action,
		AuthorityOwner: OwnerModelRuntime, CapabilityID: action, GatewayCapabilityStatus: GatewayStatusNotApplicable,
		ApprovalMechanism: ApprovalNone, AuditCategory: "model_runtime", AuditAction: action,
		ResponseVisibility: VisibilityDiagnostic, LiveAuthority: true, Status: StatusReadOnly, Notes: notes,
	}
}

func openAIModelsRow() Row {
	return Row{
		ID: "openai.models", Surface: "api.openai_compat", Method: "GET", Route: "/v1/models", Action: "model.runtime.list",
		AuthorityOwner: OwnerModelRuntime, CapabilityID: "model.runtime.list", GatewayCapabilityStatus: GatewayStatusNotApplicable,
		ApprovalMechanism: ApprovalNone, AuditCategory: "model_runtime", AuditAction: "v1.models",
		ResponseVisibility: VisibilityOperator, LiveAuthority: true, Status: StatusCompatibility,
		Notes: "OpenAI-compatible model listing mirrors governed modelruntime registry visibility.",
	}
}

func controlLaneValidationRow(id, action, auditAction string) Row {
	return Row{
		ID: id, Surface: "aios.controllane", Method: "SYSCALL", Route: "internal://aios/controllane", Action: action,
		AuthorityOwner: OwnerControlLane, CapabilityID: "semantic.validation", GatewayCapabilityStatus: GatewayStatusNotApplicable,
		ApprovalMechanism: ApprovalNone, AuditCategory: "semantic_syscall", AuditAction: auditAction,
		ResponseVisibility: VisibilityDiagnostic, LiveAuthority: true, ForgeKAuthority: false, Status: StatusPartialLiveValidation,
		Notes: "Partial live validation seam using shared deterministic contracts only; no FORGE-K simulator service authority, semantic execution, retrieval, modelruntime call, or memory mutation.",
	}
}

func memoryRow(id, method, route, action string, mutating, semanticWrite bool, approvalMechanism, notes string) Row {
	return Row{
		ID: id, Surface: "api.memory", Method: method, Route: route, Action: action,
		AuthorityOwner: OwnerMemory, CapabilityID: action, GatewayCapabilityStatus: GatewayStatusNotApplicable,
		Mutating: mutating, RequiresApproval: approvalMechanism != ApprovalNone, ApprovalMechanism: approvalMechanism,
		AuditCategory: "memory", AuditAction: action, ResponseVisibility: VisibilityEvidence,
		LiveAuthority: true, SemanticMemoryWrite: semanticWrite, Status: StatusLegacyGate, Notes: notes,
	}
}

func embeddingReembedRow() Row {
	return Row{
		ID: "memory.embeddings.reembed", Surface: "api.memory", Method: "POST", Route: "/api/embeddings/reembed", Action: "embeddings.reembed",
		AuthorityOwner: OwnerEmbedding, CapabilityID: "memory.embed_content", GatewayCapabilityStatus: GatewayStatusNotApplicable,
		Mutating: true, ApprovalMechanism: ApprovalNone, AuditCategory: "embeddings", AuditAction: "embeddings.reembed",
		ResponseVisibility: VisibilityEvidence, LiveAuthority: true, Status: StatusReal,
		Notes: "Regenerates derived embedding evidence; embeddings are not canonical memory truth.",
	}
}

func backupCreateRow() Row {
	return Row{
		ID: "backup.create", Surface: "api.backup", Method: "POST", Route: "/api/backup/bundles", Action: "backup.bundle.create",
		AuthorityOwner: OwnerBackup, CapabilityID: "config.backup", GatewayCapabilityStatus: GatewayStatusApprovalOnly,
		Mutating: true, ApprovalMechanism: ApprovalNone, AuditCategory: "backup", AuditAction: "backup.bundle.create",
		ResponseVisibility: VisibilityOperator, LiveAuthority: true, Status: StatusReal,
		Notes: "Creates backup/export artifact records; does not restore or mutate live semantic truth.",
	}
}

func backupRestoreRow() Row {
	return Row{
		ID: "backup.restore", Surface: "api.backup", Method: "POST", Route: "/api/backup/restore", Action: "backup.restore.inspect",
		AuthorityOwner: OwnerBackup, CapabilityID: "backup.restore", GatewayCapabilityStatus: GatewayStatusNotApplicable,
		Mutating: false, Destructive: false, RequiresApproval: false, ApprovalMechanism: ApprovalNone,
		AuditCategory: "backup", AuditAction: "backup.bundle.restore_inspected", ResponseVisibility: VisibilityOperator,
		LiveAuthority: true, SemanticMemoryWrite: false, Status: StatusReadOnly,
		Notes: "Dry-run inspection only; every non-dry request fails closed with FORGE_K_RESTORE_APPLY_DISABLED before approval or mutation.",
	}
}

func backupDeleteRow() Row {
	return Row{
		ID: "backup.delete", Surface: "api.backup", Method: "DELETE", Route: "/api/backup/bundles/{id}", Action: "backup.bundle.delete",
		AuthorityOwner: OwnerBackup, CapabilityID: "config.backup", GatewayCapabilityStatus: GatewayStatusApprovalOnly,
		Mutating: true, Destructive: true, ApprovalMechanism: ApprovalNone, AuditCategory: "backup", AuditAction: "backup.bundle.delete",
		ResponseVisibility: VisibilityOperator, LiveAuthority: true, Status: StatusReal,
		Notes: "Deletes backup bundle artifact metadata/files; does not grant restore authority.",
	}
}

func hostBridgeDiagnosticsRow() Row {
	return Row{
		ID: "hostbridge.diagnostics", Surface: "api.system_status", Method: "GET", Route: "/forge/system/status", Action: "hostbridge.diagnostics.read",
		AuthorityOwner: OwnerHostBridge, CapabilityID: "observability.get_metrics", GatewayCapabilityStatus: GatewayStatusNotApplicable,
		ApprovalMechanism: ApprovalNone, AuditCategory: "system_status", AuditAction: "hostbridge.diagnostics",
		ResponseVisibility: VisibilityDiagnostic, LiveAuthority: true, Status: StatusDiagnosticOnly,
		Notes: "Read-only HostBridge snapshot; command-backed probes are disabled for the shell system status surface.",
	}
}

func forgeHPostureRow() Row {
	return Row{
		ID: "forgeh.posture", Surface: "api.system_status", Method: "GET", Route: "/forge/system/status", Action: "forgeh.resource_posture.evaluate",
		AuthorityOwner: OwnerForgeH, CapabilityID: "observability.get_metrics", GatewayCapabilityStatus: GatewayStatusNotApplicable,
		ApprovalMechanism: ApprovalNone, AuditCategory: "system_status", AuditAction: "forgeh.posture",
		ResponseVisibility: VisibilityDiagnostic, LiveAuthority: true, Status: StatusDiagnosticOnly,
		Notes: "FORGE-H evaluates host posture from HostBridge diagnostics; advisory only and no host mutation.",
	}
}

func forgeHProposalRow() Row {
	return Row{
		ID: "forgeh.proposals", Surface: "api.system_status", Method: "GET", Route: "/forge/system/status", Action: "forgeh.resource_action.propose",
		AuthorityOwner: OwnerForgeH, CapabilityID: "agent.delegate_task", GatewayCapabilityStatus: GatewayStatusNotApplicable,
		ApprovalMechanism: ApprovalNone, AuditCategory: "system_status", AuditAction: "forgeh.proposals",
		ResponseVisibility: VisibilityDiagnostic, LiveAuthority: true, Status: StatusDiagnosticOnly,
		Notes: "FORGE-H emits bounded advisory proposals only; proposal generation does not approve, execute, or commit actions.",
	}
}

func systemStatusRow() Row {
	return Row{
		ID: "system.status", Surface: "api.system_status", Method: "GET", Route: "/forge/system/status", Action: "system.status.read",
		AuthorityOwner: OwnerSystemStatus, CapabilityID: "observability.get_metrics", GatewayCapabilityStatus: GatewayStatusNotApplicable,
		ApprovalMechanism: ApprovalNone, AuditCategory: "system_status", AuditAction: "system.status",
		ResponseVisibility: VisibilityDiagnostic, LiveAuthority: true, Status: StatusReadOnly,
		Notes: "System cockpit/status aggregation is read-only; no mutation buttons or canonical writes.",
	}
}

func healthRow() Row {
	return Row{
		ID: "system.health", Surface: "api.health", Method: "GET", Route: "/health", Action: "system.health.read",
		AuthorityOwner: OwnerSystemStatus, CapabilityID: "observability.get_metrics", GatewayCapabilityStatus: GatewayStatusNotApplicable,
		ApprovalMechanism: ApprovalNone, AuditCategory: "health", AuditAction: "health",
		ResponseVisibility: VisibilityDiagnostic, LiveAuthority: true, Status: StatusReadOnly,
		Notes: "Core health endpoint returns status, safe-mode, modelruntime, GPU, and embedding diagnostics only.",
	}
}
