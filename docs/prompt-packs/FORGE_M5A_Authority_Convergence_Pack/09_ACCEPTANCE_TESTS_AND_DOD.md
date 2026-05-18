# M5A Acceptance Tests and Definition of Done

## Must-pass implementation checklist

- [ ] `docs/reviews/m5a_authority_convergence_review.md` exists.
- [ ] Authority matrix exists and is tested.
- [ ] `model.delete_file` metadata matches runtime reality.
- [ ] `model.chat` posture is explicit and tested.
- [ ] `model.generate` posture is explicit and tested.
- [ ] Control Lane approval fingerprint seam exists or full enforcement is implemented.
- [ ] Fingerprint tests prove determinism.
- [ ] HostBridge/FORGE-H snapshot cache or TTL wrapper exists.
- [ ] Snapshot tests cover stale/error/no-mutation behavior.
- [ ] System status/cockpit exposes read-only authority summary.
- [ ] System page does not add mutation controls.
- [ ] Micro-agent acceleration design exists.
- [ ] Latency baseline doc exists.
- [ ] Docs updated to current truth.
- [ ] Validation commands recorded.

## Suggested validation commands

```bash
npm test
npm run lint
npm run validate:js
npm run validate:local
npm run build
npm run smoke
```

## Done means

FORGE can answer:

> Which subsystem owns this action, can it mutate anything, does it require approval, and where is the audit trail?

And the answer is the same in code, docs, route behavior, Gateway capability metadata, System Cockpit, and tests.
