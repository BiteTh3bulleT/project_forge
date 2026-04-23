{ lib }:

{
  # Systems FORGE targets for dev shells and packages.
  # Desktop/Tauri packaging is Linux-first; macOS shells are supported
  # but full desktop packaging on Darwin is not validated in Phase N1.
  defaultSystems = [
    "x86_64-linux"
    "aarch64-linux"
    "x86_64-darwin"
    "aarch64-darwin"
  ];

  linuxSystems = [
    "x86_64-linux"
    "aarch64-linux"
  ];
}
