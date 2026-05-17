# Claude Audit Summary

Critical findings:

1. No authentication on HTTP API.
2. Dockerfile defaults wildcard bind.
3. Approval requests can be self-approved by arbitrary actor string.
4. CORS allows arbitrary localhost origins.
5. Project-context import can read arbitrary paths.
6. Job queue does not recover after restart.
7. `proc.run` and `proc.terminate` assume Unix tools on Windows.
8. No root LICENSE.

High findings:
- Plaintext tokens/settings risk.
- Python/code execution inherits FORGE environment.
- Remote webhook/token hardening needed.
- SSRF gaps outside guarded fetch path.
- No metrics/race/Vitest/coverage gates.
- Desktop login is not backend auth.

Priority: M5S Security Lockdown before M5A latency/agent acceleration.
