# FORGE Full Review + Punchlist

Status date: 2026-04-26

## Review Status

GOOD: Review completed as a current-state, read-only pass. No runtime code was changed.

PARTIAL: `npm run smoke` could not run on this Windows environment because the script invokes `bash ./scripts/forge-smoke.sh` and `/bin/bash` is unavailable.

## Sources Used

- `AGENTS.md`
- `README.md`
- `docs/architecture/forge_ai_os.md`
- `docs/architecture/semantic_syscalls.md`
- `docs/data_model/cognitive_filesystem.md`
- `docs/roadmap/forge_ai_os_phases.md`
- `docs/status/*` files requested in the prompt
- `docs/reviews/forge_full_system_review_20260425/*`
- `docs/journal/forge_master_technical_journal_20260425/*`
- Static code inspection across `services/core`, `apps/desktop`, `packages`, and `scripts`
- Parallel reviewer reports for authority, persistence, modelruntime, API/UI, architecture/autonomy, and security/testing/performance

MISSING: No binding source listed by the prompt was missing.

## Current Read

GOOD: FORGE has a real CPU/RAM core, semantic syscall control lane, durable cognitive filesystem tables, governed gateway, governed modelruntime management surface, deterministic context restore scoring, Dream Mode v0, Dream report persistence, backup coverage, and a desktop operator surface.

PARTIAL: Several features are implemented but not converged into a final operating model. The largest gaps are around security hardening, operator diagnostics, exact restore candidate behavior, frontend test discipline, modelruntime M4, and Rule Cell/Hyperlane substrate.

RISK: A few authority-adjacent APIs can still reshape runtime policy without strong approval/audit semantics: gateway capability status, lanes, and permission profile management.

BROKEN: Windows smoke validation is blocked by the Bash-only smoke script.

## Top 10 Findings

1. RISK: Core HTTP binds `":" + port`, exposing local APIs beyond loopback if the network/firewall allows it.
2. RISK: Gateway capability status changes can activate dangerous capabilities without a dedicated approval gate, and the API reason is not preserved by the gateway update call.
3. RISK: Telegram polling path appears to allow normal remote chat from any sender once remote polling is enabled; wake commands are allowlisted, normal messages appear weaker.
4. RISK: Context restore SQLite candidate listing exact-filters by query before scoring, undercutting documented lexical/near-match ranking.
5. RISK: Backup restore reports checksums/counts but does not clearly fail closed before mutation on bundle tampering or count mismatch.
6. RISK: SSRF/private-network denial and symlink traversal coverage is missing for network/file/archive/model/backup paths.
7. PARTIAL: Modelruntime M3 is real, but M4 remains substantial: streaming, process supervision, delete-file approval flow, durable scheduler state, and remote provider budget/cost policy.
8. PARTIAL: Dream reports persist as non-canonical evidence, but operator review/apply is intentionally absent and same-ID report persistence is upsert-style rather than append-only.
9. PARTIAL: UI inspector surfaces exist, but global degraded/runtime state is fragmented and the shell runtime pill is not modelruntime/safe-mode aware.
10. MISSING: Rule Cell/Hyperlane is still concept/scaffold, not an implemented deterministic low-latency rule substrate.

## Top 10 Punchlist Items

1. `AUTH-001`: Govern gateway capability status changes.
2. `SEC-001`: Bind core to loopback by default and document/configure public binding explicitly.
3. `SEC-002`: Add SSRF/private-network denial for network tools and provider endpoints.
4. `CTX-001`: Fix restore candidate retrieval to rank near matches.
5. `DUR-001`: Make backup checksum/entity-count verification fail closed before restore mutation.
6. `SEC-003`: Add symlink/archive/path traversal fixture suite.
7. `UI-001`: Add global operator diagnostics and accurate safe-mode/modelruntime state.
8. `MR-001`: Implement modelruntime streaming and cancellation-safe accounting.
9. `TEST-001`: Add JS/TS test and lint lanes; replace or wrap Bash smoke for Windows.
10. `RULE-001`: Either implement Rule Cell/Hyperlane v0 or keep it explicitly concept-only in status docs.

## Next 3 Recommended Passes

1. Authority hardening: capability status governance + lane/profile audit.
2. Security boundary hardening: loopback binding, SSRF denylist, symlink/path traversal tests.
3. Restore correctness: near-match candidate retrieval, matching indexes, and restore inspector UI scope cleanup.

