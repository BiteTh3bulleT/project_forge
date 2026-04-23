package gateway

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestToolCapabilityRegistryRejectsUnknownStatusAtRegistrationBoundaries(t *testing.T) {
	t.Parallel()
	reg := NewToolCapabilityRegistry()

	if _, _, err := reg.UpdateStatus("filesystem.read_file", domain.ToolCapabilityStatus("unknown_value")); err == nil {
		t.Fatalf("expected unknown status update to fail")
	}

	err := reg.Register(domain.ToolCapability{
		ID:          "custom.test_capability",
		Domain:      "custom",
		Name:        "test_capability",
		Description: "test",
		Status:      domain.ToolCapabilityStatus("unknown_value"),
		Lane:        domain.ToolLaneIO,
		Effect:      []domain.ToolEffect{domain.ToolEffectRead},
		Risk:        domain.ToolRiskLow,
	})
	if err == nil {
		t.Fatalf("expected registration with unknown status to fail")
	}
}

func TestToolCapabilityRegistryDangerousDefaultsNotFreelyActive(t *testing.T) {
	t.Parallel()
	reg := NewToolCapabilityRegistry()
	dangerous := []string{
		"filesystem.delete_file",
		"filesystem.set_permissions",
		"filesystem.restore_snapshot",
		"config.restore",
		"config.migrate_schema",
		"process.spawn_process",
		"process.kill_process",
		"code.run_shell",
		"code.eval_code",
		"network.http_request",
		"network.open_socket",
		"network.scan_network",
		"network.open_tunnel",
		"network.intercept_traffic",
		"network.set_firewall_rule",
		"identity.retrieve_secret",
		"identity.decrypt",
		"identity.sudo",
		"identity.switch_user",
		"identity.issue_token",
		"identity.set_policy",
		"config.backup",
		"backup.restore",
		"external.send_email",
		"external.post_message",
		"external.call_api",
		"external.create_issue",
		"external.update_issue",
		"filesystem.sync_to_remote",
		"ui.inject_input",
		"device.capture_camera",
		"device.capture_audio",
		"ui.open_url",
	}
	for _, id := range dangerous {
		id := id
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			capability, ok := reg.Get(id)
			if !ok {
				t.Fatalf("missing capability %s", id)
			}
			if capability.Status == domain.ToolCapabilityActive {
				t.Fatalf("dangerous capability %s must not default to active", id)
			}
		})
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
	if disabledDecision.Error == nil || disabledDecision.Error.Code != domain.ToolErrToolDisabled {
		t.Fatalf("expected structured disabled error, got %#v", disabledDecision.Error)
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
	if approvalOnlyDecision.Error == nil || approvalOnlyDecision.Error.Code != domain.ToolErrApprovalRequired {
		t.Fatalf("expected structured approval-required error, got %#v", approvalOnlyDecision.Error)
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
	if stubbedDecision.Error == nil || stubbedDecision.Error.Code != domain.ToolErrUnsupportedOperation {
		t.Fatalf("expected structured unsupported error, got %#v", stubbedDecision.Error)
	}

	deferred := disabled
	deferred.Status = domain.ToolCapabilityDeferred
	deferredDecision := evaluator.Evaluate(context.Background(), ToolPolicyInput{
		Request:       req,
		Capability:    deferred,
		ResolvedPaths: nil,
		HasAdapter:    true,
	})
	if deferredDecision.Status != StatusUnsupported {
		t.Fatalf("expected unsupported status for deferred capability, got %s", deferredDecision.Status)
	}
	if deferredDecision.Error == nil || deferredDecision.Error.Code != domain.ToolErrUnsupportedOperation {
		t.Fatalf("expected deferred capability to return unsupported error, got %#v", deferredDecision.Error)
	}

	irisReq := req
	irisReq.Source = string(domain.SourceFutureIRIS)
	irisReq.IntentID = ""
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
	if activeDecision.Error == nil || activeDecision.Error.Code != domain.ToolErrPolicyDenied {
		t.Fatalf("expected structured policy-denied error, got %#v", activeDecision.Error)
	}

	irisReq.IntentID = "intent-1"
	approvalOnlyDecision = evaluator.Evaluate(context.Background(), ToolPolicyInput{
		Request:       irisReq,
		Capability:    approvalOnly,
		ResolvedPaths: nil,
		HasAdapter:    true,
	})
	if approvalOnlyDecision.Status != StatusNeedsApprov {
		t.Fatalf("expected future_iris approval-only capability to require approval, got %s", approvalOnlyDecision.Status)
	}
	if approvalOnlyDecision.Error == nil || approvalOnlyDecision.Error.Code != domain.ToolErrApprovalRequired {
		t.Fatalf("expected structured approval-required error for future_iris approval-only capability, got %#v", approvalOnlyDecision.Error)
	}

	deferredIrisDecision := evaluator.Evaluate(context.Background(), ToolPolicyInput{
		Request:       irisReq,
		Capability:    deferred,
		ResolvedPaths: nil,
		HasAdapter:    true,
	})
	if deferredIrisDecision.Status != StatusUnsupported {
		t.Fatalf("expected future_iris deferred capability to stay unsupported, got %s", deferredIrisDecision.Status)
	}
	if deferredIrisDecision.Error == nil || deferredIrisDecision.Error.Code != domain.ToolErrUnsupportedOperation {
		t.Fatalf("expected future_iris deferred capability to return unsupported error, got %#v", deferredIrisDecision.Error)
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
	if _, ok, err := gw.capabilities.UpdateStatus(disabledCapability.ID, domain.ToolCapabilityDisabled); err != nil || !ok {
		t.Fatalf("failed to disable capability")
	}
	disabledRes, err := gw.Execute(ctx, Request{
		ToolID:              "filesystem.read_file",
		LaneID:              "fs.read",
		CorrelationID:       "corr-disabled",
		TraceID:             "trace-disabled",
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
	if _, ok, err := gw.capabilities.UpdateStatus(disabledCapability.ID, domain.ToolCapabilityActive); err != nil || !ok {
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
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM audit_records WHERE correlation_id = ? AND action = 'tool.disabled'`, "corr-disabled").Scan(&deniedAuditCount); err != nil {
		t.Fatalf("query audit records: %v", err)
	}
	if deniedAuditCount == 0 {
		t.Fatalf("expected disabled audit record for blocked execution")
	}
	payload := mustAuditPayloadByActionAndCorrelation(t, st, "tool.disabled", "corr-disabled")
	assertAuditContext(t, payload, "corr-disabled", "trace-disabled", "workspace:test")
}

func TestGatewayDeferredCapabilityProducesUnsupportedAudit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)

	sample := filepath.Join(workspace, "sample-deferred.txt")
	if err := os.WriteFile(sample, []byte("ok"), 0o644); err != nil {
		t.Fatalf("seed sample: %v", err)
	}

	capability, ok := gw.capabilities.Get("filesystem.read_file")
	if !ok {
		t.Fatalf("missing capability filesystem.read_file")
	}
	if _, ok, err := gw.capabilities.UpdateStatus(capability.ID, domain.ToolCapabilityDeferred); err != nil || !ok {
		t.Fatalf("failed to defer capability: ok=%v err=%v", ok, err)
	}

	correlationID := "corr-deferred"
	res, err := gw.Execute(ctx, Request{
		ToolID:              "filesystem.read_file",
		LaneID:              "fs.read",
		CorrelationID:       correlationID,
		TraceID:             "trace-deferred",
		Source:              "user",
		WorkspaceID:         "workspace:test",
		ProvenanceActor:     "tester",
		ProvenanceActorType: "test",
		Paths:               []string{"sample-deferred.txt"},
		Initiator:           "tester",
	})
	if err != nil {
		t.Fatalf("deferred execute error: %v", err)
	}
	if res.Status != StatusUnsupported {
		t.Fatalf("expected deferred capability to return unsupported, got %s", res.Status)
	}

	var invocationCount int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM gateway_invocations WHERE correlation_id = ? AND status = 'unsupported'`, correlationID).Scan(&invocationCount); err != nil {
		t.Fatalf("query invocation count: %v", err)
	}
	if invocationCount == 0 {
		t.Fatalf("expected unsupported gateway invocation record")
	}

	var auditCount int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM audit_records WHERE correlation_id = ? AND action = 'tool.unsupported'`, correlationID).Scan(&auditCount); err != nil {
		t.Fatalf("query audit count: %v", err)
	}
	if auditCount == 0 {
		t.Fatalf("expected tool.unsupported audit record for deferred capability")
	}
	payload := mustAuditPayloadByActionAndCorrelation(t, st, "tool.unsupported", correlationID)
	assertAuditContext(t, payload, correlationID, "trace-deferred", "workspace:test")
}

func TestGatewayApprovalOnlyCapabilityProducesNeedsApprovalAudit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)

	sample := filepath.Join(workspace, "sample-approval-only.txt")
	if err := os.WriteFile(sample, []byte("ok"), 0o644); err != nil {
		t.Fatalf("seed sample: %v", err)
	}

	capability, ok := gw.capabilities.Get("filesystem.read_file")
	if !ok {
		t.Fatalf("missing capability filesystem.read_file")
	}
	if _, ok, err := gw.capabilities.UpdateStatus(capability.ID, domain.ToolCapabilityApprovalOnly); err != nil || !ok {
		t.Fatalf("failed to set approval_only capability: ok=%v err=%v", ok, err)
	}

	correlationID := "corr-approval-only"
	res, err := gw.Execute(ctx, Request{
		ToolID:              "filesystem.read_file",
		LaneID:              "fs.read",
		CorrelationID:       correlationID,
		TraceID:             "trace-approval-only",
		Source:              "user",
		WorkspaceID:         "workspace:test",
		ProvenanceActor:     "tester",
		ProvenanceActorType: "test",
		Paths:               []string{"sample-approval-only.txt"},
		Initiator:           "tester",
	})
	if err != nil {
		t.Fatalf("approval-only execute error: %v", err)
	}
	if res.Status != StatusNeedsApprov {
		t.Fatalf("expected approval-only capability to require approval, got %s", res.Status)
	}

	var auditCount int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM audit_records WHERE correlation_id = ? AND action = 'tool.needs_approval'`, correlationID).Scan(&auditCount); err != nil {
		t.Fatalf("query audit count: %v", err)
	}
	if auditCount == 0 {
		t.Fatalf("expected tool.needs_approval audit record for approval-only capability")
	}
	payload := mustAuditPayloadByActionAndCorrelation(t, st, "tool.needs_approval", correlationID)
	assertAuditContext(t, payload, correlationID, "trace-approval-only", "workspace:test")
}

func TestGatewayFutureIrisCannotBypassCapabilityStatusPolicy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)

	sample := filepath.Join(workspace, "sample-future-iris.txt")
	if err := os.WriteFile(sample, []byte("ok"), 0o644); err != nil {
		t.Fatalf("seed sample: %v", err)
	}

	capability, ok := gw.capabilities.Get("filesystem.read_file")
	if !ok {
		t.Fatalf("missing capability filesystem.read_file")
	}

	if _, ok, err := gw.capabilities.UpdateStatus(capability.ID, domain.ToolCapabilityDeferred); err != nil || !ok {
		t.Fatalf("failed to defer capability: ok=%v err=%v", ok, err)
	}
	deferredCorrelation := "corr-future-iris-deferred"
	deferredRes, err := gw.Execute(ctx, Request{
		ToolID:              "filesystem.read_file",
		LaneID:              "fs.read",
		CorrelationID:       deferredCorrelation,
		TraceID:             "trace-future-iris-deferred",
		Source:              string(domain.SourceFutureIRIS),
		WorkspaceID:         "workspace:test",
		IntentID:            "intent-future-iris",
		CharterID:           "charter-future-iris",
		BudgetID:            "budget-future-iris",
		ProvenanceActor:     "iris.service",
		ProvenanceActorType: "future_iris",
		Paths:               []string{"sample-future-iris.txt"},
		Initiator:           "iris.service",
	})
	if err != nil {
		t.Fatalf("future_iris deferred execute error: %v", err)
	}
	if deferredRes.Status != StatusUnsupported {
		t.Fatalf("expected future_iris deferred capability to remain unsupported, got %s", deferredRes.Status)
	}
	var deferredAuditCount int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM audit_records WHERE correlation_id = ? AND action = 'tool.unsupported'`, deferredCorrelation).Scan(&deferredAuditCount); err != nil {
		t.Fatalf("query deferred audit count: %v", err)
	}
	if deferredAuditCount == 0 {
		t.Fatalf("expected tool.unsupported audit for future_iris deferred capability")
	}
	deferredPayload := mustAuditPayloadByActionAndCorrelation(t, st, "tool.unsupported", deferredCorrelation)
	assertAuditContext(t, deferredPayload, deferredCorrelation, "trace-future-iris-deferred", "workspace:test")

	if _, ok, err := gw.capabilities.UpdateStatus(capability.ID, domain.ToolCapabilityApprovalOnly); err != nil || !ok {
		t.Fatalf("failed to set approval_only capability: ok=%v err=%v", ok, err)
	}
	approvalCorrelation := "corr-future-iris-approval-only"
	approvalRes, err := gw.Execute(ctx, Request{
		ToolID:              "filesystem.read_file",
		LaneID:              "fs.read",
		CorrelationID:       approvalCorrelation,
		TraceID:             "trace-future-iris-approval-only",
		Source:              string(domain.SourceFutureIRIS),
		WorkspaceID:         "workspace:test",
		IntentID:            "intent-future-iris",
		CharterID:           "charter-future-iris",
		BudgetID:            "budget-future-iris",
		ProvenanceActor:     "iris.service",
		ProvenanceActorType: "future_iris",
		Paths:               []string{"sample-future-iris.txt"},
		Initiator:           "iris.service",
	})
	if err != nil {
		t.Fatalf("future_iris approval-only execute error: %v", err)
	}
	if approvalRes.Status != StatusNeedsApprov {
		t.Fatalf("expected future_iris approval-only capability to require approval, got %s", approvalRes.Status)
	}
	var approvalAuditCount int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM audit_records WHERE correlation_id = ? AND action = 'tool.needs_approval'`, approvalCorrelation).Scan(&approvalAuditCount); err != nil {
		t.Fatalf("query approval audit count: %v", err)
	}
	if approvalAuditCount == 0 {
		t.Fatalf("expected tool.needs_approval audit for future_iris approval-only capability")
	}
	approvalPayload := mustAuditPayloadByActionAndCorrelation(t, st, "tool.needs_approval", approvalCorrelation)
	assertAuditContext(t, approvalPayload, approvalCorrelation, "trace-future-iris-approval-only", "workspace:test")
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

func TestGatewayDangerousCapabilityNotFreelyExecutable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, _, workspace := newToolSurfaceGatewayHarness(t)

	target := filepath.Join(workspace, "to-delete.txt")
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
		t.Fatalf("seed delete target: %v", err)
	}

	res, err := gw.Execute(ctx, Request{
		ToolID:              "fs.delete",
		LaneID:              "fs.write.bounded",
		CorrelationID:       "corr-dangerous-not-free",
		Source:              "user",
		WorkspaceID:         "workspace:test",
		ProvenanceActor:     "tester",
		ProvenanceActorType: "test",
		Paths:               []string{"to-delete.txt"},
		Initiator:           "tester",
	})
	if err != nil {
		t.Fatalf("dangerous execute: %v", err)
	}
	if res.Status != StatusNeedsApprov {
		t.Fatalf("expected dangerous tool to require approval, got %s", res.Status)
	}
}

func TestGatewaySafeReadOnlyCapabilityAllowed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)

	target := filepath.Join(workspace, "read-only.txt")
	if err := os.WriteFile(target, []byte("ok"), 0o644); err != nil {
		t.Fatalf("seed read target: %v", err)
	}

	res, err := gw.Execute(ctx, Request{
		ToolID:              "fs.read",
		LaneID:              "fs.read",
		CorrelationID:       "corr-safe-read",
		TraceID:             "trace-safe-read",
		Source:              "user",
		WorkspaceID:         "workspace:test",
		ProvenanceActor:     "tester",
		ProvenanceActorType: "test",
		Paths:               []string{"read-only.txt"},
		Initiator:           "tester",
	})
	if err != nil {
		t.Fatalf("safe read execute: %v", err)
	}
	if res.Status != StatusOK {
		t.Fatalf("expected safe read-only capability to execute, got %s", res.Status)
	}
	payload := mustAuditPayloadByActionAndCorrelation(t, st, "tool.executed", "corr-safe-read")
	assertAuditContext(t, payload, "corr-safe-read", "trace-safe-read", "workspace:test")
}

func TestGatewayExecutedAuditIncludesArtifactSummaries(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, _ := newToolSurfaceGatewayHarness(t)

	now := time.Now().UnixMilli()
	jobID := "job_artifact_summary"
	if _, err := st.DB.ExecContext(ctx, `
INSERT INTO jobs(
  id, created_at, updated_at, queued_at,
  title, requested_action, target_adapter, initiating_source,
  execution_boundary, risk_class, status, approval_status, write_intent, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		jobID, now, now, now,
		"artifact summary write", "gateway.action", "forge", "test",
		"command_execution", "safe_write", "awaiting_approval", "granted", 1, "{}",
	); err != nil {
		t.Fatalf("seed approved job: %v", err)
	}

	res, err := gw.Execute(ctx, Request{
		ToolID:              "fs.write",
		LaneID:              "fs.write.bounded",
		CorrelationID:       "corr-artifact-summary",
		TraceID:             "trace-artifact-summary",
		Source:              "user",
		WorkspaceID:         "workspace:test",
		ProvenanceActor:     "tester",
		ProvenanceActorType: "test",
		Paths:               []string{"scratch/artifact-summary.txt"},
		Input:               map[string]any{"contents": "artifact payload trace"},
		JobID:               &jobID,
		Initiator:           "tester",
	})
	if err != nil {
		t.Fatalf("write execute: %v", err)
	}
	if res.Status != StatusOK {
		t.Fatalf("expected status ok, got %s (%s)", res.Status, res.DeniedReason)
	}

	payload := mustAuditPayloadByActionAndCorrelation(t, st, "tool.executed", "corr-artifact-summary")
	assertAuditContext(t, payload, "corr-artifact-summary", "trace-artifact-summary", "workspace:test")

	if got, ok := payload["artifactCount"].(float64); !ok || int(got) != 1 {
		t.Fatalf("expected artifactCount=1, got %#v", payload["artifactCount"])
	}
	artifacts, ok := payload["artifacts"].([]any)
	if !ok || len(artifacts) != 1 {
		t.Fatalf("expected one artifact summary, got %#v", payload["artifacts"])
	}
	entry, ok := artifacts[0].(map[string]any)
	if !ok {
		t.Fatalf("expected artifact summary object, got %#v", artifacts[0])
	}
	if got := strings.TrimSpace(asString(entry["type"])); got != "writtenFile" {
		t.Fatalf("artifact type = %q want writtenFile", got)
	}
	if got := strings.TrimSpace(asString(entry["path"])); !strings.HasSuffix(got, "scratch/artifact-summary.txt") {
		t.Fatalf("artifact path = %q missing expected suffix", got)
	}
}

func TestGatewayScopeBoundaryFromCapabilityLimits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)

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
		TraceID:             "trace-scope-boundary",
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
	payload := mustAuditPayloadByActionAndCorrelation(t, st, "tool.denied", "corr-scope-boundary")
	assertAuditContext(t, payload, "corr-scope-boundary", "trace-scope-boundary", "workspace:test")
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

func mustAuditPayloadByActionAndCorrelation(t *testing.T, st *store.Store, action, correlation string) map[string]any {
	t.Helper()
	var payload string
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT payload_json
FROM audit_records
WHERE correlation_id = ? AND action = ?
ORDER BY id DESC LIMIT 1`,
		correlation, action,
	).Scan(&payload); err != nil {
		t.Fatalf("query audit payload action=%s correlation=%s: %v", action, correlation, err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		t.Fatalf("decode audit payload action=%s correlation=%s: %v payload=%s", action, correlation, err, payload)
	}
	return out
}

func assertAuditContext(t *testing.T, payload map[string]any, correlation, traceID, workspaceID string) {
	t.Helper()
	if got := strings.TrimSpace(asString(payload["correlationId"])); got != correlation {
		t.Fatalf("audit correlationId = %q want %q", got, correlation)
	}
	if got := strings.TrimSpace(asString(payload["traceId"])); got != traceID {
		t.Fatalf("audit traceId = %q want %q", got, traceID)
	}
	if got := strings.TrimSpace(asString(payload["workspaceId"])); got != workspaceID {
		t.Fatalf("audit workspaceId = %q want %q", got, workspaceID)
	}
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return ""
	}
}
