{
  lib,
  stdenv,
  forgeDesktopShell,
}:

let
  containsTauriBinary = forgeDesktopShell.passthru.containsTauriBinary or false;
in
stdenv.mkDerivation {
  name = "forge-desktop-shell-wrapper-check";

  dontUnpack = true;
  dontConfigure = true;
  dontBuild = true;

  installPhase = ''
    runHook preInstall

    wrapper="${forgeDesktopShell}/bin/forge-desktop-shell"
    script="$wrapper"
    if [ -e "${forgeDesktopShell}/bin/.forge-desktop-shell-wrapped" ]; then
      script="${forgeDesktopShell}/bin/.forge-desktop-shell-wrapped"
    fi

    test -x "$wrapper"
    test -x "$script"

    grep -F 'FORGE_SHELL_SESSION_ENABLED' "$script"
    grep -F 'FORGE_SHELL_MODE' "$script"
    grep -F 'fullscreen-shell' "$script"
    grep -F 'http://127.0.0.1:18492' "$script"
    grep -F 'VITE_FORGE_API_URL' "$script"
    grep -F 'FORGE_CORE_URL' "$script"
    grep -F 'FORGE_SHELL_SAFE_MODE' "$script"
    grep -F 'FORGE_SHELL_FULLSCREEN' "$script"
    grep -F 'FORGE_SHELL_HOST_MUTATION' "$script"
    grep -F 'FORGE_SHELL_DIRECT_SYSTEM_CONTROL' "$script"
    grep -F 'FORGE_SHELL_MODEL_MUTATION' "$script"
    grep -F 'FORGE_SHELL_SEMANTIC_MEMORY_WRITE' "$script"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY' "$script"
    grep -F 'FORGE_DESKTOP_SHELL_BINARY' "$script"

    ${lib.optionalString containsTauriBinary ''
      test -x "${forgeDesktopShell}/bin/forge_desktop"
      if grep -F 'not fully Nix-packaged yet' "$script"; then
        echo "real forge-desktop-shell package still contains placeholder text" >&2
        exit 1
      fi
    ''}

    ${lib.optionalString (!containsTauriBinary) ''
      grep -F 'not fully Nix-packaged yet' "$script"
      grep -F 'Cargo/Tauri/WebKit build integration in Nix' "$script"
    ''}

    fake="$TMPDIR/fake-forge-desktop"
    printf '%s\n' \
      '#!/bin/sh' \
      'echo "fake-desktop:$FORGE_SHELL_SESSION_ENABLED:$FORGE_CORE_URL:$*"' \
      > "$fake"
    chmod +x "$fake"
    FORGE_DESKTOP_SHELL_BINARY="$fake" FORGE_CORE_URL=http://127.0.0.1:19999 "$wrapper" alpha beta > "$TMPDIR/desktop-override.out"
    grep -F 'fake-desktop:true:http://127.0.0.1:19999:alpha beta' "$TMPDIR/desktop-override.out"

    ${lib.optionalString (!containsTauriBinary) ''
      set +e
      FORGE_CORE_URL=http://127.0.0.1:19998 "$wrapper" > "$TMPDIR/desktop-fail.out" 2> "$TMPDIR/desktop-fail.err"
      status=$?
      set -e
      test "$status" -eq 1
      grep -F 'FORGE desktop shell is not fully Nix-packaged yet.' "$TMPDIR/desktop-fail.err"
      grep -F 'http://127.0.0.1:19998' "$TMPDIR/desktop-fail.err"
    ''}

    forbidden='systemctl|nixos-rebuild|modprobe|rmmod|reboot|shutdown|apt-get|dnf|zypper|pacman|LoadModel|UnloadModel|GenerateStream|semantic memory write|os.RemoveAll|rm -rf'
    if grep -E "$forbidden" "$wrapper" "$script"; then
      echo "forbidden host/runtime mutation text found in forge-desktop-shell wrapper" >&2
      exit 1
    fi

    mkdir -p "$out"
    echo "ok" > "$out/result"
    runHook postInstall
  '';

  meta = with lib; {
    description = "Static safety checks for the FORGE desktop shell wrapper";
    license = licenses.mit;
    platforms = platforms.unix;
  };
}
