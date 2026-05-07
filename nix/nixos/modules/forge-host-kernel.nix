{ config, lib, ... }:

let
  cfg = config.forge.hostKernelBridge;
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

  config = lib.mkIf cfg.enable {
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
  };
}
