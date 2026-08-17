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
    services_module="$src/nix/nixos/modules/forge-services.nix"
    operator_session="$src/nix/packages/forge-operator-session.nix"
    desktop_main="$src/apps/desktop/src-tauri/src/main.rs"
    desktop_app="$src/apps/desktop/src/App.tsx"
    flake="$src/flake.nix"

    test -f "$config"
    grep -F 'systemd-boot.enable = true;' "$config"
    grep -F 'device = "/dev/disk/by-label/FORGE_ROOT";' "$config"
    grep -F 'device = "/dev/disk/by-label/FORGE_EFI";' "$config"
    grep -F 'device = "/dev/disk/by-label/FORGE_SWAP";' "$config"
    grep -F 'ensureProfiles.profiles."forge-direct-link"' "$config"
    grep -F 'interface-name = "enp0s31f6";' "$config"
    grep -F 'address1 = "192.168.50.2/24";' "$config"
    grep -F 'method = "manual";' "$config"
    grep -F 'gateway = "";' "$config"
    grep -F 'dns = "";' "$config"
    grep -F 'ignore-auto-dns = true;' "$config"
    grep -F 'never-default = true;' "$config"
    grep -F 'ipv6.method = "disabled";' "$config"
    grep -F 'nftables = {' "$config"
    grep -F 'table inet forge_offline' "$config"
    grep -F 'type filter hook output priority 0; policy drop;' "$config"
    grep -F 'oifname "enp0s31f6" ip daddr 192.168.50.0/24 accept' "$config"
    grep -F 'iifname "enp0s31f6" ip saddr 192.168.50.0/24 tcp dport 22 accept' "$config"
    grep -F 'FORGE_OPTIPLEX_NETWORK_MODE=offline-direct' "$config"
    grep -F 'FORGE_OPTIPLEX_DEFAULT_ROUTE=false' "$config"
    grep -F 'FORGE_OPTIPLEX_EGRESS_POLICY=loopback-and-192.168.50.0/24-only' "$config"
    grep -F 'forge.storage.workspaceMode = "2770";' "$config"
    grep -F 'd /forge/workspaces/default/Projects 2770 operator forge -' "$config"
    grep -F 'enableModelRuntime = true;' "$config"
    grep -F 'safeModeForceCPUOnly = false;' "$config"
    grep -F 'FORGE_UNSAFE_TEST_MODE = "true";' "$config"
    grep -F 'bindHost = "127.0.0.1";' "$config"
    grep -F 'OLLAMA_MODEL = "qwen2.5:1.5b";' "$config"
    grep -F 'FORGE_MODEL_DEFAULT_BACKEND = "ollama_compat";' "$config"
    grep -F 'FORGE_MODEL_MAX_LOADED_MODELS = "1";' "$config"
    grep -F 'FORGE_OLLAMA_CHAT_NUM_CTX = "2048";' "$config"
    grep -F 'FORGE_OLLAMA_CHAT_NUM_PREDICT = "192";' "$config"
    grep -F 'FORGE_OLLAMA_CHAT_THINK = "false";' "$config"
    grep -F 'FORGE_MODEL_DEFAULT_ID=qwen2.5:1.5b' "$config"
    grep -F 'FORGE_MODEL_SECONDARY_ID=gemma3:1b-it-q4_K_M' "$config"
    grep -F 'path = [ pkgs.git ];' "$services_module"
    grep -F 'loadModels = [ ];' "$config"
    grep -F 'OLLAMA_MAX_LOADED_MODELS = "1";' "$config"
    grep -F 'OLLAMA_NUM_PARALLEL = "1";' "$config"
    grep -F 'user = "ollama";' "$config"
    grep -F 'group = "forge";' "$config"
    grep -F 'd /forge/models/ollama 0750 ollama forge -' "$config"
    grep -F 'd /forge/models/ollama/models 0750 ollama forge -' "$config"
    grep -F 'autoStart = false;' "$config"
    grep -F 'renderProfile = "default";' "$config"
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
    grep -F 'FORGE_RENDER_PROFILE = "default";' "$config"
    grep -F 'FORGE_SHELL_DIRECT_SYSTEM_CONTROL = "true";' "$config"
    grep -F 'FORGE_SHELL_MODEL_MUTATION = "true";' "$config"
    grep -F 'FORGE_SHELL_SEMANTIC_MEMORY_WRITE = "true";' "$config"
    grep -F 'FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD = "true";' "$config"
    grep -F 'FORGE_ENABLE_OPENAI_COMPAT_API = "true";' "$config"
    grep -F '<windowRule identifier="forge_desktop" serverDecoration="no">' "$operator_session"
    grep -F '<windowRule identifier="dev.forge.workshop" serverDecoration="no">' "$operator_session"
    grep -F '<skipWindowSwitcher>yes</skipWindowSwitcher>' "$operator_session"
    grep -F '<fixedPosition>yes</fixedPosition>' "$operator_session"
    grep -F '<action name="ToggleAlwaysOnBottom" />' "$operator_session"
    grep -F 'window.set_resizable(false)?;' "$desktop_main"
    grep -F '"exit_session" => spawn_operator_session_exit_command(),' "$desktop_main"
    grep -F 'requestShellSessionAction("exit_session")' "$desktop_app"
    grep -F 'autoLogin.enable = lib.mkForce false;' "$config"
    grep -F 'PasswordAuthentication = false;' "$config"
    grep -F 'PermitRootLogin = "no";' "$config"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$config"
    grep -F 'polkitAgent = pkgs.polkit_gnome;' "$config"
    grep -F 'org.freedesktop.login1.power-off' "$config"
    grep -F 'org.freedesktop.login1.reboot' "$config"
    grep -F 'org.freedesktop.NetworkManager.settings.modify.system' "$config"
    grep -F 'hardware.bluetooth = {' "$config"
    grep -F 'services.blueman.enable = true;' "$config"
    grep -F 'notificationDaemon = pkgs.mako;' "$config"
    grep -F 'printing = {' "$config"
    grep -F 'pipewire = {' "$config"
    grep -F 'hardware.sane = {' "$config"
    grep -F 'gvfs.enable = true;' "$config"
    grep -F 'udisks2.enable = true;' "$config"
    grep -F 'virtualisation.podman = {' "$config"
    grep -F 'dockerCompat = true;' "$config"
    grep -F 'nix-direnv.enable = true;' "$config"
    grep -F 'nix-ld.enable = true;' "$config"
    grep -F 'FORGE_WORKSPACE_DIR = "/forge/workspaces/default";' "$config"
    for required in \
      libreoffice thunderbird evince xournalpp gimp inkscape mpv keepassxc simple-scan pavucontrol \
      networkmanagerapplet wdisplays system-config-printer blueman lxappearance \
      vscodium golang.go rust-lang.rust-analyzer ms-python.python redhat.vscode-yaml jnoortheen.nix-ide \
      go gopls gotools delve nodejs_20 typescript-language-server rustc cargo rust-analyzer cargo-tauri \
      python3 uv ruff gcc clang gdb cmake ninja pkg-config shellcheck shfmt nil nixfmt \
      podman buildah skopeo restic borgbackup rsync p7zip unzip zip nvme-cli smartmontools
    do
      grep -F "$required" "$config"
    done
    for required in network-settings display-settings audio-settings printer-settings bluetooth-settings appearance-settings; do
      grep -F 'id: "'"$required"'"' "$desktop_main"
    done
    for required in foot pcmanfm mousepad firefox micro helix xarchiver; do
      grep -F '"'"$required"'"' "$src/nix/packages/forge-operator-toolbelt.nix"
    done
    grep -F 'forge-optiplex-7000 = nixpkgs.lib.nixosSystem' "$flake"

    forbidden='boot.loader.grub.devices|virtualisation.virtualbox|autoLogin.enable = true|autologinUser = "[^"]+"|bindHost = "0.0.0.0"|PasswordAuthentication = true|PermitRootLogin = "yes"|ipv4.method = "shared"|method = "auto"|nixos-install|parted|mkfs|wipefs|rm -rf'
    if grep -E "$forbidden" "$config"; then
      echo "forbidden VM, installer, mutation, or unsafe host default found" >&2
      exit 1
    fi

    touch "$out"
  ''
