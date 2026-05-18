package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	coreevents "forge/projectforge/services/core/internal/events"
)

func (s *Server) handleAutonomyStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if s == nil || s.autonomy == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"available": false,
			"reason":    "autonomy_loop_not_configured",
		})
		return
	}

	activeIntents, err := s.autonomy.ListIntents(ctx, "active", 200)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	activeCharters, err := s.autonomy.ListCharters(ctx, true)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	budgets, err := s.autonomy.ListBudgets(ctx)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	decisions, err := s.autonomy.ListDecisions(ctx, 200)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"available": true,
		"scope":     s.autonomy.Scope(),
		"mode":      s.autonomy.Mode(),
		"dream":     s.autonomy.Status(),
		"counts": map[string]any{
			"activeIntents":   len(activeIntents),
			"activeCharters":  len(activeCharters),
			"budgets":         len(budgets),
			"recentDecisions": len(decisions),
		},
	})
}

func (s *Server) handleAutonomyIntents(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.autonomy == nil {
		writeJSON(w, http.StatusOK, map[string]any{"intents": []any{}})
		return
	}
	limit := parseAutonomyLimit(r, 100)
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	rows, err := s.autonomy.ListIntents(r.Context(), status, limit)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"intents": rows})
}

func (s *Server) handleAutonomyIntentExplain(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.autonomy == nil {
		http.Error(w, "autonomy loop is not configured", http.StatusNotFound)
		return
	}
	intentID := strings.TrimSpace(chi.URLParam(r, "id"))
	if intentID == "" {
		http.Error(w, "intent id is required", http.StatusBadRequest)
		return
	}
	explanation, err := s.autonomy.ExplainIntent(r.Context(), intentID)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, explanation)
}

func (s *Server) handleAutonomyDecisions(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.autonomy == nil {
		writeJSON(w, http.StatusOK, map[string]any{"decisions": []any{}})
		return
	}
	limit := parseAutonomyLimit(r, 100)
	rows, err := s.autonomy.ListDecisions(r.Context(), limit)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"decisions": rows})
}

func (s *Server) handleAutonomyBudgets(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.autonomy == nil {
		writeJSON(w, http.StatusOK, map[string]any{"budgets": []any{}})
		return
	}
	rows, err := s.autonomy.ListBudgets(r.Context())
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"budgets": rows})
}

func (s *Server) handleAutonomyCharters(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.autonomy == nil {
		writeJSON(w, http.StatusOK, map[string]any{"charters": []any{}})
		return
	}
	activeOnly := parseBoolSetting(r.URL.Query().Get("activeOnly"), false)
	rows, err := s.autonomy.ListCharters(r.Context(), activeOnly)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"charters": rows})
}

func (s *Server) handleAutonomyEvents(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.st == nil || s.st.DB == nil {
		http.Error(w, "database unavailable", http.StatusInternalServerError)
		return
	}
	limit := parseAutonomyLimit(r, 120)
	rows, err := listAutonomyEvents(r.Context(), s.st.DB, limit)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": rows})
}

func (s *Server) handleAutonomyMaintenanceSweep(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.autonomy == nil {
		http.Error(w, "autonomy loop is not configured", http.StatusNotFound)
		return
	}
	var body AutonomyMaintenanceSweepRequest
	if err := decodeOptionalServerJSONBody(r, &body); err != nil {
		writeServerDecodeError(w, err)
		return
	}
	report, err := s.autonomy.RunSweep(r.Context(), body)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	status := http.StatusOK
	if report.Status == "skipped" {
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]any{"report": report})
}

func parseAutonomyLimit(r *http.Request, fallback int) int {
	if fallback <= 0 {
		fallback = 100
	}
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if n <= 0 {
		return fallback
	}
	if n > 500 {
		return 500
	}
	return n
}

func listAutonomyEvents(ctx context.Context, db *sql.DB, limit int) ([]coreevents.Row, error) {
	if limit <= 0 || limit > 500 {
		limit = 120
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, created_at, type, payload_json
FROM events
WHERE type LIKE 'autonomy.%'
ORDER BY id DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]coreevents.Row, 0, limit)
	for rows.Next() {
		var row coreevents.Row
		var payload string
		if err := rows.Scan(&row.ID, &row.CreatedAtMs, &row.Type, &payload); err != nil {
			return nil, err
		}
		row.Payload = json.RawMessage(payload)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
