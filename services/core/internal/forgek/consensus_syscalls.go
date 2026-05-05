package forgek

import (
	"errors"

	"forge/projectforge/services/core/internal/forgek/consensus"
)

func (k *Kernel) registerConsensusSyscalls(register func(SyscallDefinition)) {
	for _, definition := range []SyscallDefinition{
		{Name: SyscallConsensusOpen, Handler: handleConsensusOpen},
		{Name: SyscallConsensusSubmitClaim, Handler: handleConsensusSubmitClaim},
		{Name: SyscallConsensusSubmitEvidence, Handler: handleConsensusSubmitEvidence},
		{Name: SyscallConsensusEvaluate, Handler: handleConsensusEvaluate},
	} {
		definition.Version = "v1"
		definition.AllowedLanes = []string{"arterial"}
		definition.Deterministic = true
		definition.SideEffects = true
		definition.JournalRequired = true
		definition.Replayable = true
		register(definition)
	}
	for _, definition := range []SyscallDefinition{
		{Name: SyscallConsensusGetReport, Handler: handleConsensusGetReport},
		{Name: SyscallConsensusListReports, Handler: handleConsensusListReports},
		{Name: SyscallConsensusBuildComposerInput, Handler: handleConsensusBuildComposerInput},
		{Name: SyscallConsensusRead, Handler: handleConsensusRead},
	} {
		definition.Version = "v1"
		definition.AllowedLanes = []string{"arterial"}
		definition.Deterministic = true
		definition.Replayable = true
		register(definition)
	}
}

func handleConsensusOpen(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	opened, err := kernel.consensus.OpenRequest(consensusRequestFromSyscall(kernel, request), consensusPolicyFromSyscall(kernel, request))
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapConsensusError(err)}
	}
	event, err := kernel.appendConsensusEvent(request, JournalEventConsensusOpened, opened.RequestID, opened.CaseID, capabilityRefs, opened, []string{opened.RequestID})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	kernel.objects.putObject(consensusRequestObject(opened, request.ActorID, capabilityRefs, event.EventID))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: opened.RequestID, JournalEvent: event.EventID, Output: opened}
}

func handleConsensusSubmitClaim(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	requestID := stringInput(request.Input, "request_id")
	opened, ok := kernel.consensus.GetRequest(requestID)
	if !ok || opened.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	claim, err := kernel.consensus.SubmitClaim(consensusClaimInputFromSyscall(kernel, request))
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapConsensusError(err)}
	}
	event, err := kernel.appendConsensusEvent(request, JournalEventConsensusClaimSubmitted, claim.ClaimID, opened.CaseID, capabilityRefs, claim, append([]string{claim.RequestID}, claim.EvidenceRefs...))
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	kernel.objects.putObject(consensusClaimObject(claim, opened.WorkspaceID, request.ActorID, capabilityRefs, event.EventID))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: claim.ClaimID, JournalEvent: event.EventID, Output: claim}
}

func handleConsensusSubmitEvidence(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	requestID := stringInput(request.Input, "request_id")
	opened, ok := kernel.consensus.GetRequest(requestID)
	if requestID != "" && (!ok || opened.WorkspaceID != request.WorkspaceID) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	ref, err := kernel.consensus.SubmitEvidence(consensusEvidenceFromSyscall(kernel, request))
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapConsensusError(err)}
	}
	event, err := kernel.appendConsensusEvent(request, JournalEventConsensusEvidenceSubmitted, ref.EvidenceID, firstNonEmpty(opened.CaseID, request.CaseID), capabilityRefs, ref, []string{requestID, ref.EvidenceID})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	kernel.objects.putObject(consensusEvidenceObject(ref, request.WorkspaceID, request.ActorID, capabilityRefs, event.EventID))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: ref.EvidenceID, JournalEvent: event.EventID, Output: ref}
}

func handleConsensusEvaluate(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	requestID := stringInput(request.Input, "request_id")
	opened, ok := kernel.consensus.GetRequest(requestID)
	if !ok || opened.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	reportID := firstNonEmpty(stringInput(request.Input, "report_id"), kernel.ids.NextID("consensus-report"))
	report, err := kernel.consensus.Evaluate(requestID, reportID, kernel.clock.Now())
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapConsensusError(err)}
	}
	event, err := kernel.appendConsensusEvent(request, JournalEventConsensusEvaluated, report.ReportID, report.CaseID, capabilityRefs, report, consensusReportRefs(report))
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	report.JournalRefs = []string{event.EventID}
	kernel.consensus.StoreReport(report)
	kernel.objects.putObject(consensusReportObject(report, request.ActorID, capabilityRefs, event.EventID))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: report.ReportID, JournalEvent: event.EventID, Output: report}
}

func handleConsensusGetReport(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadConsensus(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	reportID := stringInput(request.Input, "report_id")
	report, ok := kernel.consensus.GetReport(reportID)
	if !ok || report.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: reportID, Output: report}
}

func handleConsensusListReports(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadConsensus(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.consensus.ListReports(consensus.ReportListFilter{
		WorkspaceID: request.WorkspaceID,
		RequestID:   stringInput(request.Input, "request_id"),
		CaseID:      stringInput(request.Input, "case_id"),
	})}
}

func handleConsensusBuildComposerInput(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadConsensus(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	reportID := stringInput(request.Input, "report_id")
	report, ok := kernel.consensus.GetReport(reportID)
	if !ok || report.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	input, err := consensus.BuildComposerInput(
		firstNonEmpty(stringInput(request.Input, "input_id"), kernel.ids.NextID("composer-input")),
		report,
		kernel.consensus.ListClaims(consensus.ClaimListFilter{RequestID: report.RequestID}),
		stringSliceInputDefault(request.Input, "style_constraints"),
		stringInput(request.Input, "user_current_turn_text"),
		kernel.clock.Now(),
	)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapConsensusError(err)}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: input.InputID, Output: input}
}

func handleConsensusRead(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadConsensus(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	requestID := stringInput(request.Input, "request_id")
	claims := []consensus.Claim{}
	if requestID != "" {
		opened, ok := kernel.consensus.GetRequest(requestID)
		if !ok || opened.WorkspaceID != request.WorkspaceID {
			return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
		}
		claims = kernel.consensus.ListClaims(consensus.ClaimListFilter{RequestID: requestID})
	}
	output := map[string]any{
		"reports": kernel.consensus.ListReports(consensus.ReportListFilter{WorkspaceID: request.WorkspaceID, CaseID: stringInput(request.Input, "case_id")}),
		"claims":  claims,
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: output}
}

func consensusRequestFromSyscall(kernel *Kernel, request SyscallRequest) consensus.ConsensusRequest {
	return consensus.ConsensusRequest{
		RequestID:   firstNonEmpty(stringInput(request.Input, "request_id"), kernel.ids.NextID("consensus-request")),
		WorkspaceID: request.WorkspaceID,
		CaseID:      firstNonEmpty(stringInput(request.Input, "case_id"), request.CaseID),
		PolicyID:    stringInput(request.Input, "policy_id"),
		OpenedBy:    request.ActorID,
		OpenedAt:    kernel.clock.Now(),
		Metadata:    mapInput(request.Input, "metadata"),
	}
}

func consensusPolicyFromSyscall(kernel *Kernel, request SyscallRequest) consensus.ConsensusPolicy {
	criticality := consensus.Criticality(firstNonEmpty(stringInput(request.Input, "criticality"), string(consensus.CriticalityLow)))
	policy := consensus.DefaultPolicy(request.WorkspaceID, criticality, kernel.clock.Now())
	if policyID := stringInput(request.Input, "policy_id"); policyID != "" {
		policy.PolicyID = policyID
	}
	if required := intInput(request.Input, "required_agents"); required > 0 {
		policy.RequiredAgents = required
	}
	if tier1 := intInput(request.Input, "required_tier1_count"); tier1 >= 0 {
		policy.RequiredTier1Count = tier1
	}
	if ratio := floatInput(request.Input, "min_support_ratio"); ratio > 0 {
		policy.MinSupportRatio = ratio
	}
	if ratio := floatInput(request.Input, "max_conflict_ratio"); ratio > 0 {
		policy.MaxConflictRatio = ratio
	}
	policy.Metadata = mapInput(request.Input, "policy_metadata")
	return policy
}

func consensusClaimInputFromSyscall(kernel *Kernel, request SyscallRequest) consensus.ClaimInput {
	return consensus.ClaimInput{
		ClaimID:      firstNonEmpty(stringInput(request.Input, "claim_id"), kernel.ids.NextID("claim")),
		RequestID:    stringInput(request.Input, "request_id"),
		ClaimType:    consensus.ClaimType(stringInput(request.Input, "claim_type")),
		Subject:      stringInput(request.Input, "subject"),
		Predicate:    stringInput(request.Input, "predicate"),
		ValueJSON:    request.Input["value_json"],
		Scope:        stringInput(request.Input, "scope"),
		Temporal:     stringInput(request.Input, "temporal"),
		EvidenceRefs: stringSliceInputDefault(request.Input, "evidence_refs"),
		Confidence:   floatInput(request.Input, "confidence"),
		AgentID:      stringInput(request.Input, "agent_id"),
		AgentRunID:   stringInput(request.Input, "agent_run_id"),
		RiskFlags:    stringSliceInputDefault(request.Input, "risk_flags"),
		CreatedAt:    kernel.clock.Now(),
		Metadata:     mapInput(request.Input, "metadata"),
	}
}

func consensusEvidenceFromSyscall(kernel *Kernel, request SyscallRequest) consensus.EvidenceRef {
	return consensus.EvidenceRef{
		EvidenceID:       firstNonEmpty(stringInput(request.Input, "evidence_id"), kernel.ids.NextID("evidence")),
		EvidenceType:     consensus.EvidenceType(stringInput(request.Input, "evidence_type")),
		Tier:             consensus.EvidenceTier(stringInput(request.Input, "tier")),
		Source:           stringInput(request.Input, "source"),
		Locator:          stringInput(request.Input, "locator"),
		RetrievedAt:      kernel.clock.Now(),
		FreshnessScore:   firstNonZeroFloat(floatInput(request.Input, "freshness_score"), 1),
		ReliabilityScore: firstNonZeroFloat(floatInput(request.Input, "reliability_score"), 1),
		SourceHash:       stringInput(request.Input, "source_hash"),
		Metadata:         mapInput(request.Input, "metadata"),
	}
}

func (k *Kernel) appendConsensusEvent(request SyscallRequest, eventType, objectID, caseID string, capabilityRefs []string, output any, objectRefs []string) (JournalEvent, error) {
	refs := append([]string{objectID}, objectRefs...)
	return k.journal.Append(JournalEvent{
		EventType:      eventType,
		Timestamp:      k.clock.Now(),
		WorkspaceID:    request.WorkspaceID,
		CaseID:         caseID,
		ActorID:        request.ActorID,
		SyscallName:    request.Name,
		InputHash:      hashValue(request.Input),
		OutputHash:     hashValue(output),
		ObjectRefs:     snapshotsNormalizeRefs(refs),
		CapabilityRefs: capabilityRefs,
		Result:         SyscallResultCommitted,
	})
}

func consensusRequestObject(opened consensus.ConsensusRequest, ownerID string, capabilityRefs []string, journalRef string) KernelObject {
	return KernelObject{
		ObjectID:       opened.RequestID,
		ObjectType:     ObjectTypeConsensusRequest,
		WorkspaceID:    opened.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityProposal,
		State: map[string]any{
			"is_canonical_truth": false,
			"can_mutate_kernel":  false,
			"can_admit_evidence": false,
		},
		SourceRefs:      []string{opened.RequestID},
		CapabilityScope: capabilityRefs,
		CreatedAt:       opened.OpenedAt,
		UpdatedAt:       opened.OpenedAt,
		JournalRefs:     []string{journalRef},
	}
}

func consensusClaimObject(claim consensus.Claim, workspaceID string, ownerID string, capabilityRefs []string, journalRef string) KernelObject {
	return KernelObject{
		ObjectID:       claim.ClaimID,
		ObjectType:     ObjectTypeConsensusClaim,
		WorkspaceID:    workspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityProposal,
		State: map[string]any{
			"claim_key":            claim.ClaimKey,
			"claim_type":           string(claim.ClaimType),
			"status":               string(claim.Status),
			"is_canonical_truth":   false,
			"is_admitted_evidence": false,
			"executes_action":      false,
			"writes_memory":        false,
		},
		SourceRefs:      append([]string{claim.RequestID}, claim.EvidenceRefs...),
		CapabilityScope: capabilityRefs,
		CreatedAt:       claim.CreatedAt,
		UpdatedAt:       claim.CreatedAt,
		JournalRefs:     []string{journalRef},
	}
}

func consensusEvidenceObject(ref consensus.EvidenceRef, workspaceID string, ownerID string, capabilityRefs []string, journalRef string) KernelObject {
	return KernelObject{
		ObjectID:       ref.EvidenceID,
		ObjectType:     ObjectTypeConsensusEvidence,
		WorkspaceID:    workspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityProposal,
		State: map[string]any{
			"evidence_type":        string(ref.EvidenceType),
			"tier":                 string(ref.Tier),
			"is_admitted_evidence": false,
			"is_canonical_truth":   false,
		},
		SourceRefs:      []string{ref.EvidenceID, ref.Source},
		CapabilityScope: capabilityRefs,
		CreatedAt:       ref.RetrievedAt,
		UpdatedAt:       ref.RetrievedAt,
		JournalRefs:     []string{journalRef},
	}
}

func consensusReportObject(report consensus.ConsensusReport, ownerID string, capabilityRefs []string, journalRef string) KernelObject {
	return KernelObject{
		ObjectID:       report.ReportID,
		ObjectType:     ObjectTypeConsensusReport,
		WorkspaceID:    report.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityValidated,
		State: map[string]any{
			"is_canonical_truth":   false,
			"is_admitted_evidence": false,
			"commits_memory":       false,
			"executes_action":      false,
			"calls_model_runtime":  false,
		},
		SourceRefs:      consensusReportRefs(report),
		CapabilityScope: capabilityRefs,
		CreatedAt:       report.CreatedAt,
		UpdatedAt:       report.CreatedAt,
		JournalRefs:     []string{journalRef},
	}
}

func composerInputObject(input consensus.ResponseCompositionInput, ownerID string) KernelObject {
	return KernelObject{
		ObjectID:       input.InputID,
		ObjectType:     ObjectTypeComposerInput,
		WorkspaceID:    input.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityProposal,
		State: map[string]any{
			"accepted_claims_only": true,
			"is_canonical_truth":   false,
			"executes_action":      false,
			"writes_memory":        false,
		},
		SourceRefs: append([]string{input.ReportID}, input.ResponseTrace...),
		CreatedAt:  input.CreatedAt,
		UpdatedAt:  input.CreatedAt,
	}
}

func consensusReportRefs(report consensus.ConsensusReport) []string {
	refs := append([]string{report.RequestID}, report.AcceptedClaimIDs...)
	refs = append(refs, report.UncertainClaimIDs...)
	refs = append(refs, report.RejectedClaimIDs...)
	refs = append(refs, report.ConflictedClaimIDs...)
	return snapshotsNormalizeRefs(refs)
}

func (k *Kernel) canReadConsensus(request SyscallRequest) bool {
	allowed, _ := k.capabilities.CanCall(request.ActorID, request.WorkspaceID, request.Name, false, k.clock.Now())
	if allowed {
		return true
	}
	allowed, _ = k.capabilities.CanCall(request.ActorID, request.WorkspaceID, SyscallConsensusRead, false, k.clock.Now())
	return allowed
}

func mapConsensusError(err error) error {
	switch {
	case errors.Is(err, consensus.ErrObjectNotFound):
		return ErrObjectNotFound
	case errors.Is(err, consensus.ErrInvalidClaim), errors.Is(err, consensus.ErrInvalidEvidenceRef),
		errors.Is(err, consensus.ErrInvalidAgentRun), errors.Is(err, consensus.ErrInvalidPolicy),
		errors.Is(err, consensus.ErrInvalidRequest), errors.Is(err, consensus.ErrComposerInputRejected),
		errors.Is(err, consensus.ErrRejectedClaimReference):
		return ErrInvalidInput
	default:
		return err
	}
}

func firstNonZeroFloat(values ...float64) float64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
