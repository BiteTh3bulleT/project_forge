{
  lib,
  writeShellApplication,
  forgeDesktopShell ? null,
  safeMode ? true,
  hostMutation ? false,
  directSystemControl ? false,
  modelMutation ? false,
  semanticMemoryWrite ? false,
}:

# FORGE graphical shell session wrapper.
#
# Prefer a real Nix-provided Tauri desktop package when one is available.
# Preserve the G2 local-binary fallback when that package is absent or still
# advertises passthru.containsTauriBinary = false.
let
  boolString = value: if value then "true" else "false";
  packagedDesktopShell =
    if forgeDesktopShell != null && (forgeDesktopShell.passthru.containsTauriBinary or false) then
      "${forgeDesktopShell}/bin/forge-desktop-shell"
    else
      "";
in
writeShellApplication {
  name = "forge-shell-session";

  text = ''
    set -euo pipefail

    export FORGE_SHELL_SESSION_ENABLED=true
    export FORGE_SHELL_MODE="''${FORGE_SHELL_MODE:-fullscreen-shell}"
    export FORGE_CORE_URL="''${FORGE_CORE_URL:-http://127.0.0.1:18492}"
    export VITE_FORGE_API_URL="''${VITE_FORGE_API_URL:-$FORGE_CORE_URL}"
    export FORGE_SHELL_SAFE_MODE=${boolString safeMode}
    export FORGE_SHELL_FULLSCREEN="''${FORGE_SHELL_FULLSCREEN:-true}"
    export FORGE_SHELL_HOST_MUTATION=${boolString hostMutation}
    export FORGE_SHELL_DIRECT_SYSTEM_CONTROL=${boolString directSystemControl}
    export FORGE_SHELL_MODEL_MUTATION=${boolString modelMutation}
    export FORGE_SHELL_SEMANTIC_MEMORY_WRITE=${boolString semanticMemoryWrite}
    export FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false

    if [ "$FORGE_SHELL_MODE" = "operator-desktop" ] && [ -n "''${FORGE_SHELL_BINARY:-}" ]; then
      echo "FORGE_SHELL_BINARY is disabled for operator-desktop sessions; use the Nix-packaged forge-desktop-shell." >&2
      exit 1
    fi

    if [ -n "''${FORGE_SHELL_BINARY:-}" ]; then
      if [ -x "$FORGE_SHELL_BINARY" ]; then
        exec "$FORGE_SHELL_BINARY" "$@"
      fi
      echo "FORGE_SHELL_BINARY is set but is not executable: $FORGE_SHELL_BINARY" >&2
      exit 1
    fi

    nix_desktop_shell="${packagedDesktopShell}"
    if [ -n "$nix_desktop_shell" ]; then
      if [ -x "$nix_desktop_shell" ]; then
        exec "$nix_desktop_shell" "$@"
      fi
      echo "Nix-provided FORGE desktop shell is not executable: $nix_desktop_shell" >&2
      exit 1
    fi

    if [ "$FORGE_SHELL_MODE" = "operator-desktop" ]; then
      echo "operator-desktop sessions require a Nix-packaged forge-desktop-shell; local binary fallback is disabled." >&2
      exit 1
    fi

    repo_root="''${FORGE_REPO_ROOT:-$PWD}"
    for candidate in \
      "$repo_root/apps/desktop/src-tauri/target/release/forge_desktop" \
      "$repo_root/apps/desktop/src-tauri/target/debug/forge_desktop"
    do
      if [ -x "$candidate" ]; then
        exec "$candidate" "$@"
      fi
    done

    printf '%s\n' \
      "FORGE graphical shell binary is not available." \
      "" \
      "The forge-shell-session wrapper is installed and safe-mode environment is set," \
      "but no Nix-built or local Tauri desktop binary was found." \
      "" \
      "Expected binary locations:" \
      "  $repo_root/apps/desktop/src-tauri/target/release/forge_desktop" \
      "  $repo_root/apps/desktop/src-tauri/target/debug/forge_desktop" \
      "" \
      "Build or launch the current desktop shell from the repository:" \
      "  npm -w @forge/desktop run tauri -- build" \
      "  npm run desktop" \
      "" \
      "Then retry from the repository root:" \
      "  nix run .#forge-shell-session" \
      "" \
      "Override the binary explicitly when needed:" \
      "  FORGE_SHELL_BINARY=/path/to/forge_desktop forge-shell-session" \
      "" \
      "Current safe shell settings:" \
      "  FORGE_CORE_URL=$FORGE_CORE_URL" \
      "  FORGE_SHELL_MODE=$FORGE_SHELL_MODE" \
      "  FORGE_SHELL_SAFE_MODE=$FORGE_SHELL_SAFE_MODE" \
      "  FORGE_SHELL_FULLSCREEN=$FORGE_SHELL_FULLSCREEN" >&2
    exit 1
  '';

  meta = with lib; {
    description = "Safe FORGE graphical shell session launcher";
    license = licenses.mit;
    mainProgram = "forge-shell-session";
    platforms = platforms.unix;
  };
}
