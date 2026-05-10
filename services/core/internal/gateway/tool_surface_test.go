package gateway

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
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
	for _, row := range rows {
		if strings.HasPrefix(row.AdapterID, "stub.") {
			t.Fatalf("capability %s must not use synthetic stub adapter %q", row.ID, row.AdapterID)
		}
		if row.Status == domain.ToolCapabilityDeferred || row.Status == domain.ToolCapabilityStubbed {
			t.Fatalf("capability %s must be active or approval_only by default, got %q", row.ID, row.Status)
		}
		if row.Status != domain.ToolCapabilityActive && row.Status != domain.ToolCapabilityApprovalOnly {
			t.Fatalf("capability %s has unexpected default status %q", row.ID, row.Status)
		}
		if metadataString(row.Metadata, "gatewayToolId") == "" {
			t.Fatalf("capability %s missing gatewayToolId metadata", row.ID)
		}
		wantRequiresApproval := row.Risk.Rank() >= domain.ToolRiskHigh.Rank()
		if row.RequiresApprovalByDefault != wantRequiresApproval {
			t.Fatalf("capability %s requiresApprovalByDefault=%v, want %v for risk %s", row.ID, row.RequiresApprovalByDefault, wantRequiresApproval, row.Risk)
		}
		wantAutonomyEligible := row.Risk.Rank() <= domain.ToolRiskMedium.Rank()
		if row.AutonomyEligible != wantAutonomyEligible {
			t.Fatalf("capability %s autonomyEligible=%v, want %v for risk %s", row.ID, row.AutonomyEligible, wantAutonomyEligible, row.Risk)
		}
		if row.ResourceCost.CostUnits != inferCostUnits(row.Risk) {
			t.Fatalf("capability %s costUnits=%d, want %d for risk %s", row.ID, row.ResourceCost.CostUnits, inferCostUnits(row.Risk), row.Risk)
		}
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

func TestGatewayRegistersToolForEveryCapability(t *testing.T) {
	t.Parallel()
	gw, _, _ := newToolSurfaceGatewayHarness(t)
	tools := map[string]struct{}{}
	for _, tool := range gw.Tools() {
		tools[tool.ID] = struct{}{}
	}
	for _, capability := range gw.capabilities.List() {
		toolID := metadataString(capability.Metadata, "gatewayToolId")
		if toolID == "" {
			t.Fatalf("capability %s missing gatewayToolId", capability.ID)
		}
		if _, ok := tools[toolID]; !ok {
			t.Fatalf("capability %s maps to unregistered tool %s", capability.ID, toolID)
		}
	}
}

func TestToolCapabilityRegistryWithStorePersistsStatusOverrides(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	reg, err := NewToolCapabilityRegistryWithStore(ctx, &SQLiteOverrideStore{DB: st.DB})
	if err != nil {
		t.Fatalf("registry with store: %v", err)
	}
	if _, ok, err := reg.UpdateStatusWithReason(ctx, "filesystem.read_file", domain.ToolCapabilityDisabled, "tester", "test override"); err != nil || !ok {
		t.Fatalf("update status: ok=%v err=%v", ok, err)
	}

	reloaded, err := NewToolCapabilityRegistryWithStore(ctx, &SQLiteOverrideStore{DB: st.DB})
	if err != nil {
		t.Fatalf("reload registry with store: %v", err)
	}
	capability, ok := reloaded.Get("filesystem.read_file")
	if !ok {
		t.Fatalf("missing filesystem.read_file after reload")
	}
	if capability.Status != domain.ToolCapabilityDisabled {
		t.Fatalf("expected persisted disabled status, got %q", capability.Status)
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

func TestCapabilityStatusTransitionUnknownRiskIsConservative(t *testing.T) {
	t.Parallel()
	transition := ClassifyCapabilityStatusTransition(domain.ToolCapability{
		ID:     "custom.unknown_risk",
		Domain: "custom",
		Name:   "unknown_risk",
		Status: domain.ToolCapabilityDisabled,
		Effect: []domain.ToolEffect{domain.ToolEffectRead},
		Risk:   domain.ToolRisk("unknown"),
	}, domain.ToolCapabilityActive)
	if !transition.RequiresApproval || transition.RiskClass != "high" {
		t.Fatalf("expected unknown risk transition to require high-risk approval, got %#v", transition)
	}
}

func TestGatewayDirectCapabilityStatusUpdateCannotActivateDangerousCapability(t *testing.T) {
	t.Parallel()
	gw, _, _ := newToolSurfaceGatewayHarness(t)
	if _, _, _, err := gw.UpdateCapabilityStatusWithMetadata(context.Background(), "process.spawn_process", domain.ToolCapabilityActive, CapabilityStatusUpdateMetadata{
		Actor:          "operator-a",
		Reason:         "stale direct path should still be governed",
		RiskClass:      "high",
		TransitionRisk: "high",
	}); err == nil {
		t.Fatalf("expected direct dangerous activation to require approval")
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

func TestGatewayExecuteAndWaitApprovesAndReruns(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)
	laneID := "execute.wait.write"
	if _, err := gw.lanes.Save(ctx, lanes.Lane{
		ID:               laneID,
		Name:             "Execute Wait Write",
		Description:      "Approval-gated write lane for ExecuteAndWait",
		ActionType:       "invoke",
		AllowedPaths:     []string{workspace},
		WriteIntent:      true,
		RequiresApproval: true,
		RiskClass:        "safe_write",
		MaxBytes:         1024,
		Enabled:          true,
	}); err != nil {
		t.Fatalf("save lane: %v", err)
	}
	approveNextGatewayRequest(t, ctx, gw, st, "approved")

	target := "scratch/execute-and-wait-approved.txt"
	res, err := gw.ExecuteAndWait(ctx, Request{
		ToolID:        "fs.write",
		LaneID:        laneID,
		Action:        "invoke",
		CorrelationID: "execute-and-wait-approved",
		TraceID:       "trace-execute-and-wait-approved",
		Paths:         []string{target},
		Input:         map[string]any{"contents": "approved\n"},
		Initiator:     "tester",
	})
	if err != nil {
		t.Fatalf("execute and wait: %v", err)
	}
	if res.Status != StatusOK {
		t.Fatalf("expected ok after approval, got %s (%s)", res.Status, res.DeniedReason)
	}
	body, err := os.ReadFile(filepath.Join(workspace, target))
	if err != nil {
		t.Fatalf("read approved write: %v", err)
	}
	if string(body) != "approved\n" {
		t.Fatalf("unexpected approved write contents %q", string(body))
	}
}

func TestGatewayExecuteAndWaitDeniedApprovalReturnsDenied(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)
	laneID := "execute.wait.denied"
	if _, err := gw.lanes.Save(ctx, lanes.Lane{
		ID:               laneID,
		Name:             "Execute Wait Denied",
		Description:      "Approval-denied write lane for ExecuteAndWait",
		ActionType:       "invoke",
		AllowedPaths:     []string{workspace},
		WriteIntent:      true,
		RequiresApproval: true,
		RiskClass:        "safe_write",
		MaxBytes:         1024,
		Enabled:          true,
	}); err != nil {
		t.Fatalf("save lane: %v", err)
	}
	approveNextGatewayRequest(t, ctx, gw, st, "denied")

	target := "scratch/execute-and-wait-denied.txt"
	res, err := gw.ExecuteAndWait(ctx, Request{
		ToolID:        "fs.write",
		LaneID:        laneID,
		Action:        "invoke",
		CorrelationID: "execute-and-wait-denied",
		TraceID:       "trace-execute-and-wait-denied",
		Paths:         []string{target},
		Input:         map[string]any{"contents": "denied\n"},
		Initiator:     "tester",
	})
	if err != nil {
		t.Fatalf("execute and wait: %v", err)
	}
	if res.Status != StatusDenied {
		t.Fatalf("expected denied after approval denial, got %s", res.Status)
	}
	if _, err := os.Stat(filepath.Join(workspace, target)); !os.IsNotExist(err) {
		t.Fatalf("denied write should not create target, stat err=%v", err)
	}
}

func TestGatewayApprovalFingerprintRejectsReplayForDifferentShape(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)
	laneID := "approval.fingerprint.write"
	if _, err := gw.lanes.Save(ctx, lanes.Lane{
		ID:               laneID,
		Name:             "Approval Fingerprint Write",
		Description:      "Approval-gated write lane for fingerprint tests",
		ActionType:       "invoke",
		AllowedPaths:     []string{workspace},
		WriteIntent:      true,
		RequiresApproval: true,
		RiskClass:        "safe_write",
		MaxBytes:         1024,
		Enabled:          true,
	}); err != nil {
		t.Fatalf("save write lane: %v", err)
	}
	readLaneID := "approval.fingerprint.read"
	if _, err := gw.lanes.Save(ctx, lanes.Lane{
		ID:               readLaneID,
		Name:             "Approval Fingerprint Read",
		Description:      "Approval-gated read lane for fingerprint tests",
		ActionType:       "invoke",
		AllowedPaths:     []string{workspace},
		WriteIntent:      false,
		RequiresApproval: true,
		RiskClass:        "read_only",
		MaxBytes:         1024,
		Enabled:          true,
	}); err != nil {
		t.Fatalf("save read lane: %v", err)
	}

	baseReq := Request{
		ToolID:              "fs.write",
		LaneID:              laneID,
		Action:              "invoke",
		CorrelationID:       "corr-approval-fingerprint-open",
		TraceID:             "trace-approval-fingerprint-open",
		Source:              "user",
		WorkspaceID:         "workspace:fingerprint",
		ProvenanceActor:     "operator-a",
		ProvenanceActorType: "user",
		Paths:               []string{"scratch/fingerprint-approved.txt"},
		Input:               map[string]any{"contents": "approved\n"},
		Initiator:           "operator-a",
	}
	first, err := gw.Execute(ctx, baseReq)
	if err != nil {
		t.Fatalf("open approval: %v", err)
	}
	if first.Status != StatusNeedsApprov {
		t.Fatalf("expected needs_approval, got %s (%s)", first.Status, first.DeniedReason)
	}
	approvalID := approvalRequestIDFromResult(first)
	if approvalID <= 0 {
		t.Fatalf("missing approval request id in %#v", first.Data)
	}
	if _, err := gw.approvals.Decide(ctx, approvalID, "operator-a", "approved", "fingerprint test approval"); err != nil {
		t.Fatalf("approve request: %v", err)
	}
	approvalIDText := strconv.FormatInt(approvalID, 10)

	matching := baseReq
	matching.CorrelationID = "corr-approval-fingerprint-match"
	matching.ApprovalID = approvalIDText
	matchRes, err := gw.Execute(ctx, matching)
	if err != nil {
		t.Fatalf("matching execute: %v", err)
	}
	if matchRes.Status != StatusOK {
		t.Fatalf("matching approval should succeed, got %s (%s)", matchRes.Status, matchRes.DeniedReason)
	}

	cases := []struct {
		name   string
		mutate func(Request) Request
	}{
		{
			name: "different_tool",
			mutate: func(req Request) Request {
				req.ToolID = "fs.read"
				req.LaneID = readLaneID
				req.Paths = []string{"sample.txt"}
				req.Input = nil
				return req
			},
		},
		{
			name: "different_path",
			mutate: func(req Request) Request {
				req.Paths = []string{"scratch/fingerprint-other.txt"}
				return req
			},
		},
		{
			name: "different_actor",
			mutate: func(req Request) Request {
				req.ProvenanceActor = "operator-b"
				req.Initiator = "operator-b"
				return req
			},
		},
		{
			name: "different_lane",
			mutate: func(req Request) Request {
				req.LaneID = "fs.write.bounded"
				return req
			},
		},
		{
			name: "different_workspace",
			mutate: func(req Request) Request {
				req.WorkspaceID = "workspace:other"
				return req
			},
		},
		{
			name: "different_risk",
			mutate: func(req Request) Request {
				req.RiskClass = "dangerous"
				return req
			},
		},
		{
			name: "different_input_shape",
			mutate: func(req Request) Request {
				req.Input = map[string]any{"contents": "changed\n"}
				return req
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := tc.mutate(baseReq)
			req.CorrelationID = "corr-approval-fingerprint-" + tc.name
			req.ApprovalID = approvalIDText
			res, err := gw.Execute(ctx, req)
			if err != nil {
				t.Fatalf("execute mismatch: %v", err)
			}
			if res.Status != StatusDenied {
				t.Fatalf("expected denied for %s, got %s", tc.name, res.Status)
			}
			if !strings.Contains(res.DeniedReason, "fingerprint mismatch") {
				t.Fatalf("expected fingerprint mismatch reason, got %q", res.DeniedReason)
			}
			mustGatewayInvocationDeniedReasonContains(t, st, req.CorrelationID, "fingerprint mismatch")
			mustAuditActionCount(t, st, "tool.denied", req.CorrelationID)
		})
	}
}

func TestGatewayApprovalFingerprintMatchesSyntheticGatewayJobReplay(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)
	laneID := "approval.fingerprint.synthetic.job"
	if _, err := gw.lanes.Save(ctx, lanes.Lane{
		ID:               laneID,
		Name:             "Approval Fingerprint Synthetic Job",
		Description:      "Approval-gated lane for synthetic gateway job replay",
		ActionType:       "invoke",
		AllowedPaths:     []string{workspace},
		WriteIntent:      true,
		RequiresApproval: true,
		RiskClass:        "safe_write",
		MaxBytes:         1024,
		Enabled:          true,
	}); err != nil {
		t.Fatalf("save lane: %v", err)
	}

	req := Request{
		ToolID:        "fs.write",
		LaneID:        laneID,
		Action:        "invoke",
		CorrelationID: "corr-approval-fingerprint-synthetic-open",
		TraceID:       "trace-approval-fingerprint-synthetic-open",
		Paths:         []string{"scratch/fingerprint-synthetic-approved.txt"},
		Input:         map[string]any{"contents": "approved via synthetic job\n"},
		Initiator:     "chat",
		Metadata: map[string]any{
			"chatUserRequest": "write a synthetic approval replay test file",
		},
	}
	first, err := gw.Execute(ctx, req)
	if err != nil {
		t.Fatalf("open approval: %v", err)
	}
	if first.Status != StatusNeedsApprov {
		t.Fatalf("expected needs_approval, got %s (%s)", first.Status, first.DeniedReason)
	}
	approvalID := approvalRequestIDFromResult(first)
	if approvalID <= 0 {
		t.Fatalf("missing approval request id in %#v", first.Data)
	}
	jobID, _ := first.Data["jobId"].(string)
	if strings.TrimSpace(jobID) == "" {
		t.Fatalf("missing synthetic job id in %#v", first.Data)
	}
	if _, err := gw.approvals.Decide(ctx, approvalID, "operator-a", "approved", "synthetic job replay approval"); err != nil {
		t.Fatalf("approve request: %v", err)
	}
	if _, err := st.DB.ExecContext(ctx, `UPDATE jobs SET approval_status = 'granted' WHERE id = ?`, jobID); err != nil {
		t.Fatalf("mark job approval granted: %v", err)
	}

	var metadataJSON string
	if err := st.DB.QueryRowContext(ctx, `SELECT metadata_json FROM jobs WHERE id = ?`, jobID).Scan(&metadataJSON); err != nil {
		t.Fatalf("load synthetic job metadata: %v", err)
	}
	var meta struct {
		UserRequest    string         `json:"userRequest"`
		RequestPayload map[string]any `json:"requestPayload"`
	}
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	payload := meta.RequestPayload
	replayJobID := jobID
	replayReq := Request{
		ToolID:              strings.TrimSpace(testReadString(payload, "toolId")),
		LaneID:              strings.TrimSpace(testReadString(payload, "laneId")),
		Domain:              strings.TrimSpace(testReadString(payload, "domain")),
		Action:              strings.TrimSpace(testReadString(payload, "action")),
		RiskClass:           strings.TrimSpace(testReadString(payload, "riskClass")),
		ExecutionLevel:      strings.TrimSpace(testReadString(payload, "executionLevel")),
		CorrelationID:       "corr-approval-fingerprint-synthetic-replay",
		TraceID:             "trace-approval-fingerprint-synthetic-replay",
		Paths:               testReadStringSlice(payload, "paths"),
		Input:               testReadMap(payload, "input"),
		JobID:               &replayJobID,
		Initiator:           strings.TrimSpace(testReadString(payload, "initiator")),
		Source:              strings.TrimSpace(testReadString(payload, "source")),
		WorkspaceID:         strings.TrimSpace(testReadString(payload, "workspaceId")),
		ProvenanceActor:     strings.TrimSpace(testReadString(payload, "provenanceActor")),
		ProvenanceActorType: strings.TrimSpace(testReadString(payload, "provenanceActorType")),
	}
	if replayReq.Initiator == "" {
		replayReq.Initiator = "job"
	}
	res, err := gw.Execute(ctx, replayReq)
	if err != nil {
		t.Fatalf("execute synthetic replay: %v", err)
	}
	if res.Status != StatusOK {
		t.Fatalf("synthetic job replay should succeed, got %s (%s)", res.Status, res.DeniedReason)
	}
}

func TestGatewayApprovalFingerprintMatchesDesktopOpenSyntheticJobReplay(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)
	laneID := "approval.fingerprint.desktop.open"
	if _, err := gw.lanes.Save(ctx, lanes.Lane{
		ID:               laneID,
		Name:             "Approval Fingerprint Desktop Open",
		Description:      "Approval-gated desktop.open lane for synthetic gateway job replay",
		ActionType:       "invoke",
		AllowedPaths:     []string{workspace},
		WriteIntent:      true,
		RequiresApproval: true,
		RiskClass:        "dangerous",
		MaxBytes:         1024,
		Enabled:          true,
	}); err != nil {
		t.Fatalf("save lane: %v", err)
	}

	req := Request{
		ToolID:        "desktop.open",
		LaneID:        laneID,
		Action:        "invoke",
		CorrelationID: "corr-approval-fingerprint-desktop-open",
		TraceID:       "trace-approval-fingerprint-desktop-open",
		Input:         map[string]any{"application": "minecraft"},
		Initiator:     "chat",
		Metadata: map[string]any{
			"chatUserRequest": "Open minecraft please.",
		},
	}
	first, err := gw.Execute(ctx, req)
	if err != nil {
		t.Fatalf("open approval: %v", err)
	}
	if first.Status != StatusNeedsApprov {
		t.Fatalf("expected needs_approval, got %s (%s)", first.Status, first.DeniedReason)
	}
	approvalID := approvalRequestIDFromResult(first)
	if approvalID <= 0 {
		t.Fatalf("missing approval request id in %#v", first.Data)
	}
	jobID, _ := first.Data["jobId"].(string)
	if strings.TrimSpace(jobID) == "" {
		t.Fatalf("missing synthetic job id in %#v", first.Data)
	}
	if _, err := gw.approvals.Decide(ctx, approvalID, "operator-a", "approved", "desktop open replay approval"); err != nil {
		t.Fatalf("approve request: %v", err)
	}
	if _, err := st.DB.ExecContext(ctx, `UPDATE jobs SET approval_status = 'granted' WHERE id = ?`, jobID); err != nil {
		t.Fatalf("mark job approval granted: %v", err)
	}

	var metadataJSON string
	if err := st.DB.QueryRowContext(ctx, `SELECT metadata_json FROM jobs WHERE id = ?`, jobID).Scan(&metadataJSON); err != nil {
		t.Fatalf("load synthetic job metadata: %v", err)
	}
	var meta struct {
		RequestPayload map[string]any `json:"requestPayload"`
	}
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	payload := meta.RequestPayload
	replayJobID := jobID
	replayReq := Request{
		ToolID:              strings.TrimSpace(testReadString(payload, "toolId")),
		LaneID:              strings.TrimSpace(testReadString(payload, "laneId")),
		Domain:              strings.TrimSpace(testReadString(payload, "domain")),
		Action:              strings.TrimSpace(testReadString(payload, "action")),
		RiskClass:           strings.TrimSpace(testReadString(payload, "riskClass")),
		ExecutionLevel:      strings.TrimSpace(testReadString(payload, "executionLevel")),
		CorrelationID:       "corr-approval-fingerprint-desktop-open-replay",
		TraceID:             "trace-approval-fingerprint-desktop-open-replay",
		Input:               testReadMap(payload, "input"),
		JobID:               &replayJobID,
		Initiator:           strings.TrimSpace(testReadString(payload, "initiator")),
		Source:              strings.TrimSpace(testReadString(payload, "source")),
		WorkspaceID:         strings.TrimSpace(testReadString(payload, "workspaceId")),
		ProvenanceActor:     strings.TrimSpace(testReadString(payload, "provenanceActor")),
		ProvenanceActorType: strings.TrimSpace(testReadString(payload, "provenanceActorType")),
	}
	lane, err := gw.lanes.Get(ctx, laneID)
	if err != nil {
		t.Fatalf("load lane: %v", err)
	}
	tool := gw.tools["desktop.open"]
	granted, err := gw.approvalGrantPresent(ctx, replayReq, lane, tool, replayReq.RiskClass, replayReq.ExecutionLevel, nil)
	if err != nil {
		t.Fatalf("desktop.open replay approval check failed: %v", err)
	}
	if !granted {
		t.Fatalf("expected desktop.open replay approval to be granted")
	}
}

func testReadString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	switch v := m[key].(type) {
	case string:
		return v
	default:
		return ""
	}
}

func testReadMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}

func testReadStringSlice(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	switch raw := m[key].(type) {
	case []string:
		return raw
	case []any:
		out := make([]string, 0, len(raw))
		for _, v := range raw {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func TestGatewayApprovalFingerprintDryRunDoesNotOpenApproval(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)
	laneID := "approval.fingerprint.dryrun"
	if _, err := gw.lanes.Save(ctx, lanes.Lane{
		ID:               laneID,
		Name:             "Approval Fingerprint Dry Run",
		Description:      "Approval-gated dry run lane",
		ActionType:       "invoke",
		AllowedPaths:     []string{workspace},
		WriteIntent:      true,
		RequiresApproval: true,
		RiskClass:        "safe_write",
		MaxBytes:         1024,
		Enabled:          true,
	}); err != nil {
		t.Fatalf("save lane: %v", err)
	}
	res, err := gw.Execute(ctx, Request{
		ToolID:              "fs.write",
		LaneID:              laneID,
		Action:              "invoke",
		CorrelationID:       "corr-approval-fingerprint-dryrun",
		Source:              "user",
		WorkspaceID:         "workspace:fingerprint",
		ProvenanceActor:     "operator-a",
		ProvenanceActorType: "user",
		Paths:               []string{"scratch/dryrun-fingerprint.txt"},
		Input:               map[string]any{"contents": "dry run\n"},
		Initiator:           "operator-a",
		DryRun:              true,
	})
	if err != nil {
		t.Fatalf("dry run execute: %v", err)
	}
	if res.Status != StatusDryRun {
		t.Fatalf("expected dry_run, got %s (%s)", res.Status, res.DeniedReason)
	}
	var count int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM approval_requests WHERE job_id LIKE 'chat-gw-%'`).Scan(&count); err != nil {
		t.Fatalf("count approval requests: %v", err)
	}
	if count != 0 {
		t.Fatalf("dry-run should not open approval request, got %d", count)
	}
}

func approveNextGatewayRequest(t *testing.T, ctx context.Context, gw *Gateway, st *store.Store, decision string) {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			var requestID int64
			err := st.DB.QueryRowContext(ctx, `
SELECT id
FROM approval_requests
WHERE status = 'pending'
ORDER BY id DESC
LIMIT 1`).Scan(&requestID)
			if err == nil {
				_, err = gw.approvals.Decide(ctx, requestID, "tester", decision, "test decision")
				done <- err
				return
			}
			select {
			case <-ctx.Done():
				done <- ctx.Err()
				return
			case <-ticker.C:
			}
		}
	}()
	t.Cleanup(func() {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("approval decision failed: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("approval decision goroutine did not finish")
		}
	})
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

func TestGatewayPrivilegedToolRequiresApprovalWithoutCapabilityPolicy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gw, st, workspace := newToolSurfaceGatewayHarness(t)

	laneID := "test.permissive_service_control"
	if _, err := gw.lanes.Save(ctx, lanes.Lane{
		ID:               laneID,
		Name:             "Permissive service control",
		Description:      "Regression lane that cannot lower intrinsic privileged tool risk.",
		ActionType:       "system.service_control",
		AllowedPaths:     []string{workspace},
		ForbiddenPaths:   []string{},
		WriteIntent:      true,
		RequiresApproval: false,
		RiskClass:        "read_only",
		Builtin:          false,
		Enabled:          true,
	}); err != nil {
		t.Fatalf("save permissive service-control lane: %v", err)
	}

	correlationID := "corr-intrinsic-privileged-approval"
	res, err := gw.Execute(ctx, Request{
		ToolID:              "system.service_control",
		LaneID:              laneID,
		CorrelationID:       correlationID,
		TraceID:             "trace-intrinsic-privileged-approval",
		Source:              "user",
		WorkspaceID:         "workspace:test",
		ProvenanceActor:     "tester",
		ProvenanceActorType: "test",
		Input:               map[string]any{"control": "restart"},
		Initiator:           "tester",
	})
	if err != nil {
		t.Fatalf("privileged execute error: %v", err)
	}
	if res.Status != StatusNeedsApprov {
		t.Fatalf("expected privileged gateway tool to require approval, got %s (%s)", res.Status, res.DeniedReason)
	}
	if res.PolicyOutcome != OutcomeRequireApproval {
		t.Fatalf("expected require_approval policy outcome, got %s", res.PolicyOutcome)
	}
	if res.ExecutionLevel != "L3" {
		t.Fatalf("expected service control to remain L3, got %s", res.ExecutionLevel)
	}
	payload := mustAuditPayloadByActionAndCorrelation(t, st, "tool.needs_approval", correlationID)
	assertAuditContext(t, payload, correlationID, "trace-intrinsic-privileged-approval", "workspace:test")

	var errorAuditCount int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM audit_records WHERE correlation_id = ? AND action = 'tool.error'`, correlationID).Scan(&errorAuditCount); err != nil {
		t.Fatalf("query error audit count: %v", err)
	}
	if errorAuditCount != 0 {
		t.Fatalf("privileged tool reached execution before approval")
	}
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

	req := Request{
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
	}
	approvalID := approveGatewayRequestForTest(t, ctx, gw, req, "artifact summary write approved")
	req.ApprovalID = strconv.FormatInt(approvalID, 10)
	res, err := gw.Execute(ctx, req)
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
	if got := strings.TrimSpace(asString(entry["path"])); !strings.HasSuffix(filepath.ToSlash(got), "scratch/artifact-summary.txt") {
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

func mustAuditActionCount(t *testing.T, st *store.Store, action, correlation string) {
	t.Helper()
	var count int
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT COUNT(1)
FROM audit_records
WHERE correlation_id = ? AND action = ?`,
		correlation, action,
	).Scan(&count); err != nil {
		t.Fatalf("query audit count action=%s correlation=%s: %v", action, correlation, err)
	}
	if count == 0 {
		t.Fatalf("expected audit action=%s correlation=%s", action, correlation)
	}
}

func mustGatewayInvocationDeniedReasonContains(t *testing.T, st *store.Store, correlation, want string) {
	t.Helper()
	var reason string
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT denied_reason
FROM gateway_invocations
WHERE correlation_id = ?
ORDER BY id DESC LIMIT 1`,
		correlation,
	).Scan(&reason); err != nil {
		t.Fatalf("query gateway invocation denied reason: %v", err)
	}
	if !strings.Contains(reason, want) {
		t.Fatalf("gateway denied reason = %q, want substring %q", reason, want)
	}
}

func approveGatewayRequestForTest(t *testing.T, ctx context.Context, gw *Gateway, req Request, note string) int64 {
	t.Helper()
	req = normalizeGatewayRequestForApprovalTest(gw, req)
	lane, err := gw.lanes.Get(ctx, req.LaneID)
	if err != nil {
		t.Fatalf("get lane: %v", err)
	}
	tool, ok := gw.tools[req.ToolID]
	if !ok {
		t.Fatalf("tool %q not registered", req.ToolID)
	}
	capabilityRiskClass := ""
	if gw.capabilities != nil {
		if capability, ok := gw.capabilities.Resolve(req.ToolID); ok {
			capabilityRiskClass = gatewayRiskClassFromToolRisk(capability.Risk)
		}
	}
	risk := effectiveRiskClass(req.RiskClass, lane.RiskClass, tool.RiskClass(), capabilityRiskClass)
	level := normalizeExecutionLevel(req.ExecutionLevel)
	if level == "" {
		level = normalizeExecutionLevel(tool.ExecutionLevel())
	}
	if level == "" {
		level = executionLevelFromRisk(risk)
	}
	res, err := gw.recordNeedsApproval(ctx, req, lane, tool, risk, level, "test", note)
	if err != nil {
		t.Fatalf("record approval request: %v", err)
	}
	approvalID := approvalRequestIDFromResult(res)
	if approvalID <= 0 {
		t.Fatalf("missing approval request id in %#v", res.Data)
	}
	if _, err := gw.approvals.Decide(ctx, approvalID, "tester", "approved", note); err != nil {
		t.Fatalf("approve gateway request: %v", err)
	}
	return approvalID
}

func normalizeGatewayRequestForApprovalTest(gw *Gateway, req Request) Request {
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = newCorrelationID()
	}
	if strings.TrimSpace(req.Initiator) == "" {
		req.Initiator = "operator"
	}
	if strings.TrimSpace(req.Source) == "" {
		req.Source = "user"
	}
	if gw.capabilities != nil {
		if capability, ok := gw.capabilities.Resolve(req.ToolID); ok {
			if strings.TrimSpace(req.Domain) == "" {
				req.Domain = capability.Domain
			}
			req.Metadata = mergeGatewayMetadata(req.Metadata, map[string]any{
				"toolCapabilityId": capability.ID,
				"toolRisk":         capability.Risk,
			})
		}
	}
	if strings.TrimSpace(req.WorkspaceID) == "" {
		req.WorkspaceID = workspaceIDFromPath(gw.workspace)
	}
	if strings.TrimSpace(req.ProvenanceActor) == "" {
		req.ProvenanceActor = req.Initiator
	}
	if strings.TrimSpace(req.ProvenanceActorType) == "" {
		req.ProvenanceActorType = "service"
	}
	if strings.TrimSpace(req.Domain) == "" {
		req.Domain = toolDomainFromID(req.ToolID)
	}
	if strings.TrimSpace(req.LaneID) == "" {
		req.LaneID = req.ToolID
	}
	return req
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
