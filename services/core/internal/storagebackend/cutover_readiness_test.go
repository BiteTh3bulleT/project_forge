package storagebackend

import "testing"

func TestCutoverReadinessDefaultsBlockCutoverAndPreserveSQLiteAuthority(t *testing.T) {
	cfg, err := NewConfig(ConfigInput{})
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	report := EvaluateCutoverReadiness(CutoverReadinessInput{Backend: cfg})

	if report.Status != CutoverStatusBlocked {
		t.Fatalf("status=%q, want %q", report.Status, CutoverStatusBlocked)
	}
	if report.CanonicalDefault != BackendSQLite || report.RequestedBackend != BackendSQLite {
		t.Fatalf("expected sqlite default/requested backend, got default=%q requested=%q", report.CanonicalDefault, report.RequestedBackend)
	}
	if report.LiveOwner == "" || report.TargetOwner == "" {
		t.Fatalf("expected explicit live and target owners, got live=%q target=%q", report.LiveOwner, report.TargetOwner)
	}
	if report.ReadyForDualWrite || report.ReadyForReadCompare || report.ReadyForCutoverProposal {
		t.Fatalf("default readiness must not allow storage cutover stages: %#v", report)
	}
	if report.PostgresCanonicalReady || report.RedisTruthAuthority || report.QdrantTruthAuthority {
		t.Fatalf("default readiness claimed forbidden storage authority: %#v", report)
	}
	if report.NoEffect["canonicalDefaultChanged"] || report.NoEffect["dualWriteEnabled"] || report.NoEffect["readSwitchEnabled"] {
		t.Fatalf("readiness report claimed live storage effects: %#v", report.NoEffect)
	}
	if !contains(report.Blockers, "postgres backend not explicitly selected for a cutover proposal") {
		t.Fatalf("missing expected default blocker in %#v", report.Blockers)
	}
}

func TestCutoverReadinessTreatsInfrastructureAsNonAuthoritative(t *testing.T) {
	cfg, err := NewConfig(ConfigInput{
		Backend:     "postgres",
		PostgresDSN: "postgres://forge:forge@localhost:5432/forge?sslmode=disable",
		RedisAddr:   "redis:6379",
		QdrantURL:   "http://qdrant:6333",
	})
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	report := EvaluateCutoverReadiness(CutoverReadinessInput{
		Backend:                  cfg,
		SQLiteBaselineTests:      true,
		PostgresMigrationTests:   true,
		PostgresAdapterTests:     true,
		RepositoryParityTests:    true,
		DualWriteComparisonTests: true,
		ReadCompareMismatchTests: true,
		BackupRollbackTests:      true,
		OperatorApprovalRecorded: true,
		SelectedDomain:           "shadow_diagnostics",
	})

	if report.Status != CutoverStatusProposalReady {
		t.Fatalf("status=%q, want %q; blockers=%#v", report.Status, CutoverStatusProposalReady, report.Blockers)
	}
	if !report.ReadyForDualWrite || !report.ReadyForReadCompare || !report.ReadyForCutoverProposal {
		t.Fatalf("expected proposal readiness for selected domain, got %#v", report)
	}
	if !report.PostgresCanonicalReady {
		t.Fatalf("expected postgres proposal readiness when all readiness evidence is present")
	}
	if report.RedisTruthAuthority || report.QdrantTruthAuthority {
		t.Fatalf("redis/qdrant must remain non-authoritative: %#v", report)
	}
	if report.NoEffect["canonicalDefaultChanged"] || report.NoEffect["readSwitchEnabled"] {
		t.Fatalf("readiness must not change defaults or switch reads: %#v", report.NoEffect)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
