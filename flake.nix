{
  description = "FORGE AI-OS — local-first governed cognitive operating substrate and FORGE-OS shell foundation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      ...
    }:
    let
      # Overlays that introduce FORGE packages under pkgs.*
      overlays = [ (import ./nix/overlays/default.nix) ];
    in
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = false;
          overlays = overlays;
        };
        forgeDesktopShell = pkgs.callPackage ./nix/packages/forge-desktop-shell.nix { };
        forgeOperatorDesktopShell = pkgs.callPackage ./nix/packages/forge-desktop-shell.nix {
          renderProfile = "vm-safe";
          bootLogin = false;
          emptyDesktopOnBoot = true;
        };
        forgeShellSession = pkgs.callPackage ./nix/packages/forge-shell-session.nix {
          forgeDesktopShell = forgeDesktopShell;
        };
        forgeOperatorShellSession = pkgs.callPackage ./nix/packages/forge-shell-session.nix {
          forgeDesktopShell = forgeOperatorDesktopShell;
        };
      in
      {
        packages = {
          forge-core = pkgs.callPackage ./nix/packages/forge-core.nix { };
          forge-desktop-shell = forgeDesktopShell;
          forge-operator-desktop-shell = forgeOperatorDesktopShell;
          forge-shell-session = forgeShellSession;
          forge-operator-shell-session = forgeOperatorShellSession;
          forge-wayland-session = pkgs.callPackage ./nix/packages/forge-wayland-session.nix {
            forge-shell-session = forgeShellSession;
          };
          forge-operator-session = pkgs.callPackage ./nix/packages/forge-operator-session.nix {
            forge-shell-session = forgeOperatorShellSession;
          };
          forge-operator-toolbelt = pkgs.callPackage ./nix/packages/forge-operator-toolbelt.nix {
            inherit pkgs;
          };
          default = self.packages.${system}.forge-core;
        };

        apps = {
          forge-core = {
            type = "app";
            program = "${self.packages.${system}.forge-core}/bin/core";
          };
          forge-shell-session = {
            type = "app";
            program = "${self.packages.${system}.forge-shell-session}/bin/forge-shell-session";
          };
          forge-operator-shell-session = {
            type = "app";
            program = "${self.packages.${system}.forge-operator-shell-session}/bin/forge-shell-session";
          };
          forge-desktop-shell = {
            type = "app";
            program = "${self.packages.${system}.forge-desktop-shell}/bin/forge-desktop-shell";
          };
          forge-operator-desktop-shell = {
            type = "app";
            program = "${self.packages.${system}.forge-operator-desktop-shell}/bin/forge-desktop-shell";
          };
          forge-wayland-session = {
            type = "app";
            program = "${self.packages.${system}.forge-wayland-session}/bin/forge-wayland-session";
          };
          forge-operator-session = {
            type = "app";
            program = "${self.packages.${system}.forge-operator-session}/bin/forge-operator-session";
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
          forge-desktop-shell = pkgs.callPackage ./nix/checks/forge-desktop-shell.nix {
            forgeDesktopShell = self.packages.${system}.forge-desktop-shell;
          };
          go-tests = pkgs.callPackage ./nix/checks/go-tests.nix { };
          go-vet = pkgs.callPackage ./nix/checks/go-vet.nix { };
          forge-shell-session = pkgs.callPackage ./nix/checks/forge-shell-session.nix {
            forge-shell-session = self.packages.${system}.forge-shell-session;
          };
          forge-wayland-session = pkgs.callPackage ./nix/checks/forge-wayland-session.nix {
            forge-wayland-session = self.packages.${system}.forge-wayland-session;
          };
          forge-operator-session = pkgs.callPackage ./nix/checks/forge-operator-session.nix {
            forge-operator-session = self.packages.${system}.forge-operator-session;
          };
          forge-operator-toolbelt = pkgs.callPackage ./nix/checks/forge-operator-toolbelt.nix {
            forgeOperatorToolbelt = self.packages.${system}.forge-operator-toolbelt;
          };
          forge-operator-desktop = pkgs.callPackage ./nix/checks/forge-operator-desktop.nix { };
          forge-native-desktop-runtime = pkgs.callPackage ./nix/checks/forge-native-desktop-runtime.nix { };
          forge-operator-vm = pkgs.callPackage ./nix/checks/forge-operator-vm.nix { };
          forge-optiplex-7000 = pkgs.callPackage ./nix/checks/forge-optiplex-7000.nix { };
          forge-vbox-graphics-test = pkgs.callPackage ./nix/checks/forge-vbox-graphics-test.nix { };
          forge-shadow-env = pkgs.callPackage ./nix/checks/forge-shadow-env.nix { };
          forge-workspace-default = pkgs.callPackage ./nix/checks/forge-workspace-default.nix { };
          forge-core-bind-host = pkgs.callPackage ./nix/checks/forge-core-bind-host.nix { };
          js-build = pkgs.callPackage ./nix/checks/js-build.nix { };
        };

        formatter = pkgs.nixfmt;
      }
    )
    // {
      # System-independent outputs.
      overlays.default = import ./nix/overlays/default.nix;

      nixosModules = {
        forge-os = import ./nix/nixos/modules/forge-os.nix;
        forge-services = import ./nix/nixos/modules/forge-services.nix;
        forge-storage = import ./nix/nixos/modules/forge-storage.nix;
        forge-host-kernel = import ./nix/nixos/modules/forge-host-kernel.nix;
        forge-shell-session = import ./nix/nixos/modules/forge-shell-session.nix;
        forge-vbox-graphics-test = import ./nix/nixos/profiles/forge-vbox-graphics-test.nix;
        forge-operator-desktop = import ./nix/nixos/profiles/forge-operator-desktop.nix;
        forge-native-desktop-runtime = import ./nix/nixos/profiles/forge-native-desktop-runtime.nix;
        default = self.nixosModules.forge-os;
      };

      nixosConfigurations = {
        forge-optiplex-7000 = nixpkgs.lib.nixosSystem {
          system = "x86_64-linux";
          modules = [
            {
              nixpkgs.overlays = overlays;
              nixpkgs.config.allowUnfree = false;
            }
            ./nix/nixos/configurations/forge-optiplex-7000.nix
          ];
        };
        forge-operator-vm = nixpkgs.lib.nixosSystem {
          system = "x86_64-linux";
          modules = [
            {
              nixpkgs.overlays = overlays;
              nixpkgs.config.allowUnfree = false;
            }
            ./nix/nixos/configurations/forge-operator-vm.nix
          ];
        };
      };
    };
}
