{
  config,
  lib,
  pkgs,
  ...
}:

let
  forgeShellSession = pkgs.forge-shell-session;
  forgeWaylandSession = pkgs.forge-wayland-session;
in
{
  imports = [
    ../modules/forge-os.nix
  ];

  # Physical OptiPlex 7000 test target. NixOS/Linux remains the host
  # substrate; FORGE is the governed local shell and service layer.
  system.stateVersion = "26.05";

  boot = {
    initrd.availableKernelModules = [
      "xhci_pci"
      "ahci"
      "nvme"
      "usbhid"
      "usb_storage"
      "sd_mod"
    ];
    kernelModules = [ "kvm-intel" ];
    loader = {
      systemd-boot.enable = true;
      efi.canTouchEfiVariables = true;
      timeout = 3;
    };
    tmp.cleanOnBoot = true;
  };

  fileSystems."/" = {
    device = "/dev/disk/by-label/FORGE_ROOT";
    fsType = "ext4";
  };
  fileSystems."/boot" = {
    device = "/dev/disk/by-label/FORGE_EFI";
    fsType = "vfat";
    options = [
      "fmask=0022"
      "dmask=0022"
    ];
  };
  swapDevices = [
    { device = "/dev/disk/by-label/FORGE_SWAP"; }
  ];

  hardware = {
    cpu.intel.updateMicrocode = lib.mkDefault config.hardware.enableRedistributableFirmware;
    enableRedistributableFirmware = true;
    graphics.enable = true;
  };

  networking = {
    hostName = "forge-optiplex";
    networkmanager.enable = true;
  };
  time.timeZone = "America/Chicago";

  nix.settings = {
    experimental-features = [
      "nix-command"
      "flakes"
    ];
    auto-optimise-store = true;
  };

  forge.os = {
    enable = true;
    storageRoot = "/forge";
    safeMode = true;
  };
  services.forge-core = {
    bindHost = "127.0.0.1";
    enableModelRuntime = true;
    safeModeForceCPUOnly = true;
    extraEnvironment = {
      OLLAMA_BASE_URL = "http://127.0.0.1:11434";
      OLLAMA_MODEL = "gemma3:1b-it-q4_K_M";
      FORGE_MODEL_DEFAULT_BACKEND = "ollama_compat";
      FORGE_MODEL_DEFAULT_ID = "gemma3:1b-it-q4_K_M";
      FORGE_MODEL_MAX_LOADED_MODELS = "1";
      FORGE_OLLAMA_CHAT_NUM_CTX = "1024";
      FORGE_OLLAMA_CHAT_NUM_PREDICT = "96";
      FORGE_OLLAMA_CHAT_NUM_THREAD = "6";
    };
  };

  forge.shellSession = {
    enable = true;
    user = "operator";
    package = forgeShellSession;
    mode = "fullscreen-shell";
    displayBackend = "wayland";
    compositor = "cage";
    autoStart = false;
    safeMode = true;
    fullscreen = true;
    wayland = {
      enable = true;
      package = pkgs.cage;
      sessionPackage = forgeWaylandSession;
      sessionName = "forge-shell";
    };
  };

  services = {
    dbus.enable = true;
    displayManager.autoLogin.enable = lib.mkForce false;
    fstrim.enable = true;
    getty.autologinUser = lib.mkForce null;
    greetd = {
      enable = true;
      settings.default_session = {
        command = "${pkgs.tuigreet}/bin/tuigreet --time --remember --cmd ${forgeWaylandSession}/bin/forge-wayland-session";
        user = "greeter";
      };
    };
    ollama = {
      enable = true;
      home = "/forge/models/ollama";
      host = "127.0.0.1";
      port = 11434;
      loadModels = [
        "gemma3:1b-it-q4_K_M"
        "smuxo/smuxoAI:0.8b"
      ];
      environmentVariables = {
        OLLAMA_KEEP_ALIVE = "5m";
        OLLAMA_MAX_LOADED_MODELS = "1";
        OLLAMA_NUM_PARALLEL = "1";
      };
    };
    openssh = {
      enable = true;
      settings = {
        PasswordAuthentication = false;
        PermitRootLogin = "no";
      };
    };
  };

  security = {
    polkit.enable = true;
    sudo.wheelNeedsPassword = true;
  };
  programs.dconf.enable = true;

  xdg.portal = {
    enable = true;
    config.common.default = "gtk";
    extraPortals = [ pkgs.xdg-desktop-portal-gtk ];
  };

  users.users.operator = {
    isNormalUser = true;
    description = "FORGE local operator";
    extraGroups = [
      "wheel"
      "networkmanager"
      "video"
      "render"
      "forge"
    ];
    openssh.authorizedKeys.keys = [
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJQ8NMfyHtDXYQzKk3hMECIVOKlWRyoLJBismJ6eYQ8v rshort@localhost.localdomain"
    ];
  };

  environment.systemPackages = [
    forgeShellSession
    forgeWaylandSession
    pkgs.cage
    pkgs.curl
    pkgs.git
    pkgs.mesa-demos
    pkgs.vim
  ];

  zramSwap = {
    enable = true;
    memoryPercent = 50;
  };

  services.journald.extraConfig = ''
    SystemMaxUse=256M
    RuntimeMaxUse=64M
  '';

  environment.etc."forge/optiplex-test.env".text = ''
    FORGE_OPTIPLEX_TEST_TARGET=true
    FORGE_CORE_URL=http://127.0.0.1:18492
    FORGE_MODEL_RUNTIME_ENABLED=true
    FORGE_MODEL_DEFAULT_BACKEND=ollama_compat
    FORGE_MODEL_DEFAULT_ID=gemma3:1b-it-q4_K_M
    FORGE_MODEL_MAX_LOADED_MODELS=1
    FORGE_SAFE_MODE_FORCE_CPU_ONLY=true
    FORGE_SHELL_SAFE_MODE=true
    FORGE_SHELL_HOST_MUTATION=false
    FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false
    FORGE_SHELL_MODEL_MUTATION=false
    FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false
    FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false
  '';

  assertions = [
    {
      assertion = config.services.forge-core.bindHost == "127.0.0.1";
      message = "FORGE OptiPlex test core must remain loopback-only.";
    }
    {
      assertion = config.services.forge-core.enableModelRuntime == true;
      message = "FORGE OptiPlex test target must route the selected local model through governed modelruntime.";
    }
    {
      assertion = config.services.displayManager.autoLogin.enable == false;
      message = "FORGE OptiPlex test target must not enable graphical autologin.";
    }
    {
      assertion = config.forge.shellSession.safeMode == true;
      message = "FORGE OptiPlex test shell must remain in safe mode.";
    }
  ];
}
