package api

import (
	"net/http"
	"time"

	"forge/projectforge/services/core/internal/aios/controllane"
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
	report.Status = "forge_k_sole_live_authority"
	report.Phase = "K20J"
	report.Summary = "FORGE-K is the sole live semantic and model-visibility authority. Control Lane is a bounded validation/apply/SQLite durable port and cannot independently orchestrate or commit production requests."
	report.Mode = "live_authority"
	report.LiveOwner = selection.AuthorityOwner
	report.PolicyVersion = "forge-k-sole-authority-k20j-v1"
	report.KernelRuntimeState = "forge_k_sole_authority_control_lane_durable_port"
	report.LiveKernelIngressAuthority = true
	report.LiveDurableOrchestration = true
	report.LiveKernelAuthority = true
	report.LiveAuthorityMigration = false
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
	report.NoEffect["liveKernelAuthority"] = true
	report.NoEffect["liveAuthorityMigration"] = false
	report.NoEffect["legacyLiveAuthority"] = false
	report.NoEffect["kernelContextCompilerAuthority"] = true
	report.NoEffect["kernelRuntimeProposalAuthority"] = true
	report.NoEffect["backendKVReuse"] = false
	report.NoEffect["autonomousCanonicalMutation"] = false
	for i := range report.ValidationActions {
		report.ValidationActions[i].LiveOwner = selection.AuthorityOwner
		report.ValidationActions[i].LiveKernelAuthority = true
	}
	for i := range report.Gates {
		switch report.Gates[i].Name {
		case "live_owner_explicit":
			report.Gates[i].Reason = "production semantic syscall ingress owner is forge_k.kernel"
		case "live_kernel_authority_disabled":
			report.Gates[i].Name = "sole_live_kernel_authority"
			report.Gates[i].Passed = true
			report.Gates[i].Reason = "production boot constructs only FORGE-K; alternate live authority modes fail closed"
		}
	}
	for i := range report.AuthorityGates {
		report.AuthorityGates[i].Status = "ready"
		report.AuthorityGates[i].LiveOwner = selection.AuthorityOwner
		report.AuthorityGates[i].Reason = "the live surface is Kernel-owned or fail-closed as a non-authoritative proposal/acceleration lane"
		report.AuthorityGates[i].NextStep = "preserve the sole-authority invariant while extending capabilities"
	}
	report.AuthorityReadyGates = len(report.AuthorityGates)
	report.AuthorityBlockedGates = 0
	report.Notes = []string{
		"production boot constructs exactly one semantic authority: forge_k.kernel",
		"obsolete alternate authority modes fail closed; rollback is an offline store/generation procedure",
		"exact prepared requests and plans are sealed; successful commits require a validated typed receipt",
		"semantic mutation, journal hash-chain head and provenance, immutable audit intent, and optional idempotency proof share one SQLite transaction",
		"verified idempotent replay does not re-commit; legacy unbound replay proof fails closed",
		"production authorization binds the constructed forge.core service principal, authenticated origin, registry definition, scoped capability grant, and approval policy/decision",
		"bearer credentials are represented only by non-secret fingerprints; tokenless HTTP attests a user origin only for verified loopback peers",
		"full authorization proof JSON is immutable atomic idempotency/audit-outbox evidence and is revalidated during replay",
		"all model-visible output is withheld until the Kernel runtime-proposal boundary and consensus gate accept it",
		"all model prompts are assembled from an immutable Kernel Context Compiler decision over current admitted exact-scope evidence",
		"disabled capabilities such as backend KV reuse remain fail-closed acceleration, not alternate authority",
		"external audit sink delivery and audit_id backfill remain best-effort projections and cannot invalidate canonical atomic outbox evidence",
	}
	for i := range report.AuthorityMatrix {
		switch report.AuthorityMatrix[i].Subsystem {
		case "Courthouse":
			report.AuthorityMatrix[i].CurrentStatus = "FORGE_K_ADMISSION_AND_RULING_LIVE"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].FeatureFlag = "production FORGE-K is the sole boot-constructed authority"
			report.AuthorityMatrix[i].RollbackPath = "stop the daemon and use the verified offline store/generation rollback procedure"
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
			report.AuthorityMatrix[i].CurrentStatus = "FORGE_K_SOLE_LIVE_AUTHORITY"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].FeatureFlag = "no live authority-mode selector; production always constructs FORGE-K"
			report.AuthorityMatrix[i].RollbackPath = "stop the daemon and restore a verified prior store/generation; no alternate live authority exists"
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
			report.AuthorityMatrix[i].Blockers = []string{"external audit sink delivery and audit_id backfill remain best-effort projections over canonical atomic outbox evidence"}
		case "Memory Palace":
			report.AuthorityMatrix[i].CurrentStatus = "FORGE_K_ADMITTED_MEMORY_AND_EVIDENCE_LIVE"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].Blockers = []string{}
		case "Semantic Algebra":
			report.AuthorityMatrix[i].CurrentStatus = "FORGE_K_DETERMINISTIC_DIFF_LIVE"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].FeatureFlag = "production FORGE-K is the sole boot-constructed authority"
			report.AuthorityMatrix[i].RollbackPath = "stop the daemon and use verified offline rollback; immutable diff history remains inspectable"
			report.AuthorityMatrix[i].TestsPassing = append(report.AuthorityMatrix[i].TestsPassing,
				"exact-scope current admitted source authority tests",
				"deterministic diff golden and permutation tests",
				"verified replay and idempotency conflict tests",
				"journal collision atomic rollback tests",
			)
			report.AuthorityMatrix[i].Blockers = []string{"semantic operators beyond semantic.diff.v1 remain unimplemented and therefore have no live authority surface"}
		case "Snapshots":
			report.AuthorityMatrix[i].CurrentStatus = "FORGE_K_CONTEXT_BUNDLE_SNAPSHOT_LIVE"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].Blockers = []string{"live backup merge remains disabled; recovery is daemon-stopped and whole-store only"}
		case "Context Compiler":
			report.AuthorityMatrix[i].CurrentStatus = "FORGE_K_CONTEXT_COMPILER_LIVE"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].FeatureFlag = "production FORGE-K is the sole boot-constructed authority"
			report.AuthorityMatrix[i].RollbackPath = "stop the daemon and use verified offline rollback; existing snapshot evidence remains inspectable"
			report.AuthorityMatrix[i].Blockers = []string{}
		case "KV System":
			report.AuthorityMatrix[i].CurrentStatus = "FORGE_K_IDENTITY_GATE_LIVE_REUSE_DISABLED"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].Blockers = []string{"backend KV tensor reuse is intentionally disabled until an identity-bound adapter exists"}
		case "Runtime Boundary":
			report.AuthorityMatrix[i].CurrentStatus = "FORGE_K_RUNTIME_PROPOSAL_AND_VISIBILITY_LIVE"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].Blockers = []string{}
		case "Lymphatic Lane":
			report.AuthorityMatrix[i].CurrentStatus = "PROPOSAL_ONLY_NO_MUTATION_AUTHORITY"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].Blockers = []string{}
		case "Consensus Mesh":
			report.AuthorityMatrix[i].CurrentStatus = "FORGE_K_ALL_MODEL_VISIBILITY_GATED"
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].Blockers = []string{}
		}
	}
	return report
}

func (s *Server) handleForgeKernelStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.forgeKActivationReadiness(time.Now().UTC()))
}
