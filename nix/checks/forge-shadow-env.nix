{ lib, stdenv }:

stdenv.mkDerivation {
  name = "forge-shadow-env-check";

  src = lib.cleanSource ../..;

  sourceRoot = ".";
  dontUnpack = true;
  dontConfigure = true;
  dontBuild = true;

  installPhase = ''
    runHook preInstall

    config="$src/services/core/internal/config/config.go"
    module="$src/nix/nixos/modules/forge-services.nix"
    test -f "$config"
    test -f "$module"

    go_vars="$TMPDIR/go-shadow-vars"
    module_vars="$TMPDIR/module-shadow-vars"
    missing="$TMPDIR/missing-shadow-vars"

    grep -o '"FORGE_K_SHADOW_[A-Z0-9_]*"' "$config" | tr -d '"' | sort -u > "$go_vars"
    grep -o 'FORGE_K_SHADOW_[A-Z0-9_]*' "$module" | sort -u > "$module_vars"
    comm -23 "$go_vars" "$module_vars" > "$missing"

    if [ -s "$missing" ]; then
      echo "NixOS forge-core service is missing explicit FORGE-K shadow env defaults:" >&2
      cat "$missing" >&2
      exit 1
    fi

    mkdir -p "$out"
    echo "ok" > "$out/result"
    runHook postInstall
  '';

  meta = with lib; {
    description = "Checks that FORGE-K shadow config flags have explicit NixOS service defaults";
    license = licenses.mit;
    platforms = platforms.unix;
  };
}
