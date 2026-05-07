{ config, lib, pkgs, ... }:

let
  cfg = config.forge.os;
in
{
  imports = [
    ./forge-storage.nix
    ./forge-services.nix
    ./forge-host-kernel.nix
  ];

  options.forge.os = {
    enable = lib.mkEnableOption "private FORGE-OS host substrate scaffold";

    storageRoot = lib.mkOption {
      type = lib.types.str;
      default = "/forge";
      description = "Root path for the private FORGE host storage layout.";
    };

    safeMode = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Keep the host profile in CPU-only, explicit-control safe mode.";
    };

    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.callPackage ../../packages/forge-core.nix { };
      defaultText = lib.literalExpression "pkgs.callPackage ../../packages/forge-core.nix { }";
      description = "Package used by the forge-core service.";
    };
  };

  config = lib.mkIf cfg.enable {
    forge.storage = {
      enable = lib.mkDefault true;
      root = lib.mkDefault cfg.storageRoot;
    };

    services.forge-core = {
      enable = lib.mkDefault true;
      package = lib.mkDefault cfg.package;
      dataDir = lib.mkDefault "${cfg.storageRoot}/data";
      modelHome = lib.mkDefault "${cfg.storageRoot}/models";
      safeModeForceCPUOnly = lib.mkDefault cfg.safeMode;
      enableModelRuntime = lib.mkDefault false;
    };

    forge.hostKernelBridge = {
      enable = lib.mkDefault false;
      allowMutation = lib.mkDefault false;
    };
  };
}
