# ADR 0011 - FORGE Graphical Shell Session

Status: Accepted

Date: 2026-05-08

## Context

FORGE is evolving into the visible operating interface for a NixOS-based FORGE-OS machine. The intended role is not a web dashboard controlling a headless server. The intended role is a graphical shell session: desktop surface, launcher, workspace surface, command center, approval surface, and system context surface.

NixOS remains the boot, hardware, display/session, service, and declarative host configuration substrate. Existing FORGE authority boundaries remain in force: gateway, permissions, lanes, approvals, audit, controllane validation, semantic memory authority, modelruntime governance, FORGE-H bounded host diagnostics/proposals/executions, and FORGE-K simulator/live separation.

## Decision

FORGE will be documented as the graphical shell for a FORGE-OS session above the NixOS substrate.

Phase G1 defines only the shell session foundation and contract. It may add inert, opt-in NixOS module scaffolding for session metadata, environment variables, and runtime-directory preparation. The only G1 session mode is `fullscreen-shell`. Future modes such as `kiosk`, `compositor-integrated`, `remote-operator`, and `multi-monitor-shell` require later explicit implementation and tests.

The shell must talk to `forge-core` through governed local APIs/interfaces. It may display structured system/session context and submit operator requests. It must not directly mutate host state or canonical FORGE state.

FORGE as graphical shell provides full operating awareness, not full LLM context. The context compiler decides what subset of structured context reaches model calls. Raw full system state must not be dumped into prompts.

## Consequences

- NixOS remains the substrate; FORGE does not become a Linux kernel, display server, or package manager.
- G1 must remain opt-in and reversible.
- G1 may expose a NixOS module, but it must remain disabled by default and must not autostart the shell.
- Autologin must not be enabled by default.
- Existing desktop environments must not be disabled by default.
- Safe fallback through a normal desktop or TTY must remain available.
- The shell cannot bypass gateway, permissions, approvals, audit, controllane, semantic memory, modelruntime, or FORGE-K authority boundaries.
- G1 introduces no host mutation, service restarts, NixOS rebuilds, kernel/module changes, modelruntime mutation, semantic memory writes, route/API changes, or FORGE-K live authority changes.

## Alternatives Considered

- Treat FORGE as a web dashboard: rejected because the target is an operator-facing graphical shell session, not a remote control page for a headless server.
- Let the shell directly control NixOS services: rejected because host mutation must remain governed and cannot bypass existing FORGE authority boundaries.
- Make FORGE-K the live shell authority in G1: rejected because FORGE-K remains simulator/shadow authority unless a later live integration phase changes that boundary with design, tests, and documentation.
- Replace the user's desktop by default: rejected because G1 must be incremental, opt-in, and rollback-safe.
