package retrieval

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
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
	Actor           domain.ActorIdentity
	Source          domain.ActionSource
	Scope           domain.ForgeScope
	Provenance      domain.Provenance
	CorrelationID   string
	TraceID         string
	RequestID       string
	IdempotencyKey  string
	RequestedAt     int64
}

type SemanticSyscallProcessor interface {
	Process(ctx context.Context, req domain.SyscallRequest) (domain.SyscallResult, error)
}

type Service struct {
	db         *sql.DB
	search     *search.Service
	embeddings *embeddings.Service
	memory     *memory.Service
	syscalls   SemanticSyscallProcessor
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

func (s *Service) SetSyscallProcessor(processor SemanticSyscallProcessor) {
	if s != nil {
		s.syscalls = processor
	}
}

func NewEvidenceRequestID() string {
	var suffix [12]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return fmt.Sprintf("retrieval-evidence-%d", time.Now().UnixNano())
	}
	return "retrieval-evidence-" + hex.EncodeToString(suffix[:])
}

func (s *Service) Run(ctx context.Context, req RunRequest) (*Run, error) {
	if s == nil || s.syscalls == nil {
		return nil, fmt.Errorf("FORGE-K retrieval evidence authority is unavailable")
	}
	if strings.TrimSpace(req.Actor.ID) == "" || strings.TrimSpace(req.Actor.Kind) == "" ||
		strings.TrimSpace(string(req.Source)) == "" || strings.TrimSpace(req.Scope.WorkspaceID) == "" ||
		strings.TrimSpace(req.Scope.LaneID) == "" || strings.TrimSpace(req.Provenance.Actor) == "" ||
		strings.TrimSpace(req.Provenance.ActorType) == "" {
		return nil, fmt.Errorf("trusted retrieval actor, source, scope, and provenance are required")
	}
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
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		requestID = NewEvidenceRequestID()
	}
	req.RequestID = requestID
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		req.IdempotencyKey = requestID
	}
	if req.RequestedAt <= 0 {
		req.RequestedAt = time.Now().UnixMilli()
	}

	sourceIDs, err := s.resolveSourceScope(ctx, req.DossierID, req.SourceIDs)
	if err != nil {
		return nil, err
	}
	canonicalPaths, err := s.canonicalSourcePaths(ctx, sourceIDs)
	if err != nil {
		return nil, err
	}
	if len(req.Scope.SelectedPaths) > 0 && !equalCanonicalPaths(req.Scope.SelectedPaths, canonicalPaths) {
		return nil, fmt.Errorf("retrieval selected paths must exactly match resolved source roots")
	}
	req.Scope.SelectedPaths = canonicalPaths
	if existing, found, err := s.existingEvidenceRun(ctx, req, query, mode); err != nil {
		return nil, err
	} else if found {
		return existing, nil
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

	pathUsefulness, _ := s.pathUsefulnessScores(ctx, req.Scope)
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
			"vsaInfluence":      "disabled_unscoped",
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

func (s *Service) persistRun(
	ctx context.Context,
	query string,
	mode Mode,
	req RunRequest,
	wKeyword,
	wSemantic float64,
	results []Result,
) (*Run, error) {
	now := req.RequestedAt
	if now <= 0 {
		now = time.Now().UnixMilli()
	}
	weighting := map[string]any{
		"keyword":      wKeyword,
		"semantic":     wSemantic,
		"mode":         mode,
		"vsaInfluence": "disabled_unscoped",
	}
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		requestID = NewEvidenceRequestID()
	}
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = requestID
	}
	resultEvidence := make([]map[string]any, 0, len(results))
	for index := range results {
		row := &results[index]
		reason := map[string]any{}
		if len(row.SelectionReason) > 0 {
			if err := json.Unmarshal(row.SelectionReason, &reason); err != nil {
				return nil, fmt.Errorf("selection reason %d: %w", index, err)
			}
		}
		resultEvidence = append(resultEvidence, map[string]any{
			"evidenceId": requestID + ":retrieval_result:" + strconv.Itoa(index),
			"chunkId":    row.ChunkID, "fileId": row.FileID,
			"absPath": row.AbsPath, "relPath": row.RelPath, "rankIndex": index,
			"keywordScore": row.KeywordScore, "semanticScore": row.SemanticScore, "hybridScore": row.HybridScore,
			"snippet": row.Snippet, "selectedForPacket": row.SelectedForPacket, "selectionReason": reason,
		})
	}
	payload := map[string]any{
		"evidenceId": requestID + ":retrieval_run", "createdAt": now,
		"query": query, "mode": string(mode), "weighting": weighting, "notes": req.Notes,
		"results": resultEvidence,
	}
	if req.DossierID != nil {
		payload["dossierId"] = *req.DossierID
	}
	if req.PacketID != nil {
		payload["packetId"] = *req.PacketID
	}
	if req.JobID != nil {
		payload["jobId"] = *req.JobID
	}
	correlationID := strings.TrimSpace(req.CorrelationID)
	if correlationID == "" {
		correlationID = "corr-" + requestID
	}
	traceID := strings.TrimSpace(req.TraceID)
	if traceID == "" {
		traceID = correlationID
	}
	provenance := req.Provenance
	provenance.TraceID = traceID
	result, err := s.syscalls.Process(ctx, domain.SyscallRequest{
		ID: requestID, Action: domain.ActionRecordRetrievalEvidence,
		Actor: req.Actor, Source: req.Source, Scope: req.Scope, Payload: payload,
		Provenance: provenance, CorrelationID: correlationID, TraceID: traceID,
		IdempotencyKey: idempotencyKey, RequestedAt: now,
		RequiredCapability: "retrieval.evidence.record",
		Metadata:           map[string]any{"modelCanonicalAuthority": false, "memoryObservationMutation": false},
	})
	if err != nil {
		return nil, fmt.Errorf("record retrieval evidence: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("record retrieval evidence rejected: %s", result.DeterministicErrCode)
	}
	return s.GetRunByEvidenceID(ctx, requestID+":retrieval_run")
}

func (s *Service) GetRunByEvidenceID(ctx context.Context, evidenceID string) (*Run, error) {
	var runID int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM retrieval_runs WHERE evidence_id = ?`, strings.TrimSpace(evidenceID)).Scan(&runID); err != nil {
		return nil, err
	}
	return s.GetRun(ctx, runID)
}

func (s *Service) existingEvidenceRun(ctx context.Context, req RunRequest, query string, mode Mode) (*Run, bool, error) {
	// Only the deterministic in-process job principal may use the read-before-
	// search shortcut. User and proposal sources must always traverse Kernel
	// authorization, and random interactive request IDs do not need this path.
	if req.Source != domain.SourceInternal || req.JobID == nil || strings.TrimSpace(*req.JobID) == "" ||
		req.Actor.ID != "forge.jobs" || req.Actor.Kind != "service" {
		return nil, false, nil
	}
	var (
		runID                              int64
		storedQuery, storedMode, workspace string
		lane, selectedJSON                 string
		proposedBy, provenanceJSON         string
		authorizationFingerprint           string
		dossierID, packetID                sql.NullInt64
		jobID                              sql.NullString
	)
	err := s.db.QueryRowContext(ctx, `
SELECT id, query, mode, COALESCE(original_dossier_id,dossier_id), COALESCE(original_packet_id,packet_id),
       COALESCE(original_job_id,job_id), workspace_id, lane_id, selected_paths_json,
       proposed_by, provenance_json, authorization_fingerprint
FROM retrieval_runs WHERE evidence_id = ?`, req.RequestID+":retrieval_run").Scan(
		&runID, &storedQuery, &storedMode, &dossierID, &packetID, &jobID, &workspace, &lane, &selectedJSON,
		&proposedBy, &provenanceJSON, &authorizationFingerprint,
	)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("look up immutable retrieval evidence: %w", err)
	}
	var storedPaths []string
	if err := json.Unmarshal([]byte(selectedJSON), &storedPaths); err != nil {
		return nil, false, fmt.Errorf("stored retrieval evidence scope is invalid: %w", err)
	}
	var storedProvenance domain.Provenance
	if err := json.Unmarshal([]byte(provenanceJSON), &storedProvenance); err != nil {
		return nil, false, fmt.Errorf("stored retrieval evidence provenance is invalid: %w", err)
	}
	expectedProvenance := req.Provenance
	expectedProvenance.TraceID = strings.TrimSpace(req.TraceID)
	if expectedProvenance.TraceID == "" {
		expectedProvenance.TraceID = strings.TrimSpace(req.CorrelationID)
	}
	if strings.TrimSpace(storedQuery) != query || Mode(storedMode) != mode ||
		strings.TrimSpace(workspace) != strings.TrimSpace(req.Scope.WorkspaceID) ||
		strings.TrimSpace(lane) != strings.TrimSpace(req.Scope.LaneID) ||
		!equalCanonicalPaths(storedPaths, req.Scope.SelectedPaths) ||
		!sameOptionalInt64(dossierID, req.DossierID) || !sameOptionalInt64(packetID, req.PacketID) ||
		!sameOptionalString(jobID, req.JobID) || proposedBy != req.Actor.ID ||
		strings.TrimSpace(authorizationFingerprint) == "" || storedProvenance != expectedProvenance {
		return nil, false, fmt.Errorf("existing retrieval evidence does not match retry identity or scope")
	}
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return nil, false, err
	}
	return run, true, nil
}

func sameOptionalInt64(stored sql.NullInt64, expected *int64) bool {
	return (expected == nil && !stored.Valid) || (expected != nil && stored.Valid && stored.Int64 == *expected)
}

func sameOptionalString(stored sql.NullString, expected *string) bool {
	return (expected == nil && !stored.Valid) || (expected != nil && stored.Valid && stored.String == *expected)
}

func (s *Service) GetRun(ctx context.Context, runID int64) (*Run, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, query, mode, COALESCE(original_dossier_id,dossier_id),
       COALESCE(original_packet_id,packet_id), COALESCE(original_job_id,job_id), weighting_json, notes
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
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM retrieval_runs WHERE COALESCE(original_job_id,job_id) = ? ORDER BY id DESC LIMIT ?`, jobID, limit)
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

func (s *Service) resultsForRun(ctx context.Context, runID int64) ([]Result, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT rr.id, rr.retrieval_run_id, COALESCE(rr.original_chunk_id,rr.chunk_id),
       COALESCE(rr.original_file_id,rr.file_id), rr.abs_path, rr.rel_path, rr.rank_index,
       rr.keyword_score, rr.semantic_score, rr.hybrid_score, rr.snippet,
	       rr.selected_for_packet, COALESCE(rup.label,'unknown'), COALESCE(rup.note,''),
       COALESCE((SELECT reason_json FROM retrieval_result_selection WHERE retrieval_result_id = rr.id), '{}') AS reason_json,
       (SELECT observation_id FROM retrieval_result_observations WHERE retrieval_result_id = rr.id ORDER BY created_at DESC LIMIT 1) AS observation_id
FROM retrieval_results rr
LEFT JOIN retrieval_usefulness_projection rup ON rup.retrieval_result_id = rr.id
WHERE rr.retrieval_run_id = ?
ORDER BY rr.rank_index ASC`, runID)
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

type UsefulnessRequest struct {
	ResultID       int64
	Label          string
	Note           string
	JobID          *string
	PacketID       *int64
	Metadata       map[string]any
	Actor          domain.ActorIdentity
	Source         domain.ActionSource
	Scope          domain.ForgeScope
	Provenance     domain.Provenance
	CorrelationID  string
	TraceID        string
	RequestID      string
	IdempotencyKey string
	RequestedAt    int64
}

// RecordUsefulness submits utility evidence to the production FORGE-K
// boundary. The retrieval service has no direct write authority over source
// evidence, memory aggregates, VSA counters, or the rebuildable projection.
func (s *Service) RecordUsefulness(ctx context.Context, req UsefulnessRequest) (domain.SyscallResult, error) {
	if s == nil || s.syscalls == nil {
		return domain.SyscallResult{}, fmt.Errorf("FORGE-K retrieval usefulness authority is unavailable")
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return domain.SyscallResult{}, fmt.Errorf("idempotency key is required")
	}
	if strings.TrimSpace(req.RequestID) == "" {
		req.RequestID = NewEvidenceRequestID()
	}
	if req.RequestedAt <= 0 {
		req.RequestedAt = time.Now().UnixMilli()
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = "corr-" + req.RequestID
	}
	if strings.TrimSpace(req.TraceID) == "" {
		req.TraceID = req.CorrelationID
	}
	req.Provenance.TraceID = req.TraceID
	payload := map[string]any{
		"resultId": req.ResultID, "label": strings.TrimSpace(strings.ToLower(req.Label)),
		"note": strings.TrimSpace(req.Note), "metadata": req.Metadata,
	}
	if req.JobID != nil {
		payload["jobId"] = *req.JobID
	}
	if req.PacketID != nil {
		payload["packetId"] = *req.PacketID
	}
	result, err := s.syscalls.Process(ctx, domain.SyscallRequest{
		ID: req.RequestID, Action: domain.ActionRecordRetrievalUsefulness,
		Actor: req.Actor, Source: req.Source, Scope: req.Scope, Payload: payload,
		Provenance: req.Provenance, CorrelationID: req.CorrelationID, TraceID: req.TraceID,
		IdempotencyKey: req.IdempotencyKey, RequestedAt: req.RequestedAt,
		RequiredCapability: "retrieval.usefulness.record",
	})
	if err != nil {
		return result, err
	}
	if !result.Success {
		return result, fmt.Errorf("record retrieval usefulness rejected: %s", result.DeterministicErrCode)
	}
	return result, nil
}

func (s *Service) resolveSourceScope(ctx context.Context, dossierID *int64, explicit []int64) ([]int64, error) {
	if len(explicit) > 0 {
		return normalizedSourceIDs(explicit)
	}
	query := `SELECT id FROM sources ORDER BY id ASC`
	args := []any{}
	if dossierID != nil {
		query = `SELECT source_id FROM dossier_sources WHERE dossier_id = ? ORDER BY source_id ASC`
		args = append(args, *dossierID)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("retrieval requires at least one resolved source")
	}
	return normalizedSourceIDs(out)
}

func normalizedSourceIDs(ids []int64) ([]int64, error) {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("retrieval source ids must be positive")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func (s *Service) canonicalSourcePaths(ctx context.Context, sourceIDs []int64) ([]string, error) {
	if len(sourceIDs) == 0 {
		return nil, fmt.Errorf("retrieval requires at least one resolved source")
	}
	paths := make([]string, 0, len(sourceIDs))
	for _, sourceID := range sourceIDs {
		var sourcePath string
		if err := s.db.QueryRowContext(ctx, `SELECT path FROM sources WHERE id = ?`, sourceID).Scan(&sourcePath); err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("retrieval source %d does not exist", sourceID)
			}
			return nil, fmt.Errorf("resolve retrieval source %d: %w", sourceID, err)
		}
		path := filepath.Clean(strings.TrimSpace(sourcePath))
		if path == "" || path == "." {
			return nil, fmt.Errorf("retrieval source %d has no canonical path", sourceID)
		}
		paths = append(paths, path)
	}
	return canonicalPathSet(paths)
}

func canonicalPathSet(paths []string) ([]string, error) {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, raw := range paths {
		path := filepath.Clean(strings.TrimSpace(raw))
		if path == "" || path == "." {
			return nil, fmt.Errorf("retrieval selected paths must be non-empty canonical paths")
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("retrieval requires at least one selected source path")
	}
	return out, nil
}

func equalCanonicalPaths(left, right []string) bool {
	a, err := canonicalPathSet(left)
	if err != nil {
		return false
	}
	b, err := canonicalPathSet(right)
	if err != nil || len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
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

func clampFloat(v, minVal, maxVal float64) float64 {
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
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

func (s *Service) pathUsefulnessScores(ctx context.Context, scope domain.ForgeScope) (map[string]float64, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT rr.abs_path, COALESCE(rup.label, 'unknown'), COUNT(*)
FROM retrieval_results rr
JOIN retrieval_runs r ON r.id = rr.retrieval_run_id
LEFT JOIN retrieval_usefulness_projection rup ON rup.retrieval_result_id = rr.id
WHERE rr.abs_path <> ''
  AND rr.evidence_id <> '' AND r.evidence_id <> ''
  AND r.syscall_id <> '' AND r.provenance_id <> ''
  AND r.workspace_id = ? AND r.lane_id = ?
GROUP BY rr.abs_path, COALESCE(rup.label, 'unknown')`,
		strings.TrimSpace(scope.WorkspaceID), strings.TrimSpace(scope.LaneID))
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
