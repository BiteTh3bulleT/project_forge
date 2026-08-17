{
  lib,
  stdenv,
  forge-operator-session,
  callPackage,
  writeShellApplication,
}:

let
  fakeLabwc = writeShellApplication {
    name = "labwc";
    text = ''
      test "$1" = "--config"
      config_file="$2"
      test -f "$config_file"
      grep -F '<windowRule identifier="forge_desktop" serverDecoration="no">' "$config_file"
      grep -F '<windowRule identifier="dev.forge.workshop" serverDecoration="no">' "$config_file"
      grep -F '<skipTaskbar>yes</skipTaskbar>' "$config_file"
      grep -F '<skipWindowSwitcher>yes</skipWindowSwitcher>' "$config_file"
      grep -F '<fixedPosition>yes</fixedPosition>' "$config_file"
      grep -F '<action name="ToggleAlwaysOnBottom" />' "$config_file"
      shift 2
      test "$1" = "--startup"
      shift
      echo "fake-labwc:$FORGE_SHELL_SESSION_ENABLED:$FORGE_SHELL_MODE:$FORGE_CORE_URL:''${FORGE_DATA_DIR:-unset}:''${FORGE_API_TOKEN_FILE:-unset}:''${FORGE_SHELL_BINARY:-unset}:$1:$*"
    '';
  };
  fakeShellSession = writeShellApplication {
    name = "forge-shell-session";
    text = ''
      echo "fake-shell-session:$*"
    '';
  };
  testOperatorSession = callPackage ../packages/forge-operator-session.nix {
    labwc = fakeLabwc;
    forge-shell-session = fakeShellSession;
  };
in
stdenv.mkDerivation {
  name = "forge-operator-session-wrapper-check";

  dontUnpack = true;
  dontConfigure = true;
  dontBuild = true;

  installPhase = ''
    runHook preInstall

    real_wrapper="${forge-operator-session}/bin/forge-operator-session"
    real_session="${forge-operator-session}/share/wayland-sessions/forge-operator.desktop"
    wrapper="${testOperatorSession}/bin/forge-operator-session"
    startup="${testOperatorSession}/bin/forge-operator-startup"

    test -x "$real_wrapper"
    test -f "$real_session"
    test -x "$wrapper"
    test -x "$startup"

    grep -F 'Name=FORGE Operator' "$real_session"
    grep -F 'DesktopNames=FORGE' "$real_session"
    grep -F "Exec=${forge-operator-session}/bin/forge-operator-session" "$real_session"
    grep -F 'X-FORGE-Mode=operator-desktop' "$real_session"
    grep -F 'X-FORGE-SafeMode=true' "$real_session"
    grep -F 'X-FORGE-AutoStart=false' "$real_session"

    grep -F 'FORGE_SHELL_MODE=operator-desktop' "$wrapper"
    grep -F 'FORGE_SHELL_FULLSCREEN=false' "$wrapper"
    grep -F 'FORGE_SHELL_SAFE_MODE=true' "$wrapper"
    grep -F 'FORGE_SHELL_HOST_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false' "$wrapper"
    grep -F 'FORGE_SHELL_MODEL_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false' "$wrapper"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$wrapper"
    grep -F 'FORGE_DATA_DIR="''${FORGE_DATA_DIR:-/forge/data}"' "$wrapper"
    grep -F 'FORGE_WORKSPACE_DIR="''${FORGE_WORKSPACE_DIR:-/forge/workspaces/default}"' "$wrapper"
    grep -F 'FORGE_API_TOKEN_FILE="''${FORGE_API_TOKEN_FILE:-$FORGE_DATA_DIR/auth/api_token}"' "$wrapper"
    grep -F 'FORGE_SHELL_COMPOSITOR=labwc' "$wrapper"
    grep -F 'WEBKIT_DISABLE_DMABUF_RENDERER="''${WEBKIT_DISABLE_DMABUF_RENDERER:-1}"' "$wrapper"
    grep -F 'unset FORGE_SHELL_BINARY' "$wrapper"
    grep -F 'unset FORGE_REPO_ROOT' "$wrapper"
    grep -F 'FORGE_OPERATOR_LABWC_CONFIG_DIR=' "$wrapper"
    grep -F 'FORGE_OPERATOR_DESKTOP_LOCKED=true' "$wrapper"
    grep -F 'labwc_config_file="$labwc_config_dir/rc.xml"' "$wrapper"
    grep -F '<windowRule identifier="forge_desktop" serverDecoration="no">' "$wrapper"
    grep -F '<windowRule identifier="dev.forge.workshop" serverDecoration="no">' "$wrapper"
    grep -F '<skipTaskbar>yes</skipTaskbar>' "$wrapper"
    grep -F '<skipWindowSwitcher>yes</skipWindowSwitcher>' "$wrapper"
    grep -F '<fixedPosition>yes</fixedPosition>' "$wrapper"
    grep -F '<action name="MoveTo" x="0" y="0" />' "$wrapper"
    grep -F '<action name="ToggleAlwaysOnBottom" />' "$wrapper"
    grep -F 'exec "$compositor" --config "$labwc_config_file" --startup "$session_startup" "$@"' "$wrapper"
    grep -F 'exec "${fakeShellSession}/bin/forge-shell-session" "$@"' "$startup"

    if grep -F 'FORGE_OPERATOR_COMPOSITOR' "$real_wrapper" "$wrapper"; then
      echo "forge-operator-session must not accept compositor executable paths from ambient environment" >&2
      exit 1
    fi

    forbidden='systemctl|nixos-rebuild|modprobe|rmmod|reboot|shutdown|apt-get|dnf|zypper|pacman|LoadModel|UnloadModel|GenerateStream|os.RemoveAll|rm -rf'
    if grep -E "$forbidden" "$wrapper"; then
      echo "forbidden host/runtime mutation text found in forge-operator-session wrapper" >&2
      exit 1
    fi

    FORGE_OPERATOR_COMPOSITOR="$TMPDIR/must-not-run" \
      FORGE_SHELL_BINARY="$TMPDIR/must-not-run" \
      FORGE_CORE_URL=http://127.0.0.1:19994 \
      "$wrapper" arg1 arg2 > "$TMPDIR/operator.out"
    grep -F 'fake-labwc:true:operator-desktop:http://127.0.0.1:19994:/forge/data:/forge/data/auth/api_token:unset:' "$TMPDIR/operator.out"
    grep -F 'forge-operator-startup' "$TMPDIR/operator.out"
    grep -F 'arg1 arg2' "$TMPDIR/operator.out"
    "$startup" arg1 arg2 > "$TMPDIR/startup.out"
    grep -F 'fake-shell-session:arg1 arg2' "$TMPDIR/startup.out"

    mkdir -p "$out"
    echo "ok" > "$out/result"
    runHook postInstall
  '';

  meta = with lib; {
    description = "Static safety checks for the FORGE G6 operator desktop wrapper";
    license = licenses.mit;
    platforms = platforms.linux;
  };
}
