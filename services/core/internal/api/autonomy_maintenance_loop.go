package api

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"forge/projectforge/services/core/internal/aios/autonomy"
	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/aios/truth"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/events"
	"forge/projectforge/services/core/internal/memory"
)

type AutonomyMaintenanceLoopOptions struct {
	DB                  *sql.DB
	Events              *events.Logger
	Memory              *memory.Service
	Runner              *autonomy.SelfInitiatedSyscallRunner
	RuleRuntime         *autonomy.RuleAgentRuntime
	Truth               *truth.Engine
	Charters            autonomy.CharterRepository
	Intents             autonomy.IntentRepository
	Budgets             autonomy.BudgetRepository
	Decisions           autonomy.DecisionRepository
	Explainer           *autonomy.ExplanationService
	Scope               domain.ForgeScope
	Mode                domain.AutonomyMode
	NowMillis           func() int64
	IdleAfter           time.Duration
	TickEvery           time.Duration
	MaintenanceCooldown time.Duration
	ImprovementCooldown time.Duration
	HasInflightWork     func() bool
	RunMaintenance      func(ctx context.Context, idleReason string) error
	RunImprovement      func(ctx context.Context, idleReason string) error
}

type AutonomyDreamStatus struct {
	Active             bool   `json:"active"`
	EnteredAt          int64  `json:"enteredAt,omitempty"`
	LastReason         string `json:"lastReason,omitempty"`
	LastTickAt         int64  `json:"lastTickAt,omitempty"`
	LastMaintenanceAt  int64  `json:"lastMaintenanceAt,omitempty"`
	LastImprovementAt  int64  `json:"lastImprovementAt,omitempty"`
	LastError          string `json:"lastError,omitempty"`
	LastTransitionType string `json:"lastTransitionType,omitempty"`
}

type AutonomyMaintenanceLoop struct {
	db        *sql.DB
	events    *events.Logger
	memory    *memory.Service
	runner    *autonomy.SelfInitiatedSyscallRunner
	runtime   *autonomy.RuleAgentRuntime
	truth     *truth.Engine
	charters  autonomy.CharterRepository
	intents   autonomy.IntentRepository
	budgets   autonomy.BudgetRepository
	decisions autonomy.DecisionRepository
	explainer *autonomy.ExplanationService
	scope     domain.ForgeScope
	mode      domain.AutonomyMode

	nowMillis           func() int64
	idleAfter           time.Duration
	tickEvery           time.Duration
	maintenanceCooldown time.Duration
	improvementCooldown time.Duration
	hasInflightWork     func() bool

	runMaintenance func(ctx context.Context, idleReason string) error
	runImprovement func(ctx context.Context, idleReason string) error

	stopOnce sync.Once
	stopCh   chan struct{}

	mu                   sync.Mutex
	dreamActive          bool
	dreamEnteredAt       int64
	lastReason           string
	lastTickAt           int64
	lastMaintenanceAt    int64
	lastImprovementAt    int64
	lastError            string
	lastTransitionReason string
}

func NewAutonomyMaintenanceLoop(opts AutonomyMaintenanceLoopOptions) *AutonomyMaintenanceLoop {
	nowFn := opts.NowMillis
	if nowFn == nil {
		nowFn = domain.NowMillis
	}
	mode := opts.Mode
	if mode == "" {
		mode = domain.AutonomyModeMaintain
	}
	idleAfter := opts.IdleAfter
	if idleAfter <= 0 {
		idleAfter = 3 * time.Minute
	}
	tickEvery := opts.TickEvery
	if tickEvery <= 0 {
		tickEvery = 45 * time.Second
	}
	maintenanceCooldown := opts.MaintenanceCooldown
	if maintenanceCooldown <= 0 {
		maintenanceCooldown = 20 * time.Minute
	}
	improvementCooldown := opts.ImprovementCooldown
	if improvementCooldown <= 0 {
		improvementCooldown = 45 * time.Minute
	}
	loop := &AutonomyMaintenanceLoop{
		db:                  opts.DB,
		events:              opts.Events,
		memory:              opts.Memory,
		runner:              opts.Runner,
		runtime:             opts.RuleRuntime,
		truth:               opts.Truth,
		charters:            opts.Charters,
		intents:             opts.Intents,
		budgets:             opts.Budgets,
		decisions:           opts.Decisions,
		explainer:           opts.Explainer,
		scope:               opts.Scope,
		mode:                mode,
		nowMillis:           nowFn,
		idleAfter:           idleAfter,
		tickEvery:           tickEvery,
		maintenanceCooldown: maintenanceCooldown,
		improvementCooldown: improvementCooldown,
		hasInflightWork:     opts.HasInflightWork,
		runMaintenance:      opts.RunMaintenance,
		runImprovement:      opts.RunImprovement,
		stopCh:              make(chan struct{}),
	}
	if loop.runMaintenance == nil {
		loop.runMaintenance = loop.runMaintenancePass
	}
	if loop.runImprovement == nil {
		loop.runImprovement = loop.runImprovementPass
	}
	return loop
}

func (l *AutonomyMaintenanceLoop) Stop() {
	if l == nil {
		return
	}
	l.stopOnce.Do(func() {
		close(l.stopCh)
	})
}

func (l *AutonomyMaintenanceLoop) Run(ctx context.Context) {
	if l == nil {
		return
	}
	if l.tickEvery <= 0 {
		l.tickEvery = 45 * time.Second
	}
	_ = l.emit("autonomy.dream_loop.started", map[string]any{
		"workspaceId": l.scope.WorkspaceID,
		"laneId":      l.scope.LaneID,
		"mode":        l.mode,
		"idleAfterMs": int64(l.idleAfter / time.Millisecond),
		"tickEveryMs": int64(l.tickEvery / time.Millisecond),
	})
	ticker := time.NewTicker(l.tickEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = l.emit("autonomy.dream_loop.stopped", map[string]any{
				"workspaceId": l.scope.WorkspaceID,
				"reason":      "context_cancelled",
			})
			return
		case <-l.stopCh:
			_ = l.emit("autonomy.dream_loop.stopped", map[string]any{
				"workspaceId": l.scope.WorkspaceID,
				"reason":      "manual_stop",
			})
			return
		case <-ticker.C:
			_ = l.Tick(ctx)
		}
	}
}

func (l *AutonomyMaintenanceLoop) Tick(ctx context.Context) error {
	if l == nil {
		return nil
	}
	now := l.nowMillis()
	idle, reason, err := l.isIdle(ctx, now)
	l.mu.Lock()
	l.lastTickAt = now
	if err != nil {
		l.lastError = err.Error()
		l.mu.Unlock()
		_ = l.emit("autonomy.dream_loop.error", map[string]any{
			"workspaceId": l.scope.WorkspaceID,
			"error":       err.Error(),
		})
		return err
	}
	wasActive := l.dreamActive
	if idle && !l.dreamActive {
		l.dreamActive = true
		l.dreamEnteredAt = now
		l.lastTransitionReason = "entered"
	}
	if !idle && l.dreamActive {
		l.dreamActive = false
		l.lastTransitionReason = "exited"
	}
	l.lastReason = reason
	nextMaintenance := l.dreamActive && (l.lastMaintenanceAt == 0 || now-l.lastMaintenanceAt >= int64(l.maintenanceCooldown/time.Millisecond))
	nextImprovement := l.dreamActive && (l.lastImprovementAt == 0 || now-l.lastImprovementAt >= int64(l.improvementCooldown/time.Millisecond))
	l.mu.Unlock()

	if idle && !wasActive {
		_ = l.emit("autonomy.dream_state.entered", map[string]any{
			"workspaceId": l.scope.WorkspaceID,
			"reason":      reason,
			"atMs":        now,
		})
	}
	if !idle && wasActive {
		_ = l.emit("autonomy.dream_state.exited", map[string]any{
			"workspaceId": l.scope.WorkspaceID,
			"reason":      reason,
			"atMs":        now,
		})
		return nil
	}
	if !idle {
		return nil
	}

	if nextMaintenance {
		if runErr := l.runMaintenance(ctx, reason); runErr != nil {
			l.mu.Lock()
			l.lastError = runErr.Error()
			l.mu.Unlock()
			_ = l.emit("autonomy.dream_loop.maintenance_failed", map[string]any{
				"workspaceId": l.scope.WorkspaceID,
				"error":       runErr.Error(),
			})
		} else {
			l.mu.Lock()
			l.lastMaintenanceAt = now
			l.mu.Unlock()
		}
	}

	if nextImprovement {
		if runErr := l.runImprovement(ctx, reason); runErr != nil {
			l.mu.Lock()
			l.lastError = runErr.Error()
			l.mu.Unlock()
			_ = l.emit("autonomy.dream_loop.improvement_failed", map[string]any{
				"workspaceId": l.scope.WorkspaceID,
				"error":       runErr.Error(),
			})
		} else {
			l.mu.Lock()
			l.lastImprovementAt = now
			l.mu.Unlock()
		}
	}

	return nil
}

func (l *AutonomyMaintenanceLoop) Status() AutonomyDreamStatus {
	if l == nil {
		return AutonomyDreamStatus{}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return AutonomyDreamStatus{
		Active:             l.dreamActive,
		EnteredAt:          l.dreamEnteredAt,
		LastReason:         l.lastReason,
		LastTickAt:         l.lastTickAt,
		LastMaintenanceAt:  l.lastMaintenanceAt,
		LastImprovementAt:  l.lastImprovementAt,
		LastError:          l.lastError,
		LastTransitionType: l.lastTransitionReason,
	}
}

func (l *AutonomyMaintenanceLoop) Scope() domain.ForgeScope {
	if l == nil {
		return domain.ForgeScope{}
	}
	return l.scope
}

func (l *AutonomyMaintenanceLoop) Mode() domain.AutonomyMode {
	if l == nil {
		return domain.AutonomyModeOff
	}
	return l.mode
}

func (l *AutonomyMaintenanceLoop) ListIntents(ctx context.Context, status string, limit int) ([]domain.AutonomyIntent, error) {
	if l == nil || l.intents == nil {
		return []domain.AutonomyIntent{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	status = strings.TrimSpace(strings.ToLower(status))
	switch status {
	case "":
		return l.intents.ListByScope(ctx, l.scope, limit)
	case "active":
		return l.intents.ListActive(ctx, l.scope, limit)
	default:
		return l.intents.ListByStatus(ctx, l.scope, domain.IntentStatus(status), limit)
	}
}

func (l *AutonomyMaintenanceLoop) ListDecisions(ctx context.Context, limit int) ([]domain.AutonomyDecision, error) {
	if l == nil || l.decisions == nil {
		return []domain.AutonomyDecision{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return l.decisions.ListByScope(ctx, l.scope, limit)
}

func (l *AutonomyMaintenanceLoop) ListBudgets(ctx context.Context) ([]domain.FreedomBudget, error) {
	if l == nil || l.budgets == nil {
		return []domain.FreedomBudget{}, nil
	}
	return l.budgets.ListByScope(ctx, l.scope)
}

func (l *AutonomyMaintenanceLoop) ListCharters(ctx context.Context, activeOnly bool) ([]domain.AutonomyCharter, error) {
	if l == nil || l.charters == nil {
		return []domain.AutonomyCharter{}, nil
	}
	if activeOnly {
		return l.charters.ListActiveByScope(ctx, l.scope, l.nowMillis())
	}
	return l.charters.ListByScope(ctx, l.scope)
}

func (l *AutonomyMaintenanceLoop) ExplainIntent(ctx context.Context, intentID string) (map[string]any, error) {
	if l == nil || l.explainer == nil {
		return map[string]any{"intentId": strings.TrimSpace(intentID), "warning": "autonomy explainer unavailable"}, nil
	}
	return l.explainer.ExplainIntent(ctx, strings.TrimSpace(intentID))
}

func (l *AutonomyMaintenanceLoop) runMaintenancePass(ctx context.Context, idleReason string) error {
	now := l.nowMillis()
	if l.memory != nil {
		_, err := l.memory.RunRepairPass(ctx, memory.RunRepairRequest{
			Mode:       "autonomy_dream_maintenance",
			MaxAgeDays: 14,
			Limit:      120,
			Note:       "idle dream-state maintenance: " + idleReason,
		})
		if err != nil {
			return err
		}
	}
	if l.runtime == nil || l.truth == nil {
		_ = l.emit("autonomy.dream_loop.maintenance", map[string]any{
			"workspaceId": l.scope.WorkspaceID,
			"note":        "memory repair completed; autonomy runtime not configured",
		})
		return nil
	}
	input := autonomy.BuildRuleAgentInput(
		l.scope,
		fmt.Sprintf("corr-dream-maintenance-%d", now),
		fmt.Sprintf("trace-dream-maintenance-%d", now),
		l.truth,
		0,
		"dream_state_idle",
		now,
	)
	runs, err := l.runtime.RunOnce(ctx, input)
	if err != nil {
		return err
	}
	committed := 0
	for _, run := range runs {
		committed += len(run.CommittedObjectIDs)
	}
	_ = l.emit("autonomy.dream_loop.maintenance", map[string]any{
		"workspaceId":  l.scope.WorkspaceID,
		"autonomyRuns": len(runs),
		"committedIds": committed,
	})
	return nil
}

func (l *AutonomyMaintenanceLoop) runImprovementPass(ctx context.Context, idleReason string) error {
	if l.runner == nil || l.truth == nil {
		return nil
	}
	now := l.nowMillis()
	activeLoops, _ := l.truth.ListActiveLoops(ctx, l.scope, 100)
	blockedLoops, _ := l.truth.ListBlockedLoops(ctx, l.scope, 100)
	staleLoops, _ := l.truth.ListStaleLoops(ctx, l.scope, now-int64((72*time.Hour)/time.Millisecond), 100)
	contradictions, _ := l.truth.ListContradictionsByScope(ctx, l.scope, 100)

	summary := fmt.Sprintf(
		"Dream-state improvement snapshot. active_loops=%d blocked_loops=%d stale_loops=%d contradictions=%d reason=%s",
		len(activeLoops),
		len(blockedLoops),
		len(staleLoops),
		len(contradictions),
		idleReason,
	)
	correlationID := fmt.Sprintf("corr-dream-improvement-%d", now)
	traceID := fmt.Sprintf("trace-dream-improvement-%d", now)
	intentID := "intent-dream-improvement-" + shortHashLocal(fmt.Sprintf("%d", now), l.scope.WorkspaceID)

	intent := domain.AutonomyIntent{
		ID:            intentID,
		Type:          domain.IntentContextPreparation,
		Title:         "Dream-state improvement pass",
		Description:   "Generate a bounded improvement snapshot while FORGE is idle.",
		Source:        domain.IntentSourceForge,
		ProposedBy:    "forge.autonomy.dream_loop",
		Scope:         l.scope,
		Status:        domain.IntentStatusProposed,
		Risk:          domain.AutonomyRiskLow,
		AutonomyLevel: domain.AutonomyLevelAutoCommitSafe,
		CharterID:     "charter_context_preparation",
		BudgetID:      "budget_context_prep",
		Evidence:      []string{},
		Provenance: domain.Provenance{
			Actor:     "forge.autonomy.dream_loop",
			ActorType: "system",
			Source:    "autonomy.dream",
			TraceID:   traceID,
		},
		CorrelationID: correlationID,
		TraceID:       traceID,
		CreatedAt:     now,
		UpdatedAt:     now,
		Metadata: map[string]any{
			"dreamState": true,
			"idleReason": idleReason,
		},
	}

	noteID := "note-dream-improvement-" + shortHashLocal(l.scope.WorkspaceID, fmt.Sprintf("%d", now/int64((15*time.Minute)/time.Millisecond)))
	actions := []domain.SyscallRequest{
		{
			ID:     "action-dream-context-" + shortHashLocal(intentID, "context"),
			Action: domain.ActionCompileContext,
			Actor:  domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"},
			Source: domain.SourceSystem,
			Scope:  l.scope,
			Payload: map[string]any{
				"query": "dream-state improvement snapshot",
				"budget": map[string]any{
					"maxTokens": 1200,
					"maxEvents": 30,
					"maxNotes":  30,
				},
			},
			Provenance:    intent.Provenance,
			CorrelationID: correlationID,
			TraceID:       traceID,
			RequestedAt:   now,
		},
		{
			ID:     "action-dream-note-" + shortHashLocal(intentID, "note"),
			Action: domain.ActionCreateNote,
			Actor:  domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"},
			Source: domain.SourceSystem,
			Scope:  l.scope,
			Payload: map[string]any{
				"id":         noteID,
				"type":       string(domain.NoteSystem),
				"title":      "Dream-state improvement snapshot",
				"content":    summary,
				"confidence": 0.72,
				"status":     string(domain.NoteActive),
			},
			Provenance:     intent.Provenance,
			CorrelationID:  correlationID,
			TraceID:        traceID,
			RequestedAt:    now,
			IdempotencyKey: "dream-improvement-note:" + noteID,
		},
	}

	run, err := l.runner.Run(ctx, intent, actions, domain.RunModeCommitIfAuthorized, l.mode)
	if err != nil {
		return err
	}
	_ = l.emit("autonomy.dream_loop.improvement", map[string]any{
		"workspaceId":    l.scope.WorkspaceID,
		"decision":       run.Decision,
		"committedCount": len(run.CommittedObjectIDs),
		"warnings":       run.Warnings,
		"errors":         run.Errors,
	})
	return nil
}

func (l *AutonomyMaintenanceLoop) isIdle(ctx context.Context, now int64) (bool, string, error) {
	if l.db == nil {
		return false, "database_unavailable", fmt.Errorf("maintenance loop requires database")
	}
	if l.hasInflightWork != nil && l.hasInflightWork() {
		return false, "assistant_generation_inflight", nil
	}
	var activeJobs int
	if err := l.db.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM jobs
WHERE status IN ('queued', 'preparing', 'awaiting_approval', 'running')
`).Scan(&activeJobs); err != nil {
		return false, "jobs_query_failed", err
	}
	if activeJobs > 0 {
		return false, "active_jobs", nil
	}
	var lastUserMessage sql.NullInt64
	if err := l.db.QueryRowContext(ctx, `
SELECT MAX(created_at) FROM chat_messages WHERE role = 'user'
`).Scan(&lastUserMessage); err != nil {
		return false, "chat_query_failed", err
	}
	if lastUserMessage.Valid {
		idleForMs := now - lastUserMessage.Int64
		if idleForMs < int64(l.idleAfter/time.Millisecond) {
			return false, "recent_user_activity", nil
		}
	}
	return true, "idle_window", nil
}

func (l *AutonomyMaintenanceLoop) emit(kind string, payload map[string]any) error {
	if l.events == nil {
		return nil
	}
	return l.events.Emit(context.Background(), kind, payload)
}

func newDefaultAutonomyMaintenanceLoop(db *sql.DB, cfg config.Config, ev *events.Logger, memorySvc *memory.Service) *AutonomyMaintenanceLoop {
	if db == nil {
		return nil
	}
	if !parseBoolSetting(loadSetting(db, "autonomy_dream_enabled", "true"), true) {
		return nil
	}
	scope := domain.ForgeScope{
		WorkspaceID: strings.TrimSpace(loadSetting(db, "autonomy_workspace_id", defaultAutonomyWorkspaceID(cfg.WorkspaceDir))),
		LaneID:      strings.TrimSpace(loadSetting(db, "autonomy_lane_id", "control.semantic")),
	}
	if scope.WorkspaceID == "" {
		scope.WorkspaceID = defaultAutonomyWorkspaceID(cfg.WorkspaceDir)
	}
	if scope.LaneID == "" {
		scope.LaneID = "control.semantic"
	}

	nowFn := domain.NowMillis
	stateRepo := controllane.NewSQLiteStateRepository(db)
	loopRepo := controllane.NewSQLiteOpenLoopRepository(db)
	noteRepo := controllane.NewSQLiteMemoryNoteRepository(db)
	modelRepo := controllane.NewSQLiteDerivedModelRepository(db)
	contradictionRepo := controllane.NewSQLiteContradictionRepository(db)
	supersessionRepo := controllane.NewSQLiteSupersessionRepository(db)
	txRunner := controllane.NewSQLiteTransactionRunner(db)
	kernel := controllane.NewProcessor(controllane.ProcessorOptions{
		Registry:     controllane.NewStaticActionRegistry(),
		Validator:    controllane.NewDeterministicValidator(),
		Capabilities: controllane.NewStaticCapabilityService(),
		ApprovalGate: controllane.NewStaticApprovalGate(),
		TxRunner:     txRunner,
		AuditSink:    controllane.NewCoreAuditSink(audit.New(db)),
		NowMillis:    nowFn,
	})
	truthEngine := truth.NewEngine(truth.EngineOptions{
		Kernel: kernel,
		Repositories: truth.Repositories{
			State:         stateRepo,
			Loops:         loopRepo,
			Notes:         noteRepo,
			Models:        modelRepo,
			Contradiction: contradictionRepo,
			Supersession:  supersessionRepo,
		},
		NowMillis: nowFn,
	})
	bundle := autonomy.NewSQLiteBundle(db)
	for _, budget := range autonomy.DefaultBudgets(scope, nowFn(), "forge.autonomy") {
		_ = bundle.Budgets.Create(context.Background(), budget)
	}
	for _, charter := range autonomy.DefaultCharters(scope, nowFn(), "forge.autonomy") {
		_ = bundle.Charters.Create(context.Background(), charter)
	}
	intentQueue := autonomy.NewIntentQueueService(bundle.Intents, nowFn)
	budgetSvc := autonomy.NewFreedomBudgetService(bundle.Budgets, bundle.Reservations, nowFn)
	explainer := autonomy.NewExplanationService(bundle.Intents, bundle.Decisions, bundle.Budgets, bundle.Charters)
	evaluator := autonomy.NewAutonomyPolicyEvaluator(autonomy.PolicyEvaluatorOptions{
		Charters:       bundle.Charters,
		Risk:           autonomy.NewDeterministicRiskClassifier(),
		Budgets:        budgetSvc,
		Kernel:         kernel,
		NowMillis:      nowFn,
		CharterService: autonomy.NewCharterService(),
		HasActiveStateDependency: func(scope domain.ForgeScope, objectID string) bool {
			rows, err := stateRepo.ListCurrent(context.Background(), controllane.ScopeFilter{
				WorkspaceID: scope.WorkspaceID,
				LaneID:      scope.LaneID,
			}, 500)
			if err != nil {
				return false
			}
			for _, row := range rows {
				for _, id := range row.DerivedFrom {
					if id == objectID {
						return true
					}
				}
			}
			return false
		},
	})
	runner := autonomy.NewSelfInitiatedSyscallRunner(autonomy.SelfInitiatedRunnerOptions{
		Kernel:    kernel,
		Policy:    evaluator,
		Intents:   intentQueue,
		Decisions: bundle.Decisions,
		Budgets:   budgetSvc,
		Approval:  autonomy.NewStaticApprovalEscalator(),
		NowMillis: nowFn,
	})
	mode := parseAutonomyMode(loadSetting(db, "autonomy_mode", string(domain.AutonomyModeMaintain)))
	runtime := autonomy.NewRuleAgentRuntime(
		[]autonomy.RuleAgent{
			autonomy.OpenLoopStalenessAgent{},
			autonomy.CleanupProposalAgent{},
		},
		runner,
		mode,
		nowFn,
	)
	return NewAutonomyMaintenanceLoop(AutonomyMaintenanceLoopOptions{
		DB:                  db,
		Events:              ev,
		Memory:              memorySvc,
		Runner:              runner,
		RuleRuntime:         runtime,
		Truth:               truthEngine,
		Charters:            bundle.Charters,
		Intents:             bundle.Intents,
		Budgets:             bundle.Budgets,
		Decisions:           bundle.Decisions,
		Explainer:           explainer,
		Scope:               scope,
		Mode:                mode,
		NowMillis:           nowFn,
		IdleAfter:           parseDurationSetting(loadSetting(db, "autonomy_idle_after", "3m"), 3*time.Minute, 30*time.Second, 24*time.Hour),
		TickEvery:           parseDurationSetting(loadSetting(db, "autonomy_dream_tick", "45s"), 45*time.Second, 5*time.Second, 30*time.Minute),
		MaintenanceCooldown: parseDurationSetting(loadSetting(db, "autonomy_maintenance_cooldown", "20m"), 20*time.Minute, 1*time.Minute, 24*time.Hour),
		ImprovementCooldown: parseDurationSetting(loadSetting(db, "autonomy_improvement_cooldown", "45m"), 45*time.Minute, 1*time.Minute, 24*time.Hour),
	})
}

func parseBoolSetting(raw string, fallback bool) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func parseAutonomyMode(raw string) domain.AutonomyMode {
	mode := domain.AutonomyMode(strings.TrimSpace(strings.ToLower(raw)))
	switch mode {
	case domain.AutonomyModeOff, domain.AutonomyModeObserve, domain.AutonomyModePropose, domain.AutonomyModeMaintain, domain.AutonomyModeMission:
		return mode
	default:
		return domain.AutonomyModeMaintain
	}
}

func parseDurationSetting(raw string, fallback, min, max time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	if parsed, err := time.ParseDuration(raw); err == nil {
		return clampDuration(parsed, fallback, min, max)
	}
	if n, err := strconv.Atoi(raw); err == nil {
		return clampDuration(time.Duration(n)*time.Second, fallback, min, max)
	}
	return fallback
}

func clampDuration(v, fallback, min, max time.Duration) time.Duration {
	if v <= 0 {
		return fallback
	}
	if min > 0 && v < min {
		return min
	}
	if max > 0 && v > max {
		return max
	}
	return v
}

func defaultAutonomyWorkspaceID(workspaceDir string) string {
	base := strings.TrimSpace(filepath.Base(filepath.Clean(workspaceDir)))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "workspace:default"
	}
	base = strings.ToLower(base)
	base = strings.ReplaceAll(base, " ", "_")
	return "workspace:" + base
}

func shortHashLocal(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:8])
}
