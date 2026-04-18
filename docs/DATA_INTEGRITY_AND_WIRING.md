# Data Integrity and Wiring

This document records how visible UI state maps to persisted or derived data after the workspace expansion pass.

## Persisted (SQLite / disk)

| Domain | Storage |
|--------|---------|
| Chat | `chat_threads`, `chat_messages` |
| Canvas | `canvas_boards`, `canvas_notes` |
| Artifacts | `artifacts` table + files under data dir `artifacts/` subtree |
| Jobs | `jobs`, `task_packets`, job events |
| Approvals | approval request tables |
| Reviews | review records |
| Dossiers | dossier tables |
| Settings | `settings` key-value |

## Derived / ephemeral

- **Dashboard** aggregates are computed server-side per request (`/api/dashboard`).
- **Live Activity** rail polls `GET /api/jobs` with status filters; not a separate WebSocket stream.
- **Command bar “navigate”** command is explicitly client-side (core returns a note in `commands/execute`).

## UI honesty rules applied

- **No demo cards** were added for Chat, Workbench, or Canvas; empty states explain what persistence exists.
- **Routing recommendation reasons** on the dashboard tolerate non-array API shapes by normalizing to display text (avoids fake-looking blank reasons or runtime throws).
- **Job detail → Lineage** passes `?jobId=`; Lineage bootstraps from URL before defaulting to the first recent job.
- **Job detail artifacts** link to Workbench with `artifactId` and `jobId` query params.
- **Right rail jobs** are buttons to job detail (previously inert-looking rows).

## Dev-only / sample data

FORGE does not ship a hidden production demo dataset for chat or canvas. Any seed data used in tests should remain in test fixtures only.

## Stale state and refresh

- Chat and Canvas pages reload entities after mutations.
- Workbench refetches the artifact list when `jobId` query changes.
- Job detail page polls on an interval for long-running jobs.

## When the core is offline

Most pages surface fetch errors. The workspace store marks core offline; Chat/Workbench/Canvas will show error text from failed `fetch` calls rather than silent empty shells.
