package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const serverJSONRequestBodyLimit int64 = 1 << 20

var errServerRequestBodyTooLarge = errors.New("server json request body too large")

func decodeServerJSONBody(r *http.Request, target any) error {
	if r.Body == nil {
		return io.EOF
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, serverJSONRequestBodyLimit+1))
	if err != nil {
		return err
	}
	if int64(len(raw)) > serverJSONRequestBodyLimit {
		return errServerRequestBodyTooLarge
	}
	return json.Unmarshal(raw, target)
}

func decodeOptionalServerJSONBody(r *http.Request, target any) error {
	if r.Body == nil {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, serverJSONRequestBodyLimit+1))
	if err != nil {
		return err
	}
	if int64(len(raw)) > serverJSONRequestBodyLimit {
		return errServerRequestBodyTooLarge
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	return json.Unmarshal(raw, target)
}

func writeServerDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errServerRequestBodyTooLarge) {
		writeAPIError(w, http.StatusRequestEntityTooLarge, "request_failed", "server json request body too large", nil)
		return
	}
	writeAPIError(w, http.StatusBadRequest, "request_failed", "invalid json", nil)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
