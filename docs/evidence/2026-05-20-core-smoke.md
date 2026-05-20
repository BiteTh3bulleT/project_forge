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
