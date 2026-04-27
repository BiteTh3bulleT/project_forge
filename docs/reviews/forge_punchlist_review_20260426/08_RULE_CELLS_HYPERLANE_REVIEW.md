# Rule Cells / Hyperlane Review

## Scorecard

- Rule agents: PARTIAL
- Ingest cells: PARTIAL/GOOD
- Rule Cell substrate: MISSING
- Hyperlane router: MISSING
- No-mutation behavior: GOOD for existing rule agents

## Findings

GOOD: Current autonomy rule agents are propose-only and have destructive guards.

GOOD: Librarian ingest cells are real and generate proposals routed through control-lane authority.

PARTIAL: Existing rule agents are not the planned Rule Cell/Hyperlane substrate.

MISSING: There is no deterministic rule registry with lane/phase filtering, latency budgets, trace envelopes, disabled-rule handling, or starter rule packs.

MISSING: Hyperlane remains a journal/design concept, not runtime code.

## Punchlist

- `RULE-001`: Either implement Rule Cell/Hyperlane v0 or keep all docs/status clearly concept-only.
- `RULE-002`: Add deterministic rule registry with priority, enabled/disabled state, lane/phase filters, and trace output.
- `RULE-003`: Add starter rule pack for restore hygiene, Dream review, gateway safety, and operator hints.
- `RULE-004`: Add latency budget tests and no-mutation tests.
- `RULE-005`: Consider folding current autonomy rule agents into the future Rule Cell substrate.

