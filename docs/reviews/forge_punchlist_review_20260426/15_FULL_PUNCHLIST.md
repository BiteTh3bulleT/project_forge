# Full Punchlist

## Immediate Blockers

### `TEST-001` Windows-compatible smoke
- Category: testing
- Severity: high
- Complexity: medium
- Status: test
- Why it matters: current `npm run smoke` fails on Windows due missing Bash.
- Affected: `scripts/forge-smoke.sh`, `package.json`
- Fix: add Node/PowerShell smoke or wrapper.
- Acceptance: `npm run smoke` passes on Windows and Linux.
- Validation: `npm run smoke`
- Dependency: none
- Short pass: yes

## Authority / Safety

### `AUTH-001` Govern capability status changes
- Severity: high
- Complexity: medium
- Status: fixed in follow-up pass
- Affected: `api/phase5.go`, `gateway/service.go`, `tool_capability_registry.go`
- Fix: require approval/audit for activating dangerous/high-risk capabilities; persist actor/reason.
- Acceptance: dangerous capability activation without approval is denied/held for approval; reason appears in override/audit.
- Resolution note: `PATCH /api/gateway/capabilities/{id}/status` now classifies transitions, opens/verifies approval requests for high-risk elevation, preserves actor/reason/provenance metadata, and blocks stale direct gateway activation paths.
- Validation: `go test ./internal/api/... ./internal/gateway/...`
- Short pass: yes

### `AUTH-002` Remote Telegram sender allowlist
- Severity: high
- Complexity: medium
- Status: bug
- Affected: `api/telegram_gateway_service.go`, `api/remote.go`
- Fix: enforce sender/chat allowlist for normal remote polling messages.
- Acceptance: unauthorized sender cannot create/process chat.
- Validation: `go test ./internal/api/...`
- Short pass: yes

### `AUTH-003` Audit authority-shaping APIs
- Severity: medium
- Complexity: small
- Status: gap
- Affected: lanes, permission profiles
- Fix: immutable audit records for save/delete/activate.
- Acceptance: audit rows exist for every authority-shaping mutation.
- Validation: `go test ./internal/api/...`
- Short pass: yes

### `AUTH-004` Shared approval fingerprint semantics
- Severity: medium
- Complexity: medium
- Status: refactor
- Affected: `permissions.Check`, gateway approval paths
- Fix: prevent broad approval lifting outside gateway fingerprint validation.
- Acceptance: direct permission caller cannot reuse unrelated job approval.
- Validation: `go test ./internal/permissions/... ./internal/gateway/...`
- Short pass: yes

## Durability / Persistence

### `DUR-001` Restore tamper fail-closed
- Severity: high
- Complexity: medium
- Status: bug
- Affected: `backup/service.go`
- Fix: validate bundle checksums/entity counts before applying restore.
- Acceptance: tampered bundle leaves DB unchanged.
- Validation: `go test ./internal/backup/...`
- Short pass: yes

### `DUR-002` Evidence immutability decision
- Severity: medium
- Complexity: medium
- Status: gap
- Affected: migrations, snapshots, Dream reports, history tables
- Fix: add triggers or documented audited update paths.
- Acceptance: evidence/history mutation semantics are tested.
- Validation: `go test ./internal/store/... ./internal/backup/...`
- Short pass: no

## Context / Restore

### `CTX-001` Near-match restore candidate retrieval
- Severity: high
- Complexity: medium
- Status: bug
- Affected: `aios/controllane/sqlite_store.go`, restore scoring
- Fix: fetch by workspace/lane/kind/recency, rank query similarity in scorer.
- Acceptance: similar query candidates are ranked, wrong workspace excluded.
- Validation: `go test ./internal/aios/controllane/... ./internal/api/...`
- Short pass: yes

### `CTX-002` Restore indexes and benchmarks
- Severity: medium
- Complexity: medium
- Status: upgrade
- Affected: migrations, restore tests
- Fix: add composite index and large fixture benchmark.
- Acceptance: restore candidate query remains bounded.
- Validation: `go test ./internal/store/... ./internal/aios/controllane/...`
- Short pass: yes

## Dream Mode

### `DRM-001` Dream report append/upsert decision
- Severity: medium
- Complexity: small
- Status: docs/test
- Affected: `aios/dream/service.go`, docs
- Fix: declare upsert intentional or make reports append-only.
- Acceptance: tests match documented behavior.
- Validation: `go test ./internal/aios/dream/...`
- Short pass: yes

### `DRM-002` Restore feedback feed loop
- Severity: medium
- Complexity: medium
- Status: upgrade
- Affected: Dream reports, restore scoring
- Fix: record non-canonical restore outcome feedback.
- Acceptance: restore scoring can inspect prior outcome evidence without making it truth.
- Validation: targeted Go tests
- Short pass: no

## Rule Cells / Hyperlane

### `RULE-001` Rule Cell / Hyperlane v0
- Severity: medium
- Complexity: large
- Status: gap
- Affected: new/aios rule substrate
- Fix: deterministic registry, lane filters, trace, no-mutation output.
- Acceptance: starter rules run under latency budget and produce proposals only.
- Validation: new rule package tests
- Short pass: no

## Modelruntime / Providers

### `MR-001` Streaming
- Severity: high
- Complexity: large
- Status: upgrade
- Affected: modelruntime service/API
- Fix: SSE/OpenAI-compatible streaming with cancellation-safe audit/usage.
- Acceptance: streaming clients work and partial streams are accounted.
- Validation: `go test ./internal/modelruntime/... ./internal/api/...`
- Short pass: no

### `MR-002` Backend process supervision
- Severity: high
- Complexity: large
- Status: upgrade
- Affected: llama.cpp/vLLM backend control
- Fix: owned process lifecycle, logs, restart, kill, health.
- Acceptance: backend crash/degraded states are bounded and visible.
- Validation: modelruntime integration tests
- Short pass: no

### `MR-003` Provider cost/egress governance
- Severity: medium
- Complexity: medium
- Status: gap
- Affected: provider config/modelruntime
- Fix: budgets and policy for remote providers.
- Acceptance: cloud model usage cannot exceed policy budget.
- Validation: modelruntime/API tests
- Short pass: yes

## Gateway / Tools

### `GATE-001` SSRF denial
- Severity: high
- Complexity: medium
- Status: bug
- Affected: gateway network tools, provider endpoints
- Fix: deny private/link-local/metadata/loopback unless explicitly allowed.
- Acceptance: deny table tests pass.
- Validation: `go test ./internal/gateway/...`
- Short pass: yes

### `GATE-002` Process output pre-buffer cap
- Severity: medium
- Complexity: medium
- Status: bug
- Affected: gateway process tool
- Fix: stream with bounded writers.
- Acceptance: huge output does not grow memory unbounded.
- Validation: gateway stress test
- Short pass: yes

## API / UI / Operator

### `UI-001` Accurate global runtime state
- Severity: high
- Complexity: medium
- Status: gap
- Affected: `AppShell.tsx`, `/health`, diagnostics
- Fix: display modelruntime/safe-mode/GPU/provider degradation.
- Acceptance: shell shows CPU-only/modelruntime unavailable accurately.
- Validation: desktop typecheck/build + UI tests when added.
- Short pass: yes

### `UI-002` Shared frontend contracts
- Severity: medium
- Complexity: medium
- Status: refactor
- Affected: `apps/desktop/src/lib/api.ts`, `packages/shared`
- Fix: move stable response types to shared.
- Acceptance: desktop imports shared contracts.
- Validation: `npm run typecheck`
- Short pass: yes

## Security

### `SEC-001` Loopback default bind
- Severity: high
- Complexity: small
- Status: bug
- Affected: `services/core/main.go`, config
- Fix: default bind to `127.0.0.1`; explicit config for public bind.
- Acceptance: tests prove default loopback.
- Validation: `go test ./internal/config/...`
- Short pass: yes

### `SEC-003` Symlink/path traversal suite
- Severity: high
- Complexity: medium
- Status: test
- Affected: gateway, artifacts, model import, backup
- Fix: add fixtures and resolved-path containment enforcement if needed.
- Acceptance: symlink escapes denied.
- Validation: targeted Go tests
- Short pass: yes

## Performance

### `PERF-001` Restore large-set benchmark
- Severity: medium
- Complexity: medium
- Status: test
- Affected: context restore scorer
- Fix: add benchmark and query budget.
- Acceptance: documented threshold and index use.
- Validation: Go benchmark/test
- Short pass: no

## Testing / Docs

### `DOC-001` Update stale review/status docs
- Severity: medium
- Complexity: small
- Status: docs
- Affected: `docs/reviews/forge_full_system_review_20260425`, status matrix
- Fix: mark completed passes and remaining gaps.
- Acceptance: docs no longer list Dream persistence/model governance/approval fingerprinting as missing.
- Validation: doc review
- Short pass: yes

### `TEST-002` Frontend tests and lint
- Severity: medium
- Complexity: medium
- Status: test
- Affected: desktop/packages
- Fix: add Vitest/RTL or documented alternative plus lint.
- Acceptance: root scripts include JS/TS test/lint lane.
- Validation: `npm test`, `npm run lint`
- Short pass: no

## Defer / Later

- Deep Nix/NixOS modules.
- LoRA/PEFT registry.
- GPU Dream jobs.
- Dream commit/apply mode.
- Full cockpit redesign before security/authority gaps close.
