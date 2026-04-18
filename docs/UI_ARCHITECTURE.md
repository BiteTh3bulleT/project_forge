# UI Architecture

## Overview

The desktop client is organized in three layers:

1. `Route pages`: Chat, Workbench, Canvas, Dossiers, Jobs, Reviews, Approvals, Settings, and advanced routes.
2. `Desktop shell`: top bar, dock, task strip, framed workspace surface, and context panel.
3. `Stores`: workspace health, UI mode/status, and shell session state.

## Key Files

- `apps/desktop/src/App.tsx`
  Route table and shell mounting.
- `apps/desktop/src/layout/AppShell.tsx`
  Desktop shell runtime, polling, layout, and context panel wiring.
- `apps/desktop/src/layout/shellConfig.ts`
  Tool registry and route-to-tool mapping.
- `apps/desktop/src/stores/desktopShellStore.ts`
  Persisted open/recent surface state.
- `apps/desktop/src/stores/workspaceStore.ts`
  Core connectivity and workspace metadata.
- `apps/desktop/src/stores/uiStore.ts`
  Command bar draft, status line, and UI mode.

## Data Sources

Shell indicators:

- `/api/dashboard`
- `/api/settings`
- `/api/adapters`
- `workspaceStore.ping()` via `/health` and `/api/meta`

Context panel route-specific lookups:

- chat threads
- canvas boards
- artifacts
- job detail
- dossier detail
- approvals queue
- review queue

## State Boundaries

### Persisted server truth

- chat records
- canvas records
- dossiers
- jobs, events, approvals, reviews
- artifacts
- settings

### Persisted client shell truth

- open route list
- recent route list
- UI mode preference

### Derived runtime state

- current top-bar counts
- active adapter readiness
- route-specific context panel content
- task strip active focus

## Why No Fake Window Manager

A floating-window system would add a second, fragile state machine on top of routing and page data.

FORGE currently uses a single active workspace surface because it gives:

- deterministic restoration
- clear focus
- less chrome
- fewer broken states
- better fit for the current route/page architecture

## Wiring Audit Standard

Visible shell controls must be one of:

- fully wired to real state
- clearly deferred in docs and UI copy
- removed

The desktop shell rework removed static help/quick-action chrome that implied richer behavior than the app actually had.
