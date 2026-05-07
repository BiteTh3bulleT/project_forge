{
  description = "FORGE AI-OS — local-first governed cognitive operating substrate (Phase N1: light Nix foundation)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    let
      # Overlays that introduce FORGE packages under pkgs.*
      overlays = [ (import ./nix/overlays/default.nix) ];
    in
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = false;
          overlays = overlays;
        };
      in
      {
        packages = {
          forge-core = pkgs.callPackage ./nix/packages/forge-core.nix { };
          default = self.packages.${system}.forge-core;
        };

        apps = {
          forge-core = {
            type = "app";
            program = "${self.packages.${system}.forge-core}/bin/core";
          };
          default = self.apps.${system}.forge-core;
        };

        devShells = {
          default = pkgs.callPackage ./nix/shells/default.nix { };
          core = pkgs.callPackage ./nix/shells/core.nix { };
          desktop = pkgs.callPackage ./nix/shells/desktop.nix { };
          aios = pkgs.callPackage ./nix/shells/aios.nix { };
        };

        checks = {
          go-tests = pkgs.callPackage ./nix/checks/go-tests.nix { };
          go-vet = pkgs.callPackage ./nix/checks/go-vet.nix { };
          js-build = pkgs.callPackage ./nix/checks/js-build.nix { };
        };

        formatter = pkgs.nixfmt;
      }
    ) // {
      # System-independent outputs.
      overlays.default = import ./nix/overlays/default.nix;

      nixosModules = {
        forge-os = import ./nix/nixos/modules/forge-os.nix;
        forge-services = import ./nix/nixos/modules/forge-services.nix;
        forge-storage = import ./nix/nixos/modules/forge-storage.nix;
        forge-host-kernel = import ./nix/nixos/modules/forge-host-kernel.nix;
        default = self.nixosModules.forge-os;
      };
    };
}
