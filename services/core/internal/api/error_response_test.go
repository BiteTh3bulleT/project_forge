package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteAPIErrorProducesStructuredJSON(t *testing.T) {
	rr := httptest.NewRecorder()

	writeAPIError(rr, http.StatusBadRequest, "bad_request", "bad id", errors.New("internal detail"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content type=%q, want application/json", ct)
	}
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error payload: %v body=%s", err, rr.Body.String())
	}
	if payload.Error.Code != "bad_request" || payload.Error.Message != "bad id" {
		t.Fatalf("unexpected payload: %+v", payload.Error)
	}
	if strings.Contains(rr.Body.String(), "internal detail") {
		t.Fatalf("structured error leaked internal detail: %s", rr.Body.String())
	}
}

func TestWriteAPIInternalErrorDoesNotExposeGoErrorString(t *testing.T) {
	rr := httptest.NewRecorder()

	writeAPIInternalError(rr, errors.New("sql: secret failure detail"))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", rr.Code)
	}
	if strings.Contains(rr.Body.String(), "sql: secret") {
		t.Fatalf("internal error leaked detail: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"code":"internal_error"`) {
		t.Fatalf("expected internal_error code, got %s", rr.Body.String())
	}
}

func TestWritePhase2DecodeErrorUsesStructuredJSON(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		code string
		want int
	}{
		{name: "oversize", err: errPhase2RequestBodyTooLarge, code: "request_body_too_large", want: http.StatusRequestEntityTooLarge},
		{name: "invalid", err: errors.New("bad json"), code: "invalid_json", want: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			writePhase2DecodeError(rr, tc.err)

			if rr.Code != tc.want {
				t.Fatalf("status=%d, want %d", rr.Code, tc.want)
			}
			if !strings.Contains(rr.Body.String(), `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %q in body %s", tc.code, rr.Body.String())
			}
		})
	}
}
