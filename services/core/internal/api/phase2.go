package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/approvals"
	"forge/projectforge/services/core/internal/jobs"
	"forge/projectforge/services/core/internal/projectcontext"
)

const phase2JSONRequestBodyLimit = 1 << 20

var errPhase2RequestBodyTooLarge = errors.New("phase2 json request body too large")

func (s *Server) handleListJobTemplates(w http.ResponseWriter, r *http.Request) {
	list := s.jobs.ListTemplates()
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	writeJSON(w, http.StatusOK, map[string]any{"templates": list})
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var body jobs.CreateRequest
	if err := decodePhase2JSONBody(r, &body); err != nil {
		writePhase2DecodeError(w, err)
		return
	}
	j, err := s.jobs.Create(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": j})
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := s.jobs.List(r.Context(), status, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": rows})
}

func (s *Server) handleJobDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	afterID, _ := strconv.ParseInt(r.URL.Query().Get("afterEventId"), 10, 64)
	detail, err := s.jobs.Detail(r.Context(), id, afterID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Actor string `json:"actor"`
	}
	if err := decodeOptionalPhase2JSONBody(r, &body); err != nil {
		writePhase2DecodeError(w, err)
		return
	}
	if err := s.jobs.RequestCancel(r.Context(), id, body.Actor); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "jobId": id})
}

func (s *Server) handleListApprovals(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := s.approvals.ListRequests(r.Context(), status, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"approvals": rows})
}

func (s *Server) handleGetApproval(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	req, err := s.approvals.GetRequest(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"approval": req})
}

func (s *Server) handleApproveRequest(w http.ResponseWriter, r *http.Request) {
	s.handleApprovalDecision(w, r, "approved")
}

func (s *Server) handleDenyRequest(w http.ResponseWriter, r *http.Request) {
	s.handleApprovalDecision(w, r, "denied")
}

func (s *Server) handleCancelRequest(w http.ResponseWriter, r *http.Request) {
	s.handleApprovalDecision(w, r, "cancelled")
}

func (s *Server) handleApprovalDecision(w http.ResponseWriter, r *http.Request, decision string) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Actor string `json:"actor"`
		Note  string `json:"note"`
	}
	if err := decodeOptionalPhase2JSONBody(r, &body); err != nil {
		writePhase2DecodeError(w, err)
		return
	}
	if decision == "approved" && s.approvals != nil {
		ar, err := s.approvals.GetRequest(r.Context(), id)
		if err == nil && approvalDecisionRequiresNonPublicAuthority(ar) {
			http.Error(w, "approval request requires a non-public approval authority", http.StatusForbidden)
			return
		}
	}
	d, err := s.jobs.ApplyApprovalDecision(r.Context(), id, decision, body.Actor, body.Note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"decision": d})
}

func approvalDecisionRequiresNonPublicAuthority(req *approvals.Request) bool {
	if req == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(req.RequestedAction), "gateway.capability.status.update") {
		return true
	}
	var scope map[string]any
	if err := json.Unmarshal(req.ScopeSnapshot, &scope); err != nil {
		return false
	}
	for _, key := range []string{"publicDecisionAllowed", "approvalPublicDecisionAllowed"} {
		if allowed, ok := scope[key].(bool); ok && !allowed {
			return true
		}
	}
	return false
}

func (s *Server) handleGetPacket(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	pkt, err := s.packets.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, pkt)
}

func (s *Server) handleGetProjectContext(w http.ResponseWriter, r *http.Request) {
	rec, err := s.projectCtx.Latest(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"record": rec})
}

func (s *Server) handleImportProjectContext(w http.ResponseWriter, r *http.Request) {
	var body projectcontext.ImportRequest
	if err := decodeOptionalPhase2JSONBody(r, &body); err != nil {
		writePhase2DecodeError(w, err)
		return
	}
	rec, err := s.projectCtx.ImportAndNormalize(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"record": rec})
}

func decodePhase2JSONBody(r *http.Request, target any) error {
	raw, err := readPhase2RequestBody(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func decodeOptionalPhase2JSONBody(r *http.Request, target any) error {
	raw, err := readPhase2RequestBody(r)
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

func readPhase2RequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, io.EOF
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, phase2JSONRequestBodyLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > phase2JSONRequestBodyLimit {
		return nil, errPhase2RequestBodyTooLarge
	}
	return raw, nil
}

func writePhase2DecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errPhase2RequestBodyTooLarge) {
		http.Error(w, "phase2 json request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "invalid json", http.StatusBadRequest)
}

func (s *Server) handleRegenerateProjectContext(w http.ResponseWriter, r *http.Request) {
	rec, err := s.projectCtx.ImportAndNormalize(r.Context(), projectcontext.ImportRequest{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"record": rec})
}
