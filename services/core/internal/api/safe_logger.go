package api

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
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
	log.Printf("%s%d %dB in %s", e.buf.String(), status, bytes, elapsed)
}

func (e *safeLogEntry) Panic(_ interface{}, _ []byte) {}
