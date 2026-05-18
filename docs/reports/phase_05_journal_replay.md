# FORGE-K Online Phase 05 Journal And Replay Report

## Phase

FORGE-K Online Phase 05 - Journal and Replay.

## Summary

Phase 05 adds a read-only journal replay/hash-chain foundation under the existing Control Lane owner. The new `BuildJournalReplayReport` function deterministically sorts `domain.JournalEvent` records, hashes event payloads, links event hashes through `previousHash`, and reports head-hash mismatches without replaying or mutating live state.

Status: `READ_ONLY_REPLAY_FOUNDATION / HASH_CHAIN_ONLY / NO_AUTHORITY_EXPANSION`.

This phase does not change the `journal_events` schema, add a replay command, reconstruct canonical state, change mutation authority, change journal append behavior, alter API routes, or make FORGE-K simulator services live authority.

## Files changed

- `services/core/internal/aios/controllane/journal_replay.go` - adds deterministic replay records, hash-chain report generation, and mismatch diagnostics.
- `services/core/internal/aios/controllane/journal_replay_test.go` - validates deterministic ordering, hash-chain linkage, head mismatch diagnostics, and payload-sensitive hashes.
- `docs/reports/phase_05_journal_replay.md` - this report.
- `docs/status/phase_05_journal_replay.md` - Phase 05 status marker.
- `docs/reviews/current_phase_status.md` - current-status note and table entry.

## Tests run

- `cd services/core && go test ./internal/aios/controllane -run "JournalReplay" -count=1` - passed.
- `cd services/core && go test ./internal/aios/controllane -count=1` - passed.
- `npm run docs:routes:check` - passed.
- `git diff --check` - passed with expected Windows line-ending warnings.
- `npm test` - passed.
- `npm run lint` - passed.
- `npm run validate:forgek` - passed.
- `npm run build:core` - passed.

## Tests not run

- Desktop validation was not run because Phase 05 added no desktop/UI code.
- Nix checks were not run because Phase 05 added no Nix files or host-substrate behavior.

## Authority impact

No authority expansion. The live owner remains `services/core/internal/aios/controllane`. Existing append-only journal behavior, semantic mutation validation, commit boundaries, audit linkage, and storage remain unchanged.

The replay report is diagnostic-only. It does not execute a replay, write state, mutate memory, call modelruntime, execute gateway tools, run retrieval/search/embeddings, or import FORGE-K simulator packages.

## Security impact

Positive diagnostic foundation. The replay records hash payloads instead of exposing raw payloads in mismatch comparisons, and the chain can detect ordering/content drift when an expected head hash is supplied.

## NixOS impact

No NixOS files, host services, WSL state, or Nix store behavior changed in this phase.

## Rollback path

Revert the Phase 05 commit. Existing journal append/read behavior can continue without the read-only replay report helper.

## Remaining blockers

- No persisted hash-chain columns were added.
- No replay dry-run command was added.
- No canonical state reconstruction verifier was added.
- Gateway, approval, modelruntime proposal, and full audit linkage replay remain future work.
- Phase 00 prompt-vault `PACK_TREE.md` manifest mismatch remains unresolved.
