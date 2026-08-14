package api

import (
	"crypto/sha256"
	"encoding/hex"
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
	SelectedPaths     []string       `json:"selectedPaths"`
	IdempotencyKey    string         `json:"idempotencyKey"`
}

const restoreOutcomeFeedbackRequestBodyLimit int64 = 1 << 20

var errRestoreOutcomeFeedbackRequestBodyTooLarge = errors.New("restore outcome feedback request body too large")

type restoreOutcomeReadModel struct {
	Outcome            controllane.RestoreOutcomeEvent               `json:"outcome"`
	FeedbackProjection *controllane.RestoreOutcomeFeedbackProjection `json:"feedbackProjection"`
}

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
	readModels := make([]restoreOutcomeReadModel, 0, len(events))
	for _, event := range events {
		model := restoreOutcomeReadModel{Outcome: event}
		projection, found, projectionErr := store.GetRestoreOutcomeFeedbackProjection(r.Context(), event.ID, domain.ForgeScope{
			WorkspaceID: event.WorkspaceID,
			LaneID:      event.LaneID,
		})
		if projectionErr != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, projectionErr)
			return
		}
		if found {
			model.FeedbackProjection = &projection
		}
		readModels = append(readModels, model)
	}
	writeJSON(w, http.StatusOK, map[string]any{"outcomes": readModels})
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
	model := restoreOutcomeReadModel{Outcome: event}
	projection, found, err := store.GetRestoreOutcomeFeedbackProjection(r.Context(), event.ID, domain.ForgeScope{
		WorkspaceID: event.WorkspaceID,
		LaneID:      event.LaneID,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	if found {
		model.FeedbackProjection = &projection
	}
	writeJSON(w, http.StatusOK, model)
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
	laneID := strings.TrimSpace(firstNonEmptyTrimmed(body.LaneID, r.URL.Query().Get("laneId")))
	if laneID == "" || len(body.SelectedPaths) == 0 {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "laneId and selectedPaths are required", nil)
		return
	}
	idempotencyKey := strings.TrimSpace(body.IdempotencyKey)
	if idempotencyKey == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "idempotencyKey required", nil)
		return
	}
	if s.kernelAuthority.Processor == nil || !s.kernelAuthorizationReady {
		writeAPIError(w, http.StatusServiceUnavailable, "FORGE_K_UTILITY_EVIDENCE_UNAVAILABLE", "production FORGE-K utility evidence authority unavailable", nil)
		return
	}
	actor := authenticatedActorName(r)
	actorSource := authenticatedActorSource(r)
	requestID := utilitySyscallRequestID("restore-outcome-feedback", idempotencyKey, chi.URLParam(r, "id"))
	correlationID := strings.TrimSpace(body.CorrelationID)
	if correlationID == "" {
		correlationID = requestID + ":correlation"
	}
	traceID := strings.TrimSpace(body.TraceID)
	if traceID == "" {
		traceID = requestID + ":trace"
	}
	result, err := s.kernelAuthority.Processor.Process(r.Context(), domain.SyscallRequest{
		ID: requestID, Action: domain.ActionRecordRestoreOutcomeFeedback,
		Actor: domain.ActorIdentity{ID: actor, Kind: "user"}, Source: domain.SourceUser,
		Scope: domain.ForgeScope{WorkspaceID: workspaceID, LaneID: laneID, SelectedPaths: append([]string(nil), body.SelectedPaths...)},
		Payload: map[string]any{
			"restoreOutcomeId": chi.URLParam(r, "id"), "outcome": string(outcome),
			"outcomeConfidence": body.OutcomeConfidence, "operatorFeedback": body.OperatorFeedback,
			"correctionSummary": body.CorrectionSummary, "metadata": body.Metadata,
		},
		Provenance:    domain.Provenance{Actor: actor, ActorType: "user", Source: actorSource, TraceID: traceID},
		CorrelationID: correlationID, TraceID: traceID, IdempotencyKey: idempotencyKey,
		RequestedAt: time.Now().UnixMilli(), RequiredCapability: "context.restore.outcome.feedback.record",
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	if !result.Success {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"result": result})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": result, "originalEvidencePreserved": true})
}

func utilitySyscallRequestID(kind, idempotencyKey, target string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(kind) + "\x00" + strings.TrimSpace(idempotencyKey) + "\x00" + strings.TrimSpace(target)))
	return "utility-" + strings.TrimSpace(kind) + "-" + hex.EncodeToString(sum[:12])
}

func restoreOutcomeInAPIScope(event controllane.RestoreOutcomeEvent, workspaceID, laneID string) bool {
	if strings.TrimSpace(event.WorkspaceID) != strings.TrimSpace(workspaceID) {
		return false
	}
	laneID = strings.TrimSpace(laneID)
	return laneID == "" || strings.TrimSpace(event.LaneID) == "" || strings.TrimSpace(event.LaneID) == laneID
}
