# Stubs / Placeholders / Fake / Example Inventory (Runtime-Adjacent)

Updated: 2026-04-21 (Worker B scope: this file only)

Method:
- Marker sweep across `services/core/internal`, `apps/desktop`, `packages`, `docs`, `nix` for: `TODO`, `FIXME`, `stub`, `placeholder`, `fake`, `noop/no-op`, `dummy`, `mock`, `not implemented`, `unsupported`, `candidate-note`, `candidate-*`, `fakeHash`, `example`, `sample`, `in-memory only`, `disabled capability`, `approval-only capability`, `scaffold`, `future work`, `deferred`.
- This inventory is strict about runtime-adjacent behavior; UI input placeholders and ordinary SQL placeholder helper names are excluded as non-signal.

## Runtime-adjacent findings

| File path | Symbol / section | Context summary | Category | Impact | Recommended action |
|---|---|---|---|---|---|
| `services/core/internal/aios/autonomy/runner.go` | `validateAutonomyCommitAction`, `hasPlaceholderArchiveTarget` | Explicit guard blocks archive targets matching `candidate-note`, `candidate-*`, `fake-*`, `placeholder-*` before commit. | dangerous placeholder | high | test-guard |
| `services/core/internal/aios/compute/librarian/cells_phase4.go` | `CleanupRuntimeCell.Run` | Reads `archiveNoteIds` + `archiveReason` from ingest metadata and proposes `ARCHIVE_NOTE` actions without local placeholder-ID filtering; relies on downstream guard. | production placeholder | high | harden |
| `services/core/internal/aios/autonomy/rule_agents.go` | `CleanupProposalAgent.Evaluate` | Cleanup agent intentionally emits no destructive action and returns warning about missing deterministic target. | runtime scaffold | medium | defer |
| `services/core/internal/aios/autonomy/repositories.go` | `InMemory*Repository` and `NewInMemoryBundle` | Charters/intents/budgets/decisions/reservations/curiosity all have in-memory repository implementations in live package. | in-memory critical system | high | harden |
| `services/core/internal/aios/autonomy/sqlite_repositories.go` | `NewSQLiteBundle` | Nil DB falls back to `NewInMemoryBundle()` rather than fail-closed. | in-memory critical system | high | harden |
| `services/core/internal/aios/autonomy/policy_evaluator.go` | `hasDurableSelfCommitBacking`, maintain/mission gate | Self-commit is quarantined when charter/budget backing is in-memory. This is a safety guard, not a bypass. | runtime scaffold | low | leave |
| `services/core/internal/aios/controllane/processor.go` | `NewProcessor` | Default processor path creates `InMemorySemanticStore` + `InMemoryTransactionRunner` when no tx runner provided. | in-memory critical system | high | harden |
| `services/core/internal/aios/controllane/store.go` | `InMemorySemanticStore`, `InMemoryTransactionRunner` | Full semantic object/state mutation path exists in-memory variant in live package. | in-memory critical system | high | document |
| `services/core/internal/aios/controllane/processor_apply.go` | `applyCompileContext` | `COMPILE_CONTEXT` returns warning `compile_context is deterministic Phase 2 stub`; no durable compile artifact path here. | runtime scaffold | medium | defer |
| `services/core/internal/aios/controllane/store.go` | `BuildContext` | Context packet marks inclusion reason `mode: deterministic_stub`. | runtime scaffold | medium | defer |
| `services/core/internal/aios/compute/librarian/inference.go` | `NoopSemanticInference` | Semantic inference adapter defaults to no-op (`nil` candidates/suggestions/model). | runtime scaffold | medium | document |
| `services/core/internal/aios/compute/librarian/cells.go` | `StaticIntakeCell` | Deterministic stub cell with IDs like `stub-intake-create-note` and actor `librarian.intake.stub`; intended for scaffold smoke behavior. | fake/example code in live path | medium | document |
| `services/core/internal/gateway/tool_capability_registry.go` | `defaultCapabilityDescriptor` | Full taxonomy preserved; default status is `stubbed` and default adapter id is synthetic `stub.<capability>`. | stubbed capability | medium | leave |
| `services/core/internal/gateway/tool_policy.go` | `ToolPolicyEvaluator.Evaluate` | Enforces `deferred`/`stubbed` => `unsupported`, `disabled/deprecated` => `disabled`; blocks unknown status and missing adapters. | stubbed capability | low | leave |
| `services/core/internal/api/phase5.go` | `handleGatewayCapabilityStatusUpdate` | Status transitions to `deferred/disabled/stubbed/deprecated` require explicit reason text. | runtime scaffold | low | leave |
| `services/core/internal/api/server.go` | legacy adapter invoke route wiring | Legacy direct adapter invoke side door has been removed from routing; gateway-only execution path remains. | resolved hardening | low | keep |
| `services/core/internal/gateway/service.go` | denied-path audit mapping | Terminal policy outcomes (`needs_approval`, `unsupported`, `disabled`) emit explicit audit actions (`tool.needs_approval`, `tool.unsupported`, `tool.disabled`). | runtime scaffold | low | leave |
| `services/core/internal/aios/autonomy/charter.go` | `evaluateCondition` | Charter condition evaluator supports small expression subset; unknown expressions return `unsupported conditional expression`. | runtime scaffold | medium | document |
| `services/core/internal/aios/README.md` | package scope note | Declares lane scaffolds and in-memory transaction behavior; documents that old modules remain authoritative until cutover. | docs-only future work | low | document |

## Harmless test stubs / examples

| File path | Symbol / section | Context summary | Category | Impact | Recommended action |
|---|---|---|---|---|---|
| `services/core/internal/aios/compute/librarian/pipeline_phase4_test.go` | `fakeSemanticInference`, `runtimeCellStub` | Test doubles for deterministic pipeline behavior and no-op inference assertions. | harmless test stub | none | leave |
| `services/core/internal/gateway/tool_surface_test.go` | `stubAutonomyAuthorizer` | Test double for policy/autonomy path behavior. | harmless test stub | none | leave |
| `services/core/internal/aios/computelane/interfaces_test.go` | `stubInference`, `stubCell`, `stubCompiler`, `stubIrisBridge` | Compile-time interface conformance stubs, including IRIS bridge seam tests. | harmless test stub | none | leave |

## Docs-only future work / scaffold placeholders

| File path | Symbol / section | Context summary | Category | Impact | Recommended action |
|---|---|---|---|---|---|
| `nix/tool-capsules/README.md` | whole file | Explicit placeholder: no active capsules, no flake exposure, future-phase plan only. | docs-only future work | low | defer |
| `nix/modules/README.md` | whole file | Explicit placeholder: no real NixOS modules in Phase N1. | docs-only future work | low | defer |
| `nix/profiles/README.md` | whole file | Explicit placeholder: no real Nix profiles, future composition only. | docs-only future work | low | defer |
| `docs/status/desktop_nix_packaging_gap.md` | hash capture steps | Uses `lib.fakeHash` in documented future derivation bootstrap flow; clearly marked deferred status. | harmless docs example | low | document |

## Marker-sweep notes

- `TODO` / `FIXME` / `panic("TODO")` were not found in non-test runtime files under `services/core/internal`.
- `candidate-*`/`fake-*` placeholder IDs were found in runtime only as explicit blocklist guards in autonomy commit validation.
- `fakeHash` appears in documentation/planning context, not as an active core flake hash for `forge-core`.

## Immediate hardening priorities from this inventory

1. Harden `CleanupRuntimeCell.Run` pre-commit filtering for placeholder/fake archive targets so unsafe candidates are dropped before syscall proposal.
2. Keep direct adapter invoke route removed; do not reintroduce a non-gateway adapter execution ingress.
3. Fail closed for autonomy/control-lane durable paths in production wiring when DB/transaction backing is absent, rather than silently falling back to in-memory repositories.
