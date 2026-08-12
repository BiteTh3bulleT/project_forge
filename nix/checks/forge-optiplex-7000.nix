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
    grep -F '"smuxo/smuxoAI:0.8b"' "$config"
    grep -F 'OLLAMA_MAX_LOADED_MODELS = "1";' "$config"
    grep -F 'OLLAMA_NUM_PARALLEL = "1";' "$config"
    grep -F 'autoStart = false;' "$config"
    grep -F 'autoLogin.enable = lib.mkForce false;' "$config"
    grep -F 'PasswordAuthentication = false;' "$config"
    grep -F 'PermitRootLogin = "no";' "$config"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$config"
    grep -F 'forge-optiplex-7000 = nixpkgs.lib.nixosSystem' "$flake"

    forbidden='boot.loader.grub.devices|virtualisation.virtualbox|autoLogin.enable = true|autologinUser = "[^"]+"|bindHost = "0.0.0.0"|PasswordAuthentication = true|PermitRootLogin = "yes"|nixos-install|parted|mkfs|wipefs|rm -rf'
    if grep -E "$forbidden" "$config"; then
      echo "forbidden VM, installer, mutation, or unsafe host default found" >&2
      exit 1
    fi

    touch "$out"
  ''
