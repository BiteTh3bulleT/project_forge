{
  lib,
  runCommand,
}:

runCommand "forge-operator-vm-check"
  {
    src = lib.cleanSource ../..;
  }
  ''
    set -euo pipefail

    flake="$src/flake.nix"
    config="$src/nix/nixos/configurations/forge-operator-vm.nix"
    profile="$src/nix/nixos/profiles/forge-operator-desktop.nix"
    runbook="$src/docs/runbooks/forge_operator_desktop_vm.md"

    test -f "$config"
    grep -F 'forge-operator-vm = nixpkgs.lib.nixosSystem' "$flake"
    grep -F './nix/nixos/configurations/forge-operator-vm.nix' "$flake"

    grep -F '../modules/forge-os.nix' "$config"
    grep -F '../profiles/forge-operator-desktop.nix' "$config"
    grep -F 'boot.loader.grub.devices = lib.mkDefault [ "/dev/sda" ];' "$config"
    grep -F 'fileSystems."/" = {' "$config"
    grep -F 'forge.os = {' "$config"
    grep -F 'enable = true;' "$config"
    grep -F 'safeMode = lib.mkDefault true;' "$config"
    grep -F 'bindHost = lib.mkDefault "127.0.0.1";' "$config"
    grep -F 'enableModelRuntime = lib.mkDefault true;' "$config"
    grep -F 'OLLAMA_BASE_URL = lib.mkDefault "http://127.0.0.1:11434";' "$config"
    grep -F 'FORGE_MODEL_DEFAULT_BACKEND = lib.mkDefault "ollama_compat";' "$config"
    grep -F 'services.openssh.enable = lib.mkDefault false;' "$config"
    grep -F 'services.displayManager.autoLogin.enable = lib.mkForce false;' "$config"
    grep -F 'FORGE_SHELL_HOST_MUTATION=false' "$config"
    grep -F 'FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false' "$config"
    grep -F 'FORGE_SHELL_MODEL_MUTATION=false' "$config"
    grep -F 'FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false' "$config"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$config"
    grep -F 'unset FORGE_SHELL_BINARY' "$src/nix/packages/forge-operator-session.nix"
    grep -F 'virtualisation.vmVariant' "$config"
    grep -F 'forge-operator-session' "$config"

    grep -F 'forgeOperatorToolbelt' "$profile"
    grep -F 'nix build .#nixosConfigurations.forge-operator-vm.config.system.build.vm' "$runbook"

    forbidden='autoLogin.enable = true|services.openssh.enable = true|nixos-rebuild|systemctl restart|systemctl stop|systemctl start|modprobe|rmmod|reboot|shutdown|curl .*install\.sh|ollama pull|ollama run|ollama serve|LoadModel|UnloadModel|GenerateStream|rm -rf'
    if grep -RE "$forbidden" "$config"; then
      echo "forbidden unsafe VM default found in canonical operator VM config" >&2
      exit 1
    fi

    touch "$out"
  ''
