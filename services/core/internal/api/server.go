package api

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
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
	capStoreOK      bool
	capStoreErr     string
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
	pcSvc.SetAllowedRoots(cfg.ProjectContextAllowedRoots)
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
				apiLogWarn("forge-k shadow diagnostic persistence disabled", apiLogErr(err))
			} else if db, err := sql.Open("pgx", cfg.PostgresDSN); err != nil {
				apiLogWarn("forge-k shadow diagnostic postgres open failed", apiLogErr(err))
			} else if err := store.NewPostgresMigrationRunner(store.PostgresMigrations()).Run(context.Background(), db); err != nil {
				_ = db.Close()
				apiLogWarn("forge-k shadow diagnostic postgres migrations failed", apiLogErr(err))
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
	capabilityOverrideStoreDurable := true
	capabilityOverrideStoreError := ""
	capabilityRegistry, err := gateway.NewToolCapabilityRegistryWithStore(bg, &gateway.SQLiteOverrideStore{DB: st.DB})
	if err != nil {
		apiLogWarn("tool capability override store unavailable; using in-memory registry", apiLogErr(err))
		capabilityOverrideStoreDurable = false
		capabilityOverrideStoreError = err.Error()
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
		apiLogWarn("legacy adapter gateway tool registration failed", apiLogErr(err))
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
		apiLogWarn("watch disabled", apiLogErr(err))
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
		capStoreOK:     capabilityOverrideStoreDurable,
		capStoreErr:    capabilityOverrideStoreError,
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
			apiLogWarn("approval expiry sweep failed", apiLogErr(err))
			return
		}
		if n > 0 {
			apiLogInfo("approval expiry sweep expired requests", slog.Int("count", n))
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
