{ config, lib, pkgs, ... }:

let
  cfg = config.services.forge-core;
  boolString = value: if value then "true" else "false";
in
{
  imports = [
    ./forge-storage.nix
  ];

  options.services.forge-core = {
    enable = lib.mkEnableOption "FORGE core service";

    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.callPackage ../../packages/forge-core.nix { };
      defaultText = lib.literalExpression "pkgs.callPackage ../../packages/forge-core.nix { }";
      description = "Package providing the FORGE core binary.";
    };

    user = lib.mkOption {
      type = lib.types.str;
      default = config.forge.storage.user;
      description = "User used to run forge-core.";
    };

    group = lib.mkOption {
      type = lib.types.str;
      default = config.forge.storage.group;
      description = "Group used to run forge-core.";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 18492;
      description = "Local HTTP port for forge-core.";
    };

    bindHost = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1";
      description = "HTTP bind host for forge-core. Use 0.0.0.0 only when an enclosing network boundary intentionally exposes the service.";
    };

    allowWildcardBind = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Explicit opt-in required before forge-core may bind all interfaces from the NixOS service envelope.";
    };

    dataDir = lib.mkOption {
      type = lib.types.str;
      default = "${config.forge.storage.root}/data";
      description = "FORGE_DATA_DIR for forge-core.";
    };

    modelHome = lib.mkOption {
      type = lib.types.str;
      default = "${config.forge.storage.root}/models";
      description = "Model artifact directory for managed model runtime state.";
    };

    workspaceDir = lib.mkOption {
      type = lib.types.str;
      default = "${config.forge.storage.root}/workspaces/default";
      description = "Workspace root exposed to FORGE file-sensitive operations.";
    };

    enableModelRuntime = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Whether to enable governed modelruntime at service start.";
    };

    safeModeForceCPUOnly = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Force CPU-only safe mode for the N2 host substrate scaffold.";
    };

    extraEnvironment = lib.mkOption {
      type = lib.types.attrsOf lib.types.str;
      default = { };
      description = "Additional environment variables for forge-core.";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.forge-core = {
      description = "FORGE AI-OS core service";
      wantedBy = [ "multi-user.target" ];
      wants = [ "network-online.target" ];
      after = [ "network-online.target" ];

      environment = {
        FORGE_DATA_DIR = toString cfg.dataDir;
        FORGE_CORE_PORT = toString cfg.port;
        FORGE_CORE_BIND_HOST = toString cfg.bindHost;
        FORGE_ALLOW_WILDCARD_BIND = boolString cfg.allowWildcardBind;
        FORGE_MODEL_HOME = toString cfg.modelHome;
        FORGE_WORKSPACE_DIR = toString cfg.workspaceDir;
        FORGE_ENABLE_MODEL_RUNTIME = boolString cfg.enableModelRuntime;
        FORGE_SAFE_MODE_FORCE_CPU_ONLY = boolString cfg.safeModeForceCPUOnly;
        FORGE_GPU_ENABLED = "false";
        FORGE_ALLOW_LLAMA_CPP_SPAWN = "false";
        FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD = "true";
        FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD = "false";
        FORGE_MODEL_POLICY_ALLOW_CROSS_WORKSPACE = "false";
        FORGE_ENABLE_OPENAI_COMPAT_API = "false";
        FORGE_K_SHADOW_MODE_ENABLED = "false";
        FORGE_K_SHADOW_CHAT_METADATA_ENABLED = "false";
        FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED = "false";
        FORGE_K_SHADOW_ADVISORY_ENABLED = "false";
        FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED = "false";
      } // cfg.extraEnvironment;

      serviceConfig = {
        Type = "simple";
        User = cfg.user;
        Group = cfg.group;
        ExecStart = "${cfg.package}/bin/core";
        Restart = "on-failure";
        RestartSec = "5s";
        NoNewPrivileges = true;
        CapabilityBoundingSet = "";
        PrivateTmp = true;
        PrivateDevices = true;
        ProtectSystem = "strict";
        ProtectHome = "read-only";
        ProtectClock = true;
        ProtectControlGroups = true;
        ProtectKernelLogs = true;
        ProtectKernelModules = true;
        ProtectKernelTunables = true;
        RestrictAddressFamilies = [
          "AF_UNIX"
          "AF_INET"
          "AF_INET6"
        ];
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        LockPersonality = true;
        SystemCallArchitectures = "native";
        UMask = "0077";
        ReadWritePaths = [ (toString config.forge.storage.root) ];
        WorkingDirectory = cfg.dataDir;
      };
    };

    assertions = [
      {
        assertion = cfg.allowWildcardBind || !(lib.elem cfg.bindHost [ "0.0.0.0" "::" ]);
        message = "services.forge-core.bindHost may not bind all interfaces unless services.forge-core.allowWildcardBind = true.";
      }
    ];
  };
}
