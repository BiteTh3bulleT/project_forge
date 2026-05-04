package forgek

import (
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/forgek/contextcompiler"
	"forge/projectforge/services/core/internal/forgek/kv"
	forgekRuntime "forge/projectforge/services/core/internal/forgek/runtime"
)

func grantRuntimeCapability(t *testing.T, kernel *Kernel, actorID string, workspaceID string, mutationScope string, syscalls ...string) {
	t.Helper()
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-runtime-" + actorID + "-" + workspaceID,
		SubjectID:       actorID,
		AllowedSyscalls: syscalls,
		WorkspaceScope:  []string{workspaceID},
		MutationScope:   mutationScope,
		AuditRequired:   true,
	}); err != nil {
		t.Fatalf("grant runtime capability: %v", err)
	}
}

func runtimeRegisterInput(driverID string) map[string]any {
	return map[string]any{
		"driver_id":       driverID,
		"driver_name":     "Mock Runtime Driver",
		"driver_kind":     string(forgekRuntime.DriverKindMock),
		"version":         "v1",
		"runtime_backend": "mock",
		"runtime_version": "v1",
		"supported_models": []string{
			"model-a",
		},
		"supports_prefix_cache": true,
		"output_text":           "deterministic runtime proposal",
	}
}

func registerRuntimeDriver(t *testing.T, kernel *Kernel, actorID string, driverID string) forgekRuntime.RuntimeDriverManifest {
	t.Helper()
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallRuntimeRegisterDriver,
		ActorID:     actorID,
		WorkspaceID: "workspace-a",
		Input:       runtimeRegisterInput(driverID),
	})
	if !result.Success {
		t.Fatalf("runtime.register_driver failed: %v", result.Error)
	}
	return result.Output.(forgekRuntime.RuntimeDriverManifest)
}

func runtimeInputForBundle(driverID string, bundle contextcompiler.ContextBundle) map[string]any {
	return map[string]any{
		"driver_id":           driverID,
		"bundle_id":           bundle.BundleID,
		"model_id":            "model-a",
		"model_revision":      "rev-a",
		"tokenizer_id":        "tokenizer-a",
		"tokenizer_revision":  "tok-rev-a",
		"chat_template_hash":  "template-hash-a",
		"prompt_layout_hash":  kv.SHA256Text(bundle.LayoutID + "|" + bundle.LayoutVersion),
		"policy_schema_hash":  "policy-hash-a",
		"syscall_schema_hash": "syscall-hash-a",
		"token_input_hash":    bundle.TokenInputHash,
		"context_block_refs":  []string{bundle.Blocks[0].BlockID},
		"max_output_tokens":   128,
		"temperature":         0,
		"structured_output_schema": map[string]any{
			"type": "object",
		},
	}
}

func TestRuntimeSyscallsAreRegistered(t *testing.T) {
	kernel := testKernel()
	for _, name := range []string{
		SyscallRuntimeRegisterDriver,
		SyscallRuntimeListDrivers,
		SyscallRuntimeGetDriver,
		SyscallRuntimeCapabilities,
		SyscallRuntimeGenerate,
		SyscallRuntimeHealth,
	} {
		if _, ok := kernel.Syscalls().Lookup(name); !ok {
			t.Fatalf("expected runtime syscall %s to be registered", name)
		}
	}
	if kernel.Runtime() == nil {
		t.Fatal("kernel does not own runtime service")
	}
}

func TestRuntimeRegisterDriverRequiresCapabilityAndRejectsLiveDrivers(t *testing.T) {
	kernel := testKernel()
	denied := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallRuntimeRegisterDriver,
		ActorID:     "denied",
		WorkspaceID: "workspace-a",
		Input:       runtimeRegisterInput("driver-a"),
	})
	if denied.Success || !errors.Is(denied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected register denial, got %#v", denied)
	}

	grantRuntimeCapability(t, kernel, "reader", "workspace-a", MutationScopeNone, SyscallRuntimeRead)
	readOnlyDenied := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallRuntimeRegisterDriver,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       runtimeRegisterInput("driver-a"),
	})
	if readOnlyDenied.Success || !errors.Is(readOnlyDenied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected read-only register denial, got %#v", readOnlyDenied)
	}

	grantRuntimeCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallRuntimeRegisterDriver)
	liveInput := runtimeRegisterInput("live-driver")
	liveInput["driver_kind"] = string(forgekRuntime.DriverKindRemoteAPI)
	live := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeRegisterDriver, ActorID: "operator", WorkspaceID: "workspace-a", Input: liveInput})
	if live.Success || !errors.Is(live.Error, ErrInvalidInput) {
		t.Fatalf("expected live driver rejection, got %#v", live)
	}

	manifest := registerRuntimeDriver(t, kernel, "operator", "driver-a")
	obj, ok := kernel.Objects().GetObject(manifest.DriverID)
	if !ok || obj.ObjectType != ObjectTypeRuntimeDriver || obj.AuthorityLevel != AuthorityDriver || obj.WorkspaceID != "workspace-a" {
		t.Fatalf("driver object boundary not recorded: %#v ok=%v", obj, ok)
	}
	if obj.State["calls_live_runtime"] != false || obj.State["can_admit_evidence"] != false || obj.State["can_mutate_kernel"] != false {
		t.Fatalf("driver object claimed forbidden authority: %#v", obj.State)
	}
	duplicate := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeRegisterDriver, ActorID: "operator", WorkspaceID: "workspace-a", Input: runtimeRegisterInput("driver-a")})
	if duplicate.Success || !errors.Is(duplicate.Error, ErrInvalidInput) {
		t.Fatalf("expected duplicate rejection, got %#v", duplicate)
	}
	if last := kernel.Journal().ListEvents()[len(kernel.Journal().ListEvents())-1]; last.EventType != JournalEventRuntimeDriverRegistered {
		t.Fatalf("runtime.register_driver was not journaled: %#v", last)
	}
}

func TestRuntimeReadSyscallsRequireReadCapabilityAndDoNotJournal(t *testing.T) {
	kernel := testKernel()
	grantRuntimeCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallRuntimeRegisterDriver)
	manifest := registerRuntimeDriver(t, kernel, "operator", "driver-a")
	before := len(kernel.Journal().ListEvents())

	withoutRead := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeGetDriver, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"driver_id": manifest.DriverID}})
	if withoutRead.Success || !errors.Is(withoutRead.Error, ErrCapabilityDenied) {
		t.Fatalf("expected read denial, got %#v", withoutRead)
	}
	grantRuntimeCapability(t, kernel, "reader", "workspace-a", MutationScopeNone, SyscallRuntimeRead)
	getDriver := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeGetDriver, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"driver_id": manifest.DriverID}})
	listDrivers := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeListDrivers, ActorID: "reader", WorkspaceID: "workspace-a"})
	capabilities := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeCapabilities, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"driver_id": manifest.DriverID, "model_id": "model-a"}})
	health := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeHealth, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"driver_id": manifest.DriverID}})
	if !getDriver.Success || !listDrivers.Success || !capabilities.Success || !health.Success {
		t.Fatalf("runtime read syscalls failed: get=%#v list=%#v capabilities=%#v health=%#v", getDriver, listDrivers, capabilities, health)
	}
	if capabilities.Output.(forgekRuntime.RuntimeCapabilityManifest).SupportsPrefixCache != true {
		t.Fatalf("capability manifest lost driver capabilities: %#v", capabilities.Output)
	}
	if len(kernel.Journal().ListEvents()) != before {
		t.Fatal("read-only runtime syscalls journaled or mutated state")
	}
}

func TestRuntimeGenerateUsesContextRefsAndJournalsProposalOnlyResult(t *testing.T) {
	kernel := testKernel()
	compiled := compileBundleForKV(t, kernel, "compiler")
	beforeBundle, ok := kernel.ContextCompiler().GetBundle(compiled.Bundle.BundleID)
	if !ok {
		t.Fatal("compiled bundle missing")
	}
	grantRuntimeCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallRuntimeRegisterDriver, SyscallRuntimeGenerate)
	registerRuntimeDriver(t, kernel, "operator", "driver-a")

	denied := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeGenerate, ActorID: "denied", WorkspaceID: "workspace-a", Input: runtimeInputForBundle("driver-a", compiled.Bundle)})
	if denied.Success || !errors.Is(denied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected generate denial, got %#v", denied)
	}
	grantRuntimeCapability(t, kernel, "reader", "workspace-a", MutationScopeNone, SyscallRuntimeRead)
	readOnlyDenied := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeGenerate, ActorID: "reader", WorkspaceID: "workspace-a", Input: runtimeInputForBundle("driver-a", compiled.Bundle)})
	if readOnlyDenied.Success || !errors.Is(readOnlyDenied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected read-only generate denial, got %#v", readOnlyDenied)
	}
	grantRuntimeCapability(t, kernel, "scoped", "workspace-b", MutationScopeCanonical, SyscallRuntimeGenerate)
	wrongScope := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeGenerate, ActorID: "scoped", WorkspaceID: "workspace-a", Input: runtimeInputForBundle("driver-a", compiled.Bundle)})
	if wrongScope.Success || !errors.Is(wrongScope.Error, ErrCapabilityDenied) {
		t.Fatalf("expected workspace capability denial, got %#v", wrongScope)
	}
	result := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeGenerate, ActorID: "operator", WorkspaceID: "workspace-a", Input: runtimeInputForBundle("driver-a", compiled.Bundle)})
	if !result.Success || result.JournalEvent == "" {
		t.Fatalf("runtime.generate failed or did not journal: %#v", result)
	}
	generated := result.Output.(forgekRuntime.RuntimeGenerateResult)
	if generated.IsTruth() || generated.IsEvidenceAdmitted() || !generated.IsModelDriverProposal || generated.AuthorityLevel != forgekRuntime.RuntimeAuthorityProposalOnly {
		t.Fatalf("runtime result crossed authority boundary: %#v", generated)
	}
	if generated.BundleID != compiled.Bundle.BundleID || generated.OutputText == "" {
		t.Fatalf("runtime result did not cite context bundle: %#v", generated)
	}
	obj, ok := kernel.Objects().GetObject(generated.ResultID)
	if !ok || obj.ObjectType != ObjectTypeRuntimeResult || obj.AuthorityLevel != AuthorityProposal {
		t.Fatalf("runtime result object not recorded as proposal: %#v ok=%v", obj, ok)
	}
	if obj.State["is_canonical_truth"] != false || obj.State["is_admitted_evidence"] != false || obj.State["calls_live_runtime"] != false {
		t.Fatalf("runtime result claimed forbidden authority: %#v", obj.State)
	}
	afterBundle, ok := kernel.ContextCompiler().GetBundle(compiled.Bundle.BundleID)
	if !ok || afterBundle.BundleHash != beforeBundle.BundleHash {
		t.Fatalf("runtime generation mutated context bundle: before=%#v after=%#v ok=%v", beforeBundle, afterBundle, ok)
	}
	events := kernel.Journal().ListEvents()
	if !hasJournalEvent(events, JournalEventRuntimeGenerationRequested) || !hasJournalEvent(events, JournalEventRuntimeGenerationCompleted) {
		t.Fatalf("runtime generation events missing: %#v", events)
	}
}

func TestRuntimeGeneratePreservesKVMetadataWithoutMutatingKV(t *testing.T) {
	kernel := testKernel()
	compiled := compileBundleForKV(t, kernel, "compiler")
	grantKVCapability(t, kernel, "kv-operator", "workspace-a", MutationScopeCanonical, SyscallKVRegister)
	manifest := registerKVManifest(t, kernel, "kv-operator", compiled.Bundle)
	beforeManifest, ok := kernel.KV().GetManifest(manifest.CacheID)
	if !ok {
		t.Fatal("kv manifest missing before runtime generate")
	}

	grantRuntimeCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallRuntimeRegisterDriver, SyscallRuntimeGenerate)
	registerRuntimeDriver(t, kernel, "operator", "driver-a")
	input := runtimeInputForBundle("driver-a", compiled.Bundle)
	input["kv_cache_id"] = manifest.CacheID
	input["kv_lookup_id"] = "lookup-a"
	result := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeGenerate, ActorID: "operator", WorkspaceID: "workspace-a", Input: input})
	if !result.Success {
		t.Fatalf("runtime.generate with kv metadata failed: %#v", result)
	}
	generated := result.Output.(forgekRuntime.RuntimeGenerateResult)
	if generated.KVCacheID != manifest.CacheID || generated.KVLookupID != "lookup-a" {
		t.Fatalf("runtime result did not preserve kv refs: %#v", generated)
	}
	afterManifest, ok := kernel.KV().GetManifest(manifest.CacheID)
	if !ok || afterManifest.Status != beforeManifest.Status || afterManifest.ReuseCount != beforeManifest.ReuseCount || len(afterManifest.JournalRefs) != len(beforeManifest.JournalRefs) {
		t.Fatalf("runtime generation mutated kv manifest: before=%#v after=%#v ok=%v", beforeManifest, afterManifest, ok)
	}
}

func TestRuntimeGenerateDoesNotMutateCaseOrAdmitEvidence(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	open := kernel.DispatchSyscall(SyscallRequest{Name: SyscallCaseOpen, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"user_intent": "runtime boundary", "summary": "runtime boundary"}})
	if !open.Success {
		t.Fatalf("case.open failed: %#v", open)
	}
	beforeCase := open.Output.(CasePacket)
	compiled := compileBundleForKV(t, kernel, "compiler")
	grantRuntimeCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallRuntimeRegisterDriver, SyscallRuntimeGenerate)
	registerRuntimeDriver(t, kernel, "operator", "driver-a")
	result := kernel.DispatchSyscall(SyscallRequest{Name: SyscallRuntimeGenerate, ActorID: "operator", WorkspaceID: "workspace-a", CaseID: beforeCase.CaseID, Input: runtimeInputForBundle("driver-a", compiled.Bundle)})
	if !result.Success {
		t.Fatalf("runtime.generate failed: %#v", result)
	}
	afterCase, ok := kernel.objects.getCase(beforeCase.CaseID)
	if !ok || afterCase.Status != beforeCase.Status ||
		len(afterCase.SubmittedExhibitRefs) != len(beforeCase.SubmittedExhibitRefs) ||
		len(afterCase.AdmittedExhibitRefs) != len(beforeCase.AdmittedExhibitRefs) ||
		len(afterCase.RejectedExhibitRefs) != len(beforeCase.RejectedExhibitRefs) {
		t.Fatalf("runtime generation mutated case packet: before=%#v after=%#v ok=%v", beforeCase, afterCase, ok)
	}
}
