# Security / Safety Review

## Risk Register

| ID | Severity | Risk | Affected Modules | Recommendation |
|---|---:|---|---|---|
| SEC-001 | high | Core binds all interfaces by default. | `services/core/main.go` | Default to loopback; require explicit public bind config. |
| SEC-002 | high | SSRF/private-network egress controls incomplete. | gateway network tools, provider endpoints | Deny metadata, loopback, link-local, private ranges unless explicitly allowed. |
| SEC-003 | high | Symlink/path traversal test coverage incomplete. | gateway file tools, artifacts, backup, model import, archive extraction | Add fixture suite and enforce resolved-path containment. |
| SEC-004 | high | Backup restore accepts arbitrary file path. | `backup/service.go`, API restore | Add boundary tests and policy gate. |
| SEC-005 | high | Telegram polling normal chat allowlist gap. | remote Telegram service | Enforce sender/chat allowlist. |
| SEC-006 | medium | CORS/local API posture depends on bind assumptions. | API server | Add CORS matrix tests. |
| SEC-007 | medium | Secret redaction coverage thin outside `secret.get`. | config/provider/remote diagnostics | Add redaction tests. |
| SEC-008 | medium | Process execution output can grow before truncation. | gateway process tool | Stream/cap buffers. |
| SEC-009 | medium | Workspace root defaults broad. | config/workspace path policy | Better first-run warning and tests. |

## Safety Posture

GOOD: Dangerous tools are generally gateway/approval/capability governed.

GOOD: Dream Mode and context snapshots are non-canonical evidence.

RISK: Network, filesystem, and remote ingress boundaries need adversarial tests before wider exposure.

## Tests Needed

- Bind host and CORS tests.
- SSRF denial table.
- Symlink escape fixtures.
- Archive extraction escape fixtures.
- Backup restore path-boundary tests.
- Secret redaction tests across logs/API errors/provider failures.
- Process output/timeout/env leakage tests.

