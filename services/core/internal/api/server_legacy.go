package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/audit"
)

func (s *Server) legacyMemoryMutationRetired(baseAction string) http.HandlerFunc {
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
				Summary:       "legacy memory mutation endpoint retired; use production FORGE-K evidence syscalls",
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
			"message":             "legacy memory mutation endpoints are retired; use production FORGE-K MATERIALIZE_ADMITTED_EVIDENCE or REVISE_MEMORY_EVIDENCE",
			"historyPreserved":    true,
			"migrationReviewPath": migration,
		})
	}
}

func legacyMemoryObservationMigrationReviewPath() map[string]any {
	return map[string]any{
		"status":      "retired_write_surface",
		"liveOwner":   "services/core/internal/api terminal retirement gate",
		"targetOwner": "services/core/internal/forgekernel + governed semantic syscall",
		"steps": []string{
			"preserve existing memory_observations rows as historical evidence",
			"submit new observation-derived evidence through deterministic FORGE-K admission",
			"commit accepted memory evidence only through MATERIALIZE_ADMITTED_EVIDENCE",
			"preserve prior evidence and commit revisions only through REVISE_MEMORY_EVIDENCE",
		},
		"canonicalActions": []string{
			"MATERIALIZE_ADMITTED_EVIDENCE",
			"REVISE_MEMORY_EVIDENCE",
		},
		"noAuthorityClaims": []string{
			"does_not_import_forgek_simulator_services",
			"does_not_write_memory_observations",
			"does_not_admit_evidence",
			"does_not_bypass_production_forge_k_kernel",
		},
	}
}
