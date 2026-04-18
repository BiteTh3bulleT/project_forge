package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/automation"
	"forge/projectforge/services/core/internal/failurepatterns"
	"forge/projectforge/services/core/internal/jobs"
	"forge/projectforge/services/core/internal/packetopt"
	"forge/projectforge/services/core/internal/policy"
	"forge/projectforge/services/core/internal/reconciliation"
	"forge/projectforge/services/core/internal/reviews"
	"forge/projectforge/services/core/internal/strategies"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	summary, err := s.dashboard.Summary(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleListStrategies(w http.ResponseWriter, r *http.Request) {
	enabledOnly := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("enabled"))) == "true"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := s.strategies.List(r.Context(), enabledOnly, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"strategies": list})
}

func (s *Server) handleSaveStrategy(w http.ResponseWriter, r *http.Request) {
	var body strategies.SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	item, err := s.strategies.Save(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"strategy": item})
}

func (s *Server) handleListApprovalPresets(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := s.policy.ListApprovalPresets(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"presets": rows})
}

func (s *Server) handleSaveApprovalPreset(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID          string         `json:"id"`
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Profile     map[string]any `json:"profile"`
		Editable    bool           `json:"editable"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	item, err := s.policy.SaveApprovalPreset(r.Context(), body.ID, body.Name, body.Description, body.Profile, body.Editable)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preset": item})
}

func (s *Server) handleGetGlobalPreset(w http.ResponseWriter, r *http.Request) {
	presetID, err := s.policy.GlobalApprovalPreset(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"presetId": presetID})
}

func (s *Server) handleSetGlobalPreset(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PresetID string `json:"presetId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := s.policy.SetGlobalApprovalPreset(r.Context(), body.PresetID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "presetId": body.PresetID})
}

func (s *Server) handleGetDossierProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	item, err := s.policy.DossierProfile(r.Context(), &id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profile": item})
}

func (s *Server) handleSaveDossierProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body policy.SaveDossierProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	body.DossierID = id
	item, err := s.policy.SaveDossierProfile(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profile": item})
}

func (s *Server) handlePolicyRecommend(w http.ResponseWriter, r *http.Request) {
	var body policy.RecommendRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	item, err := s.policy.Recommend(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recommendation": item})
}

func (s *Server) handleListPolicyRecommendations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	dossierID := parseOptionalInt(r.URL.Query().Get("dossierId"))
	rows, err := s.policy.ListRecommendations(r.Context(), limit, dossierID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recommendations": rows})
}

func (s *Server) handleListAutomationRules(w http.ResponseWriter, r *http.Request) {
	enabledOnly := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("enabled"))) == "true"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := s.automation.ListRules(r.Context(), enabledOnly, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rows})
}

func (s *Server) handleSaveAutomationRule(w http.ResponseWriter, r *http.Request) {
	var body automation.SaveRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	item, err := s.automation.SaveRule(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rule": item})
}

func (s *Server) handleAutomationHistory(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := s.automation.ListHistory(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": rows})
}

func (s *Server) handleRunAutomationRule(w http.ResponseWriter, r *http.Request) {
	var body automation.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	out, err := s.automation.Run(r.Context(), body, func(ctx context.Context, action map[string]any, scope map[string]any, dryRun bool) (map[string]any, error) {
		typ := strings.TrimSpace(readStringAny(action, "type"))
		switch typ {
		case "create_job":
			req := jobs.CreateRequest{
				TemplateID:       readStringAny(action, "templateId"),
				Title:            readStringAny(action, "title"),
				UserRequest:      readStringAny(action, "userRequest"),
				Objective:        readStringAny(action, "objective"),
				Query:            readStringAny(action, "query"),
				InitiatingSource: "automation_rule",
				RequestPayload: map[string]any{
					"dryRun": dryRun,
				},
			}
			if req.TemplateID == "" {
				return nil, fmt.Errorf("automation create_job requires templateId")
			}
			job, err := s.jobs.Create(ctx, req)
			if err != nil {
				return nil, err
			}
			return map[string]any{"jobId": job.ID, "status": "queued"}, nil
		case "generate_dossier_brief":
			dossierID := readInt64FromScope(scope, "dossierId")
			if dossierID <= 0 {
				dossierID = readInt64FromScope(action, "dossierId")
			}
			if dossierID <= 0 {
				return nil, fmt.Errorf("automation generate_dossier_brief requires dossierId")
			}
			brief, err := s.dossiers.GenerateBrief(ctx, dossierID, "automation_rule")
			if err != nil {
				return nil, err
			}
			return map[string]any{"briefId": brief.ID, "dossierId": dossierID}, nil
		case "create_review":
			targetType := nonEmpty(readStringAny(action, "targetType"), "import")
			targetID := readStringAny(action, "targetId")
			if targetID == "" {
				if importID := readInt64FromScope(scope, "importId"); importID > 0 {
					targetID = strconv.FormatInt(importID, 10)
				}
			}
			if targetID == "" {
				return nil, fmt.Errorf("automation create_review requires targetId")
			}
			rec, err := s.reviews.Create(ctx, reviews.CreateRequest{
				TargetType: targetType,
				TargetID:   targetID,
				Status:     reviews.StatusPending,
				Summary:    "Automation-created review",
				Notes:      "Created by automation rule",
				Reviewer:   "operator",
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{"reviewId": rec.ID, "status": rec.Status}, nil
		case "suggest_strategy_adjustment":
			return map[string]any{
				"advisory": "review recent failures and adjust strategy packet rules",
			}, nil
		default:
			return nil, fmt.Errorf("unsupported automation action type %q", typ)
		}
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": out})
}

func (s *Server) handleAnalyzePacketGuidance(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PacketID  int64  `json:"packetId"`
		JobID     *string `json:"jobId"`
		DossierID *int64 `json:"dossierId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	item, err := s.packetOpt.AnalyzePacket(r.Context(), packetopt.AnalyzeRequest{
		PacketID:  body.PacketID,
		JobID:     body.JobID,
		DossierID: body.DossierID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"guidance": item})
}

func (s *Server) handleListPacketGuidance(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	packetID := parseOptionalInt(r.URL.Query().Get("packetId"))
	rows, err := s.packetOpt.List(r.Context(), limit, packetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"guidance": rows})
}

func (s *Server) handleGetImportReconciliation(w http.ResponseWriter, r *http.Request) {
	importID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	rec, err := s.reconcile.ByImport(r.Context(), importID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reconciliation": rec})
}

func (s *Server) handleSaveImportReconciliation(w http.ResponseWriter, r *http.Request) {
	importID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body reconciliation.SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	body.ImportID = importID
	rec, err := s.reconcile.Save(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reconciliation": rec})
}

func (s *Server) handleListReconciliations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	status := strings.TrimSpace(r.URL.Query().Get("reviewStatus"))
	rows, err := s.reconcile.List(r.Context(), limit, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reconciliations": rows})
}

func (s *Server) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	var body reviews.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	rec, err := s.reviews.Create(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"review": rec})
}

func (s *Server) handleListReviews(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	rows, err := s.reviews.List(r.Context(), status, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reviews": rows})
}

func (s *Server) handleUpdateReview(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body reviews.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	rec, err := s.reviews.Update(r.Context(), id, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"review": rec})
}

func (s *Server) handleAnalyzeFailurePatterns(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DossierID *int64 `json:"dossierId"`
		Lookback  int    `json:"lookback"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	rows, err := s.failures.Analyze(r.Context(), failurepatterns.AnalyzeRequest{
		DossierID: body.DossierID,
		Lookback:  body.Lookback,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"patterns": rows})
}

func (s *Server) handleListFailurePatterns(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	dossierID := parseOptionalInt(r.URL.Query().Get("dossierId"))
	rows, err := s.failures.List(r.Context(), limit, dossierID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"patterns": rows})
}

func readInt64FromScope(scope map[string]any, key string) int64 {
	if scope == nil {
		return 0
	}
	v, ok := scope[key]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return 0
	}
}

func readStringAny(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}
