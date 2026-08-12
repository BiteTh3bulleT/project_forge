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
  sessionPackageName =
    if cfg.wayland.sessionPackage == null then
      "not-packaged"
    else
      cfg.wayland.sessionPackage.pname or cfg.wayland.sessionPackage.name or "forge-wayland-session";
  compositorPackageName =
    if cfg.wayland.package == null then
      "not-packaged"
    else
      cfg.wayland.package.pname or cfg.wayland.package.name or cfg.compositor;
  shellExecLine =
    if cfg.package == null then
      "${pkgs.runtimeShell} -lc \"echo 'FORGE graphical shell package is not wired; set forge.shellSession.package = pkgs.forge-shell-session' >&2; exit 1\""
    else
      "${cfg.package}/bin/forge-shell-session";
  waylandExecLine =
    if cfg.wayland.sessionPackage == null || cfg.wayland.package == null || cfg.package == null then
      "${pkgs.runtimeShell} -lc \"echo 'FORGE Wayland shell session is not wired; set forge.shellSession.package, wayland.package, and wayland.sessionPackage' >&2; exit 1\""
    else
      let
        sessionBinary =
          if cfg.mode == "operator-desktop" then "forge-operator-session" else "forge-wayland-session";
      in
      "env FORGE_CORE_URL=${cfg.coreURL} VITE_FORGE_API_URL=${cfg.coreURL} FORGE_DATA_DIR=${config.services.forge-core.dataDir} FORGE_API_TOKEN_FILE=${config.services.forge-core.dataDir}/auth/api_token FORGE_SHELL_MODE=${cfg.mode} FORGE_SHELL_DISPLAY_BACKEND=${cfg.displayBackend} FORGE_SHELL_COMPOSITOR=${cfg.compositor} FORGE_SHELL_SAFE_MODE=${boolString cfg.safeMode} FORGE_SHELL_FULLSCREEN=${boolString cfg.fullscreen} FORGE_SHELL_RUNTIME_DIR=${cfg.runtimePath} ${cfg.wayland.sessionPackage}/bin/${sessionBinary}";
in
{
  imports = [
    ./forge-storage.nix
  ];

  options.forge.shellSession = {
    enable = lib.mkEnableOption "FORGE graphical shell session scaffold";

    mode = lib.mkOption {
      type = lib.types.enum [
        "fullscreen-shell"
        "operator-desktop"
      ];
      default = "fullscreen-shell";
      description = "FORGE graphical shell launch mode. G4 supports fullscreen-shell; G6 adds operator-desktop.";
    };

    package = lib.mkOption {
      type = lib.types.nullOr lib.types.package;
      default = if pkgs ? forge-shell-session then pkgs.forge-shell-session else null;
      defaultText = lib.literalExpression "pkgs.forge-shell-session if available, otherwise null";
      example = lib.literalExpression "pkgs.forge-shell-session";
      description = "Optional package providing the FORGE graphical shell session wrapper. Null keeps the session descriptor as inert packaging scaffolding.";
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
      description = "Display backend metadata for the FORGE graphical shell session.";
    };

    compositor = lib.mkOption {
      type = lib.types.enum [
        "cage"
        "labwc"
      ];
      default = "cage";
      description = "Lightweight Wayland compositor command used by the opt-in FORGE shell session.";
    };

    autoStart = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Reserved for a future phase. Phase G4 must not autostart or replace an existing desktop.";
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

    fullscreen = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Run the FORGE graphical shell as a fullscreen shell surface.";
    };

    runtimePath = lib.mkOption {
      type = lib.types.str;
      default = "${config.forge.storage.root}/runtime/shell-session";
      description = "Directory reserved for FORGE graphical shell session state.";
    };
  };

  options.forge.shellSession.wayland = {
    enable = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Generate the opt-in FORGE Wayland session descriptor when forge.shellSession.enable is true.";
    };

    package = lib.mkOption {
      type = lib.types.nullOr lib.types.package;
      default = if pkgs ? cage then pkgs.cage else null;
      defaultText = lib.literalExpression "pkgs.cage if available, otherwise null";
      example = lib.literalExpression "pkgs.cage";
      description = "Package providing the lightweight Wayland compositor used by the FORGE shell session.";
    };

    sessionPackage = lib.mkOption {
      type = lib.types.nullOr lib.types.package;
      default =
        if cfg.mode == "operator-desktop" then
          if pkgs ? forge-operator-session then pkgs.forge-operator-session else null
        else if pkgs ? forge-wayland-session then
          pkgs.forge-wayland-session
        else
          null;
      defaultText = lib.literalExpression "pkgs.forge-operator-session for operator-desktop, otherwise pkgs.forge-wayland-session if available";
      example = lib.literalExpression "pkgs.forge-wayland-session";
      description = "Package providing /bin/forge-wayland-session or /bin/forge-operator-session for operator-desktop mode.";
    };

    sessionName = lib.mkOption {
      type = lib.types.str;
      default = "forge-shell";
      description = "Wayland session descriptor name, without the .desktop suffix.";
    };
  };

  config = lib.mkIf cfg.enable {
    assertions = [
      {
        assertion =
          (cfg.mode == "fullscreen-shell" && cfg.compositor == "cage" && cfg.fullscreen == true)
          || (cfg.mode == "operator-desktop" && cfg.compositor == "labwc" && cfg.fullscreen == false);
        message = "FORGE shell session mode must use fullscreen-shell/cage/fullscreen or operator-desktop/labwc/non-fullscreen.";
      }
      {
        assertion = cfg.autoStart == false;
        message = "Phase G4 FORGE graphical shell session must not autostart or replace the user's desktop.";
      }
      {
        assertion = cfg.safeMode == true;
        message = "Phase G4 FORGE graphical shell session must remain in safe mode.";
      }
      {
        assertion =
          (cfg.mode == "fullscreen-shell" && cfg.fullscreen == true)
          || (cfg.mode == "operator-desktop" && cfg.fullscreen == false);
        message = "FORGE graphical shell fullscreen setting must match the selected mode.";
      }
      {
        assertion = cfg.displayBackend == "wayland";
        message = "Phase G4 FORGE graphical shell session supports only the Wayland display backend.";
      }
      {
        assertion =
          (cfg.mode == "fullscreen-shell" && cfg.compositor == "cage")
          || (cfg.mode == "operator-desktop" && cfg.compositor == "labwc");
        message = "FORGE graphical shell compositor must match the selected mode.";
      }
      {
        assertion = cfg.wayland.enable == true;
        message = "Phase G4 FORGE graphical shell session requires forge.shellSession.wayland.enable = true when forge.shellSession.enable is true.";
      }
      {
        assertion = cfg.mode != "operator-desktop" || cfg.wayland.sessionPackage != null;
        message = "FORGE operator-desktop mode requires forge.shellSession.wayland.sessionPackage = pkgs.forge-operator-session.";
      }
      {
        assertion = cfg.mode != "operator-desktop" || sessionPackageName == "forge-operator-session";
        message = "FORGE operator-desktop mode must use the forge-operator-session package, not the fullscreen forge-wayland-session package.";
      }
    ];

    services.displayManager.autoLogin.enable = lib.mkDefault false;

    systemd.tmpfiles.rules = [
      "d ${cfg.runtimePath} 0750 ${cfg.user} ${config.forge.storage.group} -"
    ];

    environment.etc."forge/shell-session.env".text = ''
      FORGE_SHELL_SESSION_ENABLED=true
      FORGE_SHELL_MODE=${cfg.mode}
      FORGE_SHELL_PACKAGE=${packageName}
      FORGE_SHELL_DISPLAY_BACKEND=${cfg.displayBackend}
      FORGE_SHELL_COMPOSITOR=${cfg.compositor}
      FORGE_SHELL_WAYLAND_PACKAGE=${compositorPackageName}
      FORGE_SHELL_WAYLAND_SESSION_PACKAGE=${sessionPackageName}
      FORGE_CORE_URL=${cfg.coreURL}
      FORGE_DATA_DIR=${config.services.forge-core.dataDir}
      FORGE_API_TOKEN_FILE=${config.services.forge-core.dataDir}/auth/api_token
      FORGE_SHELL_SAFE_MODE=${boolString cfg.safeMode}
      FORGE_SHELL_FULLSCREEN=${boolString cfg.fullscreen}
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
      Comment=FORGE graphical shell launcher
      Type=Application
      Exec=${shellExecLine}
      X-FORGE-Mode=${cfg.mode}
      X-FORGE-DisplayBackend=${cfg.displayBackend}
      X-FORGE-SafeMode=${boolString cfg.safeMode}
      X-FORGE-AutoStart=false
    '';

    environment.etc."xdg/wayland-sessions/${cfg.wayland.sessionName}.desktop".text = ''
      [Desktop Entry]
      Name=FORGE Shell Session
      Comment=Opt-in FORGE Wayland graphical shell session
      Type=Application
      DesktopNames=FORGE
      Exec=${waylandExecLine}
      X-FORGE-Mode=${cfg.mode}
      X-FORGE-DisplayBackend=wayland
      X-FORGE-Compositor=${cfg.compositor}
      X-FORGE-SafeMode=${boolString cfg.safeMode}
      X-FORGE-Fullscreen=${boolString cfg.fullscreen}
      X-FORGE-AutoStart=false
      X-FORGE-HostMutation=false
      X-FORGE-DirectSystemControl=false
      X-FORGE-ModelMutation=false
      X-FORGE-SemanticMemoryWrite=false
      X-FORGE-ForgeKLiveAuthority=false
    '';
  };
}
