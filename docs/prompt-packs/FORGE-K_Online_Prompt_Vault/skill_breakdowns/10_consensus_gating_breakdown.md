# Breakdown: Consensus Gating

## Checklist

- [ ] Read the current phase prompt.
- [ ] Locate live owner.
- [ ] Locate target FORGE-K doctrine.
- [ ] Identify current tests.
- [ ] Identify required new tests.
- [ ] Make smallest safe change.
- [ ] Run validation.
- [ ] Update docs/status.
- [ ] Produce evidence report.

## Completion standard

This skill is complete only when the change is narrow, tested, documented, and rollback-aware.

## Reviewer questions

1. Did this change create a second authority path?
2. Did it import simulator services into live authority?
3. Did it change route/API behavior unexpectedly?
4. Did it preserve gateway/modelruntime/NixOS boundaries?
5. Is rollback obvious and tested?
