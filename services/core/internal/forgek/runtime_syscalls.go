package forgek

import (
	"context"
	"errors"

	"forge/projectforge/services/core/internal/forgek/contextcompiler"
	"forge/projectforge/services/core/internal/forgek/kv"
	forgekRuntime "forge/projectforge/services/core/internal/forgek/runtime"
)

func (k *Kernel) registerRuntimeSyscalls(register func(SyscallDefinition)) {
	for _, definition := range []SyscallDefinition{
		{Name: SyscallRuntimeRegisterDriver, Handler: handleRuntimeRegisterDriver},
		{Name: SyscallRuntimeGenerate, Handler: handleRuntimeGenerate},
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
		{Name: SyscallRuntimeListDrivers, Handler: handleRuntimeListDrivers},
		{Name: SyscallRuntimeGetDriver, Handler: handleRuntimeGetDriver},
		{Name: SyscallRuntimeCapabilities, Handler: handleRuntimeCapabilities},
		{Name: SyscallRuntimeHealth, Handler: handleRuntimeHealth},
	} {
		definition.Version = "v1"
		definition.AllowedLanes = []string{"arterial"}
		definition.Deterministic = true
		definition.Replayable = true
		register(definition)
	}
}

func handleRuntimeRegisterDriver(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	manifest := runtimeManifestFromSyscall(kernel, request)
	if manifest.DriverKind != "" && manifest.DriverKind != forgekRuntime.DriverKindMock {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	driver, err := forgekRuntime.NewMockRuntimeDriver(forgekRuntime.MockRuntimeDriverOptions{
		Manifest:   manifest,
		OutputText: stringInput(request.Input, "output_text"),
		OutputJSON: mapInput(request.Input, "output_json"),
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapRuntimeError(err)}
	}
	registered, err := kernel.runtime.RegisterDriver(driver)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapRuntimeError(err)}
	}
	event, err := kernel.appendRuntimeEvent(request, JournalEventRuntimeDriverRegistered, registered.DriverID, "", capabilityRefs, registered, []string{registered.DriverID})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	kernel.objects.putObject(runtimeDriverObject(registered, request.WorkspaceID, request.ActorID, capabilityRefs, event.EventID))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: registered.DriverID, JournalEvent: event.EventID, Output: registered}
}

func handleRuntimeListDrivers(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadRuntime(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.runtime.ListDrivers()}
}

func handleRuntimeGetDriver(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadRuntime(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	driverID := stringInput(request.Input, "driver_id")
	manifest, ok := kernel.runtime.GetDriverManifest(driverID)
	if !ok {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: driverID, Output: manifest}
}

func handleRuntimeCapabilities(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadRuntime(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	capability, err := kernel.runtime.Capabilities(context.Background(), stringInput(request.Input, "driver_id"), stringInput(request.Input, "model_id"))
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapRuntimeError(err)}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: capability.RuntimeID, Output: capability}
}

func handleRuntimeHealth(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadRuntime(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	health, err := kernel.runtime.Health(context.Background(), stringInput(request.Input, "driver_id"))
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapRuntimeError(err)}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: health.DriverID, Output: health}
}

func handleRuntimeGenerate(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	generateRequest, refs, err := runtimeGenerateRequestFromSyscall(kernel, request)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	requested, err := kernel.appendRuntimeEvent(request, JournalEventRuntimeGenerationRequested, generateRequest.RequestID, generateRequest.CaseID, capabilityRefs, generateRequest, refs)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	result, err := kernel.runtime.Generate(context.Background(), generateRequest)
	if err != nil {
		failed, appendErr := kernel.appendRuntimeEvent(request, JournalEventRuntimeGenerationFailed, generateRequest.RequestID, generateRequest.CaseID, capabilityRefs, err.Error(), refs)
		if appendErr != nil {
			return SyscallResult{Success: false, SyscallName: request.Name, Error: appendErr}
		}
		return SyscallResult{Success: false, SyscallName: request.Name, JournalEvent: failed.EventID, Error: mapRuntimeError(err)}
	}
	completed, err := kernel.appendRuntimeEvent(request, JournalEventRuntimeGenerationCompleted, result.ResultID, result.CaseID, capabilityRefs, result, result.ProvenanceRefs)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	result.JournalRefs = append(result.JournalRefs, requested.EventID, completed.EventID)
	kernel.runtime.StoreResult(result)
	kernel.objects.putObject(runtimeResultObject(result, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: result.ResultID, JournalEvent: completed.EventID, Output: result}
}

func runtimeManifestFromSyscall(kernel *Kernel, request SyscallRequest) forgekRuntime.RuntimeDriverManifest {
	return forgekRuntime.RuntimeDriverManifest{
		DriverID:                   firstNonEmpty(stringInput(request.Input, "driver_id"), kernel.ids.NextID("runtime-driver")),
		DriverName:                 firstNonEmpty(stringInput(request.Input, "driver_name"), "Mock Runtime Driver"),
		DriverKind:                 forgekRuntime.DriverKind(firstNonEmpty(stringInput(request.Input, "driver_kind"), string(forgekRuntime.DriverKindMock))),
		Version:                    firstNonEmpty(stringInput(request.Input, "version"), "v1"),
		RuntimeBackend:             firstNonEmpty(stringInput(request.Input, "runtime_backend"), "mock"),
		RuntimeVersion:             firstNonEmpty(stringInput(request.Input, "runtime_version"), "v1"),
		SupportedModels:            stringSliceInputDefault(request.Input, "supported_models"),
		SupportedCapabilities:      stringSliceInputDefault(request.Input, "supported_capabilities"),
		SupportsStreaming:          boolInput(request.Input, "supports_streaming"),
		SupportsToolCalling:        boolInput(request.Input, "supports_tool_calling"),
		SupportsStructuredOutputs:  boolInput(request.Input, "supports_structured_outputs"),
		SupportsPrefixCache:        boolInput(request.Input, "supports_prefix_cache"),
		SupportsPagedKV:            boolInput(request.Input, "supports_paged_kv"),
		SupportsKVQuantization:     boolInput(request.Input, "supports_kv_quantization"),
		SupportsKVOffload:          boolInput(request.Input, "supports_kv_offload"),
		SupportsPriorityEviction:   boolInput(request.Input, "supports_priority_eviction"),
		SupportsCacheSalt:          boolInput(request.Input, "supports_cache_salt"),
		SupportsNonPrefixReuse:     boolInput(request.Input, "supports_non_prefix_reuse"),
		SupportsCrossInstanceReuse: boolInput(request.Input, "supports_cross_instance_reuse"),
		DeterministicForTests:      true,
		AuthorityLevel:             forgekRuntime.RuntimeAuthorityProposalOnly,
		CreatedAt:                  kernel.clock.Now(),
		Metadata:                   mapInput(request.Input, "metadata"),
	}
}

func runtimeGenerateRequestFromSyscall(kernel *Kernel, request SyscallRequest) (forgekRuntime.RuntimeGenerateRequest, []string, error) {
	bundleID := stringInput(request.Input, "bundle_id")
	var bundle contextcompiler.ContextBundle
	refs := make([]string, 0)
	if bundleID != "" {
		found, ok := kernel.context.GetBundle(bundleID)
		if !ok || found.WorkspaceID != request.WorkspaceID {
			return forgekRuntime.RuntimeGenerateRequest{}, nil, ErrObjectNotFound
		}
		bundle = found
		refs = append(refs, bundle.BundleID)
		refs = append(refs, bundle.SourceRefs...)
	}
	kvCacheID := stringInput(request.Input, "kv_cache_id")
	if kvCacheID != "" {
		manifest, ok := kernel.kv.GetManifest(kvCacheID)
		if !ok || manifest.WorkspaceID != request.WorkspaceID {
			return forgekRuntime.RuntimeGenerateRequest{}, nil, ErrObjectNotFound
		}
		refs = append(refs, manifest.CacheID)
	}
	contextBlockRefs := stringSliceInputDefault(request.Input, "context_block_refs")
	for _, blockID := range contextBlockRefs {
		block, ok := kernel.context.GetBlock(blockID)
		if !ok || block.WorkspaceID != request.WorkspaceID {
			return forgekRuntime.RuntimeGenerateRequest{}, nil, ErrObjectNotFound
		}
		refs = append(refs, blockID)
	}
	generateRequest := forgekRuntime.RuntimeGenerateRequest{
		RequestID:              firstNonEmpty(stringInput(request.Input, "request_id"), kernel.ids.NextID("runtime-request")),
		DriverID:               stringInput(request.Input, "driver_id"),
		WorkspaceID:            request.WorkspaceID,
		CaseID:                 firstNonEmpty(stringInput(request.Input, "case_id"), request.CaseID, bundle.CaseID),
		BundleID:               firstNonEmpty(bundleID, bundle.BundleID),
		ContextBlockRefs:       contextBlockRefs,
		CanonicalPromptText:    firstNonEmpty(stringInput(request.Input, "canonical_prompt_text"), bundle.CanonicalPromptText),
		ModelID:                stringInput(request.Input, "model_id"),
		ModelRevision:          stringInput(request.Input, "model_revision"),
		TokenizerID:            stringInput(request.Input, "tokenizer_id"),
		TokenizerRevision:      stringInput(request.Input, "tokenizer_revision"),
		ChatTemplateHash:       stringInput(request.Input, "chat_template_hash"),
		PromptLayoutHash:       firstNonEmpty(stringInput(request.Input, "prompt_layout_hash"), kv.SHA256Text(bundle.LayoutID+"|"+bundle.LayoutVersion)),
		PolicySchemaHash:       stringInput(request.Input, "policy_schema_hash"),
		SyscallSchemaHash:      stringInput(request.Input, "syscall_schema_hash"),
		TokenInputHash:         firstNonEmpty(stringInput(request.Input, "token_input_hash"), bundle.TokenInputHash),
		KVLookupID:             stringInput(request.Input, "kv_lookup_id"),
		KVCacheID:              kvCacheID,
		MaxOutputTokens:        intInput(request.Input, "max_output_tokens"),
		Temperature:            floatInput(request.Input, "temperature"),
		StructuredOutputSchema: mapInput(request.Input, "structured_output_schema"),
		RequestedBy:            request.ActorID,
		CreatedAt:              kernel.clock.Now(),
		Metadata:               mapInput(request.Input, "metadata"),
	}
	if err := forgekRuntime.ValidateGenerateRequest(generateRequest); err != nil {
		return forgekRuntime.RuntimeGenerateRequest{}, nil, mapRuntimeError(err)
	}
	refs = append(refs, generateRequest.RequestID, generateRequest.BundleID, generateRequest.KVLookupID, generateRequest.KVCacheID)
	return generateRequest, snapshotsNormalizeRefs(refs), nil
}

func (k *Kernel) appendRuntimeEvent(request SyscallRequest, eventType, objectID, caseID string, capabilityRefs []string, output any, objectRefs []string) (JournalEvent, error) {
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

func runtimeDriverObject(manifest forgekRuntime.RuntimeDriverManifest, workspaceID, ownerID string, capabilityRefs []string, journalRef string) KernelObject {
	return KernelObject{
		ObjectID:       manifest.DriverID,
		ObjectType:     ObjectTypeRuntimeDriver,
		WorkspaceID:    workspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityDriver,
		State: map[string]any{
			"driver_kind":             string(manifest.DriverKind),
			"runtime_backend":         manifest.RuntimeBackend,
			"runtime_version":         manifest.RuntimeVersion,
			"supported_models":        append([]string(nil), manifest.SupportedModels...),
			"deterministic_for_tests": manifest.DeterministicForTests,
			"authority_level":         manifest.AuthorityLevel,
			"grants_truth_authority":  false,
			"can_mutate_kernel":       false,
			"can_write_journal":       false,
			"can_admit_evidence":      false,
			"can_mutate_kv":           false,
			"calls_live_runtime":      false,
		},
		SourceRefs:      []string{manifest.DriverID},
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       manifest.CreatedAt,
		UpdatedAt:       manifest.CreatedAt,
		JournalRefs:     []string{journalRef},
	}
}

func runtimeResultObject(result forgekRuntime.RuntimeGenerateResult, ownerID string, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       result.ResultID,
		ObjectType:     ObjectTypeRuntimeResult,
		WorkspaceID:    result.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityProposal,
		State: map[string]any{
			"request_id":                result.RequestID,
			"driver_id":                 result.DriverID,
			"bundle_id":                 result.BundleID,
			"kv_lookup_id":              result.KVLookupID,
			"kv_cache_id":               result.KVCacheID,
			"model_id":                  result.ModelID,
			"finish_reason":             string(result.FinishReason),
			"is_canonical_truth":        false,
			"is_admitted_evidence":      false,
			"mutates_case_packet":       false,
			"mutates_semantic_object":   false,
			"registers_kv_cache":        false,
			"performs_live_cache_reuse": false,
			"calls_live_runtime":        false,
		},
		SourceRefs:      append([]string(nil), result.ProvenanceRefs...),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       result.CreatedAt,
		UpdatedAt:       result.CreatedAt,
		JournalRefs:     append([]string(nil), result.JournalRefs...),
	}
}

func (k *Kernel) canReadRuntime(request SyscallRequest) bool {
	allowed, _ := k.capabilities.CanCall(request.ActorID, request.WorkspaceID, request.Name, false, k.clock.Now())
	if allowed {
		return true
	}
	allowed, _ = k.capabilities.CanCall(request.ActorID, request.WorkspaceID, SyscallRuntimeRead, false, k.clock.Now())
	return allowed
}

func floatInput(input map[string]any, key string) float64 {
	switch value := input[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}

func mapRuntimeError(err error) error {
	switch {
	case errors.Is(err, forgekRuntime.ErrDriverNotFound):
		return ErrObjectNotFound
	case errors.Is(err, forgekRuntime.ErrInvalidDriverManifest),
		errors.Is(err, forgekRuntime.ErrInvalidDriverKind),
		errors.Is(err, forgekRuntime.ErrDriverAlreadyRegistered),
		errors.Is(err, forgekRuntime.ErrInvalidCapabilityManifest),
		errors.Is(err, forgekRuntime.ErrInvalidGenerateRequest),
		errors.Is(err, forgekRuntime.ErrSecretInManifest):
		return ErrInvalidInput
	default:
		return err
	}
}
