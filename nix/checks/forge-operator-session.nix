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
      grep -F '<application identifier="dev.forge.workshop">' "$config_file"
      grep -F '<decor>no</decor>' "$config_file"
      shift 2
      test "$1" = "--startup"
      shift
      echo "fake-labwc:$FORGE_SHELL_SESSION_ENABLED:$FORGE_SHELL_MODE:$FORGE_CORE_URL:''${FORGE_SHELL_BINARY:-unset}:$1:$*"
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
    wrapper="${testOperatorSession}/bin/forge-operator-session"

    test -x "$real_wrapper"
    test -x "$wrapper"

    grep -F 'FORGE_SHELL_MODE=operator-desktop' "$wrapper"
    grep -F 'FORGE_SHELL_FULLSCREEN=false' "$wrapper"
    grep -F 'FORGE_SHELL_SAFE_MODE=true' "$wrapper"
    grep -F 'FORGE_SHELL_HOST_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false' "$wrapper"
    grep -F 'FORGE_SHELL_MODEL_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false' "$wrapper"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$wrapper"
    grep -F 'FORGE_SHELL_COMPOSITOR=labwc' "$wrapper"
    grep -F 'WEBKIT_DISABLE_DMABUF_RENDERER="''${WEBKIT_DISABLE_DMABUF_RENDERER:-1}"' "$wrapper"
    grep -F 'unset FORGE_SHELL_BINARY' "$wrapper"
    grep -F 'unset FORGE_REPO_ROOT' "$wrapper"
    grep -F 'FORGE_OPERATOR_LABWC_CONFIG_DIR=' "$wrapper"
    grep -F 'FORGE_OPERATOR_DESKTOP_LOCKED=true' "$wrapper"
    grep -F 'labwc_config_file="$labwc_config_dir/rc.xml"' "$wrapper"
    grep -F '<application identifier="dev.forge.workshop">' "$wrapper"
    grep -F '<decor>no</decor>' "$wrapper"
    grep -F 'exec "$compositor" --config "$labwc_config_file" --startup "$shell_session" "$@"' "$wrapper"

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
    grep -F 'fake-labwc:true:operator-desktop:http://127.0.0.1:19994:unset:' "$TMPDIR/operator.out"
    grep -F 'forge-shell-session' "$TMPDIR/operator.out"
    grep -F 'arg1 arg2' "$TMPDIR/operator.out"

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
