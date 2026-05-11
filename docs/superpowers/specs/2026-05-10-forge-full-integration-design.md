# FORGE Full Integration Design

Status: Approved for planning

Date: 2026-05-10

## Goal

Bring the FORGE VM and desktop shell forward as a rebuild-safe operator environment, then add the first practical desktop integration surface, then advance FORGE-K live integration through the next diagnostic-only Control Lane seam.

## Scope

This design covers three ordered integration tracks:

1. Rebuild-safe VM/Nix foundation.
2. Governed operator desktop app launching.
3. FORGE-K Phase 14E Control Lane validation shadow emission.

This design does not make FORGE-K live authority, replace the live AI-OS authority path, add arbitrary command execution from the desktop, enable autologin, remove TTY fallback, or grant session wrappers host mutation authority.

## Current Findings

The current main branch already exposes `forge-desktop-shell`, `forge-shell-session`, `forge-wayland-session`, and `forge-operator-session` packages and apps from `flake.nix`. It also exports the G6 `forge-operator-desktop` NixOS profile.

The VM is configured in VirtualBox as `FORGE-OS` on the E drive with 16 GiB RAM, 6 CPUs, VMSVGA, 128 MiB VRAM, 3D acceleration enabled, NAT port forwards for SSH, core, Vite, Tauri dev, and a shared folder named `ProjectForge` pointing at this checkout.

The repo-side risk is that the operator desktop profile has no static profile check yet. The module also has a footgun where a manual config can set `mode = "operator-desktop"` while leaving `wayland.sessionPackage` at the fullscreen Wayland session default.

The desktop app is a Tauri 2, Vite, and React shell. Tauri currently exposes only read-only system diagnostics. Shell tools are declared in `apps/desktop/src/layout/shellConfig.ts`, routed in `apps/desktop/src/App.tsx`, rendered through `apps/desktop/src/layout/toolRegistry.tsx`, and API calls are centralized in `apps/desktop/src/lib/api.ts`.

The live FORGE-K seams remain validation and diagnostics only. Live Control Lane validators exist for `VALIDATE_KV_IDENTITY`, `VALIDATE_REF_SHAPE`, `COMPARE_REF_SHAPE`, and `VALIDATE_SEMANTIC_OPERATION`. Shadow reporting support exists, but there is no live call site feeding those validation results into the shadow observer.

## Track 1: Rebuild-Safe VM/Nix Foundation

The first implementation slice must harden the checked-in Nix integration before touching desktop launch behavior.

Changes:

- Add a static check for `nix/nixos/profiles/forge-operator-desktop.nix`.
- Expose that check from `flake.nix`.
- Harden or test the `operator-desktop` session package selection so manual configs cannot silently generate a broken session descriptor.
- Update VM-facing docs to prefer flake module imports or the current `/mnt/projectforge` main checkout path, not stale worktree paths.

Runtime application:

- Verify the VirtualBox VM still uses E-drive storage and the `ProjectForge` shared folder.
- Rebuild the VM only after the repo checks pass.
- If SSH access remains key-blocked, use the VM console for the one-time `/etc/nixos/configuration.nix` repair and document the exact target content in the runbook.

## Track 2: Governed Operator App Launcher

FORGE should become the practical operator desktop surface without becoming an arbitrary host command launcher.

The first launcher slice is an allowlisted app launcher:

- `terminal`: launches `foot`.
- `files`: launches `pcmanfm`.

The launch surface must be deterministic. The UI may request only an app id. It must not accept arbitrary executable paths, command strings, shell fragments, model-generated arguments, or package-manager/service-control requests.

Preferred architecture:

- Add a visible `Operator Apps` shell tool/page in the React desktop.
- Add a Tauri command that launches only the fixed allowlist from the session process, because the session process has the correct Wayland environment.
- Keep the Go core as the status/audit side for launcher visibility when practical, but do not route GUI process spawning through HostBridge diagnostics.
- Record launch results in the UI state immediately and preserve a future hook for core audit.

This launcher does not install Ollama, start services, pull models, run `nixos-rebuild`, or execute shell text. Ollama setup remains manual through the terminal at this phase. Modelruntime controls remain on the existing Models page and its governed API paths.

## Track 3: FORGE-K Phase 14E Shadow Emission

The next FORGE-K integration slice should wire existing live Control Lane validation summaries into the existing `forgekshadow` observer.

Rules:

- Dual-flag gated: `FORGE_K_SHADOW_MODE_ENABLED=true` and `FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED=true`.
- Best-effort only.
- Sink failures must not affect Control Lane decisions.
- Validation results must not alter route output, user-visible output, memory, retrieval, embeddings, modelruntime, gateway, tool execution, semantic operation execution, evidence admission, or context compilation.
- No FORGE-K simulator service becomes live authority.

Implementation target:

- Attach the observer at the Control Lane processor boundary after a validation result exists.
- Emit bounded scalar summaries only.
- Add tests that prove decisions and returned validation payloads are unchanged with shadow enabled and that observer failures are isolated.

## Testing

Repo verification must include:

- `nix build .#checks.x86_64-linux.forge-operator-session`
- `nix build .#checks.x86_64-linux.forge-operator-desktop`
- `nix flake check --no-build`
- `npm run test:desktop`
- `npm run typecheck:desktop`
- `cd services/core && go test ./internal/aios/controllane ./internal/forgekshadow ./internal/config ./internal/api`

VM verification must include:

- `sudo nixos-rebuild switch`
- `command -v forge-operator-session foot pcmanfm`
- `systemctl is-active forge-core`
- Launch FORGE operator session and verify terminal/file manager windows open over or beside FORGE.

## Non-Goals

- No full GNOME/KDE desktop.
- No autologin.
- No replacement of NixOS as boot/session substrate.
- No arbitrary command launcher from FORGE desktop.
- No desktop wrapper calls to `systemctl`, `nixos-rebuild`, package managers, kernel module commands, reboot, or shutdown.
- No FORGE-K live authority promotion.
- No live KV reuse.
- No modelruntime driver authority migration.
- No semantic memory mutation from shadow diagnostics.

## Acceptance Criteria

- The repo has a checked static profile gate for the operator desktop profile.
- The VM can rebuild from current main paths without relying on deleted worktrees.
- FORGE desktop includes a visible operator app launcher for Terminal and Files.
- Launching Terminal and Files works in the VM under the operator desktop session.
- Phase 14E shadow emission is implemented only as disabled-by-default diagnostics.
- Tests prove Phase 14E has no decision, output, authority, memory, gateway, retrieval, or modelruntime effects.
