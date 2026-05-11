package api

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"forge/projectforge/services/core/internal/aios/autonomy"
	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/aios/rulecells"
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
	SweepActive        bool   `json:"sweepActive,omitempty"`
	ActiveSweepID      string `json:"activeSweepId,omitempty"`
	LastSweepAt        int64  `json:"lastSweepAt,omitempty"`
	LastSweepID        string `json:"lastSweepId,omitempty"`
	LastSweepTrigger   string `json:"lastSweepTrigger,omitempty"`
	LastSweepDryRun    bool   `json:"lastSweepDryRun,omitempty"`
	LastSweepStatus    string `json:"lastSweepStatus,omitempty"`
}

type AutonomyMaintenanceSweepRequest struct {
	DryRun bool   `json:"dryRun"`
	Reason string `json:"reason,omitempty"`
}

type AutonomyMaintenanceSweepReport struct {
	SweepID     string                          `json:"sweepId"`
	Scope       domain.ForgeScope               `json:"scope"`
	Trigger     string                          `json:"trigger"`
	DryRun      bool                            `json:"dryRun"`
	Status      string                          `json:"status"`
	IdleReason  string                          `json:"idleReason,omitempty"`
	RequestedAt int64                           `json:"requestedAt"`
	StartedAt   int64                           `json:"startedAt"`
	CompletedAt int64                           `json:"completedAt,omitempty"`
	Maintenance AutonomyMaintenancePhaseReport  `json:"maintenance"`
	Improvement AutonomyMaintenancePhaseReport  `json:"improvement"`
	Warnings    []string                        `json:"warnings,omitempty"`
	Diagnostics []AutonomyMaintenanceDiagnostic `json:"diagnostics,omitempty"`
}

type AutonomyMaintenancePhaseReport struct {
	Name        string                          `json:"name"`
	Status      string                          `json:"status"`
	DryRun      bool                            `json:"dryRun"`
	Actions     []AutonomyMaintenanceAction     `json:"actions,omitempty"`
	Warnings    []string                        `json:"warnings,omitempty"`
	Diagnostics []AutonomyMaintenanceDiagnostic `json:"diagnostics,omitempty"`
	Summary     map[string]any                  `json:"summary,omitempty"`
}

type AutonomyMaintenanceAction struct {
	ID          string         `json:"id,omitempty"`
	Kind        string         `json:"kind"`
	Summary     string         `json:"summary"`
	WouldCommit bool           `json:"wouldCommit"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type AutonomyMaintenanceDiagnostic struct {
	Code     string         `json:"code"`
	Severity string         `json:"severity"`
	Message  string         `json:"message"`
	Action   string         `json:"action,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
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

	runMaintenance    func(ctx context.Context, idleReason string) error
	runImprovement    func(ctx context.Context, idleReason string) error
	customMaintenance bool
	customImprovement bool

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
	sweepRunning         bool
	activeSweepID        string
	lastSweepAt          int64
	lastSweepID          string
	lastSweepTrigger     string
	lastSweepDryRun      bool
	lastSweepStatus      string
}

func NewAutonomyMaintenanceLoop(opts AutonomyMaintenanceLoopOptions) *AutonomyMaintenanceLoop {
	nowFn := opts.NowMillis
	if nowFn == nil {
		nowFn = domain.NowMillis
	}
	mode := opts.Mode
	if mode == "" {
		mode = domain.AutonomyModeObserve
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
		customMaintenance:   opts.RunMaintenance != nil,
		customImprovement:   opts.RunImprovement != nil,
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
	if !nextMaintenance && !nextImprovement {
		return nil
	}
	report, runErr := l.runSweepInternal(ctx, "scheduler", reason, false, nextMaintenance, nextImprovement, true)
	if runErr != nil {
		l.mu.Lock()
		l.lastError = runErr.Error()
		l.mu.Unlock()
		return runErr
	}
	for _, diag := range report.Diagnostics {
		if diag.Severity == "error" {
			l.mu.Lock()
			l.lastError = diag.Message
			l.mu.Unlock()
			break
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
		SweepActive:        l.sweepRunning,
		ActiveSweepID:      l.activeSweepID,
		LastSweepAt:        l.lastSweepAt,
		LastSweepID:        l.lastSweepID,
		LastSweepTrigger:   l.lastSweepTrigger,
		LastSweepDryRun:    l.lastSweepDryRun,
		LastSweepStatus:    l.lastSweepStatus,
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

func (l *AutonomyMaintenanceLoop) RunSweep(ctx context.Context, req AutonomyMaintenanceSweepRequest) (AutonomyMaintenanceSweepReport, error) {
	if l == nil {
		return AutonomyMaintenanceSweepReport{}, fmt.Errorf("autonomy maintenance loop is not configured")
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "manual_request"
	}
	return l.runSweepInternal(ctx, "manual_api", reason, req.DryRun, true, true, !req.DryRun)
}

func (l *AutonomyMaintenanceLoop) runSweepInternal(ctx context.Context, trigger, idleReason string, dryRun, includeMaintenance, includeImprovement, advanceTimestamps bool) (AutonomyMaintenanceSweepReport, error) {
	now := l.nowMillis()
	report := AutonomyMaintenanceSweepReport{
		Scope:       l.scope,
		Trigger:     strings.TrimSpace(trigger),
		DryRun:      dryRun,
		Status:      "running",
		IdleReason:  strings.TrimSpace(idleReason),
		RequestedAt: now,
		StartedAt:   now,
		Maintenance: AutonomyMaintenancePhaseReport{Name: "maintenance", Status: "skipped", DryRun: dryRun, Summary: map[string]any{}},
		Improvement: AutonomyMaintenancePhaseReport{Name: "improvement", Status: "skipped", DryRun: dryRun, Summary: map[string]any{}},
		Warnings:    []string{},
		Diagnostics: []AutonomyMaintenanceDiagnostic{},
	}
	if report.Trigger == "" {
		report.Trigger = "manual_api"
	}
	sweepID, ok := l.beginSweep(report.Trigger, dryRun, now)
	report.SweepID = sweepID
	if !ok {
		report.Status = "skipped"
		report.CompletedAt = l.nowMillis()
		report.Diagnostics = append(report.Diagnostics, AutonomyMaintenanceDiagnostic{
			Code:     "SWEEP_IN_PROGRESS",
			Severity: "warning",
			Message:  "maintenance sweep skipped because another sweep is already active",
			Action:   "wait for the active sweep to complete before retrying",
			Metadata: map[string]any{"activeSweepId": sweepID},
		})
		_ = l.emit("autonomy.dream_loop.sweep_skipped", map[string]any{
			"workspaceId":   l.scope.WorkspaceID,
			"trigger":       report.Trigger,
			"dryRun":        dryRun,
			"activeSweepId": sweepID,
			"reason":        "sweep_in_progress",
		})
		return report, nil
	}
	defer func() {
		l.finishSweep(report)
	}()

	_ = l.emit("autonomy.dream_loop.sweep_started", map[string]any{
		"workspaceId": l.scope.WorkspaceID,
		"laneId":      l.scope.LaneID,
		"sweepId":     sweepID,
		"trigger":     report.Trigger,
		"dryRun":      dryRun,
		"idleReason":  report.IdleReason,
	})

	var firstErr error
	if includeMaintenance {
		phase, err := l.executeMaintenancePhase(ctx, report.IdleReason, dryRun)
		report.Maintenance = phase
		if err != nil && firstErr == nil {
			firstErr = err
		}
		if advanceTimestamps && phase.Status == "completed" {
			l.mu.Lock()
			l.lastMaintenanceAt = now
			l.mu.Unlock()
		}
	}
	if includeImprovement {
		phase, err := l.executeImprovementPhase(ctx, report.IdleReason, dryRun)
		report.Improvement = phase
		if err != nil && firstErr == nil {
			firstErr = err
		}
		if advanceTimestamps && phase.Status == "completed" {
			l.mu.Lock()
			l.lastImprovementAt = now
			l.mu.Unlock()
		}
	}

	report.Diagnostics = append(report.Diagnostics, report.Maintenance.Diagnostics...)
	report.Diagnostics = append(report.Diagnostics, report.Improvement.Diagnostics...)
	report.Warnings = append(report.Warnings, report.Maintenance.Warnings...)
	report.Warnings = append(report.Warnings, report.Improvement.Warnings...)
	report.CompletedAt = l.nowMillis()
	report.Status = maintenanceSweepStatus(report.Diagnostics, firstErr)
	if firstErr != nil {
		report.Diagnostics = append(report.Diagnostics, AutonomyMaintenanceDiagnostic{
			Code:     "SWEEP_EXECUTION_FAILED",
			Severity: "error",
			Message:  firstErr.Error(),
			Action:   "inspect phase diagnostics and retry after the underlying error is resolved",
		})
	}
	_ = l.emit("autonomy.dream_loop.sweep_completed", map[string]any{
		"workspaceId":        l.scope.WorkspaceID,
		"laneId":             l.scope.LaneID,
		"sweepId":            report.SweepID,
		"trigger":            report.Trigger,
		"dryRun":             report.DryRun,
		"status":             report.Status,
		"maintenanceStatus":  report.Maintenance.Status,
		"improvementStatus":  report.Improvement.Status,
		"diagnosticCount":    len(report.Diagnostics),
		"maintenanceActions": len(report.Maintenance.Actions),
		"improvementActions": len(report.Improvement.Actions),
	})
	return report, firstErr
}

func (l *AutonomyMaintenanceLoop) beginSweep(trigger string, dryRun bool, startedAt int64) (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sweepRunning {
		return l.activeSweepID, false
	}
	sweepID := fmt.Sprintf("sweep-%s-%d-%s", strings.ReplaceAll(trigger, "_", "-"), startedAt, shortHashLocal(l.scope.WorkspaceID, trigger, fmt.Sprintf("%d", startedAt), strconv.FormatBool(dryRun)))
	l.sweepRunning = true
	l.activeSweepID = sweepID
	return sweepID, true
}

func (l *AutonomyMaintenanceLoop) finishSweep(report AutonomyMaintenanceSweepReport) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweepRunning = false
	l.activeSweepID = ""
	l.lastSweepAt = report.CompletedAt
	l.lastSweepID = report.SweepID
	l.lastSweepTrigger = report.Trigger
	l.lastSweepDryRun = report.DryRun
	l.lastSweepStatus = report.Status
	if report.Status == "failed" && len(report.Diagnostics) > 0 {
		l.lastError = report.Diagnostics[len(report.Diagnostics)-1].Message
	}
}

func maintenanceSweepStatus(diags []AutonomyMaintenanceDiagnostic, runErr error) string {
	if runErr != nil {
		return "failed"
	}
	for _, diag := range diags {
		if diag.Severity == "error" {
			return "failed"
		}
	}
	return "completed"
}

func (l *AutonomyMaintenanceLoop) executeMaintenancePhase(ctx context.Context, idleReason string, dryRun bool) (AutonomyMaintenancePhaseReport, error) {
	phase := AutonomyMaintenancePhaseReport{
		Name:        "maintenance",
		Status:      "completed",
		DryRun:      dryRun,
		Actions:     []AutonomyMaintenanceAction{},
		Warnings:    []string{},
		Diagnostics: []AutonomyMaintenanceDiagnostic{},
		Summary:     map[string]any{},
	}
	if dryRun {
		return l.previewMaintenancePhase(ctx, idleReason, phase)
	}
	if l.customMaintenance && l.runMaintenance != nil {
		if err := l.runMaintenance(ctx, idleReason); err != nil {
			phase.Status = "failed"
			phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
				Code:     "CUSTOM_MAINTENANCE_FAILED",
				Severity: "error",
				Message:  err.Error(),
				Action:   "inspect the injected maintenance handler and retry after the failure is resolved",
			})
			return phase, err
		}
		phase.Summary["mode"] = "custom"
		return phase, nil
	}

	memoryRepaired := 0
	if l.memory != nil {
		detail, err := l.memory.RunRepairPass(ctx, memory.RunRepairRequest{
			Mode:       "autonomy_dream_maintenance",
			MaxAgeDays: 14,
			Limit:      120,
			Note:       "idle dream-state maintenance: " + idleReason,
		})
		if err != nil {
			phase.Status = "failed"
			phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
				Code:     "MEMORY_REPAIR_FAILED",
				Severity: "error",
				Message:  err.Error(),
				Action:   "inspect memory_repair_runs and failing observation rows before retrying",
			})
			return phase, err
		}
		phase.Summary["memoryRunId"] = detail.Run.ID
		phase.Summary["memoryCandidates"] = detail.Run.Candidates
		phase.Summary["memoryRepaired"] = detail.Run.Repaired
		phase.Summary["memorySkipped"] = detail.Run.Skipped
		phase.Summary["memoryFailed"] = detail.Run.Failed
		memoryRepaired = detail.Run.Repaired
		for _, item := range detail.Items {
			phase.Actions = append(phase.Actions, AutonomyMaintenanceAction{
				ID:          fmt.Sprintf("memory-observation-%d", item.ObservationID),
				Kind:        "memory.observation_repair",
				Summary:     item.Note,
				WouldCommit: true,
				Metadata: map[string]any{
					"observationId": item.ObservationID,
					"status":        item.Status,
					"issue":         item.Issue,
				},
			})
			if item.Status == "failed" {
				phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
					Code:     "MEMORY_REPAIR_ITEM_FAILED",
					Severity: "error",
					Message:  item.Note,
					Action:   "inspect the failing observation and retry after correcting source data",
					Metadata: map[string]any{"observationId": item.ObservationID},
				})
			}
		}
	}

	if l.runtime == nil || l.truth == nil {
		phase.Warnings = append(phase.Warnings, "memory repair completed; autonomy rule runtime not configured")
		phase.Summary["autonomyRuns"] = 0
		phase.Summary["memoryRepaired"] = memoryRepaired
		return phase, nil
	}

	now := l.nowMillis()
	input := autonomy.BuildRuleAgentInput(
		l.scope,
		fmt.Sprintf("corr-dream-maintenance-%d", now),
		fmt.Sprintf("trace-dream-maintenance-%d", now),
		l.truth,
		0,
		"dream_state_idle",
		now,
	)
	plans, planDiagnostics := l.runtime.PlanOnce(ctx, input)
	for _, diag := range planDiagnostics {
		phase.Diagnostics = append(phase.Diagnostics, autonomyRunDiagnostics("RULE_AGENT_EVALUATION_FAILED", diag, "inspect the rule-agent error and truth inputs before retrying")...)
	}
	runs, err := l.runtime.RunOnce(ctx, input)
	phase.Summary["autonomyRuns"] = len(runs)
	if err != nil {
		phase.Status = "failed"
		phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
			Code:     "RULE_AGENT_RUNTIME_FAILED",
			Severity: "error",
			Message:  err.Error(),
			Action:   "inspect autonomy runtime and truth engine connectivity before retrying",
		})
		return phase, err
	}
	for idx, plan := range plans {
		actionCount := len(plan.Actions)
		if actionCount == 0 {
			continue
		}
		decision := domain.DecisionAllowProposeOnly
		warnings := append([]string{}, plan.Warnings...)
		errors := []domain.AutonomyError{}
		if idx < len(runs) {
			decision = runs[idx].Decision
			warnings = append(warnings, runs[idx].Warnings...)
			errors = append(errors, runs[idx].Errors...)
		}
		appendSyscallActions(&phase, "autonomy.rule_agent", plan.Actions, true, map[string]any{
			"agentId":    plan.AgentID,
			"intentId":   plan.Intent.ID,
			"decision":   decision,
			"idleReason": idleReason,
		})
		phase.Warnings = append(phase.Warnings, warnings...)
		if len(errors) > 0 {
			phase.Diagnostics = append(phase.Diagnostics, autonomyRunErrorDiagnostics("RULE_AGENT_RUN_BLOCKED", errors, "review autonomy policy/charter diagnostics before promoting this maintenance action")...)
		}
	}
	return phase, nil
}

func (l *AutonomyMaintenanceLoop) executeImprovementPhase(ctx context.Context, idleReason string, dryRun bool) (AutonomyMaintenancePhaseReport, error) {
	phase := AutonomyMaintenancePhaseReport{
		Name:        "improvement",
		Status:      "completed",
		DryRun:      dryRun,
		Actions:     []AutonomyMaintenanceAction{},
		Warnings:    []string{},
		Diagnostics: []AutonomyMaintenanceDiagnostic{},
		Summary:     map[string]any{},
	}
	if !dryRun && l.customImprovement && l.runImprovement != nil {
		if err := l.runImprovement(ctx, idleReason); err != nil {
			phase.Status = "failed"
			phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
				Code:     "CUSTOM_IMPROVEMENT_FAILED",
				Severity: "error",
				Message:  err.Error(),
				Action:   "inspect the injected improvement handler and retry after the failure is resolved",
			})
			return phase, err
		}
		phase.Summary["mode"] = "custom"
		return phase, nil
	}
	if l.truth == nil {
		phase.Warnings = append(phase.Warnings, "truth engine is not configured; improvement diagnostics are unavailable")
		return phase, nil
	}
	now := l.nowMillis()
	activeLoops, activeErr := l.truth.ListActiveLoops(ctx, l.scope, 100)
	blockedLoops, blockedErr := l.truth.ListBlockedLoops(ctx, l.scope, 100)
	staleLoops, staleErr := l.truth.ListStaleLoops(ctx, l.scope, now-int64((72*time.Hour)/time.Millisecond), 100)
	contradictions, contradictionErr := l.truth.ListContradictionsByScope(ctx, l.scope, 100)
	for code, err := range map[string]error{
		"ACTIVE_LOOPS_QUERY_FAILED":   activeErr,
		"BLOCKED_LOOPS_QUERY_FAILED":  blockedErr,
		"STALE_LOOPS_QUERY_FAILED":    staleErr,
		"CONTRADICTIONS_QUERY_FAILED": contradictionErr,
	} {
		if err != nil {
			phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
				Code:     code,
				Severity: "error",
				Message:  err.Error(),
				Action:   "verify truth repositories and maintenance scope before retrying the sweep",
			})
		}
	}
	if activeErr == nil {
		phase.Summary["activeLoops"] = len(activeLoops)
	}
	if blockedErr == nil {
		phase.Summary["blockedLoops"] = len(blockedLoops)
	}
	if staleErr == nil {
		phase.Summary["staleLoops"] = len(staleLoops)
	}
	if contradictionErr == nil {
		phase.Summary["contradictions"] = len(contradictions)
	}
	intent, actions, summary := l.buildImprovementPlan(now, idleReason, len(activeLoops), len(blockedLoops), len(staleLoops), len(contradictions))
	phase.Summary["snapshot"] = summary
	appendSyscallActions(&phase, "autonomy.improvement", actions, !dryRun, map[string]any{
		"intentId":   intent.ID,
		"idleReason": idleReason,
	})
	if dryRun {
		if l.runner == nil {
			phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
				Code:     "RUNNER_UNAVAILABLE",
				Severity: "warning",
				Message:  "self-initiated syscall runner is not configured; only action previews are available",
				Action:   "configure autonomy runner to validate maintenance improvements end-to-end",
			})
			return phase, nil
		}
		preview, err := l.runner.Preview(ctx, intent, actions, l.mode)
		if err != nil {
			phase.Status = "failed"
			phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
				Code:     "IMPROVEMENT_PREVIEW_FAILED",
				Severity: "error",
				Message:  err.Error(),
				Action:   "inspect autonomy policy and improvement payloads before retrying",
			})
			return phase, err
		}
		phase.Summary["decision"] = preview.Decision
		phase.Summary["decisionId"] = preview.DecisionID
		phase.Warnings = append(phase.Warnings, preview.Warnings...)
		if len(preview.Errors) > 0 {
			phase.Diagnostics = append(phase.Diagnostics, autonomyRunErrorDiagnostics("IMPROVEMENT_PREVIEW_BLOCKED", preview.Errors, "review policy/charter/budget diagnostics before enabling commit mode")...)
		}
		return phase, nil
	}
	if l.runner == nil {
		phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
			Code:     "RUNNER_UNAVAILABLE",
			Severity: "warning",
			Message:  "self-initiated syscall runner is not configured; improvement commit skipped",
			Action:   "configure autonomy runner to allow bounded improvement commits",
		})
		return phase, nil
	}
	run, err := l.runner.Run(ctx, intent, actions, domain.RunModeCommitIfAuthorized, l.mode)
	phase.Summary["decision"] = run.Decision
	phase.Summary["decisionId"] = run.DecisionID
	phase.Summary["committedCount"] = len(run.CommittedObjectIDs)
	phase.Warnings = append(phase.Warnings, run.Warnings...)
	if len(run.Errors) > 0 {
		phase.Diagnostics = append(phase.Diagnostics, autonomyRunErrorDiagnostics("IMPROVEMENT_RUN_BLOCKED", run.Errors, "review autonomy decision diagnostics before retrying in commit mode")...)
	}
	if err != nil {
		phase.Status = "failed"
		phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
			Code:     "IMPROVEMENT_RUN_FAILED",
			Severity: "error",
			Message:  err.Error(),
			Action:   "inspect autonomy runner errors and underlying syscall failures before retrying",
		})
		return phase, err
	}
	return phase, nil
}

func (l *AutonomyMaintenanceLoop) previewMaintenancePhase(ctx context.Context, idleReason string, phase AutonomyMaintenancePhaseReport) (AutonomyMaintenancePhaseReport, error) {
	candidates, err := l.previewMemoryRepairCandidates(ctx, 14, 120)
	if err != nil {
		phase.Status = "failed"
		phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
			Code:     "MEMORY_REPAIR_PREVIEW_FAILED",
			Severity: "error",
			Message:  err.Error(),
			Action:   "inspect memory observation tables and retrieval result rows before retrying",
		})
		return phase, err
	}
	repaired := 0
	skipped := 0
	for _, item := range candidates {
		if item.Metadata["status"] == "repaired" {
			repaired++
		} else {
			skipped++
		}
	}
	phase.Summary["memoryCandidates"] = len(candidates)
	phase.Summary["memoryRepaired"] = repaired
	phase.Summary["memorySkipped"] = skipped
	phase.Actions = append(phase.Actions, candidates...)

	if l.runtime == nil || l.truth == nil {
		phase.Warnings = append(phase.Warnings, "autonomy rule runtime not configured; dry-run contains only memory repair previews")
		return phase, nil
	}
	now := l.nowMillis()
	input := autonomy.BuildRuleAgentInput(
		l.scope,
		fmt.Sprintf("corr-dream-maintenance-preview-%d", now),
		fmt.Sprintf("trace-dream-maintenance-preview-%d", now),
		l.truth,
		0,
		"dream_state_idle_preview",
		now,
	)
	plans, planDiagnostics := l.runtime.PlanOnce(ctx, input)
	for _, diag := range planDiagnostics {
		phase.Diagnostics = append(phase.Diagnostics, autonomyRunDiagnostics("RULE_AGENT_PREVIEW_FAILED", diag, "inspect the rule-agent input and truth diagnostics before retrying")...)
	}
	phase.Summary["autonomyPlans"] = len(plans)
	for _, plan := range plans {
		previewWarnings := append([]string{}, plan.Warnings...)
		decision := domain.DecisionAllowProposeOnly
		if l.runner != nil {
			preview, err := l.runner.Preview(ctx, plan.Intent, plan.Actions, l.mode)
			if err != nil {
				phase.Diagnostics = append(phase.Diagnostics, AutonomyMaintenanceDiagnostic{
					Code:     "RULE_AGENT_POLICY_PREVIEW_FAILED",
					Severity: "error",
					Message:  err.Error(),
					Action:   "inspect autonomy policy dependencies before retrying",
					Metadata: map[string]any{"agentId": plan.AgentID, "intentId": plan.Intent.ID},
				})
			} else {
				decision = preview.Decision
				previewWarnings = append(previewWarnings, preview.Warnings...)
				if len(preview.Errors) > 0 {
					phase.Diagnostics = append(phase.Diagnostics, autonomyRunErrorDiagnostics("RULE_AGENT_PREVIEW_BLOCKED", preview.Errors, "review policy/charter diagnostics before moving this action to commit mode")...)
				}
			}
		}
		appendSyscallActions(&phase, "autonomy.rule_agent", plan.Actions, true, map[string]any{
			"agentId":    plan.AgentID,
			"intentId":   plan.Intent.ID,
			"decision":   decision,
			"idleReason": idleReason,
		})
		phase.Warnings = append(phase.Warnings, previewWarnings...)
	}
	return phase, nil
}

func (l *AutonomyMaintenanceLoop) previewMemoryRepairCandidates(ctx context.Context, maxAgeDays, limit int) ([]AutonomyMaintenanceAction, error) {
	if l.db == nil {
		return nil, fmt.Errorf("memory repair preview requires database")
	}
	if maxAgeDays <= 0 || maxAgeDays > 365 {
		maxAgeDays = 14
	}
	if limit <= 0 || limit > 300 {
		limit = 120
	}
	cutoff := time.UnixMilli(l.nowMillis()).Add(-time.Duration(maxAgeDays) * 24 * time.Hour).UnixMilli()
	rows, err := l.db.QueryContext(ctx, `
SELECT id, created_at, updated_at, observed_at, type, raw_content, summary, embedding_ref, dossier_id, project_key, source_path,
       entities_json, tags_json, related_files_json, task_type, confidence, verification_state, lineage_json, origin_kind, origin_id,
       stale, last_verified_at, usefulness_score, usefulness_count, noise_count
FROM memory_observations
WHERE (stale = 1 OR usefulness_score < -0.5 OR noise_count > usefulness_count OR COALESCE(last_verified_at, 0) < ?)
ORDER BY stale DESC, usefulness_score ASC, updated_at ASC
LIMIT ?`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actions := []AutonomyMaintenanceAction{}
	for rows.Next() {
		var obs memory.Observation
		var dossierID sql.NullInt64
		var lastVerified sql.NullInt64
		var staleInt int
		var entitiesJSON string
		var tagsJSON string
		var relatedFilesJSON string
		var lineageJSON string
		if err := rows.Scan(
			&obs.ID,
			&obs.CreatedAtMs,
			&obs.UpdatedAtMs,
			&obs.ObservedAtMs,
			&obs.Type,
			&obs.RawContent,
			&obs.Summary,
			&obs.EmbeddingRef,
			&dossierID,
			&obs.ProjectKey,
			&obs.SourcePath,
			&entitiesJSON,
			&tagsJSON,
			&relatedFilesJSON,
			&obs.TaskType,
			&obs.Confidence,
			&obs.VerificationState,
			&lineageJSON,
			&obs.OriginKind,
			&obs.OriginID,
			&staleInt,
			&lastVerified,
			&obs.UsefulnessScore,
			&obs.UsefulnessCount,
			&obs.NoiseCount,
		); err != nil {
			return nil, err
		}
		if dossierID.Valid {
			obs.DossierID = &dossierID.Int64
		}
		obs.Entities = json.RawMessage(strings.TrimSpace(entitiesJSON))
		obs.Tags = json.RawMessage(strings.TrimSpace(tagsJSON))
		obs.RelatedFiles = json.RawMessage(strings.TrimSpace(relatedFilesJSON))
		obs.Lineage = json.RawMessage(strings.TrimSpace(lineageJSON))
		obs.Stale = staleInt > 0
		if lastVerified.Valid {
			obs.LastVerifiedAtMs = &lastVerified.Int64
		}
		action, err := l.previewMemoryRepairObservation(ctx, obs)
		if err != nil {
			actions = append(actions, AutonomyMaintenanceAction{
				ID:          fmt.Sprintf("memory-observation-%d", obs.ID),
				Kind:        "memory.observation_repair",
				Summary:     "repair preview failed: " + err.Error(),
				WouldCommit: true,
				Metadata: map[string]any{
					"observationId": obs.ID,
					"status":        "failed",
				},
			})
			continue
		}
		actions = append(actions, action)
	}
	return actions, rows.Err()
}

func (l *AutonomyMaintenanceLoop) previewMemoryRepairObservation(ctx context.Context, obs memory.Observation) (AutonomyMaintenanceAction, error) {
	newRaw := strings.TrimSpace(obs.RawContent)
	newSummary := strings.TrimSpace(obs.Summary)
	changed := false
	note := ""
	if strings.TrimSpace(obs.OriginKind) == "retrieval_result" {
		resultID, parseErr := strconv.ParseInt(strings.TrimSpace(obs.OriginID), 10, 64)
		if parseErr == nil && resultID > 0 {
			var snippet sql.NullString
			if err := l.db.QueryRowContext(ctx, `SELECT snippet FROM retrieval_results WHERE id = ?`, resultID).Scan(&snippet); err != nil && err != sql.ErrNoRows {
				return AutonomyMaintenanceAction{}, err
			} else if snippet.Valid {
				fresh := strings.TrimSpace(snippet.String)
				if fresh != "" && fresh != newRaw {
					newRaw = fresh
					changed = true
					note = "raw content refreshed from retrieval result snippet"
				}
			}
		}
	}
	if newSummary == "" || obs.Stale || obs.UsefulnessScore < -0.5 {
		auto := previewSummarizeRawContent(newRaw)
		if auto != "" && auto != newSummary {
			newSummary = auto
			changed = true
			if note == "" {
				note = "summary refreshed from current raw content"
			}
		}
	}
	status := "skipped"
	issue := "no_change"
	if changed {
		status = "repaired"
		issue = "updated"
	}
	if note == "" {
		note = "observation verified"
	}
	return AutonomyMaintenanceAction{
		ID:          fmt.Sprintf("memory-observation-%d", obs.ID),
		Kind:        "memory.observation_repair",
		Summary:     note,
		WouldCommit: true,
		Metadata: map[string]any{
			"observationId":      obs.ID,
			"status":             status,
			"issue":              issue,
			"stale":              obs.Stale,
			"verificationState":  obs.VerificationState,
			"sourcePath":         obs.SourcePath,
			"wouldUpdateRaw":     changed && strings.TrimSpace(obs.RawContent) != newRaw,
			"wouldUpdateSummary": changed && strings.TrimSpace(obs.Summary) != newSummary,
		},
	}, nil
}

func (l *AutonomyMaintenanceLoop) buildImprovementPlan(now int64, idleReason string, activeCount, blockedCount, staleCount, contradictionCount int) (domain.AutonomyIntent, []domain.SyscallRequest, string) {
	summary := fmt.Sprintf(
		"Dream-state improvement snapshot. active_loops=%d blocked_loops=%d stale_loops=%d contradictions=%d reason=%s",
		activeCount,
		blockedCount,
		staleCount,
		contradictionCount,
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
	return intent, actions, summary
}

func appendSyscallActions(phase *AutonomyMaintenancePhaseReport, kind string, actions []domain.SyscallRequest, wouldCommit bool, shared map[string]any) {
	for _, action := range actions {
		metadata := map[string]any{
			"action":         action.Action,
			"scope":          action.Scope,
			"payload":        action.Payload,
			"correlationId":  action.CorrelationID,
			"traceId":        action.TraceID,
			"idempotencyKey": action.IdempotencyKey,
			"dryRun":         action.DryRun,
		}
		for key, value := range shared {
			metadata[key] = value
		}
		phase.Actions = append(phase.Actions, AutonomyMaintenanceAction{
			ID:          action.ID,
			Kind:        kind,
			Summary:     fmt.Sprintf("%s %s", action.Action, summarizeActionPayload(action.Payload)),
			WouldCommit: wouldCommit,
			Metadata:    metadata,
		})
	}
}

func summarizeActionPayload(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	if title := strings.TrimSpace(fmt.Sprintf("%v", payload["title"])); title != "" && title != "<nil>" {
		return title
	}
	if key := strings.TrimSpace(fmt.Sprintf("%v", payload["key"])); key != "" && key != "<nil>" {
		return key
	}
	if loopID := strings.TrimSpace(fmt.Sprintf("%v", payload["loopId"])); loopID != "" && loopID != "<nil>" {
		return loopID
	}
	if query := strings.TrimSpace(fmt.Sprintf("%v", payload["query"])); query != "" && query != "<nil>" {
		return query
	}
	return ""
}

func autonomyRunDiagnostics(code string, run domain.AutonomyRunSummary, action string) []AutonomyMaintenanceDiagnostic {
	if len(run.Errors) == 0 {
		return nil
	}
	return autonomyRunErrorDiagnostics(code, run.Errors, action)
}

func autonomyRunErrorDiagnostics(code string, errs []domain.AutonomyError, action string) []AutonomyMaintenanceDiagnostic {
	out := make([]AutonomyMaintenanceDiagnostic, 0, len(errs))
	for _, err := range errs {
		out = append(out, AutonomyMaintenanceDiagnostic{
			Code:     code,
			Severity: "error",
			Message:  err.Message,
			Action:   action,
			Metadata: map[string]any{
				"field": err.Field,
				"kind":  err.Code,
			},
		})
	}
	return out
}

func previewSummarizeRawContent(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	if len(v) <= 220 {
		return v
	}
	return v[:220] + "..."
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
	intent, actions, _ := l.buildImprovementPlan(now, idleReason, len(activeLoops), len(blockedLoops), len(staleLoops), len(contradictions))
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

func newDefaultAutonomyMaintenanceLoop(db *sql.DB, cfg config.Config, ev *events.Logger, memorySvc *memory.Service, controlLaneValidationObserver controllane.ControlLaneValidationObserver) *AutonomyMaintenanceLoop {
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
		Registry:                      controllane.NewStaticActionRegistry(),
		Validator:                     controllane.NewDeterministicValidator(),
		Capabilities:                  controllane.NewStaticCapabilityService(),
		ApprovalGate:                  controllane.NewStaticApprovalGate(),
		TxRunner:                      txRunner,
		AuditSink:                     controllane.NewCoreAuditSink(audit.New(db)),
		RuleEngine:                    rulecells.MustStaticEngine(),
		NowMillis:                     nowFn,
		ControlLaneValidationObserver: controlLaneValidationObserver,
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
	bundle, err := autonomy.NewSQLiteBundleStrict(db)
	if err != nil {
		log.Printf("autonomy maintenance loop disabled: %v", err)
		return nil
	}
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
	mode := parseAutonomyMode(loadSetting(db, "autonomy_mode", string(domain.AutonomyModeObserve)))
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
		return domain.AutonomyModeObserve
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
