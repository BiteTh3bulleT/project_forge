package api

import (
	"database/sql"
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
	"forge/projectforge/services/core/internal/memory"
	"forge/projectforge/services/core/internal/memory/vsaprojection"
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
		writeAPIError(w, http.StatusBadRequest, "request_failed", "bad id", nil)
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
		writeAPIError(w, http.StatusBadRequest, "request_failed", "bad id", nil)
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
		writeAPIError(w, http.StatusBadRequest, "request_failed", "bad id", nil)
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
		writeAPIError(w, http.StatusBadRequest, "request_failed", "bad id", nil)
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
	_ = id
	_ = body
	writeAPIError(w, http.StatusConflict, "memory_usefulness_requires_forge_k", "observation usefulness is immutable utility evidence and must be submitted through the governed FORGE-K utility syscall", nil)
}

func (s *Server) handleGetRetrievalSelection(w http.ResponseWriter, r *http.Request) {
	runID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "bad id", nil)
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
		writeAPIError(w, http.StatusBadRequest, "request_failed", "bad id", nil)
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
		writeAPIError(w, http.StatusBadRequest, "request_failed", "bad id", nil)
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
		writeAPIError(w, http.StatusBadRequest, "request_failed", "bad id", nil)
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
		writeAPIError(w, http.StatusBadRequest, "request_failed", "bad id", nil)
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
		DryRun     *bool         `json:"dryRun"`
	}
	if err := decodeOptionalMemoryJSONBody(r, &body); err != nil {
		writeMemoryDecodeError(w, err)
		return
	}
	if body.DryRun == nil || !*body.DryRun {
		writeAPIError(w, http.StatusConflict, "memory_maintenance_proposal_only", "memory repair is proposal-only until FORGE-K owns the governed evidence revision commit; retry with dryRun=true", nil)
		return
	}
	report, err := s.memory.PreviewRepairPass(r.Context(), memory.RunRepairRequest{
		DossierID:  body.DossierID.Value,
		Mode:       "manual_preview",
		MaxAgeDays: body.MaxAgeDays,
		Limit:      body.Limit,
		Note:       body.Note,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"report": report})
}

func (s *Server) handleRunVSAReindex(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DossierID                 optionalInt64 `json:"dossierId"`
		Mode                      string        `json:"mode"`
		Limit                     int           `json:"limit"`
		TriggeredBy               string        `json:"triggeredBy"`
		Reason                    string        `json:"reason"`
		Note                      string        `json:"note"`
		StaleOnly                 bool          `json:"staleOnly"`
		Force                     bool          `json:"force"`
		DryRun                    *bool         `json:"dryRun"`
		WorkspaceID               string        `json:"workspaceId"`
		LaneID                    string        `json:"laneId"`
		IdempotencyKey            string        `json:"idempotencyKey"`
		ExpectedManifestHash      string        `json:"expectedManifestHash"`
		ExpectedPriorManifestHash *string       `json:"expectedPriorManifestHash"`
		Dimensions                int           `json:"dimensions"`
		Seed                      int           `json:"seed"`
	}
	if err := decodeOptionalMemoryJSONBody(r, &body); err != nil {
		writeMemoryDecodeError(w, err)
		return
	}
	if strings.TrimSpace(body.Mode) == "" {
		body.Mode = "manual_preview"
	}
	if body.DryRun == nil {
		writeAPIError(w, http.StatusConflict, "memory_acceleration_explicit_mode_required", "VSA projection requests must explicitly select dryRun=true for legacy inspection or dryRun=false for governed FORGE-K rebuild", nil)
		return
	}
	if !*body.DryRun {
		s.handleGovernedMemoryAccelerationRebuild(w, r, body.WorkspaceID, body.LaneID, body.IdempotencyKey, body.ExpectedManifestHash, body.ExpectedPriorManifestHash, body.Dimensions, body.Seed)
		return
	}
	report, err := s.memory.PreviewVSAReindex(r.Context(), memory.RunVSAReindexRequest{
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
	response := map[string]any{"report": report}
	if strings.TrimSpace(body.WorkspaceID) != "" || strings.TrimSpace(body.LaneID) != "" {
		if strings.TrimSpace(body.WorkspaceID) == "" || strings.TrimSpace(body.LaneID) == "" {
			writeAPIError(w, http.StatusBadRequest, "memory_acceleration_scope_required", "both workspaceId and laneId are required for a governed manifest proposal", nil)
			return
		}
		dimensions, seed := body.Dimensions, body.Seed
		if dimensions == 0 {
			dimensions = vsaprojection.DefaultDims
		}
		if seed == 0 {
			seed = int(vsaprojection.DefaultSeed)
		}
		scope := vsaprojection.Scope{WorkspaceID: strings.TrimSpace(body.WorkspaceID), LaneID: strings.TrimSpace(body.LaneID)}
		planner := controllane.NewSQLiteSemanticStore(s.st.DB)
		projection, planErr := planner.PlanMemoryAcceleration(r.Context(), scope, vsaprojection.Algorithm{
			Name: vsaprojection.AlgorithmName, Version: vsaprojection.AlgorithmVersion, Dimensions: dimensions, Seed: uint64(seed),
		})
		if planErr != nil {
			writeAPIError(w, http.StatusConflict, "memory_acceleration_source_set_unavailable", planErr.Error(), planErr)
			return
		}
		head, _, headErr := planner.MemoryAccelerationHead(r.Context(), scope)
		if headErr != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, headErr)
			return
		}
		response["manifest"] = projection.Manifest
		response["expectedPriorManifestHash"] = head
		response["governedRebuildReady"] = true
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGovernedMemoryAccelerationRebuild(w http.ResponseWriter, r *http.Request, workspaceID, laneID, idempotencyKey, expectedManifestHash string, expectedPriorManifestHash *string, dimensions, seed int) {
	workspaceID, laneID = strings.TrimSpace(workspaceID), strings.TrimSpace(laneID)
	idempotencyKey, expectedManifestHash = strings.TrimSpace(idempotencyKey), strings.TrimSpace(expectedManifestHash)
	if workspaceID == "" || laneID == "" || idempotencyKey == "" || expectedManifestHash == "" || expectedPriorManifestHash == nil {
		writeAPIError(w, http.StatusBadRequest, "memory_acceleration_identity_required", "workspaceId, laneId, idempotencyKey, expectedManifestHash, and expectedPriorManifestHash are required", nil)
		return
	}
	if dimensions == 0 {
		dimensions = vsaprojection.DefaultDims
	}
	if seed == 0 {
		seed = int(vsaprojection.DefaultSeed)
	}
	scope := vsaprojection.Scope{WorkspaceID: workspaceID, LaneID: laneID}
	algorithm := vsaprojection.Algorithm{Name: vsaprojection.AlgorithmName, Version: vsaprojection.AlgorithmVersion, Dimensions: dimensions, Seed: uint64(seed)}
	planner := controllane.NewSQLiteSemanticStore(s.st.DB)
	projection, err := planner.PlanMemoryAcceleration(r.Context(), scope, algorithm)
	if err != nil {
		writeAPIError(w, http.StatusConflict, "memory_acceleration_source_set_unavailable", err.Error(), nil)
		return
	}
	if projection.Manifest.ManifestHash != expectedManifestHash {
		writeAPIError(w, http.StatusConflict, "memory_acceleration_manifest_diverged", "expectedManifestHash no longer matches the exact governed source set", nil)
		return
	}
	currentHead, exists, err := planner.MemoryAccelerationHead(r.Context(), scope)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	expectedPrior := strings.TrimSpace(*expectedPriorManifestHash)
	if (!exists && expectedPrior != "") || (exists && expectedPrior != currentHead) {
		writeAPIError(w, http.StatusConflict, "memory_acceleration_head_diverged", "expectedPriorManifestHash no longer matches the exact scoped active head", nil)
		return
	}
	if s.kernelAuthority.Processor == nil || !s.kernelAuthorizationReady {
		writeAPIError(w, http.StatusServiceUnavailable, "FORGE_K_MEMORY_ACCELERATION_UNAVAILABLE", "production FORGE-K memory acceleration authority unavailable", nil)
		return
	}
	actor := authenticatedActorName(r)
	actorSource := authenticatedActorSource(r)
	requestID := utilitySyscallRequestID("memory-acceleration", idempotencyKey, workspaceID+"/"+laneID)
	requestedAt := time.Now().UnixMilli()
	correlationID, traceID := requestID+":correlation", requestID+":trace"
	result, err := s.kernelAuthority.Processor.Process(r.Context(), domain.SyscallRequest{
		ID: requestID, Action: domain.ActionRebuildMemoryAcceleration,
		Actor: domain.ActorIdentity{ID: actor, Kind: "user"}, Source: domain.SourceUser,
		Scope: domain.ForgeScope{WorkspaceID: workspaceID, LaneID: laneID},
		Payload: map[string]any{
			"algorithmName": vsaprojection.AlgorithmName, "algorithmVersion": vsaprojection.AlgorithmVersion,
			"dimensions": dimensions, "seed": seed, "expectedManifestHash": expectedManifestHash,
			"expectedPriorManifestHash": expectedPrior, "requestedAtMs": requestedAt,
		},
		Provenance:    domain.Provenance{Actor: actor, ActorType: "user", Source: actorSource, TraceID: traceID},
		CorrelationID: correlationID, TraceID: traceID, IdempotencyKey: idempotencyKey,
		RequestedAt: requestedAt, RequiredCapability: controllane.CapMemoryAccelerationRebuild,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	if !result.Success {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"result": result})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": result, "manifest": projection.Manifest, "projectionOnly": true})
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
		writeAPIError(w, http.StatusRequestEntityTooLarge, "request_failed", "memory request body too large", nil)
		return
	}
	writeAPIError(w, http.StatusBadRequest, "request_failed", "invalid json", nil)
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
		writeAPIError(w, http.StatusBadRequest, "request_failed", "bad id", nil)
		return
	}
	detail, err := s.memory.GetVSAReindexRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAPIError(w, http.StatusNotFound, "request_failed", "not found", nil)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detail": detail})
}
