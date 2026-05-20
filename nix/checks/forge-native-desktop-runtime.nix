{
  lib,
  runCommand,
}:

runCommand "forge-native-desktop-runtime-check"
  {
    src = lib.cleanSource ../..;
  }
  ''
    set -euo pipefail

    flake="$src/flake.nix"
    profile="$src/nix/nixos/profiles/forge-native-desktop-runtime.nix"
    vm="$src/nix/nixos/configurations/forge-operator-vm.nix"

    test -f "$profile"
    grep -F './forge-operator-desktop.nix' "$profile"
    grep -F 'boot = {' "$profile"
    grep -F 'plymouth = {' "$profile"
    grep -F 'theme = lib.mkDefault "bgrt";' "$profile"
    grep -F 'forge-start-button.png' "$profile"
    grep -F 'renderProfile = "vm-safe";' "$profile"
    grep -F 'bootLogin = false;' "$profile"
    grep -F 'emptyDesktopOnBoot = true;' "$profile"
    grep -F 'programs.regreet = {' "$profile"
    grep -F 'forge-horizontal.png' "$profile"
    grep -F 'services.greetd = {' "$profile"
    grep -F 'enable = lib.mkDefault true;' "$profile"
    grep -F 'autoLogin.enable = lib.mkForce false;' "$profile"
    grep -F 'autoLogin.user = lib.mkForce null;' "$profile"
    grep -F 'services.getty.autologinUser = lib.mkForce null;' "$profile"
    grep -F 'defaultSession = lib.mkDefault "forge-operator";' "$profile"
    grep -F 'sessionPackages = [ forgeOperatorSession ];' "$profile"
    grep -F 'FORGE_NATIVE_DESKTOP_RUNTIME=true' "$profile"
    grep -F 'FORGE_NATIVE_DESKTOP_LOGIN=greetd-regreet' "$profile"
    grep -F 'FORGE_NATIVE_DESKTOP_DEFAULT_SESSION=forge-operator' "$profile"
    grep -F 'FORGE_NATIVE_DESKTOP_AUTOLOGIN=false' "$profile"
    grep -F 'FORGE_NATIVE_DESKTOP_TTY_FALLBACK=true' "$profile"
    grep -F '!(config.services.greetd.settings ? initial_session)' "$profile"

    grep -F '../profiles/forge-native-desktop-runtime.nix' "$vm"
    grep -F 'forge-native-desktop-runtime = import ./nix/nixos/profiles/forge-native-desktop-runtime.nix;' "$flake"
    grep -F 'forge-native-desktop-runtime = pkgs.callPackage ./nix/checks/forge-native-desktop-runtime.nix { };' "$flake"

    forbidden='autoLogin.enable = true|autologinUser = "[^"]+"|initial_session =|services.openssh.enable = true|nixos-rebuild|systemctl restart|systemctl stop|systemctl start|modprobe|rmmod|reboot|shutdown|LoadModel|UnloadModel|GenerateStream|rm -rf'
    if grep -RE "$forbidden" "$profile"; then
      echo "forbidden unsafe native desktop runtime default found" >&2
      exit 1
    fi

    touch "$out"
  ''
