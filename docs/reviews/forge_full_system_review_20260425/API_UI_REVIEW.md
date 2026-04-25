# API and UI Review

## API

GOOD: API route surface is broad and mostly coherent:

- health/meta/settings
- modelruntime and gated `/v1/*`
- chat and assistant stream
- gateway tools/capabilities/invocations
- context inspector snapshots
- process health trace
- audit trace
- dream run
- autonomy status/intents/decisions/budgets/charters/events
- retrieval/memory/VSA repair
- backup/restore

PARTIAL: Route surface is large and centralized in `server.go`; route ownership is understandable but dense.

RISK: Some mutating API routes are authority-adjacent but not approval-gated.

MISSING: Public syscall-native semantic write route.

## Desktop UI

GOOD: Desktop has real surfaces for chat, models, inspectors, tool gateway, audit, backup, memory, retrieval, project context, autonomy, and settings.

GOOD: Adapter UI uses gateway-mediated invoke in `api.ts`.

PARTIAL: Operator explainability is fragmented. Audit trace, process health, context snapshots, gateway invocations, and artifacts exist, but the "what happened and why" story still requires manual stitching.

MISSING: Dream Mode report execution/review UI is not obvious despite backend route.

MISSING: Frontend unit/component/e2e tests.

RISK: Large desktop bundle warning persists: main JS chunk is ~646 kB minified.

## UX Recommendations

RECOMMENDATION: Make Inspectors the central trace workbench: correlation id search should show chat message, gateway calls, syscall/journal, audit rows, artifacts, context snapshots, modelruntime calls, and Dream reports.

RECOMMENDATION: Add Dream Mode run/report page or a tab in Inspectors/Autonomy.

RECOMMENDATION: Make model management actions visibly governed, with approval state and audit trace.

RECOMMENDATION: Add frontend tests with Vitest/RTL plus one Playwright smoke for Models, Inspectors, Project Context, Gateway, and Dream report empty state.

