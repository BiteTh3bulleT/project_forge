package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPhase3HandlersRejectOversizeRequestBodies(t *testing.T) {
	t.Parallel()

	oversizeBody := `{"notes":"` + strings.Repeat("a", phase3JSONRequestBodyLimit+1) + `"}`
	s := &Server{}

	tests := []struct {
		name    string
		request *http.Request
		handle  func(http.ResponseWriter, *http.Request)
	}{
		{
			name:    "reembed",
			request: httptest.NewRequest(http.MethodPost, "/api/embeddings/reembed", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleReembed,
		},
		{
			name:    "create retrieval run",
			request: httptest.NewRequest(http.MethodPost, "/api/retrieval/runs", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleCreateRetrievalRun,
		},
		{
			name: "mark retrieval usefulness",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPost, "/api/retrieval/results/42/usefulness", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"42",
			),
			handle: s.handleMarkRetrievalUsefulness,
		},
		{
			name:    "create dossier",
			request: httptest.NewRequest(http.MethodPost, "/api/dossiers", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleCreateDossier,
		},
		{
			name: "update dossier",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPatch, "/api/dossiers/42", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"42",
			),
			handle: s.handleUpdateDossier,
		},
		{
			name: "generate dossier brief",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPost, "/api/dossiers/42/brief", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"42",
			),
			handle: s.handleGenerateDossierBrief,
		},
		{
			name:    "create evaluation",
			request: httptest.NewRequest(http.MethodPost, "/api/evaluations", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleCreateEvaluation,
		},
		{
			name: "replay job",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPost, "/api/jobs/job-1/replay", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"job-1",
			),
			handle: s.handleReplayJob,
		},
		{
			name:    "create imported execution",
			request: httptest.NewRequest(http.MethodPost, "/api/imports/executions", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleCreateImportedExecution,
		},
		{
			name:    "generate insights",
			request: httptest.NewRequest(http.MethodPost, "/api/insights/generate", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleGenerateInsights,
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
