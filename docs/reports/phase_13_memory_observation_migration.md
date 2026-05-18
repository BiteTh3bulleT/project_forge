# FORGE-K Online Phase 13 Memory Observation Migration Report

## Phase

FORGE-K Online Phase 13 - Memory Observation Migration.

## Summary

Phase 13 closes the legacy memory observation write surface as retired with explicit Courthouse/Kernel migration guidance.

Status: `MEMORY_OBSERVATION_WRITES_RETIRED / HISTORY_PRESERVED / COURTHOUSE_REVIEW_GUIDANCE / CONTROL_LANE_CANONICAL_COMMIT / NO_FORGE_K_AUTHORITY_MIGRATION`.

The existing `POST/PATCH /api/memory/observations*` retirement gate now returns a structured JSON response and writes audit metadata that identifies the review path:

- preserve existing `memory_observations` rows as historical evidence
- validate new observation-derived evidence through `VALIDATE_ADMISSION_CANDIDATE`
- commit accepted canonical memory only through existing Control Lane semantic syscalls

This phase does not add a batch migrator, import FORGE-K simulator services, admit evidence, or write canonical memory from the legacy endpoints.

## Files changed

- `services/core/internal/api/server_legacy.go` - adds structured retirement response and audit metadata for memory observation migration guidance.
- `services/core/internal/api/server_memory_legacy_test.go` - proves retired endpoints do not write `memory_observations`, preserve history posture, and include Courthouse/Control Lane migration guidance in response/audit metadata.
- `services/core/internal/authoritymatrix/matrix.go` - updates the memory-observation write row to mark the endpoints retired and audited.
- `docs/status/phase_13_memory_observation_migration.md` - Phase 13 status marker.
- `docs/reports/phase_13_memory_observation_migration.md` - this report.
- `docs/architecture/memory_observation_migration.md` - architecture boundary note.
- `docs/MEMORY_ARCHITECTURE.md` - records the retired write-surface and canonical replacement path.
- `docs/reviews/current_phase_status.md` - current-status note and table entry.
- `docs/status/current_authority_sources.md` - current authority navigation note.

## Tests run

- `cd services/core && go test ./internal/api -run "LegacyMemory|MemoryObservation|RouteInventory" -count=1` - passed.
- `cd services/core && go test ./internal/authoritymatrix -count=1` - passed.
- `rg -n "services/core/internal/forgek/|forgek/kernel|forgek/court|kernel_syscalls|court_syscalls" services/core/internal/api services/core/internal/authoritymatrix -g "*.go" -g "!*_test.go"` - returned no matches, confirming no simulator Courthouse/Kernel import in live API or authority-matrix production paths.
- `cd services/core && go test ./internal/api -count=1` - passed.
- `npm run docs:routes:check` - passed.
- `git diff --check` - passed with expected Windows line-ending warnings only.
- `npm run lint` - passed.
- `npm test` - passed.
- `npm run validate:forgek` - passed.
- `npm run build:core` - passed.

## Authority impact

No new canonical authority migration. The live API remains the retirement gate for legacy observation mutations, and the live Control Lane remains the canonical replacement commit boundary.

The structured response and audit payload are guidance/traceability only. They do not call `VALIDATE_ADMISSION_CANDIDATE`, do not admit evidence, do not call `CREATE_NOTE`, do not write memory, and do not make `services/core/internal/forgek` live authority.

## Security impact

Positive authority clarification. Retired legacy memory mutation attempts are now machine-readable and audited with the intended review/commit path, while tests prove the attempted write does not alter `memory_observations`.

## NixOS impact

No NixOS files, host services, WSL state, or Nix store behavior changed in this phase.

## Rollback path

Revert the Phase 13 commit. The write surface remains retired; rollback only removes the structured migration guidance and updated docs/tests.

## Remaining blockers

- FORGE-K Courthouse and Kernel simulator services are not live authority.
- `VALIDATE_ADMISSION_CANDIDATE` remains validation-only and does not admit evidence.
- No automatic observation-to-canonical-memory batch migrator exists in this phase.
- Phase 00 prompt-vault `PACK_TREE.md` manifest mismatch remains unresolved.
