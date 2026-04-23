# Smoke Test Status

_Added 2026-04-21 as part of the bring-up pass. Updated 2026-04-22 for VSA authority lane consistency._

## What the smoke test does

[`scripts/forge-smoke.sh`](scripts/forge-smoke.sh) — invocable as
`npm run smoke` — performs a black-box bring-up check:

1. Asserts port `18492` is free.
2. Creates ephemeral `$DATA_DIR` and `$WORKSPACE_DIR` under `/tmp`.
3. Runs `scripts/check-vsa-files.sh --require-tracked`, then starts `go run .` from `services/core` with those env vars.
4. Polls `/health` for up to 30 s.
5. Probes these endpoints and asserts HTTP 200:
   - `/health`
   - `/api/meta`
   - `/api/autonomy/status`
   - `/api/telegram/status`
   - `/api/discord/status`
   - `/api/adapters`
   - `/api/jobs`
6. Kills the core process, cleans up the temp dirs.
7. On failure, tails the core log to stderr.

## Why these endpoints

They exercise the core bring-up path without mutating durable state
in the operator's main data dir:

- `/health` — basic HTTP liveness.
- `/api/meta` — config resolution (data/workspace/DB paths).
- `/api/autonomy/status` — AI-OS runner initialization and default
  charter/budget seeding.
- `/api/telegram/status`, `/api/discord/status` — optional gateways
  report honest disabled state (not 5xx when unconfigured).
- `/api/adapters` — gateway + adapter registry booted.
- `/api/jobs` — job service + dependencies booted.

This covers gateway, audit, artifact, approvals, events, and memory
init implicitly (they're all in the dependency graph of `jobs` and
`autonomy`).

## Current status: **GREEN** on 2026-04-21

```
$ npm run smoke
==> port 18492 must be free
==> starting forge-core (data=/tmp/forge-smoke.XXXXXX)
==> waiting for /health (up to 30s)
ok    /health after 2 attempts
==> probing endpoints
ok    /health -> 200
ok    /api/meta -> 200
ok    /api/autonomy/status -> 200
ok    /api/telegram/status -> 200
ok    /api/discord/status -> 200
ok    /api/adapters -> 200
ok    /api/jobs -> 200
==> shutting down
==> smoke OK
```

`scripts/forge-smoke.sh` still calls `scripts/check-vsa-files.sh --require-tracked` as a guardrail.
VSA status for smoke lane is authoritative source (not generated, not optional).
With authoritative tracked VSA files in this branch state, smoke passes.

## What smoke does NOT cover

- **Mutating flows** (syscall commits, approval decisions,
  artifact creation, backup/restore). Those are covered by the Go test
  suite under `services/core`.
- **Desktop runtime.** Tauri window bring-up requires an interactive
  GUI environment and WebKit/GTK; out of scope for automated smoke.
- **Nix flake builds** (see [nix_foundation_status.md](nix_foundation_status.md))
  — a separate `nix flake check` covers the flake surface.
- **Dangerous capability gating.** Tool policy is enforced by
  `go test ./internal/gateway/...`, not by this smoke script.

## Safety of running smoke

- Uses ephemeral `/tmp/forge-smoke.*` dirs per run; never touches
  `~/.config/forge` or any operator data.
- Kills only its own PID plus anything left listening on port 18492
  via `lsof -ti tcp:18492`.
- Trap cleanup runs on any exit path.
- Autonomy runs in the default `maintain` mode during the ~3 s of
  boot the smoke covers. No external-call budget, no intents in flight
  — verified via `/api/autonomy/status`.
