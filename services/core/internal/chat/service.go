package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type ThreadSummary struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	CreatedAtMs int64  `json:"createdAtMs"`
	UpdatedAtMs int64  `json:"updatedAtMs"`
	DossierID   *int64 `json:"dossierId,omitempty"`
}

type Message struct {
	ID          int64          `json:"id"`
	ThreadID    int64          `json:"threadId"`
	Role        string         `json:"role"`
	Content     string         `json:"content"`
	CreatedAtMs int64          `json:"createdAtMs"`
	Metadata    map[string]any `json:"metadata"`
}

type ThreadDetail struct {
	ThreadSummary
	Messages []Message `json:"messages"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) CreateThread(ctx context.Context, title string, dossierID *int64) (*ThreadSummary, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Conversation"
	}
	now := time.Now().UnixMilli()
	var did any
	if dossierID != nil {
		did = *dossierID
	} else {
		did = nil
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO chat_threads(title, created_at, updated_at, dossier_id) VALUES(?,?,?,?)`,
		title, now, now, did,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &ThreadSummary{ID: id, Title: title, CreatedAtMs: now, UpdatedAtMs: now, DossierID: dossierID}, nil
}

func (s *Service) UpdateThreadTitle(ctx context.Context, id int64, title string) (*ThreadSummary, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title required")
	}
	now := time.Now().UnixMilli()
	if _, err := s.db.ExecContext(ctx, `UPDATE chat_threads SET title = ?, updated_at = ? WHERE id = ?`, title, now, id); err != nil {
		return nil, err
	}
	return s.getThreadSummary(ctx, id)
}

func (s *Service) getThreadSummary(ctx context.Context, id int64) (*ThreadSummary, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, title, created_at, updated_at, dossier_id FROM chat_threads WHERE id = ?`, id)
	var t ThreadSummary
	var did sql.NullInt64
	if err := row.Scan(&t.ID, &t.Title, &t.CreatedAtMs, &t.UpdatedAtMs, &did); err != nil {
		return nil, err
	}
	if did.Valid {
		v := did.Int64
		t.DossierID = &v
	}
	return &t, nil
}

func (s *Service) ListThreads(ctx context.Context, limit int) ([]ThreadSummary, error) {
	if limit <= 0 || limit > 300 {
		limit = 80
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, created_at, updated_at, dossier_id FROM chat_threads ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ThreadSummary
	for rows.Next() {
		var t ThreadSummary
		var did sql.NullInt64
		if err := rows.Scan(&t.ID, &t.Title, &t.CreatedAtMs, &t.UpdatedAtMs, &did); err != nil {
			return nil, err
		}
		if did.Valid {
			v := did.Int64
			t.DossierID = &v
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Service) GetThread(ctx context.Context, id int64) (*ThreadDetail, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, title, created_at, updated_at, dossier_id FROM chat_threads WHERE id = ?`, id)
	var t ThreadDetail
	var did sql.NullInt64
	if err := row.Scan(&t.ID, &t.Title, &t.CreatedAtMs, &t.UpdatedAtMs, &did); err != nil {
		return nil, err
	}
	if did.Valid {
		v := did.Int64
		t.DossierID = &v
	}
	msgs, err := s.listMessages(ctx, id, 500)
	if err != nil {
		return nil, err
	}
	t.Messages = msgs
	return &t, nil
}

func (s *Service) listMessages(ctx context.Context, threadID int64, limit int) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, thread_id, role, content, created_at, metadata_json FROM chat_messages WHERE thread_id = ? ORDER BY id ASC LIMIT ?`,
		threadID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		var meta string
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.Role, &m.Content, &m.CreatedAtMs, &meta); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(meta), &m.Metadata)
		if m.Metadata == nil {
			m.Metadata = map[string]any{}
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) AppendMessage(ctx context.Context, threadID int64, role, content string, metadata map[string]any) (*Message, error) {
	role = strings.TrimSpace(strings.ToLower(role))
	if role != "user" && role != "assistant" && role != "system" && role != "tool" {
		return nil, fmt.Errorf("invalid role %q", role)
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("content required")
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	raw, _ := json.Marshal(metadata)
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO chat_messages(thread_id, role, content, created_at, metadata_json) VALUES(?,?,?,?,?)`,
		threadID, role, content, now, string(raw),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if _, err := s.db.ExecContext(ctx, `UPDATE chat_threads SET updated_at = ? WHERE id = ?`, now, threadID); err != nil {
		return nil, err
	}
	if role == "user" {
		_ = s.maybeAutoTitleThread(ctx, threadID, content)
	}
	return &Message{ID: id, ThreadID: threadID, Role: role, Content: content, CreatedAtMs: now, Metadata: metadata}, nil
}

func (s *Service) maybeAutoTitleThread(ctx context.Context, threadID int64, content string) error {
	var title string
	var userCount int
	err := s.db.QueryRowContext(ctx, `
SELECT
	(SELECT title FROM chat_threads WHERE id = ?),
	(SELECT COUNT(1) FROM chat_messages WHERE thread_id = ? AND role = 'user')
`, threadID, threadID).Scan(&title, &userCount)
	if err != nil {
		return err
	}
	if userCount != 1 {
		return nil
	}
	normTitle := strings.ToLower(strings.TrimSpace(title))
	if normTitle != "new chat" && normTitle != "conversation" {
		return nil
	}
	next := suggestTitle(content)
	if next == "" {
		return nil
	}
	_, err = s.db.ExecContext(ctx, `UPDATE chat_threads SET title = ? WHERE id = ?`, next, threadID)
	return err
}

func suggestTitle(content string) string {
	v := strings.TrimSpace(content)
	v = strings.ReplaceAll(v, "\n", " ")
	v = strings.Join(strings.Fields(v), " ")
	if v == "" {
		return "Conversation"
	}
	max := 64
	if utf8.RuneCountInString(v) <= max {
		return v
	}
	r := []rune(v)
	v = strings.TrimSpace(string(r[:max]))
	if v == "" {
		return "Conversation"
	}
	return v + "…"
}

// FindAssistantReplyTo returns an assistant message that replies to the given user message id, if stored with replyToUserMessageId metadata.
func (s *Service) FindAssistantReplyTo(ctx context.Context, threadID, userMessageID int64) (*Message, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, thread_id, role, content, created_at, metadata_json FROM chat_messages
WHERE thread_id = ? AND role = 'assistant'
AND json_extract(metadata_json, '$.replyToUserMessageId') IS NOT NULL
AND CAST(json_extract(metadata_json, '$.replyToUserMessageId') AS INTEGER) = ?
ORDER BY id DESC LIMIT 1`, threadID, userMessageID)
	var m Message
	var meta string
	if err := row.Scan(&m.ID, &m.ThreadID, &m.Role, &m.Content, &m.CreatedAtMs, &meta); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	_ = json.Unmarshal([]byte(meta), &m.Metadata)
	if m.Metadata == nil {
		m.Metadata = map[string]any{}
	}
	return &m, nil
}

func (s *Service) BuildTranscript(messages []Message, maxMessages int) string {
	if maxMessages <= 0 {
		maxMessages = 24
	}
	start := 0
	if len(messages) > maxMessages {
		start = len(messages) - maxMessages
	}
	var b strings.Builder
	for _, m := range messages[start:] {
		b.WriteString(strings.ToUpper(m.Role))
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}

func (s *Service) BuildBoundedTranscript(messages []Message, maxMessages, maxMessageRunes, maxTotalRunes int) string {
	if maxMessages <= 0 {
		maxMessages = 24
	}
	if maxMessageRunes <= 0 {
		maxMessageRunes = 1200
	}
	if maxTotalRunes <= 0 {
		maxTotalRunes = maxMessages * maxMessageRunes
	}
	start := 0
	if len(messages) > maxMessages {
		start = len(messages) - maxMessages
	}
	selected := messages[start:]
	lines := make([]string, 0, len(selected))
	remaining := maxTotalRunes
	for i := len(selected) - 1; i >= 0; i-- {
		if remaining <= 0 {
			break
		}
		m := selected[i]
		role := strings.ToUpper(strings.TrimSpace(m.Role))
		if role == "" {
			role = "MESSAGE"
		}
		content := boundedRunes(strings.TrimSpace(m.Content), maxMessageRunes)
		line := role + ": " + content + "\n\n"
		if utf8.RuneCountInString(line) > remaining {
			line = boundedRunes(line, remaining)
		}
		lines = append(lines, line)
		remaining -= utf8.RuneCountInString(line)
	}
	var b strings.Builder
	for i := len(lines) - 1; i >= 0; i-- {
		b.WriteString(lines[i])
	}
	return strings.TrimSpace(b.String())
}

func boundedRunes(value string, max int) string {
	if max <= 0 || utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	if max <= len("... (truncated)") {
		return string(runes[:max])
	}
	return string(runes[:max-len("... (truncated)")]) + "... (truncated)"
}

func (s *Service) DeleteThread(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM chat_threads WHERE id = ?`, id)
	return err
}
