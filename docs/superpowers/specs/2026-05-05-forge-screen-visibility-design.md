# FORGE Screen Visibility Design

Date: 2026-05-05
Status: proposed design, approved for planning after operator review

## Goal

Give FORGE the ability to see what the operator is seeing in the active or selected window, with explicit operator control. Screen visibility is perception only: it produces evidence/diagnostics for interpretation and cannot directly mutate memory, execute tools, alter responses, or become authority.

## Scope

Initial scope:

- Capture the active or operator-selected window, not the whole desktop.
- Support a one-shot **Look now** action.
- Support a **Watch** mode that samples repeatedly only while a visible toggle is on.
- Store captures as bounded local artifacts or diagnostic records with provenance.
- Keep all captures non-authoritative unless a later governed evidence-admission path explicitly promotes summaries or refs.

Out of scope for the first implementation:

- Whole-desktop or all-monitor capture.
- Silent background capture.
- Remote screen streaming.
- OCR or vision model calls as automatic truth.
- Automatic tool execution based on screen contents.
- Memory writes without governed commit paths.
- Public unauthenticated screen-capture APIs.

## Authority Boundary

Screen visibility must follow existing FORGE doctrine:

- The model is not authority.
- Visual observations are evidence or diagnostics, not truth.
- Any durable memory write must go through existing governed paths.
- Any tool execution still goes through gateway/approval policy.
- Captures must preserve provenance: timestamp, workspace, capture mode, target kind, target label if available, actor, and correlation id.

## Recommended Architecture

Use a hybrid desktop/core design.

Desktop responsibilities:

- Provide the visible **Look now** action.
- Provide the visible **Watch** toggle.
- Select the active or chosen window.
- Perform local OS/window capture through Tauri-side capability code.
- Show obvious state when watch mode is active.
- Stop watch mode immediately when toggled off or when the app exits.

Core responsibilities:

- Define the screen-observation record and artifact metadata.
- Enforce retention and size limits.
- Record provenance and correlation metadata.
- Expose only local, permission-aware internal endpoints or commands needed by the desktop.
- Keep observations non-authoritative and separate from canonical memory.

Gateway/permissions responsibilities:

- Define a screen-capture capability boundary, such as `device.capture_screen` or `ui.screenshot`.
- Require explicit operator enablement for watch mode.
- Keep screen capture auditable.
- Prevent screen observations from becoming a second authority path.

## Data Flow

Look now:

1. Operator clicks **Look now** or asks FORGE to inspect the visible window.
2. Desktop asks the Tauri capture layer for active/selected-window image data.
3. Capture layer returns image bytes plus target metadata.
4. Desktop sends the capture artifact and metadata to core.
5. Core stores a bounded artifact/diagnostic record and returns a capture ref.
6. FORGE can use the capture ref as visual evidence for the current task.

Watch:

1. Operator turns on a visible **Watch** toggle.
2. Desktop samples the active/selected window at a conservative interval.
3. Each sample follows the same artifact/metadata path as **Look now**.
4. Watch mode stops on toggle off, app exit, permission loss, or capture failure policy.
5. Sampling never continues silently.

## Capture Model

Suggested record fields:

- `capture_id`
- `workspace_id`
- `actor`
- `mode`: `look_now` or `watch`
- `target_kind`: `active_window` or `selected_window`
- `target_label`
- `image_artifact_id`
- `mime_type`
- `width`
- `height`
- `captured_at`
- `correlation_id`
- `trace_id`
- `retention_policy`
- `diagnostic_only`
- `metadata`

The record should not include OCR text or model interpretation by default. If later added, interpretations should be separate proposal records linked to the capture.

## Privacy And Safety

Required safeguards:

- Disabled by default.
- Operator-visible capture controls.
- Watch mode has a persistent visible indicator.
- No capture of the whole desktop in the first phase.
- No capture when the selected window is unavailable.
- No automatic capture of password/secret dialogs if the platform exposes a reliable privacy signal; otherwise document the limitation.
- Bounded artifact size and retention.
- No public diagnostic route for raw captures.
- No automatic memory write.
- No automatic tool execution.

## Error Handling

- Permission denied: show a local desktop error and store no capture.
- No active/selected window: return a recoverable no-target result.
- Capture failure: stop one-shot capture; for watch mode, stop or back off visibly.
- Oversized capture: reject or downscale according to policy before storage.
- Core store failure: report failure to desktop and do not fabricate a capture ref.

No failure may trigger live tool execution, modelruntime calls, retrieval, memory writes, or route/API behavior changes.

## Testing

Required tests:

- Screen capture is disabled by default.
- **Look now** requires explicit operator action.
- Watch mode samples only while enabled.
- Watch mode stops when toggled off.
- Only active/selected-window target is accepted in the first phase.
- Whole-desktop/all-monitor capture is rejected or unavailable.
- Capture records are diagnostic-only.
- Capture artifacts preserve provenance.
- Capture failures do not write memory or execute tools.
- Gateway/permission capability checks reject unauthorized capture.
- Retention and size limits are enforced.
- No public route is added unless separately approved.

## Documentation Updates

Implementation should update:

- README current checkpoint
- `AGENTS.md` if a new device-capture rule is added
- architecture docs for tool/device surface or screen visibility
- current phase status
- config reference if a new env/config flag is introduced
- user manual for the **Look now** and **Watch** controls

## Open Decisions For Planning

- Exact Tauri crate/API used for active-window capture on Linux.
- Whether first implementation stores image bytes as artifacts or keeps them in a dedicated diagnostic store.
- Watch interval and retention defaults.
- UI location for the controls.
- Whether screen captures require the existing approval queue or an operator settings toggle plus visible control.
