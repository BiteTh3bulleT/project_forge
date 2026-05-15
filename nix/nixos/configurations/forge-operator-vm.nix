{
  config,
  lib,
  pkgs,
  ...
}:

{
  imports = [
    ../modules/forge-os.nix
    ../profiles/forge-operator-desktop.nix
  ];

  # Canonical local VM target for Nix-first FORGE operator bring-up.
  # This is intentionally conservative: manual login, safe mode, local core
  # bind, no display-manager autologin, and no FORGE UI host-mutation path.
  system.stateVersion = "25.11";

  boot.loader.grub.devices = lib.mkDefault [ "/dev/sda" ];
  fileSystems."/" = {
    device = lib.mkDefault "/dev/disk/by-label/nixos";
    fsType = lib.mkDefault "ext4";
  };

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
    enableModelRuntime = true;
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
  '';

  services.getty.helpLine = ''
    FORGE operator VM
    Login: operator / forge
    Start shell: forge-operator-session
    Health: curl -fsS http://127.0.0.1:18492/health
  '';

  virtualisation.vmVariant = {
    virtualisation = {
      memorySize = lib.mkDefault 4096;
      cores = lib.mkDefault 4;
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
      assertion = config.forge.shellSession.safeMode == true;
      message = "FORGE operator VM must keep shell safe mode enabled.";
    }
    {
      assertion = config.services.forge-core.bindHost == "127.0.0.1";
      message = "FORGE operator VM must keep forge-core bound to localhost by default.";
    }
  ];
}
