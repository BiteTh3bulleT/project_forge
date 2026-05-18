# FORGE Native Desktop Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the canonical FORGE operator VM boot to a FORGE-OS splash, graphical password login, and default FORGE native desktop session.

**Architecture:** Add a dedicated NixOS profile that composes the existing operator desktop profile with Plymouth and greetd/ReGreet. Point the canonical VM at that profile while keeping the manual operator desktop profile available as a recovery building block.

**Tech Stack:** NixOS modules/profiles, Plymouth, greetd, ReGreet, labwc/Wayland, existing `forge-operator-session`, static Nix checks, Markdown runbooks.

---

### Task 1: Add Native Desktop Runtime Profile

**Files:**
- Create: `nix/nixos/profiles/forge-native-desktop-runtime.nix`
- Modify: `flake.nix`
- Modify: `nix/nixos/configurations/forge-operator-vm.nix`

- [ ] **Step 1: Create the profile**

Create `nix/nixos/profiles/forge-native-desktop-runtime.nix` with:

```nix
{
  config,
  lib,
  pkgs,
  ...
}:

{
  imports = [
    ./forge-operator-desktop.nix
  ];

  boot = {
    plymouth = {
      enable = lib.mkDefault true;
      theme = lib.mkDefault "bgrt";
      logo = lib.mkDefault ../../../apps/desktop/public/brand/forge-start-button.png;
    };
    kernelParams = lib.mkDefault [
      "quiet"
      "splash"
      "loglevel=3"
      "udev.log_level=3"
    ];
  };

  programs.regreet = {
    enable = lib.mkDefault true;
    extraCss = ''
      window {
        background: #05070a;
      }

      box {
        color: #d8f7ff;
      }
    '';
    settings = {
      background = {
        path = ../../../apps/desktop/public/brand/forge-horizontal.png;
        fit = "Contain";
      };
      GTK = {
        application_prefer_dark_theme = true;
      };
    };
  };

  services.greetd = {
    enable = lib.mkDefault true;
    restart = lib.mkDefault true;
  };

  services.displayManager.autoLogin.enable = lib.mkForce false;
  services.getty.autologinUser = lib.mkForce null;

  environment.etc."forge/native-desktop-runtime.env".text = ''
    FORGE_NATIVE_DESKTOP_RUNTIME=true
    FORGE_NATIVE_DESKTOP_BOOT_SPLASH=plymouth
    FORGE_NATIVE_DESKTOP_LOGIN=greetd-regreet
    FORGE_NATIVE_DESKTOP_DEFAULT_SESSION=forge-operator
    FORGE_NATIVE_DESKTOP_AUTOLOGIN=false
    FORGE_NATIVE_DESKTOP_TTY_FALLBACK=true
  '';

  assertions = [
    {
      assertion = config.boot.plymouth.enable == true;
      message = "FORGE native desktop runtime requires Plymouth boot splash.";
    }
    {
      assertion = config.services.greetd.enable == true;
      message = "FORGE native desktop runtime requires greetd graphical login.";
    }
    {
      assertion = config.programs.regreet.enable == true;
      message = "FORGE native desktop runtime requires ReGreet graphical greeter.";
    }
    {
      assertion = config.services.displayManager.autoLogin.enable == false;
      message = "FORGE native desktop runtime must not enable autologin.";
    }
  ];
}
```

- [ ] **Step 2: Expose the profile in flake outputs**

Add to `flake.nix` `nixosModules`:

```nix
forge-native-desktop-runtime = import ./nix/nixos/profiles/forge-native-desktop-runtime.nix;
```

- [ ] **Step 3: Make canonical VM import native profile**

In `nix/nixos/configurations/forge-operator-vm.nix`, replace:

```nix
../profiles/forge-operator-desktop.nix
```

with:

```nix
../profiles/forge-native-desktop-runtime.nix
```

Update comments and help text so the normal flow is graphical login to FORGE desktop, while manual `forge-operator-session` remains recovery.

### Task 2: Add Static Native Desktop Checks

**Files:**
- Create: `nix/checks/forge-native-desktop-runtime.nix`
- Modify: `flake.nix`
- Modify: `nix/checks/forge-operator-vm.nix`

- [ ] **Step 1: Add native desktop check**

Create `nix/checks/forge-native-desktop-runtime.nix` that asserts:

```bash
profile="$src/nix/nixos/profiles/forge-native-desktop-runtime.nix"
vm="$src/nix/nixos/configurations/forge-operator-vm.nix"
runbook="$src/docs/runbooks/forge_operator_desktop_vm.md"

test -f "$profile"
grep -F './forge-operator-desktop.nix' "$profile"
grep -F 'boot.plymouth = {' "$profile"
grep -F 'theme = lib.mkDefault "bgrt";' "$profile"
grep -F 'programs.regreet = {' "$profile"
grep -F 'services.greetd = {' "$profile"
grep -F 'autoLogin.enable = lib.mkForce false;' "$profile"
grep -F 'services.getty.autologinUser = lib.mkForce null;' "$profile"
grep -F 'FORGE_NATIVE_DESKTOP_RUNTIME=true' "$profile"
grep -F '../profiles/forge-native-desktop-runtime.nix' "$vm"
grep -F 'FORGE-OS Runtime boot splash' "$runbook"
grep -F 'graphical password login' "$runbook"
```

Also fail if the profile contains:

```bash
autoLogin.enable = true|autologinUser = "[^"]+"|services.openssh.enable = true|nixos-rebuild|systemctl restart|systemctl stop|systemctl start|modprobe|rmmod|reboot|shutdown|LoadModel|UnloadModel|GenerateStream|rm -rf
```

- [ ] **Step 2: Wire check into flake**

Add:

```nix
forge-native-desktop-runtime = pkgs.callPackage ./nix/checks/forge-native-desktop-runtime.nix { };
```

- [ ] **Step 3: Update VM check**

Update `nix/checks/forge-operator-vm.nix` to expect:

```bash
grep -F '../profiles/forge-native-desktop-runtime.nix' "$config"
grep -F 'forge-native-desktop-runtime.nix' "$flake"
```

Keep existing safety assertions for local bind, safe mode, no SSH default, no autologin, and no host mutation.

### Task 3: Update Docs and Punch List

**Files:**
- Modify: `docs/runbooks/forge_operator_desktop_vm.md`
- Modify: `docs/operations/operator_desktop.md`
- Modify: `docs/reports/FORGE_PUNCHLIST.md`

- [ ] **Step 1: Update runbook status**

Change the runbook status to native desktop runtime bring-up and document:

```text
Power on -> FORGE-OS Runtime boot splash -> graphical password login -> FORGE native desktop.
```

Keep a recovery section for TTY/manual `forge-operator-session`.

- [ ] **Step 2: Update operator desktop operations doc**

Describe the native runtime as the canonical VM path and the manual session as recovery/fallback.

- [ ] **Step 3: Update punch list**

Mark the local model loop items that were verified and add/check the native desktop runtime item:

```markdown
- [x] Native desktop runtime spec drafted
- [x] Canonical VM imports native desktop runtime profile
- [ ] VM boot evidence: Plymouth/regreet/FORGE desktop screenshot
```

### Task 4: Verify

**Files:**
- No source changes unless verification exposes a defect.

- [ ] **Step 1: Run formatting/static checks**

Run:

```bash
git diff --check
nix build --extra-experimental-features 'nix-command flakes' .#checks.x86_64-linux.forge-native-desktop-runtime
nix build --extra-experimental-features 'nix-command flakes' .#checks.x86_64-linux.forge-operator-vm
nix build --extra-experimental-features 'nix-command flakes' .#checks.x86_64-linux.forge-operator-desktop
```

- [ ] **Step 2: Build VM target**

Run:

```bash
nix build --extra-experimental-features 'nix-command flakes' .#nixosConfigurations.forge-operator-vm.config.system.build.vm --no-link
```

- [ ] **Step 3: Verify active local VM if available**

If the `FORGE-OS` VirtualBox VM is available, update its `/etc/nixos/configuration.nix` import to the native profile, switch the system, reboot, and verify:

```text
Plymouth/boot splash appears
graphical login appears
operator password login starts FORGE desktop
forge-core health is available
modelruntime remains governed/local
TTY fallback still reachable
```

Do not mark screenshot evidence complete unless screenshots/logs are captured.
