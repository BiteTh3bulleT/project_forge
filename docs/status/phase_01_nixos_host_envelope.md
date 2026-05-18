# FORGE-K Online Phase 01 NixOS Host Envelope Status

## Phase

FORGE-K Online Phase 01 - NixOS Host Envelope.

## Status marker

`PARTIAL / OPT_IN_ONLY / NO_HOST_MUTATION / NO_LIVE_AUTHORITY_CHANGE`

## Summary

The NixOS `forge-core` service envelope is hardened around localhost-first binding, explicit wildcard-bind opt-in, and systemd sandboxing. This is a host-envelope improvement only; it does not move live daemon authority into FORGE-K.

## Live owner

Live runtime authority remains unchanged:

- API routes: `services/core/internal/api`
- semantic mutation: `services/core/internal/aios/controllane`
- tool execution: `services/core/internal/gateway`
- model runtime: `services/core/internal/modelruntime`
- memory/retrieval: `services/core/internal/memory`, `services/core/internal/retrieval`
- audit/approval: `services/core/internal/audit`, `services/core/internal/approvals`
- host configuration: operator-owned NixOS configuration and runbooks

## Target FORGE-K owner

NixOS is the target host envelope for future FORGE-K online work. It remains outside FORGE-K truth authority and does not become a semantic mutation path.

## Authority impact

No host mutation authority, no `nixos-rebuild` execution path, no `systemctl` control path, no autologin change, no modelruntime mutation, no semantic memory write, and no FORGE-K live authority.

## Tests/evidence

Final validation commands are recorded in `docs/reports/phase_01_nixos_host_envelope.md`.

## Rollback

Revert the Phase 01 commit. No live data, host state, database schema, or daemon runtime migration is involved.

## Blockers

- Nix checks require a Nix-enabled host for authoritative evaluation evidence.
- Tool execution sandbox profiles and VM smoke automation remain future work.

## Next phase

Run exactly one next phase prompt after operator selection. Do not chain the authority gate matrix phase into this commit.
