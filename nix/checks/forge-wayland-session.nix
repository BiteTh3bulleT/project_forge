{
  lib,
  stdenv,
  forge-wayland-session,
}:

stdenv.mkDerivation {
  name = "forge-wayland-session-wrapper-check";

  dontUnpack = true;
  dontConfigure = true;
  dontBuild = true;

  installPhase = ''
    runHook preInstall

    wrapper="${forge-wayland-session}/bin/forge-wayland-session"

    test -x "$wrapper"

    grep -F 'FORGE_SHELL_SESSION_ENABLED=true' "$wrapper"
    grep -F 'FORGE_SHELL_MODE=fullscreen-shell' "$wrapper"
    grep -F 'FORGE_CORE_URL="''${FORGE_CORE_URL:-http://127.0.0.1:18492}"' "$wrapper"
    grep -F 'VITE_FORGE_API_URL="''${VITE_FORGE_API_URL:-$FORGE_CORE_URL}"' "$wrapper"
    grep -F 'FORGE_SHELL_SAFE_MODE=true' "$wrapper"
    grep -F 'FORGE_SHELL_FULLSCREEN=true' "$wrapper"
    grep -F 'FORGE_SHELL_HOST_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false' "$wrapper"
    grep -F 'FORGE_SHELL_MODEL_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false' "$wrapper"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$wrapper"
    grep -F 'FORGE_WAYLAND_COMPOSITOR' "$wrapper"
    grep -F 'forge-shell-session' "$wrapper"
    grep -F 'exec "$compositor" -- "$shell_session" "$@"' "$wrapper"

    forbidden='systemctl|nixos-rebuild|modprobe|rmmod|reboot|shutdown|apt-get|dnf|zypper|pacman|LoadModel|UnloadModel|GenerateStream|os.RemoveAll|rm -rf'
    if grep -E "$forbidden" "$wrapper"; then
      echo "forbidden host/runtime mutation text found in forge-wayland-session wrapper" >&2
      exit 1
    fi

    fake_compositor="$TMPDIR/fake-cage"
    printf '%s\n' \
      '#!/bin/sh' \
      'test "$1" = "--"' \
      'shift' \
      'echo "fake-cage:$FORGE_SHELL_SESSION_ENABLED:$FORGE_CORE_URL:$1:$*"' \
      > "$fake_compositor"
    chmod +x "$fake_compositor"

    FORGE_WAYLAND_COMPOSITOR="$fake_compositor" \
      FORGE_CORE_URL=http://127.0.0.1:19994 \
      "$wrapper" arg1 arg2 > "$TMPDIR/wayland.out"
    grep -F 'fake-cage:true:http://127.0.0.1:19994:' "$TMPDIR/wayland.out"
    grep -F 'forge-shell-session' "$TMPDIR/wayland.out"
    grep -F 'arg1 arg2' "$TMPDIR/wayland.out"

    set +e
    FORGE_WAYLAND_COMPOSITOR="$TMPDIR/missing-cage" "$wrapper" > "$TMPDIR/wayland-fail.out" 2> "$TMPDIR/wayland-fail.err"
    status=$?
    set -e
    test "$status" -eq 1
    grep -F 'FORGE Wayland compositor is not executable:' "$TMPDIR/wayland-fail.err"

    mkdir -p "$out"
    echo "ok" > "$out/result"
    runHook postInstall
  '';

  meta = with lib; {
    description = "Static safety checks for the FORGE Wayland shell session wrapper";
    license = licenses.mit;
    platforms = platforms.linux;
  };
}
