# Multi-Monitor Layouts

## What This Feature Does

FORGE can now coordinate multiple real shell windows across multiple real monitors.

A layout preset stores:

- layout id and name
- which windows belong to the layout
- each window role
- each window's assigned surfaces
- each window's default active surface
- target monitor assignment
- captured bounds when available
- fallback metadata

## What Counts As Real Support

FORGE multi-monitor support is real only when running inside the Tauri desktop runtime.

Real features:

- `availableMonitors()` for display enumeration
- current window placement and focus state
- real extra shell windows created by Tauri
- per-window route navigation
- persisted layout restore and reflow

Not implemented as fake support:

- synthetic monitor lists in the browser
- pretend windows inside one giant shell pane
- off-screen saved coordinates with no recovery path

## Layout Presets

Seeded presets:

- `Build`
- `Research`
- `Ops`
- `Deep Work`

Operators can also:

- create
- rename
- duplicate
- delete
- edit
- activate
- capture current runtime window placement back into a preset

## Window Roles

Window roles are semantic helpers for layout clarity:

- chat
- workbench
- canvas
- dossier
- ops
- review
- settings
- mixed

These are editable. They are not hard walls.

## Monitor Assignment

Each layout window stores:

- preferred monitor key
- preferred monitor ordinal

Activation uses this order:

1. exact preferred monitor key if present
2. saved monitor ordinal if that display still exists
3. first available display as fallback

## Monitor Change Handling

When the detected monitor signature changes:

- FORGE refreshes the real monitor list
- the main window reapplies the active layout
- windows with missing targets are reflowed onto available displays
- a fallback notice is recorded and shown in the shell

## Runtime Window Registry

FORGE tracks runtime windows with:

- runtime label
- linked layout/window record when applicable
- current route
- focused/background state
- current monitor assignment
- last known bounds

This data powers the shell open-windows list and restore logic.
