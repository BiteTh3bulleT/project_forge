# FORGE Mutation Loop Parking Lot

Status: parked idea / future design input.

This note preserves the future mutation-loop idea without making it active scope.

Authority note: current authority sources live in `docs/status/current_authority_sources.md`, with current phase truth in `docs/reviews/current_phase_status.md`. This parking-lot note is not an active roadmap, does not authorize self-mutation, and does not make FORGE-K simulator services live daemon authority.

## Concept

FORGE may eventually generate code or policy mutation candidates while running, but candidates must be isolated, tested repeatedly, and promoted through governed gates before they become permanent.

Proposed promotion chain:

`MutationCandidate -> FitnessReport -> KBitUnlockProposal -> OperatorApproval -> JournaledPromotion -> CanaryActivation -> PermanentAdmission`

## Hard Constraints

- Mutation candidates must be created in isolated branches/worktrees, not directly in the running live tree.
- A candidate starts from a measurable failed criterion, failing test, blocked readiness gate, or explicit operator request.
- Tests make a candidate eligible for admission; tests do not directly make it permanent.
- Permanent activation requires operator approval, journaled promotion, scoped canary activation, and rollback evidence.
- K-bits are typed and scoped activation markers, not global permission switches.
- No mutation loop may bypass semantic syscalls, Courthouse admission, Kernel commit boundaries, approval gates, audit, or provenance capture.

## Candidate K-Bits

- `K.COURT.ADMISSION_MIRROR_READY`
- `K.PALACE.RETRIEVAL_PROVENANCE_READY`
- `K.CONTEXT.PARITY_STABLE`
- `K.SYSCALL.MUTATION_ROUTE_ELIGIBLE`
- `K.RUNTIME.DRIVER_TRACE_READY`

## Deferred Design Questions

- Should the first loop target code changes, runtime policy/config changes, or activation-gate readiness only?
- Which validations are mandatory before canary activation?
- Which scopes are eligible for canary activation: workspace, lane, action type, or dry-run only?
- What evidence closes the canary window and allows permanent admission?
