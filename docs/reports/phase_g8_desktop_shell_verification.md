# Phase G8 Desktop Shell Verification

Status date: 2026-05-18

Status marker: `DESKTOP_SHELL_POLISH / OPERATOR_UI_AUTHORITY / LABWC_SUBSTRATE_PRESERVED / NO_HOST_MUTATION / NO_FORGE_K_AUTHORITY_EXPANSION`

## Scope

Phase G8 verifies and tightens the operator-facing desktop shell. FORGE owns the visible desktop surface, launcher, taskbar, in-shell FORGE windows, native app registry consumption, bounded native-window requests, clean desktop boot behavior, and operator status messaging.

labwc remains the Wayland compositor substrate. It owns native window placement, low-level focus/minimize/maximize/fullscreen/close mechanics, drag/resize behavior, output routing, and compositor policy.

## Current Observed Behavior

| Behavior | Status | Evidence |
|---|---|---|
| FORGE loading/login path reaches the desktop in the VM | Verified | `FORGE-OS` VM live-start observation on 2026-05-18 at commit `0867e26`; evidence summary in `docs/evidence/vm_boot/2026-05-18-live-start/README.md`. |
| Desktop starts empty after login | Verified | VM live-start observation and shell login reset tests from the prior render pass. |
| Core reachable from the operator VM | Verified | `GET http://127.0.0.1:18492/health` returned `ok: true` during the 2026-05-18 VM check. |
| Modelruntime health visible from the operator VM | Verified | `GET http://127.0.0.1:18492/forge/model-runtime/health` reported the local `ollama_compat` backend reachable; VirtualBox GPU acceleration remained disabled as expected. |
| Native app taskbar placeholders resolve when compositor windows appear | Verified by test | `npm -w @forge/desktop run test -- AppShell.test.tsx`. |
| Refused native launches do not leave phantom taskbar entries | Fixed and verified | `AppShell` now honors `launched:false`; focused AppShell tests cover the behavior. |
| Duplicate native launch requests are blocked while a launch is pending | Fixed and verified | `AppShell` now tracks in-flight native app ids; focused AppShell tests cover the behavior. |
| Stale native launch placeholders expire | Fixed and verified | Pending placeholders expire after 30 seconds without a matching compositor window; focused AppShell tests cover the behavior. |
| Left click on focused native taskbar entries minimizes the window | Fixed and verified | Native taskbar clicks now route through bounded compositor actions; focused AppShell tests cover the behavior. |
| Unsupported native window actions fail visibly | Fixed and verified | Failed bounded compositor requests now surface a user-visible status message. |
| Multiple compositor-reported native windows can be represented separately | Fixed and verified | Taskbar keys native entries by compositor window id; focused AppShell tests cover two same-app compositor windows. Broader manual VM multi-window smoke remains open. |
| Closed compositor snapshots are ignored by the taskbar | Fixed and verified | The shell filters `lifecycle:"closed"` snapshots before taskbar rendering and post-action refresh; focused AppShell tests cover the behavior. |
| Repeated native launches do not resolve against already-visible matching windows | Fixed and verified | Pending launch records exclude matching compositor window ids visible at launch time; focused AppShell tests cover the rapid repeat-launch case. |

## What Was Fixed

- `apps/desktop/src/layout/AppShell.tsx` now treats native launch results as authoritative: only `launched:true` creates a pending taskbar placeholder.
- Native launch requests are de-duplicated while the previous request for the same app is pending.
- Pending native placeholders expire after 30 seconds if no compositor window appears.
- Native taskbar left-click behavior now focuses/restores inactive windows and minimizes focused non-minimized windows through the bounded compositor action path.
- Bounded compositor action failures now set an operator-visible status message instead of silently returning.
- Closed compositor snapshots are filtered before the taskbar treats them as active native windows.
- Pending launches now record matching native window ids visible at launch time and will not resolve against those pre-existing windows.
- `apps/desktop/src/layout/AppShell.test.tsx` now covers refused launches, duplicate pending launches, stale placeholder expiration, active-window minimize behavior, multiple same-app native windows, repeated same-app launches, closed compositor snapshots, and bounded failure diagnostics.

## Native App UX Findings

- Native app discovery remains backend-owned through the Tauri operator app registry.
- The frontend consumes registry/compositor lifecycle snapshots; it does not own native window truth.
- Launch preference resolution is still conservative and registry-backed. Terminal, file manager, and browser preference resolution beyond the current backend registry entries remains future work.
- Unsupported native action capability metadata is not yet exposed by the compositor bridge. G8 now fails unsupported actions visibly and safely; hiding or disabling unsupported menu actions needs backend support metadata.

## Taskbar Interaction Findings

- In-shell FORGE windows and compositor-native windows remain separate taskbar object classes.
- Native Linux windows are keyed by compositor window id, so multiple native windows are representable as separate taskbar entries.
- Middle-click requests bounded close.
- Right-click exposes bounded focus/minimize/maximize/fullscreen/close requests.
- Left-click now requests minimize for focused native windows and focus/restore for inactive or minimized native windows.

## Clean Boot Findings

- The VM reached the FORGE shell desktop after graphical login during the 2026-05-18 live-start check.
- The shell resets restored in-shell windows on login so the first operator desktop is empty except for the shell chrome/taskbar.
- Native compositor windows are observed, not forcibly cleared by FORGE. If labwc/session restoration later reopens native windows, that must be handled as a substrate/session policy issue, not by giving FORGE direct host mutation authority.

## Tests Run

- Red focused test run before implementation: `npm -w @forge/desktop run test -- AppShell.test.tsx` failed 5 expected new cases.
- Green focused test run after implementation: `npm -w @forge/desktop run test -- AppShell.test.tsx` passed 33 tests.
- Desktop validation: `npm run validate:desktop` passed.
- Core tests: `npm test` passed after making config file-mode assertions platform-aware for Windows.
- Core vet/static check: `npm run lint` passed.
- Whitespace check: `git diff --check` passed.

## Manual Checks Run

- VM/session: VirtualBox `FORGE-OS`, host-observed desktop session on 2026-05-18.
- Commit baseline during manual observation: `0867e26`.
- Confirmed core health and modelruntime health from the Windows host.

## Skipped Or Blocked Validation

- Full manual multi-native-window smoke in the VM remains open.
- No reboot, shutdown, `systemctl`, `nixos-rebuild`, package install, model load/unload, or host mutation checks were performed from the shell UI.

## Known Limitations

- The compositor bridge does not yet report per-window action capability metadata, so unsupported native actions cannot be hidden precisely. They now fail visibly and safely.
- Preferred terminal/file/browser resolution is still backend-registry based and not yet operator-configurable.
- VM manual smoke still needs an operator pass for two terminals, two file-manager windows, browser open/close, and shell restart after the current code lands.

## Rollback Notes

Rollback remains configuration/session rollback. Keep fallback desktop/session selection, TTY access, and Nix generation rollback available. Do not delete `/forge` data as part of shell rollback.

## Recommended Next Phase

G9 should focus on native app grouping plus semantic workspace/session persistence once the G8 manual VM smoke is completed.
