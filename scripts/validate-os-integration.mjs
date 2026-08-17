#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

export const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..");

const sourcePaths = [
  "AGENTS.md",
  "package.json",
  ".github/workflows/ci.yml",
  "flake.nix",
  "nix/overlays/default.nix",
  "nix/nixos/profiles/forge-native-desktop-runtime.nix",
  "nix/nixos/profiles/forge-operator-desktop.nix",
  "nix/nixos/profiles/forge-vbox-graphics-test.nix",
  "nix/nixos/configurations/forge-operator-vm.nix",
  "nix/nixos/configurations/forge-optiplex-7000.nix",
  "nix/checks/forge-optiplex-7000.nix",
  "nix/packages/forge-operator-session.nix",
  "docs/runbooks/current_forge_bringup.md",
  "docs/runbooks/forge_operator_desktop_vm.md",
  "docs/runbooks/forge_optiplex_7000_test.md",
  "docs/runbooks/config_reference.md",
];

const checks = [
  packageScriptEquals("package.test-os-integration-script", "test:os-integration", "node --test ./scripts/validate-os-integration.test.mjs"),
  packageScriptEquals("package.validate-os-integration-script", "validate:os-integration", "node ./scripts/validate-os-integration.mjs"),
  packageScriptIncludes("package.validate-local-os-integration-tests", "validate:local", "npm run test:os-integration"),
  packageScriptIncludes("package.validate-local-os-integration", "validate:local", "npm run validate:os-integration"),
  includes("agents.os-integration-command", "AGENTS.md", "npm run validate:os-integration"),
  includes("ci.os-integration-tests", ".github/workflows/ci.yml", "npm run test:os-integration"),
  includes("ci.os-integration-validation", ".github/workflows/ci.yml", "npm run validate:os-integration"),
  includes("flake.operator-desktop-shell-package", "flake.nix", "forge-operator-desktop-shell"),
  includes("flake.operator-shell-session-package", "flake.nix", "forge-operator-shell-session"),
  includes("flake.vm-safe-render-profile", "flake.nix", 'renderProfile = "vm-safe";'),
  includes("flake.operator-vm-configuration", "flake.nix", "forge-operator-vm = nixpkgs.lib.nixosSystem"),
  includes("flake.native-runtime-check", "flake.nix", "forge-native-desktop-runtime = pkgs.callPackage ./nix/checks/forge-native-desktop-runtime.nix { };"),
  includes("overlay.operator-desktop-shell", "nix/overlays/default.nix", "forge-operator-desktop-shell"),
  includes("overlay.operator-shell-session", "nix/overlays/default.nix", "forge-operator-shell-session"),
  includes("native.imports-operator-desktop", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "./forge-operator-desktop.nix"),
  includes("native.plymouth-enabled", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "plymouth = {"),
  includes("native.regreet-enabled", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "programs.regreet = {"),
  includes("native.greetd-enabled", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "services.greetd = {"),
  includes("native.default-session", "nix/nixos/profiles/forge-native-desktop-runtime.nix", 'defaultSession = lib.mkDefault "forge-operator";'),
  includes("native.session-package", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "sessionPackages = [ forgeOperatorSession ];"),
  includes("native.display-manager-autologin-disabled", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "autoLogin.enable = lib.mkForce false;"),
  includes("native.tty-autologin-disabled", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "services.getty.autologinUser = lib.mkForce null;"),
  includes("native.greetd-initial-session-blocked", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "!(config.services.greetd.settings ? initial_session)"),
  includes("native.boot-login-disabled", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "bootLogin = false;"),
  includes("native.empty-desktop-on-boot", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "emptyDesktopOnBoot = true;"),
  includes("native.env-runtime-marker", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "FORGE_NATIVE_DESKTOP_RUNTIME=true"),
  includes("native.env-login-marker", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "FORGE_NATIVE_DESKTOP_LOGIN=greetd-regreet"),
  includes("native.env-tty-fallback", "nix/nixos/profiles/forge-native-desktop-runtime.nix", "FORGE_NATIVE_DESKTOP_TTY_FALLBACK=true"),
  notMatches("native.forbidden-autologin", "nix/nixos/profiles/forge-native-desktop-runtime.nix", /autoLogin\.enable\s*=\s*true|autologinUser\s*=\s*"[^"]+"|initial_session\s*=/),
  includes("operator.vm-safe-render-profile", "nix/nixos/profiles/forge-operator-desktop.nix", "renderProfile = \"vm-safe\";"),
  includes("operator.boot-login-disabled", "nix/nixos/profiles/forge-operator-desktop.nix", "bootLogin = false;"),
  includes("operator.empty-desktop-on-boot", "nix/nixos/profiles/forge-operator-desktop.nix", "emptyDesktopOnBoot = true;"),
  includes("operator.safe-mode-env", "nix/nixos/profiles/forge-operator-desktop.nix", 'FORGE_SHELL_FORGE_K_LIVE_AUTHORITY = "false"'),
  includes("operator.dmabuf-disabled", "nix/nixos/profiles/forge-operator-desktop.nix", 'WEBKIT_DISABLE_DMABUF_RENDERER = "1"'),
  includes("operator-session-shell-binary-unset", "nix/packages/forge-operator-session.nix", "unset FORGE_SHELL_BINARY"),
  includes("vbox.vm-safe-render-profile", "nix/nixos/profiles/forge-vbox-graphics-test.nix", "renderProfile = \"vm-safe\";"),
  includes("vm.imports-native-runtime", "nix/nixos/configurations/forge-operator-vm.nix", "../profiles/forge-native-desktop-runtime.nix"),
  includes("vm.storage-root", "nix/nixos/configurations/forge-operator-vm.nix", 'storageRoot = lib.mkDefault "/forge";'),
  includes("vm.forge-os-safe-mode", "nix/nixos/configurations/forge-operator-vm.nix", "safeMode = lib.mkDefault true;"),
  includes("vm.core-loopback-bind", "nix/nixos/configurations/forge-operator-vm.nix", 'bindHost = lib.mkDefault "127.0.0.1";'),
  includes("vm.core-safe-mode-force-cpu", "nix/nixos/configurations/forge-operator-vm.nix", "safeModeForceCPUOnly = lib.mkDefault true;"),
  includes("vm.modelruntime-enabled", "nix/nixos/configurations/forge-operator-vm.nix", "enableModelRuntime = true;"),
  includes("vm.ollama-enabled", "nix/nixos/configurations/forge-operator-vm.nix", "services.ollama = {\n    enable = lib.mkDefault true;"),
  includes("vm.ollama-model-home", "nix/nixos/configurations/forge-operator-vm.nix", 'home = lib.mkDefault "/forge/models/ollama";'),
  includes("vm.ollama-loopback", "nix/nixos/configurations/forge-operator-vm.nix", 'host = lib.mkDefault "127.0.0.1";'),
  includes("vm.ollama-port", "nix/nixos/configurations/forge-operator-vm.nix", "port = lib.mkDefault 11434;"),
  includes("vm.default-model-backend", "nix/nixos/configurations/forge-operator-vm.nix", 'FORGE_MODEL_DEFAULT_BACKEND = lib.mkDefault "ollama_compat";'),
  includes("vm.operator-env-default-backend", "nix/nixos/configurations/forge-operator-vm.nix", "FORGE_MODEL_DEFAULT_BACKEND=ollama_compat"),
  includes("vm.openssh-disabled", "nix/nixos/configurations/forge-operator-vm.nix", "services.openssh.enable = lib.mkDefault false;"),
  includes("vm.display-manager-autologin-disabled", "nix/nixos/configurations/forge-operator-vm.nix", "services.displayManager.autoLogin.enable = lib.mkForce false;"),
  includes("vm.host-mutation-disabled", "nix/nixos/configurations/forge-operator-vm.nix", "FORGE_SHELL_HOST_MUTATION=false"),
  includes("vm.system-control-disabled", "nix/nixos/configurations/forge-operator-vm.nix", "FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false"),
  includes("vm.model-mutation-disabled", "nix/nixos/configurations/forge-operator-vm.nix", "FORGE_SHELL_MODEL_MUTATION=false"),
  includes("vm.semantic-memory-write-disabled", "nix/nixos/configurations/forge-operator-vm.nix", "FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false"),
  includes("vm.forge-k-live-authority-disabled", "nix/nixos/configurations/forge-operator-vm.nix", "FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false"),
  notMatches("vm.forbidden-host-mutation", "nix/nixos/configurations/forge-operator-vm.nix", /autoLogin\.enable\s*=\s*true|services\.openssh\.enable\s*=\s*true|initial_session\s*=|nixos-rebuild|systemctl (restart|stop|start)|modprobe|rmmod|rm -rf/),
  includes("optiplex.flake-configuration", "flake.nix", "forge-optiplex-7000 = nixpkgs.lib.nixosSystem"),
  includes("optiplex.flake-check", "flake.nix", "forge-optiplex-7000 = pkgs.callPackage ./nix/checks/forge-optiplex-7000.nix { };"),
  includes("optiplex.nftables-enabled", "nix/nixos/configurations/forge-optiplex-7000.nix", "nftables = {"),
  includes("optiplex.output-default-drop", "nix/nixos/configurations/forge-optiplex-7000.nix", "type filter hook output priority 0; policy drop;"),
  includes("optiplex.direct-subnet-only", "nix/nixos/configurations/forge-optiplex-7000.nix", 'oifname "enp0s31f6" ip daddr 192.168.50.0/24 accept'),
  includes("optiplex.ssh-direct-only", "nix/nixos/configurations/forge-optiplex-7000.nix", 'iifname "enp0s31f6" ip saddr 192.168.50.0/24 tcp dport 22 accept'),
  includes("optiplex.nix-check-firewall", "nix/checks/forge-optiplex-7000.nix", "table inet forge_offline"),
  includes("optiplex.runbook-kernel-status", "docs/runbooks/forge_optiplex_7000_test.md", 'forge_k_sole_live_authority'),
  includes("optiplex.runbook-negative-egress", "docs/runbooks/forge_optiplex_7000_test.md", "! curl --connect-timeout 3 http://1.1.1.1"),
  includes("optiplex.developer-workspace", "nix/nixos/configurations/forge-optiplex-7000.nix", 'forge.storage.workspaceMode = "2770";'),
  includes("optiplex.office-suite", "nix/nixos/configurations/forge-optiplex-7000.nix", "libreoffice"),
  includes("optiplex.offline-ide", "nix/nixos/configurations/forge-optiplex-7000.nix", "vscode-with-extensions.override"),
  includes("optiplex.project-toolchains", "nix/nixos/configurations/forge-optiplex-7000.nix", "workstationDevelopmentPackages"),
  includes("optiplex.desktop-audio", "nix/nixos/configurations/forge-optiplex-7000.nix", "pipewire = {"),
  includes("optiplex.printing-scanning", "nix/nixos/configurations/forge-optiplex-7000.nix", "hardware.sane = {"),
  includes("optiplex.removable-media", "nix/nixos/configurations/forge-optiplex-7000.nix", "udisks2.enable = true;"),
  includes("optiplex.rootless-containers", "nix/nixos/configurations/forge-optiplex-7000.nix", "virtualisation.podman = {"),
  includes("optiplex.workstation-runbook", "docs/runbooks/forge_optiplex_7000_test.md", "Full development workstation contract"),
  includes("runbook.native-desktop-path", "docs/runbooks/current_forge_bringup.md", "Native Desktop VM Path"),
  includes("runbook.os-integration-test-command", "docs/runbooks/current_forge_bringup.md", "npm run test:os-integration"),
  includes("runbook.password-login", "docs/runbooks/current_forge_bringup.md", "graphical password login"),
  includes("operator-runbook.governed-model-controls", "docs/runbooks/forge_operator_desktop_vm.md", "governed modelruntime approval-gated model load/unload controls"),
  includes("operator-runbook.no-service-rebuild-controls", "docs/runbooks/forge_operator_desktop_vm.md", "Do not add service restart or rebuild controls to the UI."),
  notMatches("operator-runbook.no-stale-model-control-ban", "docs/runbooks/forge_operator_desktop_vm.md", /Do not add model load\/unload, service restart, or rebuild controls to the UI\./),
  includes("config.os-integration-test-command", "docs/runbooks/config_reference.md", "npm run test:os-integration"),
  includes("config.os-integration-command", "docs/runbooks/config_reference.md", "npm run validate:os-integration"),
];

export function loadOSIntegrationSources(rootDir = repoRoot) {
  const out = {};
  for (const path of sourcePaths) {
    const abs = join(rootDir, path);
    if (!existsSync(abs)) {
      out[path] = null;
      continue;
    }
    out[path] = readFileSync(abs, "utf8");
  }
  return out;
}

export function validateOSIntegrationSources(sources) {
  const failures = [];
  const passed = [];
  for (const check of checks) {
    const source = sources[check.path];
    if (source == null) {
      failures.push({
        id: check.id,
        path: check.path,
        message: `missing required source ${check.path}`,
      });
      continue;
    }
    let ok;
    try {
      ok = check.test(source);
    } catch (error) {
      failures.push({
        id: check.id,
        path: check.path,
        message: `check threw: ${error instanceof Error ? error.message : String(error)}`,
      });
      continue;
    }
    if (ok) {
      passed.push(check.id);
    } else {
      failures.push({
        id: check.id,
        path: check.path,
        message: check.message,
      });
    }
  }
  return { ok: failures.length === 0, checks: passed, failures };
}

export function validateOSIntegration(rootDir = repoRoot) {
  return validateOSIntegrationSources(loadOSIntegrationSources(rootDir));
}

function includes(id, path, needle) {
  return {
    id,
    path,
    test: (source) => source.includes(needle),
    message: `${path} must contain ${JSON.stringify(needle)}`,
  };
}

function notMatches(id, path, pattern) {
  return {
    id,
    path,
    test: (source) => !pattern.test(source),
    message: `${path} must not match ${pattern}`,
  };
}

function packageScriptEquals(id, scriptName, expected) {
  return {
    id,
    path: "package.json",
    test: (source) => {
      const pkg = JSON.parse(source);
      return pkg.scripts?.[scriptName] === expected;
    },
    message: `package.json script ${scriptName} must equal ${JSON.stringify(expected)}`,
  };
}

function packageScriptIncludes(id, scriptName, needle) {
  return {
    id,
    path: "package.json",
    test: (source) => {
      const pkg = JSON.parse(source);
      return String(pkg.scripts?.[scriptName] ?? "").includes(needle);
    },
    message: `package.json script ${scriptName} must contain ${JSON.stringify(needle)}`,
  };
}

function printResult(result) {
  if (result.ok) {
    console.log(`OS integration readiness OK (${result.checks.length} checks).`);
    return;
  }
  console.error(`OS integration readiness FAILED (${result.failures.length} failure(s)).`);
  for (const failure of result.failures) {
    console.error(`- ${failure.id} [${failure.path}]: ${failure.message}`);
  }
}

function usage() {
  console.log(`usage: node scripts/validate-os-integration.mjs [--json]

Validates static FORGE native desktop / operator VM OS integration invariants.
This complements Nix flake checks and runs on hosts where nix is unavailable.`);
}

if (process.argv[1] && relative(repoRoot, fileURLToPath(import.meta.url)) === relative(repoRoot, process.argv[1])) {
  if (process.argv.includes("-h") || process.argv.includes("--help")) {
    usage();
    process.exit(0);
  }
  const result = validateOSIntegration(repoRoot);
  if (process.argv.includes("--json")) {
    console.log(JSON.stringify(result, null, 2));
  } else {
    printResult(result);
  }
  process.exit(result.ok ? 0 : 1);
}
