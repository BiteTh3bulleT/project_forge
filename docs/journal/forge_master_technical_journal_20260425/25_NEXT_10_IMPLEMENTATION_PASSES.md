# Next 10 Implementation Passes

## 1. Approval Fingerprint Hardening

Goal: Ensure approvals cannot be reused for different request shapes.  
Status: IMPLEMENTED in current working tree for gateway approval grants.  
Acceptance: Tool/path/actor/lane/workspace/risk/input mismatches are denied and audited.  
Tests: Gateway replay tests and approvals pending-request fingerprint tests.  
What not to do: Do not weaken approval policy.

## 2. Model Management Governance

Goal: Put import/archive/remove/load/unload behind gateway-equivalent governance.  
Why: Model registry state is runtime authority.  
Likely files: `api/model_runtime.go`, `modelruntime`, `gateway`.  
Acceptance: Risky model actions require approval and audit with trace.  
Tests: approval required, approved success, denied failure, audit linkage.

## 3. Backup / Restore Parity

Goal: Close retrieval/observation/VSA restore posture.  
Likely files: `backup`, `store`, `retrieval`, `memory`.  
Acceptance: Supported sections restore; export-only sections are explicit and tested.  
Tests: round-trip, missing table, rollback, hash/count integrity.

## 4. Context Candidate / Scoring Fix

Goal: Let scorer rank candidates instead of exact-query prefiltering them away.  
Likely files: `aios/controllane/*restore_scoring*`, SQLite snapshot list path.  
Acceptance: Related prior snapshots can be scored, rejected, or selected deterministically.  
Tests: query similarity, wrong workspace exclusion, stale penalty, fresh fallback.

## 5. Dream Report Persistence

Goal: Persist Dream Mode reports as non-canonical evidence.  
Likely files: `store/migrate.go`, `aios/dream`, `api/dream.go`, desktop later.  
Acceptance: Dream run creates durable report/proposal rows without canonical commit.  
Tests: dry-run persists evidence, no semantic mutation, trace/audit linkage.

## 6. Operator Restore / Dream Inspector

Goal: Let operators inspect restore scoring and Dream proposals.  
Likely files: `apps/desktop`, `api`, trace report.  
Acceptance: UI shows candidate scores, fresh-compile reason, Dream proposals, no truth confusion.  
Tests: API tests plus Playwright smoke.

## 7. Rule Cell Substrate / Hyperlane v0

Goal: Implement deterministic rule packs and routing traces.  
Likely files: `aios/autonomy`, new rule substrate package, docs.  
Acceptance: Rule Cells emit proposal/score/route traces only.  
Tests: no direct mutation, latency budget, trace output.

## 8. Restore Outcome Feedback Loop

Goal: Feed operator/result feedback into future restore scoring.  
Likely files: `controllane`, `store`, `api`, UI.  
Acceptance: Restore outcomes are non-canonical evidence and influence scoring deterministically.  
Tests: feedback persistence, score effect, no truth mutation.

## 9. Modelruntime M4 Supervision / Streaming Planning

Goal: Design and implement stronger backend supervision, streaming, and lifecycle safety.  
Likely files: `modelruntime`, `api/model_runtime*`, config.  
Acceptance: Streaming is bounded/audited; backend process state is inspectable.  
Tests: cancellation, timeout, queue pressure, audit, no retry storm.

## 10. UI Dual-Mode Cleanup

Goal: Separate Cognitive State Viewer from Metrics Board.  
Likely files: `apps/desktop/src/pages`, shell/nav config.  
Acceptance: Operator can follow trace, approvals, memory, restore, Dream, models without page stitching.  
Tests: frontend unit tests and Playwright operator smoke.

