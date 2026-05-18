# Refactor Control

## Rule

Refactor only when it directly supports the current phase.

## Allowed

- mechanical extraction with tests
- moving pure validators into shared packages
- splitting oversized files only when behavior stays identical
- renaming for clarity with tests and references updated

## Forbidden

- opportunistic rewrite of unrelated systems
- changing route/API shape without explicit phase instruction
- replacing live owners during validation-only phases
- mixing storage cutover with semantic authority migration
- broad UI redesign during backend authority migration

## Required proof

Behavior must be identical unless the phase explicitly changes it.
