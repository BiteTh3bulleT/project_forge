# MASTER PROMPT — FORGE M5S Security Lockdown

You are working inside `BiteTh3bulleT/project_forge`.

Latest visible HEAD at pack creation: `96fe8a84814c44b9446eb03935c82c2103665391` — `feat: boot operator VM into FORGE login`.

Your task is to implement **M5S — Security Lockdown**.

Do not output code in chat. Make changes in files. Final response must summarize changed files, tests run, pass/fail, and remaining blockers.

---

## Mission

Make FORGE safe enough to run as a local-first single-user system without relying on an unenforced loopback-only assumption.

The M5A authority/latency work remains valid, but security comes first.

---

## Mandatory reading

Read before editing:

- `README.md`
- `AGENTS.md`
- `docs/status/current_authority_sources.md`
- `docs/status/implementation_matrix.md`
- `docs/status/current_baseline_gate.md`
- `docs/status/test_gap_analysis.md`
- `services/core/main.go`
- `services/core/Dockerfile`
- `services/core/internal/api/routes.go`
- `services/core/internal/api/phase2.go`
- `services/core/internal/projectcontext/service.go`
- `services/core/internal/jobs/service.go`
- `services/core/internal/gateway/service_process.go`
- `services/core/internal/api/server_settings.go`
- `services/core/internal/api/remote.go`
- `services/core/internal/gateway/capability_backing_execute.go`
- `.github/workflows/ci.yml`
- `apps/desktop/src/pages/ForgeLoginPage.tsx` if present
- `apps/desktop/src-tauri/src/main.rs`

---

## Required outcomes

### 1. Security baseline review

Create:

- `docs/reviews/m5s_security_lockdown_review.md`

Include:

- current HEAD,
- API auth posture,
- wildcard bind posture,
- Dockerfile posture,
- CORS posture,
- approval decision posture,
- project-context import posture,
- job restart recovery posture,
- Windows process parity posture,
- secrets/token posture,
- CI/test posture.

### 2. Backend API auth

Implement real auth on all non-health routes.

Minimum requirements:

- `/health` remains public.
- `/forge/*`, `/api/*`, and `/v1/*` require backend auth.
- Use `Authorization: Bearer <token>`.
- Token source may be `FORGE_API_TOKEN`, `FORGE_API_TOKEN_FILE`, or generated first-run token under `FORGE_DATA_DIR`.
- Do not log token.
- Missing/invalid token returns structured `401`.
- Desktop/Tauri must send token through backend-verified auth, not client-only sessionStorage.

Tests:

- `/health` no token: allowed.
- `/forge/models` no token: rejected.
- `/api/gateway/invoke` no token: rejected.
- `/v1/chat/completions` no token: rejected.
- valid token allows read-only route.

### 3. Wildcard bind fail-closed

Rules:

- If bind host is `0.0.0.0` or `::`, startup fails unless auth is enabled and token exists.
- Dockerfile must not default to unauthenticated wildcard bind.
- Secure Docker run example must be documented.

Tests:

- wildcard bind without token fails config validation.
- wildcard bind with token succeeds.
- Dockerfile no longer sets `FORGE_ALLOW_WILDCARD_BIND=true` by default.

### 4. Approval authority separation

Fix approval decisions.

Rules:

- Approval actor must come from authenticated request identity.
- Body `actor` is not authority.
- High-risk approvals default non-public.
- Public approval must be explicit and low-risk only.
- Approval decisions must audit authenticated actor, decision, and request id.

Tests:

- unauthenticated approval rejected.
- body actor alone rejected.
- high-risk public/self approval rejected.
- authorized approval succeeds and audits.

### 5. Tighten CORS

Rules:

- Production/default allows exact Tauri origins and explicitly configured origins only.
- Localhost wildcard allowed only under explicit dev flag.
- CORS is not an auth boundary.

Tests:

- random `http://localhost:*` rejected by default.
- Tauri origin accepted if required.
- explicit configured origin accepted.
- dev flag allows localhost.

### 6. Scope project-context import

Rules:

- Default allowed root is workspace dir.
- Optional `FORGE_PROJECT_CONTEXT_ALLOWED_ROOTS` may extend it.
- Reject absolute paths outside allowed roots.
- Reject `..` escapes.
- Symlink behavior must be explicit and tested.
- Preserve size limits.

Tests:

- workspace context allowed.
- `/etc/passwd` rejected.
- `../../secret` rejected.
- symlink escape rejected or explicitly handled.

### 7. Job recovery on restart

Implement job reconciliation on startup.

Rules:

- queued jobs requeue,
- running jobs become failed/recoverable or requeue only if safe/idempotent,
- terminal jobs unchanged,
- recovery is idempotent,
- recovery emits event/audit evidence,
- shutdown cancels workers cleanly.

Tests:

- running job before restart reconciled.
- queued job requeued.
- completed untouched.
- recovery idempotent.
- queue overflow goroutine does not leak past Close.

### 8. Windows process tool parity

Rules:

- No unconditional `bash -lc` on Windows.
- No unconditional `kill` command on Windows.
- Use build tags or platform-specific implementation.
- Unsupported shell semantics return structured unsupported errors.

Tests:

- Windows code path does not depend on bash/kill.
- Unix behavior preserved.

### 9. Secrets/token storage

Immediate posture:

- generated auth token not logged,
- settings reads redact secrets,
- backup/export secret exposure is either prevented or explicitly marked,
- create `docs/architecture/secret_storage.md` if OS keychain is deferred.

Future target:

- DPAPI on Windows,
- Keychain on macOS,
- libsecret/KWallet on Linux.

### 10. LICENSE, CI, and observability

Add:

- root `LICENSE`,
- desktop Vitest in CI or `npm run validate:js`,
- security-focused tests.

Consider:

- `/metrics` behind auth/config,
- scheduled `go test -race`,
- security event counters.

---

## Validation commands

Attempt:

```bash
npm test
npm run lint
npm run validate:js
npm run validate:local
npm run build
npm run smoke
```

Focused fallback:

```bash
cd services/core && go test ./internal/api ./internal/approvals ./internal/jobs ./internal/projectcontext ./internal/gateway -count=1
npm -w @forge/desktop run test
npm -w @forge/desktop run typecheck
```

Record failures honestly.

---

## Definition of Done

M5S is done when:

- non-health routes require backend auth,
- wildcard bind cannot run unauthenticated,
- Docker no longer ships wide-open,
- CORS no longer trusts arbitrary localhost by default,
- approval decisions require authenticated authority,
- project-context import is allowed-root scoped,
- jobs recover or fail safely after restart,
- Windows process tools are platform-aware,
- root LICENSE exists,
- secrets posture is documented and redaction tested,
- security tests are added,
- M5A is explicitly queued after security.

---

## WHAT NOT TO DO

- Do not output code in chat.
- Do not treat ForgeLoginPage/client sessionStorage as backend auth.
- Do not rely on CORS as auth.
- Do not rely on localhost as production security.
- Do not expose wildcard bind without auth.
- Do not let body.actor define approval authority.
- Do not allow arbitrary project-context imports.
- Do not log tokens/secrets.
- Do not make FORGE-K live authority.
- Do not add host mutation.
- Do not add new model execution side channels.
- Do not break `/health`.
- Do not mark environment-blocked checks as passed.
