# Rule Cell Operations

Status date: 2026-04-27.

Rule Cells are enabled through the static v0 engine in core server construction. They are deterministic and CPU-only.

## Operator Expectations

- Rule Cells are not agents and do not run autonomous jobs.
- They do not call LLMs, modelruntime, gateway tools, the network, or GPU providers.
- They cannot commit canonical truth.
- They may add warnings, stricter policy outputs, score adjustments, or routing hints.
- Existing kernel, gateway, approval, capability, scope, and modelruntime denials always win.

## Where To Look

Code:

- `services/core/internal/aios/rulecells`
- restore integration: `services/core/internal/aios/controllane/compile_context_restore_scoring.go`
- Dream integration: `services/core/internal/aios/dream/service.go`

Docs:

- `docs/architecture/rule_cells.md`
- `docs/architecture/hyperlane.md`
- `docs/architecture/context_restore_scoring.md`
- `docs/architecture/dream_mode.md`

## Trace Reading

Restore scoring persists compact Rule Cell trace data inside existing non-canonical restore metadata. Dream Mode includes compact Rule Cell traces in `DreamReport.Trace`.

Inspect:

- `rule_packs`: pack id/version that explain the rule surface
- `matched_rules`: rule ids and output types
- `outputs`: structured decisions or score deltas
- `warnings`: latency, authority conflict, or engine failure warnings

No per-rule database trace rows are created in v0.

## Failure Handling

If Rule Cells fail during restore scoring, `COMPILE_CONTEXT` emits a warning and continues deterministic base scoring.

If Rule Cells fail during Dream Mode, dry-run report generation emits a warning and continues deterministic base salience/routing.

No silent fallback is allowed.

## Safe Rule Additions

When adding a rule:

1. Add it to a static pack with a pack id/version already present or intentionally updated.
2. Keep the condition deterministic and local.
3. Emit one structured output.
4. Bound any score adjustment.
5. Add tests for match, trace, pack version, and failure behavior.
6. Prove it cannot loosen an authoritative denial if the rule touches policy.

Do not add file-loaded scripts, eval, LLM judgment, network calls, modelruntime calls, unbounded DB scans, or side-effect callbacks.
