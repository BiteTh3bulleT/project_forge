# Phase Exit Gates

Every phase must answer:

1. What changed?
2. What did not change?
3. What authority boundaries were preserved?
4. What tests were added?
5. What tests were run?
6. What risks remain?
7. Is rollback obvious?
8. Did any live behavior change?

## Absolute blockers

Stop if:

- worker can mutate canonical state
- FORGE-HMK can bypass Control Lane
- cache/VSA/vector output becomes truth
- shadow mode changes user-visible behavior
- provenance is missing from promotable claims
- no-effect tests fail
