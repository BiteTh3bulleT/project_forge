# Full Audit Reviewer Prompt

Review the current branch for FORGE-K online readiness.

## Review categories

- authority boundaries
- simulator/live separation
- gateway/tool execution safety
- modelruntime proposal boundary
- NixOS host substrate safety
- journal/replay readiness
- Courthouse/Memory Palace readiness
- Context Compiler readiness
- Consensus gating readiness
- test coverage
- docs/status truth

## Output

- P0 blockers
- P1 hardening items
- P2 cleanup/refactor items
- explicit go/no-go verdict

## WHAT NOT TO DO

Do not suggest wiring simulator services directly into live authority.
