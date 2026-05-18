# Skill: Crucible Validator

Validate memory claims before promotion.

## Responsibilities

- validate ClaimEnvelope shape
- require provenance
- detect contradictions
- validate supersession chains
- compare current state
- emit decision states: rejected, needs_more_evidence, shadow_only, accepted_for_review, promotable

Crucible does not commit truth directly.
