package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMemoryMutationHandlersRejectOversizeRequestBodies(t *testing.T) {
	t.Parallel()

	oversizeBody := `{"note":"` + strings.Repeat("a", memoryMutationRequestBodyLimit+1) + `"}`
	s := &Server{}

	tests := []struct {
		name    string
		request *http.Request
		handle  func(http.ResponseWriter, *http.Request)
	}{
		{
			name:    "create observation",
			request: httptest.NewRequest(http.MethodPost, "/api/memory/observations", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleCreateMemoryObservation,
		},
		{
			name: "patch observation",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPatch, "/api/memory/observations/42", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"42",
			),
			handle: s.handlePatchMemoryObservation,
		},
		{
			name: "mark usefulness",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPost, "/api/memory/observations/42/usefulness", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"42",
			),
			handle: s.handleMarkMemoryObservationUsefulness,
		},
		{
			name:    "repair run",
			request: httptest.NewRequest(http.MethodPost, "/api/memory/repair/run", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleRunMemoryRepair,
		},
		{
			name:    "vsa reindex run",
			request: httptest.NewRequest(http.MethodPost, "/api/memory/vsa/reindex/run", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleRunVSAReindex,
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
