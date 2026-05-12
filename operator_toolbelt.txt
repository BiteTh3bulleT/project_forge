# FORGE Operator Desktop — Full Operator Toolbelt

## Mission

Expand the FORGE operator desktop from a tiny launcher into a real operator workstation profile.

The operator desktop must include the tools required to manage FORGE, inspect the host, run local models, debug services, edit files, inspect databases, view logs, and operate safely.

## Read First

- `apps/desktop/src-tauri/src/main.rs`
- `apps/desktop/src/layout/AppShell.tsx`
- `apps/desktop/src/lib/desktop.ts`
- `nix/nixos/modules/forge-shell-session.nix`
- `nix/nixos/modules/forge-os.nix`
- `nix/packages/forge-shell-session.nix`
- any existing desktop/operator Nix checks

## Current State

The Tauri operator app allowlist currently includes only:

- Terminal / `foot`
- Files / `pcmanfm`
- Browser / `firefox`

This is not enough for a real FORGE operator environment.

## Goals

Add a complete operator toolbelt with safe, allowlisted tools.

The operator should have:

### Workspace

- terminal
- file manager
- text editor
- code editor if available
- archive manager

### Internet / Docs

- browser
- docs viewer or markdown viewer if available
- REST/API testing path, even if initially browser/curl based

### AI Runtime

- Ollama CLI available on PATH
- Ollama service/profile support if appropriate
- model runtime status access
- GPU/VRAM telemetry tools where available

### System / Host

- process monitor
- disk usage viewer
- logs viewer
- service/status helper
- network diagnostics
- hardware diagnostics

### Developer Tools

- git
- lazygit
- sqlite browser
- jq/yq
- ripgrep/fd/bat/eza/tree
- curl/wget
- Go/Node/Rust/Python toolchain where already consistent with repo needs

### FORGE Control

- keep existing FORGE internal surfaces
- expose operator-facing launchers only for safe tools
- do not grant arbitrary command execution from the UI

## Implementation Tasks

### 1. Add Nix packages to the operator desktop/session profile

Ensure the operator VM/session includes a practical first-pass package set:

- `foot`
- `pcmanfm`
- `firefox`
- `ollama`
- `btop` or `htop`
- `sqlitebrowser` if available
- `jq`
- `yq`
- `ripgrep`
- `fd`
- `bat`
- `eza`
- `tree`
- `git`
- `lazygit`
- `curl`
- `wget`
- `nmap`
- `dnsutils`
- `iproute2`
- `pciutils`
- `usbutils`
- `lsof`
- `strace`
- `micro` or `helix`
- `xarchiver` or equivalent archive manager

Add GPU tools conditionally or best-effort:

- `nvtop`
- NVIDIA tools only when available
- AMD/Intel tools only when applicable

Do not break evaluation if a package name is unavailable. Use package guards or choose stable nixpkgs package names.

### 2. Expand the operator app allowlist

Update the operator app allowlist so the Start menu can launch safe GUI/operator tools.

Add categories:

- Workspace
- Internet
- AI Runtime
- System
- Developer
- FORGE

Add launcher entries where GUI tools exist.

Examples:

- Terminal
- Files
- Browser
- System Monitor
- SQLite Browser
- Git UI / lazygit terminal wrapper if safe
- Editor
- Archive Manager

For CLI-only tools, do not launch raw arbitrary commands from UI unless they are wrapped in a fixed terminal command.

### 3. Add fixed terminal wrappers for CLI tools

For CLI tools like `ollama`, `btop`, `lazygit`, logs, and service status, create safe fixed launch wrappers.

Examples:

- `forge-operator-ollama-status`
- `forge-operator-models`
- `forge-operator-btop`
- `forge-operator-lazygit`
- `forge-operator-core-logs`

These wrappers should run fixed commands only.

No arbitrary command textbox.

### 4. Make Ollama Nix-native

Do not rely on `curl https://ollama.com/install.sh | sh`.

Add Ollama through Nix/profile/module.

If adding service support, default it safely:

- service disabled unless explicitly enabled, or
- enabled only in operator VM profile if that is the intended local runtime

Make sure `ollama` is on PATH in the operator terminal.

### 5. Keep the security model

The Start menu is an allowlist, not a shell injection surface.

Rules:

- no arbitrary command execution from launcher UI
- no arbitrary path execution
- no user-provided launch args
- no host mutation tools enabled by default
- no system rebuild button unless it generates a reviewed proposal
- service restart actions must remain explicit and guarded

### 6. Add tests/checks

Update or add tests proving:

- operator app list includes the new categories
- no launcher accepts arbitrary command input
- CLI wrappers use fixed commands
- Ollama is available in the operator profile
- existing terminal/files/browser entries still work
- desktop Nix check validates required wrappers/packages
- missing optional GPU tools do not fail the whole build

### 7. Update docs

Create or update:

- `docs/operations/operator_desktop.md`
- `docs/operations/operator_toolbelt.md`

Document:

- included operator tools
- what each category is for
- what is intentionally forbidden
- how Ollama is installed correctly
- why `curl | sh` is not used in NixOS
- how to enable/disable optional AI runtime services

## What Not To Do

- Do not use `curl | sh` installers.
- Do not add arbitrary command execution to the UI.
- Do not expose a freeform launcher.
- Do not enable dangerous host mutation tools by default.
- Do not make Nix evaluation depend on GPU-specific packages that may not exist.
- Do not hardcode fragile absolute paths unless wrapped by Nix.
- Do not remove the existing safe terminal/files/browser launchers.
- Do not make the operator desktop require internet just to boot.
- Do not make Ollama install outside Nix.
- Do not mark optional tools as required if they are platform-specific.

## Definition of Done

- Operator profile includes a real toolbelt.
- Ollama is available through Nix, not manual installer.
- Start menu has useful categorized operator apps.
- CLI tools are launched only through fixed safe wrappers.
- Tests/checks prove the allowlist stays safe.
- Docs explain the operator toolbelt and safety boundaries.
