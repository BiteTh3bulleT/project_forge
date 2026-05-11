package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/aios/dream"
	"forge/projectforge/services/core/internal/aios/rulecells"
	"forge/projectforge/services/core/internal/approvals"
	"forge/projectforge/services/core/internal/artifacts"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/automation"
	"forge/projectforge/services/core/internal/backup"
	"forge/projectforge/services/core/internal/canvas"
	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/dashboard"
	"forge/projectforge/services/core/internal/dossiers"
	"forge/projectforge/services/core/internal/embeddings"
	"forge/projectforge/services/core/internal/evaluations"
	"forge/projectforge/services/core/internal/events"
	"forge/projectforge/services/core/internal/failurepatterns"
	"forge/projectforge/services/core/internal/forgekshadow"
	"forge/projectforge/services/core/internal/gateway"
	"forge/projectforge/services/core/internal/gpu"
	"forge/projectforge/services/core/internal/imports"
	"forge/projectforge/services/core/internal/ingest"
	"forge/projectforge/services/core/internal/insights"
	"forge/projectforge/services/core/internal/jobs"
	"forge/projectforge/services/core/internal/lanes"
	"forge/projectforge/services/core/internal/lineage"
	"forge/projectforge/services/core/internal/memory"
	"forge/projectforge/services/core/internal/packetopt"
	"forge/projectforge/services/core/internal/packets"
	"forge/projectforge/services/core/internal/permissions"
	"forge/projectforge/services/core/internal/policy"
	"forge/projectforge/services/core/internal/projectcontext"
	"forge/projectforge/services/core/internal/reconciliation"
	"forge/projectforge/services/core/internal/release"
	"forge/projectforge/services/core/internal/retrieval"
	"forge/projectforge/services/core/internal/reviews"
	"forge/projectforge/services/core/internal/search"
	"forge/projectforge/services/core/internal/store"
	"forge/projectforge/services/core/internal/strategies"
	"forge/projectforge/services/core/internal/watch"
)

const serverJSONRequestBodyLimit int64 = 1 << 20

var errServerRequestBodyTooLarge = errors.New("server json request body too large")

type Server struct {
	st              *store.Store
	cfg             config.Config
	log             *events.Logger
	ingest          *ingest.Service
	search          *search.Service
	adapters        *adapters.Registry
	approvals       *approvals.Service
	packets         *packets.Service
	artifacts       *artifacts.Service
	chat            *chat.Service
	canvas          *canvas.Service
	projectCtx      *projectcontext.Service
	embeddings      *embeddings.Service
	retrieval       *retrieval.Service
	memory          *memory.Service
	dossiers        *dossiers.Service
	evals           *evaluations.Service
	lineage         *lineage.Service
	imports         *imports.Service
	insights        *insights.Service
	strategies      *strategies.Service
	policy          *policy.Service
	automation      *automation.Service
	packetOpt       *packetopt.Service
	reviews         *reviews.Service
	reconcile       *reconciliation.Service
	failures        *failurepatterns.Service
	dashboard       *dashboard.Service
	jobs            *jobs.Service
	gateway         *gateway.Gateway
	lanes           *lanes.Service
	permissions     *permissions.Service
	auditSvc        *audit.Service
	backup          *backup.Service
	release         *release.Service
	modelRuntime    modelRuntimeService
	dream           *dream.Service
	gpuTelemetry    *gpu.Service
	intelTelemetry  *gpu.IntelService
	watch           *watch.Manager
	watchStop       context.CancelFunc
	shutdownOnce    sync.Once
	autonomy        *AutonomyMaintenanceLoop
	telegramMu      sync.RWMutex
	telegramGateway *TelegramGateway
	telegramErr     string
	discordMu       sync.RWMutex
	discordGateway  *DiscordGateway
	discordErr      string
	forgeKShadow    *forgekshadow.Observer
	shadowDB        *sql.DB

	// chatAssistInflight tracks assistant generation (async job or SSE) per thread/user-message key.
	chatAssistInflight sync.Map
}

func decodeServerJSONBody(r *http.Request, target any) error {
	if r.Body == nil {
		return io.EOF
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, serverJSONRequestBodyLimit+1))
	if err != nil {
		return err
	}
	if int64(len(raw)) > serverJSONRequestBodyLimit {
		return errServerRequestBodyTooLarge
	}
	return json.Unmarshal(raw, target)
}

func decodeOptionalServerJSONBody(r *http.Request, target any) error {
	if r.Body == nil {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, serverJSONRequestBodyLimit+1))
	if err != nil {
		return err
	}
	if int64(len(raw)) > serverJSONRequestBodyLimit {
		return errServerRequestBodyTooLarge
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	return json.Unmarshal(raw, target)
}

func writeServerDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errServerRequestBodyTooLarge) {
		http.Error(w, "server json request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "invalid json", http.StatusBadRequest)
}

func NewServer(st *store.Store, cfg config.Config) *Server {
	bg := context.Background()
	cfg = runtimeConfigFromSettings(st.DB, cfg)
	ev := events.New(st.DB)
	ext := loadSetting(st.DB, "extensions_csv", ingest.DefaultExtensionsCSV())
	ing := ingest.New(st.DB, ev, ext)
	ing.SetRootScope(cfg.WorkspaceDir)
	searchSvc := search.New(st.DB)
	embedSvc := embeddings.New(st.DB)
	ensureEmbeddingProviderConfig(bg, st.DB, cfg)
	memorySvc := memory.New(st.DB)
	retrievalSvc := retrieval.New(st.DB, searchSvc, embedSvc, memorySvc)
	artSvc := artifacts.New(st.DB, cfg.DataDir)
	chatSvc := chat.New(st.DB)
	canvasSvc := canvas.New(st.DB)
	pktSvc := packets.New(st.DB, searchSvc, memorySvc)
	appSvc := approvals.New(st.DB)
	pcSvc := projectcontext.New(st.DB, ev, cfg.WorkspaceDir, cfg.DataDir)
	dossierSvc := dossiers.New(st.DB)
	evalSvc := evaluations.New(st.DB)
	lineageSvc := lineage.New(st.DB)
	importSvc := imports.New(st.DB)
	insightSvc := insights.New(st.DB)
	strategySvc := strategies.New(st.DB)
	policySvc := policy.New(st.DB, strategySvc)
	automationSvc := automation.New(st.DB)
	packetOptSvc := packetopt.New(st.DB)
	reviewSvc := reviews.New(st.DB)
	reconcileSvc := reconciliation.New(st.DB)
	failureSvc := failurepatterns.New(st.DB)
	dashboardSvc := dashboard.New(st.DB)
	reg := adapters.NewRegistry(adapters.Options{
		DB:           st.DB,
		WorkspaceDir: cfg.WorkspaceDir,
	})
	permSvc := permissions.New(st.DB)
	laneSvc := lanes.New(st.DB)
	auditSvc := audit.New(st.DB)
	_ = permSvc.EnsureDefaults(bg, cfg.WorkspaceDir)
	_ = os.MkdirAll(filepath.Join(cfg.WorkspaceDir, "scratch"), 0o755)
	_ = permSvc.EnsureMkdirChatPolicy(bg, cfg.WorkspaceDir)
	_ = permSvc.EnsureGatewayToolPolicy(bg, cfg.WorkspaceDir)
	if filepath.Clean(cfg.WorkspaceDir) == string(filepath.Separator) {
		_, _ = permSvc.Activate(bg, "workspace-write")
	}
	_ = laneSvc.EnsureDefaults(bg, cfg.WorkspaceDir)
	backupSvc := backup.New(st.DB, cfg.DataDir)
	releaseSvc := release.New(st.DB, cfg.DataDir, cfg.WorkspaceDir)
	gpuTelemetrySvc := gpu.New(gpu.Options{
		Enabled:                 cfg.NVIDIADCGMEnabled && !cfg.SafeModeForceCPUOnly,
		Endpoint:                cfg.NVIDIADCGMEndpoint,
		Timeout:                 time.Duration(cfg.NVIDIADCGMTimeoutMs) * time.Millisecond,
		MemoryPressureThreshold: cfg.GPUBackgroundMemoryPressureBlockThreshold,
	})
	intelTelemetrySvc := gpu.NewIntel(gpu.IntelOptions{
		Enabled:     cfg.IntelLevelZeroEnabled && !cfg.SafeModeForceCPUOnly,
		ZEInfoPath:  cfg.IntelLevelZeroZEInfoPath,
		IntelGPUTop: cfg.IntelGPUTopPath,
		Timeout:     time.Duration(cfg.IntelGPUTelemetryTimeoutMs) * time.Millisecond,
	})
	modelRuntimeSvc := initModelRuntimeService(cfg, auditSvc, gpuTelemetrySvc, intelTelemetrySvc)
	ruleEngine := rulecells.MustStaticEngine()
	dreamSvc := dream.NewService(st.DB)
	dreamSvc.SetRuleEngine(ruleEngine)
	var shadowObserver *forgekshadow.Observer
	var shadowDB *sql.DB
	if cfg.ForgeKShadowModeEnabled {
		shadowConfig := forgekshadow.Config{
			Enabled:                      true,
			ChatMetadataEnabled:          cfg.ForgeKShadowChatMetadataEnabled,
			RetrievalMetadataEnabled:     cfg.ForgeKShadowRetrievalMetadataEnabled,
			AdvisoryEnabled:              cfg.ForgeKShadowAdvisoryEnabled,
			ControlLaneValidationEnabled: cfg.ForgeKShadowControlLaneValidationEnabled,
		}
		var shadowSink forgekshadow.Sink = forgekshadow.NewMemorySink(forgekshadow.DefaultMaxReports)
		if cfg.ShadowDiagnosticPersistenceEnabled {
			if err := cfg.ValidateShadowDiagnosticPersistence(); err != nil {
				log.Printf("forge-k shadow diagnostic persistence disabled: %v", err)
			} else if db, err := sql.Open("pgx", cfg.PostgresDSN); err != nil {
				log.Printf("forge-k shadow diagnostic postgres open failed: %v", err)
			} else if err := store.NewPostgresMigrationRunner(store.PostgresMigrations()).Run(context.Background(), db); err != nil {
				_ = db.Close()
				log.Printf("forge-k shadow diagnostic postgres migrations failed: %v", err)
			} else {
				shadowDB = db
				shadowSink = forgekshadow.NewDiagnosticPersistenceSink(shadowSink, forgekshadow.NewPostgresDiagnosticRepository(db), forgekshadow.DiagnosticPersistenceOptions{
					Enabled:         true,
					RetentionDays:   cfg.ShadowDiagnosticRetentionDays,
					MaxPayloadBytes: cfg.ShadowDiagnosticMaxPayloadBytes,
				})
			}
		}
		shadowObserver = forgekshadow.NewObserverWithSink(shadowConfig, shadowSink, nil)
	}
	var autonomyLoop *AutonomyMaintenanceLoop
	if loop := newDefaultAutonomyMaintenanceLoop(st.DB, cfg, ev, memorySvc, shadowObserver); loop != nil {
		autonomyLoop = loop
	}
	capabilityRegistry, err := gateway.NewToolCapabilityRegistryWithStore(bg, &gateway.SQLiteOverrideStore{DB: st.DB})
	if err != nil {
		log.Printf("tool capability override store unavailable; using in-memory registry: %v", err)
		capabilityRegistry = gateway.NewToolCapabilityRegistry()
	}
	gw := gateway.New(gateway.Options{
		DB:                 st.DB,
		Permissions:        permSvc,
		Lanes:              laneSvc,
		Approvals:          appSvc,
		Audit:              auditSvc,
		WorkspaceDir:       cfg.WorkspaceDir,
		DataDir:            cfg.DataDir,
		CapabilityRegistry: capabilityRegistry,
		AutonomyPolicy:     newGatewayAutonomyAuthorizer(autonomyLoop),
	})
	if err := gw.RegisterTool(newLegacyAdapterGatewayTool(reg)); err != nil {
		log.Printf("legacy adapter gateway tool registration failed: %v", err)
	}
	jobSvc := jobs.NewService(jobs.Dependencies{
		DB:           st.DB,
		Logger:       ev,
		Ingest:       ing,
		Adapters:     reg,
		Packets:      pktSvc,
		Approvals:    appSvc,
		Artifacts:    artSvc,
		ProjectCtx:   pcSvc,
		Dossiers:     dossierSvc,
		Retrieval:    retrievalSvc,
		Gateway:      gw,
		WorkspaceDir: cfg.WorkspaceDir,
	})
	_ = strategySvc.EnsureDefaults(context.Background())
	_ = policySvc.EnsureDefaults(context.Background())
	_ = automationSvc.EnsureDefaults(context.Background())
	wm, err := watch.New(ing, ev)
	if err != nil {
		log.Printf("watch disabled: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	startApprovalExpiryReaper(ctx, appSvc)
	if wm != nil {
		wm.Run(ctx)
		_ = wm.SyncSources(context.Background(), listSourcePaths(st.DB))
	}
	if autonomyLoop != nil {
		go autonomyLoop.Run(ctx)
	}
	srv := &Server{
		st:             st,
		cfg:            cfg,
		log:            ev,
		ingest:         ing,
		search:         searchSvc,
		adapters:       reg,
		approvals:      appSvc,
		packets:        pktSvc,
		artifacts:      artSvc,
		chat:           chatSvc,
		canvas:         canvasSvc,
		projectCtx:     pcSvc,
		embeddings:     embedSvc,
		retrieval:      retrievalSvc,
		memory:         memorySvc,
		dossiers:       dossierSvc,
		evals:          evalSvc,
		lineage:        lineageSvc,
		imports:        importSvc,
		insights:       insightSvc,
		strategies:     strategySvc,
		policy:         policySvc,
		automation:     automationSvc,
		packetOpt:      packetOptSvc,
		reviews:        reviewSvc,
		reconcile:      reconcileSvc,
		failures:       failureSvc,
		dashboard:      dashboardSvc,
		jobs:           jobSvc,
		gateway:        gw,
		lanes:          laneSvc,
		permissions:    permSvc,
		auditSvc:       auditSvc,
		backup:         backupSvc,
		release:        releaseSvc,
		modelRuntime:   modelRuntimeSvc,
		dream:          dreamSvc,
		gpuTelemetry:   gpuTelemetrySvc,
		intelTelemetry: intelTelemetrySvc,
		watch:          wm,
		watchStop:      cancel,
		autonomy:       autonomyLoop,
		forgeKShadow:   shadowObserver,
		shadowDB:       shadowDB,
	}
	srv.telegramGateway = srv.tryStartTelegramGateway(ctx, cfg)
	srv.discordGateway = srv.tryStartDiscordGateway(ctx, cfg)
	return srv
}

func startApprovalExpiryReaper(ctx context.Context, svc *approvals.Service) {
	if svc == nil {
		return
	}
	run := func() {
		n, err := svc.Expire(ctx)
		if err != nil && ctx.Err() == nil {
			log.Printf("approval expiry sweep failed: %v", err)
			return
		}
		if n > 0 {
			log.Printf("approval expiry sweep expired %d request(s)", n)
		}
	}
	run()
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

func (s *Server) ShutdownWatch() {
	s.shutdownOnce.Do(func() {
		if s.jobs != nil {
			s.jobs.Close()
		}
		if s.autonomy != nil {
			s.autonomy.Stop()
		}
		s.telegramMu.Lock()
		if s.telegramGateway != nil {
			s.telegramGateway.Stop()
			s.telegramGateway = nil
		}
		s.telegramMu.Unlock()
		s.discordMu.Lock()
		if s.discordGateway != nil {
			s.discordGateway.Stop()
			s.discordGateway = nil
		}
		s.discordMu.Unlock()
		if s.watchStop != nil {
			s.watchStop()
		}
		if s.watch != nil {
			_ = s.watch.Close()
		}
		if s.shadowDB != nil {
			_ = s.shadowDB.Close()
		}
	})
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	s.mountMiddleware(r)
	s.mountHealthRoutes(r)
	s.mountForgeRoutes(r)
	s.mountOpenAICompatRoutes(r)
	s.mountAPIRoutes(r)
	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	payload := map[string]any{
		"ok":               true,
		"service":          "forge-core",
		"cpuAuthoritative": true,
	}
	safeModeReasons := []string{}
	if s.cfg.SafeModeForceCPUOnly {
		safeModeReasons = append(safeModeReasons, "safe_mode.force_cpu_only is enabled")
	}
	payload["safeMode"] = map[string]any{
		"active":  s.cfg.SafeModeForceCPUOnly,
		"reasons": safeModeReasons,
	}
	modelRuntimeStatus := map[string]any{
		"available": s.modelRuntime != nil,
		"status":    "unavailable",
	}
	if s.modelRuntime != nil {
		meta := modelRuntimeMetaFromRequestAudit(requestAuditMetaForBackup(r, "", "", "", "health"))
		health, err := s.modelRuntime.Health(r.Context(), meta)
		if err != nil {
			modelRuntimeStatus["status"] = "degraded"
			modelRuntimeStatus["error"] = err.Error()
		} else {
			modelRuntimeStatus["status"] = health.Status
			modelRuntimeStatus["runtimeEnabled"] = health.RuntimeEnabled
			modelRuntimeStatus["gpuAware"] = health.GPUAware
			modelRuntimeStatus["degradedReasons"] = append([]string(nil), health.DegradedReasons...)
			modelRuntimeStatus["policyWarnings"] = append([]string(nil), health.PolicyWarnings...)
		}
	}
	payload["modelRuntime"] = modelRuntimeStatus
	if s.gpuTelemetry != nil {
		payload["gpuTelemetry"] = s.gpuTelemetry.Snapshot(r.Context())
	}
	if s.intelTelemetry != nil {
		payload["intelTelemetry"] = s.intelTelemetry.Snapshot(r.Context())
	}
	if s.embeddings != nil {
		cfg := s.embeddings.CurrentConfig(r.Context())
		payload["embeddings"] = map[string]any{
			"config":         cfg,
			"health":         s.embeddings.ProviderHealth(r.Context(), cfg.Provider, cfg.Model),
			"truthAuthority": false,
		}
	}
	writeJSON(w, http.StatusOK, payload)
	if s.forgeKShadow != nil && s.forgeKShadow.Enabled() {
		s.forgeKShadow.ObserveBestEffort(r.Context(), forgekshadow.ObservationInput{
			WorkspaceID:    s.cfg.WorkspaceDir,
			RequestID:      r.Header.Get("X-Request-ID"),
			LivePath:       "GET /health",
			Method:         r.Method,
			Path:           r.URL.Path,
			RequestSummary: "health route metadata only",
			Metadata: map[string]any{
				"route":      "/health",
				"method":     r.Method,
				"touchpoint": "health",
			},
		})
	}
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"dataDir":      s.cfg.DataDir,
		"dbPath":       filepath.Join(s.cfg.DataDir, "forge.sqlite"),
		"workspaceDir": s.cfg.WorkspaceDir,
	})
}

const redactedSettingSecret = "[redacted]"

func redactedSettingSecretValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return redactedSettingSecret
}

func shouldPersistSettingSecret(value string) bool {
	return strings.TrimSpace(value) != redactedSettingSecret
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	ext := loadSetting(s.st.DB, "extensions_csv", ingest.DefaultExtensionsCSV())
	theme := loadSetting(s.st.DB, "theme", "dark")
	ollamaBase := loadSetting(s.st.DB, "ollama_base_url", "http://127.0.0.1:11434")
	ollamaModel := normalizeOllamaModel(loadSetting(s.st.DB, "ollama_model", ""))
	embeddingProvider := loadSetting(s.st.DB, "embedding_provider", "local_hash")
	embeddingModel := loadSetting(s.st.DB, "embedding_model", "")
	embeddingDims := loadSetting(s.st.DB, "embedding_dims", "128")
	embeddingTEIEndpoint := loadSetting(s.st.DB, "embedding_tei_endpoint", "")
	embeddingTEITimeoutMs := loadSetting(s.st.DB, "embedding_tei_timeout_ms", "30000")
	retrievalWeightKeyword := loadSetting(s.st.DB, "retrieval_weight_keyword", "0.45")
	retrievalWeightSemantic := loadSetting(s.st.DB, "retrieval_weight_semantic", "0.55")
	retrievalVSAMode := loadSetting(s.st.DB, "retrieval_vsa_mode", "off")
	retrievalVSADims := loadSetting(s.st.DB, "retrieval_vsa_dims", "128")
	retrievalVSASeed := loadSetting(s.st.DB, "retrieval_vsa_seed", "17")
	retrievalVSAWeightAssociative := loadSetting(s.st.DB, "retrieval_vsa_weight_associative", "0.06")
	retrievalVSAWeightRoleMatch := loadSetting(s.st.DB, "retrieval_vsa_weight_role_match", "0.04")
	retrievalVSAWeightRelational := loadSetting(s.st.DB, "retrieval_vsa_weight_relational", "0.03")
	retrievalVSAWeightFeedback := loadSetting(s.st.DB, "retrieval_vsa_weight_feedback", "0.03")
	retrievalVSAMaxAdditive := loadSetting(s.st.DB, "retrieval_vsa_max_additive", "0.12")
	chatPersonalityPrompt := loadSetting(s.st.DB, "chat_personality_prompt", defaultChatOperatorSystemPrompt())
	remoteAccessEnabled := parseRemoteBool(loadSetting(s.st.DB, remoteAccessEnabledKey, "false"))
	remoteAccessToken := strings.TrimSpace(loadSetting(s.st.DB, remoteAccessTokenKey, ""))
	remoteCrossChatContext := parseRemoteBool(loadSetting(s.st.DB, remoteCrossChatContextKey, "false"))
	telegramBotToken := strings.TrimSpace(loadSetting(s.st.DB, telegramBotTokenKey, ""))
	telegramDefaultChatID := strings.TrimSpace(loadSetting(s.st.DB, telegramDefaultChatIDKey, ""))
	discordBotToken := strings.TrimSpace(loadSetting(s.st.DB, discordBotTokenKey, ""))
	discordDefaultChannelID := strings.TrimSpace(loadSetting(s.st.DB, discordDefaultChannelIDKey, ""))
	discordWebhookURL := strings.TrimSpace(loadSetting(s.st.DB, discordWebhookURLKey, ""))
	discordCrossChatContext := parseRemoteBool(loadSetting(s.st.DB, discordGatewayCrossChatContextKey, "false"))
	remoteDefaultThreadID := strings.TrimSpace(loadSetting(s.st.DB, remoteDefaultThreadIDKey, ""))
	dreamModeEnabled := parseRemoteBool(loadSetting(s.st.DB, "dream_mode_enabled", "true"))
	dreamModeDefaultDryRun := parseRemoteBool(loadSetting(s.st.DB, "dream_mode_default_dry_run", "true"))
	dreamModeMode := strings.TrimSpace(loadSetting(s.st.DB, "dream_mode_mode", "microdream"))
	dreamModeWindowHours := loadSetting(s.st.DB, "dream_mode_window_hours", "6")
	dreamModeMaxCandidates := loadSetting(s.st.DB, "dream_mode_max_candidates", "8")
	dreamModeAllowLongTermPromotion := parseRemoteBool(loadSetting(s.st.DB, "dream_mode_allow_long_term_promotion", "false"))
	dreamModeRequireOperatorReviewForLongTerm := parseRemoteBool(loadSetting(s.st.DB, "dream_mode_require_operator_review_for_long_term", "true"))
	dreamModeAllowCommits := parseRemoteBool(loadSetting(s.st.DB, "dream_mode_allow_commits", "false"))
	runtimeControls := runtimeControlsFromSettings(s.st.DB, s.cfg)
	shadowMode := shadowModeFromSettings(s.st.DB, s.cfg)
	writeJSON(w, http.StatusOK, map[string]any{
		"extensionsCsv":                 ext,
		"theme":                         theme,
		"ollamaBaseUrl":                 ollamaBase,
		"ollamaModel":                   ollamaModel,
		"embeddingProvider":             embeddingProvider,
		"embeddingModel":                embeddingModel,
		"embeddingDims":                 embeddingDims,
		"embeddingTeiEndpoint":          embeddingTEIEndpoint,
		"embeddingTeiTimeoutMs":         embeddingTEITimeoutMs,
		"retrievalWeightKeyword":        retrievalWeightKeyword,
		"retrievalWeightSemantic":       retrievalWeightSemantic,
		"retrievalVSAMode":              retrievalVSAMode,
		"retrievalVSADims":              retrievalVSADims,
		"retrievalVSASeed":              retrievalVSASeed,
		"retrievalVSAWeightAssociative": retrievalVSAWeightAssociative,
		"retrievalVSAWeightRoleMatch":   retrievalVSAWeightRoleMatch,
		"retrievalVSAWeightRelational":  retrievalVSAWeightRelational,
		"retrievalVSAWeightFeedback":    retrievalVSAWeightFeedback,
		"retrievalVSAMaxAdditive":       retrievalVSAMaxAdditive,
		"retrievalVsaMode":              retrievalVSAMode,
		"retrievalVsaDims":              retrievalVSADims,
		"retrievalVsaSeed":              retrievalVSASeed,
		"retrievalVsaWeightAssociative": retrievalVSAWeightAssociative,
		"retrievalVsaWeightRoleMatch":   retrievalVSAWeightRoleMatch,
		"retrievalVsaWeightRelational":  retrievalVSAWeightRelational,
		"retrievalVsaWeightFeedback":    retrievalVSAWeightFeedback,
		"retrievalVsaMaxAdditive":       retrievalVSAMaxAdditive,
		"chatPersonalityPrompt":         chatPersonalityPrompt,
		"chatPromptDefault":             defaultChatOperatorSystemPrompt(),
		"remoteAccessEnabled":           remoteAccessEnabled,
		"remoteAccessToken":             redactedSettingSecretValue(remoteAccessToken),
		"remoteAccessTokenConfigured":   remoteAccessToken != "",
		"remoteCrossChatContext":        remoteCrossChatContext,
		"remoteDefaultThreadId":         remoteDefaultThreadID,
		"telegramBotToken":              redactedSettingSecretValue(telegramBotToken),
		"telegramBotTokenConfigured":    telegramBotToken != "",
		"telegramDefaultChatId":         telegramDefaultChatID,
		"discordBotToken":               redactedSettingSecretValue(discordBotToken),
		"discordBotTokenConfigured":     discordBotToken != "",
		"discordDefaultChannelId":       discordDefaultChannelID,
		"discordWebhookUrl":             redactedSettingSecretValue(discordWebhookURL),
		"discordWebhookUrlConfigured":   discordWebhookURL != "",
		"discordCrossChatContext":       discordCrossChatContext,
		"dreamMode": map[string]any{
			"enabled":                          dreamModeEnabled,
			"defaultDryRun":                    dreamModeDefaultDryRun,
			"mode":                             dreamModeMode,
			"windowHours":                      dreamModeWindowHours,
			"maxCandidates":                    dreamModeMaxCandidates,
			"allowLongTermPromotion":           dreamModeAllowLongTermPromotion,
			"requireOperatorReviewForLongTerm": dreamModeRequireOperatorReviewForLongTerm,
			"allowCommits":                     dreamModeAllowCommits,
		},
		"runtimeControls": runtimeControls,
		"shadowMode":      shadowMode,
	})
}

func (s *Server) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body map[string]any
	if err := decodeServerJSONBody(r, &body); err != nil {
		writeServerDecodeError(w, err)
		return
	}
	discordConfigChanged := false
	telegramConfigChanged := false
	if v, ok := body["extensionsCsv"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "extensions_csv", v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.ingest.SetExtensionsCSV(v)
	}
	if v, ok := body["theme"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "theme", v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if v, ok := body["ollamaBaseUrl"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "ollama_base_url", v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if v, ok := body["ollamaModel"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "ollama_model", normalizeOllamaModel(v)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if v, ok := body["embeddingProvider"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "embedding_provider", v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if v, ok := body["embeddingModel"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "embedding_model", v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	switch v := body["embeddingDims"].(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "embedding_dims", v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "embedding_dims", strconv.Itoa(int(v))); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if v, ok := body["embeddingTeiEndpoint"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "embedding_tei_endpoint", v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if v, ok := body["embeddingTeiApiKey"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, "embedding_tei_api_key", v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	switch v := body["embeddingTeiTimeoutMs"].(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "embedding_tei_timeout_ms", v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "embedding_tei_timeout_ms", strconv.Itoa(int(v))); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	switch v := body["retrievalWeightKeyword"].(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_weight_keyword", v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_weight_keyword", strconv.FormatFloat(v, 'f', 4, 64)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	switch v := body["retrievalWeightSemantic"].(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_weight_semantic", v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_weight_semantic", strconv.FormatFloat(v, 'f', 4, 64)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	modeRaw, hasMode := body["retrievalVSAMode"]
	if !hasMode {
		modeRaw = body["retrievalVsaMode"]
	}
	if v, ok := modeRaw.(string); ok {
		mode := strings.ToLower(strings.TrimSpace(v))
		if mode != "off" && mode != "shadow" && mode != "active" {
			mode = "off"
		}
		if err := upsertSetting(ctx, s.st.DB, "retrieval_vsa_mode", mode); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	dimsRaw, hasDims := body["retrievalVSADims"]
	if !hasDims {
		dimsRaw = body["retrievalVsaDims"]
	}
	switch v := dimsRaw.(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_vsa_dims", strings.TrimSpace(v)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_vsa_dims", strconv.Itoa(int(v))); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	seedRaw, hasSeed := body["retrievalVSASeed"]
	if !hasSeed {
		seedRaw = body["retrievalVsaSeed"]
	}
	switch v := seedRaw.(type) {
	case string:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_vsa_seed", strings.TrimSpace(v)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case float64:
		if err := upsertSetting(ctx, s.st.DB, "retrieval_vsa_seed", strconv.Itoa(int(v))); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for _, item := range []struct {
		BodyKeys   []string
		SettingKey string
	}{
		{BodyKeys: []string{"retrievalVSAWeightAssociative", "retrievalVsaWeightAssociative"}, SettingKey: "retrieval_vsa_weight_associative"},
		{BodyKeys: []string{"retrievalVSAWeightRoleMatch", "retrievalVsaWeightRoleMatch"}, SettingKey: "retrieval_vsa_weight_role_match"},
		{BodyKeys: []string{"retrievalVSAWeightRelational", "retrievalVsaWeightRelational"}, SettingKey: "retrieval_vsa_weight_relational"},
		{BodyKeys: []string{"retrievalVSAWeightFeedback", "retrievalVsaWeightFeedback"}, SettingKey: "retrieval_vsa_weight_feedback"},
		{BodyKeys: []string{"retrievalVSAMaxAdditive", "retrievalVsaMaxAdditive"}, SettingKey: "retrieval_vsa_max_additive"},
	} {
		var (
			raw any
			ok  bool
		)
		for _, key := range item.BodyKeys {
			if raw, ok = body[key]; ok {
				break
			}
		}
		if ok {
			switch v := raw.(type) {
			case string:
				if err := upsertSetting(ctx, s.st.DB, item.SettingKey, strings.TrimSpace(v)); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			case float64:
				if err := upsertSetting(ctx, s.st.DB, item.SettingKey, strconv.FormatFloat(v, 'f', 4, 64)); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	}
	if v, ok := body["remoteAccessEnabled"]; ok {
		if err := upsertSetting(ctx, s.st.DB, remoteAccessEnabledKey, parseRemoteBoolValue(v)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		discordConfigChanged = true
		telegramConfigChanged = true
	}
	if v, ok := body["remoteAccessToken"].(string); ok {
		if shouldPersistSettingSecret(v) {
			if err := upsertSetting(ctx, s.st.DB, remoteAccessTokenKey, strings.TrimSpace(v)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	if v, ok := body["remoteCrossChatContext"]; ok {
		if err := upsertSetting(ctx, s.st.DB, remoteCrossChatContextKey, parseRemoteBoolValue(v)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if raw, ok := body["remoteDefaultThreadId"]; ok {
		if v := parseAnyInt64(raw); v > 0 {
			if err := upsertSetting(ctx, s.st.DB, remoteDefaultThreadIDKey, strconv.FormatInt(v, 10)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if threadIDRaw, ok := raw.(string); ok && strings.TrimSpace(threadIDRaw) == "" {
			if err := upsertSetting(ctx, s.st.DB, remoteDefaultThreadIDKey, ""); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	if v, ok := body["telegramBotToken"].(string); ok {
		if shouldPersistSettingSecret(v) {
			if err := upsertSetting(ctx, s.st.DB, telegramBotTokenKey, strings.TrimSpace(v)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			telegramConfigChanged = true
		}
	}
	if v, ok := body["telegramDefaultChatId"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, telegramDefaultChatIDKey, strings.TrimSpace(v)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if v, ok := body["discordBotToken"].(string); ok {
		if shouldPersistSettingSecret(v) {
			if err := upsertSetting(ctx, s.st.DB, discordBotTokenKey, strings.TrimSpace(v)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			discordConfigChanged = true
		}
	}
	if v, ok := body["discordDefaultChannelId"].(string); ok {
		if err := upsertSetting(ctx, s.st.DB, discordDefaultChannelIDKey, strings.TrimSpace(v)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		discordConfigChanged = true
	}
	if v, ok := body["discordWebhookUrl"].(string); ok {
		if shouldPersistSettingSecret(v) {
			if err := upsertSetting(ctx, s.st.DB, discordWebhookURLKey, strings.TrimSpace(v)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			discordConfigChanged = true
		}
	}
	if v, ok := body["discordCrossChatContext"]; ok {
		if err := upsertSetting(ctx, s.st.DB, discordGatewayCrossChatContextKey, parseRemoteBoolValue(v)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		discordConfigChanged = true
	}
	if v, ok := body["chatPersonalityPrompt"].(string); ok {
		next := strings.TrimSpace(v)
		if next == "" {
			next = defaultChatOperatorSystemPrompt()
		}
		if err := upsertSetting(ctx, s.st.DB, "chat_personality_prompt", next); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if rawDreamMode, ok := body["dreamMode"]; ok {
		dreamMode, ok := rawDreamMode.(map[string]any)
		if !ok {
			http.Error(w, "dreamMode must be an object", http.StatusBadRequest)
			return
		}
		for _, item := range []struct {
			bodyKey    string
			settingKey string
		}{
			{bodyKey: "enabled", settingKey: "dream_mode_enabled"},
			{bodyKey: "defaultDryRun", settingKey: "dream_mode_default_dry_run"},
			{bodyKey: "allowLongTermPromotion", settingKey: "dream_mode_allow_long_term_promotion"},
			{bodyKey: "requireOperatorReviewForLongTerm", settingKey: "dream_mode_require_operator_review_for_long_term"},
			{bodyKey: "allowCommits", settingKey: "dream_mode_allow_commits"},
		} {
			if v, ok := dreamMode[item.bodyKey]; ok {
				if err := upsertSetting(ctx, s.st.DB, item.settingKey, parseRemoteBoolValue(v)); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
		if v, ok := dreamMode["mode"].(string); ok {
			mode := strings.TrimSpace(v)
			switch mode {
			case "microdream", "nap", "deep_dream":
			default:
				mode = "microdream"
			}
			if err := upsertSetting(ctx, s.st.DB, "dream_mode_mode", mode); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		for _, item := range []struct {
			bodyKey    string
			settingKey string
		}{
			{bodyKey: "windowHours", settingKey: "dream_mode_window_hours"},
			{bodyKey: "maxCandidates", settingKey: "dream_mode_max_candidates"},
		} {
			if raw, ok := dreamMode[item.bodyKey]; ok {
				if value := parseAnyInt64(raw); value > 0 {
					if err := upsertSetting(ctx, s.st.DB, item.settingKey, strconv.FormatInt(value, 10)); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
			}
		}
	}
	if err := s.patchShadowMode(ctx, body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if discordConfigChanged {
		s.reloadDiscordGateway(ctx)
	}
	if telegramConfigChanged {
		s.reloadTelegramGateway(ctx)
	}
	if err := s.patchRuntimeControls(ctx, body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.log.Emit(ctx, "command.executed", map[string]any{"command": "settings.patch"})
	s.handleGetSettings(w, r)
}

func (s *Server) handleGetOllamaModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := strings.TrimSpace(r.URL.Query().Get("baseUrl"))
	if baseURL == "" {
		baseURL = strings.TrimSpace(loadSetting(s.st.DB, "ollama_base_url", "http://127.0.0.1:11434"))
	}
	// Always allow best-effort discovery so settings UX can render even if Ollama is offline.
	resp := map[string]any{
		"baseUrl": baseURL,
	}

	ollamaAdapter, err := s.adapters.Get("ollama")
	if err != nil {
		resp["status"] = "unavailable"
		resp["error"] = err.Error()
		resp["models"] = []string{}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	ollama, ok := ollamaAdapter.(adapters.Ollama)
	if !ok {
		resp["status"] = "unavailable"
		resp["error"] = "ollama adapter has unexpected type"
		resp["models"] = []string{}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	models, err := ollama.FetchModels(ctx, baseURL, 1800*time.Millisecond)
	if err != nil {
		resp["status"] = "unavailable"
		resp["error"] = err.Error()
		resp["models"] = []string{}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	sort.Strings(models)
	resp["status"] = "ready"
	resp["models"] = models
	writeJSON(w, http.StatusOK, resp)
}

func normalizeOllamaModel(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if s == "qwen-coder:30b" {
		return "qwen3-coder:30b"
	}
	return s
}

func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := s.st.DB.QueryContext(ctx, `
SELECT id, path, created_at, last_scan_started_at, last_scan_completed_at, last_error
FROM sources ORDER BY id`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type src struct {
		ID                  int64   `json:"id"`
		Path                string  `json:"path"`
		CreatedAtMs         int64   `json:"createdAtMs"`
		LastScanStartedMs   *int64  `json:"lastScanStartedMs"`
		LastScanCompletedMs *int64  `json:"lastScanCompletedMs"`
		LastError           *string `json:"lastError"`
	}
	var out []src
	for rows.Next() {
		var srow src
		var ls, lc sql.NullInt64
		var le sql.NullString
		if err := rows.Scan(&srow.ID, &srow.Path, &srow.CreatedAtMs, &ls, &lc, &le); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ls.Valid {
			v := ls.Int64
			srow.LastScanStartedMs = &v
		}
		if lc.Valid {
			v := lc.Int64
			srow.LastScanCompletedMs = &v
		}
		if le.Valid {
			v := le.String
			srow.LastError = &v
		}
		out = append(out, srow)
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": out})
}

type addSourceBody struct {
	Path string `json:"path"`
}

func canonicalExistingDir(path string) (string, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", fmt.Errorf("path required")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	resolved = filepath.Clean(resolved)
	fi, err := os.Stat(resolved)
	if err != nil || !fi.IsDir() {
		return "", fmt.Errorf("not a directory")
	}
	if isFilesystemRoot(resolved) {
		return "", fmt.Errorf("filesystem root cannot be indexed as a source")
	}
	return resolved, nil
}

func isFilesystemRoot(path string) bool {
	clean := filepath.Clean(path)
	parent := filepath.Dir(clean)
	return clean == parent
}

func (s *Server) admittedSourcePath(ctx context.Context, raw string) (string, int, string) {
	resolved, err := canonicalExistingDir(raw)
	if err != nil {
		if err.Error() == "path required" || err.Error() == "not a directory" {
			return "", http.StatusBadRequest, err.Error()
		}
		return "", http.StatusBadRequest, err.Error()
	}
	decision, _, err := s.permissions.Check(ctx, permissions.CheckRequest{
		ToolID:    "fs.read",
		Action:    "source.index",
		Paths:     []string{resolved},
		Reads:     true,
		RiskClass: "low",
	})
	if err != nil {
		return "", http.StatusInternalServerError, err.Error()
	}
	if decision == nil || !decision.Allowed {
		reason := "source path denied by active permission profile"
		if decision != nil && strings.TrimSpace(decision.Reason) != "" {
			reason = decision.Reason
		}
		return "", http.StatusForbidden, reason
	}
	if decision.RequiresApproval {
		reason := strings.TrimSpace(decision.Reason)
		if reason == "" {
			reason = "source path outside active read scope"
		}
		return "", http.StatusForbidden, "source path outside active read scope: " + reason
	}
	return resolved, http.StatusOK, ""
}

func (s *Server) handleAddSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body addSourceBody
	if err := decodeServerJSONBody(r, &body); err != nil {
		writeServerDecodeError(w, err)
		return
	}
	abs, status, reason := s.admittedSourcePath(ctx, body.Path)
	if status != http.StatusOK {
		http.Error(w, reason, status)
		return
	}
	res, err := s.st.DB.ExecContext(ctx,
		`INSERT INTO sources (path, created_at) VALUES (?, ?)`,
		abs, time.Now().UnixMilli(),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			http.Error(w, "source already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	_ = s.log.Emit(ctx, "source.added", map[string]any{"sourceId": id, "path": abs})
	if err := s.ingest.IndexSource(ctx, id, abs); err != nil {
		_ = s.log.Emit(ctx, "error.raised", map[string]any{"where": "ingest after add", "message": err.Error(), "sourceId": id})
	}
	if s.watch != nil {
		_ = s.watch.SyncSources(ctx, listSourcePaths(s.st.DB))
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "path": abs})
}

func (s *Server) handleDeleteSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if _, err := s.st.DB.ExecContext(ctx, `DELETE FROM sources WHERE id = ?`, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.log.Emit(ctx, "command.executed", map[string]any{"command": "source.delete", "sourceId": id})
	if s.watch != nil {
		_ = s.watch.SyncSources(ctx, listSourcePaths(s.st.DB))
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleReindex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query().Get("sourceId")
	_ = s.log.Emit(ctx, "command.executed", map[string]any{"command": "reindex", "sourceId": q})
	if q == "" {
		if err := s.ingest.IndexAllSources(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "scope": "all"})
		return
	}
	id, err := strconv.ParseInt(q, 10, 64)
	if err != nil {
		http.Error(w, "bad sourceId", http.StatusBadRequest)
		return
	}
	var path string
	if err := s.st.DB.QueryRowContext(ctx, `SELECT path FROM sources WHERE id = ?`, id).Scan(&path); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := s.ingest.IndexSource(ctx, id, path); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "scope": "one", "sourceId": id})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	hits, err := s.search.Search(ctx, q, limit)
	if err != nil {
		_ = s.log.Emit(ctx, "error.raised", map[string]any{"where": "search", "message": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = s.log.Emit(ctx, "search.executed", map[string]any{"q": q, "hits": len(hits)})
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}

func (s *Server) handleChunk(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	h, err := s.search.ChunkByID(ctx, id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, h)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := s.log.Recent(ctx, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": rows})
}

func (s *Server) handleAdapters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	list := s.adapters.List(ctx)
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	writeJSON(w, http.StatusOK, map[string]any{"adapters": list})
}

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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type requestAuditMeta struct {
	CorrelationID string
	TraceID       string
	WorkspaceID   string
}

const (
	runtimeGPUEnabledKey              = "runtime_gpu_enabled"
	runtimeNVIDIADCGMEnabledKey       = "runtime_nvidia_dcgm_enabled"
	runtimeIntelLevelZeroEnabledKey   = "runtime_intel_level_zero_enabled"
	runtimeAllowOllamaCloudModelsKey  = "modelruntime_allow_ollama_cloud_models"
	shadowModeEnabledKey              = "forge_k_shadow_mode_enabled"
	shadowChatMetadataEnabledKey      = "forge_k_shadow_chat_metadata_enabled"
	shadowRetrievalMetadataEnabledKey = "forge_k_shadow_retrieval_metadata_enabled"
)

func runtimeConfigFromSettings(db *sql.DB, cfg config.Config) config.Config {
	cfg.GPUEnabled = parseRemoteBool(loadSetting(db, runtimeGPUEnabledKey, strconv.FormatBool(cfg.GPUEnabled)))
	cfg.NVIDIADCGMEnabled = parseRemoteBool(loadSetting(db, runtimeNVIDIADCGMEnabledKey, strconv.FormatBool(cfg.NVIDIADCGMEnabled)))
	cfg.IntelLevelZeroEnabled = parseRemoteBool(loadSetting(db, runtimeIntelLevelZeroEnabledKey, strconv.FormatBool(cfg.IntelLevelZeroEnabled)))
	cfg.ModelRuntimeAllowOllamaCloudModels = parseRemoteBool(loadSetting(db, runtimeAllowOllamaCloudModelsKey, strconv.FormatBool(cfg.ModelRuntimeAllowOllamaCloudModels)))
	cfg.ForgeKShadowModeEnabled = parseRemoteBool(loadSetting(db, shadowModeEnabledKey, strconv.FormatBool(cfg.ForgeKShadowModeEnabled)))
	cfg.ForgeKShadowChatMetadataEnabled = parseRemoteBool(loadSetting(db, shadowChatMetadataEnabledKey, strconv.FormatBool(cfg.ForgeKShadowChatMetadataEnabled)))
	cfg.ForgeKShadowRetrievalMetadataEnabled = parseRemoteBool(loadSetting(db, shadowRetrievalMetadataEnabledKey, strconv.FormatBool(cfg.ForgeKShadowRetrievalMetadataEnabled)))
	return cfg
}

func runtimeControlsFromSettings(db *sql.DB, cfg config.Config) map[string]any {
	effective := runtimeConfigFromSettings(db, cfg)
	return map[string]any{
		"gpuEnabled":              effective.GPUEnabled,
		"nvidiaDcgmEnabled":       effective.NVIDIADCGMEnabled,
		"intelLevelZeroEnabled":   effective.IntelLevelZeroEnabled,
		"allowOllamaCloudModels":  effective.ModelRuntimeAllowOllamaCloudModels,
		"safeModeForceCpuOnly":    effective.SafeModeForceCPUOnly,
		"effectiveGpuEnabled":     effective.GPUEnabled && !effective.SafeModeForceCPUOnly,
		"cloudModelsDefaultState": map[bool]string{true: "enabled", false: "disabled"}[effective.ModelRuntimeAllowOllamaCloudModels],
	}
}

func shadowModeFromSettings(db *sql.DB, cfg config.Config) map[string]any {
	effective := runtimeConfigFromSettings(db, cfg)
	return map[string]any{
		"enabled":                  effective.ForgeKShadowModeEnabled,
		"chatMetadataEnabled":      effective.ForgeKShadowChatMetadataEnabled,
		"retrievalMetadataEnabled": effective.ForgeKShadowRetrievalMetadataEnabled,
	}
}

func (s *Server) patchShadowMode(ctx context.Context, body map[string]any) error {
	raw, ok := body["shadowMode"]
	if !ok {
		return nil
	}
	shadowMode, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("shadowMode must be an object")
	}
	changed := false
	for _, item := range []struct {
		bodyKey    string
		settingKey string
	}{
		{bodyKey: "enabled", settingKey: shadowModeEnabledKey},
		{bodyKey: "chatMetadataEnabled", settingKey: shadowChatMetadataEnabledKey},
		{bodyKey: "retrievalMetadataEnabled", settingKey: shadowRetrievalMetadataEnabledKey},
	} {
		if v, exists := shadowMode[item.bodyKey]; exists {
			if err := upsertSetting(ctx, s.st.DB, item.settingKey, parseRemoteBoolValue(v)); err != nil {
				return err
			}
			changed = true
		}
	}
	if changed {
		s.reloadShadowMode(ctx)
	}
	return nil
}

func (s *Server) reloadShadowMode(ctx context.Context) {
	s.cfg = runtimeConfigFromSettings(s.st.DB, s.cfg)
	if s.cfg.ForgeKShadowModeEnabled {
		s.forgeKShadow = forgekshadow.NewObserver(forgekshadow.Config{
			Enabled:                      true,
			ChatMetadataEnabled:          s.cfg.ForgeKShadowChatMetadataEnabled,
			RetrievalMetadataEnabled:     s.cfg.ForgeKShadowRetrievalMetadataEnabled,
			AdvisoryEnabled:              s.cfg.ForgeKShadowAdvisoryEnabled,
			ControlLaneValidationEnabled: s.cfg.ForgeKShadowControlLaneValidationEnabled,
		})
	} else {
		s.forgeKShadow = nil
	}
	_ = s.log.Emit(ctx, "shadow.controls.reloaded", map[string]any{
		"enabled":                  s.cfg.ForgeKShadowModeEnabled,
		"chatMetadataEnabled":      s.cfg.ForgeKShadowChatMetadataEnabled,
		"retrievalMetadataEnabled": s.cfg.ForgeKShadowRetrievalMetadataEnabled,
		"advisoryEnabled":          s.cfg.ForgeKShadowAdvisoryEnabled,
	})
}

func (s *Server) patchRuntimeControls(ctx context.Context, body map[string]any) error {
	raw, ok := body["runtimeControls"]
	if !ok {
		return nil
	}
	controls, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("runtimeControls must be an object")
	}
	changed := false
	for _, item := range []struct {
		bodyKey    string
		settingKey string
	}{
		{bodyKey: "gpuEnabled", settingKey: runtimeGPUEnabledKey},
		{bodyKey: "nvidiaDcgmEnabled", settingKey: runtimeNVIDIADCGMEnabledKey},
		{bodyKey: "intelLevelZeroEnabled", settingKey: runtimeIntelLevelZeroEnabledKey},
		{bodyKey: "allowOllamaCloudModels", settingKey: runtimeAllowOllamaCloudModelsKey},
	} {
		if v, exists := controls[item.bodyKey]; exists {
			if err := upsertSetting(ctx, s.st.DB, item.settingKey, parseRemoteBoolValue(v)); err != nil {
				return err
			}
			changed = true
		}
	}
	if changed {
		s.reloadRuntimeControls(ctx)
	}
	return nil
}

func (s *Server) reloadRuntimeControls(ctx context.Context) {
	s.cfg = runtimeConfigFromSettings(s.st.DB, s.cfg)
	s.gpuTelemetry = gpu.New(gpu.Options{
		Enabled:                 s.cfg.NVIDIADCGMEnabled && !s.cfg.SafeModeForceCPUOnly,
		Endpoint:                s.cfg.NVIDIADCGMEndpoint,
		Timeout:                 time.Duration(s.cfg.NVIDIADCGMTimeoutMs) * time.Millisecond,
		MemoryPressureThreshold: s.cfg.GPUBackgroundMemoryPressureBlockThreshold,
	})
	s.intelTelemetry = gpu.NewIntel(gpu.IntelOptions{
		Enabled:     s.cfg.IntelLevelZeroEnabled && !s.cfg.SafeModeForceCPUOnly,
		ZEInfoPath:  s.cfg.IntelLevelZeroZEInfoPath,
		IntelGPUTop: s.cfg.IntelGPUTopPath,
		Timeout:     time.Duration(s.cfg.IntelGPUTelemetryTimeoutMs) * time.Millisecond,
	})
	s.modelRuntime = initModelRuntimeService(s.cfg, s.auditSvc, s.gpuTelemetry, s.intelTelemetry)
	_ = s.log.Emit(ctx, "runtime.controls.reloaded", map[string]any{
		"gpuEnabled":             s.cfg.GPUEnabled,
		"nvidiaDcgmEnabled":      s.cfg.NVIDIADCGMEnabled,
		"intelLevelZeroEnabled":  s.cfg.IntelLevelZeroEnabled,
		"allowOllamaCloudModels": s.cfg.ModelRuntimeAllowOllamaCloudModels,
	})
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func profileIDOrEmpty(p *permissions.Profile) string {
	if p == nil {
		return ""
	}
	return p.ID
}

func loadSetting(db *sql.DB, key, def string) string {
	var v string
	err := db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil {
		return def
	}
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func upsertSetting(ctx context.Context, db *sql.DB, key, val string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, val,
	)
	return err
}

func listSourcePaths(db *sql.DB) []string {
	rows, err := db.Query(`SELECT path FROM sources ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return out
		}
		out = append(out, p)
	}
	return out
}
