# Next Passes

## Next 3 Immediate Short Passes

1. Capability governance hardening
   - Goal: approval/audit-gate dangerous capability status activation.
   - Why now: closes an authority-shaping side door.
   - Scope: gateway capability API/service/tests.
   - Tests: dangerous capability activation denied without approval; actor/reason persisted.
   - Do not do: redesign gateway taxonomy.

2. Security boundary pass
   - Goal: loopback bind default, SSRF deny table, symlink/path traversal tests.
   - Why now: prevents local-first API becoming accidental network-exposed unsafe surface.
   - Scope: config/main/gateway/file/network tests.
   - Tests: bind/CORS/SSRF/symlink fixtures.
   - Do not do: add remote auth product features.

3. Restore candidate correctness
   - Goal: remove exact-query prefilter and add indexes.
   - Why now: restore scoring claims depend on ranking near matches.
   - Scope: `ListContextSnapshots`, restore scorer tests, migrations.
   - Tests: similar query selected, wrong workspace excluded, query remains bounded.
   - Do not do: introduce LLM/vector truth scoring.

## Next 10 Implementation Passes

1. Capability status governance.
2. Loopback/SSRF/symlink security hardening.
3. Context restore candidate retrieval/index fix.
4. Backup restore tamper fail-closed gate.
5. Telegram remote sender/chat allowlist.
6. Global operator diagnostics and shell degraded state.
7. Modelruntime provider budget/egress governance.
8. Dream report append/upsert decision plus operator review workflow docs.
9. Public semantic syscall dry-run/submit/inspect API.
10. Rule Cell/Hyperlane v0 design spike and starter registry.

## Next 5 Test-Only Passes

1. Security boundary fixture suite.
2. Backup integrity tamper suite.
3. Context restore large-candidate suite.
4. Frontend degraded-state tests.
5. Windows-compatible smoke test.

## Next 5 Documentation Passes

1. Update stale 20260425 review docs with completed passes.
2. Update roadmap phase matrix for Dream persistence/operator inspector.
3. Document public binding/security defaults.
4. Document restore candidate scoring reality after fix.
5. Document Rule Cell/Hyperlane as concept until implemented.

## Do Not Do Yet

- Do not implement Dream apply/commit mode before syscall design and operator review workflow.
- Do not add GPU Dream jobs.
- Do not make cloud providers default fallback.
- Do not launch a full UI cockpit redesign before diagnostics/security hardening.
- Do not add Rule Cells that can mutate canonical truth directly.

