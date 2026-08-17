# FORGE-K offline recovery status

Status: `IMPLEMENTED / LINUX_NIXOS_ONLY / DAEMON_STOPPED / WHOLE_STORE`

The production recovery implementation is
`services/core/internal/offlinerecovery`, exposed only by the standalone
`services/core/cmd/forge-recover` command and optional Nix app command
(`nix run .#forge-recover`). `store.Open` and recovery share a
non-blocking Linux process lock. Unsupported platforms fail closed for recovery
without changing ordinary cross-platform store-open behavior.

Implemented evidence:

- strict `full_backup` manifest, required-section, count, checksum, and exact
  byte-digest validation;
- current-schema database construction followed by exact whole-section import;
- SQLite integrity and foreign-key verification;
- FORGE-K journal event hash-chain and independently persisted head validation;
- typed request/plan/seal/receipt/authorization/idempotency verification,
  including rehydration of persisted Kernel-owned typed metadata contracts;
- audit-outbox request/receipt/journal identity validation, append-only delivery
  attempt-to-outbox fingerprint validation, and exact audit projection IDs;
- Court exhibit/current-ruling/history/appeal scope and journal/audit identity
  validation;
- same-filesystem atomic swap, durable prior-store copy, owner/group/mode
  preservation, post-swap re-verification, and automatic rollback;
- restart, daemon-lock refusal, checksum tamper, chain tamper, exact proof ID,
  and injected post-swap rollback tests.

The live backup API still cannot apply restores. Generic exact-row import is
private to the offline package and has no API, service, gateway, or daemon
callsite.

Remaining limits:

- recovery requires a current-schema `full_backup`; it is not a schema
  downgrade or selective migration facility;
- intentionally excluded secret material is not recreated from the bundle;
- the advisory stopped-daemon guarantee becomes enforceable for processes
  using the new shared lock; operators must still stop and verify the NixOS
  service as documented;
- external filesystem/hardware failure after the atomic rename is outside the
  SQLite proof boundary, so the preserved prior copy must remain until restart
  and operator smoke checks pass.

Operator procedure: `docs/runbooks/forge_k_offline_recovery.md`.
