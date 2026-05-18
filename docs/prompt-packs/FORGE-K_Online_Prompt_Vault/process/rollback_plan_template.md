# Rollback Plan Template

## Phase

`<phase id>`

## Change summary

`<what changed>`

## Disable flag

`<config/feature flag>`

## Revert path

`<commit revert / config disable / migration rollback>`

## Data left behind

`<audit rows, diagnostic rows, shadow reports, generated files>`

## Tests proving rollback

- [ ] disabled-mode test
- [ ] old behavior still works
- [ ] no route/API regression
- [ ] no orphan authority path remains

## Operator instructions

`<exact commands or UI path>`
