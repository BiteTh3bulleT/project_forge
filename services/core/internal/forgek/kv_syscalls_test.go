package forgek

import (
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/forgek/contextcompiler"
	"forge/projectforge/services/core/internal/forgek/kv"
)

func grantKVCapability(t *testing.T, kernel *Kernel, actorID string, workspaceID string, mutationScope string, syscalls ...string) {
	t.Helper()
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-kv-" + actorID + "-" + workspaceID,
		SubjectID:       actorID,
		AllowedSyscalls: syscalls,
		WorkspaceScope:  []string{workspaceID},
		MutationScope:   mutationScope,
		AuditRequired:   true,
	}); err != nil {
		t.Fatalf("grant kv capability: %v", err)
	}
}

func compileBundleForKV(t *testing.T, kernel *Kernel, actorID string) contextcompiler.ContextCompileResult {
	t.Helper()
	grantContextCapability(t, kernel, actorID, "workspace-a", MutationScopeCanonical, SyscallContextCompile)
	return compileContext(t, kernel, actorID, map[string]any{
		"case_id":                "case-a",
		"source_object_refs":     []string{"case-a"},
		"admitted_exhibit_refs":  []string{"exhibit-a"},
		"current_task_summary":   "compile bundle for kv",
		"layout_version":         "phase-8-test",
		"policy_version":         "policy-v1",
		"syscall_schema_version": "syscall-v1",
	})
}

func kvInputForBundle(bundle contextcompiler.ContextBundle) map[string]any {
	return map[string]any{
		"bundle_id":            bundle.BundleID,
		"bundle_hash":          bundle.BundleHash,
		"stable_prefix_hash":   bundle.StablePrefixHash,
		"volatile_suffix_hash": bundle.VolatileSuffixHash,
		"model_id":             "model-a",
		"model_revision":       "rev-a",
		"tokenizer_id":         "tokenizer-a",
		"tokenizer_revision":   "tok-rev-a",
		"chat_template_hash":   "template-hash-a",
		"prompt_layout_hash":   kv.SHA256Text(bundle.LayoutID + "|" + bundle.LayoutVersion),
		"policy_schema_hash":   "policy-hash-a",
		"syscall_schema_hash":  "syscall-hash-a",
		"token_input_hash":     bundle.TokenInputHash,
		"runtime_backend":      "simulator",
		"runtime_version":      "v1",
		"attention_backend":    "attention-a",
		"rope_config_hash":     "rope-hash-a",
		"kv_precision":         "fp16",
		"cache_salt":           "salt-a",
	}
}

func registerKVManifest(t *testing.T, kernel *Kernel, actorID string, bundle contextcompiler.ContextBundle) kv.KVCacheManifest {
	t.Helper()
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallKVRegister,
		ActorID:     actorID,
		WorkspaceID: "workspace-a",
		Input:       kvInputForBundle(bundle),
	})
	if !result.Success {
		t.Fatalf("kv.register failed: %v", result.Error)
	}
	return result.Output.(kv.KVCacheManifest)
}

func TestKVSyscallsAreRegistered(t *testing.T) {
	kernel := testKernel()
	for _, name := range []string{
		SyscallKVRegister,
		SyscallKVLookup,
		SyscallKVRecordHit,
		SyscallKVRecordMiss,
		SyscallKVInvalidate,
		SyscallKVEvict,
		SyscallKVPromote,
		SyscallKVDemote,
		SyscallKVGetManifest,
		SyscallKVListManifests,
		SyscallKVValidateIdentity,
	} {
		if _, ok := kernel.Syscalls().Lookup(name); !ok {
			t.Fatalf("expected kv syscall %s to be registered", name)
		}
	}
	if kernel.KV() == nil {
		t.Fatal("kernel does not own kv service")
	}
}

func TestKVRegisterRequiresCapabilityAndJournalsAccelerationMetadata(t *testing.T) {
	kernel := testKernel()
	compiled := compileBundleForKV(t, kernel, "compiler")
	denied := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallKVRegister,
		ActorID:     "denied",
		WorkspaceID: "workspace-a",
		Input:       kvInputForBundle(compiled.Bundle),
	})
	if denied.Success || !errors.Is(denied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected register denial, got %#v", denied)
	}
	grantKVCapability(t, kernel, "reader", "workspace-a", MutationScopeNone, SyscallKVRead)
	readOnlyDenied := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallKVRegister,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       kvInputForBundle(compiled.Bundle),
	})
	if readOnlyDenied.Success || !errors.Is(readOnlyDenied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected read-only register denial, got %#v", readOnlyDenied)
	}
	grantKVCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallKVRegister)
	manifest := registerKVManifest(t, kernel, "operator", compiled.Bundle)
	obj, ok := kernel.Objects().GetObject(manifest.CacheID)
	if !ok || obj.ObjectType != ObjectTypeKVCacheManifest || obj.AuthorityLevel != AuthorityAcceleration {
		t.Fatalf("manifest object not registered as acceleration metadata: %#v", obj)
	}
	if obj.State["is_canonical_truth"] != false || obj.State["is_semantic_evidence"] != false || obj.State["is_memory"] != false || obj.State["stores_real_kv_tensors"] != false {
		t.Fatalf("manifest claimed forbidden authority: %#v", obj.State)
	}
	if last := kernel.Journal().ListEvents()[len(kernel.Journal().ListEvents())-1]; last.EventType != JournalEventKVCacheRegistered {
		t.Fatalf("kv.register was not journaled: %#v", last)
	}
}

func TestKVReadSyscallsRequireReadCapabilityAndDoNotJournal(t *testing.T) {
	kernel := testKernel()
	compiled := compileBundleForKV(t, kernel, "compiler")
	grantKVCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallKVRegister)
	manifest := registerKVManifest(t, kernel, "operator", compiled.Bundle)
	before := len(kernel.Journal().ListEvents())

	withoutRead := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallKVGetManifest,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"cache_id": manifest.CacheID},
	})
	if withoutRead.Success || !errors.Is(withoutRead.Error, ErrCapabilityDenied) {
		t.Fatalf("expected read denial, got %#v", withoutRead)
	}
	grantKVCapability(t, kernel, "reader", "workspace-a", MutationScopeNone, SyscallKVRead)
	getManifest := kernel.DispatchSyscall(SyscallRequest{Name: SyscallKVGetManifest, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"cache_id": manifest.CacheID}})
	list := kernel.DispatchSyscall(SyscallRequest{Name: SyscallKVListManifests, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"bundle_id": compiled.Bundle.BundleID}})
	lookup := kernel.DispatchSyscall(SyscallRequest{Name: SyscallKVLookup, ActorID: "reader", WorkspaceID: "workspace-a", Input: kvInputForBundle(compiled.Bundle)})
	validate := kernel.DispatchSyscall(SyscallRequest{Name: SyscallKVValidateIdentity, ActorID: "reader", WorkspaceID: "workspace-a", Input: withCacheID(kvInputForBundle(compiled.Bundle), manifest.CacheID)})
	if !getManifest.Success || !list.Success || !lookup.Success || !validate.Success {
		t.Fatalf("read syscalls failed: get=%#v list=%#v lookup=%#v validate=%#v", getManifest, list, lookup, validate)
	}
	if !lookup.Output.(kv.KVLookupResult).Hit || !validate.Output.(kv.ValidationResult).Passed {
		t.Fatalf("lookup/validate did not pass gates: lookup=%#v validate=%#v", lookup.Output, validate.Output)
	}
	if len(kernel.Journal().ListEvents()) != before {
		t.Fatal("read-only kv syscalls journaled or mutated state")
	}
}

func TestKVMutatingSyscallsRequireCapabilityAndJournal(t *testing.T) {
	kernel := testKernel()
	compiled := compileBundleForKV(t, kernel, "compiler")
	grantKVCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallKVRegister)
	manifest := registerKVManifest(t, kernel, "operator", compiled.Bundle)

	readOnlyInvalidate := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallKVInvalidate,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"cache_id": manifest.CacheID, "reason": "policy changed"},
	})
	if readOnlyInvalidate.Success || !errors.Is(readOnlyInvalidate.Error, ErrCapabilityDenied) {
		t.Fatalf("expected mutate denial before capability, got %#v", readOnlyInvalidate)
	}
	grantKVCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical,
		SyscallKVRegister,
		SyscallKVRecordHit,
		SyscallKVRecordMiss,
		SyscallKVInvalidate,
		SyscallKVEvict,
		SyscallKVPromote,
		SyscallKVDemote,
	)
	for _, call := range []SyscallRequest{
		{Name: SyscallKVPromote, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"cache_id": manifest.CacheID}},
		{Name: SyscallKVDemote, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"cache_id": manifest.CacheID}},
		{Name: SyscallKVRecordHit, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"cache_id": manifest.CacheID}},
		{Name: SyscallKVRecordMiss, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"cache_id": manifest.CacheID, "miss_reason": "identity_gates_failed", "failed_gates": []string{kv.GateTokenIdentity}}},
		{Name: SyscallKVInvalidate, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"cache_id": manifest.CacheID, "reason": "policy changed"}},
	} {
		result := kernel.DispatchSyscall(call)
		if !result.Success || result.JournalEvent == "" {
			t.Fatalf("%s failed or did not journal: %#v", call.Name, result)
		}
	}
	second := registerKVManifest(t, kernel, "operator", compiled.Bundle)
	evict := kernel.DispatchSyscall(SyscallRequest{Name: SyscallKVEvict, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"cache_id": second.CacheID, "reason": "capacity"}})
	if !evict.Success || evict.Output.(kv.KVCacheManifest).Status != kv.StatusEvicted {
		t.Fatalf("evict failed: %#v", evict)
	}
	events := kernel.Journal().ListEvents()
	for _, eventType := range []string{
		JournalEventKVCacheHit,
		JournalEventKVCacheMiss,
		JournalEventKVCacheInvalidated,
		JournalEventKVCacheEvicted,
		JournalEventKVCachePromoted,
		JournalEventKVCacheDemoted,
	} {
		if !hasJournalEvent(events, eventType) {
			t.Fatalf("missing journal event %s in %#v", eventType, events)
		}
	}
}

func TestKVWorkspaceScopeAndAccelerationBoundary(t *testing.T) {
	kernel := testKernel()
	compiled := compileBundleForKV(t, kernel, "compiler")
	grantKVCapability(t, kernel, "scoped", "workspace-b", MutationScopeCanonical, SyscallKVRegister)
	wrongScope := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallKVRegister,
		ActorID:     "scoped",
		WorkspaceID: "workspace-a",
		Input:       kvInputForBundle(compiled.Bundle),
	})
	if wrongScope.Success || !errors.Is(wrongScope.Error, ErrCapabilityDenied) {
		t.Fatalf("expected workspace capability denial, got %#v", wrongScope)
	}

	grantKVCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallKVRegister, SyscallKVEvict)
	manifest := registerKVManifest(t, kernel, "operator", compiled.Bundle)
	beforeBundle, ok := kernel.ContextCompiler().GetBundle(compiled.Bundle.BundleID)
	if !ok {
		t.Fatal("compiled bundle missing")
	}
	evict := kernel.DispatchSyscall(SyscallRequest{Name: SyscallKVEvict, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"cache_id": manifest.CacheID}})
	if !evict.Success {
		t.Fatalf("evict failed: %#v", evict)
	}
	afterBundle, ok := kernel.ContextCompiler().GetBundle(compiled.Bundle.BundleID)
	if !ok || afterBundle.BundleHash != beforeBundle.BundleHash {
		t.Fatalf("kv mutation changed context bundle: before=%#v after=%#v ok=%v", beforeBundle, afterBundle, ok)
	}
	obj, ok := kernel.Objects().GetObject(manifest.CacheID)
	if !ok || obj.State["performs_live_cache_reuse"] != false || obj.State["requires_context_compiler"] != true {
		t.Fatalf("manifest object violated simulator boundary: %#v ok=%v", obj, ok)
	}
}

func withCacheID(input map[string]any, cacheID string) map[string]any {
	out := make(map[string]any, len(input)+1)
	for key, value := range input {
		out[key] = value
	}
	out["cache_id"] = cacheID
	return out
}

func hasJournalEvent(events []JournalEvent, eventType string) bool {
	for _, event := range events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}
