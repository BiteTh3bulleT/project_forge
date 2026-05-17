package api

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

var (
	logSecretAssignmentPattern = regexp.MustCompile(`(?i)\b(token|api[_-]?key|key|secret|password|authorization)=([^&\s]+)`)
	logBearerPattern           = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/\-]+=*`)
)

type safeLogFormatter struct{}

func (safeLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	entry := &safeLogEntry{buf: &bytes.Buffer{}}
	reqID := middleware.GetReqID(r.Context())
	if reqID != "" {
		entry.buf.WriteString("[")
		entry.buf.WriteString(reqID)
		entry.buf.WriteString("] ")
	}
	entry.buf.WriteString(`"`)
	entry.buf.WriteString(r.Method)
	entry.buf.WriteString(" ")
	if r.URL != nil {
		entry.buf.WriteString(r.URL.Path)
	}
	entry.buf.WriteString(" ")
	entry.buf.WriteString(r.Proto)
	entry.buf.WriteString(`" from `)
	entry.buf.WriteString(r.RemoteAddr)
	entry.buf.WriteString(" - ")
	return entry
}

type safeLogEntry struct {
	buf *bytes.Buffer
}

func (e *safeLogEntry) Write(status, bytes int, _ http.Header, elapsed time.Duration, _ interface{}) {
	slog.Info("api request completed",
		slog.String("request", strings.TrimSpace(e.buf.String())),
		slog.Int("status", status),
		slog.Int("bytes", bytes),
		slog.Duration("elapsed", elapsed),
	)
}

func (e *safeLogEntry) Panic(_ interface{}, _ []byte) {}

func apiLogInfo(message string, attrs ...slog.Attr) {
	slog.LogAttrs(contextlessLogContext(), slog.LevelInfo, message, attrs...)
}

func apiLogWarn(message string, attrs ...slog.Attr) {
	slog.LogAttrs(contextlessLogContext(), slog.LevelWarn, message, attrs...)
}

func apiLogError(message string, attrs ...slog.Attr) {
	slog.LogAttrs(contextlessLogContext(), slog.LevelError, message, attrs...)
}

func apiLogErr(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "")
	}
	return slog.String("error", sanitizeLogText(err.Error()))
}

func sanitizeLogText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = logBearerPattern.ReplaceAllString(raw, "Bearer [redacted]")
	raw = logSecretAssignmentPattern.ReplaceAllString(raw, "$1=[redacted]")
	fields := strings.Fields(raw)
	for i, field := range fields {
		fields[i] = sanitizeLogField(field)
	}
	return strings.Join(fields, " ")
}

func sanitizeLogField(field string) string {
	trimmed := strings.Trim(field, `"'()[]{}<>.,;`)
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		if idx := strings.Index(field, trimmed); idx >= 0 {
			safeURL := sanitizeLogURL(trimmed)
			return field[:idx] + safeURL + field[idx+len(trimmed):]
		}
	}
	return field
}

func sanitizeLogURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return "[redacted-url]"
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func contextlessLogContext() context.Context {
	return context.Background()
}
