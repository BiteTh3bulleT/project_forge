{ lib, stdenv }:

stdenv.mkDerivation {
  name = "forge-workspace-default-check";

  src = lib.cleanSource ../..;

  sourceRoot = ".";
  dontUnpack = true;
  dontConfigure = true;
  dontBuild = true;

  installPhase = ''
    runHook preInstall

    module="$src/nix/nixos/modules/forge-services.nix"
    storage="$src/nix/nixos/modules/forge-storage.nix"
    test -f "$module"
    test -f "$storage"

    if grep -q 'default = "/";' "$module"; then
      echo "forge-core NixOS workspaceDir must not default to host root" >&2
      exit 1
    fi

    grep -q 'default = "''${config.forge.storage.root}/workspaces/default";' "$module"
    grep -q 'FORGE_WORKSPACE_DIR = toString cfg.workspaceDir;' "$module"
    grep -q 'workspaceMode = lib.mkOption' "$storage"
    grep -q '/workspaces/default ''${cfg.workspaceMode}' "$storage"

    mkdir -p "$out"
    echo "ok" > "$out/result"
    runHook postInstall
  '';

  meta = with lib; {
    description = "Checks that the forge-core NixOS service workspace default is not host root";
    license = licenses.mit;
    platforms = platforms.unix;
  };
}
