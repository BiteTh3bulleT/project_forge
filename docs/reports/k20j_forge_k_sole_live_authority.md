# K20J FORGE-K Sole Live Authority

Status date: 2026-08-16

## Outcome

Production boot has one authority: `forge_k.kernel`. Alternate live authority modes fail closed. Control Lane remains a bounded deterministic validation/apply/SQLite port; it is not independently boot-selectable and does not own stage order, authorization, decisions, commit success, or replay.

The live Context Compiler is `services/core/internal/forgekernel/contextcompile`. It uses exact workspace/lane/selected-path scope, current Courthouse-admitted immutable memory evidence, governed prior bundle candidates and CAS head, a sealed integer-only policy, and complete output commitments. The Kernel computes and plan-binds the decision. The durable port re-reads authority inputs inside the transaction and atomically stores the immutable bundle/head, provenance, journal chain transition, authorization proof, audit-outbox intent, and idempotency replay proof.

Every ordinary chat, native Ollama, modelruntime, gateway-synthesis, direct FORGE model-chat, and OpenAI-compatible prompt passes through that compiler. Every corresponding model output remains hidden until the pure runtime-proposal decision and consensus gate accept it. Raw chat history and legacy memory observations/snapshots are not authoritative prompt inputs.

Tokenless HTTP is trusted only from a verified loopback peer; remote requests fail closed. Live backup merge remains disabled. Disabled or unimplemented features—including backend KV tensor reuse and semantic operators beyond the implemented deterministic diff—have no alternate authority path.

## Evidence

- Pure Context Compiler golden, permutation, bounds, tamper, and fuzz tests.
- Production SQLite integration over admitted exact-scope evidence, with cross-scope and legacy-input exclusion.
- Atomic bundle/head/journal/outbox/idempotency rollback on injected journal collision.
- Verified replay adds no bundle and does not advance the scope head.
- Immutable bundle triggers and exact writer/callsite guards.
- Model-prompt and model-visibility static guards cover every live surface.
- Full backup exports context bundle/head proof as offline-recovery-only sections.

## Bounded non-authority surfaces

- The external audit sink and legacy `audit_id` backfill are best-effort projections over canonical immutable audit-outbox evidence.
- SQLite is the live store; Redis and Qdrant are non-authoritative coordination/acceleration.
- Backend KV reuse is disabled; validation eligibility is not reuse authority.
- Lymphatic maintenance emits proposals only and cannot mutate canonical state.
- Whole-store recovery is daemon-stopped; live row-merge restore is disabled.

## Final validation

- `npm test`
- `npm run lint`
- `npm run validate:js`
- `npm run test:repo-hygiene && npm run validate:repo-hygiene`
- `npm run test:os-integration && npm run validate:os-integration` (73 static checks)
- `npm run smoke` (includes the authenticated live `/forge/kernel/status` sole-authority assertion)
- `npm run validate:local` (integration-env preflight, hygiene, Obsidian bridge, OS integration, desktop test/build, Go/Rust FORGE-K parity, and core build)
- `go test -race -count=1 ./internal/forgekernel/... ./internal/aios/controllane ./internal/api ./internal/backup ./internal/store`
- `nix eval .#nixosConfigurations.forge-optiplex-7000.config.system.build.toplevel.drvPath`
- `nix build .#checks.x86_64-linux.forge-optiplex-7000 --no-link`
- `git diff --check`

All repository, desktop, Go, race, Nix evaluation, and offline-policy static gates passed. Physical OptiPlex boot/network acceptance remains a deployment evidence step and is not represented as complete by this repository cutover.
