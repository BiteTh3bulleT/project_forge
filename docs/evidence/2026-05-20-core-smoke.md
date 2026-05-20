# 2026-05-20 Core Smoke Evidence

Command:

```sh
npm run smoke
```

Result: passed.

Observed probes:

- `/health` -> `200`
- `/api/meta` -> `200`
- `/api/autonomy/status` -> `200`
- `/api/telegram/status` -> `200`
- `/api/discord/status` -> `200`
- `/api/adapters` -> `200`
- `/api/jobs` -> `200`

The smoke harness started `forge-core` against an ephemeral data directory,
waited for `/health`, probed the endpoints above, and shut the process down.

## Follow-up A5 Verification

2026-05-20 follow-up commands:

- `npm run smoke` -> passed with the same seven endpoint probes.
- `cd services/core && go test -race ./internal/api` -> first uncaptured run
  exited non-zero after heavy log output; captured rerun passed in 128.658s.
- `cd services/core && go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
  -> initially found reachable `GO-2025-3770` in `github.com/go-chi/chi/v5`
  v5.2.1. After upgrading chi to v5.2.2, rerun reported:
  `Your code is affected by 0 vulnerabilities.`
