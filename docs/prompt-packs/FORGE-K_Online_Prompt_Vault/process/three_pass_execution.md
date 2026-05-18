# Three-Pass Execution

Every agent run must follow this loop.

## Pass 1 — Understand and map

- Read current authority docs.
- Identify current live owner.
- Identify target FORGE-K component.
- List files likely to change.
- List tests likely to run.
- Identify rollback path.
- State what will not be touched.

## Pass 2 — Execute smallest safe change

- Implement only the requested phase.
- Keep changes narrow.
- Preserve live authority boundaries.
- Add/adjust tests with code changes.
- Update docs/status files.

## Pass 3 — Verify and report

- Run required tests.
- Record commands and results.
- Update phase report.
- Report blockers honestly.
- Do not claim unrun validation passed.

## Completion gate

If tests cannot run, document why and provide the narrowest substitute validation.
