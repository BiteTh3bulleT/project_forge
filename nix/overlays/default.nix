final: prev: {
  # Expose FORGE packages through a Nixpkgs overlay so downstream
  # consumers (future NixOS modules, release flakes) can reference
  # `pkgs.forge-core` uniformly.
  forge-core = final.callPackage ../packages/forge-core.nix { };
}
