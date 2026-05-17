package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListObservations(ctx context.Context, req ListObservationsRequest) ([]Observation, error) {
	if req.Limit <= 0 || req.Limit > 200 {
		req.Limit = 100
	}
	query := `SELECT id FROM memory_observations WHERE 1=1`
	args := []any{}
	if req.DossierID != nil && *req.DossierID > 0 {
		query += ` AND dossier_id = ?`
		args = append(args, *req.DossierID)
	}
	if strings.TrimSpace(req.Type) != "" {
		query += ` AND type = ?`
		args = append(args, strings.TrimSpace(req.Type))
	}
	if strings.TrimSpace(req.OriginKind) != "" {
		query += ` AND origin_kind = ?`
		args = append(args, strings.TrimSpace(req.OriginKind))
	}
	if req.StaleOnly {
		query += ` AND stale = 1`
	}
	query += ` ORDER BY observed_at DESC, id DESC LIMIT ?`
	args = append(args, req.Limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	out := make([]Observation, 0, len(ids))
	for _, id := range ids {
		obs, err := s.GetObservation(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, obs.Observation)
	}
	return out, nil
}

func (s *Service) RecordObservation(ctx context.Context, req RecordObservationRequest) (*Observation, error) {
	typeName := strings.TrimSpace(req.Type)
	if typeName == "" {
		return nil, fmt.Errorf("observation type is required")
	}
	now := time.Now().UnixMilli()
	observedAt := req.ObservedAtMs
	if observedAt <= 0 {
		observedAt = now
	}
	if req.Confidence <= 0 {
		req.Confidence = 0.5
	}
	if req.Confidence > 1 {
		req.Confidence = 1
	}
	if strings.TrimSpace(req.VerificationState) == "" {
		req.VerificationState = "unknown"
	}
	entitiesJSON, _ := json.Marshal(nonNilStrings(req.Entities))
	tagsJSON, _ := json.Marshal(nonNilStrings(req.Tags))
	relatedFilesJSON, _ := json.Marshal(nonNilStrings(req.RelatedFiles))
	lineageJSON, _ := json.Marshal(nonNilStrings(req.Lineage))

	if strings.TrimSpace(req.OriginKind) != "" && strings.TrimSpace(req.OriginID) != "" {
		var existingID int64
		err := s.db.QueryRowContext(ctx,
			`SELECT id FROM memory_observations WHERE origin_kind = ? AND origin_id = ? LIMIT 1`,
			strings.TrimSpace(req.OriginKind), strings.TrimSpace(req.OriginID),
		).Scan(&existingID)
		if err == nil {
			_, execErr := s.db.ExecContext(ctx, `
UPDATE memory_observations
SET updated_at = ?, observed_at = ?, type = ?, raw_content = ?, summary = ?, embedding_ref = ?,
    dossier_id = ?, project_key = ?, source_path = ?, entities_json = ?, tags_json = ?, related_files_json = ?,
    task_type = ?, confidence = ?, verification_state = ?, lineage_json = ?
WHERE id = ?`,
				now,
				observedAt,
				typeName,
				strings.TrimSpace(req.RawContent),
				strings.TrimSpace(req.Summary),
				strings.TrimSpace(req.EmbeddingRef),
				req.DossierID,
				strings.TrimSpace(req.ProjectKey),
				strings.TrimSpace(req.SourcePath),
				string(entitiesJSON),
				string(tagsJSON),
				string(relatedFilesJSON),
				strings.TrimSpace(req.TaskType),
				req.Confidence,
				strings.TrimSpace(req.VerificationState),
				string(lineageJSON),
				existingID,
			)
			if execErr != nil {
				return nil, execErr
			}
			s.triggerObservationVSAReindex(ctx, existingID, "observation_upsert")
			detail, fetchErr := s.getObservation(ctx, existingID)
			if fetchErr != nil {
				return nil, fetchErr
			}
			return &detail.Observation, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	res, err := s.db.ExecContext(ctx, `
INSERT INTO memory_observations(
  created_at, updated_at, observed_at, type, raw_content, summary, embedding_ref,
  dossier_id, project_key, source_path, entities_json, tags_json, related_files_json,
  task_type, confidence, verification_state, lineage_json, origin_kind, origin_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		now,
		now,
		observedAt,
		typeName,
		strings.TrimSpace(req.RawContent),
		strings.TrimSpace(req.Summary),
		strings.TrimSpace(req.EmbeddingRef),
		req.DossierID,
		strings.TrimSpace(req.ProjectKey),
		strings.TrimSpace(req.SourcePath),
		string(entitiesJSON),
		string(tagsJSON),
		string(relatedFilesJSON),
		strings.TrimSpace(req.TaskType),
		req.Confidence,
		strings.TrimSpace(req.VerificationState),
		string(lineageJSON),
		strings.TrimSpace(req.OriginKind),
		strings.TrimSpace(req.OriginID),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	s.triggerObservationVSAReindex(ctx, id, "observation_insert")
	detail, err := s.getObservation(ctx, id)
	if err != nil {
		return nil, err
	}
	return &detail.Observation, nil
}

func (s *Service) GetObservation(ctx context.Context, id int64) (*ObservationDetail, error) {
	return s.getObservation(ctx, id)
}

func (s *Service) getObservation(ctx context.Context, id int64) (*ObservationDetail, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, updated_at, observed_at, type, raw_content, summary, embedding_ref,
       dossier_id, project_key, source_path, entities_json, tags_json, related_files_json,
       task_type, confidence, verification_state, lineage_json, origin_kind, origin_id,
       stale, last_verified_at, usefulness_score, usefulness_count, noise_count
FROM memory_observations
WHERE id = ?`, id)
	var o Observation
	var dossierID sql.NullInt64
	var stale int
	var lastVerified sql.NullInt64
	var entities, tags, related, lineage string
	if err := row.Scan(
		&o.ID, &o.CreatedAtMs, &o.UpdatedAtMs, &o.ObservedAtMs, &o.Type, &o.RawContent, &o.Summary, &o.EmbeddingRef,
		&dossierID, &o.ProjectKey, &o.SourcePath, &entities, &tags, &related,
		&o.TaskType, &o.Confidence, &o.VerificationState, &lineage, &o.OriginKind, &o.OriginID,
		&stale, &lastVerified, &o.UsefulnessScore, &o.UsefulnessCount, &o.NoiseCount,
	); err != nil {
		return nil, err
	}
	if dossierID.Valid {
		v := dossierID.Int64
		o.DossierID = &v
	}
	if lastVerified.Valid {
		v := lastVerified.Int64
		o.LastVerifiedAtMs = &v
	}
	o.Stale = stale == 1
	o.Entities = asRawJSONArray(entities)
	o.Tags = asRawJSONArray(tags)
	o.RelatedFiles = asRawJSONArray(related)
	o.Lineage = asRawJSONArray(lineage)

	incoming, err := s.linksFor(ctx, id, true)
	if err != nil {
		return nil, err
	}
	outgoing, err := s.linksFor(ctx, id, false)
	if err != nil {
		return nil, err
	}
	signals, err := s.signalsForObservation(ctx, id, 100)
	if err != nil {
		return nil, err
	}
	var vsa *ObservationVSADetail
	if v, vErr := s.GetObservationVSA(ctx, id); vErr == nil {
		if (v.Pointer != nil && v.Pointer.ID > 0) || len(v.RoleBindings) > 0 || len(v.Associations) > 0 {
			vsa = v
		}
	}
	return &ObservationDetail{Observation: o, IncomingLinks: incoming, OutgoingLinks: outgoing, Signals: signals, VSA: vsa}, nil
}

func (s *Service) UpdateObservation(ctx context.Context, id int64, req UpdateObservationRequest) (*ObservationDetail, error) {
	now := time.Now().UnixMilli()
	cur, err := s.getObservation(ctx, id)
	if err != nil {
		return nil, err
	}
	summary := cur.Summary
	if req.Summary != nil {
		summary = strings.TrimSpace(*req.Summary)
	}
	verification := cur.VerificationState
	if req.VerificationState != nil {
		verification = strings.TrimSpace(*req.VerificationState)
	}
	stale := cur.Stale
	if req.Stale != nil {
		stale = *req.Stale
	}
	lastVerified := cur.LastVerifiedAtMs
	if req.LastVerifiedAtMs != nil {
		lastVerified = req.LastVerifiedAtMs
	}
	tags := cur.Tags
	if req.Tags != nil {
		b, _ := json.Marshal(nonNilStrings(req.Tags))
		tags = json.RawMessage(b)
	}
	related := cur.RelatedFiles
	if req.RelatedFiles != nil {
		b, _ := json.Marshal(nonNilStrings(req.RelatedFiles))
		related = json.RawMessage(b)
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE memory_observations
SET updated_at = ?, summary = ?, verification_state = ?, stale = ?, last_verified_at = ?, tags_json = ?, related_files_json = ?
WHERE id = ?`,
		now,
		summary,
		verification,
		boolToInt(stale),
		nullInt64(lastVerified),
		string(tags),
		string(related),
		id,
	)
	if err != nil {
		return nil, err
	}
	s.triggerObservationVSAReindex(ctx, id, "observation_update")
	return s.getObservation(ctx, id)
}

func (s *Service) AddLink(ctx context.Context, fromObs, toObs int64, relationType, note string) error {
	if fromObs <= 0 || toObs <= 0 {
		return fmt.Errorf("observation ids are required")
	}
	if strings.TrimSpace(relationType) == "" {
		relationType = "related"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO memory_observation_links(created_at, from_observation_id, to_observation_id, relation_type, note)
VALUES(?,?,?,?,?)`,
		time.Now().UnixMilli(),
		fromObs,
		toObs,
		strings.TrimSpace(relationType),
		strings.TrimSpace(note),
	)
	if err == nil {
		_, _, _, _ = s.reindexObservationVSAWithEvidence(ctx, fromObs, "link_update", nil, true, nil, nil, "")
		if toObs != fromObs {
			_, _, _, _ = s.reindexObservationVSAWithEvidence(ctx, toObs, "link_update", nil, true, nil, nil, "")
		}
	}
	return err
}

func (s *Service) triggerObservationVSAReindex(ctx context.Context, observationID int64, reason string) {
	if s == nil || s.db == nil || observationID <= 0 {
		return
	}
	_ = s.ReindexObservationVSA(ctx, observationID, reason, nil)
}

func (s *Service) linksFor(ctx context.Context, observationID int64, incoming bool) ([]ObservationLink, error) {
	query := `
SELECT id, created_at, from_observation_id, to_observation_id, relation_type, note
FROM memory_observation_links
WHERE `
	if incoming {
		query += `to_observation_id = ?`
	} else {
		query += `from_observation_id = ?`
	}
	query += ` ORDER BY id DESC LIMIT 100`
	rows, err := s.db.QueryContext(ctx, query, observationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ObservationLink{}
	for rows.Next() {
		var l ObservationLink
		if err := rows.Scan(&l.ID, &l.CreatedAtMs, &l.FromObservationID, &l.ToObservationID, &l.RelationType, &l.Note); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Service) signalsForObservation(ctx context.Context, observationID int64, limit int) ([]UsefulnessEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 80
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, created_at, observation_id, retrieval_result_id, retrieval_run_id, packet_id, job_id, signal, weight, note
FROM memory_usefulness_events
WHERE observation_id = ?
ORDER BY id DESC
LIMIT ?`, observationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UsefulnessEvent{}
	for rows.Next() {
		var e UsefulnessEvent
		var resultID, runID, packetID sql.NullInt64
		var jobID sql.NullString
		if err := rows.Scan(
			&e.ID,
			&e.CreatedAtMs,
			&e.ObservationID,
			&resultID,
			&runID,
			&packetID,
			&jobID,
			&e.Signal,
			&e.Weight,
			&e.Note,
		); err != nil {
			return nil, err
		}
		if resultID.Valid {
			v := resultID.Int64
			e.RetrievalResultID = &v
		}
		if runID.Valid {
			v := runID.Int64
			e.RetrievalRunID = &v
		}
		if packetID.Valid {
			v := packetID.Int64
			e.PacketID = &v
		}
		if jobID.Valid {
			v := jobID.String
			e.JobID = &v
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
