package gateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type InvocationRecord struct {
	ID                  int64           `json:"id"`
	CorrelationID       string          `json:"correlationId"`
	TraceID             string          `json:"traceId,omitempty"`
	CreatedAtMs         int64           `json:"createdAtMs"`
	CompletedAtMs       *int64          `json:"completedAtMs,omitempty"`
	ToolID              string          `json:"toolId"`
	CapabilityID        string          `json:"capabilityId,omitempty"`
	LaneID              *string         `json:"laneId,omitempty"`
	JobID               *string         `json:"jobId,omitempty"`
	PacketID            *int64          `json:"packetId,omitempty"`
	Initiator           string          `json:"initiator"`
	Action              string          `json:"action"`
	Domain              string          `json:"domain"`
	RiskClass           string          `json:"riskClass"`
	ExecutionLevel      string          `json:"executionLevel"`
	PolicyOutcome       string          `json:"policyOutcome"`
	WriteIntent         bool            `json:"writeIntent"`
	Scope               json.RawMessage `json:"scope"`
	Input               json.RawMessage `json:"input"`
	Status              string          `json:"status"`
	DeniedReason        string          `json:"deniedReason"`
	Result              json.RawMessage `json:"result"`
	Artifacts           json.RawMessage `json:"artifacts"`
	PermissionProfileID string          `json:"permissionProfileId"`
	ApprovalRequestID   *int64          `json:"approvalRequestId,omitempty"`
}

func (g *Gateway) ListInvocations(ctx context.Context, limit int, status string) ([]InvocationRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 150
	}
	where := ""
	args := []any{}
	if strings.TrimSpace(status) != "" {
		where = "WHERE status = ?"
		args = append(args, status)
	}
	args = append(args, limit)
	query := fmt.Sprintf(`
SELECT id, correlation_id, created_at, completed_at, tool_id, lane_id, job_id, packet_id,
       initiator, action, risk_class, write_intent, scope_json, input_json,
       status, denied_reason, result_json, artifacts_json, permission_profile_id, approval_request_id
FROM gateway_invocations %s ORDER BY id DESC LIMIT ?`, where)
	rows, err := g.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInvocationRows(rows)
}

// ListInvocationsByCorrelation returns invocation rows for one correlation id.
func (g *Gateway) ListInvocationsByCorrelation(ctx context.Context, correlationID string, limit int) ([]InvocationRecord, error) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return []InvocationRecord{}, nil
	}
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	rows, err := g.db.QueryContext(ctx, `
SELECT id, correlation_id, created_at, completed_at, tool_id, lane_id, job_id, packet_id,
       initiator, action, risk_class, write_intent, scope_json, input_json,
       status, denied_reason, result_json, artifacts_json, permission_profile_id, approval_request_id
FROM gateway_invocations
WHERE correlation_id = ?
ORDER BY id ASC LIMIT ?`, correlationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInvocationRows(rows)
}

func scanInvocationRows(rows *sql.Rows) ([]InvocationRecord, error) {
	out := []InvocationRecord{}
	for rows.Next() {
		var r InvocationRecord
		var completed sql.NullInt64
		var lane sql.NullString
		var job sql.NullString
		var packet sql.NullInt64
		var approvalReq sql.NullInt64
		var profile sql.NullString
		var writeInt int
		var scope, input, result, artifacts string
		if err := rows.Scan(
			&r.ID, &r.CorrelationID, &r.CreatedAtMs, &completed, &r.ToolID, &lane, &job, &packet,
			&r.Initiator, &r.Action, &r.RiskClass, &writeInt, &scope, &input,
			&r.Status, &r.DeniedReason, &result, &artifacts, &profile, &approvalReq,
		); err != nil {
			return nil, err
		}
		if completed.Valid {
			v := completed.Int64
			r.CompletedAtMs = &v
		}
		if lane.Valid {
			v := lane.String
			r.LaneID = &v
		}
		if job.Valid {
			v := job.String
			r.JobID = &v
		}
		if packet.Valid {
			v := packet.Int64
			r.PacketID = &v
		}
		if profile.Valid {
			r.PermissionProfileID = profile.String
		}
		r.Domain = toolDomainFromID(r.ToolID)
		r.ExecutionLevel = executionLevelFromRisk(r.RiskClass)
		switch r.Status {
		case StatusDenied:
			r.PolicyOutcome = OutcomeDeny
		case StatusNeedsApprov:
			r.PolicyOutcome = OutcomeRequireApproval
		default:
			r.PolicyOutcome = OutcomeAllow
		}
		if approvalReq.Valid {
			v := approvalReq.Int64
			r.ApprovalRequestID = &v
		}
		r.WriteIntent = writeInt == 1
		r.Scope = json.RawMessage(scope)
		r.Input = json.RawMessage(input)
		r.Result = json.RawMessage(result)
		r.Artifacts = json.RawMessage(artifacts)
		scopeObj := map[string]any{}
		if err := json.Unmarshal([]byte(scope), &scopeObj); err == nil {
			r.TraceID = metadataString(scopeObj, "traceId")
			r.CapabilityID = metadataString(scopeObj, "capabilityId")
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
