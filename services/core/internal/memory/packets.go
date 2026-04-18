package memory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (s *Service) AddAlignmentNote(ctx context.Context, req AddAlignmentNoteRequest) (*PacketAlignmentNote, error) {
	if req.PacketID <= 0 {
		return nil, fmt.Errorf("packet id is required")
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		return nil, fmt.Errorf("alignment note is required")
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO packet_alignment_notes(packet_id, observation_id, retrieval_result_id, note, created_at)
VALUES(?,?,?,?,?)`,
		req.PacketID,
		nullInt64(req.ObservationID),
		nullInt64(req.RetrievalResultID),
		note,
		time.Now().UnixMilli(),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.alignmentNoteByID(ctx, id)
}

func (s *Service) alignmentNoteByID(ctx context.Context, id int64) (*PacketAlignmentNote, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, packet_id, observation_id, retrieval_result_id, note, created_at
FROM packet_alignment_notes
WHERE id = ?`, id)
	var n PacketAlignmentNote
	var observationID sql.NullInt64
	var resultID sql.NullInt64
	if err := row.Scan(&n.ID, &n.PacketID, &observationID, &resultID, &n.Note, &n.CreatedAtMs); err != nil {
		return nil, err
	}
	n.ObservationID = scanNullableInt64(observationID)
	n.RetrievalResultID = scanNullableInt64(resultID)
	return &n, nil
}

func (s *Service) PacketAlignmentNotes(ctx context.Context, packetID int64, limit int) ([]PacketAlignmentNote, error) {
	if packetID <= 0 {
		return []PacketAlignmentNote{}, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 80
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, packet_id, observation_id, retrieval_result_id, note, created_at
FROM packet_alignment_notes
WHERE packet_id = ?
ORDER BY id DESC
LIMIT ?`, packetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PacketAlignmentNote{}
	for rows.Next() {
		var item PacketAlignmentNote
		var observationID sql.NullInt64
		var resultID sql.NullInt64
		if err := rows.Scan(&item.ID, &item.PacketID, &observationID, &resultID, &item.Note, &item.CreatedAtMs); err != nil {
			return nil, err
		}
		item.ObservationID = scanNullableInt64(observationID)
		item.RetrievalResultID = scanNullableInt64(resultID)
		out = append(out, item)
	}
	return out, rows.Err()
}
