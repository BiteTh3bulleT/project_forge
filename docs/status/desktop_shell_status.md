# Desktop Shell Status (Phase 5.996)

Date: 2026-04-21
Scope: desktop/API runtime truth without UI redesign.

## Supersession Notice

This file is historical status context. Current operator desktop truth is recorded in:

- `docs/status/phase_g8_desktop_shell_verification.md`
- `docs/reports/phase_g8_desktop_shell_verification.md`
- `docs/operations/forge_graphical_shell_session.md`
- `docs/runbooks/forge_operator_desktop_vm.md`

As of the G8/CA2 alignment, the native shell path is:

```text
FORGE boot splash -> graphical OS login -> forge-operator session -> labwc -> forge-shell-session -> forge-desktop-shell
```

Use this file only for older desktop/API surface discovery. Do not use the "Still missing" section below as current validation truth; `npm run validate:desktop` is now the current desktop validation bundle and includes typecheck, Vitest, and build.

## Current desktop/API surfaces

Visible surfaces remain broad and wired to backend APIs:
- jobs/packets/approvals/artifacts
- gateway tools/capabilities/invocations
- autonomy status/intents/decisions/budgets/charters/events
- memory/retrieval/VSA views
- audit trace list
- backup bundles/restore
- release/readiness

## Mutation boundary reality

- Desktop mutates state through backend `/api/*` only.
- No client-side DB bypass path observed.
- Legacy backend mutation boundaries still exist, now with stricter guards:
  - legacy adapter invoke route removed (`/api/adapters/{id}/invoke` is not routed)
  - memory observation mutation endpoints: retired (`410 Gone`) with audit records; read-only inspection remains

## Trace/explain visibility

- Backend audit/correlation records are real and queryable.
- Legacy boundary audits now carry richer correlation/trace/workspace payload context.
- Dedicated end-to-end trace/explain desktop surface is still partial.

## Frontend validation surface

Configured scripts now include:
- root: `npm run build`, `npm run typecheck`
- root: `npm test`, `npm run lint` (delegate to core go test/vet + VSA preflight)
- desktop: `npm -w @forge/desktop run build`, `npm -w @forge/desktop run typecheck`

Current outcomes in this environment:
- desktop build: pass
- root typecheck: pass

Still missing:
- desktop `test`
- desktop `lint`
