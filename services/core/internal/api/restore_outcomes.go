package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/aios/controllane"
)

type restoreOutcomeFeedbackRequest struct {
	WorkspaceID       string         `json:"workspaceId"`
	LaneID            string         `json:"laneId,omitempty"`
	Outcome           string         `json:"outcome"`
	OutcomeConfidence float64        `json:"outcomeConfidence"`
	OperatorFeedback  string         `json:"operatorFeedback,omitempty"`
	CorrectionSummary string         `json:"correctionSummary,omitempty"`
	CorrelationID     string         `json:"correlationId,omitempty"`
	TraceID           string         `json:"traceId,omitempty"`
	UpdatedBy         string         `json:"updatedBy,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

const restoreOutcomeFeedbackRequestBodyLimit int64 = 1 << 20

var errRestoreOutcomeFeedbackRequestBodyTooLarge = errors.New("restore outcome feedback request body too large")

func decodeRestoreOutcomeFeedbackJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, restoreOutcomeFeedbackRequestBodyLimit))
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			return errRestoreOutcomeFeedbackRequestBodyTooLarge
		}
		return err
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return err
	}
	return nil
}

func writeRestoreOutcomeFeedbackDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errRestoreOutcomeFeedbackRequestBodyTooLarge) {
		writeAPIError(w, http.StatusRequestEntityTooLarge, "request_failed", "request body too large", nil)
		return
	}
	writeAPIError(w, http.StatusBadRequest, "request_failed", "invalid json", nil)
}

func (s *Server) handleRestoreOutcomeList(w http.ResponseWriter, r *http.Request) {
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId"))
	if workspaceID == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "workspaceId required", nil)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	store := controllane.NewSQLiteSemanticStore(s.st.DB)
	events, err := store.ListRestoreOutcomes(r.Context(), controllane.RestoreOutcomeFilter{
		WorkspaceID: workspaceID,
		LaneID:      strings.TrimSpace(r.URL.Query().Get("laneId")),
		Query:       strings.TrimSpace(r.URL.Query().Get("query")),
		SnapshotID:  strings.TrimSpace(r.URL.Query().Get("snapshotId")),
		Outcome:     controllane.RestoreOutcome(strings.TrimSpace(r.URL.Query().Get("outcome"))),
		Since:       since,
		Limit:       limit,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"outcomes": events})
}

func (s *Server) handleRestoreOutcomeGet(w http.ResponseWriter, r *http.Request) {
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId"))
	if workspaceID == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "workspaceId required", nil)
		return
	}
	laneID := strings.TrimSpace(r.URL.Query().Get("laneId"))
	store := controllane.NewSQLiteSemanticStore(s.st.DB)
	event, ok, err := store.GetRestoreOutcome(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok || !restoreOutcomeInAPIScope(event, workspaceID, laneID) {
		writeAPIError(w, http.StatusNotFound, "request_failed", "restore outcome not found", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"outcome": event})
}

func (s *Server) handleRestoreOutcomeFeedback(w http.ResponseWriter, r *http.Request) {
	var body restoreOutcomeFeedbackRequest
	if err := decodeRestoreOutcomeFeedbackJSONBody(w, r, &body); err != nil {
		writeRestoreOutcomeFeedbackDecodeError(w, err)
		return
	}
	workspaceID := strings.TrimSpace(firstNonEmptyTrimmed(body.WorkspaceID, r.URL.Query().Get("workspaceId")))
	if workspaceID == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "workspaceId required", nil)
		return
	}
	outcome := controllane.RestoreOutcome(strings.TrimSpace(body.Outcome))
	if !controllane.ValidateRestoreOutcome(outcome) || outcome == controllane.RestoreOutcomeUnknown {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "valid non-unknown outcome required", nil)
		return
	}
	writeAPIError(
		w,
		http.StatusConflict,
		"FORGE_K_RESTORE_OUTCOME_FEEDBACK_DISABLED",
		"restore outcome feedback mutation is disabled until it is routed through a production FORGE-K syscall",
		nil,
	)
}

func restoreOutcomeInAPIScope(event controllane.RestoreOutcomeEvent, workspaceID, laneID string) bool {
	if strings.TrimSpace(event.WorkspaceID) != strings.TrimSpace(workspaceID) {
		return false
	}
	laneID = strings.TrimSpace(laneID)
	return laneID == "" || strings.TrimSpace(event.LaneID) == "" || strings.TrimSpace(event.LaneID) == laneID
}
