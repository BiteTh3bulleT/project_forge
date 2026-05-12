{
  lib,
  runCommand,
}:

runCommand "forge-operator-desktop-profile-check"
  {
    src = lib.cleanSource ../..;
  }
  ''
    set -euo pipefail
    profile="$src/nix/nixos/profiles/forge-operator-desktop.nix"
    module="$src/nix/nixos/modules/forge-shell-session.nix"
    toolbelt="$src/nix/packages/forge-operator-toolbelt.nix"
    runbook="$src/docs/runbooks/forge_operator_desktop_vm.md"

    test -f "$profile"
    test -f "$toolbelt"
    grep -F 'forge-operator-session' "$profile"
    grep -F 'forgeOperatorToolbelt' "$profile"
    grep -F 'pkgs.labwc' "$profile"
    grep -F 'pkgs.lswt' "$profile"
    grep -F 'pkgs.wlrctl' "$profile"
    grep -F 'pkgs.foot' "$profile"
    grep -F 'pkgs.pcmanfm' "$profile"
    grep -F 'forge-wayland-session' "$profile"
    grep -F 'autoStart = lib.mkDefault false' "$profile"
    grep -F 'autoLogin.enable = lib.mkDefault false' "$profile"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY = "false"' "$profile"
    grep -F 'WEBKIT_DISABLE_DMABUF_RENDERER = "1"' "$profile"

    for required in foot pcmanfm firefox ollama btop sqlitebrowser jq yq ripgrep fd bat eza tree git lazygit curl wget nmap dnsutils iproute2 pciutils usbutils lsof strace micro xarchiver; do
      grep -F "\"$required\"" "$toolbelt"
    done
    for wrapper in forge-operator-ollama-status forge-operator-models forge-operator-btop forge-operator-lazygit forge-operator-core-logs forge-operator-core-status forge-operator-editor forge-operator-network-diagnostics forge-operator-hardware-diagnostics; do
      grep -F "name = \"$wrapper\"" "$toolbelt"
    done
    grep -F '"nvtop"' "$toolbelt"
    grep -F 'optionalGpuToolNames' "$toolbelt"

    grep -F 'operator-desktop' "$module"
    grep -F 'forge-operator-session' "$module"
    grep -F 'FORGE_SHELL_BINARY is disabled for operator-desktop sessions' "$src/nix/packages/forge-shell-session.nix"
    grep -F 'unset FORGE_SHELL_BINARY' "$src/nix/packages/forge-operator-session.nix"
    grep -F '/projectforge/nix/nixos/profiles/forge-operator-desktop.nix' "$runbook"

    forbidden='(^|[^A-Za-z])(kde|KDE|plasma|Plasma|gnome|GNOME|xfce|XFCE)([^A-Za-z]|$)|autologin = true|autoLogin.enable = true|xrdp|vnc|systemctl|nixos-rebuild|modprobe|rmmod|reboot|shutdown|LoadModel|UnloadModel|GenerateStream|semantic memory write|os.RemoveAll|rm -rf'
    if grep -RE "$forbidden" "$profile"; then
      echo "forbidden desktop/autologin/host-mutation text found in operator desktop profile" >&2
      exit 1
    fi
    if grep -RE 'curl .*install\.sh|freeform|arbitrary command|nixos-rebuild|systemctl restart|systemctl stop|systemctl start|curl .*-(X|request) *(POST|PUT|PATCH|DELETE)|ollama pull|ollama run|ollama serve|ollama rm|ollama create|ollama cp|ollama push|ollama stop|LoadModel|UnloadModel|GenerateStream|rm -rf|sudo |nix build .*--switch|nix profile|nix-env' "$toolbelt"; then
      echo "forbidden installer/freeform/mutation text found in operator toolbelt" >&2
      exit 1
    fi

    touch "$out"
  ''
