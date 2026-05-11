# FORGE Full Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the FORGE VM/operator desktop rebuild-safe, add a governed Terminal/Files launcher to the desktop shell, and wire the next diagnostic-only FORGE-K Control Lane shadow seam.

**Architecture:** The VM/Nix work stays in flake/profile/check files. GUI app launching stays in the Tauri session process with a fixed allowlist because it owns the Wayland environment. FORGE-K live integration remains diagnostic-only by injecting a best-effort observer into Control Lane validation result handling without changing decisions.

**Tech Stack:** Nix flakes/NixOS modules, Tauri 2/Rust, React/TypeScript/Vitest, Go Control Lane and `forgekshadow` tests.

---

## File Structure

- Modify `flake.nix`: expose `checks.${system}.forge-operator-desktop`.
- Create `nix/checks/forge-operator-desktop.nix`: static profile safety and session-package check.
- Modify `nix/nixos/modules/forge-shell-session.nix`: prevent `operator-desktop` from using the fullscreen session package default.
- Modify `docs/runbooks/forge_operator_desktop_vm.md`: document rebuild-safe main checkout import and VM repair steps.
- Modify `apps/desktop/src-tauri/src/main.rs`: add fixed allowlisted `launch_operator_app` and `list_operator_apps` commands.
- Modify `apps/desktop/src/lib/desktop.ts`: add typed Tauri wrappers for operator apps.
- Create `apps/desktop/src/pages/OperatorAppsPage.tsx`: visible launcher page.
- Modify `apps/desktop/src/layout/shellConfig.ts`: register `operator-apps`.
- Modify `apps/desktop/src/layout/toolRegistry.tsx`: mount `OperatorAppsPage`.
- Modify `apps/desktop/src/App.tsx`: route `/operator-apps`.
- Add desktop tests near existing shell/page tests.
- Modify `services/core/internal/aios/controllane/processor.go`: add optional shadow observer and emit validation summaries.
- Create `services/core/internal/aios/controllane/shadow_validation.go`: map syscall results to `forgekshadow.ControlLaneValidationInput`.
- Add tests under `services/core/internal/aios/controllane`.
- Update docs/status or reviews only if behavior boundaries change during implementation.

---

### Task 1: Rebuild-Safe Operator Profile Gate

**Files:**
- Create: `nix/checks/forge-operator-desktop.nix`
- Modify: `flake.nix`
- Modify: `nix/nixos/modules/forge-shell-session.nix`
- Modify: `docs/runbooks/forge_operator_desktop_vm.md`

- [ ] **Step 1: Add the static check**

Create `nix/checks/forge-operator-desktop.nix` with assertions that:

```nix
{
  lib,
  runCommand,
  src ? ../..,
}:

runCommand "forge-operator-desktop-profile-check" { } ''
  set -euo pipefail
  profile="$src/nix/nixos/profiles/forge-operator-desktop.nix"
  module="$src/nix/nixos/modules/forge-shell-session.nix"
  runbook="$src/docs/runbooks/forge_operator_desktop_vm.md"

  test -f "$profile"
  grep -F 'forge-operator-session' "$profile"
  grep -F 'pkgs.labwc' "$profile"
  grep -F 'pkgs.foot' "$profile"
  grep -F 'pkgs.pcmanfm' "$profile"
  grep -F 'forge-wayland-session' "$profile"
  grep -F 'autoStart = lib.mkDefault false' "$profile"
  grep -F 'autoLogin.enable = lib.mkDefault false' "$profile"
  grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY = "false"' "$profile"

  grep -F 'operator-desktop' "$module"
  grep -F 'forge-operator-session' "$module"
  grep -F '/mnt/projectforge' "$runbook"

  forbidden='(^|[^A-Za-z])(kde|KDE|plasma|Plasma|gnome|GNOME|xfce|XFCE)([^A-Za-z]|$)|autologin = true|autoLogin.enable = true|xrdp|vnc|systemctl|nixos-rebuild|modprobe|rmmod|reboot|shutdown|LoadModel|UnloadModel|GenerateStream|semantic memory write|os.RemoveAll|rm -rf'
  if grep -RE "$forbidden" "$profile"; then
    echo "forbidden desktop/autologin/host-mutation text found in operator desktop profile" >&2
    exit 1
  fi

  touch "$out"
''
```

- [ ] **Step 2: Expose the check**

In `flake.nix`, add:

```nix
forge-operator-desktop = pkgs.callPackage ./nix/checks/forge-operator-desktop.nix { };
```

inside the existing `checks` attribute set.

- [ ] **Step 3: Harden manual operator session defaults**

In `nix/nixos/modules/forge-shell-session.nix`, update the generated session package selection so `mode = "operator-desktop"` selects `pkgs.forge-operator-session` when available. If unavailable, the module must fail with a clear assertion instead of generating an Exec path to the wrong package.

- [ ] **Step 4: Update VM runbook**

In `docs/runbooks/forge_operator_desktop_vm.md`, add a rebuild-safe section showing `/etc/nixos/configuration.nix` should import:

```nix
/mnt/projectforge/nix/nixos/profiles/forge-operator-desktop.nix
```

and must not import `.worktrees/*` paths.

- [ ] **Step 5: Verify Task 1**

Run:

```powershell
nix build .#checks.x86_64-linux.forge-operator-desktop
nix flake check --no-build
```

Expected: both commands succeed.

---

### Task 2: Operator Apps Tauri Allowlist

**Files:**
- Modify: `apps/desktop/src-tauri/src/main.rs`
- Modify: `apps/desktop/src/lib/desktop.ts`

- [ ] **Step 1: Add Rust data types**

Add serializable structs for app metadata and launch results:

```rust
#[derive(Serialize, Clone)]
struct OperatorApp {
    id: &'static str,
    label: &'static str,
    description: &'static str,
    executable: &'static str,
}

#[derive(Serialize)]
struct OperatorAppLaunchResult {
    app_id: String,
    label: String,
    executable: String,
    launched: bool,
    pid: Option<u32>,
    message: String,
}
```

- [ ] **Step 2: Add fixed allowlist**

Add:

```rust
const OPERATOR_APPS: &[OperatorApp] = &[
    OperatorApp {
        id: "terminal",
        label: "Terminal",
        description: "Open a Foot terminal in the current FORGE operator session.",
        executable: "foot",
    },
    OperatorApp {
        id: "files",
        label: "Files",
        description: "Open the PCManFM file manager in the current FORGE operator session.",
        executable: "pcmanfm",
    },
];
```

- [ ] **Step 3: Add list and launch commands**

Add Tauri commands:

```rust
#[tauri::command]
fn list_operator_apps() -> Vec<OperatorApp> {
    OPERATOR_APPS.to_vec()
}

#[tauri::command]
fn launch_operator_app(app_id: String) -> Result<OperatorAppLaunchResult, String> {
    let app = OPERATOR_APPS
        .iter()
        .find(|candidate| candidate.id == app_id.trim())
        .ok_or_else(|| "operator app is not allowlisted".to_string())?;

    let child = std::process::Command::new(app.executable)
        .spawn()
        .map_err(|err| format!("failed to launch {}: {}", app.label, err))?;

    Ok(OperatorAppLaunchResult {
        app_id: app.id.to_string(),
        label: app.label.to_string(),
        executable: app.executable.to_string(),
        launched: true,
        pid: Some(child.id()),
        message: format!("{} launch requested", app.label),
    })
}
```

Register both commands in `tauri::generate_handler!`.

- [ ] **Step 4: Add TypeScript wrappers**

In `apps/desktop/src/lib/desktop.ts`, export types and wrappers:

```ts
export type OperatorApp = {
  id: string;
  label: string;
  description: string;
  executable: string;
};

export type OperatorAppLaunchResult = {
  appId: string;
  label: string;
  executable: string;
  launched: boolean;
  pid?: number | null;
  message: string;
};
```

Use `invoke("list_operator_apps")` and `invoke("launch_operator_app", { appId })`, and normalize snake_case Rust fields.

- [ ] **Step 5: Verify Task 2**

Run:

```powershell
npm -w @forge/desktop run typecheck
```

Expected: TypeScript compiles.

---

### Task 3: Operator Apps Desktop Page

**Files:**
- Create: `apps/desktop/src/pages/OperatorAppsPage.tsx`
- Modify: `apps/desktop/src/layout/shellConfig.ts`
- Modify: `apps/desktop/src/layout/toolRegistry.tsx`
- Modify: `apps/desktop/src/App.tsx`
- Add or modify desktop tests.

- [ ] **Step 1: Add shell tool id and definition**

Add `"operator-apps"` to `ShellToolId` and add a secondary tool:

```ts
{
  id: "operator-apps",
  label: "Operator Apps",
  shortLabel: "OA",
  route: "/operator-apps",
  description: "Allowlisted terminal and file manager launch surface.",
  primary: false,
}
```

- [ ] **Step 2: Add page**

Create `OperatorAppsPage.tsx` that:

- calls `listOperatorApps()` on mount;
- shows Terminal and Files entries;
- disables launch buttons outside Tauri with a clear status line;
- calls `launchOperatorApp(app.id)` on click;
- shows last launch status;
- never accepts free-form command input.

- [ ] **Step 3: Route and registry**

Add:

```tsx
<Route path="/operator-apps" element={<OperatorAppsPage />} />
```

and map `"operator-apps": OperatorAppsPage` in `toolRegistry.tsx`.

- [ ] **Step 4: Add tests**

Add focused Vitest coverage that confirms:

- `operator-apps` is present in `allShellTools`;
- `getShellTool("/operator-apps")` resolves to the new tool;
- the page does not render a free-form command input.

- [ ] **Step 5: Verify Task 3**

Run:

```powershell
npm run test:desktop
npm run typecheck:desktop
```

Expected: desktop tests and typecheck pass.

---

### Task 4: Phase 14E Control Lane Shadow Emission

**Files:**
- Modify: `services/core/internal/aios/controllane/processor.go`
- Create: `services/core/internal/aios/controllane/shadow_validation.go`
- Add tests under `services/core/internal/aios/controllane`

- [ ] **Step 1: Add narrow observer interface**

In `processor.go`, add:

```go
type ControlLaneValidationObserver interface {
    ObserveControlLaneValidationBestEffort(ctx context.Context, input forgekshadow.ControlLaneValidationInput)
}
```

and add `ControlLaneValidationObserver ControlLaneValidationObserver` to `ProcessorOptions` plus a field on `Processor`.

- [ ] **Step 2: Add mapper**

Create `shadow_validation.go` with a helper that returns `(forgekshadow.ControlLaneValidationInput, bool)` only for the four validation actions. It must derive:

- action from `req.Action`;
- request/workspace/correlation from the syscall request;
- passed from `result.Success`;
- decision from `accepted`, `rejected`, `mismatch`, or `validated` summary fields;
- counts from existing state summary keys when present;
- failure count from `len(result.RejectedReasons)`;
- warning count from `len(result.Warnings)`;
- forbidden effect booleans all false.

- [ ] **Step 3: Emit best-effort in every return path**

Before every return of a validation action result, call a helper:

```go
func (p *Processor) observeControlLaneValidation(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) {
    if p == nil || p.controlLaneValidationObserver == nil {
        return
    }
    input, ok := controlLaneValidationShadowInput(req, result)
    if !ok {
        return
    }
    p.controlLaneValidationObserver.ObserveControlLaneValidationBestEffort(ctx, input)
}
```

The helper must not modify `result`.

- [ ] **Step 4: Add tests**

Add tests proving:

- observer is called for `VALIDATE_REF_SHAPE` dry-run success;
- observer is called for rejected malformed validation;
- observer is not called for a normal semantic write action;
- a panic inside the observer does not change the returned result.

- [ ] **Step 5: Verify Task 4**

Run:

```powershell
cd services/core
go test ./internal/aios/controllane ./internal/forgekshadow ./internal/config
```

Expected: all pass.

---

### Task 5: VM Apply and End-to-End Verification

**Files:**
- No repo file edits unless runbook evidence needs correction.

- [ ] **Step 1: Verify host VirtualBox config**

Run:

```powershell
& 'C:\Program Files\Oracle\VirtualBox\VBoxManage.exe' showvminfo FORGE-OS --machinereadable
```

Expected: VM storage paths are under `E:\VirtualBox VMs\FORGE-OS`, shared folder `ProjectForge` points at `E:\dev\imrobman-dev\project_forge`, and port forward `ssh` maps host `127.0.0.1:2222` to guest `22`.

- [ ] **Step 2: Repair VM Nix config if needed**

In the VM, ensure `/etc/nixos/configuration.nix` imports the current main checkout profile:

```nix
/mnt/projectforge/nix/nixos/profiles/forge-operator-desktop.nix
```

and no `.worktrees` path.

- [ ] **Step 3: Rebuild VM**

Run in the VM:

```sh
cd /mnt/projectforge
sudo nixos-rebuild switch
```

Expected: rebuild succeeds.

- [ ] **Step 4: Verify session tools**

Run in the VM:

```sh
command -v forge-operator-session foot pcmanfm
systemctl is-active forge-core
```

Expected: commands exist and `forge-core` is active.

- [ ] **Step 5: Verify desktop launch behavior**

Launch the operator session, open FORGE, navigate to Operator Apps, and launch Terminal and Files. Expected: both apps open as normal labwc windows over or beside FORGE.

---

### Task 6: Final Verification and Commit

**Files:**
- All files changed by prior tasks.

- [ ] **Step 1: Run aggregate checks**

Run:

```powershell
npm run test:desktop
npm run typecheck:desktop
cd services/core
go test ./internal/aios/controllane ./internal/forgekshadow ./internal/config ./internal/api
```

Run Nix checks in the VM or Nix environment:

```sh
nix build .#checks.x86_64-linux.forge-operator-session
nix build .#checks.x86_64-linux.forge-operator-desktop
nix flake check --no-build
```

- [ ] **Step 2: Commit**

Commit with:

```powershell
git add flake.nix nix/checks/forge-operator-desktop.nix nix/nixos/modules/forge-shell-session.nix docs/runbooks/forge_operator_desktop_vm.md apps/desktop services/core docs/superpowers/plans/2026-05-10-forge-full-integration.md
git commit -m "feat: integrate forge operator desktop and shadow validation"
```

- [ ] **Step 3: Push**

Push main:

```powershell
git push origin main
```
