# Placeholders and Stubs Inventory (Phase 5.99)

Date: 2026-04-20
Scope: high-signal placeholder/stub posture after minimal hardening

Classification labels used:
- harmless test stub
- intentional disabled capability
- safe scaffold
- production risk
- dangerous placeholder
- missing implementation
- documentation-only future work

## High-signal findings

| Location | Finding | Classification | Reachable? | Current posture |
|---|---|---|---|---|
| `services/core/internal/aios/autonomy/rule_agents.go` + `runner.go` | `CleanupProposalAgent` previously emitted `candidate-note` placeholder payload | dangerous placeholder | proposal path no (now disabled), commit path guarded | `CleanupProposalAgent` now emits no destructive actions by default; runner still blocks placeholder targets for defense-in-depth. |
| `services/core/internal/api/server.go` memory mutation handlers | Observation mutation endpoints previously bypassed syscall semantics if opened | resolved | no | Mutation endpoints are retired (`410 Gone`) and retired attempts are audited; there is no env opt-in path. |
| `services/core/internal/aios/*` lane layout | `compute/librarian` runtime coexists with `computelane` scaffold/doc surfaces | safe scaffold | yes | Explicitly documented as partial and non-duplicative target for cutover. |
| `services/core/internal/gateway/tool_capability_registry.go` | Full taxonomy with default gateway-backed capability rows | resolved | yes | Default capabilities are `active` or `approval_only` with concrete `gatewayToolId`; explicit `stubbed`/`deferred` remains only as override/test status semantics. |
| `nix/tool-capsules/README.md`, `nix/modules/README.md`, `nix/profiles/README.md` | README-only placeholders | documentation-only future work | no | Deferred until authoritative runtime paths are stable. |
| `docs/status/desktop_nix_packaging_gap.md` | Desktop package capture still deferred | documentation-only future work | no | Kept deferred; not marked complete. |
| `docs/architecture/nix_substrate.md` | Earlier fake-hash wording drift for core package | production risk (docs drift) | docs only | Marked for alignment: core package currently has real `vendorHash`. |

## Mandatory-actions status

1. Dangerous placeholders reachable in default destructive flows: `satisfied` (guarded/blocked).
2. Cleanup/archive placeholders disabled, guarded, or fixed: `satisfied` (guarded in autonomy runner).
3. Fake object IDs cannot produce destructive commit actions: `satisfied`.
4. Stubbed dangerous capabilities remain disabled/approval-only/stubbed: `mostly satisfied`.
5. In-memory-only critical subsystems clearly marked and blocked from serious autonomy: `satisfied`.
6. `fakeHash` posture documented honestly: `satisfied for core`, `deferred for desktop package work`.

## Remaining placeholder risks

1. Proposal-time placeholder payloads have been removed from cleanup agent outputs.
2. Some architecture docs still describe scaffold lanes with more certainty than code reality.
3. Deferred Nix module/capsule/profile placeholders remain intentionally non-runtime.
