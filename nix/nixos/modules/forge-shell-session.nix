{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.forge.shellSession;
  boolString = value: if value then "true" else "false";
  packageName =
    if cfg.package == null then
      "not-packaged"
    else
      cfg.package.pname or cfg.package.name or "forge-shell";
  execLine =
    if cfg.package == null then
      "${pkgs.runtimeShell} -lc \"echo 'FORGE graphical shell package is not wired in Phase G1' >&2; exit 1\""
    else
      "${cfg.package}/bin/forge-shell-session";
in
{
  imports = [
    ./forge-storage.nix
  ];

  options.forge.shellSession = {
    enable = lib.mkEnableOption "FORGE graphical shell session scaffold";

    mode = lib.mkOption {
      type = lib.types.enum [ "fullscreen-shell" ];
      default = "fullscreen-shell";
      description = "FORGE graphical shell launch mode. Phase G1 only scaffolds fullscreen-shell.";
    };

    package = lib.mkOption {
      type = lib.types.nullOr lib.types.package;
      default = null;
      description = "Optional package providing the FORGE graphical shell binary. Null keeps the session descriptor as inert packaging scaffolding.";
    };

    user = lib.mkOption {
      type = lib.types.str;
      default = config.forge.storage.user;
      description = "User expected to own the FORGE graphical shell runtime directory.";
    };

    displayBackend = lib.mkOption {
      type = lib.types.enum [
        "wayland"
        "x11"
      ];
      default = "wayland";
      description = "Display backend metadata for the FORGE graphical shell session scaffold.";
    };

    autoStart = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Reserved for a future phase. Phase G1 must not autostart or replace an existing desktop.";
    };

    coreURL = lib.mkOption {
      type = lib.types.str;
      default = "http://127.0.0.1:18492";
      description = "Governed local forge-core URL used by the graphical shell.";
    };

    safeMode = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Keep the graphical shell in safe mode with host mutation and direct system control disabled.";
    };

    runtimePath = lib.mkOption {
      type = lib.types.str;
      default = "${config.forge.storage.root}/runtime/shell-session";
      description = "Directory reserved for FORGE graphical shell session state.";
    };
  };

  config = lib.mkIf cfg.enable {
    assertions = [
      {
        assertion = cfg.mode == "fullscreen-shell";
        message = "Phase G1 supports only forge.shellSession.mode = fullscreen-shell.";
      }
      {
        assertion = cfg.autoStart == false;
        message = "Phase G1 FORGE graphical shell session must not autostart or replace the user's desktop.";
      }
    ];

    systemd.tmpfiles.rules = [
      "d ${cfg.runtimePath} 0750 ${cfg.user} ${config.forge.storage.group} -"
    ];

    environment.etc."forge/shell-session.env".text = ''
      FORGE_SHELL_SESSION_ENABLED=true
      FORGE_SHELL_MODE=${cfg.mode}
      FORGE_SHELL_PACKAGE=${packageName}
      FORGE_SHELL_DISPLAY_BACKEND=${cfg.displayBackend}
      FORGE_CORE_URL=${cfg.coreURL}
      FORGE_SHELL_SAFE_MODE=${boolString cfg.safeMode}
      FORGE_SHELL_AUTOSTART=false
      FORGE_SHELL_HOST_MUTATION=false
      FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false
      FORGE_SHELL_MODEL_MUTATION=false
      FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false
      FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false
      FORGE_SHELL_RUNTIME_DIR=${cfg.runtimePath}
    '';

    environment.etc."forge/shell-session.desktop".text = ''
      [Desktop Entry]
      Name=FORGE Graphical Shell
      Comment=FORGE graphical shell session scaffold
      Type=Application
      Exec=${execLine}
      X-FORGE-Mode=${cfg.mode}
      X-FORGE-DisplayBackend=${cfg.displayBackend}
      X-FORGE-SafeMode=${boolString cfg.safeMode}
      X-FORGE-AutoStart=false
    '';
  };
}
