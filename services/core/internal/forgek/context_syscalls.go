package forgek

import (
	"errors"

	"forge/projectforge/services/core/internal/forgek/contextcompiler"
	"forge/projectforge/services/core/internal/forgek/snapshots"
)

func (k *Kernel) registerContextSyscalls(register func(SyscallDefinition)) {
	for _, definition := range []SyscallDefinition{
		{Name: SyscallContextCompile, Handler: handleContextCompile},
		{Name: SyscallContextCompileFromSnapshot, Handler: handleContextCompileFromSnapshot},
		{Name: SyscallContextCompileFromRestoreSeed, Handler: handleContextCompileFromRestoreSeed},
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
		{Name: SyscallContextGetBundle, Handler: handleContextGetBundle},
		{Name: SyscallContextListBundles, Handler: handleContextListBundles},
		{Name: SyscallContextGetBlock, Handler: handleContextGetBlock},
		{Name: SyscallContextListBlocks, Handler: handleContextListBlocks},
		{Name: SyscallContextValidateLayout, Handler: handleContextValidateLayout},
		{Name: SyscallContextHash, Handler: handleContextHash},
	} {
		definition.Version = "v1"
		definition.AllowedLanes = []string{"arterial"}
		definition.Deterministic = true
		definition.Replayable = true
		register(definition)
	}
}

func handleContextCompile(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	compileRequest := contextCompileRequestFromSyscall(kernel, request)
	result, err := kernel.context.Compile(compileRequest)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapContextError(err)}
	}
	return commitContextCompileResult(kernel, request, capabilityRefs, JournalEventContextCompiled, result)
}

func handleContextCompileFromSnapshot(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	snapshotID := stringInput(request.Input, "snapshot_id")
	snapshot, ok := kernel.snapshots.GetSnapshot(snapshotID)
	if !ok || snapshot.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	compileRequest := contextCompileRequestFromSyscall(kernel, request)
	result, err := kernel.context.CompileFromSnapshot(snapshot, compileRequest)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapContextError(err)}
	}
	return commitContextCompileResult(kernel, request, capabilityRefs, JournalEventContextCompiledFromSnapshot, result)
}

func handleContextCompileFromRestoreSeed(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	restoreSeedID := stringInput(request.Input, "restore_seed_id")
	seed, ok := kernel.snapshots.GetRestoreSeed(restoreSeedID)
	if !ok || seed.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	compileRequest := contextCompileRequestFromSyscall(kernel, request)
	result, err := kernel.context.CompileFromRestoreSeed(seed, compileRequest)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapContextError(err)}
	}
	return commitContextCompileResult(kernel, request, capabilityRefs, JournalEventContextCompiledFromRestoreSeed, result)
}

func handleContextGetBundle(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadContext(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	bundleID := stringInput(request.Input, "bundle_id")
	bundle, ok := kernel.context.GetBundle(bundleID)
	if !ok || bundle.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: bundleID, Output: bundle}
}

func handleContextListBundles(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadContext(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	filter := contextcompiler.BundleListFilter{
		WorkspaceID:   request.WorkspaceID,
		CaseID:        stringInput(request.Input, "case_id"),
		SnapshotID:    stringInput(request.Input, "snapshot_id"),
		RestoreSeedID: stringInput(request.Input, "restore_seed_id"),
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.context.ListBundles(filter)}
}

func handleContextGetBlock(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadContext(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	blockID := stringInput(request.Input, "block_id")
	block, ok := kernel.context.GetBlock(blockID)
	if !ok || block.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: blockID, Output: block}
}

func handleContextListBlocks(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadContext(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	filter := contextcompiler.BlockListFilter{
		WorkspaceID: request.WorkspaceID,
		CaseID:      stringInput(request.Input, "case_id"),
		BundleID:    stringInput(request.Input, "bundle_id"),
		BlockType:   contextcompiler.ContextBlockType(stringInput(request.Input, "block_type")),
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.context.ListBlocks(filter)}
}

func handleContextValidateLayout(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadContext(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	layout := contextcompiler.DefaultPromptLayout(
		request.WorkspaceID,
		stringInput(request.Input, "layout_version"),
		stringInput(request.Input, "policy_version"),
		stringInput(request.Input, "syscall_schema_version"),
		kernel.clock.Now(),
		mapInput(request.Input, "metadata"),
	)
	if order, ok := stringSliceInput(request.Input, "block_order"); ok {
		layout.BlockOrder = make([]contextcompiler.ContextBlockType, 0, len(order))
		for _, blockType := range order {
			layout.BlockOrder = append(layout.BlockOrder, contextcompiler.ContextBlockType(blockType))
		}
	}
	if err := contextcompiler.ValidateLayout(layout); err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapContextError(err)}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: layout.LayoutID, Output: layout}
}

func handleContextHash(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadContext(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	text := stringInput(request.Input, "canonical_text")
	if text == "" {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	output := map[string]any{
		"content_hash":         contextcompiler.SHA256Text(text),
		"token_input_hash":     contextcompiler.TokenInputHash(text),
		"token_count_estimate": contextcompiler.EstimateTokens(text),
		"token_hash_scope":     contextcompiler.DefaultTokenizerNeutralHashDescriptor,
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: output}
}

func contextCompileRequestFromSyscall(kernel *Kernel, request SyscallRequest) contextcompiler.ContextCompileRequest {
	caseID := stringInput(request.Input, "case_id")
	if caseID == "" {
		caseID = request.CaseID
	}
	return contextcompiler.ContextCompileRequest{
		RequestID:                      kernel.ids.NextID("context-request"),
		BundleID:                       kernel.ids.NextID("context-bundle"),
		WorkspaceID:                    request.WorkspaceID,
		CaseID:                         caseID,
		SnapshotID:                     stringInput(request.Input, "snapshot_id"),
		RestoreSeedID:                  stringInput(request.Input, "restore_seed_id"),
		UserMessage:                    stringInput(request.Input, "user_message"),
		CurrentTaskSummary:             stringInput(request.Input, "current_task_summary"),
		ActiveConstraints:              stringSliceInputDefault(request.Input, "active_constraints"),
		SourceObjectRefs:               stringSliceInputDefault(request.Input, "source_object_refs"),
		SourceRefs:                     stringSliceInputDefault(request.Input, "source_refs"),
		AdmittedExhibitRefs:            stringSliceInputDefault(request.Input, "admitted_exhibit_refs"),
		RejectedExhibitRefs:            stringSliceInputDefault(request.Input, "rejected_exhibit_refs"),
		RulingRefs:                     stringSliceInputDefault(request.Input, "ruling_refs"),
		ContradictionRefs:              stringSliceInputDefault(request.Input, "contradiction_refs"),
		SupersessionRefs:               stringSliceInputDefault(request.Input, "supersession_refs"),
		PalaceRouteRefs:                stringSliceInputDefault(request.Input, "palace_route_refs"),
		SemanticOperationRefs:          stringSliceInputDefault(request.Input, "semantic_operation_refs"),
		DerivedObjectRefs:              stringSliceInputDefault(request.Input, "derived_object_refs"),
		IncludeRejectedEvidenceSummary: boolInput(request.Input, "include_rejected_evidence_summary"),
		IncludeContradictions:          boolInput(request.Input, "include_contradictions"),
		IncludeRestoreSeed:             boolInput(request.Input, "include_restore_seed"),
		LayoutVersion:                  stringInput(request.Input, "layout_version"),
		PolicyVersion:                  stringInput(request.Input, "policy_version"),
		SyscallSchemaVersion:           stringInput(request.Input, "syscall_schema_version"),
		TokenBudget:                    intInput(request.Input, "token_budget"),
		CreatedBy:                      request.ActorID,
		CreatedAt:                      kernel.clock.Now(),
		Metadata:                       mapInput(request.Input, "metadata"),
	}
}

func commitContextCompileResult(kernel *Kernel, request SyscallRequest, capabilityRefs []string, eventType string, result contextcompiler.ContextCompileResult) SyscallResult {
	event, err := kernel.appendContextEvent(request, eventType, result.Bundle.BundleID, result.CaseID, capabilityRefs, result, result.Bundle.SourceRefs)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	result.Bundle.JournalRefs = append(result.Bundle.JournalRefs, event.EventID)
	for i := range result.Bundle.Blocks {
		result.Bundle.Blocks[i].JournalRefs = append(result.Bundle.Blocks[i].JournalRefs, event.EventID)
	}
	result.Bundle = contextcompiler.FinalizeBundle(result.Bundle)
	result.Blocks = result.Bundle.Blocks
	result.ProvenanceRefs = result.Bundle.SourceRefs
	kernel.context.StoreBundle(result.Bundle)
	kernel.objects.putObject(contextBundleObject(result.Bundle, request.ActorID, capabilityRefs))
	for _, block := range result.Bundle.Blocks {
		kernel.objects.putObject(contextBlockObject(block, request.ActorID, capabilityRefs))
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: result.Bundle.BundleID, JournalEvent: event.EventID, Output: result}
}

func (k *Kernel) appendContextEvent(request SyscallRequest, eventType, objectID, caseID string, capabilityRefs []string, output any, objectRefs []string) (JournalEvent, error) {
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

func contextBlockObject(block contextcompiler.ContextBlock, ownerID string, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       block.BlockID,
		ObjectType:     ObjectTypeContextBlock,
		WorkspaceID:    block.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityCompiled,
		State: map[string]any{
			"block_type":                string(block.BlockType),
			"case_id":                   block.CaseID,
			"snapshot_id":               block.SnapshotID,
			"restore_seed_id":           block.RestoreSeedID,
			"content_hash":              block.ContentHash,
			"token_input_hash":          block.TokenInputHash,
			"cache_eligibility":         string(block.CacheEligibility),
			"layout_position":           block.LayoutPosition,
			"is_canonical_truth":        false,
			"is_model_response":         false,
			"is_deterministic_kv_cache": false,
		},
		SourceRefs:      block.AllRefs(),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       block.CreatedAt,
		UpdatedAt:       block.CreatedAt,
		JournalRefs:     append([]string(nil), block.JournalRefs...),
	}
}

func contextBundleObject(bundle contextcompiler.ContextBundle, ownerID string, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       bundle.BundleID,
		ObjectType:     ObjectTypeContextBundle,
		WorkspaceID:    bundle.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityCompiled,
		State: map[string]any{
			"case_id":                   bundle.CaseID,
			"snapshot_id":               bundle.SnapshotID,
			"restore_seed_id":           bundle.RestoreSeedID,
			"layout_id":                 bundle.LayoutID,
			"layout_version":            bundle.LayoutVersion,
			"bundle_hash":               bundle.BundleHash,
			"token_input_hash":          bundle.TokenInputHash,
			"stable_prefix_hash":        bundle.StablePrefixHash,
			"volatile_suffix_hash":      bundle.VolatileSuffixHash,
			"is_canonical_truth":        false,
			"is_model_response":         false,
			"is_deterministic_kv_cache": false,
		},
		SourceRefs:      append([]string(nil), bundle.SourceRefs...),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       bundle.CreatedAt,
		UpdatedAt:       bundle.CreatedAt,
		JournalRefs:     append([]string(nil), bundle.JournalRefs...),
	}
}

func (k *Kernel) canReadContext(request SyscallRequest) bool {
	allowed, _ := k.capabilities.CanCall(request.ActorID, request.WorkspaceID, request.Name, false, k.clock.Now())
	if allowed {
		return true
	}
	allowed, _ = k.capabilities.CanCall(request.ActorID, request.WorkspaceID, SyscallContextRead, false, k.clock.Now())
	return allowed
}

func intInput(input map[string]any, key string) int {
	switch value := input[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func mapContextError(err error) error {
	switch {
	case errors.Is(err, contextcompiler.ErrContextBundleNotFound),
		errors.Is(err, contextcompiler.ErrContextBlockNotFound),
		errors.Is(err, snapshots.ErrSnapshotNotFound),
		errors.Is(err, snapshots.ErrRestoreSeedNotFound):
		return ErrObjectNotFound
	case errors.Is(err, contextcompiler.ErrWorkspaceMismatch):
		return ErrObjectNotFound
	case errors.Is(err, contextcompiler.ErrInvalidContextBlock),
		errors.Is(err, contextcompiler.ErrInvalidBlockType),
		errors.Is(err, contextcompiler.ErrInvalidCompileRequest),
		errors.Is(err, contextcompiler.ErrInvalidPromptLayout):
		return ErrInvalidInput
	default:
		return err
	}
}
