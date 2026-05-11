package api

import (
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
		if s.auditSvc != nil {
			_, _ = s.auditSvc.Record(r.Context(), audit.CreateRequest{
				CorrelationID: meta.CorrelationID,
				Category:      "memory",
				Action:        baseAction + ".retired",
				Actor:         "api",
				SubjectType:   "observation",
				SubjectID:     subjectID,
				Outcome:       "denied",
				Summary:       "legacy memory mutation endpoint retired; use semantic syscall path",
				Payload: requestAuditPayload(map[string]any{
					"method":               r.Method,
					"path":                 r.URL.Path,
					"legacyMemoryMutation": true,
					"retired":              true,
				}, meta),
			})
		}
		http.Error(w, "legacy memory mutation endpoints are retired; use the authoritative semantic syscall path", http.StatusGone)
	}
}
