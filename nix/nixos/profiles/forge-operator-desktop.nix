{
  config,
  lib,
  options,
  pkgs,
  ...
}:

let
  forgeDesktopShell = pkgs.callPackage ../../packages/forge-desktop-shell.nix { };
  forgeShellSession = pkgs.callPackage ../../packages/forge-shell-session.nix {
    forgeDesktopShell = forgeDesktopShell;
  };
  forgeOperatorSession = pkgs.callPackage ../../packages/forge-operator-session.nix {
    forge-shell-session = forgeShellSession;
  };
  forgeOperatorToolbelt = pkgs.callPackage ../../packages/forge-operator-toolbelt.nix {
    inherit pkgs;
  };
  forgeWaylandSession = pkgs.callPackage ../../packages/forge-wayland-session.nix {
    forge-shell-session = forgeShellSession;
  };
  notoEmoji = pkgs.noto-fonts-color-emoji or pkgs.noto-fonts-emoji;
  fileManager = pkgs.pcmanfm;
in
{
  imports = [
    ../modules/forge-shell-session.nix
  ];

  # OPERATOR DESKTOP PROFILE ONLY
  # OPT-IN ONLY
  # FORGE REMAINS THE DESKTOP SURFACE; NIXOS REMAINS THE SUBSTRATE.

  forge.shellSession = {
    enable = lib.mkDefault true;
    package = lib.mkDefault forgeShellSession;
    mode = lib.mkDefault "operator-desktop";
    displayBackend = lib.mkDefault "wayland";
    compositor = lib.mkDefault "labwc";
    autoStart = lib.mkDefault false;
    safeMode = lib.mkDefault true;
    fullscreen = lib.mkDefault false;
    wayland = {
      enable = lib.mkDefault true;
      package = lib.mkDefault pkgs.labwc;
      sessionPackage = lib.mkDefault forgeOperatorSession;
      sessionName = lib.mkDefault "forge-operator";
    };
  };

  services.displayManager.autoLogin.enable = lib.mkDefault false;
  services.dbus.enable = lib.mkDefault true;
  security.polkit.enable = lib.mkDefault true;

  hardware.graphics.enable = lib.mkDefault true;

  xdg.portal = {
    enable = lib.mkDefault true;
    config.common.default = lib.mkDefault "gtk";
    extraPortals = lib.mkDefault [
      pkgs.xdg-desktop-portal-gtk
    ];
  };

  virtualisation.virtualbox.guest = {
    enable = lib.mkDefault true;
  }
  // lib.optionalAttrs (lib.hasAttrByPath [ "virtualisation" "virtualbox" "guest" "x11" ] options) {
    x11 = lib.mkDefault false;
  };

  fonts.packages = lib.mkDefault [
    pkgs.noto-fonts
    notoEmoji
    pkgs.dejavu_fonts
  ];

  environment.systemPackages = [
    forgeOperatorSession
    forgeWaylandSession
    forgeShellSession
    forgeDesktopShell
    forgeOperatorToolbelt
    pkgs.labwc
    pkgs.cage
    pkgs.foot
    fileManager
    pkgs.firefox
    pkgs.dbus
    pkgs.xdg-utils
    pkgs.xdg-desktop-portal
    pkgs.xdg-desktop-portal-gtk
    pkgs.mesa-demos
    pkgs.noto-fonts
    notoEmoji
    pkgs.dejavu_fonts
  ];

  environment.sessionVariables = {
    FORGE_SHELL_SESSION_ENABLED = "true";
    FORGE_SHELL_MODE = "operator-desktop";
    FORGE_SHELL_DISPLAY_BACKEND = "wayland";
    FORGE_SHELL_COMPOSITOR = "labwc";
    FORGE_SHELL_SAFE_MODE = "true";
    FORGE_SHELL_FULLSCREEN = "false";
    FORGE_SHELL_HOST_MUTATION = "false";
    FORGE_SHELL_DIRECT_SYSTEM_CONTROL = "false";
    FORGE_SHELL_MODEL_MUTATION = "false";
    FORGE_SHELL_SEMANTIC_MEMORY_WRITE = "false";
    FORGE_SHELL_FORGE_K_LIVE_AUTHORITY = "false";
    FORGE_CORE_URL = lib.mkDefault "http://127.0.0.1:18492";
    VITE_FORGE_API_URL = lib.mkDefault "http://127.0.0.1:18492";
    XDG_SESSION_TYPE = "wayland";
    GDK_BACKEND = "wayland,x11";
    WEBKIT_DISABLE_DMABUF_RENDERER = "1";
  };

  assertions = [
    {
      assertion = config.forge.shellSession.autoStart == false;
      message = "G6 operator desktop profile must not autostart FORGE Shell.";
    }
    {
      assertion = config.services.displayManager.autoLogin.enable == false;
      message = "G6 operator desktop profile must keep automatic login disabled.";
    }
  ];
}
