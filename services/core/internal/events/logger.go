package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	structuredPayloadLogMaxBytes      = 8 << 10
	structuredPayloadTruncatedSuffix  = "...(truncated)"
	structuredLogAttributeCorrelation = "correlation_id"
	structuredLogAttributeRequest     = "request_id"
	structuredLogAttributeTrace       = "trace_id"
)

type Logger struct {
	db        *sql.DB
	structLog *slog.Logger
	nowMillis func() int64
}

func New(db *sql.DB) *Logger {
	return NewWithStructuredOutput(db, os.Stderr)
}

func NewWithStructuredOutput(db *sql.DB, w io.Writer) *Logger {
	if w == nil {
		w = io.Discard
	}
	return &Logger{
		db:        db,
		structLog: slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{})),
		nowMillis: func() int64 {
			return time.Now().UnixMilli()
		},
	}
}

func (l *Logger) Emit(ctx context.Context, typ string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		b = []byte("{}")
	}
	_, err = l.db.ExecContext(ctx,
		`INSERT INTO events (created_at, type, payload_json) VALUES (?, ?, ?)`,
		l.nowMillis(), typ, string(b),
	)
	if err != nil {
		return fmt.Errorf("emit event %s: %w", typ, err)
	}
	l.emitStructured(ctx, typ, b, payload)
	return nil
}

func (l *Logger) emitStructured(ctx context.Context, typ string, payloadJSON []byte, payload any) {
	if l == nil || l.structLog == nil {
		return
	}
	attrs := []any{
		"event_type", strings.TrimSpace(typ),
		"payload_json", boundedPayloadJSON(payloadJSON),
	}
	fields := payloadFields(payload)
	if v := firstField(fields, "correlation_id", "correlationId"); v != "" {
		attrs = append(attrs, structuredLogAttributeCorrelation, v)
	}
	if v := firstField(fields, "request_id", "requestId", "requestID"); v != "" {
		attrs = append(attrs, structuredLogAttributeRequest, v)
	}
	if v := firstField(fields, "trace_id", "traceId", "traceID"); v != "" {
		attrs = append(attrs, structuredLogAttributeTrace, v)
	}
	l.structLog.InfoContext(ctx, "forge.event", attrs...)
}

func boundedPayloadJSON(payloadJSON []byte) string {
	out := string(payloadJSON)
	if len(out) <= structuredPayloadLogMaxBytes {
		return out
	}
	return out[:structuredPayloadLogMaxBytes] + structuredPayloadTruncatedSuffix
}

func payloadFields(payload any) map[string]any {
	switch typed := payload.(type) {
	case map[string]any:
		return typed
	case map[string]string:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			out[k] = v
		}
		return out
	default:
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil
		}
		out := map[string]any{}
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil
		}
		return out
	}
}

func firstField(fields map[string]any, keys ...string) string {
	for _, key := range keys {
		if fields == nil {
			return ""
		}
		if value, ok := fields[key]; ok {
			if s := strings.TrimSpace(fmt.Sprintf("%v", value)); s != "" {
				return s
			}
		}
	}
	return ""
}

type Row struct {
	ID          int64           `json:"id"`
	CreatedAtMs int64           `json:"createdAtMs"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
}

func (l *Logger) Recent(ctx context.Context, limit int) ([]Row, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := l.db.QueryContext(ctx,
		`SELECT id, created_at, type, payload_json FROM events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Row, 0, limit)
	for rows.Next() {
		var r Row
		var payload string
		if err := rows.Scan(&r.ID, &r.CreatedAtMs, &r.Type, &payload); err != nil {
			return nil, err
		}
		r.Payload = json.RawMessage(payload)
		out = append(out, r)
	}
	return out, rows.Err()
}
