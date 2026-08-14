{
  lib,
  runCommand,
}:

runCommand "forge-optiplex-7000-check"
  {
    src = lib.cleanSource ../..;
  }
  ''
    set -euo pipefail

    config="$src/nix/nixos/configurations/forge-optiplex-7000.nix"
    operator_session="$src/nix/packages/forge-operator-session.nix"
    desktop_main="$src/apps/desktop/src-tauri/src/main.rs"
    flake="$src/flake.nix"

    test -f "$config"
    grep -F 'systemd-boot.enable = true;' "$config"
    grep -F 'device = "/dev/disk/by-label/FORGE_ROOT";' "$config"
    grep -F 'device = "/dev/disk/by-label/FORGE_EFI";' "$config"
    grep -F 'device = "/dev/disk/by-label/FORGE_SWAP";' "$config"
    grep -F 'enableModelRuntime = true;' "$config"
    grep -F 'safeModeForceCPUOnly = true;' "$config"
    grep -F 'bindHost = "127.0.0.1";' "$config"
    grep -F 'OLLAMA_MODEL = "gemma3:1b-it-q4_K_M";' "$config"
    grep -F 'FORGE_MODEL_DEFAULT_BACKEND = "ollama_compat";' "$config"
    grep -F 'FORGE_MODEL_MAX_LOADED_MODELS = "1";' "$config"
    grep -F '"gemma3:1b-it-q4_K_M"' "$config"
    grep -F 'FORGE_MODEL_SECONDARY_ID=smuxo/smuxoAI:0.8b' "$config"
    grep -F 'loadModels = [ ];' "$config"
    grep -F 'OLLAMA_MAX_LOADED_MODELS = "1";' "$config"
    grep -F 'OLLAMA_NUM_PARALLEL = "1";' "$config"
    grep -F 'user = "ollama";' "$config"
    grep -F 'group = "forge";' "$config"
    grep -F 'd /forge/models/ollama 0750 ollama forge -' "$config"
    grep -F 'd /forge/models/ollama/models 0750 ollama forge -' "$config"
    grep -F 'autoStart = false;' "$config"
    grep -F 'renderProfile = "vm-safe";' "$config"
    grep -F 'emptyDesktopOnBoot = true;' "$config"
    grep -F 'forgeOperatorSession' "$config"
    grep -F 'forgeOperatorToolbelt' "$config"
    grep -F 'mode = "operator-desktop";' "$config"
    grep -F 'compositor = "labwc";' "$config"
    grep -F 'fullscreen = false;' "$config"
    grep -F 'package = pkgs.labwc;' "$config"
    grep -F 'sessionPackage = forgeOperatorSession;' "$config"
    grep -F 'sessionName = "forge-operator";' "$config"
    grep -F 'forge-operator-session' "$config"
    grep -F 'forgeWaylandSession' "$config"
    grep -F 'forgeOperatorToolbelt' "$config"
    grep -F 'pkgs.lswt' "$config"
    grep -F 'pkgs.wlrctl' "$config"
    grep -F 'WEBKIT_DISABLE_DMABUF_RENDERER = "1";' "$config"
    grep -F 'FORGE_RENDER_PROFILE = "vm-safe";' "$config"
    grep -F '<windowRule identifier="dev.forge.workshop" serverDecoration="no">' "$operator_session"
    grep -F '<skipWindowSwitcher>yes</skipWindowSwitcher>' "$operator_session"
    grep -F '<fixedPosition>yes</fixedPosition>' "$operator_session"
    grep -F '<action name="ToggleAlwaysOnBottom" />' "$operator_session"
    grep -F 'window.set_resizable(false)?;' "$desktop_main"
    grep -F 'autoLogin.enable = lib.mkForce false;' "$config"
    grep -F 'PasswordAuthentication = false;' "$config"
    grep -F 'PermitRootLogin = "no";' "$config"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$config"
    for required in foot pcmanfm mousepad firefox micro helix xarchiver; do
      grep -F '"'"$required"'"' "$src/nix/packages/forge-operator-toolbelt.nix"
    done
    grep -F 'forge-optiplex-7000 = nixpkgs.lib.nixosSystem' "$flake"

    forbidden='boot.loader.grub.devices|virtualisation.virtualbox|autoLogin.enable = true|autologinUser = "[^"]+"|bindHost = "0.0.0.0"|PasswordAuthentication = true|PermitRootLogin = "yes"|nixos-install|parted|mkfs|wipefs|rm -rf'
    if grep -E "$forbidden" "$config"; then
      echo "forbidden VM, installer, mutation, or unsafe host default found" >&2
      exit 1
    fi

    touch "$out"
  ''
