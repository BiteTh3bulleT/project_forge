final: prev: {
  # Expose FORGE packages through a Nixpkgs overlay so downstream
  # consumers (future NixOS modules, release flakes) can reference
  # `pkgs.forge-core` uniformly.
  forge-core = final.callPackage ../packages/forge-core.nix { };
  forge-recover = final.callPackage ../packages/forge-recover.nix { };
  forge-desktop-shell = final.callPackage ../packages/forge-desktop-shell.nix { };
  forge-operator-desktop-shell = final.callPackage ../packages/forge-desktop-shell.nix {
    renderProfile = "vm-safe";
    bootLogin = false;
    emptyDesktopOnBoot = true;
  };
  forge-shell-session = final.callPackage ../packages/forge-shell-session.nix {
    forgeDesktopShell = final.forge-desktop-shell;
  };
  forge-operator-shell-session = final.callPackage ../packages/forge-shell-session.nix {
    forgeDesktopShell = final.forge-operator-desktop-shell;
  };
  forge-wayland-session = final.callPackage ../packages/forge-wayland-session.nix {
    forge-shell-session = final.forge-shell-session;
  };
  forge-operator-session = final.callPackage ../packages/forge-operator-session.nix {
    forge-shell-session = final.forge-operator-shell-session;
  };
}
