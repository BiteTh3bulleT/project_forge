package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/aios/dream"
)

type dreamRunRequest struct {
	Mode                             string         `json:"mode"`
	WorkspaceID                      string         `json:"workspaceId"`
	LaneID                           string         `json:"laneId,omitempty"`
	WindowHours                      int            `json:"windowHours,omitempty"`
	MaxCandidates                    int            `json:"maxCandidates,omitempty"`
	DryRun                           *bool          `json:"dryRun,omitempty"`
	AllowLongTermPromotion           bool           `json:"allowLongTermPromotion,omitempty"`
	RequireOperatorReviewForLongTerm bool           `json:"requireOperatorReviewForLongTerm"`
	AllowCommits                     bool           `json:"allowCommits,omitempty"`
	PersistReport                    bool           `json:"persistReport,omitempty"`
	CorrelationID                    string         `json:"correlationId,omitempty"`
	TraceID                          string         `json:"traceId,omitempty"`
	ProposedBy                       string         `json:"proposedBy,omitempty"`
	Metadata                         map[string]any `json:"metadata,omitempty"`
}

func (s *Server) handleDreamRun(w http.ResponseWriter, r *http.Request) {
	if s.dream == nil {
		http.Error(w, "dream service unavailable", http.StatusServiceUnavailable)
		return
	}
	var body dreamRunRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.WorkspaceID) == "" {
		http.Error(w, "workspaceId required", http.StatusBadRequest)
		return
	}
	report, err := s.dream.Run(r.Context(), dream.RunRequest{
		Mode:                             dream.Mode(strings.TrimSpace(body.Mode)),
		WorkspaceID:                      strings.TrimSpace(body.WorkspaceID),
		LaneID:                           strings.TrimSpace(body.LaneID),
		WindowHours:                      body.WindowHours,
		MaxCandidates:                    body.MaxCandidates,
		DryRun:                           body.DryRun,
		AllowLongTermPromotion:           body.AllowLongTermPromotion,
		RequireOperatorReviewForLongTerm: body.RequireOperatorReviewForLongTerm,
		AllowCommits:                     body.AllowCommits,
		CorrelationID:                    strings.TrimSpace(body.CorrelationID),
		TraceID:                          strings.TrimSpace(body.TraceID),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "dream service unavailable", http.StatusServiceUnavailable)
		return
	}
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId"))
	if workspaceID == "" {
		http.Error(w, "workspaceId required", http.StatusBadRequest)
		return
	}
	rec, err := s.dream.GetReport(r.Context(), chi.URLParam(r, "id"), workspaceID, strings.TrimSpace(r.URL.Query().Get("laneId")))
	if err != nil {
		http.Error(w, "dream report not found", http.StatusNotFound)
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
		http.Error(w, "dream service unavailable", http.StatusServiceUnavailable)
		return
	}
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId"))
	if workspaceID == "" {
		http.Error(w, "workspaceId required", http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": reports})
}

func (s *Server) loadScopedDreamReport(w http.ResponseWriter, r *http.Request) (dream.ReportRecord, bool) {
	if s.dream == nil {
		http.Error(w, "dream service unavailable", http.StatusServiceUnavailable)
		return dream.ReportRecord{}, false
	}
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId"))
	if workspaceID == "" {
		http.Error(w, "workspaceId required", http.StatusBadRequest)
		return dream.ReportRecord{}, false
	}
	rec, err := s.dream.GetReport(r.Context(), chi.URLParam(r, "id"), workspaceID, strings.TrimSpace(r.URL.Query().Get("laneId")))
	if err != nil {
		http.Error(w, "dream report not found", http.StatusNotFound)
		return dream.ReportRecord{}, false
	}
	return rec, true
}
