package api

import (
	"log/slog"
	"net/http"
	"strings"
)

type apiErrorEnvelope struct {
	Error apiErrorBody `json:"error"`
}

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeAPIError(w http.ResponseWriter, status int, code, message string, err error) {
	code = strings.TrimSpace(code)
	if code == "" {
		code = "request_error"
	}
	message = strings.TrimSpace(message)
	if message == "" {
		message = http.StatusText(status)
	}
	if status >= http.StatusInternalServerError && err != nil {
		apiLogError("api error",
			slog.Int("status", status),
			slog.String("code", code),
			apiLogErr(err),
		)
	}
	writeJSON(w, status, apiErrorEnvelope{Error: apiErrorBody{Code: code, Message: message}})
}

func writeAPIInternalError(w http.ResponseWriter, err error) {
	writeAPIError(w, http.StatusInternalServerError, "internal_error", "internal server error", err)
}

func writeAPIRequestError(w http.ResponseWriter, status int, err error) {
	if status >= http.StatusInternalServerError {
		writeAPIInternalError(w, err)
		return
	}
	message := http.StatusText(status)
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = err.Error()
	}
	writeAPIError(w, status, "request_failed", message, err)
}
