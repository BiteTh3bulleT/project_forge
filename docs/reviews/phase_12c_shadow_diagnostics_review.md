# Phase 12C Shadow Diagnostics Review

Status: implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`.

## Implementation Summary

Phase 12C reviewed and hardened the Phase 12B read-only shadow harness. The live daemon still owns all live authority. FORGE-K shadow diagnostics remain disabled by default, diagnostic-only, in-memory only, and non-authoritative.

The only live touchpoint remains `/health` request metadata after the response is written. No new live touchpoints, routes, public diagnostics APIs, persistence, or authority paths were added.

## Current Touchpoint

The implemented observer is `services/core/internal/forgekshadow`.

Allowed metadata for the current `/health` touchpoint:

- workspace id
- optional request id
- method
- route
- touchpoint label
- short diagnostic summary

Forbidden data remains:

- request body
- response body
- prompts
- model output
- tool payloads
- retrieval result content
- memory content
- secrets or credential-like metadata

## No-Effect Guarantees

Phase 12C revalidated and hardened these guarantees:

- `FORGE_K_SHADOW_MODE_ENABLED` defaults to disabled.
- Invalid flag values do not enable shadow mode.
- Disabled mode creates no observer in normal server construction.
- Disabled mode stores no reports.
- Enabled mode does not change `/health` status, body, or content type.
- Enabled mode does not change route inventory.
- Shadow sink failures do not fail `/health`.
- Non-`/health` routes are not observed by the current harness.
- No public diagnostics route exists.

## Risk Review

| Risk | Phase 12C Result | Remaining Action |
|---|---|---|
| Shadow diagnostics alter live response | Response equivalence tests pass for disabled and enabled modes. | Keep response equivalence tests as a gate for any future touchpoint. |
| Diagnostics capture secrets | Metadata key/value rejection was expanded. | Revisit if future metadata sources are added. |
| Diagnostics capture raw content | Raw body/content/prompt keys and oversized strings are rejected. | Keep current `/health` metadata-only scope. |
| Sink growth is unbounded | Bounded retention drops oldest reports deterministically. | Add persistent retention policy only if persistence is separately approved. |
| Shadow failure affects live request | Best-effort observer and sink-failure tests preserve `/health` behavior. | Keep all future shadow work best-effort unless explicitly designed otherwise. |
| FORGE-K imports live authority packages | Forbidden import tests cover `forgek` and `forgekshadow`. | Keep tests updated if package layout changes. |

## Test Coverage Review

Existing and added tests cover:

- disabled observer stores no reports
- enabled observer stores diagnostic reports only
- disabled sink stores no reports
- bounded retention drops oldest reports
- unsafe metadata key/value rejection
- raw body/content/prompt metadata rejection
- oversized metadata rejection
- side-effect policy rejection
- sink failure isolation
- route inventory stability
- no public shadow diagnostics route
- `/health` response equivalence with shadow disabled and enabled
- non-`/health` route is not observed
- forbidden live daemon imports

## Hardening Actions Completed

- Added explicit disabled diagnostic sink support for internal tests and future rollback posture.
- Expanded unsafe metadata detection to include `authorization`, `cookie`, and `session`.
- Added raw content key rejection for request/response body, prompt, completion, model output, and content fields.
- Added maximum metadata string length enforcement.
- Added tests for all side-effect policy flags represented by the current shadow harness.
- Added route tests proving no public diagnostics endpoint and no non-`/health` observation.
- Added root FORGE-K forbidden live import test.

## Remaining Deferred Work

- No wider route observation exists.
- No public diagnostics API exists.
- No persistent diagnostic store exists.
- No operator-visible status surface exists.
- No live RAG, retrieval, embedding, modelruntime, gateway, memory, or controllane integration exists.
- Any expansion beyond `/health` requires a separate approved phase and fresh no-effect tests.

## Recommendation

Proceed next to Phase 12D only as a design/review checkpoint unless a separate prompt explicitly authorizes a bounded implementation. The recommended next action is to review Phase 12C diagnostics and decide whether to design a second read-only touchpoint or keep shadow mode limited to `/health` until more operator visibility requirements are defined.
