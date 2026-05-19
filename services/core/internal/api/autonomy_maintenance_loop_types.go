package api

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"forge/projectforge/services/core/internal/aios/autonomy"
	"forge/projectforge/services/core/internal/aios/compute/librarian"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/aios/truth"
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
	LibrarianPipeline   *librarian.IngestPipeline
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

	librarianPipeline *librarian.IngestPipeline

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
