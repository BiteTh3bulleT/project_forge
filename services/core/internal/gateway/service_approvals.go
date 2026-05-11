package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/approvals"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/lanes"
)

func (g *Gateway) recordNeedsApproval(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level, profileID, reason string) (*Result, error) {
	var approvalReqID *int64
	reqForApproval := req
	reqForApproval.Input = gatewayApprovalRequestInput(tool, reqForApproval)
	effectiveJobID, jobIDPtr, err := g.resolveApprovalJobID(ctx, reqForApproval, lane, tool, risk, level)
	if err != nil {
		return nil, err
	}
	reqForInv := reqForApproval
	reqForInv.JobID = jobIDPtr

	if g.approvals != nil && strings.TrimSpace(effectiveJobID) != "" {
		fingerprintHash, fingerprintFields := g.approvalFingerprint(reqForInv, lane, tool, risk, level, resolvePaths(g.workspace, reqForInv.Paths))
		scopeSnapshot := map[string]any{
			"laneId":                     lane.ID,
			"toolId":                     tool.ID(),
			"paths":                      req.Paths,
			"executionLevel":             level,
			"domain":                     tool.Domain(),
			"action":                     tool.Action(),
			"correlationId":              req.CorrelationID,
			"initiator":                  req.Initiator,
			"permissionNotes":            reason,
			"approvalFingerprintVersion": "gateway.v1",
			"approvalShapeHash":          fingerprintHash,
			"approvalFingerprintHash":    fingerprintHash,
			"approvalFingerprintFields":  fingerprintFields,
		}
		for k, v := range g.multiActionApprovalScope(req, tool, risk, effectiveJobID) {
			scopeSnapshot[k] = v
		}
		ar, err := g.approvals.OpenRequestForJob(ctx, effectiveJobID, approvals.CreateRequestInput{
			JobID:            effectiveJobID,
			RequestedAction:  fmt.Sprintf("%s.%s", tool.Domain(), tool.Action()),
			RiskClass:        risk,
			RequestedAdapter: "gateway",
			WriteIntent:      tool.WriteIntent(),
			ScopeSnapshot:    scopeSnapshot,
			TaskPacketID:     req.PacketID,
			RequestSummary:   fmt.Sprintf("Gateway action %s via lane %s", tool.ID(), lane.ID),
		})
		if err == nil && ar != nil {
			v := ar.ID
			approvalReqID = &v
			grantHash, grantFields := g.approvalFingerprintForRequestID(reqForInv, lane, tool, risk, level, resolvePaths(g.workspace, reqForInv.Paths), ar.ID)
			scopeSnapshot["approvalRequestId"] = ar.ID
			scopeSnapshot["approvalFingerprintHash"] = grantHash
			scopeSnapshot["approvalFingerprintFields"] = grantFields
			if err := g.updateApprovalRequestScopeSnapshot(ctx, ar.ID, scopeSnapshot); err != nil {
				return nil, fmt.Errorf("store approval fingerprint: %w", err)
			}
		}
	}
	id, err := g.openInvocation(ctx, reqForInv, lane, tool, risk, level, profileID, StatusNeedsApprov, approvalReqID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	_, _ = g.db.ExecContext(ctx, `UPDATE gateway_invocations SET completed_at=?, denied_reason=? WHERE id=?`, now, reason, id)
	_, _ = g.audit.Record(ctx, audit.CreateRequest{
		CorrelationID:       req.CorrelationID,
		Category:            "gateway",
		Action:              "tool.needs_approval",
		Actor:               req.Initiator,
		SubjectType:         "tool",
		SubjectID:           tool.ID(),
		JobID:               reqForInv.JobID,
		GatewayInvocationID: &id,
		ApprovalRequestID:   approvalReqID,
		RiskClass:           risk,
		Outcome:             "needs_approval",
		Summary:             reason,
		Payload:             gatewayAuditContextPayload(req, map[string]any{"reason": reason}),
	})
	data := map[string]any{
		"jobId": effectiveJobID,
	}
	if approvalReqID != nil {
		data["approvalRequestId"] = *approvalReqID
	}
	return &Result{
		InvocationID:   id,
		CorrelationID:  req.CorrelationID,
		TraceID:        req.TraceID,
		Status:         StatusNeedsApprov,
		PolicyOutcome:  OutcomeRequireApproval,
		Allowed:        false,
		DeniedReason:   reason,
		RiskClass:      risk,
		ExecutionLevel: level,
		Lane:           lane.ID,
		Tool:           tool.ID(),
		Domain:         tool.Domain(),
		Action:         tool.Action(),
		CapabilityID:   capabilityIDFromRequest(req),
		ProfileID:      profileID,
		Message:        reason,
		Data:           data,
	}, nil
}

// resolveApprovalJobID returns the job id used for approval_requests FK and a pointer for gateway_invocations.
// Chat and other callers often omit JobID; we insert a minimal gateway_action job row (not enqueued) so approvals can be recorded.
func (g *Gateway) resolveApprovalJobID(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level string) (effectiveJobID string, jobIDPtr *string, err error) {
	if req.JobID != nil {
		if jid := strings.TrimSpace(*req.JobID); jid != "" {
			return jid, req.JobID, nil
		}
	}
	jid, err := g.insertChatGatewayApprovalJob(ctx, req, lane, tool, risk, level)
	if err != nil {
		return "", nil, err
	}
	return jid, &jid, nil
}

func (g *Gateway) insertChatGatewayApprovalJob(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("job id entropy: %w", err)
	}
	id := "chat-gw-" + hex.EncodeToString(b)
	now := time.Now().UnixMilli()
	writeIntent := 0
	if tool.WriteIntent() {
		writeIntent = 1
	}
	userRequest := fmt.Sprintf("Chat gateway: %s (correlation %s)", tool.ID(), req.CorrelationID)
	if raw := strings.TrimSpace(metadataString(req.Metadata, "chatUserRequest")); raw != "" {
		userRequest = raw
	}
	approvalInput := gatewayApprovalRequestInput(tool, req)
	meta := map[string]any{
		"templateId":    "gateway_action",
		"userRequest":   userRequest,
		"objective":     fmt.Sprintf("Execute %s after operator approval", tool.ID()),
		"executionMode": "governed_tool",
		"createdBy":     nonEmpty(req.Initiator, "chat"),
		"requestPayload": map[string]any{
			"toolId":              tool.ID(),
			"laneId":              lane.ID,
			"action":              nonEmpty(req.Action, tool.Action()),
			"domain":              tool.Domain(),
			"riskClass":           risk,
			"executionLevel":      level,
			"correlationId":       req.CorrelationID,
			"paths":               req.Paths,
			"input":               approvalInput,
			"dryRun":              req.DryRun,
			"initiator":           req.Initiator,
			"source":              req.Source,
			"workspaceId":         req.WorkspaceID,
			"provenanceActor":     req.ProvenanceActor,
			"provenanceActorType": req.ProvenanceActorType,
		},
	}
	metaJSON, _ := json.Marshal(meta)
	_, err := g.db.ExecContext(ctx, `
INSERT INTO jobs(
  id, created_at, updated_at, queued_at,
  title, requested_action, target_adapter, initiating_source,
  execution_boundary, risk_class, status, approval_status, write_intent,
  metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id,
		now,
		now,
		nil,
		"Chat gateway approval",
		"gateway.action",
		"forge",
		nonEmpty(req.Initiator, "chat"),
		"command_execution",
		risk,
		"awaiting_approval",
		"pending",
		writeIntent,
		string(metaJSON),
	)
	if err != nil {
		return "", fmt.Errorf("insert chat gateway job: %w", err)
	}
	return id, nil
}

func (g *Gateway) multiActionApprovalScope(req Request, tool Tool, risk, jobID string) map[string]any {
	if tool == nil {
		return nil
	}
	userRequest := strings.TrimSpace(metadataString(req.Metadata, "chatUserRequest"))
	if userRequest == "" {
		userRequest = strings.TrimSpace(metadataString(req.Metadata, "userRequest"))
	}
	if userRequest == "" {
		return nil
	}
	if !looksLikeMultiActionApprovalRequest(userRequest, tool.ID()) {
		return nil
	}
	return map[string]any{
		"approvalHoldOpen":          true,
		"approvalHoldKind":          "multi_action_request",
		"approvalHoldReason":        "operator request contains multiple requested actions; approval remains scoped to this job and correlation",
		"approvalHoldCorrelationId": req.CorrelationID,
		"approvalHoldJobId":         strings.TrimSpace(jobID),
		"approvalHoldMaxRiskRank":   gatewayApprovalRiskRank(risk),
		"approvalHoldToolId":        tool.ID(),
		"approvalHoldUserRequest":   trimForApprovalScope(userRequest, 500),
	}
}

func looksLikeMultiActionApprovalRequest(userRequest, toolID string) bool {
	s := strings.TrimSpace(strings.ToLower(userRequest))
	if s == "" {
		return false
	}
	if toolID == "desktop.open" {
		if _, _, ok := desktopGenericInlineCommand(userRequest); ok {
			return true
		}
	}
	actionWords := 0
	for _, word := range []string{" open ", " launch ", " start ", " run ", " execute ", " create ", " write ", " install ", " refresh ", " build ", " test ", " fetch "} {
		if strings.Contains(" "+s+" ", word) {
			actionWords++
		}
	}
	return actionWords >= 2 && (strings.Contains(s, " and ") || strings.Contains(s, " then ") || strings.Contains(s, ", then ") || strings.Contains(s, ", run "))
}

func trimForApprovalScope(s string, max int) string {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "..."
}

func gatewayApprovalRequestInput(tool Tool, req Request) map[string]any {
	approvalInput := cloneToolOutput(nonNilMap(req.Input))
	if tool != nil && tool.ID() == "desktop.open" {
		if _, ok := approvalInput["query"]; !ok || strings.TrimSpace(fmt.Sprintf("%v", approvalInput["query"])) == "" {
			if raw := strings.TrimSpace(metadataString(req.Metadata, "chatUserRequest")); raw != "" {
				approvalInput["query"] = raw
			}
		}
	}
	return approvalInput
}
