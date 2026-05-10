package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPhase5HandlersRejectOversizeRequestBodies(t *testing.T) {
	t.Parallel()

	oversizeBody := `{"notes":"` + strings.Repeat("a", int(phase5JSONRequestBodyLimit)+1) + `"}`
	s := &Server{}

	tests := []struct {
		name    string
		request *http.Request
		handle  func(http.ResponseWriter, *http.Request)
	}{
		{
			name:    "gateway invoke",
			request: httptest.NewRequest(http.MethodPost, "/api/gateway/invoke", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleGatewayInvoke,
		},
		{
			name: "gateway capability status update",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPatch, "/api/gateway/capabilities/filesystem.read_file/status", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"filesystem.read_file",
			),
			handle: s.handleGatewayCapabilityStatusUpdate,
		},
		{
			name:    "create bundle",
			request: httptest.NewRequest(http.MethodPost, "/api/backup/bundles", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleCreateBundle,
		},
		{
			name:    "restore bundle",
			request: httptest.NewRequest(http.MethodPost, "/api/backup/restore", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleRestoreBundle,
		},
		{
			name:    "save lane",
			request: httptest.NewRequest(http.MethodPost, "/api/lanes", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleSaveLane,
		},
		{
			name:    "save permission profile",
			request: httptest.NewRequest(http.MethodPost, "/api/permissions/profiles", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleSavePermissionProfile,
		},
		{
			name:    "release record",
			request: httptest.NewRequest(http.MethodPost, "/api/release/artifacts", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleReleaseRecord,
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
