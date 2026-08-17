# OptiPlex Full Development Workstation

Status date: 2026-08-17

## Outcome

The physical `forge-optiplex-7000` NixOS target is a reproducible offline FORGE
development workstation rather than a minimal operator appliance. The profile
includes the desktop applications, IDE extensions, language toolchains,
device services, rootless container tooling, backup clients and diagnostics
needed to build, test, inspect and operate Project Forge locally.

FORGE Settings now exposes native Network Connections, Displays, Audio,
Printers, Bluetooth, and Appearance controls. The active local operator has a
bounded polkit grant for logind reboot/poweroff and NetworkManager profile
management, while sudo remains password-gated and the offline nftables policy
remains authoritative. Power requests synchronously verify that systemd
accepted the nonblocking request instead of reporting success after merely
spawning a child process.

The 2026-08-17 full-authority test-mode supersession disables the OptiPlex
safe-mode posture and enables the shell's host-power, model-mutation, and
semantic-write controls plus model autoload and the loopback OpenAI-compatible
API. This is intentionally limited to the isolated physical test profile. It
does not grant the desktop shell, model, or runtime driver authority to bypass
production FORGE-K, and nftables continues to deny off-box egress.

The shared FORGE workspace uses setgid group-write permissions so the local
`operator` can develop under `/forge/workspaces/default` while `forge-core`
retains bounded access through the existing `forge` group. All other FORGE
state remains private under the original `0750` storage contract.

The local model lane now uses `qwen2.5:1.5b` as the default tool router and
keeps `gemma3:1b-it-q4_K_M` as the bounded completion fallback. The former
Smuxo model was retired from both the staging workstation and the target after
it timed out through FORGE and returned unrelated text in a direct probe.
Qwen was promoted only after target-side probes demonstrated a correct weather
tool choice, an exact `/forge/workspaces/default/README.md` file argument, and
a no-tool response when no tool was needed.

## Boundaries

- FORGE remains the graphical desktop canvas and taskbar; workstation apps are
  native Labwc-managed windows discovered through bounded desktop entries.
- The system remains offline. Nftables permits only loopback and the direct
  `192.168.50.0/24` commissioning link.
- No package manager, extension marketplace, container daemon, backup job, or
  printer job starts automatically. Governed local model autoload is enabled,
  but the offline profile cannot contact a remote model registry.
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

The final target-side authority smoke confirmed the active FORGE-K owner,
governed Qwen model visibility, maintain-mode autonomy, active VSA policy,
deep Dream analysis policy, the direct-link-only route table, and denied public
egress. Audio, removable-media, printer/scanner and the full native-window
interaction checklist remain hands-on acceptance steps because they require a
person at the physical display and devices.
