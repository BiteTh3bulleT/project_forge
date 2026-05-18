# FORGE-K Fully Online — Phased Prompt Pack

Use this pack with Codex, Cursor, Claude Code, or another repo agent. Each phase is intentionally narrow. Do not skip phases. Do not combine authority migrations. Do not let the agent "just wire it in." That is architecture arson wearing a lab coat.

---

## Universal Header For Every Phase

Paste this at the top of every phase prompt.

```text
You are working in the FORGE repository.

ROLE:
You are a careful FORGE-K migration engineer. Your job is to move FORGE-K toward live operation without creating a second authority path, bypassing existing safety systems, or overclaiming simulator status.

CURRENT TRUTH:
- FORGE-K simulator services under services/core/internal/forgek are not live daemon authority unless a specific live integration phase proves it.
- Existing live daemon authority remains with API routes, AI-OS Control Lane, gateway, permissions, lanes, audit, modelruntime, retrieval/search/embeddings, memory, and NixOS/systemd host boundaries.
- FORGE-K should become fully online one authority seam at a time.
- NixOS is the host/declarative substrate. FORGE-K must not directly mutate host configuration or run host-control commands outside governed proposal/build/test/approval/rollback flows.
- Model output is proposal/evidence, never canonical truth.
- Gateway remains the live tool execution authority.
- The operator remains final authority for dangerous, destructive, cross-workspace, external, host, or high-risk actions.

MANDATORY BEHAVIOR:
1. Read AGENTS.md first.
2. Read docs/status/current_authority_sources.md.
3. Read docs/reviews/current_phase_status.md.
4. Read docs/architecture/forge_k_overview.md.
5. Read docs/architecture/forge_k_operational_cutover_design.md.
6. Before changing code, write a short phase plan in docs/superpowers/plans/ using the current date and phase name.
7. Keep changes small and scoped to this phase only.
8. Add or update tests for every behavior change.
9. Update status docs only when evidence exists.
10. Record validation commands and results in a phase report under docs/reviews/ or docs/status/.
11. If a command cannot run due to environment limits, record the exact command, failure, and why it is not conclusive.
12. Do not output code in chat. Make changes in files only.
13. Do not mark the phase complete until tests/docs/evidence are present.

GLOBAL VALIDATION TARGETS:
Run the narrowest relevant tests first, then attempt:
- npm test
- npm run lint
- npm run build:core
- npm run validate:desktop when desktop files changed
- npm run validate:forgek when FORGE-K/shared validation changes
- npm run validate:local when broad integration changed
- nix flake check when Nix files changed and Nix is available
- any new focused tests added in this phase

WHAT NOT TO DO:
- Do not import services/core/internal/forgek simulator services as live authority.
- Do not create a second live Kernel.
- Do not bypass gateway, permissions, lanes, approvals, audit, modelruntime, Control Lane, or NixOS/systemd boundaries.
- Do not make shadow diagnostics authoritative.
- Do not make modelruntime output truth.
- Do not write memory directly from model output.
- Do not let Qdrant/vector scores become truth.
- Do not let Redis become canonical state.
- Do not enable live KV reuse before deterministic context/runtime identity is proven.
- Do not run nixos-rebuild, systemctl mutation, host power actions, package manager mutation, destructive cleanup, or external side effects unless this phase explicitly authorizes a proposal-only path and operator approval flow.
- Do not add public routes unless the phase explicitly requires it.
- Do not claim something is live, complete, tested, or safe without evidence.
```

---

## Completion Ladder

```text
SIMULATOR_ONLY
→ SHADOW_READ_ONLY
→ VALIDATION_ONLY
→ DISABLED_BY_DEFAULT_LIVE
→ OPERATOR_APPROVED_LIVE
→ DEFAULT_LIVE
→ LEGACY_PATH_RETIRED
```

Nothing jumps stages.

---

# Phase 0 — Baseline Authority Inventory

```text
PHASE 0: FORGE-K ONLINE BASELINE AUTHORITY INVENTORY

GOAL:
Create a current, evidence-backed map of exactly what is simulator-only, shadow-read-only, partial live validation, blocked, disabled-by-default live, or truly live. This phase must not change runtime behavior.

READ FIRST:
- AGENTS.md
- README.md
- docs/status/current_authority_sources.md
- docs/reviews/current_phase_status.md
- docs/status/implementation_matrix.md
- docs/architecture/forge_k_overview.md
- docs/architecture/forge_k_operational_cutover_design.md
- docs/architecture/forge_k_live_integration_design.md
- docs/reviews/forge_k_live_path_mapping.md

TASKS:
1. Create docs/reviews/forge_k_online_baseline_inventory.md.
2. Inventory each FORGE-K component: Kernel, Neuron Fabric, Courthouse, Memory Palace, Semantic Algebra, Snapshots, Context Compiler, KV System, Runtime Boundary, Lymphatic Lane, Consensus Mesh, Shadow Harness, shared validation contracts.
3. For each component, record current status, current code path, live owner, simulator owner, live imports, existing tests, next safe promotion step, and blockers.
4. Inventory live authority owners: API, Control Lane, Gateway, Permissions, Lanes, Audit, Modelruntime, Retrieval, Search, Embeddings, Memory, Backup/release, and NixOS/systemd host layer.
5. Add a section named "No Authority Migration Performed".

FILES:
- docs/reviews/forge_k_online_baseline_inventory.md
- docs/status/current_authority_sources.md only if navigation changed
- docs/reviews/current_phase_status.md only if status corrections are evidence-backed

VALIDATION:
- npm test if possible
- npm run validate:forgek if possible
- Record docs-only status and no runtime behavior changes.

DONE WHEN:
A reviewer can tell exactly what is live, partial, shadow, simulator, blocked, or deferred.

WHAT NOT TO DO:
- Do not implement anything.
- Do not promote authority.
- Do not edit runtime code.
- Do not overclaim FORGE-K live authority.
```

---

# Phase 1 — NixOS Host Envelope

```text
PHASE 1: NIXOS HOST ENVELOPE FOR FORGE-K ONLINE PATH

GOAL:
Create or harden the NixOS/systemd envelope that will safely host FORGE-K online work. This is host substrate work, not FORGE-K authority migration.

READ FIRST:
- AGENTS.md
- docs/architecture/forge_workstation_substrate.md
- docs/architecture/nix_substrate.md
- docs/architecture/forge_os_host_substrate.md
- docs/architecture/forge_safe_mode_recovery_profiles.md
- docs/runbooks/current_forge_bringup.md
- docs/runbooks/config_reference.md

TASKS:
1. Review existing Nix files under nix/.
2. Create or update a NixOS module for forge-core if already present.
3. Ensure service defaults: loopback bind, API auth required, FORGE_DATA_DIR under /forge/data or configured equivalent, workspace default /forge/workspace not /, model home /forge/models, logs /forge/logs, backups /forge/backups.
4. Add systemd hardening where feasible: dedicated forge user, PrivateTmp, NoNewPrivileges, ProtectSystem where compatible, ReadWritePaths limited to /forge paths, restricted environment.
5. Add safe-mode profile: CPU-only, modelruntime optional/disabled by default, gateway high-risk execute disabled or approval-only, TTY/fallback recovery kept available.
6. Add or update Nix tests/checks if feasible.
7. Add docs/runbooks/forge_nixos_host_envelope.md.

FILES:
- nix/nixos/modules/forge-core.nix or existing module equivalent
- nix/nixos/modules/forge-storage.nix if storage layout exists
- nix/nixos/profiles/forge-safe-mode.nix or equivalent
- docs/runbooks/forge_nixos_host_envelope.md
- docs/reviews/phase_01_nixos_host_envelope.md

VALIDATION:
- nix flake check if available
- nix build relevant package/check if available
- npm run build:core
- npm test

DONE WHEN:
FORGE has an explicit NixOS/systemd host envelope and no FORGE-K live authority changed.

WHAT NOT TO DO:
- Do not run nixos-rebuild.
- Do not mutate the host.
- Do not enable autologin.
- Do not remove fallback desktop or TTY.
- Do not make FORGE-K live authority.
```

---

# Phase 2 — Authority Gate Matrix

```text
PHASE 2: FORGE-K AUTHORITY GATE MATRIX

GOAL:
Add a machine-readable and operator-visible authority gate matrix for FORGE-K online readiness.

TASKS:
1. Define a gate model: id, name, status, live owner, simulator owner, gate stage, feature flag, rollback method, blockers, required tests, last validation evidence.
2. Persist the gate matrix in a safe read-only store or deterministic static registry.
3. Add backend API read endpoint only if an existing authenticated/internal status surface is appropriate.
4. Add desktop read-only page/card if desktop status surfaces already exist.
5. Add tests proving valid enum values, blocked gates cannot be reported live, simulator-only services are not advertised as live, and the read-only endpoint does not mutate state.
6. Update docs/status/forge_k_authority_gates.md.

FILES:
- services/core/internal/forgekstatus or equivalent package
- services/core/internal/api/forge_k_status*.go if adding API
- apps/desktop/src/pages/... only if UI is in scope
- docs/status/forge_k_authority_gates.md
- docs/reviews/phase_02_authority_gate_matrix.md

VALIDATION:
- focused Go tests
- npm test
- npm run validate:desktop if UI changed
- npm run build:core

DONE WHEN:
Operator can see exactly which FORGE-K gates are blocked, partial, shadow, validation-only, disabled, or live.

WHAT NOT TO DO:
- Do not make gates mutable from public/normal UI.
- Do not let gate status change behavior yet.
- Do not mark blocked gates as live.
```

---

# Phase 3 — Shared Pure Contract Extraction

```text
PHASE 3: SHARED PURE CONTRACT EXTRACTION

GOAL:
Extract deterministic FORGE-K contracts into shared pure packages outside simulator service authority, so live owners can validate without importing simulator services.

CHOOSE ONE CONTRACT ONLY:
- semantic object shape validation
- CasePacket shape validation
- EvidenceRef validation
- claim shape validation
- ContextBlock shape validation
- ContextBundle shape validation
- Snapshot shape validation
- Journal event shape validation
- journal hash-chain validation
- capability predicate validation
- Consensus claim/ruling envelope validation
- Runtime proposal envelope validation

TASKS:
1. Select exactly one contract and document why.
2. Create/extend a pure package outside services/core/internal/forgek simulator service authority.
3. Keep package side-effect free: no DB, no network, no modelruntime, no gateway, no memory writes, no retrieval calls, no filesystem writes.
4. Add validators, canonical normalization, and deterministic error codes.
5. Add table-driven unit tests.
6. Add simulator parity tests where appropriate.
7. Add forbidden live import tests if relevant.
8. Add docs/architecture/forge_k_shared_contracts.md or a focused contract doc.

FILES:
- services/core/internal/<contractname>/
- services/core/internal/forgek/... parity tests if needed
- docs/architecture/forge_k_shared_contracts.md
- docs/reviews/phase_03_<contract>_extraction.md

VALIDATION:
- go test ./services/core/internal/<contractname>/...
- go test ./services/core/internal/forgek/... or repo equivalent
- npm run validate:forgek
- npm test

DONE WHEN:
One pure contract is extracted, tested, documented, and ready for live-owner validation-only use.

WHAT NOT TO DO:
- Do not integrate it into live behavior yet unless explicitly scoped.
- Do not import simulator services into live packages.
- Do not add side effects.
```

---

# Phase 4 — Live Semantic Syscall Facade

```text
PHASE 4: LIVE SEMANTIC SYSCALL FACADE

GOAL:
Normalize live semantic write intent into a syscall-shaped facade inside the existing live Control Lane without changing commit behavior yet.

TASKS:
1. Identify existing live semantic mutation paths in services/core/internal/aios/controllane.
2. Define a live semantic syscall envelope: syscall id, object type, action, actor, workspace, provenance, correlation id, trace id, capability, expected effect, idempotency key, rollback metadata.
3. Add adapter/facade that wraps current mutation requests into this envelope.
4. Keep existing live owner as commit authority.
5. Add validation-only checks using existing shared pure contracts where available.
6. Add audit fields showing syscall-shaped metadata.
7. Add tests: valid envelope, malformed envelope fail-closed, no memory write on rejected validation, idempotency preserved, no route behavior change, no simulator-service imports.
8. Add docs/architecture/live_semantic_syscall_facade.md.

VALIDATION:
- focused controllane tests
- npm test
- npm run build:core
- npm run validate:forgek if shared contracts touched

DONE WHEN:
Live semantic writes have a normalized syscall-shaped envelope and existing Control Lane remains live owner.

WHAT NOT TO DO:
- Do not replace the Control Lane.
- Do not route commits through forgek.Kernel.
- Do not migrate all object types at once.
```

---

# Phase 5 — Live Journal And Replay Foundation

```text
PHASE 5: LIVE JOURNAL AND REPLAY FOUNDATION

GOAL:
Create the journal/replay foundation required before FORGE-K Kernel commit authority can come online.

TASKS:
1. Inventory existing event, audit, job, gateway, approval, and controllane records.
2. Define canonical journal event schema with event id/type, previous hash, event hash, timestamp, workspace, actor, source, correlation id, trace id, subject, syscall id, provenance refs, and effect summary.
3. Add append-only journal writer or adapter.
4. Add replay verifier that can dry-run journal integrity.
5. Add hash-chain validation.
6. Add tests: deterministic hashing, append-only behavior, tamper detection, replay dry-run success, malformed event rejection, rollback/disable mode.
7. Add docs/architecture/forge_k_live_journal_and_replay.md.

VALIDATION:
- focused journal tests
- npm test
- npm run build:core

DONE WHEN:
FORGE has a live journal/replay foundation that can verify meaningful transitions without relying on raw chat transcripts.

WHAT NOT TO DO:
- Do not replace existing audit immediately.
- Do not delete old event paths.
- Do not make replay mutate state yet.
```

---

# Phase 6 — Courthouse Admission Online

```text
PHASE 6: COURTHOUSE ADMISSION ONLINE AS ADMISSION-ONLY

GOAL:
Bring Courthouse doctrine online as evidence admission before truth commit, without making admission equal canonical truth.

TASKS:
1. Define live CasePacket and EvidenceRef envelopes using shared pure validators.
2. Build live-owned admission service or adapter.
3. Support states: proposed, submitted, validated, admitted, rejected, superseded, contradicted.
4. Support exhibit refs: memory observation, retrieval result, artifact, audit record, gateway result, modelruntime output, user assertion.
5. Preserve provenance and source refs.
6. Add contradiction/supersession record support if not present.
7. Add tests: valid admission, rejected malformed evidence, admitted is not committed truth, contradictions are not silently merged, no memory write by admission alone, no modelruntime/tool/retrieval execution.
8. Add operator-visible admission summary if existing UI status surfaces support it.
9. Add docs/architecture/live_courthouse_admission.md.

VALIDATION:
- focused admission tests
- npm test
- npm run build:core
- npm run validate:desktop if UI changed

DONE WHEN:
Evidence can be admitted/rejected for context/commit review, and admission does not equal canonical truth.

WHAT NOT TO DO:
- Do not make simulator Courthouse live by direct import.
- Do not let model output self-admit.
- Do not write memory from admission alone.
```

---

# Phase 7 — Memory Palace Live Mirror

```text
PHASE 7: MEMORY PALACE LIVE MIRROR

GOAL:
Bring Memory Palace online as a live evidence topology/mirror before allowing it to own retrieval or memory writes.

TASKS:
1. Build or harden read-only adapters: LiveMemoryMirrorAdapter, LiveRetrievalMirrorAdapter, ReadOnlyRAGAdapter, LiveEmbeddingTraceAdapter, LiveSearchTraceAdapter.
2. Mirror metadata only: retrieval run ids, result ids, source refs, chunk refs, memory observation refs, embedding/VSA metadata refs, usefulness metadata.
3. Do not capture raw content unless an existing live-authorized source already exposes a bounded summary.
4. Do not execute retrieval/search/embedding from FORGE-K.
5. Preserve provenance.
6. Add tests: mirror existing refs, no retrieval execution, no embedding provider call, no memory write, unsafe/secret metadata rejection, bounded retention.
7. Add docs/architecture/live_memory_palace_mirror.md.

VALIDATION:
- focused adapter tests
- npm test
- npm run build:core

DONE WHEN:
Memory Palace can inspect live evidence topology without owning truth or executing retrieval.

WHAT NOT TO DO:
- Do not make Qdrant truth.
- Do not run live retrieval from FORGE-K.
- Do not write memory.
- Do not expose raw private content in diagnostics.
```

---

# Phase 8 — Context Compiler Shadow → Canary → Live

```text
PHASE 8: CONTEXT COMPILER SHADOW TO CANARY TO LIVE

GOAL:
Move Context Compiler toward live prompt/context authority safely.

EXECUTE ONLY ONE SUBSTAGE PER RUN:
A. shadow-only
B. canary disabled-by-default
C. operator-approved live for one route/thread class
D. default live after evidence

SHADOW TASKS:
1. Build live-owned Context Compiler adapter.
2. Input only admitted evidence refs and safe metadata.
3. Emit ContextBundle and PromptLayout diagnostics.
4. Compare against current live prompt construction.
5. Do not alter live prompt yet.

CANARY TASKS:
1. Add feature flag default false.
2. Allow one route/thread class to use compiled context when explicitly enabled.
3. Add fallback to legacy prompt builder.
4. Record quality/latency diagnostics.

LIVE TASKS:
1. Require prior shadow/canary evidence.
2. Promote one bounded surface only.
3. Keep rollback flag.
4. Surface included/excluded context reasons to operator.

TESTS:
- deterministic bundle hash
- admitted-only refs
- no rejected evidence included
- prompt layout stable
- fallback works
- token budget enforced
- no raw secret capture
- response behavior unchanged in shadow mode
- rollback flag restores legacy builder

VALIDATION:
- focused context compiler adapter tests
- npm test
- npm run build:core
- npm run validate:desktop if inspector UI changed

DONE WHEN:
Current substage is implemented with evidence and no broader prompt authority migration occurs.

WHAT NOT TO DO:
- Do not make simulator ContextCompilerService live by direct import.
- Do not include non-admitted evidence in live prompts.
- Do not remove legacy fallback during shadow/canary.
```

---

# Phase 9 — Runtime Proposal Boundary

```text
PHASE 9: RUNTIME PROPOSAL BOUNDARY

GOAL:
Wrap modelruntime outputs in typed FORGE-K proposal envelopes so model output cannot become implicit truth.

TASKS:
1. Define RuntimeProposal types: answer_draft, claim_proposal, action_proposal, memory_proposal, tool_proposal, contradiction_proposal, summary_proposal.
2. Attach model id, backend id, runtime config, prompt hash, context bundle hash if available, token counts, output bytes, correlation id, trace id, proposal type.
3. Ensure proposals do not mutate memory.
4. Route tool proposals through gateway only.
5. Route memory proposals to Courthouse/Kernel review only.
6. Add tests: proposal created for model output, no direct memory write, no direct tool execution, audit/provenance present, unsupported proposal type fails closed.
7. Add docs/architecture/runtime_proposal_boundary.md.

VALIDATION:
- focused modelruntime/proposal tests
- npm test
- npm run build:core

DONE WHEN:
Modelruntime output is typed proposal evidence, not implicit state.

WHAT NOT TO DO:
- Do not change modelruntime into truth authority.
- Do not let model calls approve themselves.
- Do not bypass gateway.
```

---

# Phase 10 — Consensus Mesh Response/Action Gating

```text
PHASE 10: CONSENSUS MESH RESPONSE AND ACTION GATING

GOAL:
Bring Consensus Mesh online as a governed claim/action gate before final response/action proposal surfaces.

TASKS:
1. Define live claim extraction envelope.
2. Support claim sources: admitted evidence, model proposal, user assertion, tool result, audit/gateway/modelruntime refs.
3. Add risk classes: casual, operational, filesystem, host/system, legal/financial/security.
4. Define consensus thresholds per risk class.
5. Add conflict detection.
6. Add composer guard: high-risk claim requires consensus/evidence, unresolved conflict is disclosed or withheld, consensus accepted does not equal canonical truth.
7. Add tests: supported claim accepted, unsupported claim withheld, conflict detected, high-risk claim requires stronger support, no memory commit from consensus alone.
8. Add docs/architecture/live_consensus_mesh_gating.md.

VALIDATION:
- focused consensus tests
- npm test
- npm run build:core

DONE WHEN:
Response/action surfaces can be gated by consensus without making consensus canonical truth.

WHAT NOT TO DO:
- Do not make Consensus a second Kernel.
- Do not let consensus write memory.
- Do not treat consensus accepted as permanent truth.
```

---

# Phase 11 — Kernel Commit Authority Per Object Type

```text
PHASE 11: KERNEL COMMIT AUTHORITY FOR ONE OBJECT TYPE

GOAL:
Migrate exactly one semantic object type to Kernel-style syscall commit authority through the existing live owner and journal/replay path.

CHOOSE ONE OBJECT TYPE ONLY:
Recommended order:
1. notes
2. links
3. tags/labels
4. open loops
5. state records
6. contradiction records
7. supersession records
8. derived models
9. memory observations
10. context snapshots
11. policy objects
12. workflow/process projections

TASKS:
1. Identify current live write path.
2. Define FORGE-K target syscall.
3. Add/extend shared pure validator.
4. Add live adapter/facade.
5. Add journal event.
6. Add replay verification.
7. Add rollback path.
8. Add tests: valid commit, rejected malformed commit, audit/provenance present, journal event present, replay reconstructs state, old direct write path blocked or wrapped, rollback restores previous behavior.
9. Update docs/status/forge_k_authority_gates.md.
10. Add docs/reviews/phase_11_kernel_commit_<object_type>.md.

VALIDATION:
- focused object tests
- journal/replay tests
- npm test
- npm run build:core

DONE WHEN:
Exactly one object type commits through Kernel-style syscall boundaries with journal/replay proof.

WHAT NOT TO DO:
- Do not migrate multiple object types.
- Do not remove old path before compatibility/rollback is proven.
- Do not import simulator Kernel as live authority.
```

---

# Phase 12 — Lymphatic Lane Proposal Maintenance

```text
PHASE 12: LYMPHATIC LANE ONLINE AS PROPOSAL-FIRST MAINTENANCE

GOAL:
Bring Lymphatic Lane online as proposal-first hygiene without silent mutation or destructive cleanup.

TASKS:
1. Define MaintenanceReport and CleanupProposal live envelopes.
2. Add deterministic sweeps: stale loop detection, duplicate evidence detection, contradiction sweep, cache hygiene, snapshot hygiene, modelruntime result hygiene, retrieval/index hygiene.
3. All cleanup must produce proposals first.
4. Destructive proposals require approval.
5. Add NixOS timer/service only if safe and disabled-by-default or read-only.
6. Add tests: report generation, no silent deletion, approval required for destructive cleanup, provenance preserved, no canonical truth mutation from report alone.
7. Add desktop page/card for proposals if UI scope is included.
8. Add docs/architecture/live_lymphatic_lane.md.

VALIDATION:
- focused lymphatic live tests
- npm test
- npm run build:core
- nix flake check if Nix timer/module changed
- npm run validate:desktop if UI changed

DONE WHEN:
FORGE can recommend cleanup safely without mutating truth or deleting provenance silently.

WHAT NOT TO DO:
- Do not auto-delete evidence.
- Do not mutate memory from a sweep.
- Do not run host cleanup commands.
```

---

# Phase 13 — Live KV Reuse Readiness And Canary

```text
PHASE 13: LIVE KV REUSE READINESS AND CANARY

GOAL:
Prepare live KV reuse as acceleration only after deterministic identity is proven. Do not enable default KV reuse in this phase unless explicitly scoped as canary.

TASKS:
1. Verify current KV identity validation gates.
2. Add prompt layout hash integration.
3. Add tokenizer/template/runtime identity proof.
4. Add model revision proof.
5. Add final token identity check or explicit blocker if unavailable.
6. Add cache invalidation and eviction policy.
7. Add operator-visible hit/miss diagnostics.
8. Add canary flag default false.
9. Add tests: exact identity match allows canary reuse, mismatch denies reuse, unsupported final token identity denies reuse, cache miss recomputes, cache hit never changes truth/journal/memory, kill switch works.
10. Add docs/architecture/live_kv_reuse_readiness.md.

VALIDATION:
- focused KV identity/reuse tests
- npm run validate:forgek
- npm test
- npm run build:core

DONE WHEN:
KV reuse has a safe readiness/canary path and remains acceleration only.

WHAT NOT TO DO:
- Do not make KV cache memory.
- Do not enable default live reuse without evidence.
- Do not reuse across model/tokenizer/template/prompt identity mismatch.
```

---

# Phase 14 — Storage Cutover Readiness, Dual-Write, Read-Compare

```text
PHASE 14: STORAGE CUTOVER READINESS TO DUAL-WRITE/READ-COMPARE

GOAL:
Prepare storage backend cutover without confusing database migration with FORGE-K authority migration.

CHOOSE EXACTLY ONE SUBSTAGE:
A. readiness/blocker cleanup
B. Postgres repository adapter for one domain
C. dual-write shadow for one domain
D. read-compare for one domain
E. cutover proposal report only

TASKS:
1. Keep SQLite default unless this phase is explicitly cutover proposal/report only.
2. Pick one data domain.
3. Add parity tests.
4. Add checksums/count comparison.
5. Add rollback path.
6. Keep Qdrant shadow-only unless separately approved.
7. Keep Redis ephemeral-only.
8. Add tests: SQLite remains default, Postgres adapter parity, dual-write mismatch detected, read-compare mismatch detected, rollback restores SQLite.
9. Update docs/architecture/storage_backend_migration.md and phase review.

VALIDATION:
- focused storage tests
- npm test
- npm run build:core
- optional Postgres/Qdrant/Redis integration tests only when env exists

DONE WHEN:
One storage cutover substage is proven for one domain.

WHAT NOT TO DO:
- Do not switch global default storage.
- Do not make Qdrant truth.
- Do not make Redis canonical.
- Do not mix storage cutover with Kernel authority migration.
```

---

# Phase 15 — Operator Cockpit

```text
PHASE 15: FORGE-K OPERATOR COCKPIT

GOAL:
Add operator visibility for FORGE-K online status, gates, proposals, admissions, context, consensus, journal, replay, and rollback.

TASKS:
1. Design cockpit page sections: Kernel status, Authority Gate Matrix, Live vs Simulator Boundary, semantic syscall queue/status, Courthouse cases/admissions, evidence refs, Context Bundle inspector, Runtime Proposal inspector, Consensus decisions, journal/replay status, Lymphatic proposals, KV identity/cache status, storage cutover status, safe-mode/rollback status.
2. Use authenticated/read-only backend APIs.
3. Do not add mutation controls unless explicitly approval-gated and already supported.
4. Add tests: page renders, empty/degraded states render, blocked gates display as blocked, no mutation on load, accessibility basics.
5. Extract shared UI components where useful.

VALIDATION:
- npm run validate:desktop
- npm test if backend touched
- npm run build:desktop

DONE WHEN:
Operator can inspect FORGE-K online status and blockers without terminal spelunking.

WHAT NOT TO DO:
- Do not add dangerous buttons.
- Do not allow UI to mutate gates directly.
- Do not hide simulator/live boundaries.
```

---

# Phase 16 — CI/Test Gate Hardening

```text
PHASE 16: FORGE-K ONLINE CI AND TEST GATE HARDENING

GOAL:
Make authority migration impossible to merge without tests proving no bypass, no overclaim, and rollback safety.

TASKS:
1. Add CI checks for: no simulator service imports into live authority packages, authority gate enum validity, route inventory stability, gateway-only tool execution, modelruntime output proposal-only where integrated, no memory writes from model output, no unapproved host mutation path.
2. Add coverage targets for: controllane, gateway, FORGE-K shared contracts, journal/replay, admission, modelruntime proposal boundary.
3. Add optional weekly race/fuzz workflow if appropriate.
4. Add docs/testing/forge_k_online_validation_gates.md.

VALIDATION:
- CI config syntax checks where possible
- npm test
- npm run lint
- npm run validate:forgek
- npm run validate:local if feasible

DONE WHEN:
CI blocks the most dangerous FORGE-K online mistakes.

WHAT NOT TO DO:
- Do not require unavailable external services by default.
- Do not break local dev workflows.
- Do not mark optional integration tests as required without env gating.
```

---

# Phase 17 — Legacy Path Retirement

```text
PHASE 17: LEGACY PATH RETIREMENT FOR ONE MIGRATED AUTHORITY SURFACE

GOAL:
After a FORGE-K online path is proven, retire or wrap exactly one legacy direct mutation path.

PRECONDITION:
Do not run this phase unless the target surface already has live adapter/facade, tests, journal evidence, replay proof, rollback path, operator docs, and at least one prior phase report proving stable behavior.

TASKS:
1. Select one legacy path.
2. Document current behavior and replacement path.
3. Add deprecation warning or hard block depending on risk.
4. Keep read compatibility if needed.
5. Add tests: old write path blocked or wrapped, new path succeeds, audit/journal present, rollback behavior documented.
6. Update docs/status/current_authority_sources.md.
7. Update docs/reviews/current_phase_status.md.
8. Add docs/reviews/phase_17_legacy_retirement_<surface>.md.

VALIDATION:
- focused tests
- npm test
- npm run build:core
- npm run validate:desktop if UI routes changed

DONE WHEN:
One authority class has one live write path and the legacy mutation side door is gone or safely wrapped.

WHAT NOT TO DO:
- Do not retire broad systems.
- Do not break backup/restore.
- Do not remove compatibility reads unless explicitly approved.
```

---

# Emergency Rollback Prompt

```text
EMERGENCY ROLLBACK / SAFETY RESTORE

GOAL:
Revert unsafe FORGE-K authority expansion or restore safe defaults without deleting evidence.

TASKS:
1. Identify unsafe change: simulator service imported live, gateway bypass, modelruntime truth mutation, memory write from model output, host mutation path, public route leak, route behavior change, or failed tests.
2. Revert or disable the unsafe path using the narrowest safe change.
3. Ensure feature flags default false.
4. Preserve audit/journal/evidence records.
5. Add regression test proving the unsafe behavior is blocked.
6. Update phase review with what failed, what was reverted, tests added, and remaining blockers.
7. Run relevant validation.

WHAT NOT TO DO:
- Do not hide the failure.
- Do not delete evidence.
- Do not keep broken behavior behind unclear config.
- Do not continue feature work until safety is restored.
```

---

# Recommended Execution Order

```text
0. Baseline authority inventory
1. NixOS host envelope
2. Authority gate matrix
3. Shared pure contracts
4. Live semantic syscall facade
5. Journal/replay
6. Courthouse admission
7. Memory Palace mirror
8. Context Compiler shadow/canary/live
9. Runtime proposal boundary
10. Consensus Mesh gating
11. Kernel commit authority per object type
12. Lymphatic proposal maintenance
13. KV reuse canary
14. Storage cutover staged
15. Operator cockpit
16. CI/test gate hardening
17. Legacy path retirement
```

# Final Rule

FORGE-K fully online is not one merge. It is controlled authority migration.

NixOS owns the host. Gateway owns tools. Modelruntime owns drivers. FORGE-K owns semantic truth flow. The operator owns dangerous authority.
