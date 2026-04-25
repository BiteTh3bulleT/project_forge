# Test Coverage Review

## Coverage Matrix

| Area | Status | Notes |
|---|---|---|
| Config loading | GOOD | Go tests exist; Windows path assumptions were fixed in local working tree. |
| Migrations/schema evolution | PARTIAL | Migration tests exist for VSA/modelruntime; broader schema upgrade history needs expansion. |
| Backup/restore | PARTIAL | Good core tests; retrieval/observation sections and integrity verification missing. |
| Semantic syscalls | GOOD | Controllane validator/processor/integration tests exist. |
| Rollback/dry-run | PARTIAL | Kernel dry-run and backup rollback tests exist; cross-system non-DB rollback limited by design. |
| Context snapshots/scoring | PARTIAL | Tests exist; SQLite exact-query candidate behavior needs coverage. |
| Dream Mode | PARTIAL | Service tests exist; API/operator/report durability tests missing. |
| Modelruntime | GOOD/PARTIAL | Strong backend/service/API tests; management approval and remote discovery posture missing. |
| Provider failures | PARTIAL | Some backend tests; more cooldown/blacklist/URL policy tests needed. |
| Gateway | GOOD | Strong capability/policy/smoke tests; approval replay binding missing. |
| Autonomy | PARTIAL | Policy/runner/safety tests exist; budget consumption gaps. |
| UI/API | PARTIAL | Backend API tests exist; frontend tests absent. |
| Safe mode | PARTIAL | Modelruntime safe-mode tests exist; full boot matrix still thin. |
| Startup/shutdown idempotence | PARTIAL | Server shutdown test exists; full startup/desktop/smoke cross-platform still thin. |

## Critical Missing Tests

1. Gateway approval fingerprint binding.
2. Model management approval gates.
3. Backup parity for retrieval/observation.
4. Restore bundle hash/tamper rejection.
5. SQLite context snapshot partial-query ranking.
6. Dream report persistence and UI/API retrieval.
7. Autonomy budget consumption on gateway tools.
8. Provider endpoint allowlist and secret redaction.
9. Frontend component/e2e tests.
10. Cross-platform npm script validation.

## Validation Notes

GOOD: Root build/test/lint/typecheck are green in this checkout.

BROKEN: Bash-only smoke and desktop helper scripts fail on Windows PowerShell.

PARTIAL: gofmt check reports many files. Because build/test pass and this was a review pass, no mass formatting rewrite was performed.

