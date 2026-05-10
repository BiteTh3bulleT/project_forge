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
  forgeWaylandSession = pkgs.callPackage ../../packages/forge-wayland-session.nix {
    forge-shell-session = forgeShellSession;
  };
  notoEmoji = pkgs.noto-fonts-color-emoji or pkgs.noto-fonts-emoji;
in
{
  imports = [
    ../modules/forge-shell-session.nix
  ];

  # TEST PROFILE ONLY
  # OPT-IN ONLY
  # VIRTUALBOX/MINIMAL NIXOS GRAPHICS BRING-UP
  #
  # This profile exists for manual VM testing from a TTY. It adds only the
  # small graphics/session substrate needed to run forge-wayland-session and
  # keeps NixOS/Linux responsible for boot, hardware, graphics, services, and
  # rollback.
  #
  # Package path: forge-wayland-session -> forge-shell-session -> forge-desktop-shell.

  forge.shellSession = {
    enable = lib.mkDefault true;
    package = lib.mkDefault forgeShellSession;
    mode = lib.mkDefault "fullscreen-shell";
    displayBackend = lib.mkDefault "wayland";
    compositor = lib.mkDefault "cage";
    autoStart = lib.mkDefault false;
    safeMode = lib.mkDefault true;
    fullscreen = lib.mkDefault true;
    wayland = {
      enable = lib.mkDefault true;
      package = lib.mkDefault pkgs.cage;
      sessionPackage = lib.mkDefault forgeWaylandSession;
      sessionName = lib.mkDefault "forge-shell";
    };
  };

  services.displayManager.autoLogin.enable = lib.mkDefault false;
  services.dbus.enable = lib.mkDefault true;
  security.polkit.enable = lib.mkDefault true;

  hardware.graphics.enable = lib.mkDefault true;

  xdg.portal = {
    enable = lib.mkDefault true;
    extraPortals = lib.mkDefault [
      pkgs.xdg-desktop-portal-gtk
    ];
  };

  virtualisation.virtualbox.guest =
    {
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
    forgeWaylandSession
    forgeShellSession
    forgeDesktopShell
    pkgs.cage
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
    FORGE_SHELL_MODE = "fullscreen-shell";
    FORGE_SHELL_DISPLAY_BACKEND = "wayland";
    FORGE_SHELL_COMPOSITOR = "cage";
    FORGE_SHELL_SAFE_MODE = "true";
    FORGE_SHELL_FULLSCREEN = "true";
    FORGE_SHELL_HOST_MUTATION = "false";
    FORGE_SHELL_DIRECT_SYSTEM_CONTROL = "false";
    FORGE_SHELL_MODEL_MUTATION = "false";
    FORGE_SHELL_SEMANTIC_MEMORY_WRITE = "false";
    FORGE_SHELL_FORGE_K_LIVE_AUTHORITY = "false";
    FORGE_CORE_URL = lib.mkDefault "http://127.0.0.1:18492";
    VITE_FORGE_API_URL = lib.mkDefault "http://127.0.0.1:18492";
    XDG_SESSION_TYPE = "wayland";
    GDK_BACKEND = "wayland,x11";
  };

  assertions = [
    {
      assertion = config.forge.shellSession.autoStart == false;
      message = "G5 VirtualBox graphics test profile must not autostart FORGE Shell.";
    }
    {
      assertion = config.services.displayManager.autoLogin.enable == false;
      message = "G5 VirtualBox graphics test profile must keep automatic login disabled.";
    }
  ];
}
