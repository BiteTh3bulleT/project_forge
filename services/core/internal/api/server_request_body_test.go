package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerJSONHandlersRejectOversizeRequestBodies(t *testing.T) {
	t.Parallel()

	oversizeBody := `{"value":"` + strings.Repeat("a", int(serverJSONRequestBodyLimit)+1) + `"}`
	s := &Server{}

	tests := []struct {
		name    string
		request *http.Request
		handle  func(http.ResponseWriter, *http.Request)
	}{
		{
			name:    "patch settings",
			request: httptest.NewRequest(http.MethodPatch, "/api/settings", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handlePatchSettings,
		},
		{
			name:    "add source",
			request: httptest.NewRequest(http.MethodPost, "/api/sources", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleAddSource,
		},
		{
			name:    "command execute",
			request: httptest.NewRequest(http.MethodPost, "/api/commands/execute", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleCommandExecute,
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
