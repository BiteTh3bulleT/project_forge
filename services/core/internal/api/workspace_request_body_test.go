package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestWorkspaceJSONHandlersRejectOversizeRequestBodies(t *testing.T) {
	t.Parallel()

	oversizeBody := `{"title":"` + strings.Repeat("a", workspaceJSONRequestBodyLimit+1) + `"}`
	s := &Server{}

	tests := []struct {
		name    string
		request *http.Request
		handle  func(http.ResponseWriter, *http.Request)
	}{
		{
			name:    "chat thread create",
			request: httptest.NewRequest(http.MethodPost, "/api/chat/threads", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleChatThreadCreate,
		},
		{
			name: "chat thread patch",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPatch, "/api/chat/threads/42", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"42",
			),
			handle: s.handleChatThreadPatch,
		},
		{
			name: "chat job create",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPost, "/api/chat/threads/42/jobs", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"42",
			),
			handle: s.handleChatJobCreate,
		},
		{
			name: "chat message post",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPost, "/api/chat/threads/42/messages", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"42",
			),
			handle: s.handleChatMessagePost,
		},
		{
			name:    "canvas board create",
			request: httptest.NewRequest(http.MethodPost, "/api/canvas/boards", bytes.NewReader([]byte(oversizeBody))),
			handle:  s.handleCanvasBoardCreate,
		},
		{
			name: "canvas note create",
			request: withRouteParam(
				httptest.NewRequest(http.MethodPost, "/api/canvas/boards/42/notes", bytes.NewReader([]byte(oversizeBody))),
				"id",
				"42",
			),
			handle: s.handleCanvasNoteCreate,
		},
		{
			name: "canvas note patch",
			request: withWorkspaceRouteParams(
				httptest.NewRequest(http.MethodPatch, "/api/canvas/boards/42/notes/7", bytes.NewReader([]byte(oversizeBody))),
				map[string]string{"id": "42", "noteId": "7"},
			),
			handle: s.handleCanvasNotePatch,
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

func withWorkspaceRouteParams(req *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
