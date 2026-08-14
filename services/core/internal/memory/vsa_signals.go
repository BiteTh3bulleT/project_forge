package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/sqlutil"
)

func (s *Service) ComputeVSAQuerySignals(ctx context.Context, req VSAQuerySignalsRequest) (map[int64]RetrievalResultVSASignal, error) {
	out := map[int64]RetrievalResultVSASignal{}
	if len(req.Candidates) == 0 {
		return out, nil
	}
	scopeWorkspace := strings.TrimSpace(req.WorkspaceID)
	scopeLane := strings.TrimSpace(req.LaneID)
	if scopeWorkspace == "" || scopeLane == "" {
		return out, nil
	}
	manifestDims, manifestSeed, manifestSourceCount, active, err := s.activeVSAProjectionAlgorithm(ctx, scopeWorkspace, scopeLane)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return out, nil
		}
		return nil, err
	}
	if !active {
		return out, nil
	}
	cfg := s.runtimeVSASettings(ctx)
	cfg.Dims = manifestDims
	cfg.Seed = manifestSeed
	engine := NewVSAEngine(cfg.Dims, cfg.Seed)
	queryVec := engine.EncodeText(req.Query)
	queryTokens := tokenSet(req.Query)

	paths := make([]string, 0, len(req.Candidates)*2)
	seen := map[string]struct{}{}
	for _, c := range req.Candidates {
		for _, p := range []string{strings.TrimSpace(c.AbsPath), strings.TrimSpace(c.RelPath)} {
			if p == "" {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			paths = append(paths, p)
		}
	}

	evidenceByPath, pointerByEvidence, usefulnessByEvidence, evidenceIDs, err := s.vsaMemoryEvidencePointersByPath(ctx, scopeWorkspace, scopeLane, paths)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return out, nil
		}
		return nil, err
	}
	if len(pointerByEvidence) != manifestSourceCount {
		// A supersession, Court change, or storage divergence invalidates the
		// active source set until a governed rebuild installs a matching head.
		return out, nil
	}

	evidenceRowIDs := make([]int64, 0, len(pointerByEvidence))
	for id := range pointerByEvidence {
		evidenceRowIDs = append(evidenceRowIDs, id)
	}
	bindingsByEvidence, _ := s.vsaBindingsByMemoryEvidence(ctx, scopeWorkspace, scopeLane, evidenceRowIDs)
	assocByEvidence, _ := s.vsaAssociationsByMemoryEvidence(ctx, scopeWorkspace, scopeLane, evidenceRowIDs)
	candidateEvidence := map[int64]struct{}{}
	evidenceByChunk := map[int64]int64{}

	for _, c := range req.Candidates {
		evidenceRowID := int64(0)
		if id, ok := evidenceByPath[strings.TrimSpace(c.AbsPath)]; ok {
			evidenceRowID = id
		} else if id, ok := evidenceByPath[strings.TrimSpace(c.RelPath)]; ok {
			evidenceRowID = id
		}
		evidenceByChunk[c.ChunkID] = evidenceRowID
		if evidenceRowID > 0 {
			candidateEvidence[evidenceRowID] = struct{}{}
		}
	}

	for _, c := range req.Candidates {
		evidenceRowID := evidenceByChunk[c.ChunkID]
		if evidenceRowID <= 0 {
			out[c.ChunkID] = RetrievalResultVSASignal{ChunkID: c.ChunkID, Mode: cfg.Mode}
			continue
		}
		pointer := pointerByEvidence[evidenceRowID]
		associative := clamp(engine.Similarity(queryVec, pointer), -1, 1)
		roleMatch := scoreRoleMatch(queryTokens, bindingsByEvidence[evidenceRowID])
		relational := scoreRelational(evidenceRowID, assocByEvidence[evidenceRowID], candidateEvidence)
		feedback := clamp(usefulnessByEvidence[evidenceRowID], -1, 1)
		additive := (cfg.WeightAssociative * associative) + (cfg.WeightRoleMatch * roleMatch) + (cfg.WeightRelational * relational) + (cfg.WeightFeedback * feedback)
		applied := 0.0
		if cfg.Mode == "active" {
			applied = clamp(additive, -cfg.MaxAdditive, cfg.MaxAdditive)
		}
		explain := map[string]any{
			"memoryEvidenceRowId": evidenceRowID,
			"memoryEvidenceId":    evidenceIDs[evidenceRowID],
			"associativeScore":    round(associative),
			"roleMatchScore":      round(roleMatch),
			"relationalScore":     round(relational),
			"feedbackScore":       round(feedback),
			"weights": map[string]any{
				"associative": round(cfg.WeightAssociative),
				"roleMatch":   round(cfg.WeightRoleMatch),
				"relational":  round(cfg.WeightRelational),
				"feedback":    round(cfg.WeightFeedback),
			},
			"maxAdditive": round(cfg.MaxAdditive),
			"mode":        cfg.Mode,
			"applied":     round(applied),
		}
		explainJSON, _ := json.Marshal(explain)
		e := evidenceRowID
		out[c.ChunkID] = RetrievalResultVSASignal{
			ChunkID:             c.ChunkID,
			MemoryEvidenceRowID: &e,
			MemoryEvidenceID:    evidenceIDs[evidenceRowID],
			Mode:                cfg.Mode,
			AssociativeScore:    round(associative),
			RoleMatchScore:      round(roleMatch),
			RelationalScore:     round(relational),
			FeedbackScore:       round(feedback),
			AdditiveScore:       round(additive),
			AppliedScore:        round(applied),
			Explain:             explainJSON,
		}
	}

	return out, nil
}

func (s *Service) activeVSAProjectionAlgorithm(ctx context.Context, workspaceID, laneID string) (int, uint64, int, bool, error) {
	var dimensions int
	var seed int64
	var sourceCount int
	err := s.db.QueryRowContext(ctx, `
SELECT m.dimensions, m.seed, m.source_count
FROM memory_vsa_projection_heads h
JOIN memory_vsa_projection_manifests m ON m.manifest_hash=h.manifest_hash
WHERE h.workspace_id=? AND h.lane_id=? AND h.workspace_id<>'' AND h.lane_id<>''
  AND m.workspace_id=h.workspace_id AND m.lane_id=h.lane_id
  AND m.source_kind='forge_k_memory_evidence'
  AND json_extract(m.manifest_json,'$.version')='forge.vsa.memory_evidence_projection_manifest.v2'`, workspaceID, laneID).Scan(&dimensions, &seed, &sourceCount)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, 0, false, err
	}
	if dimensions <= 0 || seed <= 0 || sourceCount <= 0 {
		return 0, 0, 0, false, nil
	}
	return dimensions, uint64(seed), sourceCount, true, nil
}

func (s *Service) vsaMemoryEvidencePointersByPath(ctx context.Context, workspaceID, laneID string, paths []string) (map[string]int64, map[int64][]float64, map[int64]float64, map[int64]string, error) {
	pathToEvidence := map[string]int64{}
	pointerByEvidence := map[int64][]float64{}
	usefulnessByEvidence := map[int64]float64{}
	evidenceIDs := map[int64]string{}
	if len(paths) == 0 {
		return pathToEvidence, pointerByEvidence, usefulnessByEvidence, evidenceIDs, nil
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT e.id,e.evidence_id,e.raw_ref,e.source_refs_json,e.selected_paths_json,
       vp.support_count,vp.noise_count,vp.pointer_json
FROM forge_k_memory_evidence e
JOIN forge_k_memory_vsa_pointers vp
  ON vp.memory_evidence_row_id=e.id AND vp.memory_evidence_id=e.evidence_id
JOIN memory_vsa_projection_heads h
  ON h.workspace_id=e.workspace_id AND h.lane_id=e.lane_id AND h.manifest_hash=vp.manifest_hash
JOIN memory_vsa_projection_manifests m
  ON m.manifest_hash=h.manifest_hash AND m.source_kind='forge_k_memory_evidence'
JOIN court_exhibits x
  ON x.id=e.court_exhibit_id AND x.current_ruling_id=e.court_ruling_id AND x.status='admitted'
 AND x.case_id=e.court_case_id AND x.content_hash=e.source_object_hash
 AND x.workspace_id=e.workspace_id AND x.lane_id=e.lane_id AND x.committed_by='forge_k.kernel'
JOIN court_rulings r
  ON r.id=e.court_ruling_id AND r.exhibit_id=e.court_exhibit_id AND r.decision='admitted'
 AND r.case_id=e.court_case_id AND r.content_hash=e.source_object_hash
 AND r.workspace_id=e.workspace_id AND r.lane_id=e.lane_id AND r.syscall_id=e.admission_syscall_id
 AND r.committed_by='forge_k.kernel'
LEFT JOIN forge_k_memory_evidence_supersessions superseded
  ON superseded.superseded_evidence_id=e.evidence_id
WHERE e.workspace_id=? AND e.lane_id=? AND e.workspace_id<>'' AND e.lane_id<>''
  AND e.source_object_id=e.court_exhibit_id AND e.source_object_version=e.court_ruling_id
  AND e.content_hash=e.source_object_hash AND e.committed_by='forge_k.kernel' AND superseded.id IS NULL
ORDER BY e.evidence_id ASC,e.id ASC`, workspaceID, laneID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer rows.Close()
	wanted := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		wanted[strings.TrimSpace(path)] = struct{}{}
	}
	for rows.Next() {
		var evidenceRowID int64
		var evidenceID, rawRef, sourceRefsJSON, selectedPathsJSON string
		var usefulCount, noiseCount int
		var pointerRaw string
		if err := rows.Scan(&evidenceRowID, &evidenceID, &rawRef, &sourceRefsJSON, &selectedPathsJSON, &usefulCount, &noiseCount, &pointerRaw); err != nil {
			return nil, nil, nil, nil, err
		}
		candidatePaths := []string{rawRef}
		candidatePaths = append(candidatePaths, decodeVSASourcePaths(sourceRefsJSON)...)
		candidatePaths = append(candidatePaths, decodeVSASourcePaths(selectedPathsJSON)...)
		for _, path := range candidatePaths {
			path = strings.TrimSpace(path)
			if _, ok := wanted[path]; !ok {
				continue
			}
			if _, exists := pathToEvidence[path]; !exists {
				pathToEvidence[path] = evidenceRowID
			}
		}
		if _, ok := pointerByEvidence[evidenceRowID]; !ok {
			vec := []float64{}
			_ = json.Unmarshal([]byte(strings.TrimSpace(pointerRaw)), &vec)
			pointerByEvidence[evidenceRowID] = vec
			usefulnessByEvidence[evidenceRowID] = normalizeUsefulness(float64(usefulCount-noiseCount), usefulCount, noiseCount)
			evidenceIDs[evidenceRowID] = evidenceID
		}
	}
	return pathToEvidence, pointerByEvidence, usefulnessByEvidence, evidenceIDs, rows.Err()
}

func decodeVSASourcePaths(raw string) []string {
	var paths []string
	if json.Unmarshal([]byte(strings.TrimSpace(raw)), &paths) != nil {
		return []string{}
	}
	return paths
}

type governedVSABinding struct {
	Role, Filler             string
	Weight                   float64
	SupportCount, NoiseCount int
}

type governedVSAAssociation struct {
	FromMemoryEvidenceRowID int64
	ToMemoryEvidenceRowID   int64
	Strength                float64
	SupportCount            int
	NoiseCount              int
}

func (s *Service) vsaBindingsByMemoryEvidence(ctx context.Context, workspaceID, laneID string, evidenceRowIDs []int64) (map[int64][]governedVSABinding, error) {
	out := map[int64][]governedVSABinding{}
	if len(evidenceRowIDs) == 0 {
		return out, nil
	}
	args := []any{workspaceID, laneID}
	for i := range evidenceRowIDs {
		args = append(args, evidenceRowIDs[i])
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT b.memory_evidence_row_id,b.role,b.filler,b.weight,b.support_count,b.noise_count
FROM forge_k_memory_vsa_role_bindings b
JOIN memory_vsa_projection_heads h
  ON h.workspace_id=b.workspace_id AND h.lane_id=b.lane_id AND h.manifest_hash=b.manifest_hash
WHERE b.workspace_id=? AND b.lane_id=? AND b.workspace_id<>'' AND b.lane_id<>''
  AND b.memory_evidence_row_id IN (`+sqlutil.Placeholders(len(evidenceRowIDs))+`)
ORDER BY b.memory_evidence_row_id ASC,b.role ASC,b.filler ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var evidenceRowID int64
		var item governedVSABinding
		if err := rows.Scan(&evidenceRowID, &item.Role, &item.Filler, &item.Weight, &item.SupportCount, &item.NoiseCount); err != nil {
			return nil, err
		}
		out[evidenceRowID] = append(out[evidenceRowID], item)
	}
	return out, rows.Err()
}

func (s *Service) vsaAssociationsByMemoryEvidence(ctx context.Context, workspaceID, laneID string, evidenceRowIDs []int64) (map[int64][]governedVSAAssociation, error) {
	out := map[int64][]governedVSAAssociation{}
	if len(evidenceRowIDs) == 0 {
		return out, nil
	}
	args := []any{workspaceID, laneID}
	for i := range evidenceRowIDs {
		args = append(args, evidenceRowIDs[i])
	}
	for i := range evidenceRowIDs {
		args = append(args, evidenceRowIDs[i])
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT a.from_memory_evidence_row_id,a.to_memory_evidence_row_id,a.strength,a.support_count,a.noise_count
FROM forge_k_memory_vsa_associations a
JOIN memory_vsa_projection_heads h
  ON h.workspace_id=a.workspace_id AND h.lane_id=a.lane_id AND h.manifest_hash=a.manifest_hash
WHERE a.workspace_id=? AND a.lane_id=? AND a.workspace_id<>'' AND a.lane_id<>''
  AND (a.from_memory_evidence_row_id IN (`+sqlutil.Placeholders(len(evidenceRowIDs))+`) OR a.to_memory_evidence_row_id IN (`+sqlutil.Placeholders(len(evidenceRowIDs))+`))
ORDER BY a.strength DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item governedVSAAssociation
		if err := rows.Scan(&item.FromMemoryEvidenceRowID, &item.ToMemoryEvidenceRowID, &item.Strength, &item.SupportCount, &item.NoiseCount); err != nil {
			return nil, err
		}
		out[item.FromMemoryEvidenceRowID] = append(out[item.FromMemoryEvidenceRowID], item)
		out[item.ToMemoryEvidenceRowID] = append(out[item.ToMemoryEvidenceRowID], item)
	}
	return out, rows.Err()
}

func scoreRoleMatch(tokens map[string]struct{}, bindings []governedVSABinding) float64 {
	if len(tokens) == 0 || len(bindings) == 0 {
		return 0
	}
	totalWeight := 0.0
	matched := 0.0
	for _, b := range bindings {
		w := clamp(math.Abs(b.Weight), 0, 2)
		totalWeight += w
		reliability := float64(b.SupportCount+1) / float64(b.SupportCount+b.NoiseCount+2)
		if _, ok := tokens[strings.ToLower(strings.TrimSpace(b.Filler))]; ok {
			matched += w * reliability
			continue
		}
		if _, ok := tokens[strings.ToLower(strings.TrimSpace(b.Role))]; ok {
			matched += 0.5 * w * reliability
		}
	}
	if totalWeight == 0 {
		return 0
	}
	return clamp(matched/totalWeight, 0, 1)
}

func scoreRelational(memoryEvidenceRowID int64, associations []governedVSAAssociation, candidateEvidence map[int64]struct{}) float64 {
	if len(associations) == 0 {
		return 0
	}
	sum := 0.0
	count := 0.0
	for _, assoc := range associations {
		other := assoc.ToMemoryEvidenceRowID
		if other == memoryEvidenceRowID {
			other = assoc.FromMemoryEvidenceRowID
		}
		if _, ok := candidateEvidence[other]; !ok {
			continue
		}
		reliability := float64(assoc.SupportCount+1) / float64(assoc.SupportCount+assoc.NoiseCount+2)
		sum += clamp(assoc.Strength, -1, 1) * reliability
		count++
	}
	if count == 0 {
		return 0
	}
	return clamp(sum/count, -1, 1)
}

func normalizeUsefulness(raw float64, usefulCount, noiseCount int) float64 {
	denom := math.Max(1, float64(usefulCount+noiseCount+1))
	countAdjust := float64(usefulCount-noiseCount) / denom
	combined := (0.7 * raw) + (0.3 * countAdjust)
	if combined == 0 {
		return 0
	}
	return clamp(math.Tanh(combined/3), -1, 1)
}

func tokenSet(query string) map[string]struct{} {
	toks := tokenize(query)
	out := make(map[string]struct{}, len(toks))
	for _, t := range toks {
		out[t] = struct{}{}
	}
	return out
}

func (s *Service) SaveRetrievalResultVSASignal(ctx context.Context, signal RetrievalResultVSASignal) error {
	if signal.RetrievalResultID <= 0 || signal.RetrievalRunID <= 0 {
		return nil
	}
	explain := strings.TrimSpace(string(signal.Explain))
	if explain == "" {
		explain = "{}"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO retrieval_result_vsa_signals(
  retrieval_result_id, retrieval_run_id, observation_id, memory_evidence_row_id, memory_evidence_id, mode,
  associative_score, role_match_score, relational_score, feedback_score,
  additive_score, applied_score, explain_json, created_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(retrieval_result_id) DO UPDATE SET
  retrieval_run_id=excluded.retrieval_run_id,
  observation_id=excluded.observation_id,
  memory_evidence_row_id=excluded.memory_evidence_row_id,
  memory_evidence_id=excluded.memory_evidence_id,
  mode=excluded.mode,
  associative_score=excluded.associative_score,
  role_match_score=excluded.role_match_score,
  relational_score=excluded.relational_score,
  feedback_score=excluded.feedback_score,
  additive_score=excluded.additive_score,
  applied_score=excluded.applied_score,
  explain_json=excluded.explain_json,
  created_at=excluded.created_at`,
		signal.RetrievalResultID,
		signal.RetrievalRunID,
		nullInt64(signal.ObservationID),
		nullInt64(signal.MemoryEvidenceRowID),
		strings.TrimSpace(signal.MemoryEvidenceID),
		nonEmptyStr(signal.Mode, "off"),
		signal.AssociativeScore,
		signal.RoleMatchScore,
		signal.RelationalScore,
		signal.FeedbackScore,
		signal.AdditiveScore,
		signal.AppliedScore,
		explain,
		time.Now().UnixMilli(),
	)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "no such table") {
		return nil
	}
	return err
}

func (s *Service) RetrievalRunVSASignals(ctx context.Context, runID int64) ([]RetrievalResultVSASignal, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, retrieval_result_id, retrieval_run_id, observation_id, memory_evidence_row_id, memory_evidence_id, mode,
       associative_score, role_match_score, relational_score, feedback_score,
       additive_score, applied_score, explain_json, created_at
FROM retrieval_result_vsa_signals
WHERE retrieval_run_id = ?
ORDER BY retrieval_result_id ASC`, runID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return []RetrievalResultVSASignal{}, nil
		}
		return nil, err
	}
	defer rows.Close()
	out := []RetrievalResultVSASignal{}
	for rows.Next() {
		item, scanErr := scanRetrievalResultVSASignal(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) RetrievalResultVSASignal(ctx context.Context, resultID int64) (*RetrievalResultVSASignal, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, retrieval_result_id, retrieval_run_id, observation_id, memory_evidence_row_id, memory_evidence_id, mode,
       associative_score, role_match_score, relational_score, feedback_score,
       additive_score, applied_score, explain_json, created_at
FROM retrieval_result_vsa_signals
WHERE retrieval_result_id = ?`, resultID)
	item, err := scanRetrievalResultVSASignal(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func scanRetrievalResultVSASignal(scanner interface{ Scan(dest ...any) error }) (RetrievalResultVSASignal, error) {
	var item RetrievalResultVSASignal
	var obsID sql.NullInt64
	var memoryEvidenceRowID sql.NullInt64
	var explain string
	if err := scanner.Scan(
		&item.ID,
		&item.RetrievalResultID,
		&item.RetrievalRunID,
		&obsID,
		&memoryEvidenceRowID,
		&item.MemoryEvidenceID,
		&item.Mode,
		&item.AssociativeScore,
		&item.RoleMatchScore,
		&item.RelationalScore,
		&item.FeedbackScore,
		&item.AdditiveScore,
		&item.AppliedScore,
		&explain,
		&item.CreatedAtMs,
	); err != nil {
		return item, err
	}
	item.ObservationID = scanNullableInt64(obsID)
	item.MemoryEvidenceRowID = scanNullableInt64(memoryEvidenceRowID)
	item.Explain = asRawJSONObject(explain)
	return item, nil
}

func toAny(items []string) []any {
	out := make([]any, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}

func rankTopSignals(signals []RetrievalResultVSASignal, limit int) []RetrievalResultVSASignal {
	if limit <= 0 || len(signals) <= limit {
		return signals
	}
	sorted := append([]RetrievalResultVSASignal(nil), signals...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].AppliedScore > sorted[j].AppliedScore })
	return sorted[:limit]
}
