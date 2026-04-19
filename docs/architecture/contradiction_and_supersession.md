# Contradiction And Supersession (Phase 5)

FORGE treats contradiction and supersession as different concepts.

- Contradiction: conflict/tension between claims; both remain preserved.
- Supersession: a newer or more authoritative record replaces another for current-truth projection.

## Why both exist

- Contradiction preserves uncertainty and opposing evidence.
- Supersession preserves lineage and successor resolution.
- Neither path hard-deletes canonical evidence.

## Contradiction rules

Implemented via `REGISTER_CONTRADICTION`.

- Stores left/right object ids and kinds, reason, severity, confidence, scope, provenance, trace fields.
- Both referenced objects remain queryable.
- Does not auto-decide current truth by itself.
- May influence truth explanations and retrieval warnings.

Deterministic severity heuristics in truth engine:

- high:
  - explicit reversal language (`instead of`, `replace`, `not anymore`, `wrong`)
  - state_item vs state_item conflict
- medium:
  - mismatch/tension language
- low:
  - weak or ambiguous mismatch

## Supersession rules

Implemented via `MARK_SUPERSEDED`.

- Stores old/new object ids and kinds, reason, scope, provenance, trace fields.
- Old record is retained.
- Current-successor chain is queryable.
- Cycle attempts are rejected.
- `old == new` is rejected.
- Cross-scope or incompatible known kind supersession is rejected by validation.

## Lifecycle effects

- Note statuses:
  - `active -> superseded -> archived` (archived terminal)
- Model statuses:
  - `provisional -> promoted -> deprecated` (deprecated terminal)
- Supersession can mark note status as superseded without deleting evidence.
- Deprecating models never removes derived evidence references.

## Current-object resolution

`truth.Engine.Resolve` considers:

- scope
- archive status
- supersession chain
- contradiction presence
- model deprecation
- loop terminal states (`resolved`, `archived`)

Resolver output includes:

- is current
- current successor id
- superseded/archived/deprecated flags
- contradiction warning
- include-in-active retrieval flag

## Examples

1. User changes preference:
   - old note: "prefer transcript replay"
   - new note: "prefer structured snapshots instead"
   - outcome: supersession proposal, optional contradiction record

2. Project decision replaced:
   - decision A superseded by decision B
   - A remains queryable for audit/history
   - current resolution points to B

3. Blocker resolved:
   - open loop transitions to resolved
   - optional archive transition later
   - loop remains inspectable historically

4. Model deprecated:
   - derived model transitions to deprecated
   - supporting note/state evidence remains durable

## Query surfaces

Truth engine methods:

- `ListContradictionsForObject`
- `ListContradictionsByScope`
- `ExplainContradiction`
- `GetSuccessor`
- `GetSupersessionChain`
- `ExplainSupersession`
- `Resolve` / `ExplainResolution`
