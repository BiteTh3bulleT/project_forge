package api

import (
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/gateway"
	"forge/projectforge/services/core/internal/jobs"
)

func TestLatestGatewayProbeSnapshot(t *testing.T) {
	t.Parallel()

	s := &Server{}
	th := &chat.ThreadDetail{
		Messages: []chat.Message{
			{
				Role: "assistant",
				Metadata: map[string]any{
					"toolGatewayActivity": map[string]any{
						"executionState": "ok",
						"toolSelected":   "fs.mkdir",
					},
					"correlationId": "chat-tools-1",
				},
			},
			{
				Role: "assistant",
				Metadata: map[string]any{
					"toolGatewayActivity": map[string]any{
						"executionState": gateway.StatusNeedsApprov,
						"executionResult": map[string]any{
							"approvalRequestId": 10,
							"jobId":             "job_123",
							"path":              "test_project/banner.py",
							"bytes":             2037,
						},
					},
					"correlationId": "chat-tools-2",
				},
			},
		},
	}

	snap, ok := s.latestGatewayProbeSnapshot(t.Context(), th)
	if !ok {
		t.Fatalf("expected probe snapshot to be found")
	}
	if snap.ExecutionState != gateway.StatusNeedsApprov {
		t.Fatalf("execution state = %q, want %q", snap.ExecutionState, gateway.StatusNeedsApprov)
	}
	if snap.JobID != "job_123" {
		t.Fatalf("job id = %q, want job_123", snap.JobID)
	}
	if snap.ApprovalReqID != 10 {
		t.Fatalf("approval request id = %d, want 10", snap.ApprovalReqID)
	}
	if snap.Path != "test_project/banner.py" {
		t.Fatalf("path = %q, want test_project/banner.py", snap.Path)
	}
	if snap.Bytes != 2037 {
		t.Fatalf("bytes = %d, want 2037", snap.Bytes)
	}
}

func TestLatestGatewayProbeSnapshotNotFound(t *testing.T) {
	t.Parallel()
	s := &Server{}
	th := &chat.ThreadDetail{
		Messages: []chat.Message{
			{Role: "assistant", Metadata: map[string]any{"noop": true}},
			{Role: "user", Metadata: map[string]any{}},
		},
	}

	_, ok := s.latestGatewayProbeSnapshot(t.Context(), th)
	if ok {
		t.Fatalf("expected no snapshot")
	}
}

func TestFormatGatewayProbeReplySuccessOverridesStaleApproval(t *testing.T) {
	t.Parallel()

	snap := gatewayProbeSnapshot{
		ExecutionState:  gateway.StatusNeedsApprov,
		InvocationState: gateway.StatusNeedsApprov,
		JobID:           "job_abc",
		JobStatus:       jobs.StatusSucceeded,
		ToolID:          "fs.write",
		Path:            "services/core/test_project/banner.py",
		Bytes:           2037,
	}

	reply := formatGatewayProbeReply(snap)
	if !strings.Contains(reply, "Yes, it worked.") {
		t.Fatalf("expected success reply, got %q", reply)
	}
	if !strings.Contains(reply, "job_abc") {
		t.Fatalf("expected job id in reply, got %q", reply)
	}
	if !strings.Contains(reply, "2037") {
		t.Fatalf("expected bytes in reply, got %q", reply)
	}
}

func TestFormatGatewayProbeReplyWaitingApproval(t *testing.T) {
	t.Parallel()

	snap := gatewayProbeSnapshot{
		ExecutionState: gateway.StatusNeedsApprov,
		ApprovalReqID:  42,
	}

	reply := formatGatewayProbeReply(snap)
	if !strings.Contains(reply, "waiting on operator approval") {
		t.Fatalf("expected waiting approval reply, got %q", reply)
	}
	if !strings.Contains(reply, "#42") {
		t.Fatalf("expected approval request id in reply, got %q", reply)
	}
}
