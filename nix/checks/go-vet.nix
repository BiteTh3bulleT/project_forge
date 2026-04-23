{ lib, stdenv, go }:

# `go vet ./...` against services/core.
# Same sandbox caveat as go-tests.nix: requires vendored deps to run
# inside `nix flake check`. Documents and skips honestly when absent.
stdenv.mkDerivation {
  name = "forge-core-go-vet";

  src = lib.cleanSource ../..;

  nativeBuildInputs = [ go ];

  dontConfigure = true;
  dontInstall = true;

  buildPhase = ''
    runHook preBuild
    export HOME=$TMPDIR
    export GOFLAGS="-mod=vendor"
    if [ ! -d services/core/vendor ]; then
      echo "SKIP: services/core/vendor not present; cannot run go vet in sandbox."
      mkdir -p $out
      echo "skipped: no vendored deps" > $out/result
      exit 0
    fi
    cd services/core
    go vet ./...
    mkdir -p $out
    echo "ok" > $out/result
    runHook postBuild
  '';
}
