# FORGE Phase M4-vLLM — Nix + Rust + vLLM Runtime Integration

## Mission

Implement the first governed vLLM backend path for FORGE.

FORGE’s runtime direction:

> Nix builds the machine. Rust guards the machine. vLLM feeds the machine. FORGE governs the machine.

This phase must add vLLM as a governed model-serving backend behind FORGE’s existing modelruntime boundary.

This is not a rewrite of modelruntime.
This is not a bypass around FORGE policy.
This is not raw vLLM access from chat.
This is a controlled backend integration.

## Current Assumptions

FORGE already has:

- Go core backend
- modelruntime abstraction
- OpenAI-compatible backend path
- Ollama adapter path
- Nix flake/packages/modules/checks
- Rust validator crate direction
- Tauri/operator desktop model surfaces
- gateway/tool governance
- audit/provenance doctrine
- safe-mode Nix defaults

vLLM should be added as a backend/provider candidate through the existing modelruntime architecture.

## Read First

Inspect these before editing:

- `docs/architecture/model_runtime.md`
- `docs/status/model_runtime_status.md`
- `docs/status/model_runtime_m3_baseline.md`
- `docs/architecture/runtime_driver_boundary.md`
- `services/core/internal/modelruntime/*`
- `services/core/internal/api/model_runtime.go`
- `services/core/internal/api/model_runtime_bridge.go`
- `services/core/internal/api/model_runtime_governance.go`
- `services/core/internal/config/config.go`
- `apps/desktop/src/pages/ModelsPage.tsx`
- `nix/packages/*`
- `nix/nixos/modules/*`
- `nix/checks/*`
- `flake.nix`
- `crates/forgek-validate/*`
- `docs/runbooks/current_forge_bringup.md`
- `docs/reviews/FORGE_FULL_CODE_REVIEW.md` if present

## Required Architecture

vLLM must sit behind FORGE modelruntime.

Correct flow:

```text
Operator Desktop
    ↓
FORGE Core API
    ↓
Model Runtime Governance
    ↓
Backend Selection
    ↓
vLLM OpenAI-Compatible Server
    ↓
GPU Inference

Forbidden flow:

Chat/UI → raw vLLM endpoint → ungoverned output/action

Model output remains proposal/evidence unless another governed path promotes it.

vLLM serves tokens.
FORGE governs routing, policy, audit, memory promotion, and tool boundaries.

Phase Strategy

Implement in three levels.

Level 1 — External vLLM Endpoint Support

Add support for connecting FORGE modelruntime to an already-running vLLM OpenAI-compatible endpoint.

This is the first working target.

Do not try to solve full CUDA/Nix packaging first.

Level 2 — Nix Operator Tooling

Add Nix/operator support so an operator can install/use vLLM tooling or wrapper scripts consistently.

This may be marked PARTIAL if full CUDA/vLLM packaging is not reliable yet.

Level 3 — Future Managed vLLM Service

Design but do not overbuild a future NixOS service/profile for managed vLLM.

Mark as FUTURE unless fully tested.

Implementation Tasks
1. Add Architecture Doc

Create:

docs/architecture/nix_rust_vllm_runtime.md

Include:

Nix as reproducible host/operator substrate
Rust as deterministic validator/systems sidecar layer
vLLM as high-throughput inference backend
FORGE as governance layer
why vLLM must sit behind modelruntime
why vLLM output cannot directly mutate memory/truth/tools
Level 1/2/3 integration plan
current LIVE/PARTIAL/FUTURE labels

Required wording:

Nix builds the machine. Rust guards the machine. vLLM feeds the machine. FORGE governs the machine.

Status labels:

LIVE: existing Nix substrate, existing Rust validator crate, existing modelruntime abstraction if verified
PARTIAL: vLLM external endpoint support once implemented
FUTURE: fully managed NixOS vLLM service, GPU-aware scheduling, operator desktop vLLM controls if not implemented
2. Add vLLM Backend Type

Inspect the existing modelruntime backend structure.

Add a vLLM backend type in the least disruptive way.

Preferred approach:

Reuse the OpenAI-compatible backend if it already supports arbitrary base URLs.
Add backend_vllm.go only if vLLM needs explicit identity/health/model parsing.
Do not duplicate the entire OpenAI-compatible backend unless required.

Expected backend identity:

backend kind: vllm
protocol: openai-compatible
default base URL: http://127.0.0.1:8000/v1

Required behavior:

health check
list models if endpoint supports it
chat/completions or responses path using existing modelruntime shape
timeout handling
clear unavailable/degraded errors
no silent fallback without audit/log note
3. Config Support

Add config fields if missing:

FORGE_VLLM_ENABLED
FORGE_VLLM_BASE_URL
FORGE_VLLM_API_KEY
FORGE_VLLM_DEFAULT_MODEL
FORGE_VLLM_TIMEOUT_SECONDS

Safe defaults:

FORGE_VLLM_ENABLED=false
FORGE_VLLM_BASE_URL=http://127.0.0.1:8000/v1
FORGE_VLLM_API_KEY empty
FORGE_VLLM_DEFAULT_MODEL empty
FORGE_VLLM_TIMEOUT_SECONDS sane bounded default

Rules:

vLLM disabled by default.
localhost default only.
no required internet access.
no model auto-load unless explicit.
no mutation of memory/truth/tools from vLLM output.

Update:

services/core/internal/config/config.go
docs/runbooks/config_reference.md
any config tests
4. Model Runtime Registration

Wire vLLM into modelruntime as a backend/provider.

Requirements:

appears in modelruntime status/health
can be enabled/disabled by config or runtime settings if existing pattern supports it
does not break existing Ollama/llama.cpp/OpenAI-compatible backends
exposes clear status:
disabled
unavailable
degraded
ready

If existing modelruntime store/manifest requires backend records, add a migration or default manifest record carefully.

Do not create duplicate model authority.

5. API Surface

Update existing model runtime APIs only if necessary.

Expected surfaces should continue to be under:

/forge/models/*
/forge/model-runtime/*

Add no new public route unless the current API cannot represent backend status.

If adding route, use:

/forge/model-runtime/backends/vllm/status

But prefer existing status route.

6. Operator Desktop Update

Update Models page to show vLLM backend status if the API exposes it.

The UI should show:

Enabled/disabled
Base URL, redacted if needed
Health
Available models if provided
Default model if set
Last error if unavailable
“external endpoint” label

Do not add raw base URL editing unless settings pattern already exists and is safe.

Do not add arbitrary command execution from UI.

7. Nix Operator Support

Add Nix-side support carefully.

Goal for this pass:

Ensure operator environment can have a vLLM client/check wrapper.
Do not require vLLM/CUDA package to build unless it is reliable.
Do not break non-GPU machines.

Create optional wrapper scripts if appropriate:

forge-vllm-status
forge-vllm-health
forge-vllm-models

These wrappers should use fixed commands only, such as curl/jq against the configured local endpoint.

If adding NixOS module options, use safe defaults:

forge.modelRuntime.vllm.enable = false;
forge.modelRuntime.vllm.baseUrl = "http://127.0.0.1:8000/v1";
forge.modelRuntime.vllm.managedService.enable = false;

Do not make vLLM required for nix flake check.

8. Optional Rust Sidecar Design

Do not overbuild Rust in this pass.

If useful, add a small design note for future Rust sidecar probes:

endpoint health probe
CUDA/VRAM telemetry normalizer
model manifest validator
backend readiness validator

If adding Rust code, keep it tiny and tested.

Do not add a Rust daemon unless explicitly needed.

9. Tests

Add tests for:

Config
vLLM disabled by default
base URL default is localhost
timeout bounds
API key redaction if surfaced
invalid URL rejected if validation exists
Backend
disabled backend reports disabled
unavailable endpoint reports unavailable/degraded
mocked vLLM /models response parsed
mocked chat/completion response parsed
timeout produces controlled error
existing OpenAI/Ollama/llama.cpp tests still pass
API
model runtime status includes vLLM when configured
no vLLM route exists if not configured unless existing status shows disabled
errors are bounded and do not leak secrets
UI
Models page renders vLLM disabled state
Models page renders vLLM unavailable state
Models page renders vLLM ready state with model list
API key is never displayed
Nix
flake check does not require GPU/vLLM service
wrapper scripts, if added, are present
vLLM managed service disabled by default
mutation flags remain false where applicable
10. Docs

Update or create:

docs/architecture/nix_rust_vllm_runtime.md
docs/status/model_runtime_status.md
docs/runbooks/config_reference.md
docs/runbooks/current_forge_bringup.md
docs/operations/operator_toolbelt.md if present
docs/public/FORGE_PUBLIC_TECHNICAL_BRIEF.md if present

Add a section:

## Runtime Direction: Nix + Rust + vLLM

Nix builds the machine. Rust guards the machine. vLLM feeds the machine. FORGE governs the machine.
11. Status Labels

Update capability/status docs honestly.

Use:

vLLM external endpoint backend: PARTIAL or LIVE depending on tests
vLLM managed NixOS service: FUTURE unless fully implemented/tested
GPU-aware scheduling: FUTURE
Rust vLLM sidecar: FUTURE unless implemented/tested

Do not overclaim.

Validation Commands

Run what is practical.

At minimum:

git status --short
rg -n "vllm|VLLM|modelruntime|openai-compatible|ollama" services docs nix apps crates flake.nix

Go:

cd services/core
go test ./internal/config
go test ./internal/modelruntime
go test ./internal/api -run 'Model|Runtime|VLLM|Backend'
go test ./...

Frontend:

npm run typecheck
npm test
npm run build

Nix if available:

nix flake check

If Nix daemon/GPU/CUDA/vLLM packaging is unavailable, record honestly.

What Not To Do
Do not bypass FORGE modelruntime.
Do not call vLLM directly from chat/UI.
Do not let vLLM output mutate memory/truth/tools.
Do not enable vLLM by default.
Do not require GPU to boot FORGE.
Do not require vLLM for normal tests.
Do not add arbitrary command execution to operator desktop.
Do not leak API keys in UI/logs/errors.
Do not claim managed vLLM service is live unless tested.
Do not solve CUDA/Nix packaging by breaking flake check.
Do not remove Ollama/OpenAI-compatible/llama.cpp support.
Do not introduce a second model governance path.
Do not expose private FORGE secret logic.
Do not mark FUTURE items as LIVE.
Definition of Done

This phase is complete when:

FORGE has documented Nix + Rust + vLLM runtime direction.
vLLM can be configured as a governed modelruntime backend or explicit OpenAI-compatible backend profile.
vLLM is disabled by default.
vLLM status appears through modelruntime health/status.
Tests cover disabled, unavailable, and mocked-ready vLLM states.
Operator desktop can display vLLM backend state if modelruntime exposes it.
Nix support does not require GPU/vLLM to evaluate.
Docs clearly label what is LIVE/PARTIAL/FUTURE.
Existing modelruntime, Ollama, and OpenAI-compatible behavior remains intact.
Final response summarizes files changed, tests run, and remaining gaps only.
Final Response Format

Do not dump code in chat.

Respond with:

Implemented Phase M4-vLLM.

Files changed:
- ...

Validation:
- ...

Status:
- vLLM external backend: ...
- managed vLLM NixOS service: ...
- GPU-aware scheduling: ...

Remaining gaps:
- ...
