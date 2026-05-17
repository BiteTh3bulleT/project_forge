package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"forge/projectforge/services/core/internal/forgekshadow"
)

func (s *Server) mountMiddleware(r chi.Router) {
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestLogger(safeLogFormatter{}))
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(_ *http.Request, origin string) bool {
			return s.corsOriginAllowed(origin)
		},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Forge-Remote-Token", "X-Telegram-Bot-Api-Secret-Token"},
		AllowCredentials: false,
	}))
	r.Use(s.routeEnvelopeShadowMiddleware)
}

func (s *Server) routeEnvelopeShadowMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s == nil || s.forgeKShadow == nil || !s.forgeKShadow.Enabled() {
			next.ServeHTTP(w, r)
			return
		}
		started := time.Now()
		next.ServeHTTP(w, r)
		if r == nil || r.URL == nil || strings.TrimSpace(r.URL.Path) == "/health" {
			return
		}
		routePattern := strings.TrimSpace(chi.RouteContext(r.Context()).RoutePattern())
		if routePattern == "" {
			return
		}
		s.forgeKShadow.ObserveRouteEnvelopeBestEffort(r.Context(), forgekshadow.RouteEnvelopeInput{
			RequestID:    middleware.GetReqID(r.Context()),
			Method:       r.Method,
			Path:         r.URL.Path,
			RoutePattern: routePattern,
			RouteClass:   forgekshadow.NormalizeRouteClass(r.URL.Path, routePattern),
			Duration:     time.Since(started),
			Metadata: map[string]any{
				"touchpoint": "route_envelope",
			},
		})
	})
}

func (s *Server) mountHealthRoutes(r chi.Router) {
	r.Get("/health", s.handleHealth)
	s.mountMetricsRoutes(r)
}

func (s *Server) corsOriginAllowed(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	defaultOrigins := map[string]struct{}{
		"tauri://localhost":       {},
		"http://tauri.localhost":  {},
		"https://tauri.localhost": {},
	}
	if _, ok := defaultOrigins[origin]; ok {
		return true
	}
	for _, allowed := range s.cfg.CORSAllowedOrigins {
		if origin == strings.TrimSpace(allowed) {
			return true
		}
	}
	if s.cfg.CORSAllowDevLocalhost {
		return strings.HasPrefix(origin, "http://localhost:") ||
			strings.HasPrefix(origin, "http://127.0.0.1:")
	}
	return false
}

func (s *Server) mountForgeRoutes(r chi.Router) {
	r.Route("/forge", func(r chi.Router) {
		r.Use(s.requireAPIAuth)
		r.Use(middleware.Timeout(120 * time.Second))
		r.Get("/models", s.handleForgeModelsList)
		r.Post("/models/import", s.handleForgeModelImport)
		r.Post("/models/scan", s.handleForgeModelsScan)
		r.Get("/models/{id}", s.handleForgeModelGet)
		r.Get("/models/{id}/compatibility", s.handleForgeModelCompatibility)
		r.Post("/models/{id}/verify", s.handleForgeModelVerify)
		r.Post("/models/{id}/enable", s.handleForgeModelEnable)
		r.Post("/models/{id}/disable", s.handleForgeModelDisable)
		r.Post("/models/{id}/archive", s.handleForgeModelArchive)
		r.Post("/models/{id}/remove", s.handleForgeModelRemove)
		r.Post("/models/{id}/delete-file", s.handleForgeModelDeleteFile)
		r.Post("/models/{id}/load", s.handleForgeModelLoad)
		r.Post("/models/{id}/unload", s.handleForgeModelUnload)
		r.Post("/models/{id}/chat", s.handleForgeModelChat)
		r.Get("/model-runtime/backends", s.handleForgeModelRuntimeBackends)
		r.Get("/model-runtime/usage", s.handleForgeModelRuntimeUsage)
		r.Get("/model-runtime/health", s.handleForgeModelRuntimeHealth)
		r.Get("/model-runtime/queue", s.handleForgeModelRuntimeQueue)
		r.Get("/model-runtime/loaded", s.handleForgeModelRuntimeLoaded)
		r.Get("/kernel/status", s.handleForgeKernelStatus)
		r.Get("/system/status", s.handleForgeSystemStatus)
	})
}

func (s *Server) mountOpenAICompatRoutes(r chi.Router) {
	if s.cfg.EnableOpenAICompatAPI {
		r.Route("/v1", func(r chi.Router) {
			r.Use(s.requireAPIAuth)
			r.Use(middleware.Timeout(120 * time.Second))
			r.Get("/models", s.handleV1Models)
			r.Post("/chat/completions", s.handleV1ChatCompletions)
		})
	}
}

func (s *Server) mountAPIRoutes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		r.Use(s.requireAPIAuth)
		// Long-lived SSE — must not use the short HTTP timeout used for the rest of /api.
		r.Get("/chat/threads/{id}/assistant-stream", s.handleChatAssistantStream)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(120 * time.Second))

			s.mountSettingsRoutes(r)
			s.mountRemoteRoutes(r)
			s.mountSourceRoutes(r)
			s.mountProviderRoutes(r)
			s.mountAutonomyRoutes(r)
			s.mountDreamRoutes(r)
			s.mountAdapterRoutes(r)
			s.mountJobRoutes(r)
			s.mountChatRoutes(r)
			s.mountCanvasRoutes(r)
			s.mountArtifactRoutes(r)
			s.mountApprovalRoutes(r)
			s.mountContextRoutes(r)
			s.mountMemoryRoutes(r)
			s.mountKnowledgeRoutes(r)
			s.mountDashboardRoutes(r)
			s.mountGovernanceRoutes(r)
			s.mountGatewayRoutes(r)
			s.mountPermissionAuditRoutes(r)
			s.mountOperationsRoutes(r)
			s.mountCommandRoutes(r)
		})
	})
}

func (s *Server) mountSettingsRoutes(r chi.Router) {
	r.Get("/meta", s.handleMeta)
	r.Get("/settings", s.handleGetSettings)
	r.Patch("/settings", s.handlePatchSettings)
	r.Get("/settings/ollama-models", s.handleGetOllamaModels)
	r.Get("/settings/ollama-models/", s.handleGetOllamaModels)
}

func (s *Server) mountRemoteRoutes(r chi.Router) {
	r.Post("/remote/telegram", s.handleRemoteTelegram)
	r.Post("/remote/discord", s.handleRemoteDiscord)
	r.Get("/telegram/status", s.handleTelegramStatus)
	r.Get("/discord/status", s.handleDiscordGatewayStatus)
}

func (s *Server) mountSourceRoutes(r chi.Router) {
	r.Get("/sources", s.handleListSources)
	r.Post("/sources", s.handleAddSource)
	r.Delete("/sources/{id}", s.handleDeleteSource)
	r.Post("/reindex", s.handleReindex)
	r.Get("/search", s.handleSearch)
	r.Get("/chunks/{id}", s.handleChunk)
	r.Get("/events", s.handleEvents)
}

func (s *Server) mountProviderRoutes(r chi.Router) {
	r.Get("/providers/capabilities", s.handleProviderCapabilities)
}

func (s *Server) mountAutonomyRoutes(r chi.Router) {
	r.Get("/autonomy/status", s.handleAutonomyStatus)
	r.Get("/autonomy/intents", s.handleAutonomyIntents)
	r.Get("/autonomy/intents/{id}/explain", s.handleAutonomyIntentExplain)
	r.Get("/autonomy/decisions", s.handleAutonomyDecisions)
	r.Get("/autonomy/budgets", s.handleAutonomyBudgets)
	r.Get("/autonomy/charters", s.handleAutonomyCharters)
	r.Get("/autonomy/events", s.handleAutonomyEvents)
	r.Post("/autonomy/maintenance/sweep", s.handleAutonomyMaintenanceSweep)
}

func (s *Server) mountDreamRoutes(r chi.Router) {
	r.Post("/dream/run", s.handleDreamRun)
	r.Get("/dream/reports", s.handleDreamReportsList)
	r.Get("/dream/reports/{id}", s.handleDreamReportGet)
	r.Get("/dream/reports/{id}/candidates", s.handleDreamReportCandidates)
	r.Get("/dream/reports/{id}/proposals", s.handleDreamReportProposals)
	r.Get("/dream/reports/{id}/warnings", s.handleDreamReportWarnings)
}

func (s *Server) mountAdapterRoutes(r chi.Router) {
	r.Get("/adapters", s.handleAdapters)
}

func (s *Server) mountJobRoutes(r chi.Router) {
	r.Get("/jobs/templates", s.handleListJobTemplates)
	r.Get("/jobs", s.handleListJobs)
	r.Post("/jobs", s.handleCreateJob)
	r.Get("/jobs/{id}", s.handleJobDetail)
	r.Post("/jobs/{id}/cancel", s.handleCancelJob)
	r.Post("/jobs/{id}/retry", s.handleRetryJob)
	r.Post("/jobs/{id}/replay", s.handleReplayJob)
}

func (s *Server) mountChatRoutes(r chi.Router) {
	r.Get("/chat/threads", s.handleChatThreadsList)
	r.Post("/chat/threads", s.handleChatThreadCreate)
	r.Get("/chat/threads/{id}", s.handleChatThreadGet)
	r.Patch("/chat/threads/{id}", s.handleChatThreadPatch)
	r.Delete("/chat/threads/{id}", s.handleChatThreadDelete)
	r.Post("/chat/threads/{id}/messages", s.handleChatMessagePost)
	r.Post("/chat/threads/{id}/attachments", s.handleChatAttachmentUpload)
	r.Post("/chat/threads/{id}/jobs", s.handleChatJobCreate)
}

func (s *Server) mountCanvasRoutes(r chi.Router) {
	r.Get("/canvas/boards", s.handleCanvasBoardsList)
	r.Post("/canvas/boards", s.handleCanvasBoardCreate)
	r.Get("/canvas/boards/{id}", s.handleCanvasBoardGet)
	r.Delete("/canvas/boards/{id}", s.handleCanvasBoardDelete)
	r.Post("/canvas/boards/{id}/notes", s.handleCanvasNoteCreate)
	r.Patch("/canvas/boards/{id}/notes/{noteId}", s.handleCanvasNotePatch)
	r.Delete("/canvas/boards/{id}/notes/{noteId}", s.handleCanvasNoteDelete)
}

func (s *Server) mountArtifactRoutes(r chi.Router) {
	r.Get("/artifacts", s.handleArtifactsList)
	r.Get("/artifacts/{id}", s.handleArtifactGet)
	r.Get("/artifacts/{id}/content", s.handleArtifactContent)
}

func (s *Server) mountApprovalRoutes(r chi.Router) {
	r.Get("/approvals", s.handleListApprovals)
	r.Get("/approvals/{id}", s.handleGetApproval)
	r.Post("/approvals/{id}/approve", s.handleApproveRequest)
	r.Post("/approvals/{id}/deny", s.handleDenyRequest)
	r.Post("/approvals/{id}/cancel", s.handleCancelRequest)
}

func (s *Server) mountContextRoutes(r chi.Router) {
	r.Get("/packets/{id}", s.handleGetPacket)
	r.Get("/context-inspector/snapshots", s.handleContextSnapshotList)
	r.Get("/context-inspector/snapshots/{id}", s.handleContextSnapshotGet)
	r.Get("/context/restore/recent", s.handleContextRestoreRecent)
	r.Get("/context/restore/{id}", s.handleContextRestoreGet)
	r.Get("/context/restore/{id}/candidates", s.handleContextRestoreCandidates)
	r.Get("/context/restore/{id}/score", s.handleContextRestoreScore)
	r.Get("/context/restore/{id}/resume-hints", s.handleContextRestoreResumeHints)
	r.Get("/context/restore/outcomes", s.handleRestoreOutcomeList)
	r.Get("/context/restore/outcomes/{id}", s.handleRestoreOutcomeGet)
	r.Post("/context/restore/outcomes/{id}/feedback", s.handleRestoreOutcomeFeedback)
	r.Get("/process/health", s.handleProcessHealthTrace)
	r.Get("/project-context", s.handleGetProjectContext)
	r.Post("/project-context/import", s.handleImportProjectContext)
	r.Post("/project-context/regenerate", s.handleRegenerateProjectContext)
}

func (s *Server) mountMemoryRoutes(r chi.Router) {
	r.Get("/embeddings/status", s.handleEmbeddingStatus)
	r.Post("/embeddings/reembed", s.handleReembed)
	r.Get("/retrieval/runs", s.handleListRetrievalRuns)
	r.Post("/retrieval/runs", s.handleCreateRetrievalRun)
	r.Get("/retrieval/runs/{id}", s.handleGetRetrievalRun)
	r.Get("/retrieval/runs/{id}/vsa-signals", s.handleGetRetrievalRunVSASignals)
	r.Get("/retrieval/results/{id}/vsa-signal", s.handleGetRetrievalResultVSASignal)
	r.Post("/retrieval/results/{id}/usefulness", s.handleMarkRetrievalUsefulness)
	r.Get("/memory/observations", s.handleListMemoryObservations)
	r.Post("/memory/observations", s.withLegacyMemoryMutationGate("legacy.memory.observation.create", s.handleCreateMemoryObservation))
	r.Get("/memory/observations/{id}", s.handleGetMemoryObservation)
	r.Get("/memory/observations/{id}/vsa", s.handleGetObservationVSA)
	r.Patch("/memory/observations/{id}", s.withLegacyMemoryMutationGate("legacy.memory.observation.patch", s.handlePatchMemoryObservation))
	r.Post("/memory/observations/{id}/usefulness", s.withLegacyMemoryMutationGate("legacy.memory.observation.usefulness", s.handleMarkMemoryObservationUsefulness))
	r.Get("/memory/retrieval-runs/{id}/selection", s.handleGetRetrievalSelection)
	r.Get("/memory/packets/{id}/alignment", s.handleGetPacketAlignmentNotes)
	r.Get("/memory/dossiers/{id}", s.handleGetDossierMemory)
	r.Get("/memory/dossiers/{id}/vsa-summary", s.handleGetDossierVSASummary)
	r.Get("/memory/vsa/reindex-runs", s.handleListVSAReindexRuns)
	r.Get("/memory/vsa/reindex-runs/{id}", s.handleGetVSAReindexRun)
	r.Get("/memory/vsa/reindex/runs", s.handleListVSAReindexRuns)    // compatibility route
	r.Get("/memory/vsa/reindex/runs/{id}", s.handleGetVSAReindexRun) // compatibility route
	r.Post("/memory/vsa/reindex/run", s.handleRunVSAReindex)
	r.Get("/memory/repair-runs", s.handleListMemoryRepairRuns)
	r.Get("/memory/repair-runs/{id}", s.handleGetMemoryRepairRun)
	r.Post("/memory/repair/run", s.handleRunMemoryRepair)
}

func (s *Server) mountKnowledgeRoutes(r chi.Router) {
	r.Get("/dossiers", s.handleListDossiers)
	r.Post("/dossiers", s.handleCreateDossier)
	r.Get("/dossiers/{id}", s.handleGetDossierDetail)
	r.Patch("/dossiers/{id}", s.handleUpdateDossier)
	r.Post("/dossiers/{id}/briefs/generate", s.handleGenerateDossierBrief)
	r.Get("/evaluations", s.handleListEvaluations)
	r.Post("/evaluations", s.handleCreateEvaluation)
	r.Get("/evaluations/metrics", s.handleAdapterMetrics)
	r.Get("/lineage/jobs/{id}", s.handleJobLineage)
	r.Get("/imports/executions", s.handleListImportedExecutions)
	r.Post("/imports/executions", s.handleCreateImportedExecution)
	r.Get("/insights", s.handleListInsights)
	r.Post("/insights/generate", s.handleGenerateInsights)
}

func (s *Server) mountDashboardRoutes(r chi.Router) {
	r.Get("/dashboard", s.handleDashboard)
}

func (s *Server) mountGovernanceRoutes(r chi.Router) {
	r.Get("/strategies", s.handleListStrategies)
	r.Post("/strategies", s.handleSaveStrategy)
	r.Get("/policy/presets", s.handleListApprovalPresets)
	r.Post("/policy/presets", s.handleSaveApprovalPreset)
	r.Get("/policy/global-preset", s.handleGetGlobalPreset)
	r.Post("/policy/global-preset", s.handleSetGlobalPreset)
	r.Get("/policy/dossiers/{id}", s.handleGetDossierProfile)
	r.Post("/policy/dossiers/{id}", s.handleSaveDossierProfile)
	r.Post("/policy/recommend", s.handlePolicyRecommend)
	r.Get("/policy/recommendations", s.handleListPolicyRecommendations)
	r.Get("/automation/rules", s.handleListAutomationRules)
	r.Post("/automation/rules", s.handleSaveAutomationRule)
	r.Get("/automation/history", s.handleAutomationHistory)
	r.Post("/automation/run", s.handleRunAutomationRule)
	r.Get("/packet-guidance", s.handleListPacketGuidance)
	r.Post("/packet-guidance/analyze", s.handleAnalyzePacketGuidance)
	r.Get("/reconciliation/imports/{id}", s.handleGetImportReconciliation)
	r.Post("/reconciliation/imports/{id}", s.handleSaveImportReconciliation)
	r.Get("/reconciliation", s.handleListReconciliations)
	r.Get("/reviews", s.handleListReviews)
	r.Post("/reviews", s.handleCreateReview)
	r.Patch("/reviews/{id}", s.handleUpdateReview)
	r.Get("/failure-patterns", s.handleListFailurePatterns)
	r.Post("/failure-patterns/analyze", s.handleAnalyzeFailurePatterns)
}

func (s *Server) mountGatewayRoutes(r chi.Router) {
	r.Get("/gateway/tools", s.handleGatewayTools)
	r.Get("/gateway/capabilities", s.handleGatewayCapabilities)
	r.Patch("/gateway/capabilities/{id}/status", s.handleGatewayCapabilityStatusUpdate)
	r.Post("/gateway/invoke", s.handleGatewayInvoke)
	r.Get("/gateway/invocations", s.handleGatewayInvocations)
}

func (s *Server) mountPermissionAuditRoutes(r chi.Router) {
	r.Get("/action-lanes", s.handleListLanes)
	r.Post("/action-lanes", s.handleSaveLane)
	r.Delete("/action-lanes/{id}", s.handleDeleteLane)
	r.Get("/permissions/profiles", s.handleListPermissionProfiles)
	r.Post("/permissions/profiles", s.handleSavePermissionProfile)
	r.Post("/permissions/profiles/{id}/activate", s.handleActivatePermissionProfile)
	r.Delete("/permissions/profiles/{id}", s.handleDeletePermissionProfile)
	r.Get("/audit", s.handleAuditList)
	r.Get("/audit/trace", s.handleAuditTraceLookup)
	r.Get("/audit/trace/{correlationId}", s.handleAuditTrace)
}

func (s *Server) mountOperationsRoutes(r chi.Router) {
	r.Get("/backup/bundles", s.handleListBundles)
	r.Post("/backup/bundles", s.handleCreateBundle)
	r.Delete("/backup/bundles/{id}", s.handleDeleteBundle)
	r.Post("/backup/restore", s.handleRestoreBundle)
	r.Get("/release/readiness", s.handleReleaseReadiness)
	r.Get("/release/artifacts", s.handleReleaseArtifacts)
	r.Post("/release/artifacts", s.handleReleaseRecord)
	r.Get("/release/first-run", s.handleFirstRun)
}

func (s *Server) mountCommandRoutes(r chi.Router) {
	r.Post("/commands/execute", s.handleCommandExecute)
}
