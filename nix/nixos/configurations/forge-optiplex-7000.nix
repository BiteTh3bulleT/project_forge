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
    inherit forgeDesktopShell;
  };
  forgeOperatorSession = pkgs.callPackage ../../packages/forge-operator-session.nix {
    forge-shell-session = forgeShellSession;
    polkitAgent = pkgs.polkit_gnome;
    notificationDaemon = pkgs.mako;
  };
  forgeOperatorToolbelt = pkgs.callPackage ../../packages/forge-operator-toolbelt.nix {
    inherit pkgs;
  };
  # Keep the single-application Cage path installed as a rollback session.
  forgeWaylandSession = pkgs.callPackage ../../packages/forge-wayland-session.nix {
    forge-shell-session = forgeShellSession;
  };
  notoEmoji = pkgs.noto-fonts-color-emoji or pkgs.noto-fonts-emoji;
  forgeIDE = pkgs.vscode-with-extensions.override {
    vscode = pkgs.vscodium;
    vscodeExtensions = with pkgs.vscode-extensions; [
      golang.go
      rust-lang.rust-analyzer
      ms-python.python
      redhat.vscode-yaml
      tamasfe.even-better-toml
      jnoortheen.nix-ide
      esbenp.prettier-vscode
    ];
  };
  workstationDesktopPackages = with pkgs; [
    libreoffice
    hunspell
    hunspellDicts.en_US
    thunderbird
    evince
    xournalpp
    gimp
    inkscape
    mpv
    imv
    grim
    slurp
    wl-clipboard
    keepassxc
    qalculate-gtk
    simple-scan
    pavucontrol
    networkmanagerapplet
    xdg-user-dirs
    mako
    polkit_gnome
  ];
  workstationDevelopmentPackages = with pkgs; [
    forgeIDE
    go
    gopls
    gotools
    delve
    nodejs_20
    typescript
    typescript-language-server
    prettier
    eslint
    rustc
    cargo
    rustfmt
    clippy
    rust-analyzer
    cargo-tauri
    cargo-watch
    cargo-edit
    python3
    uv
    ruff
    black
    gcc
    clang
    gdb
    lldb
    cmake
    ninja
    gnumake
    pkg-config
    shellcheck
    shfmt
    nil
    nixfmt
    just
    hyperfine
    sqlite
    postgresql
    redis
    gh
    git-lfs
    fzf
    httpie
    openssl
    gnupg
    pandoc
    glib.dev
    gtk3.dev
    webkitgtk_4_1.dev
    libsoup_3.dev
    openssl.dev
    cairo.dev
    pango.dev
    gdk-pixbuf.dev
    librsvg.dev
  ];
  workstationOperationsPackages = with pkgs; [
    podman
    buildah
    skopeo
    restic
    borgbackup
    rsync
    p7zip
    unzip
    zip
    lm_sensors
    nvme-cli
    smartmontools
    intel-gpu-tools
    powertop
  ];
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

    # Hard offline boundary: the target may talk only to itself and to the
    # directly attached 192.168.50.0/24 commissioning network. Additional
    # NetworkManager profiles or interfaces cannot create an egress path.
    firewall.enable = false;
    nftables = {
      enable = true;
      ruleset = ''
        table inet forge_offline {
          chain input {
            type filter hook input priority 0; policy drop;
            iifname "lo" accept
            ct state established,related accept
            iifname "enp0s31f6" ip saddr 192.168.50.0/24 icmp type echo-request accept
            iifname "enp0s31f6" ip saddr 192.168.50.0/24 tcp dport 22 accept
          }

          chain forward {
            type filter hook forward priority 0; policy drop;
          }

          chain output {
            type filter hook output priority 0; policy drop;
            oifname "lo" accept
            ct state established,related accept
            oifname "enp0s31f6" ip daddr 192.168.50.0/24 accept
          }
        }
      '';
    };
    networkmanager = {
      enable = true;
      ensureProfiles.profiles."forge-direct-link" = {
        connection = {
          id = "forge-direct-link";
          type = "ethernet";
          interface-name = "enp0s31f6";
          autoconnect = true;
          autoconnect-priority = 100;
        };
        ipv4 = {
          method = "manual";
          address1 = "192.168.50.2/24";
          gateway = "";
          dns = "";
          ignore-auto-dns = true;
          never-default = true;
        };
        ipv6.method = "disabled";
      };
    };
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
  forge.storage.workspaceMode = "2770";
  services.forge-core = {
    bindHost = "127.0.0.1";
    enableModelRuntime = true;
    safeModeForceCPUOnly = true;
    extraEnvironment = {
      OLLAMA_BASE_URL = "http://127.0.0.1:11434";
      OLLAMA_MODEL = "smuxo/smuxoAI:0.8b";
      FORGE_MODEL_DEFAULT_BACKEND = "ollama_compat";
      FORGE_MODEL_DEFAULT_ID = "smuxo/smuxoAI:0.8b";
      FORGE_MODEL_MAX_LOADED_MODELS = "1";
      FORGE_OLLAMA_CHAT_NUM_CTX = "2048";
      FORGE_OLLAMA_CHAT_NUM_PREDICT = "192";
      FORGE_OLLAMA_CHAT_NUM_THREAD = "6";
      FORGE_OLLAMA_CHAT_THINK = "false";
    };
  };

  forge.shellSession = {
    enable = true;
    user = "operator";
    package = forgeShellSession;
    mode = "operator-desktop";
    displayBackend = "wayland";
    compositor = "labwc";
    autoStart = false;
    safeMode = true;
    fullscreen = false;
    wayland = {
      enable = true;
      package = pkgs.labwc;
      sessionPackage = forgeOperatorSession;
      sessionName = "forge-operator";
    };
  };

  services = {
    dbus.enable = true;
    displayManager.autoLogin.enable = lib.mkForce false;
    fstrim.enable = true;
    fwupd.enable = true;
    gvfs.enable = true;
    udisks2.enable = true;
    tumbler.enable = true;
    thermald.enable = true;
    power-profiles-daemon.enable = true;
    printing = {
      enable = true;
      drivers = [ pkgs.gutenprint ];
    };
    pipewire = {
      enable = true;
      alsa.enable = true;
      alsa.support32Bit = true;
      pulse.enable = true;
    };
    gnome.gnome-keyring.enable = true;
    getty.autologinUser = lib.mkForce null;
    greetd = {
      enable = true;
      settings.default_session = {
        command = "${pkgs.tuigreet}/bin/tuigreet --time --remember --cmd ${forgeOperatorSession}/bin/forge-operator-session";
        user = "greeter";
      };
    };
    ollama = {
      enable = true;
      user = "ollama";
      group = "forge";
      home = "/forge/models/ollama";
      host = "127.0.0.1";
      port = 11434;
      # Models are staged from the build workstation. NixOS loadModels always
      # checks the remote registry, so it is intentionally disabled here.
      loadModels = [ ];
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
    rtkit.enable = true;
    sudo.wheelNeedsPassword = true;
  };
  hardware.sane = {
    enable = true;
    extraBackends = [ pkgs.sane-airscan ];
  };
  systemd.tmpfiles.rules = [
    "d /forge/models/ollama 0750 ollama forge -"
    "d /forge/models/ollama/models 0750 ollama forge -"
    "d /forge/workspaces/default/Projects 2770 operator forge -"
  ];
  programs = {
    bash.completion.enable = true;
    dconf.enable = true;
    direnv = {
      enable = true;
      nix-direnv.enable = true;
    };
    gnupg.agent.enable = true;
    nix-ld.enable = true;
  };

  virtualisation.podman = {
    enable = true;
    dockerCompat = true;
    defaultNetwork.settings.dns_enabled = false;
  };

  fonts.packages = [
    pkgs.noto-fonts
    notoEmoji
    pkgs.dejavu_fonts
    pkgs.liberation_ttf
    pkgs.carlito
    pkgs.caladea
    pkgs.jetbrains-mono
  ];

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
      "lp"
      "scanner"
    ];
    subUidRanges = [
      {
        startUid = 100000;
        count = 65536;
      }
    ];
    subGidRanges = [
      {
        startGid = 100000;
        count = 65536;
      }
    ];
    openssh.authorizedKeys.keys = [
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJQ8NMfyHtDXYQzKk3hMECIVOKlWRyoLJBismJ6eYQ8v rshort@localhost.localdomain"
    ];
  };

  environment.systemPackages = [
    forgeDesktopShell
    forgeShellSession
    forgeOperatorSession
    forgeOperatorToolbelt
    pkgs.labwc
    pkgs.lswt
    pkgs.wlrctl
    # The Cage session remains available for rollback and fullscreen testing.
    forgeWaylandSession
    pkgs.cage
    pkgs.curl
    pkgs.dbus
    pkgs.git
    pkgs.jq
    pkgs.mesa-demos
    pkgs.xdg-utils
    pkgs.xdg-desktop-portal
    pkgs.xdg-desktop-portal-gtk
    pkgs.vim
  ]
  ++ workstationDesktopPackages
  ++ workstationDevelopmentPackages
  ++ workstationOperationsPackages;

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
    FORGE_RENDER_PROFILE = "vm-safe";
    VITE_FORGE_RENDER_PROFILE = "vm-safe";
    FORGE_CORE_URL = "http://127.0.0.1:18492";
    FORGE_WORKSPACE_DIR = "/forge/workspaces/default";
    VITE_FORGE_API_URL = "http://127.0.0.1:18492";
    EDITOR = "hx";
    VISUAL = "codium";
    XDG_SESSION_TYPE = "wayland";
    GDK_BACKEND = "wayland,x11";
    WEBKIT_DISABLE_DMABUF_RENDERER = "1";
  };

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
    FORGE_OPTIPLEX_NETWORK_MODE=offline-direct
    FORGE_OPTIPLEX_DIRECT_ADDRESS=192.168.50.2/24
    FORGE_OPTIPLEX_DEFAULT_ROUTE=false
    FORGE_OPTIPLEX_EGRESS_POLICY=loopback-and-192.168.50.0/24-only
    FORGE_CORE_URL=http://127.0.0.1:18492
    FORGE_MODEL_RUNTIME_ENABLED=true
    FORGE_MODEL_DEFAULT_BACKEND=ollama_compat
    FORGE_MODEL_DEFAULT_ID=smuxo/smuxoAI:0.8b
    FORGE_MODEL_SECONDARY_ID=gemma3:1b-it-q4_K_M
    FORGE_MODEL_MAX_LOADED_MODELS=1
    FORGE_SAFE_MODE_FORCE_CPU_ONLY=true
    FORGE_SHELL_SAFE_MODE=true
    FORGE_SHELL_MODE=operator-desktop
    FORGE_SHELL_COMPOSITOR=labwc
    FORGE_SHELL_FULLSCREEN=false
    FORGE_RENDER_PROFILE=vm-safe
    WEBKIT_DISABLE_DMABUF_RENDERER=1
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
      assertion =
        config.networking.networkmanager.ensureProfiles.profiles."forge-direct-link".ipv4.never-default
        == true;
      message = "FORGE OptiPlex direct test link must never install a default route.";
    }
    {
      assertion = config.networking.nftables.enable == true;
      message = "FORGE OptiPlex offline target must enforce its nftables egress allowlist.";
    }
    {
      assertion = lib.hasInfix "policy drop;" config.networking.nftables.ruleset;
      message = "FORGE OptiPlex nftables policy must fail closed.";
    }
    {
      assertion = config.services.forge-core.enableModelRuntime == true;
      message = "FORGE OptiPlex test target must route the selected local model through governed modelruntime.";
    }
    {
      assertion = config.forge.storage.workspaceMode == "2770";
      message = "FORGE OptiPlex development workspace must remain setgid and group-writable for operator and forge-core collaboration.";
    }
    {
      assertion =
        config.services.pipewire.enable && config.services.printing.enable && config.hardware.sane.enable;
      message = "FORGE OptiPlex workstation must retain audio, printing, and scanning support.";
    }
    {
      assertion = config.virtualisation.podman.enable && config.virtualisation.podman.dockerCompat;
      message = "FORGE OptiPlex workstation must retain rootless container tooling without a daemon.";
    }
    {
      assertion = !(config.systemd.services ? ollama-model-loader);
      message = "FORGE OptiPlex offline target must not enable the registry-backed Ollama model loader.";
    }
    {
      assertion = config.services.displayManager.autoLogin.enable == false;
      message = "FORGE OptiPlex test target must not enable graphical autologin.";
    }
    {
      assertion = config.forge.shellSession.safeMode == true;
      message = "FORGE OptiPlex test shell must remain in safe mode.";
    }
    {
      assertion = config.forge.shellSession.mode == "operator-desktop";
      message = "FORGE OptiPlex native application launchers require the operator desktop session.";
    }
    {
      assertion = config.forge.shellSession.compositor == "labwc";
      message = "FORGE OptiPlex operator desktop must use the bounded Labwc session wrapper.";
    }
  ];
}
