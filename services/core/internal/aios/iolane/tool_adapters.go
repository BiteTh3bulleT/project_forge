package iolane

import (
	"context"

	"forge/projectforge/services/core/internal/aios/domain"
)

// Domain-specific tool adapters define execution boundaries per capability domain.
// Implementations must be invoked only through the gateway policy pipeline.
type ProcessToolAdapter interface {
	ExecuteProcessTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}

type FilesystemToolAdapter interface {
	ExecuteFilesystemTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}

type NetworkToolAdapter interface {
	ExecuteNetworkTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}

type MemoryToolAdapter interface {
	ExecuteMemoryTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}

type IdentityToolAdapter interface {
	ExecuteIdentityTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}

type TimeToolAdapter interface {
	ExecuteTimeTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}

type AgentToolAdapter interface {
	ExecuteAgentTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}

type UIToolAdapter interface {
	ExecuteUITool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}

type CodeToolAdapter interface {
	ExecuteCodeTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}

type ObservabilityToolAdapter interface {
	ExecuteObservabilityTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}

type ConfigToolAdapter interface {
	ExecuteConfigTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}

type ExternalIntegrationToolAdapter interface {
	ExecuteExternalTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error)
}
