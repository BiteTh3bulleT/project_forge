final: prev: {
  # Expose FORGE packages through a Nixpkgs overlay so downstream
  # consumers (future NixOS modules, release flakes) can reference
  # `pkgs.forge-core` uniformly.
  forge-core = final.callPackage ../packages/forge-core.nix { };
  forge-desktop-shell = final.callPackage ../packages/forge-desktop-shell.nix { };
  forge-shell-session = final.callPackage ../packages/forge-shell-session.nix {
    forgeDesktopShell = final.forge-desktop-shell;
  };
  forge-wayland-session = final.callPackage ../packages/forge-wayland-session.nix {
    forge-shell-session = final.forge-shell-session;
  };
}
