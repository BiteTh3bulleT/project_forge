# FORGE G6 Operator Desktop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an opt-in G6 operator desktop where FORGE is the primary shell surface and a lightweight Wayland compositor allows terminal/file-manager/app windows.

**Architecture:** Add a new `forge-operator-session` wrapper that launches a pinned lightweight compositor and starts FORGE as the primary desktop app. Keep the existing Cage fullscreen session as rollback, and add a separate NixOS profile for VM/operator use.

**Tech Stack:** Nix flakes, NixOS modules/profiles, Wayland, labwc, foot, pcmanfm/thunar, Tauri FORGE desktop shell, Go core health checks.

---

## File Map

- Create `nix/packages/forge-operator-session.nix`: pinned session wrapper using `labwc` and `forge-shell-session`.
- Create `nix/checks/forge-operator-session.nix`: static and behavior checks for safe wrapper invariants.
- Create `nix/nixos/profiles/forge-operator-desktop.nix`: opt-in profile with labwc, terminal, file manager, portals, FORGE packages.
- Create `docs/runbooks/forge_operator_desktop_vm.md`: VM operator runbook.
- Create `docs/adr/0013-forge-g6-operator-desktop.md`: architecture decision record.
- Modify `flake.nix`: expose package, app, check, and NixOS module/profile output.
- Modify `nix/overlays/default.nix`: expose `pkgs.forge-operator-session`.
- Modify `nix/nixos/profiles/README.md`: document the new profile.
- Later UI task modifies `apps/desktop/src/layout/shellConfig.ts`, `apps/desktop/src/layout/toolRegistry.tsx`, and related tests only if the current launcher registry supports operator entries without backend work.

## Task 1: Add `forge-operator-session` Package

**Files:**
- Create: `nix/packages/forge-operator-session.nix`
- Test: `nix/checks/forge-operator-session.nix` in Task 2

- [ ] **Step 1: Create the session wrapper**

Add:

```nix
{
  lib,
  writeShellApplication,
  forge-shell-session,
  labwc ? null,
}:

let
  defaultCompositor = if labwc != null then "${labwc}/bin/labwc" else "";
  shellSession = "${forge-shell-session}/bin/forge-shell-session";
in
writeShellApplication {
  name = "forge-operator-session";

  text = ''
    set -euo pipefail

    export FORGE_SHELL_SESSION_ENABLED=true
    export FORGE_SHELL_MODE=operator-desktop
    export FORGE_CORE_URL="''${FORGE_CORE_URL:-http://127.0.0.1:18492}"
    export VITE_FORGE_API_URL="''${VITE_FORGE_API_URL:-$FORGE_CORE_URL}"
    export FORGE_SHELL_SAFE_MODE=true
    export FORGE_SHELL_FULLSCREEN=false
    export FORGE_SHELL_HOST_MUTATION=false
    export FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false
    export FORGE_SHELL_MODEL_MUTATION=false
    export FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false
    export FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false
    export FORGE_SHELL_DISPLAY_BACKEND=wayland
    export FORGE_SHELL_COMPOSITOR=labwc
    export XDG_SESSION_TYPE=wayland

    compositor="${defaultCompositor}"
    shell_session="${shellSession}"

    if [ -z "$compositor" ]; then
      echo "FORGE operator compositor is not configured; install or pass labwc when building forge-operator-session." >&2
      exit 1
    fi

    if [ ! -x "$compositor" ]; then
      echo "FORGE operator compositor is not executable: $compositor" >&2
      exit 1
    fi

    if [ ! -x "$shell_session" ]; then
      echo "FORGE shell session wrapper is not executable: $shell_session" >&2
      exit 1
    fi

    exec "$compositor" --startup "$shell_session" "$@"
  '';

  meta = with lib; {
    description = "Opt-in FORGE operator desktop session launcher using labwc";
    license = licenses.mit;
    mainProgram = "forge-operator-session";
    platforms = platforms.linux;
  };
}
```

- [ ] **Step 2: Check package formatting**

Run: `nix fmt nix/packages/forge-operator-session.nix`

Expected: command exits 0.

## Task 2: Add Wrapper Safety Check

**Files:**
- Create: `nix/checks/forge-operator-session.nix`
- Modify: `flake.nix`

- [ ] **Step 1: Create check derivation**

Add a check modeled after `nix/checks/forge-wayland-session.nix`, but assert G6 values:

```nix
{
  lib,
  stdenv,
  forge-operator-session,
  callPackage,
  writeShellApplication,
}:

let
  fakeLabwc = writeShellApplication {
    name = "labwc";
    text = ''
      test "$1" = "--startup"
      shift
      echo "fake-labwc:$FORGE_SHELL_SESSION_ENABLED:$FORGE_SHELL_MODE:$FORGE_CORE_URL:$1:$*"
    '';
  };
  fakeShellSession = writeShellApplication {
    name = "forge-shell-session";
    text = ''
      echo "fake-shell-session:$*"
    '';
  };
  testOperatorSession = callPackage ../packages/forge-operator-session.nix {
    labwc = fakeLabwc;
    forge-shell-session = fakeShellSession;
  };
in
stdenv.mkDerivation {
  name = "forge-operator-session-wrapper-check";

  dontUnpack = true;
  dontConfigure = true;
  dontBuild = true;

  installPhase = ''
    runHook preInstall

    real_wrapper="${forge-operator-session}/bin/forge-operator-session"
    wrapper="${testOperatorSession}/bin/forge-operator-session"

    test -x "$real_wrapper"
    test -x "$wrapper"

    grep -F 'FORGE_SHELL_MODE=operator-desktop' "$wrapper"
    grep -F 'FORGE_SHELL_FULLSCREEN=false' "$wrapper"
    grep -F 'FORGE_SHELL_SAFE_MODE=true' "$wrapper"
    grep -F 'FORGE_SHELL_HOST_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false' "$wrapper"
    grep -F 'FORGE_SHELL_MODEL_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false' "$wrapper"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$wrapper"
    grep -F 'FORGE_SHELL_COMPOSITOR=labwc' "$wrapper"
    grep -F 'exec "$compositor" --startup "$shell_session" "$@"' "$wrapper"

    if grep -F 'FORGE_OPERATOR_COMPOSITOR' "$real_wrapper" "$wrapper"; then
      echo "forge-operator-session must not accept compositor executable paths from ambient environment" >&2
      exit 1
    fi

    forbidden='systemctl|nixos-rebuild|modprobe|rmmod|reboot|shutdown|apt-get|dnf|zypper|pacman|LoadModel|UnloadModel|GenerateStream|os.RemoveAll|rm -rf'
    if grep -E "$forbidden" "$wrapper"; then
      echo "forbidden host/runtime mutation text found in forge-operator-session wrapper" >&2
      exit 1
    fi

    FORGE_OPERATOR_COMPOSITOR="$TMPDIR/must-not-run" \
      FORGE_CORE_URL=http://127.0.0.1:19994 \
      "$wrapper" arg1 arg2 > "$TMPDIR/operator.out"
    grep -F 'fake-labwc:true:operator-desktop:http://127.0.0.1:19994:' "$TMPDIR/operator.out"
    grep -F 'forge-shell-session' "$TMPDIR/operator.out"
    grep -F 'arg1 arg2' "$TMPDIR/operator.out"

    mkdir -p "$out"
    echo "ok" > "$out/result"
    runHook postInstall
  '';

  meta = with lib; {
    description = "Static safety checks for the FORGE G6 operator desktop wrapper";
    license = licenses.mit;
    platforms = platforms.linux;
  };
}
```

- [ ] **Step 2: Wire check in `flake.nix`**

Add package/app/check entries:

```nix
forge-operator-session = pkgs.callPackage ./nix/packages/forge-operator-session.nix {
  forge-shell-session = self.packages.${system}.forge-shell-session;
};
```

```nix
forge-operator-session = {
  type = "app";
  program = "${self.packages.${system}.forge-operator-session}/bin/forge-operator-session";
};
```

```nix
forge-operator-session = pkgs.callPackage ./nix/checks/forge-operator-session.nix {
  forge-operator-session = self.packages.${system}.forge-operator-session;
};
```

- [ ] **Step 3: Run the focused check**

Run: `nix build .#checks.x86_64-linux.forge-operator-session`

Expected: build succeeds and writes `ok`.

## Task 3: Add NixOS Operator Desktop Profile

**Files:**
- Create: `nix/nixos/profiles/forge-operator-desktop.nix`
- Modify: `flake.nix`

- [ ] **Step 1: Create profile**

Add a profile that imports the shell module, enables safe shell metadata, and installs the operator app set:

```nix
{
  config,
  lib,
  options,
  pkgs,
  ...
}:

let
  forgeDesktopShell = pkgs.callPackage ../../packages/forge-desktop-shell.nix { };
  forgeShellSession = pkgs.callPackage ../../packages/forge-shell-session.nix {
    forgeDesktopShell = forgeDesktopShell;
  };
  forgeOperatorSession = pkgs.callPackage ../../packages/forge-operator-session.nix {
    forge-shell-session = forgeShellSession;
  };
  notoEmoji = pkgs.noto-fonts-color-emoji or pkgs.noto-fonts-emoji;
  fileManager = pkgs.pcmanfm;
in
{
  imports = [
    ../modules/forge-shell-session.nix
  ];

  forge.shellSession = {
    enable = lib.mkDefault true;
    package = lib.mkDefault forgeShellSession;
    mode = lib.mkDefault "operator-desktop";
    displayBackend = lib.mkDefault "wayland";
    compositor = lib.mkDefault "labwc";
    autoStart = lib.mkDefault false;
    safeMode = lib.mkDefault true;
    fullscreen = lib.mkDefault false;
  };

  services.displayManager.autoLogin.enable = lib.mkDefault false;
  services.dbus.enable = lib.mkDefault true;
  security.polkit.enable = lib.mkDefault true;
  hardware.graphics.enable = lib.mkDefault true;

  xdg.portal = {
    enable = lib.mkDefault true;
    extraPortals = lib.mkDefault [
      pkgs.xdg-desktop-portal-gtk
    ];
  };

  virtualisation.virtualbox.guest =
    {
      enable = lib.mkDefault true;
    }
    // lib.optionalAttrs (lib.hasAttrByPath [ "virtualisation" "virtualbox" "guest" "x11" ] options) {
      x11 = lib.mkDefault false;
    };

  fonts.packages = lib.mkDefault [
    pkgs.noto-fonts
    notoEmoji
    pkgs.dejavu_fonts
  ];

  environment.systemPackages = [
    forgeOperatorSession
    forgeShellSession
    forgeDesktopShell
    pkgs.labwc
    pkgs.foot
    fileManager
    pkgs.dbus
    pkgs.xdg-utils
    pkgs.xdg-desktop-portal
    pkgs.xdg-desktop-portal-gtk
    pkgs.mesa-demos
    pkgs.noto-fonts
    notoEmoji
    pkgs.dejavu_fonts
  ];

  environment.sessionVariables = {
    FORGE_SHELL_SESSION_ENABLED = "true";
    FORGE_SHELL_MODE = "operator-desktop";
    FORGE_SHELL_DISPLAY_BACKEND = "wayland";
    FORGE_SHELL_COMPOSITOR = "labwc";
    FORGE_SHELL_SAFE_MODE = "true";
    FORGE_SHELL_FULLSCREEN = "false";
    FORGE_SHELL_HOST_MUTATION = "false";
    FORGE_SHELL_DIRECT_SYSTEM_CONTROL = "false";
    FORGE_SHELL_MODEL_MUTATION = "false";
    FORGE_SHELL_SEMANTIC_MEMORY_WRITE = "false";
    FORGE_SHELL_FORGE_K_LIVE_AUTHORITY = "false";
    FORGE_CORE_URL = lib.mkDefault "http://127.0.0.1:18492";
    VITE_FORGE_API_URL = lib.mkDefault "http://127.0.0.1:18492";
    XDG_SESSION_TYPE = "wayland";
    GDK_BACKEND = "wayland,x11";
  };

  assertions = [
    {
      assertion = config.forge.shellSession.autoStart == false;
      message = "G6 operator desktop profile must not autostart FORGE Shell.";
    }
    {
      assertion = config.services.displayManager.autoLogin.enable == false;
      message = "G6 operator desktop profile must keep automatic login disabled.";
    }
  ];
}
```

- [ ] **Step 2: Expose profile in `flake.nix`**

Add:

```nix
forge-operator-desktop = import ./nix/nixos/profiles/forge-operator-desktop.nix;
```

- [ ] **Step 3: Evaluate profile**

Run: `nix flake check --no-build`

Expected: flake evaluation succeeds.

## Task 4: Overlay and Profile Docs

**Files:**
- Modify: `nix/overlays/default.nix`
- Modify: `nix/nixos/profiles/README.md`

- [ ] **Step 1: Add overlay package**

Add:

```nix
forge-operator-session = final.callPackage ../packages/forge-operator-session.nix {
  forge-shell-session = final.forge-shell-session;
};
```

- [ ] **Step 2: Document profile**

Append `forge-operator-desktop.nix` to the profile README with this launch flow:

```text
TTY login
-> forge-operator-session
-> labwc
-> forge-shell-session
-> packaged forge-desktop-shell
-> local forge-core
```

State that it installs terminal/file-manager support, remains opt-in, keeps TTY fallback, and does not make FORGE-K live authority.

## Task 5: Add ADR and Runbook

**Files:**
- Create: `docs/adr/0013-forge-g6-operator-desktop.md`
- Create: `docs/runbooks/forge_operator_desktop_vm.md`

- [ ] **Step 1: Write ADR**

ADR must record:

- full GNOME/KDE rejected for target architecture
- Cage-only rejected for app-window needs
- lightweight compositor under FORGE accepted
- FORGE-K remains non-authoritative
- rollback keeps Cage/TTY

- [ ] **Step 2: Write VM runbook**

Runbook must include:

```bash
forge-operator-session
```

and verification commands:

```bash
systemctl is-active forge-core
curl -fsS http://127.0.0.1:18492/health
foot &
pcmanfm &
```

It must also include rollback:

```bash
forge-wayland-session
```

## Task 6: Add Minimal FORGE UI Operator Entries

**Files:**
- Inspect first: `apps/desktop/src/layout/shellConfig.ts`
- Inspect first: `apps/desktop/src/layout/toolRegistry.tsx`
- Modify only if existing registry supports static entries without unsafe command execution.

- [ ] **Step 1: Read existing launcher model**

Run:

```bash
rg -n "assignableShellTools|toolRegistry|StartPage|dock|launcher" apps/desktop/src/layout apps/desktop/src/pages apps/desktop/src/stores
```

Expected: identify whether static non-executing entries can be added safely.

- [ ] **Step 2: If safe, add placeholder operator tools**

Add visible entries for Terminal and Files that route to an internal explanatory surface if no safe launcher API exists yet. Do not add arbitrary command execution.

- [ ] **Step 3: If not safe, defer UI launch buttons**

Record deferral in the runbook and leave launcher to TTY commands for this phase.

## Task 7: Verify Locally and in VM

**Files:**
- No new files unless fixing issues from verification.

- [ ] **Step 1: Format Nix**

Run:

```bash
nix fmt flake.nix nix/packages/forge-operator-session.nix nix/checks/forge-operator-session.nix nix/nixos/profiles/forge-operator-desktop.nix nix/overlays/default.nix
```

Expected: exits 0.

- [ ] **Step 2: Build focused checks**

Run:

```bash
nix build .#checks.x86_64-linux.forge-operator-session
nix build .#checks.x86_64-linux.forge-wayland-session
```

Expected: both pass.

- [ ] **Step 3: Build package**

Run:

```bash
nix build .#packages.x86_64-linux.forge-operator-session
```

Expected: package builds and exposes `bin/forge-operator-session`.

- [ ] **Step 4: Rebuild VM profile**

Inside the VM repository checkout or shared folder, adapt the existing NixOS config import from `forge-vbox-graphics-test.nix` to `forge-operator-desktop.nix`, then run:

```bash
sudo nixos-rebuild switch
```

Expected: rebuild succeeds and TTY remains available.

- [ ] **Step 5: Launch G6 session in VM**

Run:

```bash
forge-operator-session >/mnt/vmdisk/forge-operator-session.log 2>&1
```

Expected: labwc starts, FORGE shell opens, core status is online.

- [ ] **Step 6: Verify app windows**

From a terminal or TTY:

```bash
DISPLAY= WAYLAND_DISPLAY="${WAYLAND_DISPLAY:-wayland-0}" foot &
DISPLAY= WAYLAND_DISPLAY="${WAYLAND_DISPLAY:-wayland-0}" pcmanfm &
```

Expected: terminal and file manager open as separate windows that can overlap or sit beside FORGE.

- [ ] **Step 7: Verify rollback**

Exit G6 session and run:

```bash
forge-wayland-session >/mnt/vmdisk/forge-wayland-rollback.log 2>&1
```

Expected: old Cage fullscreen shell still launches.

## Self-Review

- Spec coverage: This plan covers wrapper, profile, checks, docs, rollback, VM validation, and minimal UI exposure from the G6 spec.
- Placeholder scan: No unfinished placeholder markers or arbitrary implementation gaps should remain before execution.
- Type consistency: Package names use `forge-operator-session`; profile output uses `forge-operator-desktop`.
