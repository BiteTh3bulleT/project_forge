package iolane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

type adapterStub struct{}

func (adapterStub) ExecuteProcessTool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}
func (adapterStub) ExecuteFilesystemTool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}
func (adapterStub) ExecuteNetworkTool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}
func (adapterStub) ExecuteMemoryTool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}
func (adapterStub) ExecuteIdentityTool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}
func (adapterStub) ExecuteTimeTool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}
func (adapterStub) ExecuteAgentTool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}
func (adapterStub) ExecuteUITool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}
func (adapterStub) ExecuteCodeTool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}
func (adapterStub) ExecuteObservabilityTool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}
func (adapterStub) ExecuteConfigTool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}
func (adapterStub) ExecuteExternalTool(_ context.Context, _ domain.ToolRequest) (domain.ToolResult, error) {
	return domain.ToolResult{Success: true, Status: domain.ToolStatusSucceeded}, nil
}

func TestToolAdapterInterfacesCompile(t *testing.T) {
	t.Parallel()
	var stub adapterStub
	var _ ProcessToolAdapter = stub
	var _ FilesystemToolAdapter = stub
	var _ NetworkToolAdapter = stub
	var _ MemoryToolAdapter = stub
	var _ IdentityToolAdapter = stub
	var _ TimeToolAdapter = stub
	var _ AgentToolAdapter = stub
	var _ UIToolAdapter = stub
	var _ CodeToolAdapter = stub
	var _ ObservabilityToolAdapter = stub
	var _ ConfigToolAdapter = stub
	var _ ExternalIntegrationToolAdapter = stub
}
