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
