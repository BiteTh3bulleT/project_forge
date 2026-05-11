{
  lib,
  stdenv,
}:

stdenv.mkDerivation {
  name = "forge-vbox-graphics-test-profile-check";

  src = lib.cleanSource ../..;
  sourceRoot = ".";
  dontConfigure = true;
  dontBuild = true;

  installPhase = ''
    runHook preInstall

    profile="$src/nix/nixos/profiles/forge-vbox-graphics-test.nix"
    readme="$src/nix/nixos/profiles/README.md"
    runbook="$src/docs/operations/virtualbox_forge_shell_test.md"

    test -f "$profile"
    test -f "$readme"
    test -f "$runbook"

    grep -F 'TEST PROFILE ONLY' "$profile"
    grep -F 'OPT-IN ONLY' "$profile"
    grep -F 'VIRTUALBOX/MINIMAL NIXOS GRAPHICS BRING-UP' "$profile"
    grep -F 'forge-wayland-session' "$profile"
    grep -F 'forge-shell-session' "$profile"
    grep -F 'forge-desktop-shell' "$profile"
    grep -F 'cage' "$profile"
    grep -F 'services.displayManager.autoLogin.enable = lib.mkDefault false;' "$profile"
    grep -F 'enable = lib.mkDefault true;' "$profile"
    grep -F 'autoStart = lib.mkDefault false;' "$profile"
    grep -F 'WEBKIT_DISABLE_DMABUF_RENDERER = "1"' "$profile"
    grep -F 'environment.systemPackages' "$profile"

    grep -F 'nix run .#forge-wayland-session' "$runbook"
    grep -F 'forge-wayland-session' "$runbook"
    grep -F 'VMSVGA' "$runbook"
    grep -F '128 MB' "$runbook"
    grep -F 'TTY' "$runbook"
    grep -F 'disable the test profile' "$runbook"
    grep -F 'NixOS generations' "$runbook"

    forbidden='(^|[^A-Za-z])(kde|KDE|plasma|Plasma|gnome|GNOME|xfce|XFCE)([^A-Za-z]|$)|autologin = true|autoLogin.enable = true|xrdp|vnc|systemctl|nixos-rebuild|modprobe|rmmod|reboot|shutdown|LoadModel|UnloadModel|GenerateStream|semantic memory write|os.RemoveAll|rm -rf'
    if grep -E "$forbidden" "$profile"; then
      echo "forbidden desktop/autologin/host-mutation text found in VirtualBox graphics test profile" >&2
      exit 1
    fi

    mkdir -p "$out"
    echo "ok" > "$out/result"
    runHook postInstall
  '';

  meta = with lib; {
    description = "Static safety checks for the FORGE VirtualBox graphics test profile";
    license = licenses.mit;
    platforms = platforms.linux;
  };
}
