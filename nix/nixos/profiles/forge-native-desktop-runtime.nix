{
  config,
  lib,
  pkgs,
  ...
}:

let
  forgeDesktopShell = pkgs.callPackage ../../packages/forge-desktop-shell.nix {
    renderProfile = "vm-safe";
    bootLogin = false;
    emptyDesktopOnBoot = true;
  };
  forgeShellSession = pkgs.callPackage ../../packages/forge-shell-session.nix {
    forgeDesktopShell = forgeDesktopShell;
  };
  forgeOperatorSession = pkgs.callPackage ../../packages/forge-operator-session.nix {
    forge-shell-session = forgeShellSession;
  };
in
{
  imports = [
    ./forge-operator-desktop.nix
  ];

  # Native FORGE desktop runtime:
  # boot splash -> graphical password login -> FORGE operator desktop.
  # NixOS/Linux remains the substrate and recovery path.
  boot = {
    plymouth = {
      enable = lib.mkDefault true;
      theme = lib.mkDefault "bgrt";
      logo = lib.mkDefault ../../../apps/desktop/public/brand/forge-start-button.png;
    };
    kernelParams = lib.mkDefault [
      "quiet"
      "splash"
      "loglevel=3"
      "udev.log_level=3"
    ];
  };

  programs.regreet = {
    enable = lib.mkDefault true;
    extraCss = ''
      window {
        background: #05070a;
      }

      box {
        color: #d8f7ff;
      }
    '';
    settings = {
      background = {
        path = toString ../../../apps/desktop/public/brand/forge-horizontal.png;
        fit = "Contain";
      };
      GTK = {
        application_prefer_dark_theme = true;
      };
    };
  };

  services.greetd = {
    enable = lib.mkDefault true;
    restart = lib.mkDefault true;
  };

  services.displayManager = {
    enable = lib.mkDefault true;
    autoLogin.enable = lib.mkForce false;
    autoLogin.user = lib.mkForce null;
    defaultSession = lib.mkDefault "forge-operator";
    sessionPackages = [ forgeOperatorSession ];
  };
  services.getty.autologinUser = lib.mkForce null;

  environment.etc."forge/native-desktop-runtime.env".text = ''
    FORGE_NATIVE_DESKTOP_RUNTIME=true
    FORGE_NATIVE_DESKTOP_BOOT_SPLASH=plymouth
    FORGE_NATIVE_DESKTOP_LOGIN=greetd-regreet
    FORGE_NATIVE_DESKTOP_DEFAULT_SESSION=forge-operator
    FORGE_NATIVE_DESKTOP_AUTOLOGIN=false
    FORGE_NATIVE_DESKTOP_TTY_FALLBACK=true
  '';

  assertions = [
    {
      assertion = config.boot.plymouth.enable == true;
      message = "FORGE native desktop runtime requires Plymouth boot splash.";
    }
    {
      assertion = config.services.greetd.enable == true;
      message = "FORGE native desktop runtime requires greetd graphical login.";
    }
    {
      assertion = config.programs.regreet.enable == true;
      message = "FORGE native desktop runtime requires ReGreet graphical greeter.";
    }
    {
      assertion = !(config.services.greetd.settings ? initial_session);
      message = "FORGE native desktop runtime must not configure greetd initial-session autologin.";
    }
    {
      assertion = config.services.displayManager.autoLogin.enable == false;
      message = "FORGE native desktop runtime must not enable autologin.";
    }
    {
      assertion = config.services.getty.autologinUser == null;
      message = "FORGE native desktop runtime must not enable TTY autologin.";
    }
  ];
}
