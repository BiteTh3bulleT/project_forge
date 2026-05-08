{ config, lib, ... }:

let
  cfg = config.forge.hostKernelBridge;
  policyCfg = config.forge.resourcePolicy;
in
{
  imports = [
    ./forge-storage.nix
  ];

  options.forge.hostKernelBridge = {
    enable = lib.mkEnableOption "FORGE read-only Host Kernel Bridge diagnostics scaffold";

    mode = lib.mkOption {
      type = lib.types.enum [ "observe" "report" "recommend" ];
      default = "observe";
      description = "Diagnostic-only host bridge mode for Phase N3.";
    };

    allowMutation = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Reserved for future phases. Must remain false in Phase N3.";
    };

    reportPath = lib.mkOption {
      type = lib.types.str;
      default = "${config.forge.storage.root}/runtime/host-kernel";
      description = "Directory reserved for Host Kernel Bridge diagnostic reports.";
    };
  };

  options.forge.resourcePolicy = {
    enable = lib.mkEnableOption "FORGE-H advisory resource policy diagnostics scaffold";

    advisoryOnly = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Resource policy may classify and recommend only; it must not mutate host or runtime state.";
    };

    allowMutation = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Reserved for future phases. Must remain false in Phase N4.";
    };

    runtimePath = lib.mkOption {
      type = lib.types.str;
      default = "${config.forge.storage.root}/runtime/resource-policy";
      description = "Directory reserved for advisory FORGE-H resource policy reports.";
    };
  };

  config = lib.mkMerge [
    (lib.mkIf cfg.enable {
      assertions = [
        {
          assertion = cfg.allowMutation == false;
          message = "Phase N3 Host Kernel Bridge is read-only; allowMutation must remain false.";
        }
      ];

      systemd.tmpfiles.rules = [
        "d ${cfg.reportPath} 0750 ${config.forge.storage.user} ${config.forge.storage.group} -"
      ];

      environment.etc."forge/host-kernel-bridge.env".text = ''
        FORGE_HOST_KERNEL_BRIDGE_ENABLED=true
        FORGE_HOST_KERNEL_BRIDGE_MODE=${cfg.mode}
        FORGE_HOST_KERNEL_BRIDGE_REPORT_DIR=${cfg.reportPath}
        FORGE_HOST_KERNEL_BRIDGE_ALLOW_MUTATION=false
      '';
    })

    (lib.mkIf policyCfg.enable {
      assertions = [
        {
          assertion = policyCfg.advisoryOnly == true;
          message = "Phase N4 FORGE-H resource policy must remain advisory-only.";
        }
        {
          assertion = policyCfg.allowMutation == false;
          message = "Phase N4 FORGE-H resource policy must not enable host mutation.";
        }
      ];

      systemd.tmpfiles.rules = [
        "d ${policyCfg.runtimePath} 0750 ${config.forge.storage.user} ${config.forge.storage.group} -"
      ];

      environment.etc."forge/resource-policy.env".text = ''
        FORGE_RESOURCE_POLICY_ENABLED=true
        FORGE_RESOURCE_POLICY_ADVISORY_ONLY=true
        FORGE_RESOURCE_POLICY_RUNTIME_DIR=${policyCfg.runtimePath}
        FORGE_RESOURCE_POLICY_ALLOW_MUTATION=false
      '';
    })
  ];
}
