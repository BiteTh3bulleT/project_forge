package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Logger struct {
	db *sql.DB
}

func New(db *sql.DB) *Logger {
	return &Logger{db: db}
}

func (l *Logger) Emit(ctx context.Context, typ string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		b = []byte("{}")
	}
	_, err = l.db.ExecContext(ctx,
		`INSERT INTO events (created_at, type, payload_json) VALUES (?, ?, ?)`,
		time.Now().UnixMilli(), typ, string(b),
	)
	if err != nil {
		return fmt.Errorf("emit event %s: %w", typ, err)
	}
	return nil
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
