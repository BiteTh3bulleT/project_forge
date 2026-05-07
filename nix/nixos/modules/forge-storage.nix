{ config, lib, ... }:

let
  cfg = config.forge.storage;
  subdirs = [
    "data"
    "state"
    "models"
    "logs"
    "artifacts"
    "snapshots"
    "imports"
    "backups"
    "exports"
    "runtime"
    "journal"
    "host"
    "config"
    "workspaces"
  ];
in
{
  options.forge.storage = {
    enable = lib.mkEnableOption "FORGE private host storage layout";

    root = lib.mkOption {
      type = lib.types.str;
      default = "/forge";
      description = "Root directory for private FORGE-OS host storage.";
    };

    user = lib.mkOption {
      type = lib.types.str;
      default = "forge";
      description = "User that owns FORGE storage directories.";
    };

    group = lib.mkOption {
      type = lib.types.str;
      default = "forge";
      description = "Group that owns FORGE storage directories.";
    };
  };

  config = lib.mkIf cfg.enable {
    users.groups.${cfg.group} = { };

    users.users.${cfg.user} = {
      isSystemUser = true;
      group = cfg.group;
      home = "${cfg.root}/state";
      createHome = false;
      description = "FORGE service account";
    };

    systemd.tmpfiles.rules =
      [ "d ${cfg.root} 0750 ${cfg.user} ${cfg.group} -" ]
      ++ map (name: "d ${cfg.root}/${name} 0750 ${cfg.user} ${cfg.group} -") subdirs;
  };
}
