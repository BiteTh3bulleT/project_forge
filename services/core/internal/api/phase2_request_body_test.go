package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPhase2HandlersRejectOversizeRequestBodies(t *testing.T) {
	t.Parallel()

	oversizeBody := `{"note":"` + strings.Repeat("a", phase2JSONRequestBodyLimit+1) + `"}`
	s := &Server{}

	tests := []struct {
		name    string
		request *http.Request
		handle  func(http.ResponseWriter, *http.Request)
	}{
		{
			name:    "create job",
			request: httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleCreateJob,
		},
		{
			name: "cancel job",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPost, "/api/jobs/job-1/cancel", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"job-1",
			),
			handle: s.handleCancelJob,
		},
		{
			name: "approve request",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPost, "/api/approvals/42/approve", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"42",
			),
			handle: s.handleApproveRequest,
		},
		{
			name:    "import project context",
			request: httptest.NewRequest(http.MethodPost, "/api/project-context/import", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleImportProjectContext,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.handle(rr, tc.request)

			if rr.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusRequestEntityTooLarge, rr.Body.String())
			}
			if !strings.Contains(strings.ToLower(rr.Body.String()), "too large") {
				t.Fatalf("expected too-large response, got %q", rr.Body.String())
			}
		})
	}
}
