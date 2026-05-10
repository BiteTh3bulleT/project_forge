package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPhase4HandlersRejectOversizeRequestBodies(t *testing.T) {
	t.Parallel()

	oversizeBody := `{"name":"` + strings.Repeat("a", phase4JSONRequestBodyLimit+1) + `"}`
	s := &Server{}

	tests := []struct {
		name    string
		request *http.Request
		handle  func(http.ResponseWriter, *http.Request)
	}{
		{name: "save strategy", request: httptest.NewRequest(http.MethodPost, "/api/strategies", bytes.NewReader([]byte(oversizeBody))), handle: s.handleSaveStrategy},
		{name: "save approval preset", request: httptest.NewRequest(http.MethodPost, "/api/policy/approval-presets", bytes.NewReader([]byte(oversizeBody))), handle: s.handleSaveApprovalPreset},
		{name: "set global preset", request: httptest.NewRequest(http.MethodPost, "/api/policy/global-preset", bytes.NewReader([]byte(oversizeBody))), handle: s.handleSetGlobalPreset},
		{
			name:    "save dossier profile",
			request: withRouteParam(httptest.NewRequest(http.MethodPost, "/api/dossiers/42/policy-profile", bytes.NewReader([]byte(oversizeBody))), "id", "42"),
			handle:  s.handleSaveDossierProfile,
		},
		{name: "policy recommend", request: httptest.NewRequest(http.MethodPost, "/api/policy/recommend", bytes.NewReader([]byte(oversizeBody))), handle: s.handlePolicyRecommend},
		{name: "save automation rule", request: httptest.NewRequest(http.MethodPost, "/api/automation/rules", bytes.NewReader([]byte(oversizeBody))), handle: s.handleSaveAutomationRule},
		{name: "run automation rule", request: httptest.NewRequest(http.MethodPost, "/api/automation/run", bytes.NewReader([]byte(oversizeBody))), handle: s.handleRunAutomationRule},
		{name: "analyze packet guidance", request: httptest.NewRequest(http.MethodPost, "/api/packet-guidance/analyze", bytes.NewReader([]byte(oversizeBody))), handle: s.handleAnalyzePacketGuidance},
		{
			name:    "save import reconciliation",
			request: withRouteParam(httptest.NewRequest(http.MethodPost, "/api/imports/42/reconciliation", bytes.NewReader([]byte(oversizeBody))), "id", "42"),
			handle:  s.handleSaveImportReconciliation,
		},
		{name: "create review", request: httptest.NewRequest(http.MethodPost, "/api/reviews", bytes.NewReader([]byte(oversizeBody))), handle: s.handleCreateReview},
		{
			name:    "update review",
			request: withRouteParam(httptest.NewRequest(http.MethodPatch, "/api/reviews/42", bytes.NewReader([]byte(oversizeBody))), "id", "42"),
			handle:  s.handleUpdateReview,
		},
		{name: "analyze failure patterns", request: httptest.NewRequest(http.MethodPost, "/api/failures/analyze", bytes.NewReader([]byte(oversizeBody))), handle: s.handleAnalyzeFailurePatterns},
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
