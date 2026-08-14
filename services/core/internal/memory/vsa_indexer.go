package memory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
)

var ErrVSAProjectionAuthorityRequired = errors.New("VSA projection writes require governed FORGE-K rebuild")

type vsaRuntimeSettings struct {
	Mode              string
	Dims              int
	Seed              uint64
	WeightAssociative float64
	WeightRoleMatch   float64
	WeightRelational  float64
	WeightFeedback    float64
	MaxAdditive       float64
}

func (s *Service) runtimeVSASettings(ctx context.Context) vsaRuntimeSettings {
	mode := strings.ToLower(strings.TrimSpace(s.setting(ctx, "retrieval_vsa_mode", "off")))
	if mode != "off" && mode != "shadow" && mode != "active" {
		mode = "off"
	}
	dims := parseIntSetting(s.setting(ctx, "retrieval_vsa_dims", strconv.Itoa(defaultVSADims)), defaultVSADims)
	if dims <= 0 {
		dims = defaultVSADims
	}
	seedInt := parseIntSetting(s.setting(ctx, "retrieval_vsa_seed", strconv.Itoa(defaultVSASeed)), defaultVSASeed)
	if seedInt <= 0 {
		seedInt = defaultVSASeed
	}
	return vsaRuntimeSettings{
		Mode:              mode,
		Dims:              dims,
		Seed:              uint64(seedInt),
		WeightAssociative: clamp(s.parseFloatSetting(ctx, "retrieval_vsa_weight_associative", 0.06), 0, 1),
		WeightRoleMatch:   clamp(s.parseFloatSetting(ctx, "retrieval_vsa_weight_role_match", 0.04), 0, 1),
		WeightRelational:  clamp(s.parseFloatSetting(ctx, "retrieval_vsa_weight_relational", 0.03), 0, 1),
		WeightFeedback:    clamp(s.parseFloatSetting(ctx, "retrieval_vsa_weight_feedback", 0.03), 0, 1),
		MaxAdditive:       clamp(s.parseFloatSetting(ctx, "retrieval_vsa_max_additive", 0.12), 0, 1),
	}
}

func (s *Service) parseFloatSetting(ctx context.Context, key string, def float64) float64 {
	v := strings.TrimSpace(s.setting(ctx, key, ""))
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func parseIntSetting(raw string, def int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

func (s *Service) setting(ctx context.Context, key, def string) string {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil || strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

func (s *Service) upsertVSAPointer(ctx context.Context, pointer VSAPointerRecord) error {
	return ErrVSAProjectionAuthorityRequired
}

func (s *Service) updateVSABindings(ctx context.Context, observationID int64, engine *VSAEngine, obs Observation) error {
	return ErrVSAProjectionAuthorityRequired
}

func extractObservationRoleBindings(obs Observation) []VSARoleBindingSeed {
	out := []VSARoleBindingSeed{}
	add := func(role, filler string, weight float64) {
		role = strings.TrimSpace(strings.ToLower(role))
		filler = strings.TrimSpace(strings.ToLower(filler))
		if role == "" || filler == "" {
			return
		}
		out = append(out, VSARoleBindingSeed{Role: role, Filler: filler, Weight: weight})
	}
	add("observation_type", obs.Type, 0.9)
	add("task_type", obs.TaskType, 0.8)
	add("project_key", obs.ProjectKey, 0.7)
	add("source_path", obs.SourcePath, 0.6)
	for _, entity := range parseRawStringSlice(obs.Entities) {
		add("entity", entity, 1)
	}
	for _, tag := range parseRawStringSlice(obs.Tags) {
		add("tag", tag, 0.75)
	}
	for _, path := range parseRawStringSlice(obs.RelatedFiles) {
		add("related_file", path, 0.65)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Role == out[j].Role {
			return out[i].Filler < out[j].Filler
		}
		return out[i].Role < out[j].Role
	})
	return out
}

func (s *Service) refreshVSAAssociationsFromLinks(ctx context.Context, observationID int64) error {
	return ErrVSAProjectionAuthorityRequired
}

func nonEmptyStr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func (s *Service) observationFingerprint(obs Observation) string {
	h := sha256.New()
	_, _ = h.Write([]byte(strings.TrimSpace(obs.Type)))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(strings.TrimSpace(obs.TaskType)))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(strings.TrimSpace(obs.SourcePath)))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(strings.TrimSpace(obs.Summary)))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(strings.TrimSpace(obs.RawContent)))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write(obs.Entities)
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write(obs.Tags)
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write(obs.RelatedFiles)
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write(obs.Lineage)
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Service) reindexObservationVSA(ctx context.Context, observationID int64, reason string, runID *int64, force bool) (string, string, string, error) {
	return s.reindexObservationVSAWithEvidence(ctx, observationID, reason, runID, force, nil, nil, "")
}

func (s *Service) reindexObservationVSAWithEvidence(ctx context.Context, observationID int64, reason string, runID *int64, force bool, beforeEvidence, afterEvidence map[string]any, note string) (string, string, string, error) {
	// Observation-local indexing cannot prove an exact scoped source-set and
	// algorithm manifest. The governed FORGE-K rebuild is the sole writer.
	return "", "", "blocked", ErrVSAProjectionAuthorityRequired
	/*
		reason = nonEmptyStr(strings.TrimSpace(reason), "reindex")
		obsDetail, err := s.GetObservation(ctx, observationID)
		if err != nil {
			return "", "", "failed", err
		}
		obs := obsDetail.Observation
		fingerprint := s.observationFingerprint(obs)
		cfg := s.runtimeVSASettings(ctx)

		var existingFP sql.NullString
		_ = s.db.QueryRowContext(ctx, `SELECT source_fingerprint FROM memory_vsa_pointers WHERE observation_id = ?`, observationID).Scan(&existingFP)
		before := ""
		if existingFP.Valid {
			before = existingFP.String
		}
		beforePayload := map[string]any{"fingerprint": before}
		for k, v := range beforeEvidence {
			beforePayload[k] = v
		}
		afterPayload := map[string]any{"fingerprint": fingerprint}
		for k, v := range afterEvidence {
			afterPayload[k] = v
		}
		if cfg.Mode == "off" {
			if runID != nil {
				beforePayload["reason"] = "vsa_mode_off"
				_ = s.insertVSAReindexItem(ctx, *runID, observationID, "skipped", reason, before, fingerprint, beforePayload, afterPayload, nonEmptyStr(note, "retrieval_vsa_mode=off"))
			}
			return before, fingerprint, "skipped", nil
		}
		if !force && before != "" && before == fingerprint {
			if runID != nil {
				beforePayload["reason"] = "fingerprint_unchanged"
				_ = s.insertVSAReindexItem(ctx, *runID, observationID, "skipped", reason, before, fingerprint, beforePayload, afterPayload, note)
			}
			return before, fingerprint, "skipped", nil
		}

		engine := NewVSAEngine(cfg.Dims, cfg.Seed)
		pointerVec := engine.ComposeObservationPointer(obs)
		pointerJSON, _ := json.Marshal(pointerVec)
		pointer := VSAPointerRecord{
			ObservationID:     observationID,
			Dims:              cfg.Dims,
			Pointer:           pointerJSON,
			Norm:              vectorNorm(pointerVec),
			SourceFingerprint: fingerprint,
			Stale:             false,
			Metadata:          mustRawJSON(map[string]any{"updatedAtMs": time.Now().UnixMilli()}),
		}
		if err := s.upsertVSAPointer(ctx, pointer); err != nil {
			if runID != nil {
				_ = s.insertVSAReindexItem(ctx, *runID, observationID, "failed", reason, before, fingerprint, beforePayload, afterPayload, err.Error())
			}
			return before, fingerprint, "failed", err
		}
		if err := s.updateVSABindings(ctx, observationID, engine, obs); err != nil {
			if runID != nil {
				_ = s.insertVSAReindexItem(ctx, *runID, observationID, "failed", reason, before, fingerprint, beforePayload, afterPayload, err.Error())
			}
			return before, fingerprint, "failed", err
		}
		if err := s.refreshVSAAssociationsFromLinks(ctx, observationID); err != nil {
			if runID != nil {
				_ = s.insertVSAReindexItem(ctx, *runID, observationID, "failed", reason, before, fingerprint, beforePayload, afterPayload, err.Error())
			}
			return before, fingerprint, "failed", err
		}

		if runID != nil {
			_ = s.insertVSAReindexItem(ctx, *runID, observationID, "indexed", reason, before, fingerprint, beforePayload, afterPayload, note)
		}
		return before, fingerprint, "indexed", nil
	*/
}

func (s *Service) insertVSAReindexItem(ctx context.Context, runID, observationID int64, status, reason, beforeFP, afterFP string, before, after map[string]any, note string) error {
	return ErrVSAProjectionAuthorityRequired
}

func (s *Service) ReindexObservationVSA(ctx context.Context, observationID int64, reason string, runID *int64) error {
	_, _, _, err := s.reindexObservationVSAWithEvidence(ctx, observationID, reason, runID, false, nil, nil, "")
	return err
}

func (s *Service) createVSAReindexRun(ctx context.Context, req RunVSAReindexRequest) (int64, error) {
	return 0, ErrVSAProjectionAuthorityRequired
}

func (s *Service) completeVSAReindexRun(ctx context.Context, runID int64, candidates, indexed, skipped, failed int) {
}

func (s *Service) vsaReindexCandidates(ctx context.Context, req RunVSAReindexRequest) ([]int64, error) {
	return s.selectVSAReindexCandidates(ctx, req, false)
}

func (s *Service) selectVSAReindexCandidates(ctx context.Context, req RunVSAReindexRequest, markStale bool) ([]int64, error) {
	limit := req.Limit
	if limit <= 0 || limit > 500 {
		limit = 150
	}
	fetchLimit := limit
	if req.StaleOnly {
		fetchLimit = limit * 4
		if fetchLimit > 2000 {
			fetchLimit = 2000
		}
	}
	query := `
SELECT mo.id, mo.type, mo.task_type, mo.source_path, mo.summary, mo.raw_content,
       mo.entities_json, mo.tags_json, mo.related_files_json, mo.lineage_json, mo.stale, mo.updated_at,
       vp.source_fingerprint, vp.stale, vp.updated_at
FROM memory_observations mo
LEFT JOIN memory_vsa_pointers vp ON vp.observation_id = mo.id
WHERE 1=1`
	args := []any{}
	if req.DossierID != nil && *req.DossierID > 0 {
		query += ` AND mo.dossier_id = ?`
		args = append(args, *req.DossierID)
	}
	query += ` ORDER BY mo.updated_at DESC, mo.id DESC LIMIT ?`
	args = append(args, fetchLimit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	staleMarker := []int64{}
	for rows.Next() {
		var (
			id         int64
			typeName   string
			taskType   string
			sourcePath string
			summary    string
			rawContent string
			entities   string
			tags       string
			related    string
			lineage    string
			obsStale   int
			obsUpdated int64
			vpFP       sql.NullString
			vpStale    sql.NullInt64
			vpUpdated  sql.NullInt64
		)
		if err := rows.Scan(&id, &typeName, &taskType, &sourcePath, &summary, &rawContent, &entities, &tags, &related, &lineage, &obsStale, &obsUpdated, &vpFP, &vpStale, &vpUpdated); err != nil {
			return nil, err
		}
		currentFP := s.observationFingerprint(Observation{
			Type:         typeName,
			TaskType:     taskType,
			SourcePath:   sourcePath,
			Summary:      summary,
			RawContent:   rawContent,
			Entities:     asRawJSONArray(entities),
			Tags:         asRawJSONArray(tags),
			RelatedFiles: asRawJSONArray(related),
			Lineage:      asRawJSONArray(lineage),
		})
		pointerMissing := !vpFP.Valid
		pointerStale := vpStale.Valid && vpStale.Int64 == 1
		fingerprintChanged := vpFP.Valid && strings.TrimSpace(vpFP.String) != "" && strings.TrimSpace(vpFP.String) != currentFP
		sourceNewerThanPointer := pointerMissing || !vpUpdated.Valid || obsUpdated > vpUpdated.Int64
		if req.StaleOnly {
			if obsStale != 1 && !pointerMissing && !pointerStale && !fingerprintChanged && !sourceNewerThanPointer {
				continue
			}
		}
		if (fingerprintChanged || sourceNewerThanPointer) && !pointerStale && !pointerMissing {
			staleMarker = append(staleMarker, id)
		}
		out = append(out, id)
		if len(out) >= limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	_ = markStale
	_ = staleMarker
	return out, nil
}

// PreviewVSAReindex returns the deterministic candidate set without marking
// pointers stale, creating run evidence, or replacing any VSA projection.
func (s *Service) PreviewVSAReindex(ctx context.Context, req RunVSAReindexRequest) (*MaintenancePreview, error) {
	if req.Limit <= 0 || req.Limit > 500 {
		req.Limit = 150
	}
	ids, err := s.selectVSAReindexCandidates(ctx, req, false)
	if err != nil {
		return nil, err
	}
	return &MaintenancePreview{
		Kind:          "memory.vsa_reindex",
		DryRun:        true,
		ProposalOnly:  true,
		DossierID:     req.DossierID,
		CandidateIDs:  ids,
		Candidates:    len(ids),
		WouldWrite:    []string{"memory_vsa_pointers", "memory_vsa_role_bindings", "memory_vsa_associations", "memory_vsa_reindex_runs", "memory_vsa_reindex_items"},
		RequiresOwner: "forge_k.kernel",
		Note:          strings.TrimSpace(req.Note),
	}, nil
}

func (s *Service) RunVSAReindex(ctx context.Context, req RunVSAReindexRequest) (*VSAReindexRunDetail, error) {
	return nil, ErrVSAProjectionAuthorityRequired
}

func (s *Service) ListVSAReindexRuns(ctx context.Context, limit int, dossierID *int64) ([]VSAReindexRun, error) {
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	query := `
SELECT id, created_at, started_at, completed_at, dossier_id, mode, status, candidates, indexed, skipped, failed, triggered_by, note, settings_json
FROM memory_vsa_reindex_runs`
	args := []any{}
	if dossierID != nil && *dossierID > 0 {
		query += ` WHERE dossier_id = ?`
		args = append(args, *dossierID)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VSAReindexRun{}
	for rows.Next() {
		item, scanErr := scanVSAReindexRun(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) GetVSAReindexRun(ctx context.Context, runID int64) (*VSAReindexRunDetail, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, started_at, completed_at, dossier_id, mode, status, candidates, indexed, skipped, failed, triggered_by, note, settings_json
FROM memory_vsa_reindex_runs
WHERE id = ?`, runID)
	run, err := scanVSAReindexRun(row)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, reindex_run_id, observation_id, status, reason, before_fingerprint, after_fingerprint, before_json, after_json, note, created_at
FROM memory_vsa_reindex_items
WHERE reindex_run_id = ?
ORDER BY id DESC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []VSAReindexItem{}
	for rows.Next() {
		var item VSAReindexItem
		var beforeJSON, afterJSON string
		if err := rows.Scan(
			&item.ID,
			&item.ReindexRunID,
			&item.ObservationID,
			&item.Status,
			&item.Reason,
			&item.BeforeFingerprint,
			&item.AfterFingerprint,
			&beforeJSON,
			&afterJSON,
			&item.Note,
			&item.CreatedAtMs,
		); err != nil {
			return nil, err
		}
		item.Before = asRawJSONObject(beforeJSON)
		item.After = asRawJSONObject(afterJSON)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &VSAReindexRunDetail{Run: run, Items: items}, nil
}

// GetObservationVSA exposes legacy observation projection rows for historical
// inspection only. Governed active-head scoring reads forge_k_memory_vsa_* in
// vsa_signals.go and never treats these rows as current authority.
func (s *Service) GetObservationVSA(ctx context.Context, observationID int64) (*ObservationVSADetail, error) {
	detail := &ObservationVSADetail{ObservationID: observationID}
	if s == nil || s.db == nil {
		return detail, nil
	}
	row := s.db.QueryRowContext(ctx, `
SELECT vp.id, vp.observation_id, vp.dims, vp.pointer_json, vp.norm, vp.source_fingerprint,
       vp.support_count, vp.noise_count, vp.stale, vp.metadata_json, vp.created_at, vp.updated_at
FROM memory_vsa_pointers vp
JOIN memory_observations mo ON mo.id=vp.observation_id
WHERE vp.observation_id=? AND vp.workspace_id<>'' AND vp.lane_id<>'' AND vp.manifest_hash<>''
  AND mo.workspace_id=vp.workspace_id AND mo.lane_id=vp.lane_id`, observationID)
	var stale int
	var pointer, metadata string
	pointerRecord := VSAPointerRecord{}
	if err := row.Scan(
		&pointerRecord.ID,
		&pointerRecord.ObservationID,
		&pointerRecord.Dims,
		&pointer,
		&pointerRecord.Norm,
		&pointerRecord.SourceFingerprint,
		&pointerRecord.SupportCount,
		&pointerRecord.NoiseCount,
		&stale,
		&metadata,
		&pointerRecord.CreatedAtMs,
		&pointerRecord.UpdatedAtMs,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return detail, nil
		}
		return nil, err
	}
	pointerRecord.Stale = stale == 1
	pointerRecord.Pointer = asRawJSONArray(pointer)
	pointerRecord.Metadata = asRawJSONObject(metadata)
	detail.Pointer = &pointerRecord

	bindings, err := s.loadVSARoleBindings(ctx, observationID)
	if err != nil {
		return nil, err
	}
	detail.RoleBindings = bindings
	associations, err := s.loadVSAAssociations(ctx, observationID, 120)
	if err != nil {
		return nil, err
	}
	detail.Associations = associations
	return detail, nil
}

func (s *Service) loadVSARoleBindings(ctx context.Context, observationID int64) ([]VSARoleBinding, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT b.id, b.observation_id, b.role, b.filler, b.weight, b.support_count, b.noise_count, b.binding_json, b.created_at, b.updated_at
FROM memory_vsa_role_bindings b
WHERE b.observation_id=? AND b.workspace_id<>'' AND b.lane_id<>'' AND b.manifest_hash<>''
ORDER BY b.role ASC, b.filler ASC`, observationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VSARoleBinding{}
	for rows.Next() {
		var item VSARoleBinding
		var binding string
		if err := rows.Scan(
			&item.ID,
			&item.ObservationID,
			&item.Role,
			&item.Filler,
			&item.Weight,
			&item.SupportCount,
			&item.NoiseCount,
			&binding,
			&item.CreatedAtMs,
			&item.UpdatedAtMs,
		); err != nil {
			return nil, err
		}
		item.Binding = asRawJSONArray(binding)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) loadVSAAssociations(ctx context.Context, observationID int64, limit int) ([]VSAAssociation, error) {
	if limit <= 0 || limit > 300 {
		limit = 120
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT a.id, a.from_observation_id, a.to_observation_id, a.association_type, a.strength, a.support_count, a.noise_count, a.evidence_json, a.created_at, a.updated_at
FROM memory_vsa_associations a
WHERE a.workspace_id<>'' AND a.lane_id<>'' AND a.manifest_hash<>''
  AND (a.from_observation_id=? OR a.to_observation_id=?)
ORDER BY a.strength DESC, a.id DESC
LIMIT ?`, observationID, observationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VSAAssociation{}
	for rows.Next() {
		var item VSAAssociation
		var evidence string
		if err := rows.Scan(
			&item.ID,
			&item.FromObservationID,
			&item.ToObservationID,
			&item.AssociationType,
			&item.Strength,
			&item.SupportCount,
			&item.NoiseCount,
			&evidence,
			&item.CreatedAtMs,
			&item.UpdatedAtMs,
		); err != nil {
			return nil, err
		}
		item.Evidence = asRawJSONObject(evidence)
		out = append(out, item)
	}
	return out, rows.Err()
}

// DossierVSASummary reports legacy observation projection coverage. It is not
// an active governed-head status surface.
func (s *Service) DossierVSASummary(ctx context.Context, dossierID int64) (*DossierVSASummary, error) {
	summary := &DossierVSASummary{DossierID: dossierID}
	if dossierID <= 0 {
		return summary, nil
	}
	_ = s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM memory_vsa_pointers vp
JOIN memory_observations mo ON mo.id = vp.observation_id
WHERE mo.dossier_id=? AND vp.workspace_id<>'' AND vp.lane_id<>'' AND vp.manifest_hash<>''`, dossierID).Scan(&summary.PointerCount)
	_ = s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM memory_vsa_role_bindings vb
JOIN memory_observations mo ON mo.id = vb.observation_id
WHERE mo.dossier_id=? AND vb.workspace_id<>'' AND vb.lane_id<>'' AND vb.manifest_hash<>''`, dossierID).Scan(&summary.BindingCount)
	_ = s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM memory_vsa_associations va
JOIN memory_observations mo ON mo.id = va.from_observation_id
WHERE mo.dossier_id=? AND va.workspace_id<>'' AND va.lane_id<>'' AND va.manifest_hash<>''`, dossierID).Scan(&summary.AssociationCount)

	var runID sql.NullInt64
	var created sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
SELECT id, created_at
FROM memory_vsa_reindex_runs
WHERE dossier_id = ?
ORDER BY id DESC
LIMIT 1`, dossierID).Scan(&runID, &created)
	if err == nil {
		summary.LastReindexRunID = scanNullableInt64(runID)
		summary.LastReindexAtMs = scanNullableInt64(created)
	}
	coverageDenom := summary.PointerCount + summary.BindingCount + summary.AssociationCount
	if coverageDenom <= 0 {
		summary.Health = "unindexed"
		return summary, nil
	}
	score := clamp(float64(summary.PointerCount)/math.Max(1, float64(summary.PointerCount+summary.BindingCount/2)), 0, 1)
	summary.CoverageScore = score
	switch {
	case score >= 0.8:
		summary.Health = "healthy"
	case score >= 0.45:
		summary.Health = "partial"
	default:
		summary.Health = "low"
	}
	return summary, nil
}

func (s *Service) TouchVSAReliabilityFromUsefulness(ctx context.Context, observationID int64, signal string, weight float64) error {
	return ErrVSAProjectionAuthorityRequired
}

func scanVSAReindexRun(scanner interface{ Scan(dest ...any) error }) (VSAReindexRun, error) {
	var run VSAReindexRun
	var completed sql.NullInt64
	var dossier sql.NullInt64
	var settings string
	if err := scanner.Scan(
		&run.ID,
		&run.CreatedAtMs,
		&run.StartedAtMs,
		&completed,
		&dossier,
		&run.Mode,
		&run.Status,
		&run.Candidates,
		&run.Indexed,
		&run.Skipped,
		&run.Failed,
		&run.TriggeredBy,
		&run.Note,
		&settings,
	); err != nil {
		return run, err
	}
	run.CompletedAtMs = scanNullableInt64(completed)
	run.DossierID = scanNullableInt64(dossier)
	run.Settings = asRawJSONObject(settings)
	return run, nil
}

func mustRawJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
