# Storage Bootstrap Status

_Observed 2026-04-21._

## Storage used

| Store | Backend | Path | Created by |
|---|---|---|---|
| Main operational DB | SQLite (WAL) via `modernc.org/sqlite` (pure Go, no cgo) | `${FORGE_DATA_DIR}/forge.sqlite` | [store.Open()](services/core/internal/store/store.go) on first boot |
| Backups | filesystem | `${FORGE_DATA_DIR}/backups/` | auto-created |
| Exports | filesystem | `${FORGE_DATA_DIR}/exports/` | auto-created |
| Workspace files | filesystem | `${FORGE_WORKSPACE_DIR}` | operator-provided |

No external DB (Postgres, Redis, etc.) required. No separate vector DB
— semantic/VSA indexes are SQLite-backed.

## Migrations

- Embedded SQL migrations run automatically inside `store.Open()`.
- WAL mode and `PRAGMA foreign_keys = ON` are applied at every
  connection.
- `SetMaxOpenConns(1)` — single-writer posture.
- No manual migration tool required for bring-up.

## How to initialize local storage

**Nothing to do.** First `go run .` or `npm run core` boots against an
empty `FORGE_DATA_DIR` and creates everything it needs.

To reset local state:

```sh
rm -rf "${FORGE_DATA_DIR:-$HOME/.config/forge}"
```

(Only do this on a dev box. Back up first if you have work there.)

## What boot does automatically

1. `os.UserConfigDir()` → `${FORGE_DATA_DIR}` default, with fallback to CWD.
2. `store.Open(dataDir)`:
   - Creates the directory if missing.
   - Opens/creates `forge.sqlite`.
   - Runs embedded migrations in one transaction.
   - Applies WAL pragmas.
3. Sub-service constructors create their own subdirs
   (`backups/`, `exports/`) as needed.

## What must be done manually

Nothing for bring-up. For production/operator tasks:

- Set `FORGE_WORKSPACE_DIR` to the project workspace you want FORGE to
  operate over (defaults to `/`, which is broad — in real use, point
  at a specific directory).
- Configure optional adapters (Telegram token, Discord token, Ollama
  endpoint) via the Settings API or desktop UI.

## Observed on 2026-04-21 clean boot

```
$ ls -la /tmp/forge-bringup/data/
drwxr-xr-x  backups/
drwxr-xr-x  exports/
-rw-r--r--  forge.sqlite      (4 KiB)
-rw-r--r--  forge.sqlite-shm  (32 KiB)
-rw-r--r--  forge.sqlite-wal  (3.8 MiB — expected; WAL grows during migrations and compacts later)
```

## Known gaps (not bring-up blockers)

From [implementation_matrix.md](implementation_matrix.md) and
[backup_export_coverage.md](backup_export_coverage.md):

- Backup **export** coverage is broader than **import** restore
  coverage for several AI-OS tables (autonomy repos, approvals events,
  artifacts). Restore-from-backup is not full-fidelity. Fine for
  bring-up; relevant for disaster recovery and release planning.
- Projection repair/rebuild utilities are scaffold-only; not needed
  for fresh-boot but flagged for production readiness.
