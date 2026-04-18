# Workspace Guide

## Layout

- **Header**: core connectivity, last error hint when offline, and Guided/Pro mode toggle.
- **Left navigation**: grouped by workflow (`Start`, `Daily Work`, `Project Memory`, `Control`, `System`), with guided blurbs in Guided mode.
- **Main column**: page content plus command bar with one-click quick actions and optional text command input.
- **Right rail** (xl+): page help, quick actions, status line, and live job queue snapshot (queued / running / awaiting approval). Rows are clickable.

## Mental model

1. **Start** is the guided launch path for setup, quick actions, and queue triage.
2. **Memory** is indexed evidence from configured **sources**.
3. **Jobs** transform memory and operator intent into **packets**, **adapter invocations**, and **artifacts**.
4. **Chat** is conversational memory *about* that work—not a second job runner. It can enqueue jobs explicitly.
5. **Workbench** is read-oriented inspection of **artifacts** and optional job telemetry.
6. **Canvas** is unstructured human state, not indexed unless you copy content into jobs or notes elsewhere.

## URL conventions

The app uses **hash-based routing** (React Router `HashRouter`). Examples:

- `http://localhost:5173/#/chat?threadId=1`
- `http://localhost:5173/#/workbench?artifactId=2&jobId=job_abc`

| Query | Page |
|-------|------|
| `?threadId=` | Chat (thread selection). |
| `?artifactId=` & optional `?jobId=` | Workbench. |
| `?boardId=` | Canvas. |
| `?jobId=` | Lineage (job focus). |

## Theming

The UI targets a dark, high-signal operator aesthetic (Tailwind tokens such as `forge-iron`, `forge-ash`, `forge-mist`). Avoid treating decorative copy as system state; operational text is labeled or tied to API-backed panels.
