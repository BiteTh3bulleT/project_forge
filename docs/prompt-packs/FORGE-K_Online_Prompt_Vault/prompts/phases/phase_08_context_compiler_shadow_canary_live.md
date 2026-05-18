# Context Compiler Shadow/Canary/Live

Append this prompt after `prompts/MASTER_PROMPT.md`.

## Goal

Generate ContextBundles from admitted evidence in shadow first, then canary one route only.

## Required work

1. Confirm current status from authority docs.
2. Identify live owner and target FORGE-K owner.
3. Make only the smallest change required by this phase.
4. Add or update tests.
5. Update docs/status files.
6. Produce a phase report.

## Validation

Run targeted validation for changed packages and the broad validation commands that apply. If a command cannot run, explain why and run the narrowest substitute.

## Deliverables

- Implementation or docs update for this phase
- Tests
- Updated status/report docs
- Final response with evidence

## WHAT NOT TO DO

- Do not expand into the next phase.
- Do not change unrelated live authority paths.
- Do not import simulator services into live daemon authority.
- Do not mark this phase complete without validation evidence.
