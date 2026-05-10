{
  lib,
  writeShellApplication,
  forge-shell-session,
  labwc ? null,
}:

let
  defaultCompositor = if labwc != null then "${labwc}/bin/labwc" else "";
  shellSession = "${forge-shell-session}/bin/forge-shell-session";
in
writeShellApplication {
  name = "forge-operator-session";

  text = ''
    set -euo pipefail

    export FORGE_SHELL_SESSION_ENABLED=true
    export FORGE_SHELL_MODE=operator-desktop
    export FORGE_CORE_URL="''${FORGE_CORE_URL:-http://127.0.0.1:18492}"
    export VITE_FORGE_API_URL="''${VITE_FORGE_API_URL:-$FORGE_CORE_URL}"
    export FORGE_SHELL_SAFE_MODE=true
    export FORGE_SHELL_FULLSCREEN=false
    export FORGE_SHELL_HOST_MUTATION=false
    export FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false
    export FORGE_SHELL_MODEL_MUTATION=false
    export FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false
    export FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false
    export FORGE_SHELL_DISPLAY_BACKEND=wayland
    export FORGE_SHELL_COMPOSITOR=labwc
    export XDG_SESSION_TYPE=wayland

    compositor="${defaultCompositor}"
    shell_session="${shellSession}"

    if [ -z "$compositor" ]; then
      echo "FORGE operator compositor is not configured; install or pass labwc when building forge-operator-session." >&2
      exit 1
    fi

    if [ ! -x "$compositor" ]; then
      echo "FORGE operator compositor is not executable: $compositor" >&2
      exit 1
    fi

    if [ ! -x "$shell_session" ]; then
      echo "FORGE shell session wrapper is not executable: $shell_session" >&2
      exit 1
    fi

    exec "$compositor" --startup "$shell_session" "$@"
  '';

  meta = with lib; {
    description = "Opt-in FORGE operator desktop session launcher using labwc";
    license = licenses.mit;
    mainProgram = "forge-operator-session";
    platforms = platforms.linux;
  };
}
