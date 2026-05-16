# M5S Security Lockdown Review

Date: 2026-05-16
Reviewed base HEAD: `7294952`
Task packet: `FORGE_M5S_Security_Lockdown_Prompt_Pack`

## Findings And Actions

| Area | Status | M5S action |
|---|---|---|
| API auth posture | Locked down | `/api/*`, `/forge/*`, and enabled `/v1/*` now require `Authorization: Bearer <token>`. `/health` remains public. Tokens load from `FORGE_API_TOKEN`, `FORGE_API_TOKEN_FILE`, or first-run `${FORGE_DATA_DIR}/auth/api_token`. |
| Wildcard bind | Fail-closed | `0.0.0.0` and `::` require both `FORGE_ALLOW_WILDCARD_BIND=true` and a non-empty API token. |
| Dockerfile | Locked down | The standalone core image no longer defaults to wildcard bind. Compose remains the explicit dev-network profile and passes API auth/CORS settings. |
| CORS | Tightened | Default CORS allows only Tauri origins and no-origin local calls. Localhost browser origins require `FORGE_CORS_ALLOW_DEV_LOCALHOST=true`; configured origins must be exact matches. |
| Approval decision authority | Tightened | Approval and cancellation actor authority comes from authenticated request context. Request-body `actor` is ignored for authority. |
| Project-context import | Scoped | Imports are restricted to the workspace root plus `FORGE_PROJECT_CONTEXT_ALLOWED_ROOTS`; absolute outside paths, `..` escapes, and symlink escapes are rejected. Existing size limits remain enforced. |
| Job restart recovery | Added | Startup recovery requeues queued jobs, marks interrupted preparing/running jobs failed with `interrupted`, leaves terminal jobs unchanged, emits bounded recovery events, and avoids unbounded queue goroutines. Shutdown cancels running job contexts before waiting. |
| Windows process parity | Added | `proc.run` no longer attempts `bash -lc` on Windows; it returns a structured unsupported result. `proc.terminate` uses Go process APIs instead of an unconditional `kill` command. |
| Secrets/token storage | Documented | `docs/architecture/secret_storage.md` documents current token custody, redaction expectations, and the deferred OS keychain hardening path. |
| CI/test posture | Improved | CI now runs `npm run validate:js` so desktop Vitest is covered alongside typecheck/build. Focused security tests were added for auth, CORS, import scoping, job recovery, and Windows process behavior. |

## Remaining Boundaries

- This does not make FORGE-K simulator services live daemon authority.
- CORS is not treated as authentication.
- The generated file token is a local baseline, not a long-term multi-user credential store.
- M5A authority convergence remains queued after M5S verification and push.

## Validation Evidence

- `cd services/core; go test ./internal/config ./internal/api ./internal/projectcontext ./internal/jobs ./internal/gateway` passed.
- `npm run validate:js` passed.
- `cd apps/desktop/src-tauri; cargo fmt --check; cargo check; cargo test` passed.
- `npm test` passed.
- `npm run lint` passed.
- `npm run build` passed.
- `npm run smoke` on the default port was blocked by an existing local listener on `127.0.0.1:18492` (PID 17468); rerun with `FORGE_CORE_PORT=18493` passed.
- `npm run validate:local` passed.
- `git diff --check` passed.
