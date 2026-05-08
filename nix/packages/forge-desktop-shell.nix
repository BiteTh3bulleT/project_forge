{
  lib,
  writeShellApplication,
}:

let
  app = writeShellApplication {
    name = "forge-desktop-shell";

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

      if [ -n "''${FORGE_DESKTOP_SHELL_BINARY:-}" ]; then
        if [ -x "$FORGE_DESKTOP_SHELL_BINARY" ]; then
          exec "$FORGE_DESKTOP_SHELL_BINARY" "$@"
        fi
        echo "FORGE_DESKTOP_SHELL_BINARY is set but is not executable: $FORGE_DESKTOP_SHELL_BINARY" >&2
        exit 1
      fi

      printf '%s\n' \
        "FORGE desktop shell is not fully Nix-packaged yet." \
        "" \
        "This G3 package exposes the stable forge-desktop-shell command and safe" \
        "shell-mode environment defaults, but it does not contain a Nix-built" \
        "apps/desktop/src-tauri forge_desktop binary." \
        "" \
        "Known limitation:" \
        "  Full Tauri packaging still needs npm dependency vendoring plus" \
        "  Cargo/Tauri/WebKit build integration in Nix. This package fails loudly" \
        "  at runtime instead of pretending that binary packaging is complete." \
        "" \
        "Current local build path:" \
        "  npm -w @forge/desktop run tauri -- build" \
        "" \
        "Then launch through the session wrapper:" \
        "  FORGE_SHELL_BINARY=/path/to/forge_desktop forge-shell-session" \
        "" \
        "Or launch this stable command with an explicit desktop binary:" \
        "  FORGE_DESKTOP_SHELL_BINARY=/path/to/forge_desktop forge-desktop-shell" \
        "" \
        "Current safe shell settings:" \
        "  FORGE_CORE_URL=$FORGE_CORE_URL" \
        "  FORGE_SHELL_MODE=$FORGE_SHELL_MODE" \
        "  FORGE_SHELL_SAFE_MODE=$FORGE_SHELL_SAFE_MODE" \
        "  FORGE_SHELL_FULLSCREEN=$FORGE_SHELL_FULLSCREEN" >&2
      exit 1
    '';

    meta = with lib; {
      description = "Stable FORGE desktop shell launcher placeholder for the Tauri shell";
      license = licenses.mit;
      mainProgram = "forge-desktop-shell";
      platforms = platforms.unix;
    };
  };
in
app.overrideAttrs (old: {
  passthru = (old.passthru or { }) // {
    containsTauriBinary = false;
  };
})
