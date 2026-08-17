# FORGE-K offline whole-store recovery

Status: implemented and tested for Linux/NixOS. This is the only supported
restore-apply path. The live `/api/backup/restore` route remains inspection-only
and every non-dry request remains disabled.

## Safety boundary

Recovery accepts only a complete `full_backup` JSON bundle staged beneath the
target data directory's `backups/` or `exports/` directory. It validates the
bundle schema, manifest, required section inventory, declared row counts, and
section checksums before creating a new database. It then verifies SQLite
integrity/foreign keys, exact restored section checksums, the FORGE-K journal
chain and head, sealed idempotency and audit receipt identities, audit-delivery
attempt bindings, and Court history/current-ruling identities.

The command takes the same Linux advisory process lock as `forge-core`; it
fails if the upgraded daemon is still running. Recovery builds and verifies a
new SQLite file, checkpoints the stopped current database, copies the current
database to `recovery-prior/`, and atomically renames the new database into
place on the same filesystem. It preserves the current database owner, group,
and mode. A failed post-swap check restores the prior database; the preserved
copy remains available for manual recovery.

## OptiPlex/NixOS procedure

1. Put the verified `full_backup` bundle under `/forge/data/backups/`. Do not
   use a symlink, and do not stage it outside `/forge/data`.

2. Obtain the recovery executable from your normal package workflow:

   ```sh
   # Dev shell path (works on any Linux checkout)
   cd /forge/workspaces/default/ProjectForge/services/core
   go build -o /tmp/forge-recover ./cmd/forge-recover

   # Nix path
   nix run .#forge-recover -- --help
   # optionally pin to output
   # sudo nix build .#forge-recover --out-link /tmp/forge-recover-result
   ```

3. Stop the daemon and confirm it is inactive:

   ```sh
   sudo systemctl stop forge-core.service
   systemctl is-active forge-core.service
   ```

   The expected second-command output is `inactive`. Stop here if it is not.

4. Run recovery. Replace the bundle name with the exact staged file:

   ```sh
   sudo /tmp/forge-recover \
     --data-dir /forge/data \
     --bundle /forge/data/backups/<full-backup-file>.json
   ```

   Success prints one JSON result with `"applied":true`, the inspected bundle
   and plan digests, proof counts, and `priorStoreBackup`. Preserve that output
   with the maintenance record.

5. Start FORGE and verify both service health and the journal chain through
   normal operator diagnostics:

   ```sh
   sudo systemctl start forge-core.service
   systemctl is-active forge-core.service
   curl --fail --silent http://127.0.0.1:18492/api/health
  ```

Or with the packaged app launcher:

```sh
sudo nix run .#forge-recover -- \
  --data-dir /forge/data \
  --bundle /forge/data/backups/<full-backup-file>.json
```

## Failure and rollback

- A validation error occurs before the current store is replaced.
- `offline recovery requires stopped daemon` means `forge-core` or another
  cooperative store user still holds `/forge/data/forge.sqlite.lock`.
- A post-swap verification failure automatically restores the prior database
  and returns `"rolledBack":true` with a non-zero exit status.
- Never copy individual rows from the bundle into the live database. Never
  delete the `recovery-prior/` copy until restart and application-level checks
  pass.
- To manually return to a prior file, stop `forge-core`, preserve the failed
  current file, copy the selected `recovery-prior/*.sqlite` to a temporary file
  on `/forge/data`, preserve its owner/group/mode, and rename it atomically to
  `/forge/data/forge.sqlite`. Do not do this while the daemon is running.

This first recovery slice does not restore secrets excluded from full backups,
does not support row selection or schema downgrades, and does not claim support
outside Linux/NixOS.
