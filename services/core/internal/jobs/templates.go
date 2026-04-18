package jobs

type Template struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	Description           string    `json:"description"`
	RequestedAction       string    `json:"requestedAction"`
	TargetAdapter         string    `json:"targetAdapter"`
	Capability            string    `json:"capability"`
	ExecutionBoundary     string    `json:"executionBoundary"`
	RiskClass             RiskClass `json:"riskClass"`
	WriteIntent           bool      `json:"writeIntent"`
	ApprovalRequired      bool      `json:"approvalRequired"`
	DefaultExecutionMode  string    `json:"defaultExecutionMode"`
	ExpectedArtifactTypes []string  `json:"expectedArtifactTypes"`
}

var templates = map[string]Template{
	"search_packet": {
		ID:                    "search_packet",
		Name:                  "Search Memory + Packet",
		Description:           "Retrieves relevant memory and materializes a versioned task packet.",
		RequestedAction:       "search.memory.packet",
		TargetAdapter:         "forge",
		Capability:            "packet_only",
		ExecutionBoundary:     "memory_retrieval",
		RiskClass:             RiskReadOnly,
		WriteIntent:           false,
		ApprovalRequired:      false,
		DefaultExecutionMode:  "packet_build",
		ExpectedArtifactTypes: []string{"task_packet", "job_result"},
	},
	"ollama_summary": {
		ID:                    "ollama_summary",
		Name:                  "Ollama Summary",
		Description:           "Builds packet and asks local Ollama for a concise summary.",
		RequestedAction:       "ollama.summary",
		TargetAdapter:         "ollama",
		Capability:            "generate_summary",
		ExecutionBoundary:     "reasoning",
		RiskClass:             RiskExternalReasoning,
		WriteIntent:           false,
		ApprovalRequired:      true,
		DefaultExecutionMode:  "reasoning",
		ExpectedArtifactTypes: []string{"task_packet", "adapter_output", "job_result"},
	},
	"plan_from_index": {
		ID:                    "plan_from_index",
		Name:                  "Plan From Index",
		Description:           "Uses local retrieval context and Ollama to draft an implementation plan.",
		RequestedAction:       "ollama.plan",
		TargetAdapter:         "ollama",
		Capability:            "draft_plan",
		ExecutionBoundary:     "reasoning",
		RiskClass:             RiskExternalReasoning,
		WriteIntent:           false,
		ApprovalRequired:      true,
		DefaultExecutionMode:  "reasoning",
		ExpectedArtifactTypes: []string{"task_packet", "adapter_output", "job_result"},
	},
	"prepare_codex_handoff": {
		ID:                    "prepare_codex_handoff",
		Name:                  "Prepare Codex Handoff",
		Description:           "Prepares bounded Codex handoff package with explicit scope and deliverable.",
		RequestedAction:       "codex.handoff.prepare",
		TargetAdapter:         "codex",
		Capability:            "prepare_handoff",
		ExecutionBoundary:     "write_proposal",
		RiskClass:             RiskWriteFiles,
		WriteIntent:           true,
		ApprovalRequired:      true,
		DefaultExecutionMode:  "handoff_prep",
		ExpectedArtifactTypes: []string{"task_packet", "agent_guidance", "adapter_output", "job_result"},
	},
	"prepare_claude_handoff": {
		ID:                    "prepare_claude_handoff",
		Name:                  "Prepare Claude Code Handoff",
		Description:           "Prepares bounded Claude Code handoff package with explicit scope and deliverable.",
		RequestedAction:       "claude_code.handoff.prepare",
		TargetAdapter:         "claude_code",
		Capability:            "prepare_handoff",
		ExecutionBoundary:     "write_proposal",
		RiskClass:             RiskWriteFiles,
		WriteIntent:           true,
		ApprovalRequired:      true,
		DefaultExecutionMode:  "handoff_prep",
		ExpectedArtifactTypes: []string{"task_packet", "agent_guidance", "adapter_output", "job_result"},
	},
	"reindex_sources": {
		ID:                    "reindex_sources",
		Name:                  "Re-index Sources",
		Description:           "Runs local re-index operation on all configured sources.",
		RequestedAction:       "sources.reindex",
		TargetAdapter:         "forge",
		Capability:            "reindex",
		ExecutionBoundary:     "command_execution",
		RiskClass:             RiskRunCommands,
		WriteIntent:           false,
		ApprovalRequired:      true,
		DefaultExecutionMode:  "command",
		ExpectedArtifactTypes: []string{"job_result"},
	},
	"safe_local_analysis": {
		ID:                    "safe_local_analysis",
		Name:                  "Safe Local Analysis",
		Description:           "Runs read-only analysis using local context and local model reasoning.",
		RequestedAction:       "analysis.safe_local",
		TargetAdapter:         "ollama",
		Capability:            "analysis",
		ExecutionBoundary:     "reasoning",
		RiskClass:             RiskExternalReasoning,
		WriteIntent:           false,
		ApprovalRequired:      true,
		DefaultExecutionMode:  "reasoning",
		ExpectedArtifactTypes: []string{"task_packet", "adapter_output", "job_result"},
	},
	"normalize_project_context": {
		ID:                    "normalize_project_context",
		Name:                  "Normalize Project Context",
		Description:           "Imports source context and regenerates AGENTS.md, CLAUDE.md, and FORGE briefing artifacts.",
		RequestedAction:       "project_context.normalize",
		TargetAdapter:         "forge",
		Capability:            "normalize_context",
		ExecutionBoundary:     "write_execution",
		RiskClass:             RiskWriteFiles,
		WriteIntent:           true,
		ApprovalRequired:      true,
		DefaultExecutionMode:  "context_normalization",
		ExpectedArtifactTypes: []string{"context_normalization", "agent_guidance", "job_result"},
	},
	"gateway_action": {
		ID:                    "gateway_action",
		Name:                  "Gateway Action",
		Description:           "Execute a governed local tool action through the central gateway.",
		RequestedAction:       "gateway.action",
		TargetAdapter:         "forge",
		Capability:            "gateway_action",
		ExecutionBoundary:     "command_execution",
		RiskClass:             RiskRunCommands,
		WriteIntent:           true,
		ApprovalRequired:      false,
		DefaultExecutionMode:  "governed_tool",
		ExpectedArtifactTypes: []string{"job_result"},
	},
}

func TemplateByID(id string) (Template, bool) {
	t, ok := templates[id]
	return t, ok
}

func ListTemplates() []Template {
	out := make([]Template, 0, len(templates))
	for _, t := range templates {
		out = append(out, t)
	}
	return out
}
