package api

import (
	"context"
	"net/http"
	"time"

	"forge/projectforge/services/core/internal/aios/controllane"
)

func (s *Server) forgeKActivationReadiness(now time.Time) controllane.ForgeKActivationReadinessReport {
	report := controllane.ForgeKActivationReadiness(controllane.NewStaticActionRegistry(), now)
	selection := s.kernelAuthority
	kernelAuthorityExclusive := selection.SingleAuthority && selection.Processor != nil
	hostReady := false
	if s != nil && s.st != nil && s.st.DB != nil {
		hostReady = s.st.DB.Ping() == nil
	}
	report.KernelAuthorityExclusive = kernelAuthorityExclusive
	readinessComplete := report.Status == "partial_live_validation_ready"
	report.HostReady = hostReady
	report.UnsafeTestMode = parseEnvBoolWithDefault("FORGE_UNSAFE_TEST_MODE", false)
	if kernelAuthorityExclusive {
		report.LiveOwner = selection.AuthorityOwner
	}
	report.CapabilityImplemented = report.Status == "partial_live_validation_ready"
	report.ProjectionHealthy = false
	if s != nil && s.auditProjectionReady && s.st != nil && s.st.DB != nil {
		if projection, err := controllane.ReadAuditProjectionStatus(context.Background(), s.st.DB); err == nil {
			report.ProjectionHealthy = projection.Healthy
			if report.NoEffect == nil {
				report.NoEffect = map[string]any{}
			}
			report.NoEffect["externalAuditProjection"] = projection
		} else if report.NoEffect == nil {
			report.NoEffect = map[string]any{}
		}
	}
	report.RecoveryVerified = report.KernelAuthorityExclusive && report.CapabilityImplemented && report.ProjectionHealthy && report.HostReady
	if !kernelAuthorityExclusive {
		report.Status = "kernel_authority_unavailable"
		report.Summary = "No production semantic kernel authority was selected at daemon boot; semantic mutation paths are fail-closed."
		report.Mode = "fail_closed"
		report.LiveOwner = "none"
		report.KernelRuntimeState = "unavailable"
		report.LiveKernelIngressAuthority = false
		report.LiveDurableOrchestration = false
		report.LiveKernelAuthority = false
		report.LiveAuthorityMigration = false
		report.RecoveryVerified = false
		if s.kernelErr != "" {
			report.Notes = append(report.Notes, "boot authority selection failed: "+s.kernelErr)
		}
		if report.NoEffect == nil {
			report.NoEffect = map[string]any{}
		}
		report.NoEffect["kernelIngressAuthority"] = false
		report.NoEffect["durableOrchestrationAuthority"] = false
		report.NoEffect["kernelContextCompilerAuthority"] = false
		report.NoEffect["kernelRuntimeProposalAuthority"] = false
		report.NoEffect["externalAuditSinkDelivery"] = s.auditProjectionReady
		if s.auditProjectionReady && s.st != nil && s.st.DB != nil {
			projection, err := controllane.ReadAuditProjectionStatus(context.Background(), s.st.DB)
			if err != nil {
				report.NoEffect["externalAuditProjection"] = map[string]any{"healthy": false, "healthReason": err.Error()}
			}
		} else {
			reason := s.auditProjectionErr
			if reason == "" {
				reason = "durable projector is not assembled"
			}
			report.NoEffect["externalAuditProjection"] = map[string]any{"healthy": false, "healthReason": reason}
		}
		return report
	}
	report.Notes = append(report.Notes,
		"Kernel authorization is selection-bound and single-source in the production bootstrap path.",
		"FORGE-K controls staged authority for kernel ingress/durable orchestration assertions in this phase; simulator services remain non-authoritative.",
	)
	report.CapabilityImplemented = readinessComplete
	liveAuthorityDerived := readinessComplete && report.HostReady
	report.LiveKernelAuthority = false
	report.LiveAuthorityMigration = false
	report.LiveKernelIngressAuthority = liveAuthorityDerived
	report.LiveDurableOrchestration = liveAuthorityDerived
	report.NoEffect["kernelContextCompilerAuthority"] = false
	report.NoEffect["kernelRuntimeProposalAuthority"] = false
	report.NoEffect["externalAuditSinkDelivery"] = s.auditProjectionReady
	report.NoEffect["commitIntegrityAuthority"] = report.LiveAuthorityMigration == false
	report.NoEffect["sealedPreparedPlans"] = true
	report.NoEffect["typedCommitReceiptValidation"] = true
	report.NoEffect["atomicAuditOutboxEvidence"] = true
	report.NoEffect["verifiedIdempotentReplay"] = true
	report.NoEffect["authenticatedAuthorizationProof"] = s.kernelAuthorizationReady
	report.NoEffect["durableAuthorizationReplayProof"] = s.kernelAuthorizationReady
	report.NoEffect["uniqueForgeCoreServicePrincipal"] = s.kernelAuthorizationReady
	report.NoEffect["authenticatedUserOriginRequired"] = s.kernelAuthorizationReady
	report.NoEffect["unauthenticatedRemoteOrigin"] = false
	report.NoEffect["kernelIngressAuthority"] = report.LiveKernelIngressAuthority == false
	report.NoEffect["durableOrchestrationAuthority"] = report.LiveDurableOrchestration == false
	report.NoEffect["liveAuthorityMigration"] = false
	report.NoEffect["legacyLiveAuthority"] = false
	report.NoEffect["backendKVReuse"] = false
	report.NoEffect["autonomousCanonicalMutation"] = false

	for i := range report.ValidationActions {
		report.ValidationActions[i].LiveOwner = selection.AuthorityOwner
		report.ValidationActions[i].LiveKernelAuthority = false
	}

	for i := range report.AuthorityMatrix {
		switch report.AuthorityMatrix[i].Subsystem {
		case "Kernel":
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
		case "Courthouse":
			report.AuthorityMatrix[i].LiveOwner = selection.AuthorityOwner
			report.AuthorityMatrix[i].TargetOwner = selection.AuthorityOwner
		}
	}
	return report
}

func (s *Server) handleForgeKernelStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.forgeKActivationReadiness(time.Now().UTC()))
}
