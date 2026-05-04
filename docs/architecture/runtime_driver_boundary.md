# Runtime Driver Boundary

Status: Phase 9 implemented in the FORGE-K simulator. Scope is `SIMULATOR_ONLY / DRIVER_BOUNDARY_ONLY`.

The Runtime Driver Boundary defines how FORGE-K treats model runtimes as governed drivers. It does not wire FORGE-K into the live daemon, replace live `modelruntime`, add routes, change gateway behavior, call real model backends, or perform live KV cache reuse.

## Doctrine

Models are drivers, not authority. A runtime driver may generate proposal output, but it cannot mutate canonical truth, admit evidence, write snapshots, write ContextBundles, register KV manifests, execute tools, decide capability authority, or bypass semantic syscalls.

Runtime output remains proposal evidence until an existing FORGE-K validation, admission, and Kernel commit path accepts it. Manifest and capability declarations describe what a driver can do; they do not grant authority.

## Boundary Components

Phase 9 implements the simulator boundary for these concepts:

- `RuntimeDriver`: bounded interface for manifest inspection, capability reporting, and generation.
- `RuntimeDriverManifest`: serializable driver identity and declared backend capability metadata.
- `RuntimeCapabilityManifest`: per-runtime/model/tokenizer/template capability and limit metadata.
- `RuntimeGenerateRequest`: deterministic request envelope with workspace scope, prompt text or ContextBundle refs, policy/schema refs, and optional KV lookup metadata.
- `RuntimeGenerateResult`: proposal output plus provenance, driver/model identity, context refs, KV metadata refs, warnings, and non-authoritative status.
- `RuntimeDriverRegistry`: simulator registry for selecting an explicitly registered driver by id.
- `RuntimeService` or `RuntimeBroker`: simulator service that validates request shape and invokes a registered driver without granting authority.
- `MockRuntimeDriver`: deterministic test driver used to prove proposal-only behavior.

Only the mock driver is active in this phase. Driver kinds such as local development, vLLM, SGLang, TensorRT-LLM, Ollama, llama.cpp, and remote API backends are declarations for later phases, not active integrations. Phase 9 syscall registration rejects non-mock driver registration.

## Simulator Syscalls

The FORGE-K simulator registers:

- `runtime.register_driver`
- `runtime.list_drivers`
- `runtime.get_driver`
- `runtime.capabilities`
- `runtime.generate`
- `runtime.health`

`runtime.register_driver` and `runtime.generate` require runtime mutation capability. Read syscalls require the specific read syscall capability or `runtime.read`. Runtime registration journals `RUNTIME_DRIVER_REGISTERED`; generation journals `RUNTIME_GENERATION_REQUESTED` and `RUNTIME_GENERATION_COMPLETED`, or `RUNTIME_GENERATION_FAILED` when the mock driver returns an error.

## Implemented Tests

Phase 9 tests cover manifest validation, secret metadata rejection, deterministic mock generation, registry duplicate rejection, service result storage, syscall registration, capability denial for register/generate, read-only capability behavior, non-mock registration rejection, context bundle refs by reference only, KV metadata preservation without KV mutation, journal events, and proposal-only model-as-driver output.

## Context And KV Rules

Runtime requests may carry ContextBundle, ContextBlock, snapshot, restore-seed, and case refs. The driver may receive canonical prompt text prepared by the Context Compiler, but it must not compile context, fetch live external content, admit evidence, or mutate source objects.

Runtime requests may also carry KV lookup metadata or KVCacheManifest refs. In Phase 9 those refs are metadata only. The boundary must not store real KV tensors, perform backend prefix reuse, perform non-prefix reuse, register runtime cache artifacts, or treat KV cache as memory.

## Live Boundary

Phase 9 has no live daemon authority. It does not change:

- live AI-OS controllane behavior
- live gateway policy, route, or approval behavior
- live `modelruntime` management or inference paths
- public APIs
- live `COMPILE_CONTEXT`
- live snapshot, context, or KV behavior

Any real model backend integration, live KV reuse, streaming runtime output, or live daemon route change requires a later explicitly scoped `LIVE_INTEGRATION` phase with design, tests, and documentation updates.
