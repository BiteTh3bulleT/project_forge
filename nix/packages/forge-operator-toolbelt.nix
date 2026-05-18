{
  lib,
  pkgs,
  symlinkJoin,
  writeShellApplication,
}:

let
  requiredPackage =
    name:
    let
      value = lib.attrByPath [ name ] null pkgs;
    in
    if value != null then
      value
    else
      throw "forge-operator-toolbelt required package is unavailable in nixpkgs: ${name}";
  requiredPackages = names: builtins.map requiredPackage names;

  optionalPackage =
    name:
    let
      value = lib.attrByPath [ name ] null pkgs;
    in
    lib.optional (value != null) value;
  optionalPackages = names: lib.concatMap optionalPackage names;

  coreToolNames = [
    "foot"
    "pcmanfm"
    "mousepad"
    "firefox"
    "ollama"
    "btop"
    "htop"
    "sqlitebrowser"
    "jq"
    "yq"
    "ripgrep"
    "fd"
    "bat"
    "eza"
    "tree"
    "git"
    "lazygit"
    "curl"
    "wget"
    "nmap"
    "dnsutils"
    "iproute2"
    "pciutils"
    "usbutils"
    "lsof"
    "strace"
    "micro"
    "helix"
    "xarchiver"
  ];

  optionalGpuToolNames = [
    "nvtop"
  ];

  terminalPause = ''
    printf '\nPress Enter to close this operator tool...'
    IFS= read -r _ || true
  '';

  operatorBtop = writeShellApplication {
    name = "forge-operator-btop";
    runtimeInputs = optionalPackages [
      "btop"
      "htop"
    ];
    text = ''
      if command -v btop >/dev/null 2>&1; then
        exec btop
      fi
      if command -v htop >/dev/null 2>&1; then
        exec htop
      fi
      echo "No btop/htop package is available in this operator profile." >&2
      ${terminalPause}
    '';
  };

  operatorLazygit = writeShellApplication {
    name = "forge-operator-lazygit";
    runtimeInputs = optionalPackages [ "lazygit" ];
    text = ''
      if ! command -v lazygit >/dev/null 2>&1; then
        echo "lazygit is not available in this operator profile." >&2
        ${terminalPause}
        exit 0
      fi
      if [ -d /projectforge/.git ]; then
        cd /projectforge
      fi
      exec lazygit
    '';
  };

  operatorOllamaStatus = writeShellApplication {
    name = "forge-operator-ollama-status";
    runtimeInputs = optionalPackages [ "ollama" ];
    text = ''
      if ! command -v ollama >/dev/null 2>&1; then
        echo "Ollama is not available in this operator profile." >&2
        ${terminalPause}
        exit 0
      fi
      echo "Ollama processes:"
      ollama ps || true
      echo
      echo "Installed Ollama models:"
      ollama list || true
      ${terminalPause}
    '';
  };

  operatorModels = writeShellApplication {
    name = "forge-operator-models";
    runtimeInputs = optionalPackages [
      "curl"
      "jq"
    ];
    text = ''
      core_url="''${FORGE_CORE_URL:-http://127.0.0.1:18492}"
      echo "FORGE modelruntime status from $core_url"
      if command -v jq >/dev/null 2>&1; then
        curl -fsS "$core_url/forge/models" | jq . || true
      else
        curl -fsS "$core_url/forge/models" || true
      fi
      ${terminalPause}
    '';
  };

  operatorCoreStatus = writeShellApplication {
    name = "forge-operator-core-status";
    runtimeInputs = optionalPackages [
      "curl"
      "jq"
    ];
    text = ''
      core_url="''${FORGE_CORE_URL:-http://127.0.0.1:18492}"
      echo "FORGE core health from $core_url"
      if command -v jq >/dev/null 2>&1; then
        curl -fsS "$core_url/health" | jq . || true
      else
        curl -fsS "$core_url/health" || true
      fi
      ${terminalPause}
    '';
  };

  operatorCoreLogs = writeShellApplication {
    name = "forge-operator-core-logs";
    runtimeInputs = optionalPackages [ "systemd" ];
    text = ''
      echo "Recent forge-core logs, if journal access is available."
      if command -v journalctl >/dev/null 2>&1; then
        journalctl -u forge-core.service -n 200 --no-pager || \
          journalctl --user -u forge-core.service -n 200 --no-pager || true
      else
        echo "journalctl is not available in this operator profile." >&2
      fi
      ${terminalPause}
    '';
  };

  operatorNetworkDiagnostics = writeShellApplication {
    name = "forge-operator-network-diagnostics";
    runtimeInputs = optionalPackages [
      "iproute2"
      "dnsutils"
      "curl"
    ];
    text = ''
      echo "Addresses:"
      ip addr show || true
      echo
      echo "Routes:"
      ip route show || true
      echo
      echo "Listening TCP/UDP sockets:"
      ss -tulpen || true
      echo
      echo "DNS localhost lookup:"
      dig localhost || true
      ${terminalPause}
    '';
  };

  operatorHardwareDiagnostics = writeShellApplication {
    name = "forge-operator-hardware-diagnostics";
    runtimeInputs = optionalPackages [
      "pciutils"
      "usbutils"
      "lsof"
    ];
    text = ''
      echo "PCI devices:"
      lspci || true
      echo
      echo "USB devices:"
      lsusb || true
      echo
      echo "Open files for this shell:"
      lsof -p "$$" || true
      ${terminalPause}
    '';
  };

  wrappers = [
    operatorBtop
    operatorLazygit
    operatorOllamaStatus
    operatorModels
    operatorCoreStatus
    operatorCoreLogs
    operatorNetworkDiagnostics
    operatorHardwareDiagnostics
  ];
in
symlinkJoin {
  name = "forge-operator-toolbelt";
  paths = requiredPackages coreToolNames ++ optionalPackages optionalGpuToolNames ++ wrappers;
  passthru = {
    requiredToolNames = coreToolNames;
    optionalGpuToolNames = optionalGpuToolNames;
    wrapperNames = builtins.map (pkg: pkg.name) wrappers;
  };

  meta = with lib; {
    description = "Nix-native FORGE operator desktop toolbelt and fixed command wrappers";
    license = licenses.mit;
    platforms = platforms.linux;
  };
}
