# FORGE Full System Review - Executive Summary

Date: 2026-04-25  
Scope: full repository review against FORGE AI-OS doctrine, current code, status docs, and local validation on Windows PowerShell.

## Overall Verdict

PARTIAL: FORGE is no longer a toy scaffold. The kernel/control-lane, gateway, modelruntime, audit, restore snapshots, Dream Mode v0, desktop shell, and operator inspection surfaces are real code with meaningful tests. The project has made the right architectural choices: kernel-owned truth, gateway-owned tools, modelruntime-owned inference, CPU/RAM authority, and proposal-only autonomy paths.

RISK: The system is not yet converged enough to call the AI-OS boundary complete. Several authority-adjacent APIs mutate operational state outside the same approval/gateway discipline expected for tools. Retrieval/observation persistence is not fully covered by backup/restore. Dream Mode is safe but ephemeral. Context restore scoring is deterministic but over-filtered in the SQLite path. Operator visibility exists, but it is not yet cohesive enough for a human to follow every chain without stitching pages together.

BROKEN: No critical build break remains in this checkout after the pre-existing Windows build fix. `npm run smoke`, `npm run desktop:check`, and `npm run desktop:clean-port` still fail on this host because they call Bash. Nix cannot be verified because `nix` is not installed.

GOOD: `npm run build`, `npm run build:core`, `npm run build:desktop`, `npm test`, `npm run lint`, `npm run typecheck`, `npm run validate:desktop`, direct `go test ./...`, direct `go vet ./...`, and Tauri `cargo check` pass.

## Top Findings

1. GOOD: The semantic syscall/control-lane kernel is implemented, tested, and append-only journal immutability is enforced by SQLite triggers.
2. GOOD: Gateway is the main tool execution path; the legacy adapter direct invoke route is removed and tested.
3. GOOD: Modelruntime M3 is real: registry, manifest parsing, lifecycle state, queueing, runtime policy, OpenAI-compatible backend, and audit exist.
4. PARTIAL: There is no public syscall-native operator/API facade for semantic memory/state mutation; internal compute/autonomy can use the kernel, but operator write ingress is incomplete.
5. RISK: Model management APIs import/archive/remove/load/unload directly, not through gateway-equivalent approval semantics.
6. RISK: Gateway approval grants can be reused when no job binding is supplied; approval should bind to request fingerprint.
7. RISK: Backup coverage misses retrieval runs/results, packet retrieval links, observation compatibility tables, and usefulness feedback.
8. RISK: Context restore scoring over-filters SQLite candidates by exact query before scoring, weakening the claimed ranking behavior.
9. PARTIAL: Dream Mode v0 is deterministic and safe, but reports/proposals are not durably persisted for later operator audit.
10. MISSING: Desktop/frontend has no unit/component/e2e test suite; current validation is build/typecheck only.

## Highest Priority Recommendations

1. Approval-bind model management and gateway approval grants.
2. Add full retrieval/observation backup coverage and restore integrity verification.
3. Loosen context snapshot candidate listing so scorer actually ranks, instead of exact-query prefiltering.
4. Persist Dream Mode reports as append-only non-canonical evidence.
5. Add a narrow syscall API facade for operator-controlled semantic writes with dry-run default.
6. Convert remaining Bash-only npm operational scripts to cross-platform Node wrappers.
7. Add frontend tests and a small Playwright operator smoke.
8. Tighten provider URL policy and external `/v1/*` exposure assumptions.
9. Add trace-first operator workflow linking chat/gateway/syscall/audit/artifacts.
10. Decide whether legacy observation memory remains compatibility-only or becomes fully governed/recoverable.

## Confidence

High for code paths covered by local validation and direct inspection.  
Medium for runtime behavior requiring long-lived GUI/dev-server interaction because this pass did not start the full desktop shell.  
Low for Nix behavior on this host because Nix is unavailable.

