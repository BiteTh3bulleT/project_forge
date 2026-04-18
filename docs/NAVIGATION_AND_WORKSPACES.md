# Navigation And Workspaces

## Opening Tools

Open a primary tool from the dock.

What happens:

1. the route becomes active
2. the route is added to the open-session list if not already open
3. the route moves to the front of recent surfaces
4. the task strip updates immediately

## Switching Context

Use the task strip to switch active surfaces.

The task strip is the workspace switcher for FORGE. It is intentionally simpler than floating windows and stronger than a plain sidebar router.

## Closing A Surface

Closing a task-strip item removes it from the shell session list.

This does not delete:

- threads
n- boards
- jobs
- dossiers
- artifacts
- reviews
- approvals

It only closes the current shell surface.

## Route-Carried Selection

The shell inspector depends on real selection state being in the route when needed.

Current route-carried selections:

- `#/chat?threadId=...`
- `#/workbench?jobId=...&artifactId=...`
- `#/canvas?boardId=...`
- `#/dossiers?dossierId=...`
- `#/jobs/:id`

This allows reload, deep link, and session restore without phantom local shell memory.

## Secondary Routes

Advanced routes can still be opened directly.

They are not in the primary dock because they are not daily-driver tools, but once opened they behave like any other workspace surface in the task strip.

## Honest States

The shell follows these rules:

- if a surface is open, it has a real route
- if a counter is shown, it comes from a real API
- if an object is inspected, it comes from route-backed selection or the API
- if a behavior is not implemented, the UI says so or omits it
