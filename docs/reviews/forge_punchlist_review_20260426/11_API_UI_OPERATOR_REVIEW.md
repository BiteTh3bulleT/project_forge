# API / UI / Operator Review

## API Scorecard

- Health routes: GOOD/PARTIAL
- Restore inspector routes: GOOD
- Dream inspector routes: GOOD/PARTIAL
- Process routes: PARTIAL
- Websocket: MISSING / not found
- SSE chat stream: GOOD/PARTIAL

## UI Scorecard

- Operator inspectability: PARTIAL
- Empty states: PARTIAL/GOOD
- Degraded modelruntime/no-GPU visibility: PARTIAL
- Dual-mode direction: PARTIAL
- Frontend contract centralization: PARTIAL

## Findings

GOOD: Restore and Dream inspector APIs exist and expose non-canonical evidence.

GOOD: Desktop has a compact Inspectors page with snapshots, Dream reports, packet inspection, trace, and process trace.

RISK: App shell runtime pill is not modelruntime/safe-mode/GPU-aware and can imply "online" while subsystems are degraded.

PARTIAL: `/api/process/health` is trace-scoped process health, not global process health.

PARTIAL: Dream list/get and Dream subresource responses use different envelope styles.

PARTIAL: Model lifecycle UI does not clearly show governance state: direct, approval required, approval pending, dry-run, blocked, unavailable.

PARTIAL: Many desktop API contracts live locally in `apps/desktop/src/lib/api.ts` instead of `packages/shared`.

MISSING: No WebSocket implementation found; chat streaming appears SSE-only.

## Punchlist

- `UI-001`: Make shell runtime state reflect `/health`, modelruntime, safe mode, GPU, and embedding provider state.
- `UI-002`: Add global diagnostics/operator state page or endpoint.
- `UI-003`: Rename UI language around process trace health unless a global process endpoint is added.
- `UI-004`: Pass workspace/lane into generic snapshot detail calls.
- `UI-005`: Standardize evidence badges across Dream/restore views.
- `UI-006`: Surface model action governance state in Models UI.
- `UI-007`: Move stable API contracts into `packages/shared`.

