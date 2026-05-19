# Phase CA1 Full Codebase Integrity Audit Report

Status: `AUDIT_ONLY / NO_RUNTIME_BEHAVIOR_CHANGE / NO_AUTHORITY_EXPANSION / NO_HOST_MUTATION / NO_FORGE_K_AUTHORITY_MIGRATION`

Date: 2026-05-18

## Summary

Phase CA1 completed the requested audit-only sweep and produced the main review, status, CSV findings index, and prioritized fix queue.

No runtime code was changed. Build/test validation is passing for Go, desktop JS/TS, FORGE-K parity, and Tauri Rust tests. Nix checks were attempted and blocked because `nix` is not available in this PowerShell environment.

## Files Created

- `docs/reviews/full_codebase_integrity_audit.md`
- `docs/reviews/full_codebase_integrity_findings.csv`
- `docs/reviews/full_codebase_integrity_fix_queue.md`
- `docs/reports/phase_ca1_full_codebase_integrity_audit.md`
- `docs/status/phase_ca1_full_codebase_integrity_audit.md`

## Validation Summary

Passed:

- `npm test`
- `npm run lint`
- `npm run validate:js`
- `npm run validate:desktop`
- `npm run validate:local`
- `npm run build`
- `npm run build:core`
- `npm run build:desktop`
- `cd services/core && go test ./...`
- `cd services/core && go vet ./...`
- `npm -w @forge/desktop run test`
- `npm -w @forge/desktop run typecheck`
- `npm -w @forge/desktop run build`
- `cd apps/desktop/src-tauri && cargo test`

Skipped or blocked:

- `npm install`: skipped because dependencies are already installed.
- `nix --version`, `nix flake check`, `nix build .#forge-core`, `nix build .#forge-desktop-shell`: blocked because `nix` is not recognized on PATH.

## Finding Counts

| Severity | Count |
| --- | ---: |
| Critical | 1 |
| High | 6 |
| Medium | 8 |
| Low | 2 |

## Top Findings

1. Critical: desktop Start menu shutdown/reboot calls direct Tauri host power commands despite no-direct-host-mutation docs.
2. High: unset `FORGE_WORKSPACE_DIR` defaults to `/` and can activate broad workspace-write policy.
3. High: Docker Compose still supplies wildcard bind and wildcard opt-in defaults.
4. High: `legacy.adapter.invoke` can invoke Ollama/model/network behavior with low/non-network metadata.
5. High: packaged shell login uses static client-side credentials.
6. High: persisted `ollamaBaseUrl` lacks the stricter validation used by the query override path.
7. High: operator VM/Ollama docs conflict with current Nix VM service configuration.
8. Medium: plain `http.Error` responses remain across API families.
9. Medium: dashboard can display runtime queue depth as `0` when telemetry is unavailable.
10. Medium: operator app catalogs are duplicated between Rust backend and TS fallbacks.

## Completion Criteria Check

| Criterion | Status |
| --- | --- |
| full audit report exists | done |
| status file exists | done |
| phase report exists | done |
| validation commands attempted or explicitly marked blocked | done |
| duplicate systems listed | done |
| placeholders/mocks classified | done |
| truncation/corruption scan documented | done |
| build/test errors recorded | done |
| stale docs identified | done |
| fix queue prioritized | done |
| no runtime behavior changed | done |

## Recommended Next Phase

Phase CA2 should be an explicit fix pass focused on authority/security corrections first:

1. remove or govern direct host power controls
2. fail closed on root/default workspace authority
3. remove wildcard bind defaults from Compose
4. retire or correctly classify `legacy.adapter.invoke`
5. validate persisted Ollama URLs

