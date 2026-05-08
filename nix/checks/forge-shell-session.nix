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

    test -x "$wrapper"
    test -f "$module"

    grep -F 'FORGE_SHELL_SESSION_ENABLED=true' "$wrapper"
    grep -F 'FORGE_SHELL_MODE=fullscreen-shell' "$wrapper"
    grep -F 'http://127.0.0.1:18492' "$wrapper"
    grep -F 'FORGE_SHELL_SAFE_MODE=true' "$wrapper"
    grep -F 'FORGE_SHELL_FULLSCREEN=true' "$wrapper"
    grep -F 'FORGE_SHELL_HOST_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false' "$wrapper"
    grep -F 'FORGE_SHELL_MODEL_MUTATION=false' "$wrapper"
    grep -F 'FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false' "$wrapper"
    grep -F 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false' "$wrapper"
    grep -F 'FORGE_SHELL_BINARY' "$wrapper"
    grep -F 'nix_desktop_shell' "$wrapper"
    grep -F 'target/release/forge_desktop' "$wrapper"
    grep -F 'target/debug/forge_desktop' "$wrapper"
    grep -F 'does not yet contain a' "$wrapper"

    grep -F 'options.forge.shellSession' "$module"
    grep -F 'enable = lib.mkEnableOption' "$module"
    grep -F 'default = false;' "$module"
    grep -F 'assertion = cfg.autoStart == false;' "$module"
    grep -F 'assertion = cfg.safeMode == true;' "$module"

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
    grep -F 'Nix-built Tauri desktop binary.' "$TMPDIR/shell-fail.err"

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
