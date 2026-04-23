# VSA Authority Report

Date: 2026-04-22

## Decision

Final disposition: **A. Commit/source-control path**.

VSA status: **authoritative source**.

VSA files are authoritative **source files**, not generated artifacts, cache outputs, or optional experiments. Current FORGE compiles and boots with direct references to VSA methods/types in memory, retrieval, and API surfaces.

## Required Files

| File | Required | Dependency type | Notes |
|---|---|---|---|
| `services/core/internal/memory/vsa_engine.go` | yes | authoritative source | core vector encoding/bind/similarity runtime |
| `services/core/internal/memory/vsa_indexer.go` | yes | authoritative source | reindex pipeline, pointer/binding/association maintenance |
| `services/core/internal/memory/vsa_signals.go` | yes | authoritative source | retrieval VSA signal computation/persistence |

Related tracked test coverage aligned in this pass:
- `services/core/internal/api/phase_memory_vsa_test.go`
- `services/core/internal/memory/vsa_engine_test.go`
- `services/core/internal/memory/vsa_indexer_test.go`
- `services/core/internal/retrieval/service_vsa_test.go`
- `services/core/internal/store/migrate_vsa_test.go`

## Tracked State Verification

Verified with:

```sh
git ls-files \
  services/core/internal/memory/vsa_engine.go \
  services/core/internal/memory/vsa_indexer.go \
  services/core/internal/memory/vsa_signals.go
```

Result in this branch/workspace: all required VSA source files are now git-tracked.

## Command Dependency Map

| Command | Depends on required VSA sources | Why |
|---|---|---|
| `npm run build:core` | yes | preflight enforces required VSA files and tracked-state before `go build ./...` |
| `npm run test:core` | yes | preflight enforces required VSA files and tracked-state before `go test ./...` |
| `npm run vet:core` | yes | preflight enforces required VSA files and tracked-state before `go vet ./...` |
| `cd services/core && go test ./...` | yes | `internal/memory`, `internal/retrieval`, and API handlers compile/link against VSA methods |
| `cd services/core && go vet ./...` | yes | same compilation graph |
| `npm run build` (`build:core`) | yes | `go build ./...` traverses VSA call sites |
| `npm run core` | yes | core binary boots with memory/retrieval/API VSA wiring present |
| `npm run smoke` | yes | smoke starts core and probes VSA-capable runtime surfaces |

## Why Not Generated/Optional

1. No deterministic VSA source generator exists in repository scripts/tooling.
2. Core code calls VSA methods directly (not behind build tags or plugin boundaries).
3. API routes for VSA endpoints are registered in normal server wiring.
4. Retrieval code invokes VSA signal flow under runtime mode settings; files must compile regardless of mode.

## Outcome

- Fresh-clone/source-control ambiguity is resolved in this branch state by tracking required VSA sources.
- Strict `--require-tracked` preflight remains useful as an integrity guard across core/smoke/core build-test-vet scripts, but no longer blocks due to old untracked-file ambiguity when repo state is authoritative.
