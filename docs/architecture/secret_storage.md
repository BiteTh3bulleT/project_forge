# Secret Storage

Status: Phase M5S local lockdown baseline.

FORGE now requires a local API bearer token for `/api/*`, `/forge/*`, and enabled `/v1/*` routes. `/health` remains public for readiness checks.

## Current Storage

- `FORGE_API_TOKEN` is the highest-precedence source and is never logged by the core.
- `FORGE_API_TOKEN_FILE` may point at an operator-managed token file.
- When neither is set, the core generates a first-run token at `${FORGE_DATA_DIR}/auth/api_token` with owner-only file permissions where the platform supports them.
- Native Tauri reads the token from `FORGE_API_TOKEN`, `FORGE_API_TOKEN_FILE`, or the default data-dir token file through `read_forge_api_token`, then sends `Authorization: Bearer <token>` on API requests.
- The desktop login session in `sessionStorage` remains a UI lock state only. It is not API authority.

## Deferred Hardening

OS keychain integration is intentionally deferred until the shell/session boundary is stable. The current file-backed token is acceptable only for the local single-operator baseline and container development flows.

Future work should move long-lived operator secrets into platform keychains or a sealed FORGE credential service with deterministic audit refs. Backups and exports must either exclude token material or explicitly mark secret-bearing artifacts as non-shareable operator custody material.
