package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/gateway"
	"forge/projectforge/services/core/internal/jobs"
)

type gatewayProbeSnapshot struct {
	ExecutionState  string
	JobID           string
	CorrelationID   string
	ToolID          string
	ApprovalReqID   int64
	InvocationState string
	DeniedReason    string
	Path            string
	Bytes           int64
	JobStatus       jobs.Status
	JobFailure      string
}

func (s *Server) maybeRespondGatewayStatusProbe(
	ctx context.Context,
	threadID, userMessageID int64,
	th *chat.ThreadDetail,
	lastUserContent string,
) *chat.Message {
	if !gateway.IsLikelyStatusProbeTurn(lastUserContent) || th == nil {
		return nil
	}
	snap, ok := s.latestGatewayProbeSnapshot(ctx, th)
	if !ok {
		return nil
	}
	text := formatGatewayProbeReply(snap)
	am, _ := s.chat.AppendMessage(ctx, threadID, "assistant", text, map[string]any{
		"command":              "gateway_status_probe",
		"replyToUserMessageId": userMessageID,
		"gatewayStatusProbe": map[string]any{
			"jobId":           snap.JobID,
			"correlationId":   snap.CorrelationID,
			"toolId":          snap.ToolID,
			"executionState":  snap.ExecutionState,
			"invocationState": snap.InvocationState,
			"approvalRequest": snap.ApprovalReqID,
		},
	})
	return am
}

func (s *Server) latestGatewayProbeSnapshot(ctx context.Context, th *chat.ThreadDetail) (gatewayProbeSnapshot, bool) {
	var snap gatewayProbeSnapshot
	for i := len(th.Messages) - 1; i >= 0; i-- {
		msg := th.Messages[i]
		if !strings.EqualFold(strings.TrimSpace(msg.Role), "assistant") || msg.Metadata == nil {
			continue
		}
		rawActivity, ok := msg.Metadata["toolGatewayActivity"]
		if !ok {
			continue
		}
		activity, ok := rawActivity.(map[string]any)
		if !ok || activity == nil {
			continue
		}
		snap.ExecutionState = strings.TrimSpace(asString(activity["executionState"]))
		snap.ApprovalReqID = findApprovalRequestID(activity)
		snap.JobID = strings.TrimSpace(findJobID(activity))
		snap.CorrelationID = strings.TrimSpace(asString(msg.Metadata["correlationId"]))
		snap.ToolID = strings.TrimSpace(asString(activity["toolSelected"]))

		if execMap, ok := activity["executionResult"].(map[string]any); ok {
			if snap.ToolID == "" {
				snap.ToolID = strings.TrimSpace(asString(execMap["tool"]))
			}
			if snap.CorrelationID == "" {
				snap.CorrelationID = strings.TrimSpace(asString(execMap["correlationId"]))
			}
			if snap.Path == "" {
				snap.Path = strings.TrimSpace(asString(execMap["path"]))
			}
			if snap.Bytes == 0 {
				snap.Bytes = parseAnyInt64(execMap["bytes"])
			}
		}
		break
	}
	if strings.TrimSpace(snap.JobID) == "" && strings.TrimSpace(snap.CorrelationID) == "" && strings.TrimSpace(snap.ExecutionState) == "" {
		return snap, false
	}
	s.enrichGatewayProbeSnapshot(ctx, &snap)
	return snap, true
}

func (s *Server) enrichGatewayProbeSnapshot(ctx context.Context, snap *gatewayProbeSnapshot) {
	if snap == nil {
		return
	}
	if strings.TrimSpace(snap.JobID) != "" && s.jobs != nil {
		if j, err := s.jobs.Get(ctx, snap.JobID); err == nil && j != nil {
			snap.JobStatus = j.Status
			if j.LastError != nil && strings.TrimSpace(*j.LastError) != "" {
				snap.JobFailure = strings.TrimSpace(*j.LastError)
			} else if j.FailureInfo != nil && strings.TrimSpace(*j.FailureInfo) != "" {
				snap.JobFailure = strings.TrimSpace(*j.FailureInfo)
			}
		}
	}
	if strings.TrimSpace(snap.JobID) != "" {
		if state, toolID, denied, path, bytes, ok := s.latestInvocationByJob(ctx, snap.JobID); ok {
			if snap.InvocationState == "" {
				snap.InvocationState = state
			}
			if snap.ToolID == "" {
				snap.ToolID = toolID
			}
			if snap.DeniedReason == "" {
				snap.DeniedReason = denied
			}
			if snap.Path == "" {
				snap.Path = path
			}
			if snap.Bytes == 0 {
				snap.Bytes = bytes
			}
		}
		return
	}
	if strings.TrimSpace(snap.CorrelationID) != "" {
		if state, toolID, denied, path, bytes, ok := s.latestInvocationByCorrelation(ctx, snap.CorrelationID); ok {
			if snap.InvocationState == "" {
				snap.InvocationState = state
			}
			if snap.ToolID == "" {
				snap.ToolID = toolID
			}
			if snap.DeniedReason == "" {
				snap.DeniedReason = denied
			}
			if snap.Path == "" {
				snap.Path = path
			}
			if snap.Bytes == 0 {
				snap.Bytes = bytes
			}
		}
	}
}

func (s *Server) latestInvocationByJob(ctx context.Context, jobID string) (state, toolID, denied, path string, bytes int64, ok bool) {
	if s == nil || s.st == nil || s.st.DB == nil || strings.TrimSpace(jobID) == "" {
		return "", "", "", "", 0, false
	}
	var resultJSON sql.NullString
	err := s.st.DB.QueryRowContext(ctx, `
SELECT status, tool_id, denied_reason, result_json
FROM gateway_invocations
WHERE job_id = ?
ORDER BY id DESC
LIMIT 1`, strings.TrimSpace(jobID)).Scan(&state, &toolID, &denied, &resultJSON)
	if err != nil {
		return "", "", "", "", 0, false
	}
	path, bytes = extractPathAndBytes(resultJSON.String)
	return state, toolID, denied, path, bytes, true
}

func (s *Server) latestInvocationByCorrelation(ctx context.Context, correlationID string) (state, toolID, denied, path string, bytes int64, ok bool) {
	if s == nil || s.st == nil || s.st.DB == nil || strings.TrimSpace(correlationID) == "" {
		return "", "", "", "", 0, false
	}
	var resultJSON sql.NullString
	err := s.st.DB.QueryRowContext(ctx, `
SELECT status, tool_id, denied_reason, result_json
FROM gateway_invocations
WHERE correlation_id = ?
ORDER BY id DESC
LIMIT 1`, strings.TrimSpace(correlationID)).Scan(&state, &toolID, &denied, &resultJSON)
	if err != nil {
		return "", "", "", "", 0, false
	}
	path, bytes = extractPathAndBytes(resultJSON.String)
	return state, toolID, denied, path, bytes, true
}

func extractPathAndBytes(resultJSON string) (string, int64) {
	resultJSON = strings.TrimSpace(resultJSON)
	if resultJSON == "" {
		return "", 0
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(resultJSON), &m); err != nil {
		return "", 0
	}
	path := strings.TrimSpace(asString(m["path"]))
	bytes := parseAnyInt64(m["bytes"])
	return path, bytes
}

func formatGatewayProbeReply(snap gatewayProbeSnapshot) string {
	worked := snap.JobStatus == jobs.StatusSucceeded || strings.EqualFold(snap.InvocationState, gateway.StatusOK) || strings.EqualFold(snap.ExecutionState, "ok")
	waitingApproval := snap.JobStatus == jobs.StatusAwaitingApproval || strings.EqualFold(snap.InvocationState, gateway.StatusNeedsApprov) || strings.EqualFold(snap.ExecutionState, gateway.StatusNeedsApprov)
	inProgress := snap.JobStatus == jobs.StatusQueued || snap.JobStatus == jobs.StatusPreparing || snap.JobStatus == jobs.StatusRunning
	failed := snap.JobStatus == jobs.StatusFailed || snap.JobStatus == jobs.StatusCancelled || strings.EqualFold(snap.InvocationState, gateway.StatusDenied) || strings.EqualFold(snap.InvocationState, gateway.StatusError)

	if worked {
		var b strings.Builder
		b.WriteString("Yes, it worked.")
		if strings.TrimSpace(snap.ToolID) != "" {
			b.WriteString(fmt.Sprintf(" FORGE completed `%s` successfully.", snap.ToolID))
		}
		if strings.TrimSpace(snap.Path) != "" && snap.Bytes > 0 {
			b.WriteString(fmt.Sprintf(" It wrote %d bytes to `%s`.", snap.Bytes, snap.Path))
		} else if strings.TrimSpace(snap.Path) != "" {
			b.WriteString(fmt.Sprintf(" Output path: `%s`.", snap.Path))
		}
		if strings.TrimSpace(snap.JobID) != "" {
			b.WriteString(fmt.Sprintf(" Job `%s` is `succeeded`.", snap.JobID))
		}
		return b.String()
	}

	if waitingApproval {
		if snap.ApprovalReqID > 0 {
			return fmt.Sprintf("Not yet. It is waiting on operator approval (request #%d).", snap.ApprovalReqID)
		}
		return "Not yet. It is still waiting on operator approval."
	}

	if inProgress {
		return fmt.Sprintf("Not finished yet. The gateway job is currently `%s`.", snap.JobStatus)
	}

	if failed {
		reason := strings.TrimSpace(snap.JobFailure)
		if reason == "" {
			reason = strings.TrimSpace(snap.DeniedReason)
		}
		if reason == "" {
			return "It did not complete successfully."
		}
		return "It did not complete successfully: " + reason
	}

	if strings.TrimSpace(snap.ExecutionState) != "" {
		return fmt.Sprintf("Latest gateway execution state is `%s`.", snap.ExecutionState)
	}
	return "I could not verify a completed gateway result yet."
}
