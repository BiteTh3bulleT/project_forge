# Workspace Layouts

## Creating A Layout

Open `#/layouts`.

From there you can:

1. create a new named layout
2. edit the layout name
3. add or remove layout windows
4. assign a role to each window
5. choose a target display
6. select which surfaces belong in the window
7. choose the default focused surface for that window

## Activating A Layout

Activation is available from:

- the shell top-bar layout selector
- the `#/layouts` page

Activation performs real runtime work:

- spawns missing windows
- repositions existing windows
- routes each window to its assigned surface
- closes no-longer-needed secondary layout windows

## Capturing Current Runtime State

Use `Capture running windows` in `#/layouts` to update a preset from the current desktop state.

This captures:

- window bounds
- current route per window
- detected monitor assignment
- current title

## Editing Surface Assignments

Surface assignments are route-based and explicit.

If a default surface is removed from a window's allowed surface list, FORGE automatically picks the first remaining route in that list.

## Deleting Layouts

Layouts can be deleted, but FORGE keeps at least one layout available so the shell always has a recoverable configuration.

## Main Window Rule

The first layout window is pinned to the real Tauri `main` window.

This ensures:

- restore always has a starting shell window
- monitor fallback always has a recovery anchor
- FORGE does not lose the operator in a layout transition
