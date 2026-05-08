{ lib, writeShellApplication }:

# FORGE graphical shell session wrapper.
#
# Phase G2 intentionally does not package the Tauri desktop binary yet. This
# command sets the governed shell-session environment, launches an existing
# local forge_desktop binary when available, and otherwise fails loudly with
# exact next steps.
writeShellApplication {
  name = "forge-shell-session";

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

        if [ -n "''${FORGE_SHELL_BINARY:-}" ]; then
          if [ -x "$FORGE_SHELL_BINARY" ]; then
            exec "$FORGE_SHELL_BINARY" "$@"
          fi
          echo "FORGE_SHELL_BINARY is set but is not executable: $FORGE_SHELL_BINARY" >&2
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

        cat >&2 <<EOF
    FORGE graphical shell binary is not available.

    The forge-shell-session wrapper is installed and safe-mode environment is set,
    but Phase G2 does not yet Nix-package the Tauri desktop binary.

    Expected binary locations:
      $repo_root/apps/desktop/src-tauri/target/release/forge_desktop
      $repo_root/apps/desktop/src-tauri/target/debug/forge_desktop

    Build or launch the current desktop shell from the repository:
      npm run build:desktop
      npm run desktop

    Then retry from the repository root:
      nix run .#forge-shell-session

    Override the binary explicitly when needed:
      FORGE_SHELL_BINARY=/path/to/forge_desktop forge-shell-session

    Current safe shell settings:
      FORGE_CORE_URL=$FORGE_CORE_URL
      FORGE_SHELL_MODE=$FORGE_SHELL_MODE
      FORGE_SHELL_SAFE_MODE=$FORGE_SHELL_SAFE_MODE
      FORGE_SHELL_FULLSCREEN=$FORGE_SHELL_FULLSCREEN
    EOF
        exit 1
  '';

  meta = with lib; {
    description = "Safe FORGE graphical shell session launcher";
    license = licenses.mit;
    mainProgram = "forge-shell-session";
    platforms = platforms.unix;
  };
}
