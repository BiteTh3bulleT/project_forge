# Validation Results

## Scripts Inspected

Root scripts from `package.json`:

- `npm run build`
- `npm run build:core`
- `npm run build:desktop`
- `npm run typecheck`
- `npm run validate:desktop`
- `npm test`
- `npm run lint`
- `npm run smoke`

`npm install` was not run because `node_modules` and `package-lock.json` are present.

## Commands Run

| Command | Result | Notes | Confidence Impact |
|---|---:|---|---|
| `npm run build` | PASS | Desktop Vite build + VSA preflight + Go build passed. Vite emitted chunk-size warning. | High |
| `npm run build:core` | PASS | VSA tracked-file preflight + `go build ./...`. | High |
| `npm run build:desktop` | PASS | Vite build passed; chunk-size warning. | Medium |
| `npm run typecheck` | PASS | Desktop TypeScript typecheck passed. | Medium |
| `npm run validate:desktop` | PASS | Desktop typecheck + build passed. | Medium |
| `npm test` | PASS | VSA preflight + `cd services/core && go test ./...`. | High |
| `npm run lint` | PASS | Root lint delegates to `go vet ./...`; no JS lint configured. | Medium |
| `cd services/core && go test ./...` | PASS | Full Go test suite passed. | High |
| `cd services/core && go vet ./...` | PASS | Go vet passed. | High |
| `npm run smoke` | FAIL | Environment/tooling: invokes `bash ./scripts/forge-smoke.sh`; Windows shell reports `/bin/bash` missing. | Medium |

## Validation Notes

GOOD: Core build/test/vet and desktop build/typecheck are green.

RISK: Root `lint` is Go-only. There is no dedicated JS/TS lint lane.

RISK: No frontend test suite is configured.

BROKEN: Smoke is Bash-only and fails on this Windows machine. This is not evidence that core smoke behavior is broken, but it blocks Windows validation confidence.

