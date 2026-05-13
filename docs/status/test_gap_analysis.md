# Test Gap Analysis (Phase 5.996)

Date: 2026-04-21
Scope: branch-local validation evidence after convergence hardening pass.

## Coverage snapshot

| Area | Status | Evidence | Main gap |
|---|---|---|---|
| Gateway/tool policy and dangerous capability gating | mostly complete | gateway unit/smoke tests; approval/disabled/unsupported/deferred/future_iris tests | broaden workspace/path boundary coverage |
| Semantic syscall + truth persistence | mostly complete | controllane + truth integration tests | expand projection rebuild/repair behavior tests |
| Backup export/restore parity | improved, near-complete | `internal/backup` tests now cover project context/evaluations/audit/gateway restore + rollback safety + unsupported reporting + `atomicScope`/warning reporting | VSA-derived export-only sections remain |
| Retired side-door hardening | improved | adapter invoke removed-route tests + retired memory mutation boundary tests + audit propagation checks + route/body mismatch rejection | keep retired routes non-executable and broaden syscall-native API coverage |
| Rule-agent safety | partial but safe | cleanup placeholder/destructive guard tests; propose-only runtime | broader deterministic agent set still missing |
| Desktop/frontend validation | partial | desktop build + root/desktop typecheck pass; root build/test/lint/typecheck commands runnable; Tauri `cargo check` passes | no dedicated JS/TS lint/test scripts |
| Nix foundation checks | blocked in env | flake/build commands attempted | local nix daemon unavailable |

## Validation command evidence (executed)

- `cd services/core && go test ./...` -> pass
- `cd services/core && go vet ./...` -> pass
- `cd services/core && go test ./internal/backup -count=1` -> pass
- `cd services/core && go test ./internal/api -run 'Adapter|LegacyMemory|Backup' -count=1` -> pass
- `npm install` -> not rerun in latest pass; installed dependencies were already present
- `npm run build` -> pass
- `npm run validate:local` -> pass (integration env preflight, desktop validation, FORGE-K validation, Go core build)
- `npm run smoke` -> pass
- `npm run typecheck` -> pass
- `npm test` -> pass (delegates to `go test ./...`)
- `npm run lint` -> pass (delegates to `go vet ./...`)
- `npm -w @forge/desktop run build` -> pass
- `cd apps/desktop/src-tauri && cargo check` -> pass
- `nix flake check` -> fail (`experimental Nix feature 'nix-command' is disabled`)
- `nix build .#forge-core` -> fail (`experimental Nix feature 'nix-command' is disabled`)
- `nix --extra-experimental-features 'nix-command flakes' flake check` -> fail (`cannot connect to nix daemon socket`)
- `nix --extra-experimental-features 'nix-command flakes' build .#forge-core` -> fail (`cannot connect to nix daemon socket`)

## Frontend validation posture

`npm run typecheck` now passes at root/desktop levels in this environment. `npm run validate:local` is available as a local aggregate check across integration env readiness, desktop validation, FORGE-K validation, and Go core build.

Remaining gap:
- no dedicated JS/TS lint/test scripts (current root `lint`/`test` delegate to Go tooling).
- Nix validation remains environment-blocked by local daemon availability.
