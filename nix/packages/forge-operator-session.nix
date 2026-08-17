{
  lib,
  symlinkJoin,
  writeShellApplication,
  forge-shell-session,
  labwc ? null,
  polkitAgent ? null,
  notificationDaemon ? null,
  safeMode ? true,
  hostMutation ? false,
  directSystemControl ? false,
  modelMutation ? false,
  semanticMemoryWrite ? false,
}:

let
  boolString = value: if value then "true" else "false";
  defaultCompositor = if labwc != null then "${labwc}/bin/labwc" else "";
  shellSession = "${forge-shell-session}/bin/forge-shell-session";
  startup = writeShellApplication {
    name = "forge-operator-startup";
    text = ''
      set -euo pipefail
      ${lib.optionalString (polkitAgent != null) ''
        if [ -x "${polkitAgent}/libexec/polkit-gnome-authentication-agent-1" ]; then
          "${polkitAgent}/libexec/polkit-gnome-authentication-agent-1" &
        fi
      ''}
      ${lib.optionalString (notificationDaemon != null) ''
        if [ -x "${notificationDaemon}/bin/mako" ]; then
          "${notificationDaemon}/bin/mako" &
        fi
      ''}
      exec "${shellSession}" "$@"
    '';
  };
  launcher = writeShellApplication {
    name = "forge-operator-session";

    text = ''
          set -euo pipefail

          export FORGE_SHELL_SESSION_ENABLED=true
          export FORGE_SHELL_MODE=operator-desktop
          export FORGE_CORE_URL="''${FORGE_CORE_URL:-http://127.0.0.1:18492}"
          export VITE_FORGE_API_URL="''${VITE_FORGE_API_URL:-$FORGE_CORE_URL}"
          export FORGE_DATA_DIR="''${FORGE_DATA_DIR:-/forge/data}"
          export FORGE_WORKSPACE_DIR="''${FORGE_WORKSPACE_DIR:-/forge/workspaces/default}"
          export FORGE_API_TOKEN_FILE="''${FORGE_API_TOKEN_FILE:-$FORGE_DATA_DIR/auth/api_token}"
          export FORGE_SHELL_SAFE_MODE=${boolString safeMode}
          export FORGE_SHELL_FULLSCREEN=false
          export FORGE_SHELL_HOST_MUTATION=${boolString hostMutation}
          export FORGE_SHELL_DIRECT_SYSTEM_CONTROL=${boolString directSystemControl}
          export FORGE_SHELL_MODEL_MUTATION=${boolString modelMutation}
          export FORGE_SHELL_SEMANTIC_MEMORY_WRITE=${boolString semanticMemoryWrite}
          export FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false
          export FORGE_RENDER_PROFILE="''${FORGE_RENDER_PROFILE:-vm-safe}"
          export VITE_FORGE_RENDER_PROFILE="''${VITE_FORGE_RENDER_PROFILE:-$FORGE_RENDER_PROFILE}"
          export FORGE_SHELL_DISPLAY_BACKEND=wayland
          export FORGE_SHELL_COMPOSITOR=labwc
          export XDG_SESSION_TYPE=wayland
          export WEBKIT_DISABLE_DMABUF_RENDERER="''${WEBKIT_DISABLE_DMABUF_RENDERER:-1}"
          unset FORGE_SHELL_BINARY
          unset FORGE_REPO_ROOT

          compositor="${defaultCompositor}"
          session_startup="${startup}/bin/forge-operator-startup"
          labwc_config_dir="''${XDG_RUNTIME_DIR:-/tmp}/forge-operator-labwc"
          labwc_config_file="$labwc_config_dir/rc.xml"

          if [ -z "$compositor" ]; then
            echo "FORGE operator compositor is not configured; install or pass labwc when building forge-operator-session." >&2
            exit 1
          fi

          if [ ! -x "$compositor" ]; then
            echo "FORGE operator compositor is not executable: $compositor" >&2
            exit 1
          fi

          if [ ! -x "$session_startup" ]; then
            echo "FORGE operator startup wrapper is not executable: $session_startup" >&2
            exit 1
          fi

          mkdir -p "$labwc_config_dir"
          export FORGE_OPERATOR_LABWC_CONFIG_DIR="$labwc_config_dir"
          export FORGE_OPERATOR_DESKTOP_LOCKED=true
          cat > "$labwc_config_file" <<'EOF'
      <?xml version="1.0" encoding="UTF-8"?>
      <labwc_config>
        <core>
          <decoration>server</decoration>
        </core>
        <keyboard>
          <default />
        </keyboard>
        <mouse>
          <default />
        </mouse>
        <windowRules>
          <!-- FORGE is the desktop canvas, not a peer application window.
               The Tauri shell sizes itself to the active output; Labwc keeps
               that surface fixed below the native windows it launches. -->
          <!-- The packaged Wayland binary currently reports forge_desktop.
               Keep the Tauri bundle identifier as a compatibility rule for
               runtimes that expose the configured application identifier. -->
          <windowRule identifier="forge_desktop" serverDecoration="no">
            <skipTaskbar>yes</skipTaskbar>
            <skipWindowSwitcher>yes</skipWindowSwitcher>
            <fixedPosition>yes</fixedPosition>
            <action name="MoveTo" x="0" y="0" />
            <action name="ToggleAlwaysOnBottom" />
          </windowRule>
          <windowRule identifier="dev.forge.workshop" serverDecoration="no">
            <skipTaskbar>yes</skipTaskbar>
            <skipWindowSwitcher>yes</skipWindowSwitcher>
            <fixedPosition>yes</fixedPosition>
            <action name="MoveTo" x="0" y="0" />
            <action name="ToggleAlwaysOnBottom" />
          </windowRule>
        </windowRules>
      </labwc_config>
      EOF

          exec "$compositor" --config "$labwc_config_file" --startup "$session_startup" "$@"
    '';

    meta = with lib; {
      description = "Opt-in FORGE operator desktop session launcher using labwc";
      license = licenses.mit;
      mainProgram = "forge-operator-session";
      platforms = platforms.linux;
    };
  };
in
symlinkJoin {
  name = "forge-operator-session";
  paths = [
    launcher
    startup
  ];

  postBuild = ''
        mkdir -p "$out/share/wayland-sessions"
        cat > "$out/share/wayland-sessions/forge-operator.desktop" <<EOF
    [Desktop Entry]
    Name=FORGE Operator
    Comment=FORGE native desktop session
    Type=Application
    DesktopNames=FORGE
    Exec=$out/bin/forge-operator-session
    X-FORGE-Mode=operator-desktop
    X-FORGE-SafeMode=true
    X-FORGE-AutoStart=false
    EOF
  '';

  passthru.providedSessions = [ "forge-operator" ];
  meta = launcher.meta;
}
