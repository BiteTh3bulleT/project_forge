package api

import (
	"net/http"
	"strings"

	"forge/projectforge/services/core/internal/jobs"
)

type cmdBody struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

func (s *Server) handleCommandExecute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body cmdBody
	if err := decodeServerJSONBody(r, &body); err != nil {
		writeServerDecodeError(w, err)
		return
	}
	name := strings.TrimSpace(strings.ToLower(body.Name))
	switch name {
	case "reindex":
		job, err := s.jobs.Create(ctx, jobs.CreateRequest{
			TemplateID:       "reindex_sources",
			Title:            "Re-index sources",
			UserRequest:      "Re-index all configured sources",
			Objective:        "Refresh indexed memory from current source folders.",
			InitiatingSource: "command_bar",
			RequestPayload:   body.Args,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = s.log.Emit(ctx, "command.executed", map[string]any{"command": "reindex", "jobId": job.ID})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "jobId": job.ID})
		return
	case "search_packet":
		s.createTemplateJob(w, r, "search_packet", "Search memory packet", "Build a packet from retrieved local memory context.", body.Args)
		return
	case "ollama_summary":
		s.createTemplateJob(w, r, "ollama_summary", "Ollama summary", "Summarize relevant retrieved context.", body.Args)
		return
	case "plan_from_index":
		s.createTemplateJob(w, r, "plan_from_index", "Plan from index", "Draft implementation plan from indexed context.", body.Args)
		return
	case "prepare_codex_handoff":
		s.createTemplateJob(w, r, "prepare_codex_handoff", "Prepare Codex handoff", "Prepare bounded Codex handoff packet.", body.Args)
		return
	case "prepare_claude_handoff":
		s.createTemplateJob(w, r, "prepare_claude_handoff", "Prepare Claude Code handoff", "Prepare bounded Claude Code handoff packet.", body.Args)
		return
	case "safe_local_analysis":
		s.createTemplateJob(w, r, "safe_local_analysis", "Safe local analysis", "Run read-only local analysis.", body.Args)
		return
	case "normalize_project_context":
		s.createTemplateJob(w, r, "normalize_project_context", "Normalize project context", "Import context and regenerate guidance artifacts.", body.Args)
		return
	case "navigate":
		_ = s.log.Emit(ctx, "command.executed", map[string]any{"command": "navigate", "args": body.Args})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "note": "Navigation is handled client-side."})
		return
	default:
		_ = s.log.Emit(ctx, "command.executed", map[string]any{"command": body.Name})
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "note": "Unknown command. Use a known command template."})
	}
}

func (s *Server) createTemplateJob(w http.ResponseWriter, r *http.Request, templateID, title, objective string, args map[string]any) {
	ctx := r.Context()
	userRequest, _ := args["query"].(string)
	if strings.TrimSpace(userRequest) == "" {
		userRequest = title
	}
	job, err := s.jobs.Create(ctx, jobs.CreateRequest{
		TemplateID:       templateID,
		Title:            title,
		UserRequest:      userRequest,
		Objective:        objective,
		Query:            userRequest,
		InitiatingSource: "command_bar",
		RequestPayload:   args,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "jobId": job.ID})
}
