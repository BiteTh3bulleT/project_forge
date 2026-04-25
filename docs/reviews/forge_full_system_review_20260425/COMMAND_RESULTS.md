# Command Results

Host: Windows PowerShell  
Date: 2026-04-25  
Note: `npm install` was not run because dependencies were already present and build/test commands executed successfully.

| Command | Result | Summary | Failure Type |
|---|---|---|---|
| `npm run build` | PASS | Desktop Vite build and Go core build completed. Vite emitted chunk-size warning. | n/a |
| `npm run build:core` | PASS | VSA preflight via Node script, then `go build ./...`. | n/a |
| `npm run build:desktop` | PASS | Vite build completed; main JS chunk warning. | n/a |
| `npm test` | PASS | Delegates to Go tests; all packages passed. | n/a |
| `npm run lint` | PASS | Delegates to `go vet ./...`; passed. | n/a |
| `npm run typecheck` | PASS | Desktop TypeScript checks passed. | n/a |
| `npm run validate:desktop` | PASS | Desktop typecheck plus Vite build passed; chunk warning only. | n/a |
| `cd services/core; go test ./...` | PASS | All Go package tests passed. | n/a |
| `cd services/core; go vet ./...` | PASS | No vet failures. | n/a |
| `cargo check` in `apps/desktop/src-tauri` | PASS | Tauri Rust crate checked successfully. | n/a |
| Go `gofmt -l` check | FAIL | Many existing Go files reported by `gofmt -l`; no mass formatting rewrite done in review pass. | repo hygiene / line-ending or formatting |
| `npm run smoke` | FAIL | Calls `bash ./scripts/forge-smoke.sh`; Bash unavailable through WSL relay. | environment/tooling + script portability |
| `npm run desktop:check` | FAIL | Calls Bash helper; Bash unavailable. | environment/tooling + script portability |
| `npm run desktop:clean-port` | FAIL | Calls Bash helper; Bash unavailable. | environment/tooling + script portability |
| `nix --version` | FAIL | `nix` executable not installed/available. | environment |

## Review Confidence Impact

Build/test confidence is high for Go core and desktop static build/typecheck paths.

Runtime smoke confidence is medium because the smoke script could not run on this host due Bash dependency.

Nix confidence is low because Nix was unavailable.

