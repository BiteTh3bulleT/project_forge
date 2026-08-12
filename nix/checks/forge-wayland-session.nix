{
  lib,
  stdenv,
  forge-wayland-session,
  callPackage,
  writeShellApplication,
}:

let
  fakeCage = writeShellApplication {
    name = "cage";
    text = ''
      test "$1" = "--"
      shift
      echo "fake-cage:$FORGE_SHELL_SESSION_ENABLED:$FORGE_CORE_URL:$1:$*"
    '';
  };
  fakeShellSession = writeShellApplication {
    name = "forge-shell-session";
    text = ''
      echo "fake-shell-session:$*"
    '';
  };
  testWaylandSession = callPackage ../packages/forge-wayland-session.nix {
    cage = fakeCage;
    forge-shell-session = fakeShellSession;
  };
in
stdenv.mkDerivation {
  name = "forge-wayland-session-wrapper-check";

  dontUnpack = true;
  dontConfigure = true;
  dontBuild = true;

  installPhase = ''
    runHook preInstall

    real_wrapper="${forge-wayland-session}/bin/forge-wayland-session"
    wrapper="${testWaylandSession}/bin/forge-wayland-session"

    test -x "$real_wrapper"
    test -x "$wrapper"

    grep -F 'FORGE_SHELL_SESSION_ENABLED=true' "$wrapper"
    grep -F 'FORGE_SHELL_MODE=fullscreen-shell' "$wrapper"
    grep -F 'FORGE_CORE_URL="''${FORGE_CORE_URL:-http://127.0.0.1:18492}"' "$wrapper"
    grep -F 'VITE_FORGE_API_URL="''${VITE_FORGE_API_URL:-$FORGE_CORE_URL}"' "$wrapper"
    grep -F 'FORGE_DATA_DIR="''${FORGE_DATA_DIR:-/forge/data}"' "$wrapper"
    grep -F 'FORGE_API_TOKEN_FILE="''${FORGE_API_TOKEN_FILE:-$FORGE_DATA_DIR/auth/api_token}"' "$wrapper"
    grep -F 'FORGE_SHELL_SAFE_MODE=true' "$wrapper"
    grep -F 'FORGE_SHELL_FULLSCREEN=true' "$wrapper"
    grep -F 'FORGE_SHELL_HOST_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false' "$wrapper"
    grep -F 'FORGE_SHELL_MODEL_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false' "$wrapper"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$wrapper"
    grep -F 'forge-shell-session' "$wrapper"
    grep -F 'exec "$compositor" -- "$shell_session" "$@"' "$wrapper"

    if grep -F 'FORGE_WAYLAND_COMPOSITOR' "$real_wrapper" "$wrapper"; then
      echo "forge-wayland-session must not accept compositor executable paths from ambient environment" >&2
      exit 1
    fi

    if grep -F 'FORGE_SHELL_SESSION_BINARY' "$real_wrapper" "$wrapper"; then
      echo "forge-wayland-session must not accept shell executable paths from ambient environment" >&2
      exit 1
    fi

    forbidden='systemctl|nixos-rebuild|modprobe|rmmod|reboot|shutdown|apt-get|dnf|zypper|pacman|LoadModel|UnloadModel|GenerateStream|os.RemoveAll|rm -rf'
    if grep -E "$forbidden" "$wrapper"; then
      echo "forbidden host/runtime mutation text found in forge-wayland-session wrapper" >&2
      exit 1
    fi

    fake_compositor="$TMPDIR/ambient-cage"
    printf '%s\n' \
      '#!/bin/sh' \
      'echo "ambient compositor override must not run" >&2' \
      'exit 42' \
      > "$fake_compositor"
    chmod +x "$fake_compositor"

    fake_shell_session="$TMPDIR/ambient-forge-shell-session"
    printf '%s\n' \
      '#!/bin/sh' \
      'echo "ambient shell override must not run" >&2' \
      'exit 43' \
      > "$fake_shell_session"
    chmod +x "$fake_shell_session"

    FORGE_WAYLAND_COMPOSITOR="$fake_compositor" \
      FORGE_SHELL_SESSION_BINARY="$fake_shell_session" \
      FORGE_CORE_URL=http://127.0.0.1:19994 \
      "$wrapper" arg1 arg2 > "$TMPDIR/wayland.out"
    grep -F 'fake-cage:true:http://127.0.0.1:19994:' "$TMPDIR/wayland.out"
    grep -F 'forge-shell-session' "$TMPDIR/wayland.out"
    grep -F 'arg1 arg2' "$TMPDIR/wayland.out"

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
