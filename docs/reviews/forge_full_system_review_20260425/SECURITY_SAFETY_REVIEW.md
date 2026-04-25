# Security and Safety Review

## Strengths

GOOD: Dangerous tool taxonomy and approval-only status are explicit.

GOOD: Gateway policy checks paths, risk, approvals, provenance, and lane/profile permissions.

GOOD: Legacy memory mutation and adapter direct execution side doors are retired/bounded.

GOOD: Modelruntime does not default to cloud providers when unconfigured.

GOOD: CPU-only safe/degraded mode is documented and implemented enough for core survival.

## Risk Register

RISK: Approval ID replay without request fingerprint binding.
- Severity: high.
- Fix: bind approval to job/tool/lane/capability/paths/risk/write intent/actor.

RISK: Model management direct routes.
- Severity: high.
- Fix: approval-gate or gateway-route import/archive/remove/load/unload.

RISK: Provider endpoint SSRF/internal access if endpoints are user-writable.
- Severity: medium.
- Fix: scheme/host allowlist, local-only default, blocked metadata IPs, explicit cloud enablement.

RISK: `/v1/*` compatibility API exposure.
- Severity: medium.
- Fix: bind to loopback by default, require explicit enablement, add rate/auth policy, audit caller metadata.

RISK: Filesystem/path handling needs continued cross-platform hardening.
- Severity: medium.
- Fix: maintain tests for Windows and Unix absolute path behavior, symlinks, traversal, forbidden paths, and workspace-root behavior.

RISK: Bundle restore lacks integrity verification.
- Severity: medium/high.
- Fix: verify hash, section counts, schema version, and optional signature before restore.

RISK: Secrets logging.
- Severity: not confirmed.
- Fix: add tests that provider API keys/tokens are redacted from audit, settings responses, and errors.

## Tests to Add

- Approval replay rejection tests.
- Model management approval-required tests.
- Provider URL allowlist tests.
- Secret redaction tests.
- Symlink/path traversal tests.
- Restore tampering tests.
- `/v1/*` disabled/enabled exposure tests.

