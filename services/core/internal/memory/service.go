package memory

import (
	"context"
	"database/sql"
	"strings"
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

// RecordObservation is retained only as a source-compatibility fail-closed
// seam. Production evidence creation is owned by the governed FORGE-K
// materialization syscall.
func (s *Service) RecordObservation(ctx context.Context, req RecordObservationRequest) (*Observation, error) {
	return nil, ErrMemoryEvidenceAuthorityRequired
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

// UpdateObservation is a retired compatibility seam. Evidence revisions must
// preserve the original row through the governed FORGE-K revision syscall.
func (s *Service) UpdateObservation(ctx context.Context, id int64, req UpdateObservationRequest) (*ObservationDetail, error) {
	return nil, ErrMemoryEvidenceAuthorityRequired
}

// AddLink is a retired compatibility seam. Callers cannot directly mutate the
// legacy observation graph.
func (s *Service) AddLink(ctx context.Context, fromObs, toObs int64, relationType, note string) error {
	return ErrMemoryEvidenceAuthorityRequired
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
