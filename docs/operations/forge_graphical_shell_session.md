# FORGE Graphical Shell Session

This runbook describes the Phase G1 graphical shell session contract.

G1 defines how an opt-in NixOS session should launch FORGE as the primary visible shell while preserving the existing FORGE authority boundaries. The implementation is scaffolding only: an inert NixOS module can prepare environment/session metadata, but it does not autostart, replace a desktop, add a compositor, or launch a production shell package.

## Operator Meaning

FORGE graphical shell means FORGE is the desktop shell for a FORGE-OS session:

- desktop/workspace surface
- launcher and command palette
- system context surface
- approvals and notifications
- resource and service status
- governed access to `forge-core`

It does not mean a browser dashboard controlling a remote or headless server.

## Expected Future Defaults

The session scaffolding remains inert unless explicitly enabled.

| Option | Expected G1 Default |
|---|---|
| `forge.shellSession.enable` | `false` |
| `forge.shellSession.mode` | `"fullscreen-shell"` |
| `forge.shellSession.displayBackend` | `"wayland"` |
| `forge.shellSession.autoStart` | `false` |
| `forge.shellSession.coreURL` | `"http://127.0.0.1:18492"` |
| `forge.shellSession.safeMode` | `true` |

Autologin must remain disabled by default. Existing desktop environments must remain available by default. G1 only defines `fullscreen-shell`; other modes require later design, implementation, tests, and rollback notes.

## Safe Session Flow

1. Operator boots NixOS normally.
2. Operator selects or starts the opt-in FORGE shell session.
3. Session metadata points at the existing FORGE desktop shell package/binary when packaging is available.
4. Shell receives `FORGE_CORE_URL` pointing at the local governed core endpoint.
5. Shell renders shell-safe status surfaces and requests structured state through `forge-core`.
6. Any requested action goes through the existing gateway, permissions, lane, approval, audit, controllane, and memory/modelruntime governance paths.

If a shell package is unavailable, the G1 placeholder fails visibly and safely. It must not fake production behavior.

## Allowed Operations

The shell may:

- display local service health
- display resource posture from governed diagnostics when available
- display approval queues when supported by existing APIs
- display model/runtime status when supported by existing APIs
- display memory/journal browser views through governed read paths
- submit operator-initiated requests to `forge-core`
- show bounded placeholders for unavailable surfaces

## Forbidden Operations

The shell must not:

- run `systemctl` directly
- run `nixos-rebuild` directly
- load or unload kernel modules
- restart host services
- mutate NixOS configuration
- clean filesystems
- load or unload models directly
- write semantic memory directly
- bypass gateway execution authority
- bypass approvals
- treat model output as canonical truth
- use FORGE-K simulator services as live authority

G1 introduces no host mutation, modelruntime mutation, semantic memory mutation, route/API mutation, gateway mutation, or FORGE-K authority mutation.

## System Context Handling

The shell may observe structured system/session context:

- active workspace
- open panels/windows
- current project
- resource posture
- service health
- model status
- approval state
- recent errors
- user-triggered actions

This is operating awareness for the shell. It is not automatic LLM prompt context.

The context compiler decides what subset reaches model calls. Raw full system state, raw logs, raw process lists, raw window state, raw host diagnostics, and raw memory contents must not be dumped into prompts.

## Fallback

Implementations must preserve a fallback path:

- keep a normal desktop/session available
- keep TTY login available
- keep shell session opt-in
- keep autostart disabled by default
- keep safe mode enabled by default
- keep existing `npm run desktop` and `npm run dev` workflows usable

If the shell session fails, the operator should log into the fallback desktop or TTY and disable the opt-in shell session configuration. Canonical FORGE data under `/forge` must not be deleted as part of shell rollback.

## Validation Expectations

G1 validation must include Nix module evaluation plus documentation review. Future runtime phases must add evidence for:

- opt-in disabled-by-default behavior
- no autologin by default
- no existing desktop removal by default
- session launch in `fullscreen-shell`
- safe failure when the shell binary/package is unavailable
- `FORGE_CORE_URL` environment wiring
- no direct host mutation from shell controls
- no direct modelruntime mutation
- no direct semantic memory writes
- no FORGE-K live authority routing
