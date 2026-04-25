# Recommendation Backlog

## 1. Bind Gateway Approvals to Request Fingerprints

- Why: prevents approved request ID replay.
- Modules: `services/core/internal/gateway`, `approvals`, jobs.
- Complexity: medium.
- Risk: high.
- Dependency: approval schema/payload update.
- Acceptance: approval cannot be reused for different tool/lane/path/risk/actor/job.

## 2. Govern Model Management Actions

- Why: import/archive/remove/load/unload mutate runtime authority state.
- Modules: `api/model_runtime.go`, `modelruntime`, `gateway`.
- Complexity: medium.
- Risk: high.
- Dependency: capability taxonomy for `model.*`.
- Acceptance: destructive/privilege-affecting model actions require approval and emit traceable audit.

## 3. Add Retrieval/Observation Backup Parity

- Why: current restore loses retrieval provenance and usefulness signals.
- Modules: `backup`, `store`, `retrieval`, `memory`.
- Complexity: medium.
- Risk: high.
- Dependency: restore ordering decision.
- Acceptance: full backup exports/restores retrieval and observation compatibility evidence with tests.

## 4. Verify Backup Bundle Integrity Before Restore

- Why: prevents tampered/truncated restore bundles.
- Modules: `backup`.
- Complexity: small/medium.
- Risk: medium/high.
- Dependency: bundle metadata format.
- Acceptance: hash/count mismatch fails before mutation.

## 5. Fix Context Candidate Listing

- Why: scorer cannot rank candidates it never receives.
- Modules: `aios/controllane`, `sqlite_store.go`.
- Complexity: medium.
- Risk: medium.
- Dependency: scoring tests.
- Acceptance: related-query snapshots can be considered and ranked deterministically.

## 6. Persist Dream Reports

- Why: reports are evidence and need operator audit.
- Modules: `aios/dream`, `store`, `api`, desktop.
- Complexity: medium.
- Risk: medium.
- Dependency: schema migration.
- Acceptance: Dream run creates durable non-canonical report with correlation/trace.

## 7. Add Public Syscall Facade

- Why: replaces legacy mutation endpoints with governed semantic write ingress.
- Modules: `api`, `controllane`.
- Complexity: medium.
- Risk: medium/high.
- Dependency: approval UI/contract.
- Acceptance: dry-run default, commit behind approval, idempotent, audited.

## 8. Convert Remaining Bash Scripts

- Why: Windows operators cannot run smoke/dev helper scripts.
- Modules: `scripts`, `package.json`.
- Complexity: small/medium.
- Risk: medium.
- Dependency: none.
- Acceptance: `npm run smoke`, `desktop:check`, and `desktop:clean-port` run or gracefully skip on Windows.

## 9. Add Frontend Test Harness

- Why: desktop is feature-rich but untested beyond type/build.
- Modules: `apps/desktop`.
- Complexity: medium.
- Risk: medium.
- Dependency: test stack selection.
- Acceptance: Vitest/RTL plus Playwright smoke in CI.

## 10. Build Trace-First Operator Workflow

- Why: operator must explain what happened without stitching pages.
- Modules: `apps/desktop/src/pages/InspectorsPage.tsx`, audit/context/gateway APIs.
- Complexity: large.
- Risk: medium.
- Dependency: stable trace report schema.
- Acceptance: one correlation page shows chat, model, gateway, syscall, audit, artifacts, snapshots, Dream reports.

