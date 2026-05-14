# MASTER PROMPT — FORGE M5A Authority Convergence + Latency Foundation

You are working inside the `BiteTh3bulleT/project_forge` repository.

Latest visible target HEAD while this pack was prepared: `cd1b2986a9c9d51eea9af87fd0e70789f651ee4d`.

Your mission is to implement **M5A — Authority Matrix + Runtime Policy Convergence + First Latency Foundation**.

This is a production-grade architecture hardening pass. The goal is to make FORGE’s live authority surfaces honest, testable, visible, and faster without expanding live authority.

Do not output code in chat. Make changes in files. End with changed files, tests run, pass/fail, and remaining gaps.

---

## Mandatory reading before editing

Read these first:

- `README.md`
- `AGENTS.md`
- `docs/status/current_authority_sources.md`
- `docs/reviews/current_phase_status.md`
- `docs/status/implementation_matrix.md`
- `docs/status/model_runtime_status.md`
- `docs/status/runtime_truth_vs_docs.md`
- `docs/architecture/model_runtime.md`
- `docs/architecture/forge_system_cockpit.md`
- `docs/architecture/forge_workstation_substrate.md`
- `services/core/internal/gateway/tool_capability_registry.go`
- `services/core/internal/api/model_runtime.go`
- `services/core/internal/api/model_runtime_governance.go`
- `services/core/internal/aios/controllane/processor.go`
- `services/core/internal/aios/controllane/approval.go`
- `services/core/internal/hostbridge/service.go`
- `services/core/internal/forgeh/service.go`

---

## Current doctrine to preserve

- FORGE is the AI-OS operating environment.
- Linux/NixOS remains the substrate.
- FORGE-K is target cognitive microkernel architecture, not live daemon authority in this sprint.
- Models are drivers, not authority.
- Model output is evidence, not truth.
- Tool execution goes through Gateway.
- Canonical semantic mutation goes through Control Lane.
- Approval and audit remain separate durable systems.
- HostBridge is read-only.
- FORGE-H is advisory-only.
- Qdrant is not truth.
- Redis is not canonical memory.
- KV cache is acceleration, not memory.
- System Cockpit is read-only first.
- Micro-agents may draft/propose/cache; they may not commit/approve/mutate.

---

## Mission outcomes

Implement these outcomes in order.

### Outcome 1 — Repository reality and authority audit

Create/update:

- `docs/reviews/m5a_authority_convergence_review.md`

It must record:
- current HEAD/branch,
- modelruntime/gateway drift,
- `model.delete_file` status drift,
- model chat/generate policy decision,
- Control Lane approval gate status,
- HostBridge/FORGE-H snapshot status,
- System Cockpit readiness,
- tests run,
- unresolved blockers.

### Outcome 2 — Machine-readable authority matrix

Create a machine-readable authority map that covers at least:
- `/api/gateway/invoke`
- `/forge/models*`
- `/v1/models`
- `/v1/chat/completions`
- modelruntime management and chat routes
- Control Lane validation actions
- memory write/read surfaces
- backup restore
- HostBridge diagnostics
- FORGE-H posture/proposal surfaces
- System status endpoint

Each row must define:
- route or action,
- authority owner,
- capability id,
- gateway capability status if applicable,
- mutating/read-only,
- destructive flag,
- requires approval,
- approval mechanism,
- audit category/action,
- response visibility,
- live/FORGE-K authority flags,
- notes.

Preferred: deterministic static matrix first, tests before clever dynamic generation.

### Outcome 3 — Fix modelruntime/gateway policy drift

Resolve the drift:
- `model.delete_file` is implemented and approval-required in modelruntime.
- Gateway registry must not say it is deferred/future if it is live.
- `model.chat` and `model.generate` policy must be explicit and honest.

Recommended M5A posture:
- Modelruntime owns `/forge/models*` and `/v1/*` execution governance.
- Gateway registry is policy-visible taxonomy unless a shared capability evaluator already exists cleanly.
- Authority matrix must say modelruntime owns chat/generate.
- Tests must prove no hidden Gateway requirement is implied.

### Outcome 4 — Control Lane approval fingerprint seam

Create a durable approval fingerprint model for Control Lane.

Minimum deliverable:
- document approval fingerprint fields,
- add pure helper/model if clean,
- add tests proving fingerprint shape is deterministic,
- do not break existing Control Lane behavior unless implementing full enforcement.

Fingerprint should include:
- action,
- capability,
- mutating flag,
- actor,
- source,
- workspace,
- trace/correlation when applicable,
- target object type,
- payload shape hash,
- risk class,
- approval id,
- expiry/status if verifying live approvals.

### Outcome 5 — HostBridge/FORGE-H snapshot cache

Implement first latency foundation:
- background-safe snapshot cache or TTL cache,
- cached reads are read-only/advisory-only,
- missing cache degrades honestly,
- source errors preserved,
- snapshot age/stale state visible,
- no host mutation, no semantic write, no FORGE-K authority expansion.

### Outcome 6 — System Cockpit read-only authority display

Extend existing System status/UI with:
- authority matrix summary,
- modelruntime backend profile status,
- modelruntime/gateway drift status,
- HostBridge/FORGE-H snapshot age,
- safe-mode flags,
- current blockers,
- `not wired`/`unavailable` where applicable.

No mutation buttons.

### Outcome 7 — Micro-agent acceleration plan

Create:

- `docs/architecture/micro_agent_acceleration.md`

Design safe background workers for runtime preflight, retrieval pre-rank, context precompile, approval packet drafting, artifact summarization, HostBridge/FORGE-H sampler, and model compatibility precheck.

Every worker must be proposal/cache/advisory only.

### Outcome 8 — Validation

At minimum attempt:

```bash
npm test
npm run lint
npm run validate:js
npm run validate:local
npm run build
npm run smoke
```

If broad commands fail due environment, run narrow equivalents and document why.

---

## Definition of Done

M5A is done when:

- Gateway/modelruntime drift is documented and resolved.
- `model.delete_file` metadata matches runtime reality.
- `model.chat`/`model.generate` authority posture is explicit and tested.
- Authority matrix exists and is covered by tests.
- System Cockpit can display read-only authority summary.
- HostBridge/FORGE-H status reads can use cached snapshots.
- Control Lane approval fingerprint seam exists with deterministic tests.
- Micro-agent acceleration design exists with strict no-authority rules.
- Docs point to current truth, not stale phase claims.
- Validation commands are recorded.
- No live authority is expanded accidentally.

---


# WHAT NOT TO DO

This applies to every prompt in this pack.

## Do not output code in chat

Make changes in files. Final response should summarize files changed, tests run, pass/fail, and remaining gaps. Do not paste large code blocks in chat.

## Do not expand live authority

Do not make FORGE-K live authority. Do not route live memory mutation through FORGE-K simulator services. Do not make FORGE-H mutate the host. Do not make HostBridge execute host changes. Do not make the System Cockpit mutate anything. Do not make micro-agents write canonical memory. Do not let models approve actions. Do not bypass Gateway for tool execution. Do not bypass Control Lane for canonical semantic mutation. Do not bypass modelruntime governance for inference/model lifecycle. Do not treat Qdrant as truth. Do not treat Redis as canonical memory. Do not treat KV cache as memory.

## Do not add dangerous buttons

Do not add UI buttons for restart, shutdown, reboot, `systemctl`, `nixos-rebuild`, package-manager actions, kernel/module manipulation, destructive cleanup, raw model load/unload outside governed modelruntime paths, host mutation, or shell command execution.

## Do not hide missing data

If data is missing, return/report `unavailable`, `not_wired`, `unknown`, `disabled_by_default`, `deferred`, or `design_only`. Do not display missing data as healthy.

## Do not create duplicate authority paths

Do not create a second modelruntime execution path, approval system, audit system, memory write path, tool execution path, host mutation path, or route registry. Prefer extending existing authority surfaces with narrow contracts.

## Do not perform broad unrelated refactors

This sprint is about authority convergence and latency foundation. Do not rewrite the desktop design, whole gateway, whole modelruntime, whole Control Lane, Nix architecture, or memory model. Make narrow, testable changes.


Final response must include changed files, tests run, pass/fail, unresolved blockers, and next recommended phase.


---


# FORGE Current Truth Context

## Project identity

FORGE means **Foundry for Organic Reasoning, Growth, and Execution**.

Tagline: **Turn chaos into cognition. Turn cognition into action.**

FORGE is a local-first AI workspace / AI-OS operating layer for inspectable, approval-gated engineering work.

## Current authority model

1. **Canonical mutation authority**
   - Lives in `services/core/internal/aios/controllane`.
   - Durable semantic writes must pass deterministic validation, journal/commit boundaries, audit, and provenance.
   - FORGE-K simulator services are not live daemon authority.

2. **Tool execution authority**
   - Lives in `services/core/internal/gateway`.
   - Gateway-only execution is the intended governed tool path.
   - Legacy adapter invoke ingress is not authority.

3. **Model runtime authority**
   - Lives in `services/core/internal/modelruntime` and `services/core/internal/api/model_runtime*.go`.
   - Models are governed drivers, not truth authority.
   - Streaming, vLLM-compatible external endpoint support, and managed delete-file approval exist inside modelruntime boundaries.
   - Model output is evidence, not automatic memory truth.

4. **Memory and retrieval**
   - Memory and retrieval are evidence/projection systems unless committed through canonical paths.
   - Tool/model output does not become truth by existing.

5. **Approvals and audit**
   - Approval requests and approval decisions must remain distinct.
   - Audit records are durable evidence.

6. **FORGE-K boundary**
   - FORGE-K remains target cognitive microkernel architecture and simulator-first implementation.
   - Current live seams are narrow validation/enforcement surfaces only.
   - No live state mutation should route through FORGE-K simulator services in this phase.

7. **FORGE-H and HostBridge**
   - HostBridge is read-only host diagnostics.
   - FORGE-H is advisory resource policy.
   - FORGE-H may recommend; it must not mutate host/modelruntime/memory directly.

8. **Nix / workstation**
   - NixOS/Linux remains the host substrate.
   - FORGE Workstation mutation controls are future, proposal/approval/rollback-gated.
   - No direct shell-to-host mutation.
   - No `nixos-rebuild`, `systemctl`, reboot, shutdown, package-manager, or kernel/module controls from UI in this phase.

## M5A sprint thesis

Before adding more capability, make the existing capability map honest.

M5A should produce:
- a machine-readable authority matrix,
- fixed modelruntime/gateway capability drift,
- durable approval fingerprint design and first implementation seam,
- read-only System Cockpit authority display,
- HostBridge/FORGE-H snapshot caching,
- first safe background micro-agent plan,
- validation tests that prevent authority drift from returning.


---


# Deep Dive Findings Summary

## Critical / High findings

### 1. Modelruntime vs Gateway policy drift

Gateway capability metadata says some `model.*` actions are approval-only or deferred, but modelruntime routes are authoritative under `/forge/models*` and `/v1/*`.

Specific drift:
- `model.chat` and `model.generate` are policy-visible as approval-sensitive in Gateway.
- `/forge/models/{id}/chat` and `/v1/chat/completions` route through modelruntime directly.
- `model.delete_file` is implemented in modelruntime governance but may still appear deferred in Gateway metadata.

Impact:
- UI can misrepresent what is real.
- Agents can choose the wrong path.
- Future workers may build duplicate governance lanes.
- Operator trust degrades because capability truth and route truth disagree.

Required fix:
- Create one route-capability-authority contract.
- Add tests that fail when registry metadata and route behavior drift.
- Align `model.delete_file`.
- Decide policy posture for `model.chat` and `model.generate`.

### 2. Control Lane approval gate too shallow

Control Lane has deterministic processor structure, but approval is too source-label based for future autonomy or high-risk mutation.

Required fix:
- Introduce approval fingerprinting.
- Approval must bind to action, capability, actor, source, workspace, risk, mutation type, payload shape hash, approval request id, and expiration/decision status.

### 3. Autonomy approval bridge is not durable enough

Autonomy remains mostly propose-only, but any commit-capable path must use the durable approvals service.

Required fix:
- Default autonomy approval escalator must not synthesize approval truth.
- Proposal-only mode may draft packets.
- Commit mode requires durable approval record and fingerprint match.

### 4. HostBridge/FORGE-H synchronous sampling is a latency smell

HostBridge reads proc/sys and may shell out for diagnostics. FORGE-H consumes HostBridge posture to produce advisory resource policy.

Because both are read-only/advisory-only, repeated synchronous sampling is wasteful.

Required fix:
- Background sampler or TTL cache.
- Force-refresh option gated and bounded.
- Read-only System Cockpit consumes cached posture.

### 5. Modelruntime scheduler and health paths are likely latency hotspots

The scheduler is safe but centralized. Health checks can do too much live work.

Required fix:
- Baseline measurements first.
- Split admission from per-backend queues later.
- Cache cheap readiness separately from deep backend probes.
- Add latency instrumentation.

### 6. Nix/CI validation can give false confidence

Some Nix checks can skip or be environment-blocked. CI is broad and monolithic.

Required fix:
- Make skipped checks visibly skipped.
- Split CI eventually.
- For this sprint, at least document validation truth and add local validation commands.

## Good news

FORGE already has the right doctrine:
- models propose,
- kernel commits,
- gateway executes tools,
- HostBridge observes,
- FORGE-H advises,
- FORGE-K simulator is not live authority,
- approval/audit are separate.

M5A is about making code, docs, UI, and tests all agree.
