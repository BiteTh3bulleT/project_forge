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
