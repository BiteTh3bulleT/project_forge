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
	cfg := s.runtimeVSASettings(ctx)
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

	obsByPath, pointerByObs, usefulnessByObs, err := s.vsaObservationPointersByPath(ctx, paths)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return out, nil
		}
		return nil, err
	}

	obsIDs := make([]int64, 0, len(pointerByObs))
	for id := range pointerByObs {
		obsIDs = append(obsIDs, id)
	}
	bindingsByObs, _ := s.vsaBindingsByObservation(ctx, obsIDs)
	assocByObs, _ := s.vsaAssociationsByObservation(ctx, obsIDs)
	candidateObs := map[int64]struct{}{}
	obsByChunk := map[int64]int64{}

	for _, c := range req.Candidates {
		obsID := int64(0)
		if id, ok := obsByPath[strings.TrimSpace(c.AbsPath)]; ok {
			obsID = id
		} else if id, ok := obsByPath[strings.TrimSpace(c.RelPath)]; ok {
			obsID = id
		}
		obsByChunk[c.ChunkID] = obsID
		if obsID > 0 {
			candidateObs[obsID] = struct{}{}
		}
	}

	for _, c := range req.Candidates {
		obsID := obsByChunk[c.ChunkID]
		if obsID <= 0 {
			out[c.ChunkID] = RetrievalResultVSASignal{ChunkID: c.ChunkID, Mode: cfg.Mode}
			continue
		}
		pointer := pointerByObs[obsID]
		associative := clamp(engine.Similarity(queryVec, pointer), -1, 1)
		roleMatch := scoreRoleMatch(queryTokens, bindingsByObs[obsID])
		relational := scoreRelational(obsID, assocByObs[obsID], candidateObs)
		feedback := clamp(usefulnessByObs[obsID], -1, 1)
		additive := (cfg.WeightAssociative * associative) + (cfg.WeightRoleMatch * roleMatch) + (cfg.WeightRelational * relational) + (cfg.WeightFeedback * feedback)
		applied := 0.0
		if cfg.Mode == "active" {
			applied = clamp(additive, -cfg.MaxAdditive, cfg.MaxAdditive)
		}
		explain := map[string]any{
			"observationId":    obsID,
			"associativeScore": round(associative),
			"roleMatchScore":   round(roleMatch),
			"relationalScore":  round(relational),
			"feedbackScore":    round(feedback),
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
		o := obsID
		out[c.ChunkID] = RetrievalResultVSASignal{
			ChunkID:          c.ChunkID,
			ObservationID:    &o,
			Mode:             cfg.Mode,
			AssociativeScore: round(associative),
			RoleMatchScore:   round(roleMatch),
			RelationalScore:  round(relational),
			FeedbackScore:    round(feedback),
			AdditiveScore:    round(additive),
			AppliedScore:     round(applied),
			Explain:          explainJSON,
		}
	}

	return out, nil
}

func (s *Service) vsaObservationPointersByPath(ctx context.Context, paths []string) (map[string]int64, map[int64][]float64, map[int64]float64, error) {
	pathToObs := map[string]int64{}
	pointerByObs := map[int64][]float64{}
	usefulnessByObs := map[int64]float64{}
	if len(paths) == 0 {
		return pathToObs, pointerByObs, usefulnessByObs, nil
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT mo.id, mo.source_path, mo.usefulness_score, mo.usefulness_count, mo.noise_count, vp.pointer_json
FROM memory_observations mo
JOIN memory_vsa_pointers vp ON vp.observation_id = mo.id
WHERE mo.source_path IN (`+sqlutil.Placeholders(len(paths))+`)
ORDER BY mo.observed_at DESC, mo.id DESC`, toAny(paths)...)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var obsID int64
		var sourcePath string
		var usefulness float64
		var usefulCount int
		var noiseCount int
		var pointerRaw string
		if err := rows.Scan(&obsID, &sourcePath, &usefulness, &usefulCount, &noiseCount, &pointerRaw); err != nil {
			return nil, nil, nil, err
		}
		if _, ok := pathToObs[sourcePath]; !ok {
			pathToObs[sourcePath] = obsID
		}
		if _, ok := pointerByObs[obsID]; !ok {
			vec := []float64{}
			_ = json.Unmarshal([]byte(strings.TrimSpace(pointerRaw)), &vec)
			pointerByObs[obsID] = vec
			usefulnessByObs[obsID] = normalizeUsefulness(usefulness, usefulCount, noiseCount)
		}
	}
	return pathToObs, pointerByObs, usefulnessByObs, rows.Err()
}

func (s *Service) vsaBindingsByObservation(ctx context.Context, obsIDs []int64) (map[int64][]VSARoleBinding, error) {
	out := map[int64][]VSARoleBinding{}
	if len(obsIDs) == 0 {
		return out, nil
	}
	args := make([]any, len(obsIDs))
	for i := range obsIDs {
		args[i] = obsIDs[i]
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, observation_id, role, filler, weight, support_count, noise_count, binding_json, created_at, updated_at
FROM memory_vsa_role_bindings
WHERE observation_id IN (`+sqlutil.Placeholders(len(obsIDs))+`)
ORDER BY observation_id ASC, role ASC, filler ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item VSARoleBinding
		var binding string
		if err := rows.Scan(&item.ID, &item.ObservationID, &item.Role, &item.Filler, &item.Weight, &item.SupportCount, &item.NoiseCount, &binding, &item.CreatedAtMs, &item.UpdatedAtMs); err != nil {
			return nil, err
		}
		item.Binding = asRawJSONArray(binding)
		out[item.ObservationID] = append(out[item.ObservationID], item)
	}
	return out, rows.Err()
}

func (s *Service) vsaAssociationsByObservation(ctx context.Context, obsIDs []int64) (map[int64][]VSAAssociation, error) {
	out := map[int64][]VSAAssociation{}
	if len(obsIDs) == 0 {
		return out, nil
	}
	args := make([]any, len(obsIDs)*2)
	for i := range obsIDs {
		args[i] = obsIDs[i]
		args[i+len(obsIDs)] = obsIDs[i]
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, from_observation_id, to_observation_id, association_type, strength, support_count, noise_count, evidence_json, created_at, updated_at
FROM memory_vsa_associations
WHERE from_observation_id IN (`+sqlutil.Placeholders(len(obsIDs))+`) OR to_observation_id IN (`+sqlutil.Placeholders(len(obsIDs))+`)
ORDER BY strength DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item VSAAssociation
		var evidence string
		if err := rows.Scan(&item.ID, &item.FromObservationID, &item.ToObservationID, &item.AssociationType, &item.Strength, &item.SupportCount, &item.NoiseCount, &evidence, &item.CreatedAtMs, &item.UpdatedAtMs); err != nil {
			return nil, err
		}
		item.Evidence = asRawJSONObject(evidence)
		out[item.FromObservationID] = append(out[item.FromObservationID], item)
		out[item.ToObservationID] = append(out[item.ToObservationID], item)
	}
	return out, rows.Err()
}

func scoreRoleMatch(tokens map[string]struct{}, bindings []VSARoleBinding) float64 {
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

func scoreRelational(observationID int64, associations []VSAAssociation, candidateObs map[int64]struct{}) float64 {
	if len(associations) == 0 {
		return 0
	}
	sum := 0.0
	count := 0.0
	for _, assoc := range associations {
		other := assoc.ToObservationID
		if other == observationID {
			other = assoc.FromObservationID
		}
		if _, ok := candidateObs[other]; !ok {
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
  retrieval_result_id, retrieval_run_id, observation_id, mode,
  associative_score, role_match_score, relational_score, feedback_score,
  additive_score, applied_score, explain_json, created_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(retrieval_result_id) DO UPDATE SET
  retrieval_run_id=excluded.retrieval_run_id,
  observation_id=excluded.observation_id,
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
SELECT id, retrieval_result_id, retrieval_run_id, observation_id, mode,
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
SELECT id, retrieval_result_id, retrieval_run_id, observation_id, mode,
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
	var explain string
	if err := scanner.Scan(
		&item.ID,
		&item.RetrievalResultID,
		&item.RetrievalRunID,
		&obsID,
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
