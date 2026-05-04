package forgek

import (
	"errors"
	"time"

	"forge/projectforge/services/core/internal/forgek/contextcompiler"
	"forge/projectforge/services/core/internal/forgek/kv"
)

func (k *Kernel) registerKVSyscalls(register func(SyscallDefinition)) {
	for _, definition := range []SyscallDefinition{
		{Name: SyscallKVRegister, Handler: handleKVRegister},
		{Name: SyscallKVRecordHit, Handler: handleKVRecordHit},
		{Name: SyscallKVRecordMiss, Handler: handleKVRecordMiss},
		{Name: SyscallKVInvalidate, Handler: handleKVInvalidate},
		{Name: SyscallKVEvict, Handler: handleKVEvict},
		{Name: SyscallKVPromote, Handler: handleKVPromote},
		{Name: SyscallKVDemote, Handler: handleKVDemote},
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
		{Name: SyscallKVLookup, Handler: handleKVLookup},
		{Name: SyscallKVGetManifest, Handler: handleKVGetManifest},
		{Name: SyscallKVListManifests, Handler: handleKVListManifests},
		{Name: SyscallKVValidateIdentity, Handler: handleKVValidateIdentity},
	} {
		definition.Version = "v1"
		definition.AllowedLanes = []string{"arterial"}
		definition.Deterministic = true
		definition.Replayable = true
		register(definition)
	}
}

func handleKVRegister(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	bundle, ok := kvBundleFromRequest(kernel, request)
	if !ok {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	input := kvManifestInputFromSyscall(kernel, request, bundle)
	manifest, err := kv.NewManifest(input)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapKVError(err)}
	}
	event, err := kernel.appendKVEvent(request, JournalEventKVCacheRegistered, manifest.CacheID, manifest.CaseID, capabilityRefs, manifest, manifest.AllRefs())
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	manifest.JournalRefs = append(manifest.JournalRefs, event.EventID)
	kernel.kv.StoreManifest(manifest)
	kernel.objects.putObject(kvManifestObject(manifest, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: manifest.CacheID, JournalEvent: event.EventID, Output: manifest}
}

func handleKVLookup(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadKV(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	if _, ok := kvBundleFromRequest(kernel, request); !ok {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	lookup := kvLookupRequestFromSyscall(kernel, request)
	result, err := kernel.kv.Lookup(lookup, kernel.ids.NextID("kv-validation"), kernel.clock.Now())
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapKVError(err)}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: result.CacheID, Output: result}
}

func handleKVRecordHit(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	cacheID := stringInput(request.Input, "cache_id")
	manifest, ok := kernel.kv.GetManifest(cacheID)
	if !ok || manifest.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if !kv.HitEligibleStatus(manifest.Status) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidStateTransition}
	}
	event, err := kernel.appendKVEvent(request, JournalEventKVCacheHit, cacheID, manifest.CaseID, capabilityRefs, manifest, manifest.AllRefs())
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	updated, err := kernel.kv.RecordHit(cacheID, kernel.clock.Now(), event.EventID)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapKVError(err)}
	}
	kernel.objects.putObject(kvManifestObject(updated, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: cacheID, JournalEvent: event.EventID, Output: updated}
}

func handleKVRecordMiss(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	result := kv.KVLookupResult{
		Hit:         false,
		CacheID:     stringInput(request.Input, "cache_id"),
		MissReason:  stringInput(request.Input, "miss_reason"),
		FailedGates: stringSliceInputDefault(request.Input, "failed_gates"),
		CreatedAt:   kernel.clock.Now(),
		Metadata:    mapInput(request.Input, "metadata"),
	}
	if result.MissReason == "" {
		result.MissReason = kv.MissReasonIdentityGatesFailed
	}
	event, err := kernel.appendKVEvent(request, JournalEventKVCacheMiss, result.CacheID, request.CaseID, capabilityRefs, result, result.FailedGates)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	kernel.kv.RecordMiss(result)
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: result.CacheID, JournalEvent: event.EventID, Output: result}
}

func handleKVInvalidate(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return handleKVStatusMutation(kernel, request, capabilityRefs, JournalEventKVCacheInvalidated, func(cacheID, reason, eventID string) (kv.KVCacheManifest, error) {
		return kernel.kv.Invalidate(cacheID, reason, kernel.clock.Now(), eventID)
	})
}

func handleKVEvict(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return handleKVStatusMutation(kernel, request, capabilityRefs, JournalEventKVCacheEvicted, func(cacheID, reason, eventID string) (kv.KVCacheManifest, error) {
		return kernel.kv.Evict(cacheID, reason, kernel.clock.Now(), eventID)
	})
}

func handleKVPromote(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return handleKVTierMutation(kernel, request, capabilityRefs, JournalEventKVCachePromoted, kernel.kv.Promote)
}

func handleKVDemote(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return handleKVTierMutation(kernel, request, capabilityRefs, JournalEventKVCacheDemoted, kernel.kv.Demote)
}

func handleKVGetManifest(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadKV(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	cacheID := stringInput(request.Input, "cache_id")
	manifest, ok := kernel.kv.GetManifest(cacheID)
	if !ok || manifest.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: cacheID, Output: manifest}
}

func handleKVListManifests(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadKV(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	filter := kv.ManifestListFilter{
		WorkspaceID: request.WorkspaceID,
		CaseID:      stringInput(request.Input, "case_id"),
		BundleID:    stringInput(request.Input, "bundle_id"),
		BlockID:     stringInput(request.Input, "block_id"),
		CacheMode:   kv.CacheMode(stringInput(request.Input, "cache_mode")),
		Status:      kv.ManifestStatus(stringInput(request.Input, "status")),
		MemoryTier:  kv.MemoryTier(stringInput(request.Input, "memory_tier")),
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.kv.ListManifests(filter)}
}

func handleKVValidateIdentity(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	if !kernel.canReadKV(request) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
	}
	cacheID := stringInput(request.Input, "cache_id")
	lookup := kvLookupRequestFromSyscall(kernel, request)
	result, err := kernel.kv.ValidateIdentity(cacheID, kernel.ids.NextID("kv-validation"), lookup, kernel.clock.Now())
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapKVError(err)}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: cacheID, Output: result}
}

func handleKVStatusMutation(kernel *Kernel, request SyscallRequest, capabilityRefs []string, eventType string, mutate func(cacheID, reason, eventID string) (kv.KVCacheManifest, error)) SyscallResult {
	cacheID := stringInput(request.Input, "cache_id")
	manifest, ok := kernel.kv.GetManifest(cacheID)
	if !ok || manifest.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	reason := stringInput(request.Input, "reason")
	if reason == "" {
		reason = "manual"
	}
	event, err := kernel.appendKVEvent(request, eventType, cacheID, manifest.CaseID, capabilityRefs, manifest, manifest.AllRefs())
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	updated, err := mutate(cacheID, reason, event.EventID)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapKVError(err)}
	}
	kernel.objects.putObject(kvManifestObject(updated, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: cacheID, JournalEvent: event.EventID, Output: updated}
}

func handleKVTierMutation(kernel *Kernel, request SyscallRequest, capabilityRefs []string, eventType string, mutate func(cacheID, journalRef string) (kv.KVCacheManifest, error)) SyscallResult {
	cacheID := stringInput(request.Input, "cache_id")
	manifest, ok := kernel.kv.GetManifest(cacheID)
	if !ok || manifest.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	event, err := kernel.appendKVEvent(request, eventType, cacheID, manifest.CaseID, capabilityRefs, manifest, manifest.AllRefs())
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	updated, err := mutate(cacheID, event.EventID)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapKVError(err)}
	}
	kernel.objects.putObject(kvManifestObject(updated, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: cacheID, JournalEvent: event.EventID, Output: updated}
}

func kvBundleFromRequest(kernel *Kernel, request SyscallRequest) (contextcompiler.ContextBundle, bool) {
	bundleID := stringInput(request.Input, "bundle_id")
	bundle, ok := kernel.context.GetBundle(bundleID)
	if !ok || bundle.WorkspaceID != request.WorkspaceID {
		return contextcompiler.ContextBundle{}, false
	}
	if blockID := stringInput(request.Input, "block_id"); blockID != "" {
		block, ok := kernel.context.GetBlock(blockID)
		if !ok || block.WorkspaceID != request.WorkspaceID || !bundleContainsBlock(bundle, blockID) {
			return contextcompiler.ContextBundle{}, false
		}
	}
	return bundle, true
}

func bundleContainsBlock(bundle contextcompiler.ContextBundle, blockID string) bool {
	for _, block := range bundle.Blocks {
		if block.BlockID == blockID {
			return true
		}
	}
	return false
}

func kvManifestInputFromSyscall(kernel *Kernel, request SyscallRequest, bundle contextcompiler.ContextBundle) kv.ManifestInput {
	return kv.ManifestInput{
		CacheID:            firstNonEmpty(stringInput(request.Input, "cache_id"), kernel.ids.NextID("kv-cache")),
		CacheMode:          kv.CacheMode(stringInput(request.Input, "cache_mode")),
		WorkspaceID:        request.WorkspaceID,
		CaseID:             firstNonEmpty(stringInput(request.Input, "case_id"), request.CaseID, bundle.CaseID),
		BundleID:           bundle.BundleID,
		BlockID:            stringInput(request.Input, "block_id"),
		SnapshotID:         firstNonEmpty(stringInput(request.Input, "snapshot_id"), bundle.SnapshotID),
		RestoreSeedID:      firstNonEmpty(stringInput(request.Input, "restore_seed_id"), bundle.RestoreSeedID),
		BundleHash:         firstNonEmpty(stringInput(request.Input, "bundle_hash"), bundle.BundleHash),
		StablePrefixHash:   firstNonEmpty(stringInput(request.Input, "stable_prefix_hash"), bundle.StablePrefixHash),
		VolatileSuffixHash: firstNonEmpty(stringInput(request.Input, "volatile_suffix_hash"), bundle.VolatileSuffixHash),
		ModelID:            stringInput(request.Input, "model_id"),
		ModelRevision:      stringInput(request.Input, "model_revision"),
		TokenizerID:        stringInput(request.Input, "tokenizer_id"),
		TokenizerRevision:  stringInput(request.Input, "tokenizer_revision"),
		ChatTemplateHash:   stringInput(request.Input, "chat_template_hash"),
		PromptLayoutHash:   firstNonEmpty(stringInput(request.Input, "prompt_layout_hash"), kv.SHA256Text(bundle.LayoutID+"|"+bundle.LayoutVersion)),
		PolicySchemaHash:   stringInput(request.Input, "policy_schema_hash"),
		SyscallSchemaHash:  stringInput(request.Input, "syscall_schema_hash"),
		TokenInputHash:     firstNonEmpty(stringInput(request.Input, "token_input_hash"), bundle.TokenInputHash),
		FinalTokenIDsHash:  stringInput(request.Input, "final_token_ids_hash"),
		RuntimeBackend:     stringInput(request.Input, "runtime_backend"),
		RuntimeVersion:     stringInput(request.Input, "runtime_version"),
		AttentionBackend:   stringInput(request.Input, "attention_backend"),
		RopeConfigHash:     stringInput(request.Input, "rope_config_hash"),
		KVPrecision:        stringInput(request.Input, "kv_precision"),
		MemoryTier:         kv.MemoryTier(stringInput(request.Input, "memory_tier")),
		CacheSalt:          stringInput(request.Input, "cache_salt"),
		Status:             kv.ManifestStatus(stringInput(request.Input, "status")),
		CreatedAt:          kernel.clock.Now(),
		Metadata:           mapInput(request.Input, "metadata"),
	}
}

func kvLookupRequestFromSyscall(kernel *Kernel, request SyscallRequest) kv.KVLookupRequest {
	bundle, _ := kvBundleFromRequest(kernel, request)
	return kv.KVLookupRequest{
		RequestID:          firstNonEmpty(stringInput(request.Input, "request_id"), kernel.ids.NextID("kv-lookup")),
		WorkspaceID:        request.WorkspaceID,
		CaseID:             firstNonEmpty(stringInput(request.Input, "case_id"), request.CaseID, bundle.CaseID),
		BundleID:           firstNonEmpty(stringInput(request.Input, "bundle_id"), bundle.BundleID),
		BlockID:            stringInput(request.Input, "block_id"),
		SnapshotID:         firstNonEmpty(stringInput(request.Input, "snapshot_id"), bundle.SnapshotID),
		RestoreSeedID:      firstNonEmpty(stringInput(request.Input, "restore_seed_id"), bundle.RestoreSeedID),
		BundleHash:         firstNonEmpty(stringInput(request.Input, "bundle_hash"), bundle.BundleHash),
		StablePrefixHash:   firstNonEmpty(stringInput(request.Input, "stable_prefix_hash"), bundle.StablePrefixHash),
		VolatileSuffixHash: firstNonEmpty(stringInput(request.Input, "volatile_suffix_hash"), bundle.VolatileSuffixHash),
		ModelID:            stringInput(request.Input, "model_id"),
		ModelRevision:      stringInput(request.Input, "model_revision"),
		TokenizerID:        stringInput(request.Input, "tokenizer_id"),
		TokenizerRevision:  stringInput(request.Input, "tokenizer_revision"),
		ChatTemplateHash:   stringInput(request.Input, "chat_template_hash"),
		PromptLayoutHash:   firstNonEmpty(stringInput(request.Input, "prompt_layout_hash"), kv.SHA256Text(bundle.LayoutID+"|"+bundle.LayoutVersion)),
		PolicySchemaHash:   stringInput(request.Input, "policy_schema_hash"),
		SyscallSchemaHash:  stringInput(request.Input, "syscall_schema_hash"),
		TokenInputHash:     firstNonEmpty(stringInput(request.Input, "token_input_hash"), bundle.TokenInputHash),
		FinalTokenIDsHash:  stringInput(request.Input, "final_token_ids_hash"),
		RuntimeBackend:     stringInput(request.Input, "runtime_backend"),
		RuntimeVersion:     stringInput(request.Input, "runtime_version"),
		AttentionBackend:   stringInput(request.Input, "attention_backend"),
		RopeConfigHash:     stringInput(request.Input, "rope_config_hash"),
		KVPrecision:        stringInput(request.Input, "kv_precision"),
		CacheSalt:          stringInput(request.Input, "cache_salt"),
		CacheMode:          kv.CacheMode(stringInput(request.Input, "cache_mode")),
		CreatedAt:          kernel.clock.Now(),
	}
}

func (k *Kernel) appendKVEvent(request SyscallRequest, eventType, objectID, caseID string, capabilityRefs []string, output any, objectRefs []string) (JournalEvent, error) {
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

func kvManifestObject(manifest kv.KVCacheManifest, ownerID string, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       manifest.CacheID,
		ObjectType:     ObjectTypeKVCacheManifest,
		WorkspaceID:    manifest.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityAcceleration,
		State: map[string]any{
			"cache_mode":                string(manifest.CacheMode),
			"case_id":                   manifest.CaseID,
			"bundle_id":                 manifest.BundleID,
			"block_id":                  manifest.BlockID,
			"snapshot_id":               manifest.SnapshotID,
			"restore_seed_id":           manifest.RestoreSeedID,
			"status":                    string(manifest.Status),
			"memory_tier":               string(manifest.MemoryTier),
			"reuse_count":               manifest.ReuseCount,
			"token_input_hash":          manifest.TokenInputHash,
			"final_token_ids_hash":      manifest.FinalTokenIDsHash,
			"is_canonical_truth":        false,
			"is_semantic_evidence":      false,
			"is_memory":                 false,
			"stores_real_kv_tensors":    false,
			"calls_model_runtime":       false,
			"performs_live_cache_reuse": false,
			"requires_context_compiler": true,
			"simulator_metadata_only":   true,
		},
		SourceRefs:      manifest.AllRefs(),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       manifest.CreatedAt,
		UpdatedAt:       firstManifestUpdateTime(manifest),
		JournalRefs:     append([]string(nil), manifest.JournalRefs...),
	}
}

func firstManifestUpdateTime(manifest kv.KVCacheManifest) time.Time {
	if manifest.InvalidatedAt != nil {
		return *manifest.InvalidatedAt
	}
	if manifest.LastUsedAt != nil {
		return *manifest.LastUsedAt
	}
	return manifest.CreatedAt
}

func (k *Kernel) canReadKV(request SyscallRequest) bool {
	allowed, _ := k.capabilities.CanCall(request.ActorID, request.WorkspaceID, request.Name, false, k.clock.Now())
	if allowed {
		return true
	}
	allowed, _ = k.capabilities.CanCall(request.ActorID, request.WorkspaceID, SyscallKVRead, false, k.clock.Now())
	return allowed
}

func mapKVError(err error) error {
	switch {
	case errors.Is(err, kv.ErrManifestNotFound):
		return ErrObjectNotFound
	case errors.Is(err, kv.ErrWorkspaceMismatch):
		return ErrObjectNotFound
	case errors.Is(err, kv.ErrInvalidManifest),
		errors.Is(err, kv.ErrInvalidCacheMode),
		errors.Is(err, kv.ErrInvalidMemoryTier),
		errors.Is(err, kv.ErrInvalidStatus),
		errors.Is(err, kv.ErrInvalidLookupRequest):
		return ErrInvalidInput
	case errors.Is(err, kv.ErrInvalidStateTransition):
		return ErrInvalidStateTransition
	default:
		return err
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
