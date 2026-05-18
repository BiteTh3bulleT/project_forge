# FORGE-K Online Phase 01 NixOS Host Envelope Report

## Phase

FORGE-K Online Phase 01 - NixOS Host Envelope.

## Summary

Phase 01 hardens the opt-in NixOS `forge-core` service boundary while preserving the existing live authority map. The service remains localhost-only by default, wildcard binding now requires an explicit NixOS option, and the unit has a stricter systemd sandbox.

Status: `PARTIAL / OPT_IN_ONLY / NO_HOST_MUTATION / NO_LIVE_AUTHORITY_CHANGE`.

This is a narrow hardening/reconciliation pass over the already-present NixOS substrate from earlier N/G/operator VM work. It does not claim the entire host envelope is newly implemented in this phase.

## Files changed

- `nix/nixos/modules/forge-services.nix` - added `allowWildcardBind`, exported `FORGE_ALLOW_WILDCARD_BIND`, asserted all-interface binds require opt-in, and tightened service sandboxing.
- `nix/checks/forge-core-bind-host.nix` - aligned the check with the hardened Dockerfile default and the NixOS wildcard-bind guard.
- `docs/architecture/nix_substrate.md` - updated stale NixOS module status and recorded the Phase 01 host envelope boundary.
- `docs/status/nix_foundation_status.md` - updated current Nix substrate status.
- `docs/reports/phase_01_nixos_host_envelope.md` - this report.
- `docs/status/phase_01_nixos_host_envelope.md` - Phase 01 status marker.
- `docs/reviews/current_phase_status.md` - concise Phase 01 status note and table entry.

## Tests run

- PowerShell Phase 01 text contract check for NixOS service guard/check strings - passed.
- `npm run docs:routes:check` - passed.
- `git diff --check` - passed with expected Windows line-ending warnings.
- `npm test` - passed.
- `npm run lint` - passed.
- `npm run validate:forgek` - passed.
- `npm run build:core` - passed.

## Tests not run

- `nix --version` failed because `nix` is not installed in this Windows shell.
- `nix build .#checks.x86_64-linux.forge-core-bind-host` and `nix flake check` were not run for the same reason; authoritative Nix validation remains blocked until this commit is evaluated on a Nix-enabled host.
- Nix was not installed from this Windows shell because that would mutate the host outside Phase 01's non-mutating host-envelope scope.
- Desktop validation was not run because Phase 01 made no desktop/UI changes.

## Authority impact

No live authority change. NixOS remains an opt-in host envelope and does not become a runtime mutation authority.

The live daemon still owns runtime behavior through the existing API, Control Lane, gateway, modelruntime, memory/retrieval, audit, and approval paths. FORGE-K simulator packages remain non-authoritative.

## Security impact

Positive NixOS envelope hardening:

- `services.forge-core.bindHost` still defaults to `127.0.0.1`.
- `services.forge-core.allowWildcardBind` defaults to `false`.
- `FORGE_ALLOW_WILDCARD_BIND` is now explicit in the NixOS service environment.
- Binding `0.0.0.0` or `::` from the NixOS module fails evaluation unless wildcard binding is explicitly enabled.
- The `forge-core` systemd service adds stricter sandboxing, including no ambient capabilities, private devices, kernel/control-group protections, restricted address families, and SUID/realtime restrictions.

## NixOS impact

Only opt-in NixOS module behavior changed. No current Windows/Docker/npm/dev workflow changed. No host was rebuilt or mutated by this phase.

## Rollback path

Revert the Phase 01 commit. Existing operators using the default localhost bind are unaffected. Operators intentionally binding all interfaces through the NixOS module must now set `services.forge-core.allowWildcardBind = true`; reverting removes that guard.

## Remaining blockers

- Nix evaluation/checks still need to be run on a Nix-enabled host for authoritative Nix evidence.
- Phase 01 does not implement tool sandbox profiles, VM smoke automation, or a governed host mutation adapter.
- The Phase 00 prompt-vault `PACK_TREE.md` manifest mismatch remains unresolved.
