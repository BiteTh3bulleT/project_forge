# Contributing to FORGE

FORGE is an engineering-alpha local AI workstation with explicit authority, execution, evidence, and recovery boundaries. Contributions are welcome, but changes that appear small can affect durable state, operator trust, or host safety.

Start with:

1. `README.md`
2. `docs/status/CURRENT_STATE.md`
3. `docs/onboarding.md`
4. `AGENTS.md`
5. the nearest subsystem tests and authority documentation

## Core invariants

Contributions must preserve these rules unless an explicitly reviewed architecture change replaces them:

- Models propose; they do not grant themselves authority.
- Gateway is the only tool-execution authority.
- Production semantic syscalls enter through `services/core/internal/forgekernel`.
- `services/core/internal/forgek` is simulator and target-architecture work, not production authority.
- Unsupported or unimplemented authority paths fail closed.
- Tool completion claims require actual execution evidence.
- Canonical writes require deterministic validation, authorization, provenance, journal/audit evidence, and replay-safe behavior.
- Live raw backup row-merge restore remains disabled until an approved recovery design replaces it.
- Linux/NixOS host mutation must remain explicit, bounded, reviewable, and recoverable.

## Development workflow

Create a focused branch from current `main`:

```bash
git switch main
git pull --ff-only
git switch -c <type>/<short-description>
```

Use a descriptive prefix such as `fix/`, `feat/`, `docs/`, `test/`, `refactor/`, or `chore/`.

Keep pull requests narrow enough to review. A large program should be divided into independently verifiable phases rather than one opaque rewrite.

Do not push feature, authority, recovery, security, or documentation-overhaul work directly to `main`.

## Pull-request expectations

Every pull request should explain:

- the problem being solved;
- the chosen approach;
- affected authority and persistence boundaries;
- operator-visible behavior changes;
- tests and validation performed;
- skipped checks and why;
- rollback, recovery, or compatibility considerations;
- documentation updated with the change.

Authority-sensitive changes should identify the current owner before and after the change. “No authority change” is a valid answer, but it must be intentional.

## Validation

Run the narrowest tests while developing and the broader relevant gates before requesting review.

Common commands:

```bash
npm test
npm run lint
npm run validate:js
npm run validate:desktop
npm run validate:forgek
npm run build:core
npm run build:desktop
npm run validate:local
npm run smoke
```

Use targeted Go tests during development:

```bash
cd services/core
go test ./internal/<package>
go test -race ./internal/<concurrency-sensitive-package>
```

For Nix/NixOS changes, run the relevant evaluation and check targets. Record physical-machine checks separately from static build evidence.

Do not turn a flaky failure into a blind retry. Find and fix the synchronization, lifecycle, state-isolation, or environment defect.

## Tests

Add tests for both successful behavior and fail-closed behavior.

Important categories include:

- authorization and capability scope;
- idempotency and replay;
- path traversal and workspace isolation;
- malformed or oversized input;
- cancellation and timeout;
- concurrent update and shutdown behavior;
- unsupported authority claims;
- model hallucination or false completion claims;
- recovery after partial failure.

Prefer deterministic fixtures. Never place real credentials, private data, bearer tokens, API keys, or sensitive audit payloads in tests.

## Documentation

Documentation is part of the implementation.

Update the appropriate current source in the same pull request when behavior changes:

- `docs/status/CURRENT_STATE.md` — concise current posture and blockers.
- `docs/status/current_authority_sources.md` — detailed live authority ownership.
- `docs/reviews/current_phase_status.md` — cumulative implementation evidence and phase history.
- `docs/runbooks/` — executable operator procedures.
- `docs/architecture/` — design and boundaries.
- `docs/api/routes.md` — generated API inventory.

Label future or non-production material clearly with terms such as `PLANNED`, `DESIGN_ONLY`, `SIMULATOR_ONLY`, `SHADOW_ONLY`, `DISABLED_BY_DEFAULT`, or `NO_LIVE_AUTHORITY_CHANGE`.

Avoid duplicating the same “current truth” across many documents. Link to the canonical current source instead.

## Commit guidance

Use imperative, scoped commit messages, for example:

```text
fix: join approval decision goroutines before cleanup
docs: clarify production kernel and simulator boundary
test: add small-model tool continuation regression fixture
```

Do not mix generated artifacts, formatting churn, unrelated refactors, and behavior changes in one commit unless they are inseparable.

## Security-sensitive changes

Changes involving authentication, remote ingress, Gateway, approvals, audit, secrets, host control, backup/recovery, model visibility, or durable authority require extra care.

- Follow `SECURITY.md`.
- Avoid logging secrets or raw sensitive prompts/tool arguments.
- Use constant-time credential comparison where applicable.
- Preserve authenticated actor and provenance identities through asynchronous work.
- Add explicit bounds to request bodies, strings, collections, retries, and timeouts.
- Document recovery and failure behavior.

## Review and merge

A pull request is ready to merge when:

- the diff is focused and understandable;
- required validation is green;
- authority and persistence effects are documented;
- operator documentation matches behavior;
- recovery or rollback is adequate for the risk;
- no unresolved review concerns remain.

The maintainer may request that a broad change be split into smaller governed phases.
