{
  lib,
  stdenv,
  forgeDesktopShell,
}:

assert (forgeDesktopShell.passthru.containsTauriBinary or false) == false;

stdenv.mkDerivation {
  name = "forge-desktop-shell-wrapper-check";

  dontUnpack = true;
  dontConfigure = true;
  dontBuild = true;

  installPhase = ''
    runHook preInstall

    wrapper="${forgeDesktopShell}/bin/forge-desktop-shell"

    test -x "$wrapper"

    grep -F 'FORGE_SHELL_SESSION_ENABLED=true' "$wrapper"
    grep -F 'FORGE_SHELL_MODE=fullscreen-shell' "$wrapper"
    grep -F 'http://127.0.0.1:18492' "$wrapper"
    grep -F 'VITE_FORGE_API_URL' "$wrapper"
    grep -F 'FORGE_SHELL_SAFE_MODE=true' "$wrapper"
    grep -F 'FORGE_SHELL_FULLSCREEN=true' "$wrapper"
    grep -F 'FORGE_SHELL_HOST_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false' "$wrapper"
    grep -F 'FORGE_SHELL_MODEL_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false' "$wrapper"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$wrapper"
    grep -F 'FORGE_DESKTOP_SHELL_BINARY' "$wrapper"
    grep -F 'not fully Nix-packaged yet' "$wrapper"
    grep -F 'Cargo/Tauri/WebKit build integration in Nix' "$wrapper"

    fake="$TMPDIR/fake-forge-desktop"
    printf '%s\n' \
      '#!/bin/sh' \
      'echo "fake-desktop:$FORGE_SHELL_SESSION_ENABLED:$FORGE_CORE_URL:$*"' \
      > "$fake"
    chmod +x "$fake"
    FORGE_DESKTOP_SHELL_BINARY="$fake" FORGE_CORE_URL=http://127.0.0.1:19999 "$wrapper" alpha beta > "$TMPDIR/desktop-override.out"
    grep -F 'fake-desktop:true:http://127.0.0.1:19999:alpha beta' "$TMPDIR/desktop-override.out"

    set +e
    FORGE_CORE_URL=http://127.0.0.1:19998 "$wrapper" > "$TMPDIR/desktop-fail.out" 2> "$TMPDIR/desktop-fail.err"
    status=$?
    set -e
    test "$status" -eq 1
    grep -F 'FORGE desktop shell is not fully Nix-packaged yet.' "$TMPDIR/desktop-fail.err"
    grep -F 'http://127.0.0.1:19998' "$TMPDIR/desktop-fail.err"

    forbidden='systemctl|nixos-rebuild|modprobe|rmmod|reboot|shutdown|apt-get|dnf|zypper|pacman|LoadModel|UnloadModel|GenerateStream|semantic memory write|os.RemoveAll|rm -rf'
    if grep -E "$forbidden" "$wrapper"; then
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
