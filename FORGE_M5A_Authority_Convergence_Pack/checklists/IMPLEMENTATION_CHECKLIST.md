# M5A Implementation Checklist

## Phase 0
- [ ] Read mandatory files.
- [ ] Create/update `docs/reviews/m5a_authority_convergence_review.md`.

## Phase 1
- [ ] Implement authority matrix.
- [ ] Add tests for required route coverage.
- [ ] Add no-FORGE-K-live-authority assertions.

## Phase 2
- [ ] Fix `model.delete_file` registry drift.
- [ ] Decide chat/generate policy posture.
- [ ] Add modelruntime/gateway alignment tests.
- [ ] Update docs.

## Phase 3
- [ ] Add Control Lane approval fingerprint doc.
- [ ] Add pure fingerprint helper if clean.
- [ ] Add deterministic tests.

## Phase 4
- [ ] Add HostBridge/FORGE-H cache or TTL wrapper.
- [ ] Add stale/error/no-mutation tests.
- [ ] Expose age/stale fields.

## Phase 5
- [ ] Extend System status/cockpit read-only display.
- [ ] Add UI tests if frontend changes.
- [ ] Verify no mutation buttons.

## Phase 6
- [ ] Add latency baseline doc.
- [ ] Run validation commands.
- [ ] Record pass/fail honestly.

## Phase 7
- [ ] Add micro-agent acceleration architecture doc.
- [ ] Verify all micro-agents are proposal/cache/advisory only.

## Final
- [ ] No hidden authority expansion.
- [ ] No direct host mutation.
- [ ] No raw sensitive data in status.
- [ ] Tests pass or blockers documented.
- [ ] Final summary prepared.
