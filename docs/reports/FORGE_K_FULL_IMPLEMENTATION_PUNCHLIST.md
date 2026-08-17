# FORGE-K Full Implementation Punch List

**Created:** 2026-08-17
**Objective:** Implement the complete documented production FORGE architecture while preserving FORGE-K as the sole live semantic authority.
**Status vocabulary:** `DONE`, `IN PROGRESS`, `NOT STARTED`, `BLOCKED`, `INTENTIONALLY NON-AUTHORITATIVE`.

This is the active execution ledger for the instruction **“implement everything.”**
Older phase reports and simulator documents are design/history inputs, not proof that a
capability is live. An item is complete only when its production path, failure behavior,
tests, operator surface, documentation, and deployment evidence are all complete.

## Completion rule

A subsystem is complete only when all of these are true:

- [ ] Its production contract is implemented through `internal/forgekernel` where it can affect semantic truth, admission, context, execution authority, or model visibility.
- [ ] Models, adapters, Future IRIS, desktop applications, Nix, and infrastructure cannot claim Kernel authority.
- [ ] Every meaningful mutation is authenticated, capability-scoped, provenance-bound, journaled, audited, idempotent, and atomic.
- [ ] Current truth, historical evidence, proposals, projections, and acceleration data remain distinct.
- [ ] Replay is deterministic and does not recommit.
- [ ] Corrupt, stale, cross-scope, unbound, unsupported, and legacy inputs fail closed.
- [ ] There is no alternate production writer or orchestration path.
- [ ] Unit, integration, rollback, concurrency, tamper, restart, race, static-authority, and operator acceptance tests pass where applicable.
- [ ] Status surfaces report observed capability state rather than phase-name optimism.
- [ ] Architecture, ADR, API, runbook, implementation matrix, and recovery documentation agree with the code.

## 0. Baseline that must remain true

- [x] **DONE — One production authority.** Daemon assembly constructs one `forgekernel.Kernel`; the old live authority selector is removed.
- [x] **DONE — Atomic semantic commit proof.** Implemented mutations bind authorization, prepared plan, receipt, provenance, journal chain/head, audit intent, and idempotency evidence.
- [x] **DONE — Production Courthouse core.** Admission and appeal decisions are deterministic Kernel decisions with immutable rulings.
- [x] **DONE — Governed admitted memory core.** Materialization and revision use current admitted Court evidence and append-only supersession.
- [x] **DONE — Governed context decision core.** Model prompts bind a production Kernel Context Compiler decision.
- [x] **DONE — Model visibility boundary.** Current model-output surfaces pass through runtime-proposal and consensus decisions before persistence or visibility.
- [x] **DONE — Legacy memory writers retired.** Observation/link/repair mutation paths are terminal, proposal-only, or fail closed.
- [x] **DONE — Full offline development-workstation profile.** The OptiPlex profile packages the declared desktop, office, IDE, language, container, peripheral, and FORGE tooling without granting semantic authority.

## 1. P0 — Integrity, recovery, and authority foundations

These items block an honest claim of complete production readiness.

### 1.1 Daemon-stopped whole-store recovery

- [x] **DONE — Build the offline recovery command/service and add Nix distribution.** Live row merge remains permanently disabled.
- [x] Validate raw bundle digest, schema version, normalized section set, manifest, row counts, checksums, duplicate identities, and workspace identity before writing.
- [x] Restore into a newly created store, never into the live database.
- [x] Verify migrations, foreign keys, immutable-table triggers, Court chains, memory supersession, semantic-operation identity, authorization proofs, idempotency proofs, audit outbox, and the journal chain/head.
- [x] Refuse recovery while the daemon owns the store or the target lock cannot be proven.
- [x] Preserve the previous store as a timestamped recoverable generation.
- [x] Atomically swap the verified store and fsync the file and containing directory.
- [x] Automatically roll back the swap when post-open verification fails.
- [x] Add corrupt bundle, truncated chain, divergent head, duplicate identity, stale schema, disk failure, interrupted swap, restart, and rollback tests.
- [x] Add Nix package, operator command, runbook, status evidence.
- [ ] Add physical OptiPlex recovery drill (command and checklist complete; execution pending on operator station).

### 1.2 Durable audit projection delivery

- [x] **DONE — Replace best-effort `RecordResult` delivery with a durable projector.** Canonical outbox insertion remains in the Kernel transaction.
- [x] Add append-only delivery-attempt records; never mutate the canonical outbox intent.
- [x] Make delivery idempotent by outbox identity and request fingerprint.
- [x] Verify the stored request, authorization proof, receipt, journal entry, and committed object identities before delivery.
- [x] Backfill external `audit_id` linkage through a bounded projection record rather than rewriting immutable semantic evidence.
- [x] Add retry/backoff, poison-record quarantine, restart resume, observability, and an operator replay command.
- [x] Prove sink outage cannot invalidate or block a canonical Kernel commit and cannot lose the delivery intent.
- [x] Add duplicate-delivery, tamper, partial-failure, restart, and long-outage tests.

### 1.3 Finish the Kernel/durable-port split

- [ ] **NOT STARTED — Move decision and policy ownership out of `aios/controllane`.** Control Lane may remain a package temporarily, but only typed gather/apply/storage ports may remain.
- [ ] Split the combined Control Lane `Processor` test facade from production durable-port implementations.
- [ ] Move action registry, capability bindings, semantic validation policy, object planning, and deterministic decisions into production Kernel-owned packages.
- [ ] Keep SQLite statements and transaction mechanics behind narrow storage adapters.
- [ ] Remove exported write-capable repositories from feature packages or split them into read-only and Kernel-transaction-only types.
- [ ] Add exact constructor/call-site/SQL authority guards for every canonical and behavior-affecting table.
- [ ] Delete dormant direct-writer bodies after all consumers migrate.
- [ ] Demonstrate that importing a storage adapter cannot authorize, decide, replay, or expose model output.

### 1.4 Status truth and readiness

- [ ] **NOT STARTED — Replace phase-derived readiness with observed gates.** Each claimed capability must probe its actual dependency and integrity state.
- [ ] Separate `kernelAuthorityExclusive`, `capabilityImplemented`, `projectionHealthy`, `recoveryVerified`, `hostReady`, and `unsafeTestMode` fields.
- [ ] Never use “all authority gates ready” as shorthand for “all architecture implemented.”
- [ ] Remove stale simulator-only, rollback-selector, live-restore, and shell-authority wording from active docs and UI.
- [ ] Add a generated status consistency test across API status, implementation matrix, current phase, runbooks, and the desktop System page.

## 2. P0 — Govern every behavior-affecting subsystem

### 2.1 Memory Palace completion

- [ ] Add authenticated, scope-exact read/inspection endpoints for Court cases, exhibits, rulings, admitted memory evidence, revisions, and provenance.
- [ ] Implement governed memory relationship edges as immutable Kernel contracts; reject caller authority claims and cycles.
- [ ] Implement governed usefulness/quality signals for immutable admitted evidence as append-only events with rebuildable aggregates.
- [ ] Migrate every remaining reader from legacy observations/notes where admitted memory is the intended authority source.
- [ ] Keep legacy observation and legacy VSA tables historical/read-only; provide an offline verified migration tool if preservation is required.
- [ ] Add proposal-only repair generation plus separately approved append-only evidence revision application.
- [ ] Prove cross-workspace/lane/selected-path isolation and current-leaf behavior across every reader.

### 2.2 Retrieval, search, and embeddings

- [ ] Move retrieval request planning, source eligibility, ordering, budgets, and result commitments into a pure Kernel-owned contract.
- [ ] Keep search and embedding engines bounded compute drivers; bind their exact version/config/model/input/output hashes.
- [ ] Add governed batch retrieval-outcome evidence and remove remaining retrieval-adjacent direct writers.
- [ ] Make source indexing/ingestion an authenticated, resumable, Kernel-governed workflow with immutable source/file/chunk manifests.
- [ ] Add exact-scope vector projection manifests, atomic staging/swap, and stale-head rejection.
- [ ] Make Qdrant optional acceleration only; SQLite/K evidence remains recoverable truth.
- [ ] Add deterministic fallback when embeddings or vector infrastructure are unavailable.
- [ ] Test two-workspace identical content, deletion/tombstone, reindex failure, model-version change, stale vector, partial batch, and replay.

### 2.3 Semantic Algebra

- [ ] Retain `COMPUTE_SEMANTIC_DIFF` as the completed first operator.
- [ ] Implement separate pure contracts for merge, intersect, contradiction, supersede, compress/summarize, derive, classify, and route only where product behavior requires them.
- [ ] Bind each operator to immutable admitted evidence, exact scope, fixed versioned policy, deterministic limits, and content commitments.
- [ ] Persist operation facts/results as immutable evidence; never auto-promote output to canonical truth.
- [ ] Require a separate Court admission/materialization action for any result that may become current memory.
- [ ] Retire or narrow legacy `CREATE_LINK`, `MARK_SUPERSEDED`, `REGISTER_CONTRADICTION`, and `DERIVE_MODEL` paths after equivalent governed contracts land.
- [ ] Add algebraic golden vectors, permutation tests, Unicode/bounds fuzzing, cycle tests, rollback, replay, and simulator-import guards.

### 2.4 State, loops, artifacts, and derived models

- [ ] Audit every currently supported low-risk action for exact kind/scope/version/hash resolution—not existence-only checks.
- [ ] Require nonempty idempotency for every persisted action.
- [ ] Make lifecycle transitions append-only or versioned; remove discarded update errors.
- [ ] Bind artifacts to immutable byte hashes and verify file/DB atomicity or explicitly stage/promote them.
- [ ] Add artifact, provenance, and journal-ID trace lookup surfaces.
- [ ] Retire generic legacy semantic mutation actions when typed Kernel contracts supersede them.

### 2.5 Context Compiler completion

- [ ] Preserve the current pure Kernel decision and immutable exact-scope bundle path.
- [ ] Add typed source kinds only through explicit admitted/governed manifests.
- [ ] Migrate Dream, automation, librarian, and every prompt consumer to the same immutable bundle contract.
- [ ] Remove remaining legacy context snapshot mutation seams and make verified snapshots immutable/inspection-only.
- [ ] Add content-commitment receipt verification for every persisted bundle object.
- [ ] Add durable reread verification at model-visibility time so an in-memory binding cannot outlive tampered storage.

## 3. P0 — Model, tool, and execution authority

### 3.1 Runtime proposal and consensus completion

- [ ] Supply governed deterministic/no-model evidence so ordinary safe model responses do not remain unnecessarily withheld.
- [ ] Forward exact gateway request/result/audit evidence into the final consensus input.
- [ ] Reject denied, errored, unsupported, or merely approval-required tool states as evidence of completed action.
- [ ] Persist the final visibility decision and exact visible bytes as immutable noncanonical response evidence.
- [ ] Re-read and verify context/runtime/gateway commitments before visibility and on replay.
- [ ] Add all sync/stream/cancel/disconnect/tamper/restart tests proving no pre-gate token leakage.

### 3.2 Tool router replacement and gateway hardening

- [ ] Remove `smuxoAI` from configured routing, packaging, fixtures, docs, and target model inventory.
- [ ] Evaluate and select a small local router using a reproducible task/tool benchmark on OptiPlex CPU/RAM limits.
- [ ] Make FORGE—not the model—the final tool-selection authority; the router may only produce a bounded proposal.
- [ ] Validate autonomy intent and charter existence rather than accepting nonempty identifiers.
- [ ] Reject unknown budget identities; reserve and consume tool budget atomically for successful calls.
- [ ] Bind tool schema selection, arguments, approval, execution result, audit identity, and model completion claim.
- [ ] Complete workspace/path/network/device capability boundary tests and configured-dependency failure harnesses.
- [ ] Surface approval-required capability state and replay controls consistently in the desktop.

### 3.3 Model Runtime completion

- [ ] Finish backend/process supervision for backends FORGE actually launches; distinguish external operator-managed backends.
- [ ] Implement bounded scheduling, admission control, queue fairness, backpressure, cancellation, and usage accounting under concurrency.
- [ ] Enforce per-model memory/CPU/GPU budgets and hybrid P-core/E-core scheduling policy where the platform exposes it.
- [ ] Add model health, crash-loop quarantine, restart policy, load/unload lifecycle, and operator override evidence.
- [ ] Eliminate configuration precedence that lets stale database settings silently override Nix security constraints.
- [ ] Keep remote/cloud models disabled on the isolated OptiPlex unless the operator explicitly changes the host policy.
- [ ] Add soak, cancellation, overload, backend-loss, malformed-stream, token-accounting, and restart tests.

### 3.4 Live KV acceleration

- [ ] Implement tokenizer/model/prompt/context/runtime-policy identity commitments for reusable KV state.
- [ ] Add a bounded runtime cache adapter with atomic manifests and content-addressed entries.
- [ ] Verify identity before every reuse; mismatch is a cache miss, never a semantic error or memory source.
- [ ] Add eviction, quota, corruption, restart, version-change, cancellation, and cross-scope isolation tests.
- [ ] Keep simulator KV packages non-authoritative or retire them after the production pure contract supersedes them.

## 4. P1 — Automation, autonomy, learning, and maintenance

### 4.1 Durable autonomy execution

- [ ] Replace the static approval placeholder with resolution of durable approval decisions bound to intent/charter/budget/action fingerprints.
- [ ] Validate that intent, charter, budget, reservation, and approval records exist, are current, scope-compatible, and unexpired.
- [ ] Atomically reserve/consume/refund budgets across syscall and gateway execution outcomes.
- [ ] Add a durable scheduler with leases, crash recovery, deduplication, bounded retries, and operator pause/kill controls.
- [ ] Make maintain/mission mode production-capable without weakening Kernel/gateway approval rules.
- [ ] Provide complete trace/audit visibility from trigger through intent, decision, proposal, execution, and result.

### 4.2 Automation rules

- [ ] Route non-dry automation effects through autonomy plus typed Kernel/gateway contracts; remove direct job/dossier/review writes.
- [ ] Make rules and histories immutable/versioned and authority-scoped.
- [ ] Add scheduled/event triggers, leases, missed-run policy, retry policy, and deterministic replay.
- [ ] Add loop prevention, budget exhaustion, duplicate trigger, crash recovery, and cross-workspace tests.

### 4.3 Dream and learning

- [ ] Wire Academy sources, lab, grader, curriculum, and evaluation evidence to real bounded implementations.
- [ ] Let Dream produce durable proposals/reports by default; allow commits only through autonomy and Kernel contracts.
- [ ] Remove hard-coded dry-run behavior once commit gates and acceptance tests exist.
- [ ] Preserve model/training output as evidence; require deterministic validation and Court admission before memory promotion.
- [ ] Implement governed feedback consumption from append-only projections rather than legacy mutable outcome rows.
- [ ] Add learning-version manifests, reproducible evaluation sets, regression thresholds, rollback, and operator review.

### 4.4 Lymphatic maintenance and repair

- [ ] Port useful simulator Lymphatic policies into pure production proposal contracts.
- [ ] Keep cleanup/repair proposals non-mutating until separately authorized apply contracts run.
- [ ] Implement archive/compaction only with verified recovery points and immutable retention evidence.
- [ ] Add stale-evidence, orphaned-projection, corrupt-cache, disk-pressure, partial-failure, and no-silent-delete tests.

### 4.5 Deterministic rule agents

- [ ] Expand beyond the two current agents only with versioned signals, fixed policy, bounded scope, tests, and trace output.
- [ ] Add agents for recovery readiness, audit-delivery backlog, stale projection detection, model health, budget anomalies, and source-integrity drift.
- [ ] Keep every agent proposal-only unless it invokes a separately authorized Kernel/gateway action.

## 5. P1 — Storage and infrastructure completion

### 5.1 SQLite production hardening

- [ ] Add explicit busy/locking policy, WAL checkpoint supervision, disk-full handling, integrity checks, backup cadence, and operator diagnostics.
- [ ] Add schema migration rehearsal against real prior generations and recovery fixtures.
- [ ] Verify all immutable/projection/authority triggers and foreign keys on every boot.

### 5.2 Postgres option

- [ ] Decide whether Postgres is a supported production profile or remove it from the completion target.
- [ ] If supported: implement repository parity, migrations, transactional proof parity, journal ordering/CAS, idempotency, recovery, and integration tests.
- [ ] Add an explicit operator-controlled migration/cutover/rollback path; never silently dual-author truth.

### 5.3 Redis option

- [ ] Use Redis only for ephemeral coordination/cache after durable SQLite/Postgres records exist.
- [ ] Implement lease/queue/cache adapters with TTL, fencing tokens, restart recovery, and fail-open-as-cache-miss semantics where safe.
- [ ] Prove Redis loss cannot lose jobs, approvals, semantic state, audit evidence, or autonomy budgets.

### 5.4 Qdrant option

- [ ] Implement it only as a rebuildable exact-manifest retrieval acceleration.
- [ ] Add collection/version lifecycle, atomic rebuild/swap, stale vector rejection, scope filtering, and deterministic fallback.
- [ ] Prove deletion or corruption of Qdrant changes performance, not truth or recoverability.

## 6. P1 — Desktop and operator experience

- [ ] Make Settings a complete OS/operator control center for network adapters, routes, DNS, displays, audio, printing, scanning, removable media, storage, power, services, models, autonomy, recovery, and security posture.
- [ ] Route host mutations through bounded authenticated host-control contracts; show exact command/policy/result evidence.
- [ ] Replace interim shell lock with compositor/session-manager locking.
- [ ] Implement real display topology/resolution/scale management.
- [ ] Complete network profile creation/edit/activation with offline-policy visibility and a recovery path.
- [ ] Add recovery, audit delivery, journal integrity, model health, autonomy budgets, and blocked-capability dashboards.
- [ ] Add object-centric Court/memory/provenance/journal inspection and navigation.
- [ ] Add accessibility, keyboard-only, multi-monitor, suspend/resume, crash recovery, and long-session acceptance tests.

## 7. P1 — NixOS and physical OptiPlex completion

- [ ] Preserve the complete office, IDE, language, container, graphics, media, backup, peripheral, and developer workstation package set.
- [ ] Make the display manager, fallback desktop, FORGE session, TTY recovery, power controls, settings tools, and session environment reproducible.
- [ ] Enforce the requested offline policy declaratively: loopback + direct management subnet only, no default route, no cloud-model egress.
- [ ] Prevent other NetworkManager profiles/interfaces from silently restoring internet access.
- [ ] Add Nix assertions and static checks for offline firewall, direct-link SSH, service configuration, packages, extensions, permissions, and safety flags.
- [ ] Add the OptiPlex configuration/check to generic OS-integration validation and CI evaluation.
- [ ] Capture physical evidence: cold boot, graphical login, FORGE session, fallback desktop, IDE, office apps, terminal, containers, audio, printer/scanner visibility, USB, shutdown/reboot, direct NIC SSH, model chat, tool execution, autonomy, recovery, and offline-negative checks.

## 8. P2 — Product completeness and secondary integrations

- [ ] Implement currently narrow Discord/Telegram commands through the same authenticated intent and Kernel/gateway boundaries, or explicitly remove them from the product target.
- [ ] Finish trace lookups by artifact, provenance, journal, Court, memory-evidence, model-request, gateway-invocation, and autonomy-decision identity.
- [ ] Add automated audit/archive retention only after verified recovery and archive integrity exist.
- [ ] Add tests for `failurepatterns`, `packetopt`, `packets`, and `strategies` if they remain in production use; remove dead packages otherwise.
- [ ] Add real JS/TS linting and non-desktop package tests.
- [ ] Review all scaffolds, deferred flags, shadow systems, and simulator packages: promote through typed production contracts or archive/remove them.
- [ ] Reconcile every architecture/roadmap document with implemented code and tag target-state material explicitly.

## 9. Final system acceptance

These gates are run only after Sections 1–8 are complete.

- [ ] `npm test`
- [ ] `npm run lint`
- [ ] `npm run validate:js`
- [ ] `npm run validate:desktop`
- [ ] `npm run test:repo-hygiene`
- [ ] `npm run validate:repo-hygiene`
- [ ] `npm run test:os-integration`
- [ ] `npm run validate:os-integration`
- [ ] `npm run validate:local`
- [ ] `npm run smoke`
- [ ] `cd services/core && go test -race ./...`
- [ ] `cd services/core && go vet ./...`
- [ ] Relevant fuzz targets and sustained runtime/load/chaos tests pass.
- [ ] `nix flake check`
- [ ] `nix build .#forge-core .#forge-desktop-shell .#checks.x86_64-linux.forge-optiplex-7000`
- [ ] Clean-machine installation and daemon-stopped recovery drill pass.
- [ ] Physical OptiPlex acceptance checklist passes with captured logs/screenshots and no unintended internet route.
- [ ] Authority audit finds exactly one live Kernel authority and no unapproved canonical or behavior-affecting writers.
- [ ] Capability audit finds no partial, blocked, scaffold, deferred, or simulator-only item still represented as production-complete.
- [ ] Status/docs/UI all report the same observed completion state.
- [ ] Commit, merge to `main`, push, deploy the pinned generation, and record the recovery generation.

## Explicit non-goals and boundary decisions

These are not missing Kernel implementations:

- NixOS, the compositor, the desktop shell, office applications, VSCodium, containers, databases, and model engines remain host/runtime infrastructure.
- Models and routing models remain proposal generators, never authority owners.
- Qdrant, Redis, KV state, and VSA remain acceleration or coordination, never canonical truth.
- Raw chat, tool output, model output, snapshots, and Dream reports remain evidence/proposals until admitted by an explicit Kernel contract.
- “Unsafe full test mode” may lower operator approval policy on the isolated host, but it must not create alternate writers, bypass the journal, forge identity, or grant model authority.

## Progress reporting format

Every completed checkbox must record:

1. production path and authority owner;
2. migrations and compatibility behavior;
3. tests and exact commands run;
4. rollback/recovery behavior;
5. documentation and operator-surface changes;
6. deployment generation/evidence when host behavior changes;
7. remaining blockers without completion overclaim.
