package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/audit"
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
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "invalid json", http.StatusBadRequest)
}

func (s *Server) handleRestoreOutcomeList(w http.ResponseWriter, r *http.Request) {
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId"))
	if workspaceID == "" {
		http.Error(w, "workspaceId required", http.StatusBadRequest)
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
		http.Error(w, "workspaceId required", http.StatusBadRequest)
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
		http.Error(w, "restore outcome not found", http.StatusNotFound)
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
		http.Error(w, "workspaceId required", http.StatusBadRequest)
		return
	}
	outcome := controllane.RestoreOutcome(strings.TrimSpace(body.Outcome))
	if !controllane.ValidateRestoreOutcome(outcome) || outcome == controllane.RestoreOutcomeUnknown {
		http.Error(w, "valid non-unknown outcome required", http.StatusBadRequest)
		return
	}
	laneID := strings.TrimSpace(firstNonEmptyTrimmed(body.LaneID, r.URL.Query().Get("laneId")))
	now := time.Now().UnixMilli()
	if body.OutcomeConfidence == 0 {
		body.OutcomeConfidence = 1
	}
	store := controllane.NewSQLiteSemanticStore(s.st.DB)
	event, err := store.UpdateRestoreOutcomeFeedback(r.Context(), chi.URLParam(r, "id"), domain.ForgeScope{WorkspaceID: workspaceID, LaneID: laneID}, controllane.RestoreOutcomeFeedback{
		Outcome:           outcome,
		OutcomeConfidence: body.OutcomeConfidence,
		OperatorFeedback:  body.OperatorFeedback,
		CorrectionSummary: body.CorrectionSummary,
		Metadata:          body.Metadata,
		CorrelationID:     body.CorrelationID,
		TraceID:           body.TraceID,
		UpdatedBy:         body.UpdatedBy,
		UpdatedAt:         now,
	})
	if err != nil {
		if strings.Contains(err.Error(), "outside requested scope") || strings.Contains(err.Error(), "not found") {
			http.Error(w, "restore outcome not found", http.StatusNotFound)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	meta := requestAuditMetaForBackup(r, body.CorrelationID, body.TraceID, workspaceID, "restore.outcome.feedback")
	_, _ = s.auditSvc.Record(r.Context(), audit.CreateRequest{
		CorrelationID: meta.CorrelationID,
		Category:      "context_restore",
		Action:        "restore_outcome.feedback_updated",
		SubjectType:   "restore_outcome_event",
		SubjectID:     event.ID,
		Outcome:       "ok",
		Summary:       "restore outcome feedback updated",
		Payload: requestAuditPayload(map[string]any{
			"workspaceId":          workspaceID,
			"laneId":               laneID,
			"outcome":              event.Outcome,
			"outcomeConfidence":    event.OutcomeConfidence,
			"nonCanonicalEvidence": true,
			"requestPath":          r.URL.Path,
		}, meta),
	})
	_ = s.log.Emit(r.Context(), "context.restore.outcome.feedback", map[string]any{
		"id":          event.ID,
		"workspaceId": workspaceID,
		"laneId":      laneID,
		"outcome":     event.Outcome,
	})
	writeJSON(w, http.StatusOK, map[string]any{"outcome": event})
}

func restoreOutcomeInAPIScope(event controllane.RestoreOutcomeEvent, workspaceID, laneID string) bool {
	if strings.TrimSpace(event.WorkspaceID) != strings.TrimSpace(workspaceID) {
		return false
	}
	laneID = strings.TrimSpace(laneID)
	return laneID == "" || strings.TrimSpace(event.LaneID) == "" || strings.TrimSpace(event.LaneID) == laneID
}
