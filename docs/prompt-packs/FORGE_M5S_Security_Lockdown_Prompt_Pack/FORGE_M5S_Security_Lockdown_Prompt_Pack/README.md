# FORGE M5S Security Lockdown Prompt Pack

Generated: 2026-05-16  
Target repo: `BiteTh3bulleT/project_forge`  
Latest visible HEAD at pack build: `96fe8a84814c44b9446eb03935c82c2103665391` — `feat: boot operator VM into FORGE login`

## Sprint

**M5S — Security Lockdown**

M5S supersedes the prior M5A authority/latency sprint as the immediate next move. M5A remains queued after security.

## Why

Claude's audit identified a front-door security problem: a broad local daemon API, no obvious global backend auth, wildcard Docker bind defaults, permissive localhost CORS, approval self-approval risk, arbitrary project-context import risk, missing restart recovery for jobs, and platform-specific process tools.

The goal is to make FORGE defensible as a single-user local-first system before adding faster background agents, deeper cockpit controls, or more live authority.

## Use order

1. `00_MASTER_PROMPT_M5S.md`
2. `01_PHASE0_SECURITY_BASELINE_VERIFY.md`
3. `02_PHASE1_API_AUTH_AND_BIND_POLICY.md`
4. `03_PHASE2_APPROVAL_AUTHORITY_SEPARATION.md`
5. `04_PHASE3_PROJECT_CONTEXT_IMPORT_SCOPE.md`
6. `05_PHASE4_CORS_REMOTE_AND_SECRET_HARDENING.md`
7. `06_PHASE5_JOB_RECOVERY_AND_SHUTDOWN.md`
8. `07_PHASE6_WINDOWS_PROCESS_PARITY.md`
9. `08_PHASE7_OBSERVABILITY_CI_LICENSE.md`
10. `09_PHASE8_SECURITY_DOCS_AND_RUNBOOKS.md`
11. `10_POST_SECURITY_M5A_QUEUE.md`

## Stop condition

Do not resume M5A until:

- non-health routes require backend auth,
- wildcard bind cannot run unauthenticated,
- approval decisions require real authority,
- project-context imports are scoped,
- job recovery exists,
- security tests are present.
