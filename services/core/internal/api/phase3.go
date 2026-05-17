package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"forge/projectforge/services/core/internal/artifacts"
	"forge/projectforge/services/core/internal/dossiers"
	"forge/projectforge/services/core/internal/evaluations"
	"forge/projectforge/services/core/internal/forgekshadow"
	"forge/projectforge/services/core/internal/imports"
	"forge/projectforge/services/core/internal/insights"
	"forge/projectforge/services/core/internal/jobs"
	"forge/projectforge/services/core/internal/reconciliation"
	"forge/projectforge/services/core/internal/retrieval"
	"forge/projectforge/services/core/internal/reviews"
)

const phase3JSONRequestBodyLimit = 1 << 20

var errPhase3RequestBodyTooLarge = errors.New("phase3 json request body too large")

type optionalInt64 struct {
	Value *int64
}

func (o *optionalInt64) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "null" || s == "" {
		o.Value = nil
		return nil
	}
	var asNum json.Number
	if err := json.Unmarshal(data, &asNum); err == nil {
		i, err := asNum.Int64()
		if err != nil {
			return err
		}
		o.Value = &i
		return nil
	}
	var asStr string
	if err := json.Unmarshal(data, &asStr); err == nil {
		asStr = strings.TrimSpace(asStr)
		if asStr == "" {
			o.Value = nil
			return nil
		}
		i, err := strconv.ParseInt(asStr, 10, 64)
		if err != nil {
			return err
		}
		o.Value = &i
		return nil
	}
	return fmt.Errorf("expected integer or string integer")
}

func parseOptionalInt(raw string) *int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		return nil
	}
	return &v
}

func (s *Server) handleEmbeddingStatus(w http.ResponseWriter, r *http.Request) {
	provider := strings.TrimSpace(r.URL.Query().Get("provider"))
	model := strings.TrimSpace(r.URL.Query().Get("model"))
	status, err := s.embeddings.StatusBySource(r.Context(), provider, model)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	cfg := s.embeddings.CurrentConfig(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"config":         cfg,
		"health":         s.embeddings.ProviderHealth(r.Context(), provider, model),
		"truthAuthority": false,
		"status":         status,
	})
}

func (s *Server) handleReembed(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SourceID optionalInt64 `json:"sourceId"`
		Provider string        `json:"provider"`
		Model    string        `json:"model"`
	}
	if err := decodePhase3JSONBody(r, &body); err != nil {
		writePhase3DecodeError(w, err)
		return
	}
	var (
		out any
		err error
	)
	if body.SourceID.Value != nil {
		out, err = s.embeddings.ReembedSource(r.Context(), *body.SourceID.Value, body.Provider, body.Model)
	} else {
		out, err = s.embeddings.ReembedAll(r.Context(), body.Provider, body.Model)
	}
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	_ = s.log.Emit(r.Context(), "embeddings.reembedded", map[string]any{
		"sourceId": body.SourceID.Value,
		"provider": body.Provider,
		"model":    body.Model,
	})
	writeJSON(w, http.StatusOK, map[string]any{"result": out})
}

func (s *Server) handleCreateRetrievalRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query           string        `json:"query"`
		Mode            string        `json:"mode"`
		Limit           int           `json:"limit"`
		SelectForPacket int           `json:"selectForPacket"`
		DossierID       optionalInt64 `json:"dossierId"`
		SourceIDs       []int64       `json:"sourceIds"`
		WeightKeyword   float64       `json:"weightKeyword"`
		WeightSemantic  float64       `json:"weightSemantic"`
		Provider        string        `json:"provider"`
		Model           string        `json:"model"`
		JobID           *string       `json:"jobId"`
		PacketID        optionalInt64 `json:"packetId"`
		Notes           string        `json:"notes"`
	}
	if err := decodePhase3JSONBody(r, &body); err != nil {
		writePhase3DecodeError(w, err)
		return
	}
	started := time.Now()
	run, err := s.retrieval.Run(r.Context(), retrieval.RunRequest{
		Query:           body.Query,
		Mode:            retrieval.Mode(strings.TrimSpace(body.Mode)),
		Limit:           body.Limit,
		SelectForPacket: body.SelectForPacket,
		DossierID:       body.DossierID.Value,
		SourceIDs:       body.SourceIDs,
		WeightKeyword:   body.WeightKeyword,
		WeightSemantic:  body.WeightSemantic,
		Provider:        body.Provider,
		Model:           body.Model,
		JobID:           body.JobID,
		PacketID:        body.PacketID.Value,
		Notes:           body.Notes,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	_ = s.log.Emit(r.Context(), "retrieval.run.created", map[string]any{"runId": run.ID, "mode": run.Mode, "query": run.Query})
	s.observeRetrievalRunMetadata(r.Context(), run, strings.TrimSpace(body.Model), time.Since(started))
	writeJSON(w, http.StatusOK, map[string]any{"run": run})
}

func (s *Server) observeRetrievalRunMetadata(ctx context.Context, run *retrieval.Run, embeddingModelID string, duration time.Duration) {
	if s == nil || s.forgeKShadow == nil || run == nil {
		return
	}
	resultCount := len(run.Results)
	selectedCount := 0
	firstResultID := ""
	firstSourceRefID := ""
	firstRank := 0
	scores := make([]float64, 0, resultCount)
	for _, result := range run.Results {
		if result.SelectedForPacket {
			selectedCount++
		}
		if firstResultID == "" {
			firstResultID = strconv.FormatInt(result.ID, 10)
			if result.ChunkID != nil {
				firstSourceRefID = strconv.FormatInt(*result.ChunkID, 10)
			}
			if result.RankIndex >= 0 {
				firstRank = result.RankIndex + 1
			}
		}
		scores = append(scores, result.HybridScore)
	}
	s.forgeKShadow.ObserveRetrievalMetadataBestEffort(ctx, forgekshadow.RetrievalMetadataInput{
		WorkspaceID:       s.cfg.WorkspaceDir,
		RequestID:         middleware.GetReqID(ctx),
		RetrievalRunID:    strconv.FormatInt(run.ID, 10),
		RetrievalResultID: firstResultID,
		SourceType:        sourceTypeForRetrievalResult(firstSourceRefID),
		SourceRefID:       firstSourceRefID,
		ResultCount:       resultCount,
		SelectedCount:     selectedCount,
		ScoreSummary:      retrievalScoreSummary(scores),
		RankingPosition:   firstRank,
		RetrievalStrategy: string(run.Mode),
		IndexType:         indexTypeForRetrievalMode(run.Mode),
		EmbeddingModelID:  embeddingModelID,
		Duration:          duration,
		Metadata: map[string]any{
			"touchpoint": "retrieval_run_created",
		},
	})
}

func sourceTypeForRetrievalResult(sourceRefID string) string {
	if strings.TrimSpace(sourceRefID) == "" {
		return ""
	}
	return "chunk"
}

func indexTypeForRetrievalMode(mode retrieval.Mode) string {
	switch mode {
	case retrieval.ModeKeyword:
		return "fts"
	case retrieval.ModeSemantic:
		return "vector"
	case retrieval.ModeHybrid:
		return "hybrid"
	default:
		return "unknown"
	}
}

func retrievalScoreSummary(scores []float64) string {
	if len(scores) == 0 {
		return ""
	}
	maxScore := scores[0]
	sum := 0.0
	for _, score := range scores {
		if score > maxScore {
			maxScore = score
		}
		sum += score
	}
	return fmt.Sprintf("max=%.3f avg=%.3f", maxScore, sum/float64(len(scores)))
}

func (s *Server) handleListRetrievalRuns(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	dossierID := parseOptionalInt(r.URL.Query().Get("dossierId"))
	jobID := strings.TrimSpace(r.URL.Query().Get("jobId"))
	var (
		runs []retrieval.Run
		err  error
	)
	if jobID != "" {
		runs, err = s.retrieval.ListRunsByJob(r.Context(), jobID, limit)
	} else {
		runs, err = s.retrieval.ListRuns(r.Context(), limit, dossierID)
	}
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (s *Server) handleGetRetrievalRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	run, err := s.retrieval.GetRun(r.Context(), id)
	if err != nil {
		writeAPIRequestError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"run": run})
}

func (s *Server) handleGetRetrievalRunVSASignals(w http.ResponseWriter, r *http.Request) {
	runID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	signals, err := s.memory.RetrievalRunVSASignals(r.Context(), runID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"signals": signals})
}

func (s *Server) handleGetRetrievalResultVSASignal(w http.ResponseWriter, r *http.Request) {
	resultID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	signal, err := s.memory.RetrievalResultVSASignal(r.Context(), resultID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"signal": signal})
}

func (s *Server) handleMarkRetrievalUsefulness(w http.ResponseWriter, r *http.Request) {
	resultID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Label    string        `json:"label"`
		Note     string        `json:"note"`
		JobID    *string       `json:"jobId"`
		PacketID optionalInt64 `json:"packetId"`
	}
	if err := decodePhase3JSONBody(r, &body); err != nil {
		writePhase3DecodeError(w, err)
		return
	}
	if err := s.retrieval.MarkUsefulness(r.Context(), resultID, body.Label, body.Note, body.JobID, body.PacketID.Value); err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "resultId": resultID})
}

func (s *Server) handleCreateDossier(w http.ResponseWriter, r *http.Request) {
	var body dossiers.CreateRequest
	if err := decodePhase3JSONBody(r, &body); err != nil {
		writePhase3DecodeError(w, err)
		return
	}
	d, err := s.dossiers.Create(r.Context(), body)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dossier": d})
}

func (s *Server) handleListDossiers(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := s.dossiers.List(r.Context(), limit)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dossiers": list})
}

func (s *Server) handleGetDossierDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	detail, err := s.dossiers.Detail(r.Context(), id)
	if err != nil {
		writeAPIRequestError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detail": detail})
}

func (s *Server) handleUpdateDossier(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body dossiers.UpdateRequest
	if err := decodePhase3JSONBody(r, &body); err != nil {
		writePhase3DecodeError(w, err)
		return
	}
	d, err := s.dossiers.Update(r.Context(), id, body)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dossier": d})
}

func (s *Server) handleGenerateDossierBrief(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Notes string `json:"notes"`
	}
	if err := decodeOptionalPhase3JSONBody(r, &body); err != nil {
		writePhase3DecodeError(w, err)
		return
	}
	brief, err := s.dossiers.GenerateBrief(r.Context(), id, body.Notes)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	_, _ = s.artifacts.CreateTextArtifact(r.Context(), artifacts.CreateTextArtifactRequest{
		Type:     "dossier_brief",
		Title:    fmt.Sprintf("Dossier brief %d", brief.ID),
		FileName: fmt.Sprintf("dossier-brief-%d.md", brief.ID),
		Subdir:   "dossiers",
		Content:  brief.SummaryMarkdown,
		MimeType: "text/markdown",
		Metadata: map[string]any{"dossierId": id, "briefId": brief.ID},
	})
	writeJSON(w, http.StatusOK, map[string]any{"brief": brief})
}

func (s *Server) handleCreateEvaluation(w http.ResponseWriter, r *http.Request) {
	var body evaluations.SaveRequest
	if err := decodePhase3JSONBody(r, &body); err != nil {
		writePhase3DecodeError(w, err)
		return
	}
	rec, err := s.evals.Save(r.Context(), body)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"evaluation": rec})
}

func (s *Server) handleListEvaluations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	dossierID := parseOptionalInt(r.URL.Query().Get("dossierId"))
	list, err := s.evals.List(r.Context(), limit, dossierID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"evaluations": list})
}

func (s *Server) handleAdapterMetrics(w http.ResponseWriter, r *http.Request) {
	dossierID := parseOptionalInt(r.URL.Query().Get("dossierId"))
	rows, err := s.evals.AdapterMetrics(r.Context(), dossierID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"metrics": rows})
}

type cloneMetadata struct {
	TemplateID             string          `json:"templateId"`
	UserRequest            string          `json:"userRequest"`
	Objective              string          `json:"objective"`
	Query                  string          `json:"query"`
	Scope                  jobs.ScopeInput `json:"scope"`
	SourceContextRecordIDs []int64         `json:"sourceContextRecordIds"`
	Constraints            []string        `json:"constraints"`
	Instructions           string          `json:"instructions"`
	ExecutionMode          string          `json:"executionMode"`
	ExpectedOutput         map[string]any  `json:"expectedOutput"`
	RequestPayload         map[string]any  `json:"requestPayload"`
}

type cloneRequest struct {
	TemplateID             string           `json:"templateId"`
	Title                  string           `json:"title"`
	UserRequest            string           `json:"userRequest"`
	Objective              string           `json:"objective"`
	Query                  string           `json:"query"`
	Scope                  *jobs.ScopeInput `json:"scope"`
	SourceContextRecordIDs []int64          `json:"sourceContextRecordIds"`
	Constraints            []string         `json:"constraints"`
	Instructions           string           `json:"instructions"`
	ExecutionMode          string           `json:"executionMode"`
	RequestPayload         map[string]any   `json:"requestPayload"`
	Note                   string           `json:"note"`
}

func (s *Server) handleReplayJob(w http.ResponseWriter, r *http.Request) {
	s.cloneJobWithRelation(w, r, "replay")
}

func (s *Server) handleRetryJob(w http.ResponseWriter, r *http.Request) {
	s.cloneJobWithRelation(w, r, "retry")
}

func (s *Server) cloneJobWithRelation(w http.ResponseWriter, r *http.Request, relation string) {
	parentID := strings.TrimSpace(chi.URLParam(r, "id"))
	if parentID == "" {
		http.Error(w, "job id required", http.StatusBadRequest)
		return
	}
	var body cloneRequest
	if err := decodeOptionalPhase3JSONBody(r, &body); err != nil {
		writePhase3DecodeError(w, err)
		return
	}
	parent, err := s.jobs.Get(r.Context(), parentID)
	if err != nil {
		writeAPIRequestError(w, http.StatusNotFound, err)
		return
	}
	var meta cloneMetadata
	if err := json.Unmarshal(parent.Metadata, &meta); err != nil {
		http.Error(w, "job metadata decode failed", http.StatusBadRequest)
		return
	}
	req := jobs.CreateRequest{
		TemplateID:             nonEmpty(body.TemplateID, meta.TemplateID),
		Title:                  nonEmpty(body.Title, parent.Title+" ("+relation+")"),
		UserRequest:            nonEmpty(body.UserRequest, meta.UserRequest),
		Objective:              nonEmpty(body.Objective, meta.Objective),
		InitiatingSource:       "job_" + relation,
		Query:                  nonEmpty(body.Query, meta.Query),
		Scope:                  meta.Scope,
		SourceContextRecordIDs: chooseInt64Slice(body.SourceContextRecordIDs, meta.SourceContextRecordIDs),
		Constraints:            chooseStringSlice(body.Constraints, meta.Constraints),
		Instructions:           nonEmpty(body.Instructions, meta.Instructions),
		ExecutionMode:          nonEmpty(body.ExecutionMode, meta.ExecutionMode),
		ExpectedOutput:         copyMap(meta.ExpectedOutput),
		RequestPayload:         mergeMaps(meta.RequestPayload, body.RequestPayload),
	}
	if body.Scope != nil {
		req.Scope = *body.Scope
	}
	if relation == "replay" {
		req.TemplateID = meta.TemplateID
		req.UserRequest = meta.UserRequest
		req.Objective = meta.Objective
		req.Query = meta.Query
		req.Scope = meta.Scope
		req.SourceContextRecordIDs = meta.SourceContextRecordIDs
		req.Constraints = meta.Constraints
		req.Instructions = meta.Instructions
		req.ExecutionMode = meta.ExecutionMode
		req.ExpectedOutput = copyMap(meta.ExpectedOutput)
		req.RequestPayload = copyMap(meta.RequestPayload)
	}

	child, err := s.jobs.Create(r.Context(), req)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	changeSummary := map[string]any{
		"relationType": relation,
		"note":         strings.TrimSpace(body.Note),
		"changes": map[string]any{
			"templateId":     diffString(meta.TemplateID, req.TemplateID),
			"query":          diffString(meta.Query, req.Query),
			"executionMode":  diffString(meta.ExecutionMode, req.ExecutionMode),
			"instructions":   diffString(meta.Instructions, req.Instructions),
			"constraints":    req.Constraints,
			"scope":          req.Scope.ToMap(),
			"requestPayload": req.RequestPayload,
		},
	}
	oldTpl, _ := jobs.TemplateByID(meta.TemplateID)
	newTpl, _ := jobs.TemplateByID(req.TemplateID)
	changeSummary["approval"] = map[string]any{
		"fromRequired": oldTpl.ApprovalRequired,
		"toRequired":   newTpl.ApprovalRequired,
		"fromRisk":     oldTpl.RiskClass,
		"toRisk":       newTpl.RiskClass,
	}

	link, err := s.lineage.Link(r.Context(), parentID, child.ID, relation, changeSummary)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": child, "lineage": link, "changeSummary": changeSummary})
}

func (s *Server) handleJobLineage(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(chi.URLParam(r, "id"))
	line, err := s.lineage.ForJob(r.Context(), jobID)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, line)
}

func (s *Server) handleCreateImportedExecution(w http.ResponseWriter, r *http.Request) {
	var body imports.CreateRequest
	if err := decodePhase3JSONBody(r, &body); err != nil {
		writePhase3DecodeError(w, err)
		return
	}
	rec, err := s.imports.Create(r.Context(), body)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}

	if rec.OriginJobID != nil {
		content := fmt.Sprintf("# Imported Execution\n\nAdapter: %s\nExternal Run: %s\n\n## Summary\n%s\n\n## Diff/Patch Summary\n%s\n\n## Notes\n%s\n",
			rec.AdapterID, rec.ExternalRunID, rec.Summary, rec.DiffSummary, rec.ExecutionNotes)
		_, _ = s.artifacts.CreateTextArtifact(r.Context(), artifacts.CreateTextArtifactRequest{
			JobID:    rec.OriginJobID,
			PacketID: rec.OriginPacketID,
			Type:     "imported_execution_summary",
			Title:    "Imported execution result",
			FileName: fmt.Sprintf("imported-execution-%d.md", rec.ID),
			Subdir:   "imports",
			Content:  content,
			MimeType: "text/markdown",
			Metadata: map[string]any{"importId": rec.ID, "adapterId": rec.AdapterID},
		})
	}
	_, _ = s.reconcile.Save(r.Context(), reconciliation.SaveRequest{
		ImportID:           rec.ID,
		ChangedFiles:       []string{},
		FailureReasons:     []string{},
		UnresolvedIssues:   []string{},
		SuggestedNextSteps: []string{},
		AgentNotes:         rec.ExecutionNotes,
		PatchSummary:       rec.DiffSummary,
		ReviewStatus:       "pending",
	})
	targetID := strconv.FormatInt(rec.ID, 10)
	_, _ = s.reviews.Create(r.Context(), reviews.CreateRequest{
		TargetType: "import",
		TargetID:   targetID,
		DossierID:  rec.DossierID,
		Status:     reviews.StatusPending,
		Summary:    "Imported execution awaiting review",
		Notes:      "Auto-created on import. Review before downstream retries.",
		Reviewer:   "operator",
	})
	_ = s.log.Emit(r.Context(), "import.execution.created", map[string]any{"importId": rec.ID, "adapterId": rec.AdapterID})
	writeJSON(w, http.StatusOK, map[string]any{"importedExecution": rec})
}

func (s *Server) handleListImportedExecutions(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	dossierID := parseOptionalInt(r.URL.Query().Get("dossierId"))
	list, err := s.imports.List(r.Context(), limit, dossierID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"imports": list})
}

func (s *Server) handleGenerateInsights(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DossierID optionalInt64 `json:"dossierId"`
	}
	if err := decodeOptionalPhase3JSONBody(r, &body); err != nil {
		writePhase3DecodeError(w, err)
		return
	}
	rows, err := s.insights.Generate(r.Context(), insights.GenerateRequest{DossierID: body.DossierID.Value})
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"insights": rows})
}

func (s *Server) handleListInsights(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	dossierID := parseOptionalInt(r.URL.Query().Get("dossierId"))
	rows, err := s.insights.List(r.Context(), limit, dossierID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"insights": rows})
}

func chooseInt64Slice(v, fallback []int64) []int64 {
	if len(v) == 0 {
		return fallback
	}
	return v
}

func chooseStringSlice(v, fallback []string) []string {
	if len(v) == 0 {
		return fallback
	}
	return v
}

func copyMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	for k, val := range v {
		out[k] = val
	}
	return out
}

func mergeMaps(a, b map[string]any) map[string]any {
	out := copyMap(a)
	for k, v := range b {
		out[k] = v
	}
	return out
}

func diffString(before, after string) map[string]any {
	return map[string]any{
		"before":  before,
		"after":   after,
		"changed": strings.TrimSpace(before) != strings.TrimSpace(after),
	}
}

func nonEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

func decodePhase3JSONBody(r *http.Request, target any) error {
	raw, err := readPhase3RequestBody(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func decodeOptionalPhase3JSONBody(r *http.Request, target any) error {
	raw, err := readPhase3RequestBody(r)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	return json.Unmarshal(raw, target)
}

func readPhase3RequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, io.EOF
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, phase3JSONRequestBodyLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > phase3JSONRequestBodyLimit {
		return nil, errPhase3RequestBodyTooLarge
	}
	return raw, nil
}

func writePhase3DecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errPhase3RequestBodyTooLarge) {
		http.Error(w, "phase3 json request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "invalid json", http.StatusBadRequest)
}
