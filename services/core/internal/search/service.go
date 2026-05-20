package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"unicode"

	"forge/projectforge/services/core/internal/sqlutil"
)

type Hit struct {
	ChunkID    int64   `json:"chunkId"`
	FileID     int64   `json:"fileId"`
	SourceID   int64   `json:"sourceId"`
	Score      float64 `json:"score"`
	Snippet    string  `json:"snippet"`
	AbsPath    string  `json:"absPath"`
	RelPath    string  `json:"relPath"`
	MtimeNs    int64   `json:"mtimeNs"`
	ChunkIdx   int     `json:"chunkIndex"`
	Content    string  `json:"content"`
	ContentLen int     `json:"contentLength"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func ftsMatchQuery(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var tokens []string
	var cur strings.Builder
	flush := func() {
		t := strings.TrimSpace(cur.String())
		cur.Reset()
		if t == "" {
			return
		}
		t = strings.Map(func(r rune) rune {
			if strings.ContainsRune(`"*()^:`, r) {
				return ' '
			}
			return r
		}, t)
		t = strings.TrimSpace(t)
		if t == "" {
			return
		}
		tokens = append(tokens, fmt.Sprintf("%q", t))
	}
	for _, r := range raw {
		if unicode.IsSpace(r) {
			flush()
			continue
		}
		cur.WriteRune(r)
	}
	flush()
	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, " AND ")
}

func (s *Service) Search(ctx context.Context, q string, limit int) ([]Hit, error) {
	return s.SearchScoped(ctx, q, limit, nil)
}

func (s *Service) SearchScoped(ctx context.Context, q string, limit int, sourceIDs []int64) ([]Hit, error) {
	match := ftsMatchQuery(q)
	if match == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	sqlText := `
SELECT
  c.id,
  c.file_id,
  f.source_id,
  c.chunk_index,
  c.content,
  f.abs_path,
  f.rel_path,
  f.mtime_ns,
  bm25(chunks_fts) AS rank,
  snippet(chunks_fts, 0, '«', '»', '…', 64) AS snip
FROM chunks_fts
JOIN chunks c ON c.id = chunks_fts.rowid
JOIN files f ON f.id = c.file_id
WHERE chunks_fts MATCH ?`
	args := []any{match}
	if len(sourceIDs) > 0 {
		sqlText += ` AND f.source_id IN (` + sqlutil.Placeholders(len(sourceIDs)) + `)`
		for _, id := range sourceIDs {
			args = append(args, id)
		}
	}
	sqlText += `
ORDER BY rank
LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Hit
	for rows.Next() {
		var h Hit
		var snip sql.NullString
		if err := rows.Scan(&h.ChunkID, &h.FileID, &h.SourceID, &h.ChunkIdx, &h.Content, &h.AbsPath, &h.RelPath, &h.MtimeNs, &h.Score, &snip); err != nil {
			return nil, err
		}
		if snip.Valid {
			h.Snippet = snip.String
		}
		h.ContentLen = len(h.Content)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Service) ChunkByID(ctx context.Context, id int64) (*Hit, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT c.id, c.file_id, f.source_id, c.chunk_index, c.content, f.abs_path, f.rel_path, f.mtime_ns
FROM chunks c JOIN files f ON f.id = c.file_id WHERE c.id = ?`, id)
	var h Hit
	if err := row.Scan(&h.ChunkID, &h.FileID, &h.SourceID, &h.ChunkIdx, &h.Content, &h.AbsPath, &h.RelPath, &h.MtimeNs); err != nil {
		return nil, err
	}
	h.ContentLen = len(h.Content)
	return &h, nil
}
