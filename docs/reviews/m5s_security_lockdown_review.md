# M5S Security Lockdown Review

Date: 2026-05-16
Follow-up: 2026-05-17
Reviewed base HEAD: `7294952`
Task packet: `FORGE_M5S_Security_Lockdown_Prompt_Pack`

## Findings And Actions

| Area | Status | M5S action |
|---|---|---|
| API auth posture | Locked down | `/api/*`, `/forge/*`, and enabled `/v1/*` now require `Authorization: Bearer <token>`. `/health` remains public. Tokens load from `FORGE_API_TOKEN`, `FORGE_API_TOKEN_FILE`, or first-run `${FORGE_DATA_DIR}/auth/api_token`. |
| Wildcard bind | Fail-closed | `0.0.0.0` and `::` require both `FORGE_ALLOW_WILDCARD_BIND=true` and a non-empty API token. |
| Dockerfile | Locked down | The standalone core image no longer defaults to wildcard bind. Compose remains the explicit dev-network profile and passes API auth/CORS settings. |
| CORS | Tightened | Default CORS allows only Tauri origins and no-origin local calls. Localhost browser origins require `FORGE_CORS_ALLOW_DEV_LOCALHOST=true`; configured origins must be exact matches. |
| Approval decision authority | Split enforced | Approval and cancellation actor authority comes from authenticated request context. Request-body `actor` is ignored for authority. Approval requests now reject `approved` decisions from the same initiating/requesting authority and leave the request pending for a separate authority. |
| Project-context import | Scoped | Imports are restricted to the workspace root plus `FORGE_PROJECT_CONTEXT_ALLOWED_ROOTS`; absolute outside paths, `..` escapes, and symlink escapes are rejected. Existing size limits remain enforced. |
| Job restart recovery | Added | Startup recovery requeues queued jobs, marks interrupted preparing/running jobs failed with `interrupted`, leaves terminal jobs unchanged, emits bounded recovery events, and avoids unbounded queue goroutines. Shutdown cancels running job contexts before waiting. |
| Windows process parity | Added | `proc.run` no longer attempts `bash -lc` on Windows; it returns a structured unsupported result. `proc.terminate` uses Go process APIs instead of an unconditional `kill` command. |
| Secrets/token storage | Documented | `docs/architecture/secret_storage.md` documents current token custody, redaction expectations, and the deferred OS keychain hardening path. |
| API error responses | Structured | Remaining API handlers that wrote `http.Error(w, err.Error(), ...)` now route through structured JSON error helpers. Server-side failures return generic `internal_error` responses and log the internal error; a static API test blocks regressions. |
| Remote ingress | Hardened | Discord webhook ingress can require Ed25519 request signatures via `discord_public_key` / `FORGE_DISCORD_PUBLIC_KEY`, remote token compares use fixed-length hashes, and operator-entered remote access/Telegram/Discord secrets now persist through encrypted `secrets_vault` updates with legacy plaintext reads preserved for compatibility. |
| Settings SSRF/logging | Hardened | Explicit `/api/settings/ollama-models?baseUrl=` overrides reject localhost, loopback, private, link-local, multicast, unspecified, userinfo, unsupported schemes, oversized URLs, and unsafe DNS resolutions before HTTP. Router logging now emits paths only instead of raw URLs with query strings. |
| Gateway audit authority | Hardened | `/api/gateway/invoke` no longer accepts request-body initiator/provenance actor authority; authenticated request context supplies the audit actor fields. |
| Desktop logout | Hardened | Desktop logout clears the cached Tauri API-token promise so subsequent calls in the same window must re-read current auth state. |
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

Follow-up validation on 2026-05-17:

- `cd services/core; go test ./internal/approvals -count=1` passed.
- `cd services/core; go test ./internal/api -count=1` passed.
- `cd services/core; go test ./internal/gateway -count=1` passed.
- `cd services/core; go test ./internal/jobs -count=1` passed.
- `npm test` passed.

Additional high-priority follow-up validation on 2026-05-17:

- `cd services/core; go test ./internal/api -run "Test(PatchSettingsExplicitRemoteSecretUpdatesStoredValue|GetSettingsRedactsStoredRemoteSecrets|PatchSettingsRedactedRemoteSecretsPreserveStoredValues|Remote|Discord|GetOllamaModels|GatewayInvokeDoesNotTrustBodyAuthorityFields|ConstantTimeTokenMatchDoesNotEarlyReturnOnLengthMismatch|RoutesDoNotUseRawURLMiddlewareLogger|RequireAPIAuth)" -count=1` passed.
- `cd services/core; go test ./internal/api -count=1` passed.
- `cd services/core; go test ./internal/approvals ./internal/gateway ./internal/jobs -count=1` passed.
- `npm run validate:js` passed.
- `git diff --check` passed with line-ending warnings only.
