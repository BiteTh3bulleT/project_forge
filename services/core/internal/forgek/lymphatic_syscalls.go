package forgek

import (
	"errors"

	"forge/projectforge/services/core/internal/forgek/contextcompiler"
	"forge/projectforge/services/core/internal/forgek/kv"
	"forge/projectforge/services/core/internal/forgek/lymphatic"
	forgekRuntime "forge/projectforge/services/core/internal/forgek/runtime"
	"forge/projectforge/services/core/internal/forgek/snapshots"
)

func (k *Kernel) registerLymphaticSyscalls(register func(SyscallDefinition)) {
	for _, definition := range []SyscallDefinition{
		{Name: SyscallLymphRunSweep, Handler: handleLymphRunSweep},
		{Name: SyscallLymphCreateProposal, Handler: handleLymphCreateProposal},
	} {
		definition.Version = "v1"
		definition.AllowedLanes = []string{"lymphatic"}
		definition.Deterministic = true
		definition.SideEffects = true
		definition.JournalRequired = true
		definition.Replayable = true
		register(definition)
	}
	for _, definition := range []SyscallDefinition{
		{Name: SyscallLymphGetReport, Handler: handleLymphGetReport},
		{Name: SyscallLymphListReports, Handler: handleLymphListReports},
		{Name: SyscallLymphGetProposal, Handler: handleLymphGetProposal},
		{Name: SyscallLymphListProposals, Handler: handleLymphListProposals},
		{Name: SyscallLymphRead, Handler: handleLymphRead},
	} {
		definition.Version = "v1"
		definition.AllowedLanes = []string{"lymphatic"}
		definition.Deterministic = true
		definition.Replayable = true
		register(definition)
	}
}

func handleLymphRunSweep(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	sweepRequest := lymphSweepRequestFromSyscall(kernel, request)
	policy := lymphPolicyFromSyscall(kernel, request, sweepRequest)
	sources := lymphSourcesFromKernel(kernel, sweepRequest)
	report, err := kernel.lymphatic.RunSweep(sweepRequest, policy, sources)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapLymphError(err)}
	}
	event, err := kernel.appendLymphEvent(request, JournalEventLymphaticSweepCompleted, report.ReportID, report.CaseID, capabilityRefs, report, lymphReportRefs(report))
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	report.JournalRefs = []string{event.EventID}
	kernel.lymphatic.StoreReport(report)
	kernel.objects.putObject(lymphReportObject(report, request.ActorID, capabilityRefs))
	for _, proposal := range report.CleanupProposals {
		kernel.objects.putObject(lymphProposalObject(proposal, request.ActorID, capabilityRefs, event.EventID))
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: report.ReportID, JournalEvent: event.EventID, Output: report}
}

func handleLymphCreateProposal(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	proposal := lymphProposalFromSyscall(kernel, request)
	if err := validateLymphTargetRefs(kernel, proposal.WorkspaceID, proposal.TargetObjectRefs); err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	created, err := kernel.lymphatic.CreateProposal(proposal)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapLymphError(err)}
	}
	event, err := kernel.appendLymphEvent(request, JournalEventLymphaticProposalCreated, created.ProposalID, created.CaseID, capabilityRefs, created, created.TargetObjectRefs)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	created.Metadata = lymphatic.CloneMap(created.Metadata)
	if created.Metadata == nil {
		created.Metadata = map[string]any{}
	}
	created.Metadata["journal_ref"] = event.EventID
	kernel.lymphatic.StoreProposal(created)
	kernel.objects.putObject(lymphProposalObject(created, request.ActorID, capabilityRefs, event.EventID))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: created.ProposalID, JournalEvent: event.EventID, Output: created}
}

func handleLymphGetReport(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadLymph(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	reportID := stringInput(request.Input, "report_id")
	report, ok := kernel.lymphatic.GetReport(reportID)
	if !ok || report.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: reportID, Output: report}
}

func handleLymphListReports(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadLymph(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	filter := lymphatic.ReportListFilter{
		WorkspaceID: request.WorkspaceID,
		CaseID:      stringInput(request.Input, "case_id"),
		Status:      lymphatic.ReportStatus(stringInput(request.Input, "status")),
		SweepKind:   lymphatic.SweepKind(stringInput(request.Input, "sweep_kind")),
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.lymphatic.ListReports(filter)}
}

func handleLymphGetProposal(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadLymph(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	proposalID := stringInput(request.Input, "proposal_id")
	proposal, ok := kernel.lymphatic.GetProposal(proposalID)
	if !ok || proposal.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: proposalID, Output: proposal}
}

func handleLymphListProposals(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadLymph(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	filter := lymphatic.ProposalListFilter{
		WorkspaceID:  request.WorkspaceID,
		CaseID:       stringInput(request.Input, "case_id"),
		ProposalType: lymphatic.ProposalType(stringInput(request.Input, "proposal_type")),
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.lymphatic.ListProposals(filter)}
}

func handleLymphRead(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadLymph(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	output := map[string]any{
		"reports": kernel.lymphatic.ListReports(lymphatic.ReportListFilter{
			WorkspaceID: request.WorkspaceID,
			CaseID:      stringInput(request.Input, "case_id"),
		}),
		"proposals": kernel.lymphatic.ListProposals(lymphatic.ProposalListFilter{
			WorkspaceID: request.WorkspaceID,
			CaseID:      stringInput(request.Input, "case_id"),
		}),
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: output}
}

func lymphSweepRequestFromSyscall(kernel *Kernel, request SyscallRequest) lymphatic.LymphaticSweepRequest {
	caseID := firstNonEmpty(stringInput(request.Input, "case_id"), request.CaseID)
	return lymphatic.LymphaticSweepRequest{
		RequestID:      firstNonEmpty(stringInput(request.Input, "request_id"), kernel.ids.NextID("lymph-sweep")),
		WorkspaceID:    request.WorkspaceID,
		CaseID:         caseID,
		SweepKinds:     lymphSweepKindsInput(request.Input),
		PolicyID:       stringInput(request.Input, "policy_id"),
		DryRun:         true,
		IncludeDetails: boolInput(request.Input, "include_details"),
		RequestedBy:    request.ActorID,
		CreatedAt:      kernel.clock.Now(),
		Metadata:       mapInput(request.Input, "metadata"),
	}
}

func lymphPolicyFromSyscall(kernel *Kernel, request SyscallRequest, sweepRequest lymphatic.LymphaticSweepRequest) lymphatic.LymphaticPolicy {
	policy := lymphatic.DefaultPolicy(sweepRequest.WorkspaceID, kernel.clock.Now())
	if policyID := stringInput(request.Input, "policy_id"); policyID != "" {
		policy.PolicyID = policyID
	}
	if max := intInput(request.Input, "max_report_items"); max > 0 {
		policy.MaxReportItems = max
	}
	if stale := intInput(request.Input, "runtime_result_stale_after_ms"); stale > 0 {
		policy.RuntimeResultStaleAfterMS = int64(stale)
	}
	if expiry := intInput(request.Input, "snapshot_expire_after_duration_ms"); expiry > 0 {
		policy.SnapshotExpireAfterDurationMS = int64(expiry)
	}
	if expiry := intInput(request.Input, "kv_expire_after_duration_ms"); expiry > 0 {
		policy.KvExpireAfterDurationMS = int64(expiry)
	}
	if cold := intInput(request.Input, "kv_cold_after_reuse_count"); cold >= 0 {
		policy.KvColdAfterReuseCount = cold
	}
	policy.IncludeRejectedEvidence = boolInput(request.Input, "include_rejected_evidence")
	policy.IncludeSupersededObjects = true
	policy.IncludeInvalidatedKV = true
	policy.DryRun = true
	policy.Metadata = mapInput(request.Input, "policy_metadata")
	return policy
}

func lymphSourcesFromKernel(kernel *Kernel, request lymphatic.LymphaticSweepRequest) lymphatic.SweepSources {
	sources := lymphatic.SweepSources{
		Snapshots:      kernel.snapshots.ListSnapshots(snapshots.ListFilter{WorkspaceID: request.WorkspaceID, CaseID: request.CaseID}),
		KVManifests:    kernel.kv.ListManifests(kv.ManifestListFilter{WorkspaceID: request.WorkspaceID, CaseID: request.CaseID}),
		RuntimeResults: filterRuntimeResultsByCase(kernel.runtime.ListResults(request.WorkspaceID), request.CaseID),
		ContextBundles: kernel.context.ListBundles(contextcompiler.BundleListFilter{WorkspaceID: request.WorkspaceID, CaseID: request.CaseID}),
		ContextBlocks:  kernel.context.ListBlocks(contextcompiler.BlockListFilter{WorkspaceID: request.WorkspaceID, CaseID: request.CaseID}),
		KnownRefKinds:  map[string]string{},
		Now:            kernel.clock.Now(),
	}
	for _, obj := range kernel.objects.ListObjects() {
		if obj.WorkspaceID != request.WorkspaceID {
			continue
		}
		sources.KnownObjectRefs = append(sources.KnownObjectRefs, obj.ObjectID)
		sources.KnownRefKinds[obj.ObjectID] = obj.ObjectType
		if obj.ObjectType == ObjectTypeCasePacket && (request.CaseID == "" || request.CaseID == obj.ObjectID) {
			sources.Contradictions = append(sources.Contradictions, kernel.court.ListCaseContradictions(obj.ObjectID)...)
		}
	}
	if request.CaseID != "" && len(sources.Contradictions) == 0 {
		sources.Contradictions = append(sources.Contradictions, kernel.court.ListCaseContradictions(request.CaseID)...)
	}
	return sources
}

func lymphProposalFromSyscall(kernel *Kernel, request SyscallRequest) lymphatic.CleanupProposal {
	proposal := lymphatic.CleanupProposal{
		ProposalID:          firstNonEmpty(stringInput(request.Input, "proposal_id"), kernel.ids.NextID("lymph-proposal")),
		ProposalType:        lymphatic.ProposalType(stringInput(request.Input, "proposal_type")),
		WorkspaceID:         request.WorkspaceID,
		CaseID:              firstNonEmpty(stringInput(request.Input, "case_id"), request.CaseID),
		TargetObjectRefs:    stringSliceInputDefault(request.Input, "target_object_refs"),
		ProposedSyscallName: stringInput(request.Input, "proposed_syscall_name"),
		ProposedPayload:     mapInput(request.Input, "proposed_payload"),
		Reason:              stringInput(request.Input, "reason"),
		SafetyNotes:         stringSliceInputDefault(request.Input, "safety_notes"),
		RequiresReview:      true,
		CreatedAt:           kernel.clock.Now(),
		Metadata:            mapInput(request.Input, "metadata"),
	}
	if proposal.ProposalType == "" {
		proposal.ProposalType = lymphatic.ProposalNoOpReview
	}
	if len(proposal.SafetyNotes) == 0 {
		proposal.SafetyNotes = []string{"dry_run_only", "does_not_execute_cleanup"}
	}
	return proposal
}

func lymphSweepKindsInput(input map[string]any) []lymphatic.SweepKind {
	values := stringSliceInputDefault(input, "sweep_kinds")
	if single := stringInput(input, "sweep_kind"); single != "" {
		values = append(values, single)
	}
	out := make([]lymphatic.SweepKind, 0, len(values))
	for _, value := range values {
		out = append(out, lymphatic.SweepKind(value))
	}
	return lymphatic.NormalizeSweepKinds(out)
}

func filterRuntimeResultsByCase(results []forgekRuntime.RuntimeGenerateResult, caseID string) []forgekRuntime.RuntimeGenerateResult {
	if caseID == "" {
		return results
	}
	out := make([]forgekRuntime.RuntimeGenerateResult, 0)
	for _, result := range results {
		if result.CaseID == "" || result.CaseID == caseID {
			out = append(out, result)
		}
	}
	return out
}

func validateLymphTargetRefs(kernel *Kernel, workspaceID string, refs []string) error {
	if len(refs) == 0 {
		return ErrInvalidInput
	}
	for _, ref := range refs {
		obj, ok := kernel.objects.GetObject(ref)
		if !ok || obj.WorkspaceID != workspaceID {
			return ErrObjectNotFound
		}
	}
	return nil
}

func (k *Kernel) appendLymphEvent(request SyscallRequest, eventType, objectID, caseID string, capabilityRefs []string, output any, objectRefs []string) (JournalEvent, error) {
	return k.journal.Append(JournalEvent{
		EventType:      eventType,
		Timestamp:      k.clock.Now(),
		WorkspaceID:    request.WorkspaceID,
		CaseID:         caseID,
		ActorID:        request.ActorID,
		SyscallName:    request.Name,
		InputHash:      hashValue(request.Input),
		OutputHash:     hashValue(output),
		ObjectRefs:     snapshotsNormalizeRefs(append([]string{objectID}, objectRefs...)),
		CapabilityRefs: append([]string(nil), capabilityRefs...),
		Result:         SyscallResultCommitted,
	})
}

func lymphReportObject(report lymphatic.MaintenanceReport, ownerID string, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       report.ReportID,
		ObjectType:     ObjectTypeMaintenanceReport,
		WorkspaceID:    report.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityProposal,
		State: map[string]any{
			"case_id":                report.CaseID,
			"status":                 string(report.Status),
			"sweep_kinds":            sweepKindStrings(report.SweepKinds),
			"finding_count":          len(report.Findings),
			"proposal_count":         len(report.CleanupProposals),
			"summary":                report.Summary,
			"dry_run":                true,
			"is_canonical_truth":     false,
			"is_admitted_evidence":   false,
			"executes_cleanup":       false,
			"mutates_source_objects": false,
			"calls_live_runtime":     false,
		},
		SourceRefs:      lymphReportRefs(report),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       report.CreatedAt,
		UpdatedAt:       report.CreatedAt,
		JournalRefs:     append([]string(nil), report.JournalRefs...),
	}
}

func lymphProposalObject(proposal lymphatic.CleanupProposal, ownerID string, capabilityRefs []string, journalRef string) KernelObject {
	journalRefs := []string{}
	if journalRef != "" {
		journalRefs = append(journalRefs, journalRef)
	}
	return KernelObject{
		ObjectID:       proposal.ProposalID,
		ObjectType:     ObjectTypeCleanupProposal,
		WorkspaceID:    proposal.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityProposal,
		State: map[string]any{
			"case_id":                 proposal.CaseID,
			"proposal_type":           string(proposal.ProposalType),
			"target_object_refs":      append([]string(nil), proposal.TargetObjectRefs...),
			"proposed_syscall_name":   proposal.ProposedSyscallName,
			"reason":                  proposal.Reason,
			"requires_review":         proposal.RequiresReview,
			"is_canonical_truth":      false,
			"is_admitted_evidence":    false,
			"executes_cleanup":        false,
			"mutates_source_objects":  false,
			"silent_mutation_allowed": false,
		},
		SourceRefs:      append([]string(nil), proposal.TargetObjectRefs...),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       proposal.CreatedAt,
		UpdatedAt:       proposal.CreatedAt,
		JournalRefs:     journalRefs,
	}
}

func lymphReportRefs(report lymphatic.MaintenanceReport) []string {
	refs := make([]string, 0)
	for _, finding := range report.Findings {
		refs = append(refs, finding.ObjectRefs...)
		refs = append(refs, finding.SourceRefs...)
	}
	for _, proposal := range report.CleanupProposals {
		refs = append(refs, proposal.TargetObjectRefs...)
	}
	return snapshotsNormalizeRefs(refs)
}

func sweepKindStrings(values []lymphatic.SweepKind) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}
	return snapshotsNormalizeRefs(out)
}

func (k *Kernel) canReadLymph(request SyscallRequest) bool {
	allowed, _ := k.capabilities.CanCall(request.ActorID, request.WorkspaceID, request.Name, false, k.clock.Now())
	if allowed {
		return true
	}
	allowed, _ = k.capabilities.CanCall(request.ActorID, request.WorkspaceID, SyscallLymphRead, false, k.clock.Now())
	return allowed
}

func mapLymphError(err error) error {
	switch {
	case errors.Is(err, lymphatic.ErrReportNotFound), errors.Is(err, lymphatic.ErrProposalNotFound):
		return ErrObjectNotFound
	case errors.Is(err, lymphatic.ErrInvalidPolicy), errors.Is(err, lymphatic.ErrInvalidSweepKind),
		errors.Is(err, lymphatic.ErrInvalidSweepRequest), errors.Is(err, lymphatic.ErrInvalidProposal):
		return ErrInvalidInput
	default:
		return err
	}
}
