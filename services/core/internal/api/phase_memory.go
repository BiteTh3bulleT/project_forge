package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/memory"
)

const memoryMutationRequestBodyLimit = 1 << 20

var errMemoryRequestBodyTooLarge = errors.New("memory request body too large")

func (s *Server) handleListMemoryObservations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	dossierID := parseOptionalInt(r.URL.Query().Get("dossierId"))
	obsType := strings.TrimSpace(r.URL.Query().Get("type"))
	originKind := strings.TrimSpace(r.URL.Query().Get("originKind"))
	staleOnly := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("staleOnly"))) == "true"
	rows, err := s.memory.ListObservations(r.Context(), memory.ListObservationsRequest{
		Limit:      limit,
		DossierID:  dossierID,
		Type:       obsType,
		OriginKind: originKind,
		StaleOnly:  staleOnly,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"observations": rows})
}

func (s *Server) handleCreateMemoryObservation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type              string        `json:"type"`
		RawContent        string        `json:"rawContent"`
		Summary           string        `json:"summary"`
		EmbeddingRef      string        `json:"embeddingRef"`
		DossierID         optionalInt64 `json:"dossierId"`
		ProjectKey        string        `json:"projectKey"`
		SourcePath        string        `json:"sourcePath"`
		Entities          []string      `json:"entities"`
		Tags              []string      `json:"tags"`
		RelatedFiles      []string      `json:"relatedFiles"`
		TaskType          string        `json:"taskType"`
		Confidence        float64       `json:"confidence"`
		VerificationState string        `json:"verificationState"`
		Lineage           []string      `json:"lineage"`
		OriginKind        string        `json:"originKind"`
		OriginID          string        `json:"originId"`
		ObservedAtMs      int64         `json:"observedAtMs"`
	}
	if err := decodeMemoryJSONBody(r, &body); err != nil {
		writeMemoryDecodeError(w, err)
		return
	}
	obs, err := s.memory.RecordObservation(r.Context(), memory.RecordObservationRequest{
		Type:              body.Type,
		RawContent:        body.RawContent,
		Summary:           body.Summary,
		EmbeddingRef:      body.EmbeddingRef,
		DossierID:         body.DossierID.Value,
		ProjectKey:        body.ProjectKey,
		SourcePath:        body.SourcePath,
		Entities:          body.Entities,
		Tags:              body.Tags,
		RelatedFiles:      body.RelatedFiles,
		TaskType:          body.TaskType,
		Confidence:        body.Confidence,
		VerificationState: body.VerificationState,
		Lineage:           body.Lineage,
		OriginKind:        body.OriginKind,
		OriginID:          body.OriginID,
		ObservedAtMs:      body.ObservedAtMs,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"observation": obs})
}

func (s *Server) handleGetMemoryObservation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	obs, err := s.memory.GetObservation(r.Context(), id)
	if err != nil {
		writeAPIRequestError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"observation": obs})
}

func (s *Server) handleGetObservationVSA(w http.ResponseWriter, r *http.Request) {
	observationID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	detail, err := s.memory.GetObservationVSA(r.Context(), observationID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detail": detail})
}

func (s *Server) handlePatchMemoryObservation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Summary           *string  `json:"summary"`
		VerificationState *string  `json:"verificationState"`
		Stale             *bool    `json:"stale"`
		LastVerifiedAtMs  *int64   `json:"lastVerifiedAtMs"`
		Tags              []string `json:"tags"`
		RelatedFiles      []string `json:"relatedFiles"`
	}
	if err := decodeMemoryJSONBody(r, &body); err != nil {
		writeMemoryDecodeError(w, err)
		return
	}
	updated, err := s.memory.UpdateObservation(r.Context(), id, memory.UpdateObservationRequest{
		Summary:           body.Summary,
		VerificationState: body.VerificationState,
		Stale:             body.Stale,
		LastVerifiedAtMs:  body.LastVerifiedAtMs,
		Tags:              body.Tags,
		RelatedFiles:      body.RelatedFiles,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"observation": updated})
}

func (s *Server) handleMarkMemoryObservationUsefulness(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Signal            string        `json:"signal"`
		Weight            float64       `json:"weight"`
		Note              string        `json:"note"`
		RetrievalResultID optionalInt64 `json:"retrievalResultId"`
		RetrievalRunID    optionalInt64 `json:"retrievalRunId"`
		PacketID          optionalInt64 `json:"packetId"`
		JobID             *string       `json:"jobId"`
	}
	if err := decodeMemoryJSONBody(r, &body); err != nil {
		writeMemoryDecodeError(w, err)
		return
	}
	if err := s.memory.MarkObservationUsefulness(r.Context(), memory.MarkUsefulnessRequest{
		ObservationID:     id,
		RetrievalResultID: body.RetrievalResultID.Value,
		RetrievalRunID:    body.RetrievalRunID.Value,
		PacketID:          body.PacketID.Value,
		JobID:             body.JobID,
		Signal:            body.Signal,
		Weight:            body.Weight,
		Note:              body.Note,
	}); err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "observationId": id})
}

func (s *Server) handleGetRetrievalSelection(w http.ResponseWriter, r *http.Request) {
	runID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	selection, err := s.memory.SelectionByRun(r.Context(), runID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"selection": selection})
}

func (s *Server) handleGetPacketAlignmentNotes(w http.ResponseWriter, r *http.Request) {
	packetID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	notes, err := s.memory.PacketAlignmentNotes(r.Context(), packetID, limit)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notes": notes})
}

func (s *Server) handleGetDossierMemory(w http.ResponseWriter, r *http.Request) {
	dossierID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	view, err := s.memory.DossierView(r.Context(), dossierID, limit)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"view": view})
}

func (s *Server) handleGetDossierVSASummary(w http.ResponseWriter, r *http.Request) {
	dossierID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	summary, err := s.memory.DossierVSASummary(r.Context(), dossierID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"summary": summary})
}

func (s *Server) handleListMemoryRepairRuns(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	dossierID := parseOptionalInt(r.URL.Query().Get("dossierId"))
	runs, err := s.memory.ListRepairRuns(r.Context(), limit, dossierID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (s *Server) handleGetMemoryRepairRun(w http.ResponseWriter, r *http.Request) {
	runID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	detail, err := s.memory.GetRepairRun(r.Context(), runID)
	if err != nil {
		writeAPIRequestError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detail": detail})
}

func (s *Server) handleRunMemoryRepair(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DossierID  optionalInt64 `json:"dossierId"`
		MaxAgeDays int           `json:"maxAgeDays"`
		Limit      int           `json:"limit"`
		Note       string        `json:"note"`
	}
	if err := decodeOptionalMemoryJSONBody(r, &body); err != nil {
		writeMemoryDecodeError(w, err)
		return
	}
	detail, err := s.memory.RunRepairPass(r.Context(), memory.RunRepairRequest{
		DossierID:  body.DossierID.Value,
		Mode:       "manual",
		MaxAgeDays: body.MaxAgeDays,
		Limit:      body.Limit,
		Note:       body.Note,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detail": detail})
}

func (s *Server) handleRunVSAReindex(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DossierID   optionalInt64 `json:"dossierId"`
		Mode        string        `json:"mode"`
		Limit       int           `json:"limit"`
		TriggeredBy string        `json:"triggeredBy"`
		Reason      string        `json:"reason"`
		Note        string        `json:"note"`
		StaleOnly   bool          `json:"staleOnly"`
		Force       bool          `json:"force"`
	}
	if err := decodeOptionalMemoryJSONBody(r, &body); err != nil {
		writeMemoryDecodeError(w, err)
		return
	}
	if strings.TrimSpace(body.Mode) == "" {
		body.Mode = "manual"
	}
	detail, err := s.memory.RunVSAReindex(r.Context(), memory.RunVSAReindexRequest{
		DossierID:   body.DossierID.Value,
		Mode:        strings.TrimSpace(body.Mode),
		Limit:       body.Limit,
		TriggeredBy: strings.TrimSpace(body.TriggeredBy),
		Reason:      strings.TrimSpace(body.Reason),
		Note:        strings.TrimSpace(body.Note),
		StaleOnly:   body.StaleOnly,
		Force:       body.Force,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detail": detail})
}

func decodeMemoryJSONBody(r *http.Request, target any) error {
	raw, err := readMemoryRequestBody(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func decodeOptionalMemoryJSONBody(r *http.Request, target any) error {
	raw, err := readMemoryRequestBody(r)
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

func readMemoryRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, io.EOF
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, memoryMutationRequestBodyLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > memoryMutationRequestBodyLimit {
		return nil, errMemoryRequestBodyTooLarge
	}
	return raw, nil
}

func writeMemoryDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errMemoryRequestBodyTooLarge) {
		http.Error(w, "memory request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "invalid json", http.StatusBadRequest)
}

func (s *Server) handleListVSAReindexRuns(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	dossierID := parseOptionalInt(r.URL.Query().Get("dossierId"))
	runs, err := s.memory.ListVSAReindexRuns(r.Context(), limit, dossierID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (s *Server) handleGetVSAReindexRun(w http.ResponseWriter, r *http.Request) {
	runID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	detail, err := s.memory.GetVSAReindexRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detail": detail})
}
