# Testing And Validation

## Current Commands

- `cd services/core && go test ./...`
- `cd services/core && go vet ./...`
- `npm test`
- `npm run lint`
- `npm run typecheck`
- `npm run validate:desktop`
- `npm run build`
- `npm run build:desktop`
- `npm run build:core`

## Coverage Matrix

| Subsystem | Coverage | Status |
|---|---|---|
| Semantic syscalls | Registry, validation, processor, SQLite, restore scoring tests | GOOD |
| Gateway | Tool surface, policy, approvals, dangerous capabilities, fingerprint replay tests | GOOD |
| Approvals | Request/decision lifecycle and fingerprint pending reuse | PARTIAL/GOOD |
| Modelruntime | Manifest/store/registry/backend/service/management tests | GOOD |
| Backup | Export/restore/rollback/context/VSA export-only tests | PARTIAL |
| Dream Mode | Dry-run and no-modelruntime/no-canonical-mutation tests | PARTIAL |
| Autonomy | Policy/runner/safety/repository tests | PARTIAL |
| API | Many Go route tests | PARTIAL |
| Desktop | Build/typecheck only | MISSING tests |
| Nix | Not verified here | NOT VERIFIED |

## Critical Tests To Add

1. Model management approval tests.
2. Retrieval/observation backup/restore tests.
3. Restore hash/entity-count tamper rejection.
4. Context candidate ranking beyond exact query.
5. Dream report persistence tests.
6. Public syscall facade dry-run/approval/idempotency tests.
7. Trace-first UI Playwright smoke.
8. Provider SSRF/secret redaction tests.
9. Cross-platform script tests.
10. Frontend unit/component tests.
11. Shared contract test for approval `expired` state.
12. Backup restore approval/confirmation policy tests.
