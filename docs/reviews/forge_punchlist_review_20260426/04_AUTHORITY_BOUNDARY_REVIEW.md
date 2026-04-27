# Authority Boundary Review

## Scorecard

- Semantic syscall kernel: GOOD
- Gateway-only tool execution: GOOD
- Approval fingerprinting: GOOD/PARTIAL
- Model management governance: GOOD/PARTIAL
- Capability/lane/profile governance: RISK
- Remote ingress authority: RISK

## Findings

GOOD: Canonical semantic mutations use `aios/controllane` registry, validator, processor, transaction runner, journal/audit linkage, and SQLite store.

GOOD: Legacy direct adapter API route is not registered; adapter execution is gateway-wrapped.

GOOD: Gateway approval fingerprinting exists and tests reject mismatched replay.

RISK: Gateway capability status updates can reshape dangerous tool posture without a dedicated high-risk approval gate. `handleGatewayCapabilityStatusUpdate` requires reason for some changes, but `Gateway.UpdateCapabilityStatus` persists via a path that does not preserve that reason in the override table.

RISK: Lane and permission profile save/delete paths are authority-shaping but not consistently immutable-audit-backed.

RISK: `permissions.Check` can lift soft gates for a granted job approval. Gateway later checks approval fingerprints, but direct non-gateway callers could rely on the broader permission decision.

RISK: Telegram polling wake commands are allowlisted, but normal remote messages appear to lack sender/chat allowlist enforcement once remote polling is enabled.

PARTIAL: Dream Mode is proposal/dry-run-only and Dream reports are non-canonical evidence. No commit/apply path exists, which is correct for current doctrine.

## Exact Files / Symbols

- `services/core/internal/aios/controllane/processor.go`
- `services/core/internal/aios/controllane/validator.go`
- `services/core/internal/gateway/service.go`
- `services/core/internal/gateway/tool_capability_registry.go`
- `services/core/internal/api/phase5.go`
- `services/core/internal/permissions/service.go`
- `services/core/internal/api/telegram_gateway_service.go`
- `services/core/internal/api/remote.go`

## Punchlist

- `AUTH-001`: Route capability status changes through governance and persist actor/reason.
- `AUTH-002`: Add sender/chat allowlist enforcement for Telegram remote message processing.
- `AUTH-003`: Add immutable audit records for lane/profile save/delete.
- `AUTH-004`: Make approval fingerprint validation shared or prevent broad direct permission lifting.
- `AUTH-005`: Add regression test that legacy adapter invoke remains unrouted.

