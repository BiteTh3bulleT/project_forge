# Validation Plan

## Required for every phase

- `npm test`
- `npm run lint`
- `npm run validate:forgek`
- `npm run build:core`
- targeted Go tests for changed packages
- targeted desktop tests if UI changed
- Nix checks if Nix files changed

## Required for authority migration

- simulator tests
- shared pure package tests
- live owner tests
- malformed input fail-closed tests
- capability denial tests
- audit/diagnostic field tests
- no route/API change tests unless route change is explicit
- no unauthorized gateway/tool execution tests
- no unauthorized modelruntime call tests
- no unauthorized retrieval/search/embedding tests
- no unauthorized memory write tests
- no simulator-service live import tests
- rollback/disabled-mode tests

## Report format

Use `docs/reports/PHASE_REPORT_TEMPLATE.md`.
