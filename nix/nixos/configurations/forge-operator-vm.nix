{
  config,
  lib,
  pkgs,
  ...
}:

let
  forgePlymouthTheme = pkgs.runCommand "forge-plymouth-theme" { } ''
    theme_dir="$out/share/plymouth/themes/forge"
    mkdir -p "$theme_dir"
    cat > "$theme_dir/forge.plymouth" <<'EOF'
[Plymouth Theme]
Name=FORGE
Description=FORGE-OS Runtime boot screen
ModuleName=script

[script]
ImageDir=/share/plymouth/themes/forge
ScriptFile=/share/plymouth/themes/forge/forge.script
EOF
    cat > "$theme_dir/forge.script" <<'EOF'
Window.SetBackgroundTopColor(0.02, 0.03, 0.035);
Window.SetBackgroundBottomColor(0.0, 0.0, 0.0);

title = Text("FORGE-OS", 0.95, 0.98, 1.0);
subtitle = Text("Operator Runtime", 0.55, 0.68, 0.72);

title.SetX(Window.GetWidth() / 2 - title.GetWidth() / 2);
title.SetY(Window.GetHeight() / 2 - title.GetHeight() / 2 - 24);
subtitle.SetX(Window.GetWidth() / 2 - subtitle.GetWidth() / 2);
subtitle.SetY(Window.GetHeight() / 2 + 18);
EOF
  '';
in
{
  imports = [
    ../modules/forge-os.nix
    ../profiles/forge-native-desktop-runtime.nix
  ];

  # Canonical local VM target for Nix-first FORGE native desktop bring-up.
  # This is intentionally conservative: boot splash, graphical password login,
  # safe mode, local core bind, no autologin, and no FORGE UI host-mutation path.
  system.stateVersion = "25.11";

  boot.loader.grub.devices = lib.mkDefault [ "/dev/sda" ];
  boot.loader.timeout = lib.mkDefault 0;
  boot.consoleLogLevel = lib.mkDefault 3;
  boot.initrd.verbose = lib.mkDefault false;
  boot.kernelParams = lib.mkDefault [
    "quiet"
    "splash"
    "loglevel=3"
    "udev.log_priority=3"
    "systemd.show_status=false"
    "rd.systemd.show_status=false"
  ];
  boot.plymouth = {
    enable = lib.mkDefault true;
    theme = lib.mkForce "forge";
    themePackages = lib.mkDefault [ forgePlymouthTheme ];
    extraConfig = ''
      ShowDelay=0
    '';
  };
  boot.tmp.cleanOnBoot = lib.mkDefault true;

  nix = {
    settings = {
      experimental-features = lib.mkDefault [
        "nix-command"
        "flakes"
      ];
      max-jobs = lib.mkDefault 6;
      cores = lib.mkDefault 0;
      auto-optimise-store = lib.mkDefault true;
    };
    gc = {
      automatic = lib.mkDefault true;
      dates = lib.mkDefault "weekly";
      options = lib.mkDefault "--delete-older-than 14d";
    };
  };

  fileSystems."/" = {
    device = lib.mkDefault "/dev/disk/by-label/nixos";
    fsType = lib.mkDefault "ext4";
  };

  services.fstrim.enable = lib.mkDefault true;
  zramSwap = {
    enable = lib.mkDefault true;
    memoryPercent = lib.mkDefault 25;
  };

  services.journald.extraConfig = lib.mkDefault ''
    SystemMaxUse=512M
    RuntimeMaxUse=128M
  '';

  networking.hostName = "forge-operator-vm";
  networking.networkmanager.enable = lib.mkDefault true;
  time.timeZone = lib.mkDefault "America/Chicago";

  forge.os = {
    enable = true;
    storageRoot = lib.mkDefault "/forge";
    safeMode = lib.mkDefault true;
  };

  services.forge-core = {
    bindHost = lib.mkDefault "127.0.0.1";
    enableModelRuntime = lib.mkDefault true;
    safeModeForceCPUOnly = lib.mkDefault true;
    extraEnvironment = {
      OLLAMA_BASE_URL = lib.mkDefault "http://127.0.0.1:11434";
      FORGE_MODEL_DEFAULT_BACKEND = lib.mkDefault "ollama_compat";
    };
  };

  services.openssh.enable = lib.mkDefault false;
  services.displayManager.autoLogin.enable = lib.mkForce false;

  users.users.operator = {
    isNormalUser = true;
    description = "FORGE local VM operator";
    initialPassword = lib.mkDefault "forge";
    extraGroups = [
      "wheel"
      "networkmanager"
      "video"
      "render"
      "forge"
    ];
  };

  security.sudo.wheelNeedsPassword = lib.mkDefault true;

  environment.systemPackages = [
    pkgs.git
    pkgs.vim
  ];

  environment.etc."forge/operator-vm.env".text = ''
    FORGE_OPERATOR_VM=true
    FORGE_OPERATOR_VM_CANONICAL=true
    FORGE_OPERATOR_VM_LOGIN_USER=operator
    FORGE_OPERATOR_VM_LOCAL_ONLY=true
    FORGE_CORE_URL=http://127.0.0.1:18492
    OLLAMA_BASE_URL=http://127.0.0.1:11434
    FORGE_MODEL_DEFAULT_BACKEND=ollama_compat
    FORGE_SHELL_MODE=operator-desktop
    FORGE_SHELL_SAFE_MODE=true
    FORGE_SHELL_HOST_MUTATION=false
    FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false
    FORGE_SHELL_MODEL_MUTATION=false
    FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false
    FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false
    FORGE_BOOT_SPLASH=plymouth
    FORGE_BOOT_LOGIN=greetd-regreet
  '';

  services.getty.helpLine = ''
    FORGE operator VM
    Login: operator / forge
    Native desktop: graphical password login starts FORGE
    Recovery shell: forge-operator-session
    Health: curl -fsS http://127.0.0.1:18492/health
  '';

  virtualisation.vmVariant = {
    virtualisation = {
      memorySize = lib.mkDefault 12288;
      cores = lib.mkDefault 6;
      diskSize = lib.mkDefault 32768;
      graphics = lib.mkDefault true;
    };
  };

  assertions = [
    {
      assertion = config.services.displayManager.autoLogin.enable == false;
      message = "FORGE operator VM must not enable display-manager autologin.";
    }
    {
      assertion = config.services.greetd.enable == true;
      message = "FORGE operator VM must start the graphical FORGE session through greetd.";
    }
    {
      assertion = config.boot.plymouth.enable == true;
      message = "FORGE operator VM must present the FORGE boot splash by default.";
    }
    {
      assertion = config.forge.shellSession.safeMode == true;
      message = "FORGE operator VM must keep shell safe mode enabled.";
    }
    {
      assertion = config.services.forge-core.bindHost == "127.0.0.1";
      message = "FORGE operator VM must keep forge-core bound to localhost by default.";
    }
  ];
}
