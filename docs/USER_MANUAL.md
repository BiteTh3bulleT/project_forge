# FORGE Operator Manual

This manual describes the shipped FORGE desktop shell, including monitor-aware multi-window layouts.

## 1. Operating Model

FORGE is a local-first AI workspace backed by a Go core service and a Tauri desktop client.

The desktop shell now consists of:

1. `Top bar`
2. `Dock`
3. `Per-window task strip`
4. `Workspace surface`
5. `Context panel`
6. `Layout system`

## 2. Windows And Monitors

FORGE can use multiple real shell windows.

Each window:

- has a real Tauri runtime label
- belongs to the same shared workspace session
- can be assigned a role
- can be placed on a specific monitor through a named layout
- reports its real focus, route, and display state back to the shell

## 3. Layout Presets

A layout preset defines:

- which windows exist
- which display each window targets
- which surfaces each window can host
- which surface each window should show by default

Use `#/layouts` to manage presets.

## 4. Activating A Layout

When you activate a layout, FORGE will:

1. refresh the real monitor list
2. move or spawn the required windows
3. route each window to its assigned surface
4. close unneeded secondary layout windows
5. update the shell state across all windows

## 5. Restore On Relaunch

On relaunch, the `main` shell window restores the active layout and recreates the secondary windows that belong to it.

Restore is guided by the current monitor list, not by stale blind coordinates.

## 6. Monitor Fallback

If a saved display is missing:

- the affected window is reflowed to an available display
- the shell records a fallback notice
- the operator can reassign the window in `#/layouts`

FORGE does not intentionally leave layout windows off-screen.

## 7. Browser vs Desktop Runtime

Monitor-aware multi-window behavior requires the Tauri runtime.

In the browser dev shell:

- saved layouts remain visible and editable
- monitor detection is not simulated
- extra shell windows are not simulated

## 8. Primary Tools

Primary tools remain:

- `Chat`
- `Workbench`
- `Canvas`
- `Dossiers`
- `Jobs`
- `Reviews`
- `Approvals`
- `Settings`

Additional operator routes include:

- `Logs` (`#/events`)
- `Layouts` (`#/layouts`)

## 9. Persisted State

Persisted by the desktop shell:

- layout presets
- active layout id
- runtime window registry
- last known monitor snapshot
- fallback notice
- per-window task-strip session state
- UI mode preference

Persisted by FORGE core:

- chat threads and messages
- chat file attachments (stored as artifacts and linked in message metadata)
- boards and notes
- jobs, events, approvals, reviews, dossiers, artifacts, settings

## 10. Chat Attachments And Inspector

Chat supports direct file upload from the composer:

- upload docs, images, and other files
- send a message with selected attachment references
- inspect attachments in the right-side `Inspector` panel (`Files` tab)
- open any attachment in `Workbench` for deeper inspection

When assistant messages contain fenced code blocks, the `Inspector` panel `Code` tab lists and previews those blocks with copy support.

## 11. Memory Operations

Use `#/memory` for operator-facing memory control:

- browse persisted observations
- filter by dossier/type/stale status
- inspect raw content, provenance, links, and signal history
- mark observations `useful`, `not_useful`, `noisy`, or `insufficient`
- toggle stale state and verification timestamp
- trigger repair runs and inspect per-observation repair outcomes

Use `#/retrieval-runs` to inspect:

- retrieval mode and weighting
- ranking scores
- per-result selection reason JSON
- observation linkage for each result

## 12. Dossier Memory Scope

In `#/dossiers`, each dossier now shows:

- observation count and stale count
- recent observations in scope
- recent usefulness signals
- recent packet alignment notes

This is the operator-visible memory scope for that project profile.

## 13. Packet Alignment Visibility

In `#/jobs/:id`, Packet Preview includes alignment notes explaining why retrieval evidence entered that packet.

This is the inspectable bridge between retrieval and execution contracts.

## 14. Personality Prompt Control

In `#/settings`, chat personality prompt can be edited live and saved to settings.

Changes apply to subsequent assistant turns without restart.

## 15. Deferred Or Limited

Implemented limits:

- no freeform floating window manager
- no synthetic browser monitor emulation
- monitor identity is derived from real Tauri monitor data because the frontend API does not expose a permanent opaque monitor UUID

## 16. Operator Guidance

Use named layouts deliberately.

Recommended pattern:

1. keep `Build`, `Research`, `Ops`, and `Deep Work` as baseline presets
2. activate the closest preset
3. tweak the layout in `#/layouts`
4. use `Capture running windows` to save the refined arrangement back into the preset

## 17. Governed Tool Layer

FORGE machine actions run through one governed gateway.

Use these operator surfaces:

- `#/gateway` to inspect and invoke typed actions
- `#/action-lanes` to edit lane scopes and write intent
- `#/permissions` to control allowed roots/tools/risk gates
- `#/approvals` to approve or deny gated requests
- `#/audit` to trace outcomes by correlation id

## 18. Chat Tool Dispatch

Chat supports explicit governed tool commands:

- `/tool {\"toolId\":\"fs.list\",\"laneId\":\"fs.list\",\"paths\":[\".\"],\"dryRun\":true}`

This queues a `gateway_action` job rather than bypassing jobs/approvals/audit.

## 19. Risk And Execution Levels

Risk classes:

- `read_only`
- `safe_write`
- `scoped_execute`
- `privileged`
- `dangerous`

Execution levels:

- `L0` read inspection
- `L1` safe write
- `L2` controlled execute
- `L3` privileged
- `L4` dangerous

## 20. Deferred

- non-job gateway requests that require approval do not yet create full approval tickets without a job record
