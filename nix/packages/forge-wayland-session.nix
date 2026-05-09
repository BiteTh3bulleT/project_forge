{
  lib,
  writeShellApplication,
  forge-shell-session,
  cage ? null,
}:

let
  defaultCompositor = if cage != null then "${cage}/bin/cage" else "";
  shellSession = "${forge-shell-session}/bin/forge-shell-session";
in
writeShellApplication {
  name = "forge-wayland-session";

  text = ''
    set -euo pipefail

    export FORGE_SHELL_SESSION_ENABLED=true
    export FORGE_SHELL_MODE=fullscreen-shell
    export FORGE_CORE_URL="''${FORGE_CORE_URL:-http://127.0.0.1:18492}"
    export VITE_FORGE_API_URL="''${VITE_FORGE_API_URL:-$FORGE_CORE_URL}"
    export FORGE_SHELL_SAFE_MODE=true
    export FORGE_SHELL_FULLSCREEN=true
    export FORGE_SHELL_HOST_MUTATION=false
    export FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false
    export FORGE_SHELL_MODEL_MUTATION=false
    export FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false
    export FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false
    export FORGE_SHELL_DISPLAY_BACKEND=wayland
    export FORGE_SHELL_COMPOSITOR=cage
    export XDG_SESSION_TYPE=wayland

    compositor="''${FORGE_WAYLAND_COMPOSITOR:-${defaultCompositor}}"
    shell_session="''${FORGE_SHELL_SESSION_BINARY:-${shellSession}}"

    if [ -z "$compositor" ]; then
      echo "FORGE Wayland compositor is not configured; install or pass cage when building forge-wayland-session." >&2
      exit 1
    fi

    if [ ! -x "$compositor" ]; then
      echo "FORGE Wayland compositor is not executable: $compositor" >&2
      exit 1
    fi

    if [ ! -x "$shell_session" ]; then
      echo "FORGE shell session wrapper is not executable: $shell_session" >&2
      exit 1
    fi

    exec "$compositor" -- "$shell_session" "$@"
  '';

  meta = with lib; {
    description = "Opt-in FORGE Wayland shell session launcher using Cage";
    license = licenses.mit;
    mainProgram = "forge-wayland-session";
    platforms = platforms.linux;
  };
}
