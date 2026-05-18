# ADR 0001: FORGE-K Online Cutover Doctrine

Status: Proposed

## Context

FORGE-K must become live authority gradually without creating a second authority path.

## Decision

Every online migration follows the ladder:

`SIMULATOR_ONLY -> SHADOW_READ_ONLY -> VALIDATION_ONLY -> DISABLED_BY_DEFAULT_LIVE -> OPERATOR_APPROVED_LIVE -> DEFAULT_LIVE -> LEGACY_PATH_RETIRED`.

## Consequences

No simulator service becomes live authority without explicit design, tests, rollback, and docs.
