package api

import (
	"net/http"
	"time"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/forgekernel"
)

func (s *Server) forgeKActivationReadiness(now time.Time) controllane.ForgeKActivationReadinessReport {
	report := controllane.ForgeKActivationReadiness(controllane.NewStaticActionRegistry(), now)
	selection := s.kernelAuthority
	if !selection.SingleAuthority || selection.Processor == nil {
		report.Status = "kernel_authority_unavailable"
		report.Summary = "No production semantic kernel authority was selected at daemon boot; semantic mutation paths are fail-closed."
		report.Mode = "fail_closed"
		report.LiveOwner = "none"
		report.KernelRuntimeState = "unavailable"
		if s.kernelErr != "" {
			report.Notes = append(report.Notes, "boot authority selection failed: "+s.kernelErr)
		}
		return report
	}
	if selection.Mode != forgekernel.ModeForgeK {
		return report
	}
	report.Status = "forge_k_authenticated_commit_integrity_live"
	report.Summary = "FORGE-K owns authenticated semantic syscall ingress, actor/capability/approval proof verification, durable stage order, deterministic Courthouse decisions, and sealed commit-integrity verification; Control Lane implements the temporary durable SQLite port."
	report.Mode = "live_authority_migration"
	report.LiveOwner = selection.AuthorityOwner
	report.KernelRuntimeState = "forge_k_orchestration_live_control_lane_sqlite_port"
	report.LiveKernelIngressAuthority = true
	report.LiveDurableOrchestration = true
	report.LiveAuthorityMigration = true
	if report.NoEffect == nil {
		report.NoEffect = map[string]any{}
	}
	report.NoEffect["kernelIngressAuthority"] = true
	report.NoEffect["durableOrchestrationAuthority"] = true
	report.NoEffect["commitIntegrityAuthority"] = true
	report.NoEffect["sealedPreparedPlans"] = true
	report.NoEffect["typedCommitReceiptValidation"] = true
	report.NoEffect["atomicAuditOutboxEvidence"] = true
	report.NoEffect["verifiedIdempotentReplay"] = true
	report.NoEffect["authenticatedAuthorizationProof"] = s.kernelAuthorizationReady
	report.NoEffect["durableAuthorizationReplayProof"] = s.kernelAuthorizationReady
	report.NoEffect["uniqueForgeCoreServicePrincipal"] = s.kernelAuthorizationReady
	report.NoEffect["authenticatedUserOriginRequired"] = s.kernelAuthorizationReady
	report.NoEffect["unauthenticatedRemoteOrigin"] = false
	report.NoEffect["externalAuditSinkDelivery"] = false
	report.NoEffect["auditIdBackfill"] = false
	report.NoEffect["liveAuthorityMigration"] = true
	for i := range report.ValidationActions {
		report.ValidationActions[i].LiveOwner = selection.AuthorityOwner
	}
	for i := range report.Gates {
		switch report.Gates[i].Name {
		case "live_owner_explicit":
			report.Gates[i].Reason = "production semantic syscall ingress owner is forge_k.kernel"
		case "live_kernel_authority_disabled":
			report.Gates[i].Name = "full_kernel_authority_gated"
			report.Gates[i].Reason = "FORGE-K ingress, durable stage orchestration, Courthouse decisions, and commit integrity are live; subsystem authority remains staged"
		}
	}
	report.Notes = []string{
		"production semantic syscall construction selects exactly one boot authority",
		"FORGE_KERNEL_AUTHORITY_MODE=legacy_v1 is the tested rollback mode",
		"exact prepared requests and plans are sealed; successful commits require a validated typed receipt",
		"semantic mutation, journal hash-chain head and provenance, immutable audit intent, and optional idempotency proof share one SQLite transaction",
		"verified idempotent replay does not re-commit; legacy unbound replay proof fails closed",
		"production authorization binds the constructed forge.core service principal, authenticated origin, registry definition, scoped capability grant, and approval policy/decision",
		"bearer credentials are represented only by non-secret fingerprints; tokenless HTTP attests a user origin only for verified loopback peers",
		"full authorization proof JSON is immutable atomic idempotency/audit-outbox evidence and is revalidated during replay",
		"external audit sink delivery and audit_id backfill remain best-effort projections and cannot invalidate canonical atomic outbox evidence",
		"full FORGE-K authority remains incomplete until Control Lane policy/apply implementations and remaining subsystem gates are migrated",
	}
	for i := range report.AuthorityMatrix {
		switch report.AuthorityMatrix[i].Subsystem {
		case "Courthouse":
			report.AuthorityMatrix[i].CurrentStatus = "FORGE_K_ADMISSION_AND_RULING_LIVE"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].FeatureFlag = "production FORGE-K authority mode only; legacy_v1 fails closed"
			report.AuthorityMatrix[i].RollbackPath = "select legacy_v1 to disable admission/ruling mutations; existing Court history remains inspectable"
			report.AuthorityMatrix[i].TestsPassing = append(report.AuthorityMatrix[i].TestsPassing,
				"deterministic admission/rejection tests",
				"immutable ruling and appeal history tests",
				"workspace/lane scope isolation tests",
				"journal collision atomic rollback tests",
			)
			report.AuthorityMatrix[i].Blockers = []string{
				"external audit sink delivery and audit_id backfill remain best-effort projections; canonical audit-outbox evidence is already atomic",
			}
		case "Kernel":
			report.AuthorityMatrix[i].CurrentStatus = "FORGE_K_COMMIT_INTEGRITY_LIVE"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].FeatureFlag = "FORGE_KERNEL_AUTHORITY_MODE=forge_k (default); legacy_v1 is rollback only"
			report.AuthorityMatrix[i].RollbackPath = "set FORGE_KERNEL_AUTHORITY_MODE=legacy_v1 and restart; boot selects one authority and never dual-commits"
			report.AuthorityMatrix[i].TestsPassing = append(report.AuthorityMatrix[i].TestsPassing,
				"production kernel authority selection tests",
				"single-delegate commit tests",
				"external authority-claim rejection tests",
				"FORGE-K durable stage-order tests",
				"idempotency/capability/approval/journal-rollback port tests",
				"FORGE-K Courthouse decision authority tests",
				"prepared request/plan seal and typed receipt validation tests",
				"atomic journal hash-chain, provenance, audit-outbox, and idempotency proof tests",
				"verified replay, legacy-unbound rejection, and transaction rollback tests",
				"authenticated service-principal, bearer/loopback origin, capability/approval proof, and tampered replay tests",
			)
			report.AuthorityMatrix[i].Blockers = []string{
				"Control Lane still implements validation policies, semantic apply functions, and the SQLite durable port",
				"Memory Palace, Semantic Algebra, Context Compiler, Runtime, Snapshot, KV, Lymphatic, and Consensus authority gates remain staged",
				"external audit sink delivery and audit_id backfill remain best-effort projections over canonical atomic outbox evidence",
			}
		}
	}
	return report
}

func (s *Server) handleForgeKernelStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.forgeKActivationReadiness(time.Now().UTC()))
}
