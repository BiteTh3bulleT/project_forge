package api

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/jobs"
	"forge/projectforge/services/core/internal/projectcontext"
)

func (s *Server) handleListJobTemplates(w http.ResponseWriter, r *http.Request) {
	list := s.jobs.ListTemplates()
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	writeJSON(w, http.StatusOK, map[string]any{"templates": list})
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var body jobs.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
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
	_ = json.NewDecoder(r.Body).Decode(&body)
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
	_ = json.NewDecoder(r.Body).Decode(&body)
	d, err := s.jobs.ApplyApprovalDecision(r.Context(), id, decision, body.Actor, body.Note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"decision": d})
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	rec, err := s.projectCtx.ImportAndNormalize(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"record": rec})
}

func (s *Server) handleRegenerateProjectContext(w http.ResponseWriter, r *http.Request) {
	rec, err := s.projectCtx.ImportAndNormalize(r.Context(), projectcontext.ImportRequest{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"record": rec})
}
