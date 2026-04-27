# Dream Mode v0

Status date: 2026-04-24.

Dream Mode v0 is FORGE's first deterministic replay/consolidation engine. It is CPU-first, dry-run by default, and proposal-only.

Memory taxonomy reference: [memory_taxonomy.md](memory_taxonomy.md).

## Authority

- Dream Mode reads existing cognitive filesystem tables.
- It does not create a second memory database.
- It does not call LLMs, modelruntime, vector retrieval, GPU jobs, adapters, voice, vision, or GUI systems.
- It does not commit canonical truth in v0.
- Any future commit mode must use semantic syscalls/control lane validation.

## Memory Taxonomy Role

Dream Mode is the lymphatic replay surface for mid-term reflective, salience, and utility memory.

It reads across the taxonomy:

- working memory from context snapshots and recent restore packages
- episodic memory from journal, observation, artifact, and outcome evidence
- salience memory from contradictions, active blocked loops, failed restores, and corrections
- prospective memory from open loops and next-action evidence
- reflective memory from repair/review candidates and prior dry-run reports where available
- utility memory from restore outcomes and retrieval usefulness signals
- semantic/procedural/structural memory from notes, state, links, derived models, artifacts, and snapshots

Dream Mode may propose promotion, demotion, repair, or review. It must not silently mutate canonical semantic, procedural, or structural memory. Long-term promotion remains a future governed syscall path.

## Replay Selector

The selector gathers recent candidates from:

- `journal_events`
- `context_packet_snapshots`
- `restore_outcome_events`
- `memory_notes`
- `state_items`
- `open_loops`
- `contradiction_records`
- `artifact_refs`

Candidates include source IDs, scope, timestamp range, content summary, tags, related loop/snapshot IDs, raw importance signals, and trace fields.

## Salience

Salience is deterministic:

- novelty score
- repetition score
- goal relevance score
- correction value score
- outcome impact score
- contradiction score
- retrieval utility score
- recency score

User corrections, unresolved contradictions, active blockers, repeated failures, failed restores, and recent important events receive visible score components and explanation strings.

Restore outcome feedback is first-class replay evidence:

- `operator_corrected` outcomes receive very high salience and route to review.
- `harmful`, `stale`, `contradictory`, `not_helpful`, and failed execution outcomes route to restore evidence review.
- repeated `fresh_compile_required` or `no_candidate` outcomes become memory-gap proposals.
- repeated helpful outcomes become promotion candidates for future governed consolidation.

Dream Mode still does not apply these proposals. It only reports them as non-canonical evidence for an operator or future governed syscall path.

## Rule Cell Adjustments

Phase 7 v0 adds Lymphatic Rule Cells after candidate selection and base salience scoring.

Rule Cells may:

- boost user corrections
- boost unresolved contradictions
- boost active blocked loops
- boost repeated failures
- block long-term promotion when unresolved contradiction risk is present

Rule Cells remain advisory or stricter only. They cannot commit memory, promote truth directly, call modelruntime, or bypass semantic syscalls.

Salience safety:

- individual Dream salience adjustment is capped at `0.08`
- total rule-based Dream salience adjustment is capped at `0.15`
- final salience score is clamped to `0.0..1.0`

If the rule engine fails, Dream Mode emits an explicit warning and continues deterministic base dry-run report generation. Rule traces in `DreamReport.Trace` include pack id/version and matched rule ids.

## Tier Routing

Dream Mode v0 proposes one of:

- `retain_short_term`
- `promote_mid_term`
- `promote_long_term`
- `demote`
- `merge`
- `discard`
- `needs_review`
- `repair_required`
- `no_op`

Long-term promotion requires high salience, high confidence, low contradiction risk, explicit long-term allowance, and no operator-review requirement. Unresolved contradictions route to review or repair rather than long-term truth.

## Operating Depths

- `microdream`: short window, low candidate limit, corrections/contradictions/active loops/failed restores
- `nap`: day-scale window, mid-term promotion and snapshot hygiene proposals
- `deep_dream`: longer window and larger candidate set, long-term candidates and repair proposals, still CPU-only in v0

## Report

The dry-run report returns run metadata, candidates, salience scores, tier routing proposals, memory action proposals, snapshot hygiene proposals, restore score update proposals, restore outcome candidates, memory gap proposals, stale/harmful evidence review proposals, helpful evidence promotion proposals, repair proposals, review items, no-op reasons, warnings, and trace.

Reports are non-canonical evidence. They are not memory truth.
