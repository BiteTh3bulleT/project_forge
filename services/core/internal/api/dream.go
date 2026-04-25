package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"forge/projectforge/services/core/internal/aios/dream"
)

type dreamRunRequest struct {
	Mode                             string `json:"mode"`
	WorkspaceID                      string `json:"workspaceId"`
	LaneID                           string `json:"laneId,omitempty"`
	WindowHours                      int    `json:"windowHours,omitempty"`
	MaxCandidates                    int    `json:"maxCandidates,omitempty"`
	DryRun                           *bool  `json:"dryRun,omitempty"`
	AllowLongTermPromotion           bool   `json:"allowLongTermPromotion,omitempty"`
	RequireOperatorReviewForLongTerm bool   `json:"requireOperatorReviewForLongTerm"`
	AllowCommits                     bool   `json:"allowCommits,omitempty"`
	CorrelationID                    string `json:"correlationId,omitempty"`
	TraceID                          string `json:"traceId,omitempty"`
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
	writeJSON(w, http.StatusOK, report)
}
