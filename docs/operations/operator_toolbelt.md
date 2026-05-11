# FORGE Operator Toolbelt

Status date: 2026-05-11.

The FORGE operator toolbelt is the Nix-native package set installed by the opt-in operator desktop profile. It is defined in `nix/packages/forge-operator-toolbelt.nix` and included by `nix/nixos/profiles/forge-operator-desktop.nix`.

## Included Tools

Workspace:

- `foot`
- `pcmanfm`
- `micro` or `helix`
- `xarchiver`

Internet and docs:

- `firefox`
- `curl`
- `wget`

AI runtime:

- `ollama`
- fixed modelruntime/core status wrappers

System and host inspection:

- `btop` or `htop`
- `nmap`
- `dnsutils`
- `iproute2`
- `pciutils`
- `usbutils`
- `lsof`
- `strace`

Developer tools:

- `git`
- `lazygit`
- `sqlitebrowser` when available
- `jq`
- `yq`
- `ripgrep`
- `fd`
- `bat`
- `eza`
- `tree`

Optional GPU tooling:

- `nvtop` when available in the current nixpkgs set

Core tools are required by the package and must be present in the Nix closure.
Platform-specific GPU telemetry is optional so unsupported platforms can still
evaluate.

## Fixed Wrappers

The operator desktop launches CLI tools through fixed wrappers:

- `forge-operator-ollama-status`
- `forge-operator-models`
- `forge-operator-btop`
- `forge-operator-lazygit`
- `forge-operator-core-status`
- `forge-operator-core-logs`
- `forge-operator-network-diagnostics`
- `forge-operator-hardware-diagnostics`
- `forge-operator-editor`

These wrappers run fixed commands only. They do not accept command text, executable paths, package install instructions, rebuild requests, service restart requests, model load/unload requests, or semantic memory writes.

## Ollama

Ollama is provided through Nix as part of the operator toolbelt. Do not install it with:

```bash
curl https://ollama.com/install.sh | sh
```

NixOS profiles must keep Ollama installation declarative and reviewable. This pass makes the `ollama` CLI available on `PATH`; it does not enable a mutating model-management UI and does not load or unload models from the operator launcher.

Future service support should be explicit in the NixOS configuration. The safe default is disabled unless the operator intentionally enables an Ollama service profile.

## Forbidden

The toolbelt must not add:

- arbitrary command textboxes
- arbitrary path launchers
- host rebuild buttons
- service restart buttons
- model load/unload buttons
- cleanup/delete buttons
- runtime package installers
- direct semantic memory writes

FORGE UI launchers are allowlisted operator conveniences, not a second command authority plane.
