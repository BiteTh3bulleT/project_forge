package api

import (
	"net/http"
	"time"

	"forge/projectforge/services/core/internal/aios/controllane"
)

func (s *Server) forgeKActivationReadiness(now time.Time) controllane.ForgeKActivationReadinessReport {
	return controllane.ForgeKActivationReadiness(controllane.NewStaticActionRegistry(), now)
}

func (s *Server) handleForgeKernelStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.forgeKActivationReadiness(time.Now().UTC()))
}
