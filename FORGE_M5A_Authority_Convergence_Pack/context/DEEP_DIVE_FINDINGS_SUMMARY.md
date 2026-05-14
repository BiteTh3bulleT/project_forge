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
