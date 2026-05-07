package canvas

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Board struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	DossierID   *int64 `json:"dossierId,omitempty"`
	CreatedAtMs int64  `json:"createdAtMs"`
	UpdatedAtMs int64  `json:"updatedAtMs"`
}

type Note struct {
	ID          int64            `json:"id"`
	BoardID     int64            `json:"boardId"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	X           float64          `json:"x"`
	Y           float64          `json:"y"`
	Width       float64          `json:"width"`
	Height      float64          `json:"height"`
	Pinned      bool             `json:"pinned"`
	Color       string           `json:"color"`
	Links       []map[string]any `json:"links"`
	CreatedAtMs int64            `json:"createdAtMs"`
	UpdatedAtMs int64            `json:"updatedAtMs"`
}

type BoardDetail struct {
	Board
	Notes []Note `json:"notes"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) CreateBoard(ctx context.Context, title string, dossierID *int64) (*Board, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Board"
	}
	now := time.Now().UnixMilli()
	var did any
	if dossierID != nil {
		did = *dossierID
	} else {
		did = nil
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO canvas_boards(title, dossier_id, created_at, updated_at) VALUES(?,?,?,?)`,
		title, did, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Board{ID: id, Title: title, DossierID: dossierID, CreatedAtMs: now, UpdatedAtMs: now}, nil
}

func (s *Service) ListBoards(ctx context.Context, limit int) ([]Board, error) {
	if limit <= 0 || limit > 200 {
		limit = 60
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, dossier_id, created_at, updated_at FROM canvas_boards ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Board
	for rows.Next() {
		var b Board
		var did sql.NullInt64
		if err := rows.Scan(&b.ID, &b.Title, &did, &b.CreatedAtMs, &b.UpdatedAtMs); err != nil {
			return nil, err
		}
		if did.Valid {
			v := did.Int64
			b.DossierID = &v
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Service) GetBoard(ctx context.Context, id int64) (*BoardDetail, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, title, dossier_id, created_at, updated_at FROM canvas_boards WHERE id = ?`, id)
	var b BoardDetail
	var did sql.NullInt64
	if err := row.Scan(&b.ID, &b.Title, &did, &b.CreatedAtMs, &b.UpdatedAtMs); err != nil {
		return nil, err
	}
	if did.Valid {
		v := did.Int64
		b.DossierID = &v
	}
	notes, err := s.listNotes(ctx, id)
	if err != nil {
		return nil, err
	}
	b.Notes = notes
	return &b, nil
}

func (s *Service) listNotes(ctx context.Context, boardID int64) ([]Note, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, board_id, title, body, x, y, width, height, pinned, color, links_json, created_at, updated_at
FROM canvas_notes WHERE board_id = ? ORDER BY id ASC`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Note{}
	for rows.Next() {
		var n Note
		var links string
		var pin int
		if err := rows.Scan(&n.ID, &n.BoardID, &n.Title, &n.Body, &n.X, &n.Y, &n.Width, &n.Height, &pin, &n.Color, &links, &n.CreatedAtMs, &n.UpdatedAtMs); err != nil {
			return nil, err
		}
		n.Pinned = pin != 0
		_ = json.Unmarshal([]byte(links), &n.Links)
		if n.Links == nil {
			n.Links = []map[string]any{}
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Service) CreateNote(ctx context.Context, boardID int64, title, body string, x, y, w, h float64) (*Note, error) {
	now := time.Now().UnixMilli()
	links, _ := json.Marshal([]map[string]any{})
	res, err := s.db.ExecContext(ctx, `
INSERT INTO canvas_notes(board_id, title, body, x, y, width, height, pinned, color, links_json, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		boardID, strings.TrimSpace(title), body, x, y, w, h, 0, "", string(links), now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if _, err := s.db.ExecContext(ctx, `UPDATE canvas_boards SET updated_at = ? WHERE id = ?`, now, boardID); err != nil {
		return nil, err
	}
	return &Note{ID: id, BoardID: boardID, Title: title, Body: body, X: x, Y: y, Width: w, Height: h, Links: []map[string]any{}, CreatedAtMs: now, UpdatedAtMs: now}, nil
}

type PatchNote struct {
	Title  *string          `json:"title"`
	Body   *string          `json:"body"`
	X      *float64         `json:"x"`
	Y      *float64         `json:"y"`
	Width  *float64         `json:"width"`
	Height *float64         `json:"height"`
	Pinned *bool            `json:"pinned"`
	Color  *string          `json:"color"`
	Links  []map[string]any `json:"links"`
}

func (s *Service) PatchNote(ctx context.Context, boardID, noteID int64, p PatchNote) (*Note, error) {
	cur, err := s.getNote(ctx, boardID, noteID)
	if err != nil {
		return nil, err
	}
	if p.Title != nil {
		cur.Title = strings.TrimSpace(*p.Title)
	}
	if p.Body != nil {
		cur.Body = *p.Body
	}
	if p.X != nil {
		cur.X = *p.X
	}
	if p.Y != nil {
		cur.Y = *p.Y
	}
	if p.Width != nil {
		cur.Width = *p.Width
	}
	if p.Height != nil {
		cur.Height = *p.Height
	}
	if p.Pinned != nil {
		cur.Pinned = *p.Pinned
	}
	if p.Color != nil {
		cur.Color = *p.Color
	}
	if p.Links != nil {
		cur.Links = p.Links
	}
	pin := 0
	if cur.Pinned {
		pin = 1
	}
	links, _ := json.Marshal(cur.Links)
	now := time.Now().UnixMilli()
	_, err = s.db.ExecContext(ctx, `
UPDATE canvas_notes SET title=?, body=?, x=?, y=?, width=?, height=?, pinned=?, color=?, links_json=?, updated_at=?
WHERE id = ? AND board_id = ?`,
		cur.Title, cur.Body, cur.X, cur.Y, cur.Width, cur.Height, pin, cur.Color, string(links), now, noteID, boardID,
	)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE canvas_boards SET updated_at = ? WHERE id = ?`, now, boardID)
	cur.UpdatedAtMs = now
	return cur, nil
}

func (s *Service) getNote(ctx context.Context, boardID, noteID int64) (*Note, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, board_id, title, body, x, y, width, height, pinned, color, links_json, created_at, updated_at
FROM canvas_notes WHERE id = ? AND board_id = ?`, noteID, boardID)
	var n Note
	var links string
	var pin int
	if err := row.Scan(&n.ID, &n.BoardID, &n.Title, &n.Body, &n.X, &n.Y, &n.Width, &n.Height, &pin, &n.Color, &links, &n.CreatedAtMs, &n.UpdatedAtMs); err != nil {
		return nil, err
	}
	n.Pinned = pin != 0
	_ = json.Unmarshal([]byte(links), &n.Links)
	return &n, nil
}

func (s *Service) DeleteNote(ctx context.Context, boardID, noteID int64) error {
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, `DELETE FROM canvas_notes WHERE id = ? AND board_id = ?`, noteID, boardID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("note not found")
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE canvas_boards SET updated_at = ? WHERE id = ?`, now, boardID)
	return nil
}

func (s *Service) DeleteBoard(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM canvas_boards WHERE id = ?`, id)
	return err
}
