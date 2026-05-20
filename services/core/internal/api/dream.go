package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/aios/dream"
)

const dreamRunRequestBodyLimit = 1 << 20

var errDreamRunRequestBodyTooLarge = errors.New("dream run request body too large")

type dreamRunRequest struct {
	Mode                                   string         `json:"mode"`
	Purpose                                string         `json:"purpose,omitempty"`
	WorkspaceID                            string         `json:"workspaceId"`
	LaneID                                 string         `json:"laneId,omitempty"`
	WindowHours                            int            `json:"windowHours,omitempty"`
	MaxCandidates                          int            `json:"maxCandidates,omitempty"`
	DryRun                                 *bool          `json:"dryRun,omitempty"`
	AllowLongTermPromotion                 bool           `json:"allowLongTermPromotion,omitempty"`
	RequireOperatorReviewForLongTerm       bool           `json:"requireOperatorReviewForLongTerm"`
	AllowCommits                           bool           `json:"allowCommits,omitempty"`
	SkillID                                string         `json:"skillId,omitempty"`
	LessonID                               string         `json:"lessonId,omitempty"`
	LabID                                  string         `json:"labId,omitempty"`
	ExamID                                 string         `json:"examId,omitempty"`
	AllowSkillPromotion                    bool           `json:"allowSkillPromotion,omitempty"`
	RequireOperatorReviewForSkillPromotion bool           `json:"requireOperatorReviewForSkillPromotion"`
	PersistReport                          bool           `json:"persistReport,omitempty"`
	CorrelationID                          string         `json:"correlationId,omitempty"`
	TraceID                                string         `json:"traceId,omitempty"`
	ProposedBy                             string         `json:"proposedBy,omitempty"`
	Metadata                               map[string]any `json:"metadata,omitempty"`
}

func (s *Server) handleDreamRun(w http.ResponseWriter, r *http.Request) {
	if s.dream == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "request_failed", "dream service unavailable", nil)
		return
	}
	var body dreamRunRequest
	if err := decodeDreamRunJSONBody(r, &body); err != nil {
		writeDreamRunDecodeError(w, err)
		return
	}
	if strings.TrimSpace(body.WorkspaceID) == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "workspaceId required", nil)
		return
	}
	report, err := s.dream.Run(r.Context(), dream.RunRequest{
		Mode:                                   dream.Mode(strings.TrimSpace(body.Mode)),
		Purpose:                                dream.Purpose(strings.TrimSpace(body.Purpose)),
		WorkspaceID:                            strings.TrimSpace(body.WorkspaceID),
		LaneID:                                 strings.TrimSpace(body.LaneID),
		WindowHours:                            body.WindowHours,
		MaxCandidates:                          body.MaxCandidates,
		DryRun:                                 body.DryRun,
		AllowLongTermPromotion:                 body.AllowLongTermPromotion,
		RequireOperatorReviewForLongTerm:       body.RequireOperatorReviewForLongTerm,
		AllowCommits:                           body.AllowCommits,
		SkillID:                                strings.TrimSpace(body.SkillID),
		LessonID:                               strings.TrimSpace(body.LessonID),
		LabID:                                  strings.TrimSpace(body.LabID),
		ExamID:                                 strings.TrimSpace(body.ExamID),
		AllowSkillPromotion:                    body.AllowSkillPromotion,
		RequireOperatorReviewForSkillPromotion: body.RequireOperatorReviewForSkillPromotion,
		CorrelationID:                          strings.TrimSpace(body.CorrelationID),
		TraceID:                                strings.TrimSpace(body.TraceID),
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	resp := map[string]any{
		"evidenceClass":           "non_canonical_evidence",
		"report":                  report,
		"persisted":               false,
		"nonCanonicalEvidence":    true,
		"canonicalWriteCommitted": false,
	}
	if body.PersistReport {
		id, err := s.dream.PersistReport(r.Context(), dream.PersistReportRequest{
			Report:     report,
			ProposedBy: strings.TrimSpace(body.ProposedBy),
			Metadata:   body.Metadata,
		})
		if err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
		resp["persisted"] = true
		resp["reportId"] = id
		_ = s.log.Emit(r.Context(), "dream.report.persisted", map[string]any{
			"reportId":      id,
			"workspaceId":   report.Run.WorkspaceID,
			"laneId":        report.Run.LaneID,
			"mode":          report.Run.Mode,
			"dryRun":        report.Run.DryRun,
			"correlationId": report.Run.CorrelationID,
			"traceId":       report.Run.TraceID,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDreamReportGet(w http.ResponseWriter, r *http.Request) {
	if s.dream == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "request_failed", "dream service unavailable", nil)
		return
	}
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId"))
	if workspaceID == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "workspaceId required", nil)
		return
	}
	rec, err := s.dream.GetReport(r.Context(), chi.URLParam(r, "id"), workspaceID, strings.TrimSpace(r.URL.Query().Get("laneId")))
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "request_failed", "dream report not found", nil)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) handleDreamReportCandidates(w http.ResponseWriter, r *http.Request) {
	rec, ok := s.loadScopedDreamReport(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"reportId":                rec.ID,
		"candidates":              rec.Candidates,
		"salienceScores":          rec.SalienceScores,
		"evidenceClass":           "non_canonical_evidence",
		"nonCanonicalEvidence":    true,
		"dryRun":                  rec.DryRun,
		"canonicalWriteCommitted": false,
	})
}

func (s *Server) handleDreamReportProposals(w http.ResponseWriter, r *http.Request) {
	rec, ok := s.loadScopedDreamReport(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"reportId":                 rec.ID,
		"memoryTierProposals":      rec.MemoryTierProposals,
		"repairProposals":          rec.RepairProposals,
		"snapshotHygieneProposals": rec.SnapshotHygieneProposals,
		"reviewItems":              rec.ReviewItems(),
		"evidenceClass":            "non_canonical_evidence",
		"nonCanonicalEvidence":     true,
		"dryRun":                   rec.DryRun,
		"canonicalWriteCommitted":  false,
	})
}

func (s *Server) handleDreamReportWarnings(w http.ResponseWriter, r *http.Request) {
	rec, ok := s.loadScopedDreamReport(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"reportId":                rec.ID,
		"warnings":                rec.Warnings,
		"reviewItems":             rec.ReviewItems(),
		"evidenceClass":           "non_canonical_evidence",
		"nonCanonicalEvidence":    true,
		"dryRun":                  rec.DryRun,
		"canonicalWriteCommitted": false,
	})
}

func (s *Server) handleDreamReportsList(w http.ResponseWriter, r *http.Request) {
	if s.dream == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "request_failed", "dream service unavailable", nil)
		return
	}
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId"))
	if workspaceID == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "workspaceId required", nil)
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	reports, err := s.dream.ListReports(r.Context(), dream.ListReportsRequest{
		WorkspaceID: workspaceID,
		LaneID:      strings.TrimSpace(r.URL.Query().Get("laneId")),
		Mode:        dream.Mode(strings.TrimSpace(r.URL.Query().Get("mode"))),
		Limit:       limit,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": reports})
}

func (s *Server) loadScopedDreamReport(w http.ResponseWriter, r *http.Request) (dream.ReportRecord, bool) {
	if s.dream == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "request_failed", "dream service unavailable", nil)
		return dream.ReportRecord{}, false
	}
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId"))
	if workspaceID == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "workspaceId required", nil)
		return dream.ReportRecord{}, false
	}
	rec, err := s.dream.GetReport(r.Context(), chi.URLParam(r, "id"), workspaceID, strings.TrimSpace(r.URL.Query().Get("laneId")))
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "request_failed", "dream report not found", nil)
		return dream.ReportRecord{}, false
	}
	return rec, true
}

func decodeDreamRunJSONBody(r *http.Request, target any) error {
	if r.Body == nil {
		return io.EOF
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, dreamRunRequestBodyLimit+1))
	if err != nil {
		return err
	}
	if len(raw) > dreamRunRequestBodyLimit {
		return errDreamRunRequestBodyTooLarge
	}
	return json.Unmarshal(raw, target)
}

func writeDreamRunDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errDreamRunRequestBodyTooLarge) {
		writeAPIError(w, http.StatusRequestEntityTooLarge, "request_failed", "dream run request body too large", nil)
		return
	}
	writeAPIError(w, http.StatusBadRequest, "request_failed", "invalid json", nil)
}
