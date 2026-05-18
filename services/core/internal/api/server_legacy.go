package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/audit"
)

func (s *Server) withLegacyMemoryMutationGate(baseAction string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subjectID := strings.TrimSpace(chi.URLParam(r, "id"))
		if subjectID == "" {
			subjectID = "new"
		}
		meta := requestAuditMetaForBackup(r, "", "", "", "legacy.memory.mutation")
		migration := legacyMemoryObservationMigrationReviewPath()
		if s.auditSvc != nil {
			_, _ = s.auditSvc.Record(r.Context(), audit.CreateRequest{
				CorrelationID: meta.CorrelationID,
				Category:      "memory",
				Action:        baseAction + ".retired",
				Actor:         "api",
				SubjectType:   "observation",
				SubjectID:     subjectID,
				Outcome:       "denied",
				Summary:       "legacy memory mutation endpoint retired; use Courthouse review and semantic syscall path",
				Payload: requestAuditPayload(map[string]any{
					"method":               r.Method,
					"path":                 r.URL.Path,
					"legacyMemoryMutation": true,
					"retired":              true,
					"historyPreserved":     true,
					"migrationReviewPath":  migration,
				}, meta),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":               "legacy_memory_mutation_retired",
			"message":             "legacy memory mutation endpoints are retired; submit evidence through Courthouse admission review and commit canonical memory through the Control Lane semantic syscall path",
			"historyPreserved":    true,
			"migrationReviewPath": migration,
		})
	}
}

func legacyMemoryObservationMigrationReviewPath() map[string]any {
	return map[string]any{
		"status":      "retired_write_surface",
		"liveOwner":   "services/core/internal/api legacy gate + services/core/internal/aios/controllane",
		"targetOwner": "forgek.court + forgek.kernel",
		"steps": []string{
			"preserve existing memory_observations rows as historical evidence",
			"validate new observation-derived evidence through VALIDATE_ADMISSION_CANDIDATE",
			"commit accepted canonical memory only through Control Lane semantic syscalls",
		},
		"canonicalActions": []string{
			"VALIDATE_ADMISSION_CANDIDATE",
			"CREATE_NOTE",
			"UPDATE_STATE",
			"OPEN_LOOP",
			"CLOSE_LOOP",
		},
		"noAuthorityClaims": []string{
			"does_not_import_forgek_simulator_services",
			"does_not_write_memory_observations",
			"does_not_admit_evidence",
			"does_not_commit_truth_outside_control_lane",
		},
	}
}
