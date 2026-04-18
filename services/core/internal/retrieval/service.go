package retrieval

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/embeddings"
	"forge/projectforge/services/core/internal/memory"
	"forge/projectforge/services/core/internal/search"
)

type Mode string

const (
	ModeKeyword  Mode = "keyword"
	ModeSemantic Mode = "semantic"
	ModeHybrid   Mode = "hybrid"
)

type Run struct {
	ID          int64           `json:"id"`
	CreatedAtMs int64           `json:"createdAtMs"`
	Query       string          `json:"query"`
	Mode        Mode            `json:"mode"`
	DossierID   *int64          `json:"dossierId"`
	PacketID    *int64          `json:"packetId"`
	JobID       *string         `json:"jobId"`
	Weighting   json.RawMessage `json:"weighting"`
	Notes       string          `json:"notes"`
	Results     []Result        `json:"results"`
}

type Result struct {
	ID                int64           `json:"id"`
	RetrievalRunID    int64           `json:"retrievalRunId"`
	ChunkID           *int64          `json:"chunkId"`
	FileID            *int64          `json:"fileId"`
	AbsPath           string          `json:"absPath"`
	RelPath           string          `json:"relPath"`
	RankIndex         int             `json:"rankIndex"`
	KeywordScore      float64         `json:"keywordScore"`
	SemanticScore     float64         `json:"semanticScore"`
	HybridScore       float64         `json:"hybridScore"`
	Snippet           string          `json:"snippet"`
	SelectedForPacket bool            `json:"selectedForPacket"`
	UsefulnessLabel   string          `json:"usefulnessLabel"`
	UsefulnessNote    string          `json:"usefulnessNote"`
	SelectionReason   json.RawMessage `json:"selectionReason"`
	ObservationID     *int64          `json:"observationId"`
}

type RunRequest struct {
	Query           string
	Mode            Mode
	Limit           int
	SelectForPacket int
	DossierID       *int64
	SourceIDs       []int64
	WeightKeyword   float64
	WeightSemantic  float64
	Provider        string
	Model           string
	JobID           *string
	PacketID        *int64
	Notes           string
}

type Service struct {
	db         *sql.DB
	search     *search.Service
	embeddings *embeddings.Service
	memory     *memory.Service
}

type aggregateHit struct {
	ChunkID       int64
	FileID        int64
	AbsPath       string
	RelPath       string
	Snippet       string
	KeywordScore  float64
	SemanticScore float64
}

type rankedCandidate struct {
	item               *aggregateHit
	score              float64
	baseScore          float64
	usefulnessBoost    float64
	structuralBoost    float64
	recencyBoost       float64
	selectionDiversity float64
}

func New(db *sql.DB, searchSvc *search.Service, embedSvc *embeddings.Service, memorySvc *memory.Service) *Service {
	return &Service{db: db, search: searchSvc, embeddings: embedSvc, memory: memorySvc}
}

func (s *Service) Run(ctx context.Context, req RunRequest) (*Run, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	mode := req.Mode
	if mode == "" {
		mode = ModeHybrid
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 30
	}
	if req.SelectForPacket <= 0 || req.SelectForPacket > req.Limit {
		req.SelectForPacket = min(req.Limit, 8)
	}

	sourceIDs, err := s.resolveSourceScope(ctx, req.DossierID, req.SourceIDs)
	if err != nil {
		return nil, err
	}

	wKeyword, wSemantic := s.resolveWeights(ctx, req.WeightKeyword, req.WeightSemantic)

	agg := map[int64]*aggregateHit{}

	if mode == ModeKeyword || mode == ModeHybrid {
		kHits, err := s.search.SearchScoped(ctx, query, req.Limit*2, sourceIDs)
		if err != nil {
			return nil, err
		}
		for i, h := range kHits {
			score := 1.0 / float64(i+1)
			cur := agg[h.ChunkID]
			if cur == nil {
				cur = &aggregateHit{ChunkID: h.ChunkID, FileID: h.FileID, AbsPath: h.AbsPath, RelPath: h.RelPath, Snippet: h.Snippet}
				agg[h.ChunkID] = cur
			}
			if cur.Snippet == "" {
				cur.Snippet = h.Snippet
			}
			cur.KeywordScore = score
		}
	}

	if mode == ModeSemantic || mode == ModeHybrid {
		sHits, err := s.embeddings.SemanticSearch(ctx, embeddings.SemanticSearchRequest{
			Query:     query,
			Limit:     req.Limit * 3,
			SourceIDs: sourceIDs,
			DossierID: req.DossierID,
			Provider:  req.Provider,
			Model:     req.Model,
		})
		if err != nil {
			return nil, err
		}
		for _, h := range sHits {
			semantic := normalizeSemantic(h.SemanticScore)
			cur := agg[h.ChunkID]
			if cur == nil {
				cur = &aggregateHit{ChunkID: h.ChunkID, FileID: h.FileID, AbsPath: h.AbsPath, RelPath: h.RelPath, Snippet: h.Snippet}
				agg[h.ChunkID] = cur
			}
			if cur.Snippet == "" {
				cur.Snippet = h.Snippet
			}
			if semantic > cur.SemanticScore {
				cur.SemanticScore = semantic
			}
		}
	}

	if len(agg) == 0 {
		return s.persistRun(ctx, query, mode, req, wKeyword, wSemantic, nil)
	}

	pathUsefulness, _ := s.pathUsefulnessScores(ctx)
	highValueFiles, noisyFiles := s.dossierFileBias(ctx, req.DossierID)
	rankedRows := make([]rankedCandidate, 0, len(agg))
	for _, v := range agg {
		base := 0.0
		switch mode {
		case ModeKeyword:
			base = v.KeywordScore
		case ModeSemantic:
			base = v.SemanticScore
		default:
			base = (wKeyword * v.KeywordScore) + (wSemantic * v.SemanticScore)
		}
		usefulnessBoost := 0.0
		if score, ok := pathUsefulness[v.AbsPath]; ok {
			usefulnessBoost = score
		}
		structuralBoost := 0.0
		if _, ok := highValueFiles[v.RelPath]; ok {
			structuralBoost += 0.08
		}
		if _, ok := highValueFiles[v.AbsPath]; ok {
			structuralBoost += 0.08
		}
		if _, ok := noisyFiles[v.RelPath]; ok {
			structuralBoost -= 0.08
		}
		if _, ok := noisyFiles[v.AbsPath]; ok {
			structuralBoost -= 0.08
		}
		recencyBoost := 0.0
		if v.KeywordScore > 0 && v.SemanticScore > 0 {
			recencyBoost = 0.02
		}
		score := base + usefulnessBoost + structuralBoost + recencyBoost
		rankedRows = append(rankedRows, rankedCandidate{
			item:            v,
			score:           score,
			baseScore:       base,
			usefulnessBoost: usefulnessBoost,
			structuralBoost: structuralBoost,
			recencyBoost:    recencyBoost,
		})
	}
	sort.Slice(rankedRows, func(i, j int) bool { return rankedRows[i].score > rankedRows[j].score })
	if len(rankedRows) > req.Limit {
		rankedRows = rankedRows[:req.Limit]
	}
	selectedIndices := coverageSelectIndices(rankedRows, req.SelectForPacket)

	results := make([]Result, 0, len(rankedRows))
	for idx, row := range rankedRows {
		chunk := row.item.ChunkID
		file := row.item.FileID
		_, selected := selectedIndices[idx]
		reasonPayload, _ := json.Marshal(map[string]any{
			"baseScore":         round(row.baseScore),
			"usefulnessBoost":   round(row.usefulnessBoost),
			"structuralBoost":   round(row.structuralBoost),
			"recencyBoost":      round(row.recencyBoost),
			"selectedForPacket": selected,
			"coverageBucket": func() string {
				if strings.TrimSpace(row.item.RelPath) == "" {
					return "unknown"
				}
				return strings.Split(strings.TrimSpace(row.item.RelPath), "/")[0]
			}(),
		})
		results = append(results, Result{
			ChunkID:           &chunk,
			FileID:            &file,
			AbsPath:           row.item.AbsPath,
			RelPath:           row.item.RelPath,
			RankIndex:         idx,
			KeywordScore:      round(row.item.KeywordScore),
			SemanticScore:     round(row.item.SemanticScore),
			HybridScore:       round(row.score),
			Snippet:           row.item.Snippet,
			SelectedForPacket: selected,
			UsefulnessLabel:   "unknown",
			SelectionReason:   reasonPayload,
		})
	}

	return s.persistRun(ctx, query, mode, req, wKeyword, wSemantic, results)
}

func (s *Service) persistRun(ctx context.Context, query string, mode Mode, req RunRequest, wKeyword, wSemantic float64, results []Result) (*Run, error) {
	now := time.Now().UnixMilli()
	weighting, _ := json.Marshal(map[string]any{
		"keyword":  wKeyword,
		"semantic": wSemantic,
		"mode":     mode,
	})
	res, err := s.db.ExecContext(ctx, `
INSERT INTO retrieval_runs(created_at, query, mode, dossier_id, packet_id, job_id, weighting_json, notes)
VALUES(?,?,?,?,?,?,?,?)`,
		now, query, string(mode), req.DossierID, req.PacketID, req.JobID, string(weighting), req.Notes,
	)
	if err != nil {
		return nil, err
	}
	runID, _ := res.LastInsertId()

	for i := range results {
		row := &results[i]
		resultRes, err := s.db.ExecContext(ctx, `
INSERT INTO retrieval_results(
  retrieval_run_id, chunk_id, file_id, abs_path, rel_path,
  rank_index, keyword_score, semantic_score, hybrid_score, snippet,
  selected_for_packet, usefulness_label, usefulness_note
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			runID,
			row.ChunkID,
			row.FileID,
			row.AbsPath,
			row.RelPath,
			row.RankIndex,
			row.KeywordScore,
			row.SemanticScore,
			row.HybridScore,
			row.Snippet,
			boolToInt(row.SelectedForPacket),
			"unknown",
			"",
		)
		if err != nil {
			return nil, err
		}
		id, _ := resultRes.LastInsertId()
		row.ID = id
		row.RetrievalRunID = runID
		if s.memory != nil {
			reason := map[string]any{}
			if len(row.SelectionReason) > 0 {
				_ = json.Unmarshal(row.SelectionReason, &reason)
			}
			_ = s.memory.SaveSelectionReason(ctx, memory.SaveSelectionReasonRequest{
				RetrievalResultID: id,
				Reason:            reason,
			})
			obs, obsErr := s.memory.RecordObservation(ctx, memory.RecordObservationRequest{
				Type:              "retrieval_result",
				RawContent:        row.Snippet,
				Summary:           summarizeSnippet(row.Snippet),
				DossierID:         req.DossierID,
				SourcePath:        nonEmpty(row.AbsPath, row.RelPath),
				RelatedFiles:      []string{nonEmpty(row.RelPath, row.AbsPath)},
				TaskType:          "retrieval",
				Confidence:        confidenceFromScores(row.KeywordScore, row.SemanticScore, row.HybridScore),
				VerificationState: "unverified",
				OriginKind:        "retrieval_result",
				OriginID:          strconv.FormatInt(id, 10),
			})
			if obsErr == nil && obs != nil {
				row.ObservationID = &obs.ID
				_ = s.memory.LinkResultObservation(ctx, id, obs.ID, "Created from retrieval result persistence")
			}
		}
	}

	if req.PacketID != nil {
		_, _ = s.db.ExecContext(ctx, `
INSERT INTO packet_retrieval_runs(packet_id, retrieval_run_id, created_at)
VALUES(?,?,?) ON CONFLICT(packet_id, retrieval_run_id) DO NOTHING`, *req.PacketID, runID, now)
	}

	return s.GetRun(ctx, runID)
}

func (s *Service) GetRun(ctx context.Context, runID int64) (*Run, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, query, mode, dossier_id, packet_id, job_id, weighting_json, notes
FROM retrieval_runs WHERE id = ?`, runID)
	var r Run
	var dossierID sql.NullInt64
	var packetID sql.NullInt64
	var jobID sql.NullString
	var weighting string
	if err := row.Scan(&r.ID, &r.CreatedAtMs, &r.Query, &r.Mode, &dossierID, &packetID, &jobID, &weighting, &r.Notes); err != nil {
		return nil, err
	}
	if dossierID.Valid {
		v := dossierID.Int64
		r.DossierID = &v
	}
	if packetID.Valid {
		v := packetID.Int64
		r.PacketID = &v
	}
	if jobID.Valid {
		v := jobID.String
		r.JobID = &v
	}
	r.Weighting = json.RawMessage(weighting)

	results, err := s.resultsForRun(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	r.Results = results
	return &r, nil
}

func (s *Service) ListRuns(ctx context.Context, limit int, dossierID *int64) ([]Run, error) {
	if limit <= 0 || limit > 200 {
		limit = 80
	}
	query := `SELECT id FROM retrieval_runs`
	args := []any{}
	if dossierID != nil {
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
	out := []Run{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		run, err := s.GetRun(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, *run)
	}
	return out, rows.Err()
}

func (s *Service) ListRunsByJob(ctx context.Context, jobID string, limit int) ([]Run, error) {
	if strings.TrimSpace(jobID) == "" {
		return []Run{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM retrieval_runs WHERE job_id = ? ORDER BY id DESC LIMIT ?`, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		run, err := s.GetRun(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, *run)
	}
	return out, rows.Err()
}

func (s *Service) AttachRunToPacket(ctx context.Context, runID, packetID int64) error {
	if runID <= 0 || packetID <= 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	if _, err := s.db.ExecContext(ctx, `UPDATE retrieval_runs SET packet_id = ? WHERE id = ?`, packetID, runID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO packet_retrieval_runs(packet_id, retrieval_run_id, created_at)
VALUES(?,?,?)
ON CONFLICT(packet_id, retrieval_run_id) DO NOTHING`, packetID, runID, now)
	return err
}

func (s *Service) resultsForRun(ctx context.Context, runID int64) ([]Result, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, retrieval_run_id, chunk_id, file_id, abs_path, rel_path, rank_index,
       keyword_score, semantic_score, hybrid_score, snippet,
       selected_for_packet, usefulness_label, usefulness_note,
       COALESCE((SELECT reason_json FROM retrieval_result_selection WHERE retrieval_result_id = retrieval_results.id), '{}') AS reason_json,
       (SELECT observation_id FROM retrieval_result_observations WHERE retrieval_result_id = retrieval_results.id ORDER BY created_at DESC LIMIT 1) AS observation_id
FROM retrieval_results
WHERE retrieval_run_id = ?
ORDER BY rank_index ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Result{}
	for rows.Next() {
		var r Result
		var chunkID sql.NullInt64
		var fileID sql.NullInt64
		var observationID sql.NullInt64
		var selected int
		var reasonJSON string
		if err := rows.Scan(
			&r.ID,
			&r.RetrievalRunID,
			&chunkID,
			&fileID,
			&r.AbsPath,
			&r.RelPath,
			&r.RankIndex,
			&r.KeywordScore,
			&r.SemanticScore,
			&r.HybridScore,
			&r.Snippet,
			&selected,
			&r.UsefulnessLabel,
			&r.UsefulnessNote,
			&reasonJSON,
			&observationID,
		); err != nil {
			return nil, err
		}
		if chunkID.Valid {
			v := chunkID.Int64
			r.ChunkID = &v
		}
		if fileID.Valid {
			v := fileID.Int64
			r.FileID = &v
		}
		if observationID.Valid {
			v := observationID.Int64
			r.ObservationID = &v
		}
		r.SelectionReason = json.RawMessage(strings.TrimSpace(reasonJSON))
		if len(r.SelectionReason) == 0 {
			r.SelectionReason = json.RawMessage("{}")
		}
		r.SelectedForPacket = selected == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) MarkUsefulness(ctx context.Context, resultID int64, label, note string, jobID *string, packetID *int64) error {
	label = strings.TrimSpace(strings.ToLower(label))
	if label == "" {
		label = "unknown"
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE retrieval_results SET usefulness_label = ?, usefulness_note = ? WHERE id = ?`,
		label, strings.TrimSpace(note), resultID,
	); err != nil {
		return err
	}

	var runID sql.NullInt64
	var chunkID sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT retrieval_run_id, chunk_id FROM retrieval_results WHERE id = ?`, resultID).Scan(&runID, &chunkID); err != nil {
		return err
	}
	if !runID.Valid {
		return nil
	}
	if s.memory != nil {
		obsID, err := s.memory.ObservationByResult(ctx, resultID)
		if err == nil && obsID != nil {
			runIDValue := runID.Int64
			_ = s.memory.MarkObservationUsefulness(ctx, memory.MarkUsefulnessRequest{
				ObservationID:     *obsID,
				RetrievalResultID: &resultID,
				RetrievalRunID:    &runIDValue,
				PacketID:          packetID,
				JobID:             jobID,
				Signal:            label,
				Weight:            1,
				Note:              note,
			})
		}
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO context_evidence(created_at, retrieval_result_id, retrieval_run_id, job_id, packet_id, chunk_id, evidence_type, weight, note)
VALUES(?,?,?,?,?,?,?,?,?)`,
		time.Now().UnixMilli(),
		resultID,
		runID.Int64,
		jobID,
		packetID,
		nullInt64(chunkID),
		usefulnessToEvidence(label),
		1.0,
		note,
	)
	return err
}

func (s *Service) RecordOutcome(ctx context.Context, jobID string, outcome string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil
	}
	outcome = strings.TrimSpace(strings.ToLower(outcome))
	if outcome == "" {
		outcome = "unknown"
	}
	evidenceType := "packet.in_unknown_outcome_job"
	switch outcome {
	case "succeeded", "success":
		evidenceType = "packet.in_successful_job"
	case "failed", "failure":
		evidenceType = "packet.in_failed_job"
	case "cancelled", "canceled":
		evidenceType = "packet.in_cancelled_job"
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT rr.id, rr.retrieval_run_id, rr.chunk_id
FROM retrieval_results rr
JOIN retrieval_runs r ON r.id = rr.retrieval_run_id
WHERE r.job_id = ? AND rr.selected_for_packet = 1
ORDER BY rr.id ASC`, jobID)
	if err != nil {
		return err
	}
	defer rows.Close()
	now := time.Now().UnixMilli()
	for rows.Next() {
		var resultID int64
		var runID int64
		var chunkID sql.NullInt64
		if err := rows.Scan(&resultID, &runID, &chunkID); err != nil {
			return err
		}
		var exists int
		if err := s.db.QueryRowContext(ctx, `
SELECT 1 FROM context_evidence
WHERE retrieval_result_id = ? AND job_id = ? AND evidence_type = ?
LIMIT 1`, resultID, jobID, evidenceType).Scan(&exists); err == nil {
			continue
		}
		_, _ = s.db.ExecContext(ctx, `
INSERT INTO context_evidence(created_at, retrieval_result_id, retrieval_run_id, job_id, packet_id, chunk_id, evidence_type, weight, note)
VALUES(?,?,?,?,?,?,?, ?, ?)`,
			now,
			resultID,
			runID,
			jobID,
			nil,
			nullInt64(chunkID),
			evidenceType,
			1.0,
			"",
		)
	}
	return rows.Err()
}

func (s *Service) resolveSourceScope(ctx context.Context, dossierID *int64, explicit []int64) ([]int64, error) {
	if len(explicit) > 0 {
		return explicit, nil
	}
	if dossierID == nil {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT source_id FROM dossier_sources WHERE dossier_id = ? ORDER BY source_id ASC`, *dossierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Service) resolveWeights(ctx context.Context, reqKeyword, reqSemantic float64) (float64, float64) {
	if reqKeyword > 0 || reqSemantic > 0 {
		if reqKeyword <= 0 {
			reqKeyword = 0
		}
		if reqSemantic <= 0 {
			reqSemantic = 0
		}
		sum := reqKeyword + reqSemantic
		if sum == 0 {
			return 0.5, 0.5
		}
		return reqKeyword / sum, reqSemantic / sum
	}
	kw := parseFloat(s.setting(ctx, "retrieval_weight_keyword", "0.45"), 0.45)
	se := parseFloat(s.setting(ctx, "retrieval_weight_semantic", "0.55"), 0.55)
	sum := kw + se
	if sum <= 0 {
		return 0.5, 0.5
	}
	return kw / sum, se / sum
}

func (s *Service) setting(ctx context.Context, key, def string) string {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil || strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func round(v float64) float64 {
	return math.Round(v*1_000_000) / 1_000_000
}

func normalizeSemantic(score float64) float64 {
	v := (score + 1.0) / 2.0
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func parseFloat(raw string, def float64) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return def
	}
	return f
}

func coverageSelectIndices(rows []rankedCandidate, selectCount int) map[int]struct{} {
	out := map[int]struct{}{}
	if selectCount <= 0 || len(rows) == 0 {
		return out
	}
	if selectCount > len(rows) {
		selectCount = len(rows)
	}
	pathSeen := map[string]struct{}{}
	for idx, row := range rows {
		if len(out) >= selectCount {
			break
		}
		path := strings.TrimSpace(row.item.RelPath)
		if path == "" {
			path = strings.TrimSpace(row.item.AbsPath)
		}
		if path == "" {
			continue
		}
		if _, ok := pathSeen[path]; ok {
			continue
		}
		out[idx] = struct{}{}
		pathSeen[path] = struct{}{}
	}
	for idx := range rows {
		if len(out) >= selectCount {
			break
		}
		if _, ok := out[idx]; ok {
			continue
		}
		out[idx] = struct{}{}
	}
	return out
}

func (s *Service) pathUsefulnessScores(ctx context.Context) (map[string]float64, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT abs_path, usefulness_label, COUNT(*)
FROM retrieval_results
WHERE abs_path <> ''
GROUP BY abs_path, usefulness_label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type counts struct {
		useful int
		noisy  int
		total  int
	}
	acc := map[string]*counts{}
	for rows.Next() {
		var path string
		var label string
		var count int
		if err := rows.Scan(&path, &label, &count); err != nil {
			return nil, err
		}
		cur := acc[path]
		if cur == nil {
			cur = &counts{}
			acc[path] = cur
		}
		cur.total += count
		switch strings.TrimSpace(strings.ToLower(label)) {
		case "useful":
			cur.useful += count
		case "not_useful", "noisy", "misleading":
			cur.noisy += count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := map[string]float64{}
	for path, c := range acc {
		if c.total == 0 {
			continue
		}
		score := (float64(c.useful-c.noisy) / float64(c.total)) * 0.12
		if score > 0.12 {
			score = 0.12
		}
		if score < -0.12 {
			score = -0.12
		}
		out[path] = score
	}
	return out, nil
}

func (s *Service) dossierFileBias(ctx context.Context, dossierID *int64) (map[string]struct{}, map[string]struct{}) {
	high := map[string]struct{}{}
	noisy := map[string]struct{}{}
	if dossierID == nil || *dossierID <= 0 {
		return high, noisy
	}
	var highRaw string
	var noisyRaw string
	err := s.db.QueryRowContext(ctx, `
SELECT high_value_files_json, noisy_files_json
FROM dossier_profiles
WHERE dossier_id = ?`, *dossierID).Scan(&highRaw, &noisyRaw)
	if err != nil {
		return high, noisy
	}
	var highFiles []string
	var noisyFiles []string
	_ = json.Unmarshal([]byte(strings.TrimSpace(highRaw)), &highFiles)
	_ = json.Unmarshal([]byte(strings.TrimSpace(noisyRaw)), &noisyFiles)
	for _, f := range highFiles {
		v := strings.TrimSpace(f)
		if v != "" {
			high[v] = struct{}{}
		}
	}
	for _, f := range noisyFiles {
		v := strings.TrimSpace(f)
		if v != "" {
			noisy[v] = struct{}{}
		}
	}
	return high, noisy
}

func summarizeSnippet(snippet string) string {
	snippet = strings.TrimSpace(snippet)
	if snippet == "" {
		return ""
	}
	if len(snippet) <= 220 {
		return snippet
	}
	return snippet[:220] + "..."
}

func confidenceFromScores(keyword, semantic, hybrid float64) float64 {
	score := math.Max(hybrid, math.Max(keyword, semantic))
	if score < 0.1 {
		return 0.1
	}
	if score > 1 {
		return 1
	}
	return score
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(fallback)
}

func usefulnessToEvidence(label string) string {
	switch label {
	case "useful":
		return "retrieval.useful"
	case "not_useful":
		return "retrieval.not_useful"
	case "noisy":
		return "retrieval.noisy"
	case "insufficient":
		return "retrieval.insufficient"
	default:
		return "retrieval.unknown"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func nullInt64(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}
