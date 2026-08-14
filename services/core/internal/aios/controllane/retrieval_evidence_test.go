package controllane

import (
	"math"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestRecordRetrievalEvidenceValidationFailsClosedOnTampering(t *testing.T) {
	valid := retrievalEvidenceTestRequest()
	if issues := validateRecordRetrievalEvidence(valid); len(issues) != 0 {
		t.Fatalf("valid retrieval evidence rejected: %+v", issues)
	}

	tests := []struct {
		name   string
		mutate func(*domain.SyscallRequest)
		field  string
	}{
		{name: "missing selected paths", mutate: func(req *domain.SyscallRequest) { req.Scope.SelectedPaths = nil }, field: "scope.selectedPaths"},
		{name: "blank selected path", mutate: func(req *domain.SyscallRequest) { req.Scope.SelectedPaths = []string{" "} }, field: "scope.selectedPaths[0]"},
		{name: "relative selected path", mutate: func(req *domain.SyscallRequest) { req.Scope.SelectedPaths = []string{"repo"} }, field: "scope.selectedPaths[0]"},
		{name: "nondeterministic selected path order", mutate: func(req *domain.SyscallRequest) { req.Scope.SelectedPaths = []string{"/z", "/a"} }, field: "scope.selectedPaths"},
		{name: "run evidence id", mutate: func(req *domain.SyscallRequest) { req.Payload["evidenceId"] = "forged" }, field: "payload.evidenceId"},
		{name: "creation time", mutate: func(req *domain.SyscallRequest) { req.Payload["createdAt"] = req.RequestedAt + 1 }, field: "payload.createdAt"},
		{name: "result evidence id", mutate: func(req *domain.SyscallRequest) { firstResult(req)["evidenceId"] = "forged" }, field: "payload.results[0].evidenceId"},
		{name: "rank order", mutate: func(req *domain.SyscallRequest) { firstResult(req)["rankIndex"] = 1 }, field: "payload.results[0].rankIndex"},
		{name: "result outside sealed roots", mutate: func(req *domain.SyscallRequest) { firstResult(req)["absPath"] = "/other/a.go" }, field: "payload.results[0].absPath"},
		{name: "nonfinite score", mutate: func(req *domain.SyscallRequest) { firstResult(req)["hybridScore"] = math.Inf(1) }, field: "payload"},
		{name: "legacy signal evidence", mutate: func(req *domain.SyscallRequest) {
			req.Payload["signalEvidence"] = []any{map[string]any{"observationId": 1}}
		}, field: "payload"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := retrievalEvidenceTestRequest()
			test.mutate(&req)
			issues := validateRecordRetrievalEvidence(req)
			if !containsIssueField(issues, test.field) {
				t.Fatalf("tamper field %q not rejected: %+v", test.field, issues)
			}
		})
	}
}

func TestRecordRetrievalEvidenceCommitPlanUsesExactOrderedEvidenceIDs(t *testing.T) {
	req := retrievalEvidenceTestRequest()
	ids, err := expectedCommitObjectIDs(req, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{req.ID + ":retrieval_run", req.ID + ":retrieval_result:0"}
	if len(ids) != len(want) {
		t.Fatalf("object ids=%v want=%v", ids, want)
	}
	for index := range want {
		if ids[index] != want[index] {
			t.Fatalf("object ids=%v want=%v", ids, want)
		}
	}
}

func retrievalEvidenceTestRequest() domain.SyscallRequest {
	id := "retrieval-proof-1"
	return domain.SyscallRequest{
		ID: id, Action: domain.ActionRecordRetrievalEvidence,
		Actor:  domain.ActorIdentity{ID: "forge.core", Kind: "service"},
		Source: domain.SourceInternal,
		Scope: domain.ForgeScope{
			WorkspaceID: "ws-retrieval", LaneID: "control.semantic", SelectedPaths: []string{"/repo"},
		},
		Payload: map[string]any{
			"evidenceId": id + ":retrieval_run", "createdAt": int64(100),
			"query": "kernel proof", "mode": "keyword", "weighting": map[string]any{"keyword": 1.0},
			"results": []any{map[string]any{
				"evidenceId": id + ":retrieval_result:0", "chunkId": int64(7), "fileId": int64(3),
				"absPath": "/repo/a.go", "relPath": "a.go", "rankIndex": 0,
				"keywordScore": 1.0, "semanticScore": 0.0, "hybridScore": 1.0,
				"snippet": "proof", "selectedForPacket": true,
				"selectionReason": map[string]any{"baseScore": 1.0},
			}},
		},
		Provenance:  domain.Provenance{Actor: "forge.core", ActorType: "system", Source: "test"},
		RequestedAt: 100,
	}
}

func firstResult(req *domain.SyscallRequest) map[string]any {
	return req.Payload["results"].([]any)[0].(map[string]any)
}

func containsIssueField(issues []domain.SyscallError, field string) bool {
	for _, issue := range issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}
