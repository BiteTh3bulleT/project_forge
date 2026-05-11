{
  lib,
  stdenv,
  forge-shell-session,
}:

stdenv.mkDerivation {
  name = "forge-shell-session-wrapper-check";

  src = lib.cleanSource ../..;

  sourceRoot = ".";
  dontUnpack = true;
  dontConfigure = true;
  dontBuild = true;

  installPhase = ''
    runHook preInstall

    wrapper="${forge-shell-session}/bin/forge-shell-session"
    module="$src/nix/nixos/modules/forge-shell-session.nix"
    desktop_main="$src/apps/desktop/src-tauri/src/main.rs"

    test -x "$wrapper"
    test -f "$module"
    test -f "$desktop_main"

    grep -F 'FORGE_SHELL_SESSION_ENABLED=true' "$wrapper"
    grep -F 'FORGE_SHELL_MODE="''${FORGE_SHELL_MODE:-fullscreen-shell}"' "$wrapper"
    grep -F 'http://127.0.0.1:18492' "$wrapper"
    grep -F 'FORGE_SHELL_SAFE_MODE=true' "$wrapper"
    grep -F 'FORGE_SHELL_FULLSCREEN="''${FORGE_SHELL_FULLSCREEN:-true}"' "$wrapper"
    grep -F 'FORGE_SHELL_HOST_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false' "$wrapper"
    grep -F 'FORGE_SHELL_MODEL_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false' "$wrapper"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$wrapper"
    grep -F 'FORGE_SHELL_BINARY' "$wrapper"
    grep -F 'nix_desktop_shell' "$wrapper"
    grep -F 'target/release/forge_desktop' "$wrapper"
    grep -F 'target/debug/forge_desktop' "$wrapper"
    grep -F 'no Nix-built or local Tauri desktop binary was found.' "$wrapper"

    grep -F 'options.forge.shellSession' "$module"
    grep -F 'enable = lib.mkEnableOption' "$module"
    grep -F 'default = false;' "$module"
    grep -F 'default = "wayland";' "$module"
    grep -F 'compositor = lib.mkOption' "$module"
    grep -F 'fullscreen = lib.mkOption' "$module"
    grep -F '"operator-desktop"' "$module"
    grep -F '"labwc"' "$module"
    grep -F 'forge-operator-session' "$module"
    grep -F 'options.forge.shellSession.wayland' "$module"
    grep -F 'sessionName = lib.mkOption' "$module"
    grep -F 'environment.etc."xdg/wayland-sessions/' "$module"
    grep -F 'services.displayManager.autoLogin.enable = lib.mkDefault false;' "$module"
    grep -F 'assertion = cfg.autoStart == false;' "$module"
    grep -F 'assertion = cfg.safeMode == true;' "$module"
    grep -F 'operator-desktop/labwc/non-fullscreen' "$module"
    grep -F 'FORGE graphical shell fullscreen setting must match the selected mode.' "$module"
    grep -F 'FORGE graphical shell compositor must match the selected mode.' "$module"
    grep -F 'assertion = cfg.wayland.enable == true;' "$module"

    grep -F 'fn fit_operator_desktop_window' "$desktop_main"
    grep -F '.current_monitor()' "$desktop_main"
    grep -F 'window.available_monitors()' "$desktop_main"
    grep -F 'window.set_position(PhysicalPosition::new' "$desktop_main"
    grep -F 'window.set_size(PhysicalSize::new' "$desktop_main"
    grep -F 'window.set_resizable(true)' "$desktop_main"

    forbidden='systemctl|nixos-rebuild|modprobe|rmmod|reboot|shutdown|apt-get|dnf|zypper|pacman|LoadModel|UnloadModel|GenerateStream|semantic memory write|os.RemoveAll|rm -rf'
    if grep -E "$forbidden" "$wrapper"; then
      echo "forbidden host/runtime mutation text found in forge-shell-session wrapper" >&2
      exit 1
    fi

    fake="$TMPDIR/fake-forge-shell"
    printf '%s\n' \
      '#!/bin/sh' \
      'echo "fake-shell:$FORGE_SHELL_SESSION_ENABLED:$FORGE_CORE_URL:$*"' \
      > "$fake"
    chmod +x "$fake"
    FORGE_SHELL_BINARY="$fake" FORGE_CORE_URL=http://127.0.0.1:19997 "$wrapper" override path > "$TMPDIR/shell-override.out"
    grep -F 'fake-shell:true:http://127.0.0.1:19997:override path' "$TMPDIR/shell-override.out"

    FORGE_SHELL_BINARY="$fake" \
      FORGE_SHELL_MODE=operator-desktop \
      FORGE_SHELL_FULLSCREEN=false \
      "$wrapper" preserved mode > "$TMPDIR/shell-preserve-env.out"
    grep -F 'fake-shell:true:http://127.0.0.1:18492:preserved mode' "$TMPDIR/shell-preserve-env.out"

    fake_desktop="$TMPDIR/fake-forge-desktop"
    printf '%s\n' \
      '#!/bin/sh' \
      'echo "packaged-desktop:$FORGE_SHELL_SESSION_ENABLED:$FORGE_CORE_URL:$*"' \
      > "$fake_desktop"
    chmod +x "$fake_desktop"

    if grep -F 'nix_desktop_shell=""' "$wrapper"; then
      repo="$TMPDIR/repo"
      mkdir -p "$repo/apps/desktop/src-tauri/target/debug"
      printf '%s\n' \
        '#!/bin/sh' \
        'echo "local-tauri:$FORGE_SHELL_SESSION_ENABLED:$FORGE_CORE_URL:$*"' \
        > "$repo/apps/desktop/src-tauri/target/debug/forge_desktop"
      chmod +x "$repo/apps/desktop/src-tauri/target/debug/forge_desktop"
      FORGE_REPO_ROOT="$repo" FORGE_CORE_URL=http://127.0.0.1:19996 "$wrapper" local path > "$TMPDIR/shell-local.out"
      grep -F 'local-tauri:true:http://127.0.0.1:19996:local path' "$TMPDIR/shell-local.out"

      set +e
      FORGE_REPO_ROOT="$TMPDIR/missing-repo" "$wrapper" > "$TMPDIR/shell-fail.out" 2> "$TMPDIR/shell-fail.err"
      status=$?
      set -e
      test "$status" -eq 1
      grep -F 'FORGE graphical shell binary is not available.' "$TMPDIR/shell-fail.err"
      grep -F 'no Nix-built or local Tauri desktop binary was found.' "$TMPDIR/shell-fail.err"
    else
      FORGE_DESKTOP_SHELL_BINARY="$fake_desktop" FORGE_CORE_URL=http://127.0.0.1:19995 "$wrapper" packaged path > "$TMPDIR/shell-packaged.out"
      grep -F 'packaged-desktop:true:http://127.0.0.1:19995:packaged path' "$TMPDIR/shell-packaged.out"
    fi

    mkdir -p "$out"
    echo "ok" > "$out/result"
    runHook postInstall
  '';

  meta = with lib; {
    description = "Static safety checks for the FORGE graphical shell session wrapper";
    license = licenses.mit;
    platforms = platforms.unix;
  };
}
