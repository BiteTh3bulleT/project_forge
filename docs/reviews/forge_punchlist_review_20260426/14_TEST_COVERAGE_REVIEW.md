# Test Coverage Review

| Subsystem | Current Coverage | Gaps |
|---|---|---|
| Config | GOOD | Bind-host/public exposure tests missing. |
| Migrations | GOOD/PARTIAL | More idempotence and evidence immutability tests needed. |
| Backup/restore | GOOD/PARTIAL | Tamper/count fail-closed tests missing. |
| Semantic syscalls | GOOD | Public syscall API tests missing because API is absent. |
| Gateway/approval | GOOD/PARTIAL | SSRF, symlink, capability activation governance gaps. |
| Modelruntime | GOOD/PARTIAL | Streaming, process supervision, GPU telemetry admission, cost governance missing. |
| Context restore | PARTIAL | Near-match candidate retrieval and large-set benchmarks missing. |
| Dream Mode | GOOD/PARTIAL | Upsert/append decision and no-GPU test coverage should expand. |
| Rule Cells/Hyperlane | MISSING | No substrate tests because substrate is not implemented. |
| CPU-only safe mode | PARTIAL | Runbook exists; automated Windows/Linux smoke coverage missing. |
| API/UI | PARTIAL | Go API tests good; frontend tests absent. |
| Security | PARTIAL | SSRF, symlink, CORS, bind, secrets, process stress missing. |
| Startup/shutdown | PARTIAL | Some server shutdown tests; end-to-end smoke blocked on Windows. |
| Smoke path | BROKEN on Windows | Bash-only script fails without `/bin/bash`. |

## Highest-Risk Missing Tests

1. SSRF/private-network denial.
2. Symlink/path traversal across file, artifact, backup, model import, archive extraction.
3. Capability status governance approval/audit tests.
4. Backup restore tamper fail-closed tests.
5. Context restore near-match candidate tests.
6. Bind/CORS exposure tests.
7. Telegram remote sender/chat allowlist tests.
8. Frontend degraded-state tests.
9. Process output/time/resource limit tests.
10. Windows-compatible smoke test.

## Next Test-Only PRs

- Security boundary suite.
- Filesystem escape suite.
- Backup integrity suite.
- Context restore candidate suite.
- Desktop degraded-state test lane.

