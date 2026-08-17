# OptiPlex Full Development Workstation

Status date: 2026-08-16

## Outcome

The physical `forge-optiplex-7000` NixOS target is a reproducible offline FORGE
development workstation rather than a minimal operator appliance. The profile
includes the desktop applications, IDE extensions, language toolchains,
device services, rootless container tooling, backup clients and diagnostics
needed to build, test, inspect and operate Project Forge locally.

The shared FORGE workspace uses setgid group-write permissions so the local
`operator` can develop under `/forge/workspaces/default` while `forge-core`
retains bounded access through the existing `forge` group. All other FORGE
state remains private under the original `0750` storage contract.

## Boundaries

- FORGE remains the graphical desktop canvas and taskbar; workstation apps are
  native Labwc-managed windows discovered through bounded desktop entries.
- The system remains offline. Nftables permits only loopback and the direct
  `192.168.50.0/24` commissioning link.
- No package manager, extension marketplace, container daemon, backup job,
  printer job or model download starts automatically.
- Four GiB of RAM is supported through zram plus disk swap, with the explicit
  operational expectation that heavyweight GUI applications run one at a
  time alongside the single loaded local model.
- FORGE-K remains the sole semantic and model-visibility authority; developer
  applications do not gain canonical-state authority.

## Validation contract

- Nix evaluation must resolve every package and NixOS service option.
- The OptiPlex static check pins the workstation package categories, shared
  workspace mode, desktop support services and offline firewall.
- OS-integration validation requires the workstation declaration and runbook.
- The full NixOS toplevel build proves the offline deployment closure can be
  realized before transfer to the target.

## Validation evidence

The completed profile passed:

- OptiPlex configuration evaluation;
- the OptiPlex, operator-session, operator-toolbelt, operator-desktop and
  workspace-default Nix checks;
- a full `forge-optiplex-7000` NixOS toplevel build;
- inspection of the realized system path for the office, IDE, language,
  database, container, backup, print and scan executables and representative
  desktop entries;
- repository OS-integration tests and static validation.

The realized Nix closure is approximately 16 GiB. The runbook therefore calls
for at least 40 GiB free for generations, build products, source, data and
models, with a 64 GiB root device treated as the practical minimum.

Physical boot, audio, removable-media, printer/scanner and native-window smoke
remain target-side acceptance steps. They cannot be claimed from the build
host while the direct Ethernet link has no carrier.
