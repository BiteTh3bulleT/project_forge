package gateway

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/approvals"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/lanes"
	"forge/projectforge/services/core/internal/permissions"
	"forge/projectforge/services/core/internal/store"
)

type stubAutonomyAuthorizer struct {
	decision ToolAutonomyDecision
	err      error
	calls    int
}

func (s *stubAutonomyAuthorizer) AuthorizeToolRequest(_ context.Context, _ ToolAutonomyRequest) (ToolAutonomyDecision, error) {
	s.calls++
	if s.err != nil {
		return ToolAutonomyDecision{}, s.err
	}
	return s.decision, nil
}

func TestToolCapabilityRegistryCoverageAndDuplicateRejection(t *testing.T) {
	t.Parallel()
	reg := NewToolCapabilityRegistry()
	rows := reg.List()
	if len(rows) < 130 {
		t.Fatalf("expected full taxonomy registration, got %d", len(rows))
	}
	if _, ok := reg.Get("filesystem.read_file"); !ok {
		t.Fatalf("expected filesystem.read_file capability to exist")
	}
	if _, ok := reg.Resolve("fs.read"); !ok {
		t.Fatalf("expected fs.read legacy mapping to resolve")
	}
	if len(reg.ListByDomain("network")) == 0 {
		t.Fatalf("expected network capabilities")
	}
	if len(reg.ListByRisk(domain.ToolRiskCritical)) == 0 {
		t.Fatalf("expected critical-risk capabilities")
	}
	err := reg.Register(domain.ToolCapability{
		ID:          "filesystem.read_file",
		Domain:      "filesystem",
		Name:        "read_file",
		Description: "duplicate",
		Status:      domain.ToolCapabilityActive,
		Lane:        domain.ToolLaneIO,
		Effect:      []domain.ToolEffect{domain.ToolEffectRead},
		Risk:        domain.ToolRiskLow,
	})
	if err == nil {
		t.Fatalf("expected duplicate registration to fail")
	}
}

func TestToolPolicyEvaluatorCoreDecisions(t *testing.T) {
	t.Parallel()
	evaluator := NewToolPolicyEvaluator("", DeterministicToolRiskClassifier{}, nil)
	req := Request{
		ToolID:              "filesystem.read_file",
		WorkspaceID:         "workspace:test",
		Source:              "user",
		ProvenanceActor:     "tester",
		ProvenanceActorType: "test",
	}
	disabled := domain.ToolCapability{
		ID:                "filesystem.read_file",
		Domain:            "filesystem",
		Name:              "read_file",
		Description:       "read",
		Status:            domain.ToolCapabilityDisabled,
		Lane:              domain.ToolLaneIO,
		Effect:            []domain.ToolEffect{domain.ToolEffectRead},
		Risk:              domain.ToolRiskLow,
		RequiresWorkspace: true,
		AllowedInDryRun:   true,
	}
	disabledDecision := evaluator.Evaluate(context.Background(), ToolPolicyInput{
		Request:       req,
		Capability:    disabled,
		ResolvedPaths: []string{"C:\\tmp\\sample.txt"},
		HasAdapter:    true,
	})
	if disabledDecision.Status != StatusDisabled {
		t.Fatalf("expected disabled status, got %s", disabledDecision.Status)
	}

	approvalOnly := disabled
	approvalOnly.Status = domain.ToolCapabilityApprovalOnly
	approvalOnly.Risk = domain.ToolRiskCritical
	approvalOnlyDecision := evaluator.Evaluate(context.Background(), ToolPolicyInput{
		Request:       req,
		Capability:    approvalOnly,
		ResolvedPaths: nil,
		HasAdapter:    true,
	})
	if approvalOnlyDecision.Status != StatusNeedsApprov {
		t.Fatalf("expected approval-required status, got %s", approvalOnlyDecision.Status)
	}

	stubbed := disabled
	stubbed.Status = domain.ToolCapabilityStubbed
	stubbedDecision := evaluator.Evaluate(context.Background(), ToolPolicyInput{
		Request:       req,
		Capability:    stubbed,
		ResolvedPaths: nil,
		HasAdapter:    false,
	})
	if stubbedDecision.Status != StatusUnsupported {
		t.Fatalf("expected unsupported status, got %s", stubbedDecision.Status)
	}

	irisReq := req
	irisReq.Source = string(domain.SourceFutureIRIS)
	irisReq.IntentID = ""
	irisDecision := evaluator.Evaluate(context.Background(), ToolPolicyInput{
		Request:       irisReq,
		Capability:    disabled,
		ResolvedPaths: nil,
		HasAdapter:    true,
	})
	if irisDecision.Status != StatusDisabled {
		// Disabled status blocks first; validate source behavior with active capability.
	}
	active := disabled
	active.Status = domain.ToolCapabilityActive
	activeDecision := evaluator.Evaluate(context.Background(), ToolPolicyInput{
		Request:       irisReq,
		Capability:    active,
		ResolvedPaths: nil,
		HasAdapter:    true,
	})
	if activeDecision.Status != StatusDenied {
		t.Fatalf("expected self-initiated request without intent to deny, got %s", activeDecision.Status)
	}
}

func TestGatewayToolSurfacePolicyAndDryRun(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)

	sample := filepath.Join(workspace, "sample.txt")
	if err := os.WriteFile(sample, []byte("ok"), 0o644); err != nil {
		t.Fatalf("seed sample: %v", err)
	}

	disabledCapability, ok := gw.capabilities.Get("filesystem.read_file")
	if !ok {
		t.Fatalf("missing capability filesystem.read_file")
	}
	disabledCapability.Status = domain.ToolCapabilityDisabled
	if _, ok := gw.capabilities.UpdateStatus(disabledCapability.ID, domain.ToolCapabilityDisabled); !ok {
		t.Fatalf("failed to disable capability")
	}
	disabledRes, err := gw.Execute(ctx, Request{
		ToolID:              "filesystem.read_file",
		LaneID:              "fs.read",
		CorrelationID:       "corr-disabled",
		Source:              "user",
		WorkspaceID:         "workspace:test",
		ProvenanceActor:     "tester",
		ProvenanceActorType: "test",
		Paths:               []string{"sample.txt"},
		Initiator:           "tester",
	})
	if err != nil {
		t.Fatalf("disabled execute error: %v", err)
	}
	if disabledRes.Status != StatusDisabled {
		t.Fatalf("expected disabled status, got %s", disabledRes.Status)
	}
	if _, ok := gw.capabilities.UpdateStatus(disabledCapability.ID, domain.ToolCapabilityActive); !ok {
		t.Fatalf("failed to re-enable capability")
	}

	writePath := filepath.Join(workspace, "scratch", "dry-run-dir")
	writeRes, err := gw.Execute(ctx, Request{
		ToolID:              "fs.mkdir",
		LaneID:              "fs.mkdir",
		CorrelationID:       "corr-dry-run",
		Source:              "user",
		WorkspaceID:         "workspace:test",
		ProvenanceActor:     "tester",
		ProvenanceActorType: "test",
		Paths:               []string{"scratch/dry-run-dir"},
		DryRun:              true,
		Initiator:           "tester",
	})
	if err != nil {
		t.Fatalf("dry-run execute error: %v", err)
	}
	if writeRes.Status != StatusDryRun {
		t.Fatalf("expected dry_run status, got %s", writeRes.Status)
	}
	if _, err := os.Stat(writePath); err == nil {
		t.Fatalf("expected dry-run to avoid writing file")
	}

	var deniedAuditCount int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM audit_records WHERE action IN ('tool.disabled', 'tool.denied')`).Scan(&deniedAuditCount); err != nil {
		t.Fatalf("query audit records: %v", err)
	}
	if deniedAuditCount == 0 {
		t.Fatalf("expected audit record for blocked execution")
	}
}

func TestGatewayAutonomyPolicyIntegration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	authz := &stubAutonomyAuthorizer{
		decision: ToolAutonomyDecision{
			Allowed:          false,
			RequiresApproval: true,
			Reason:           "budget exhausted",
		},
	}
	gw, _, _ := newToolSurfaceGatewayHarness(t, authz)

	res, err := gw.Execute(ctx, Request{
		ToolID:              "filesystem.read_file",
		LaneID:              "fs.read",
		CorrelationID:       "corr-autonomy-budget",
		Source:              string(domain.SourceSystem),
		WorkspaceID:         "workspace:test",
		IntentID:            "intent-1",
		CharterID:           "charter-1",
		BudgetID:            "budget-1",
		ProvenanceActor:     "forge.autonomy",
		ProvenanceActorType: "system",
		Paths:               []string{"sample.txt"},
		Initiator:           "forge.autonomy",
	})
	if err != nil {
		t.Fatalf("autonomy execute: %v", err)
	}
	if res.Status != StatusNeedsApprov {
		t.Fatalf("expected needs_approval for blocked budget, got %s", res.Status)
	}
	if authz.calls == 0 {
		t.Fatalf("expected autonomy authorizer to be called")
	}
}

func TestGatewayScopeBoundaryFromCapabilityLimits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, _, workspace := newToolSurfaceGatewayHarness(t)

	capability, ok := gw.capabilities.Get("filesystem.read_file")
	if !ok {
		t.Fatalf("missing capability")
	}
	capability.ResourceLimits.AllowedPaths = []string{filepath.Join(workspace, "allowed")}
	gw.capabilities.mu.Lock()
	gw.capabilities.byID[capability.ID] = capability
	gw.capabilities.mu.Unlock()

	if err := os.WriteFile(filepath.Join(workspace, "outside.txt"), []byte("outside"), 0o644); err != nil {
		t.Fatalf("seed outside file: %v", err)
	}
	res, err := gw.Execute(ctx, Request{
		ToolID:              "filesystem.read_file",
		LaneID:              "fs.read",
		CorrelationID:       "corr-scope-boundary",
		Source:              "user",
		WorkspaceID:         "workspace:test",
		ProvenanceActor:     "tester",
		ProvenanceActorType: "test",
		Paths:               []string{"outside.txt"},
		Initiator:           "tester",
	})
	if err != nil {
		t.Fatalf("scope boundary execute: %v", err)
	}
	if res.Status != StatusDenied {
		t.Fatalf("expected denied for scope boundary, got %s", res.Status)
	}
	if !strings.Contains(strings.ToLower(res.DeniedReason), "resource limits") {
		t.Fatalf("expected resource-limit denial reason, got %q", res.DeniedReason)
	}
}

func newToolSurfaceGatewayHarness(t *testing.T, authorizer ...ToolAutonomyAuthorizer) (*Gateway, *store.Store, string) {
	t.Helper()
	ctx := context.Background()
	workspace := t.TempDir()
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "sample.txt"), []byte("sample"), 0o644); err != nil {
		t.Fatalf("seed sample file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "scratch"), 0o755); err != nil {
		t.Fatalf("mkdir scratch: %v", err)
	}
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	laneSvc := lanes.New(st.DB)
	if err := laneSvc.EnsureDefaults(ctx, workspace); err != nil {
		t.Fatalf("ensure lanes: %v", err)
	}
	permSvc := permissions.New(st.DB)
	if err := permSvc.EnsureDefaults(ctx, workspace); err != nil {
		t.Fatalf("ensure permissions: %v", err)
	}

	gw := New(Options{
		DB:                 st.DB,
		Permissions:        permSvc,
		Lanes:              laneSvc,
		Approvals:          approvals.New(st.DB),
		Audit:              audit.New(st.DB),
		WorkspaceDir:       workspace,
		DataDir:            dataDir,
		AutonomyPolicy:     firstAuthorizer(authorizer),
		CapabilityRegistry: NewToolCapabilityRegistry(),
	})
	toolIDs := make([]string, 0, len(gw.Tools()))
	for _, row := range gw.Tools() {
		toolIDs = append(toolIDs, row.ID)
	}
	if _, err := permSvc.Save(ctx, permissions.Profile{
		ID:                   "tool_surface_test",
		Name:                 "Tool Surface Test",
		Description:          "Permissive test profile for gateway tool-surface tests",
		AllowedReadPaths:     []string{workspace},
		AllowedWritePaths:    []string{workspace},
		AllowedExecutePaths:  []string{workspace},
		ForbiddenPaths:       []string{},
		AllowedTools:         toolIDs,
		ApprovalRequiredRisk: []string{},
		MaxBytesPerWrite:     8 * 1024 * 1024,
		AllowNetwork:         true,
		Editable:             true,
		Active:               true,
	}); err != nil {
		t.Fatalf("save test profile: %v", err)
	}
	return gw, st, workspace
}

func firstAuthorizer(items []ToolAutonomyAuthorizer) ToolAutonomyAuthorizer {
	if len(items) == 0 {
		return nil
	}
	return items[0]
}
