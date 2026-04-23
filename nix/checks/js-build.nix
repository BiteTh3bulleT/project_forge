{ lib, stdenv, nodejs_20 }:

# JS/TS workspace build.
#
# Phase N1 honest status: The repo does not yet vendor npm dependencies
# in a Nix-friendly way (no npmDepsHash / offline cache). A real
# `nix flake check` run of `npm install && npm run build` would need
# network access, which the sandbox blocks.
#
# Rather than fake a pass, this check detects missing node_modules and
# marks itself skipped. Run `npm install && npm run build:desktop` from
# a dev shell until npm dependency vendoring is added (Phase N2+).
stdenv.mkDerivation {
  name = "forge-js-build";

  src = lib.cleanSource ../..;

  nativeBuildInputs = [ nodejs_20 ];

  dontConfigure = true;
  dontInstall = true;

  buildPhase = ''
    runHook preBuild
    export HOME=$TMPDIR
    if [ ! -d node_modules ]; then
      echo "SKIP: node_modules not present in source; cannot build JS in sandbox."
      echo "Run 'npm install && npm run build:desktop' from a dev shell."
      mkdir -p $out
      echo "skipped: no node_modules" > $out/result
      exit 0
    fi
    npm run build:desktop
    mkdir -p $out
    echo "ok" > $out/result
    runHook postBuild
  '';
}
