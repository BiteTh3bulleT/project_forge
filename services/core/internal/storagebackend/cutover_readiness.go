package storagebackend

const (
	CutoverStatusBlocked       = "blocked"
	CutoverStatusProposalReady = "cutover_proposal_ready"
)

type CutoverReadinessInput struct {
	Backend                  Config
	SelectedDomain           string
	SQLiteBaselineTests      bool
	PostgresMigrationTests   bool
	PostgresAdapterTests     bool
	RepositoryParityTests    bool
	DualWriteComparisonTests bool
	ReadCompareMismatchTests bool
	BackupRollbackTests      bool
	OperatorApprovalRecorded bool
}

type CutoverReadinessReport struct {
	Status                  string          `json:"status"`
	SelectedDomain          string          `json:"selected_domain"`
	CanonicalDefault        BackendKind     `json:"canonical_default"`
	RequestedBackend        BackendKind     `json:"requested_backend"`
	LiveOwner               string          `json:"live_owner"`
	TargetOwner             string          `json:"target_owner"`
	ReadyForDualWrite       bool            `json:"ready_for_dual_write"`
	ReadyForReadCompare     bool            `json:"ready_for_read_compare"`
	ReadyForCutoverProposal bool            `json:"ready_for_cutover_proposal"`
	PostgresCanonicalReady  bool            `json:"postgres_canonical_ready"`
	RedisTruthAuthority     bool            `json:"redis_truth_authority"`
	QdrantTruthAuthority    bool            `json:"qdrant_truth_authority"`
	TestsRequired           []string        `json:"tests_required"`
	TestsPassing            []string        `json:"tests_passing"`
	Blockers                []string        `json:"blockers"`
	RollbackPath            string          `json:"rollback_path"`
	NoEffect                map[string]bool `json:"no_effect"`
}

func EvaluateCutoverReadiness(input CutoverReadinessInput) CutoverReadinessReport {
	tests := []readinessTest{
		{name: "SQLite baseline tests", passed: input.SQLiteBaselineTests},
		{name: "Postgres migration idempotence tests", passed: input.PostgresMigrationTests},
		{name: "Postgres adapter tests", passed: input.PostgresAdapterTests},
		{name: "repository parity tests for selected domain", passed: input.RepositoryParityTests},
		{name: "dual-write comparison mismatch tests", passed: input.DualWriteComparisonTests},
		{name: "read-compare mismatch tests", passed: input.ReadCompareMismatchTests},
		{name: "backup and rollback restores SQLite tests", passed: input.BackupRollbackTests},
		{name: "operator approval recorded", passed: input.OperatorApprovalRecorded},
	}

	report := CutoverReadinessReport{
		Status:           CutoverStatusBlocked,
		SelectedDomain:   input.SelectedDomain,
		CanonicalDefault: BackendSQLite,
		RequestedBackend: input.Backend.Kind,
		LiveOwner:        "services/core/internal/store.Open with SQLite at ${FORGE_DATA_DIR}/forge.sqlite",
		TargetOwner:      "future storagebackend Postgres repository adapters; FORGE-K persistence only after separate authority migration",
		TestsRequired:    make([]string, 0, len(tests)),
		TestsPassing:     make([]string, 0, len(tests)),
		RollbackPath:     "leave FORGE_STORE_BACKEND unset or set to sqlite; keep Redis disposable and rebuild Qdrant from relational records",
		NoEffect: map[string]bool{
			"canonicalDefaultChanged":  false,
			"dualWriteEnabled":         false,
			"readSwitchEnabled":        false,
			"redisCanonicalTruth":      false,
			"qdrantCanonicalTruth":     false,
			"forgeKAuthorityMigration": false,
		},
	}
	if report.SelectedDomain == "" {
		report.SelectedDomain = "none"
	}
	if report.RequestedBackend == "" {
		report.RequestedBackend = BackendSQLite
	}

	if input.Backend.Kind != BackendPostgres {
		report.Blockers = append(report.Blockers, "postgres backend not explicitly selected for a cutover proposal")
	}
	if input.Backend.Kind == BackendPostgres && input.Backend.PostgresDSN == "" {
		report.Blockers = append(report.Blockers, "postgres DSN missing")
	}
	if report.SelectedDomain == "none" {
		report.Blockers = append(report.Blockers, "selected storage domain missing")
	}

	allEvidencePresent := true
	for _, test := range tests {
		report.TestsRequired = append(report.TestsRequired, test.name)
		if test.passed {
			report.TestsPassing = append(report.TestsPassing, test.name)
			continue
		}
		allEvidencePresent = false
		report.Blockers = append(report.Blockers, test.name+" missing")
	}

	report.ReadyForDualWrite = input.Backend.Kind == BackendPostgres &&
		input.Backend.PostgresDSN != "" &&
		input.SelectedDomain != "" &&
		input.SQLiteBaselineTests &&
		input.PostgresMigrationTests &&
		input.PostgresAdapterTests &&
		input.RepositoryParityTests
	report.ReadyForReadCompare = report.ReadyForDualWrite && input.DualWriteComparisonTests
	report.ReadyForCutoverProposal = report.ReadyForReadCompare &&
		input.ReadCompareMismatchTests &&
		input.BackupRollbackTests &&
		input.OperatorApprovalRecorded &&
		allEvidencePresent &&
		len(report.Blockers) == 0
	report.PostgresCanonicalReady = report.ReadyForCutoverProposal
	if report.ReadyForCutoverProposal {
		report.Status = CutoverStatusProposalReady
	}
	return report
}

type readinessTest struct {
	name   string
	passed bool
}
