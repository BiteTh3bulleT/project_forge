# Security And Risk Register

| ID | Risk | Severity | Likelihood | Current Mitigation | Recommended Fix | Modules |
|---|---|---:|---:|---|---|---|
| R-001 | Model management mutates runtime authority without gateway-equivalent approval | High | Medium | Audit and modelruntime policy hooks | Add `model.*` capabilities or equivalent approval policy | `api/model_runtime*`, `modelruntime`, `gateway` |
| R-002 | Approval reuse across request shapes | High | Low after fix | Approval fingerprint hardening | Keep tests; extend to authority-adjacent APIs | `gateway`, `approvals` |
| R-003 | Backup misses retrieval/observation/VSA parity | High | Medium | Export-only warnings | Add restore coverage or explicit recompute contract | `backup`, `retrieval`, `memory` |
| R-004 | Restore tamper/truncation not fully verified | Medium/High | Medium | Transactional DB restore | Hash/count verification before mutation | `backup` |
| R-005 | Context scoring overfilters candidates | Medium | Medium | Fresh compile fallback | Broaden candidate list then rank deterministically | `controllane` |
| R-006 | Dream reports ephemeral | Medium | High | Dry-run non-canonical safety | Persist non-canonical reports/proposals | `aios/dream`, `store`, `api` |
| R-007 | Operator trace invisibility | Medium | High | Audit/trace API and pages | Build trace-first operator workflow | `api`, `apps/desktop` |
| R-008 | Provider URL SSRF/secrets exposure | Medium | Medium | Config defaults off | Endpoint allowlist and redaction tests | `modelruntime`, `embeddings`, `config` |
| R-009 | Dangerous tools misconfiguration | High | Low | Approval-only defaults | Service-specific harness tests | `gateway`, `permissions` |
| R-010 | Bash-only operational scripts on Windows | Medium | High on Windows | Direct Go/npm commands work | Convert to Node wrappers | `scripts`, `package.json` |
| R-011 | Frontend regressions | Medium | Medium | Build/typecheck | Vitest/Playwright | `apps/desktop` |
| R-012 | Nix claims not verified here | Low/Medium | Medium | Docs label blocked | Validate on Nix host | `nix` |
| R-013 | Backup restore is administrative mutation outside gateway approval | Medium/High | Medium | Restore audit and DB transaction scope | Add approval policy or explicit operator-confirmed gate | `api/phase5.go`, `backup` |
| R-014 | Shared approval status contract may omit backend expiry state | Low/Medium | Medium | Backend handles expiry | Align shared types and UI states | `packages/shared`, `approvals` |

## Prompt / Model Threat Model

An unsafe prompt or model output should be able to propose but not directly commit truth, execute tools, alter permissions, load untrusted models, or leak secrets. The remaining work is to make every authority-adjacent mutation match that standard.
