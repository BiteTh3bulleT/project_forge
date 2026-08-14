package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

type capturingUtilityProcessor struct {
	requests []domain.SyscallRequest
	result   domain.SyscallResult
	err      error
}

func (p *capturingUtilityProcessor) Process(_ context.Context, req domain.SyscallRequest) (domain.SyscallResult, error) {
	p.requests = append(p.requests, req)
	if p.err != nil {
		return domain.SyscallResult{}, p.err
	}
	if p.result.Action != "" {
		return p.result, nil
	}
	return domain.SyscallResult{
		Success: true, Action: req.Action, RequestID: req.ID, CorrelationID: req.CorrelationID,
		TraceID: req.TraceID, IdempotencyKey: req.IdempotencyKey,
		ApprovalStatus: domain.ApprovalAllowed, CommittedObjectIDs: []string{"utility-event-test"},
	}, nil
}

func TestRetrievalUsefulnessAPIRoutesOnlyThroughForgeK(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	processor := &capturingUtilityProcessor{}
	srv.retrieval.SetSyscallProcessor(processor)
	body := []byte(`{"label":"useful","note":"confirmed","workspaceId":"ws-api","laneId":"control.semantic","selectedPaths":["/workspace/project"],"idempotencyKey":"retrieval-usefulness-api-1","metadata":{"reason":"operator review"}}`)
	req := withRouteParam(httptest.NewRequest(http.MethodPost, "/api/retrieval/results/42/usefulness", bytes.NewReader(body)), "id", "42")
	rr := httptest.NewRecorder()

	srv.handleMarkRetrievalUsefulness(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(processor.requests) != 1 {
		t.Fatalf("utility syscall count=%d", len(processor.requests))
	}
	got := processor.requests[0]
	if got.Action != "RECORD_RETRIEVAL_USEFULNESS" || got.IdempotencyKey != "retrieval-usefulness-api-1" || got.Payload["resultId"] != int64(42) {
		t.Fatalf("unexpected utility request: %+v", got)
	}
	if got.Scope.WorkspaceID != "ws-api" || got.Scope.LaneID != "control.semantic" || len(got.Scope.SelectedPaths) != 1 {
		t.Fatalf("scope mismatch: %+v", got.Scope)
	}
}

func TestUtilityEvidenceAPIsRequireExactScopeAndIdempotency(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	processor := &capturingUtilityProcessor{}
	srv.retrieval.SetSyscallProcessor(processor)
	req := withRouteParam(httptest.NewRequest(http.MethodPost, "/api/retrieval/results/42/usefulness", bytes.NewBufferString(`{"label":"useful","workspaceId":"ws-api"}`)), "id", "42")
	rr := httptest.NewRecorder()
	srv.handleMarkRetrievalUsefulness(rr, req)
	if rr.Code != http.StatusBadRequest || len(processor.requests) != 0 {
		t.Fatalf("incomplete scope reached Kernel: status=%d requests=%d body=%s", rr.Code, len(processor.requests), rr.Body.String())
	}
}
