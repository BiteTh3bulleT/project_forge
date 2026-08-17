# FORGE

FORGE is a local-first, governed AI workstation for inspectable engineering work. It combines a desktop shell, a local control plane, model runtimes, memory and retrieval, approval-gated tools, and durable audit evidence.

> **Project status: engineering alpha.** FORGE is running on real hardware and its core authority boundaries are live, but tool grounding, conversational continuation, recovery, packaging, and broad operator acceptance are still being hardened. Read [`docs/status/CURRENT_STATE.md`](docs/status/CURRENT_STATE.md) before relying on a subsystem or extending authority.

## Why FORGE exists

Most AI applications place a model directly in front of tools and hope the prompt is enough. FORGE uses a stricter rule:

> **Models propose. FORGE decides. Gateway executes. Evidence proves.**

The goal is not merely a larger chatbot. The goal is an operating layer that can use small local models, deterministic software, governed tools, memory, and specialized workers to complete useful work without granting the model authority it does not possess.

## Current system

FORGE currently includes:

- **Desktop shell** — Tauri + React operator workspace.
- **Core daemon** — Go service with chat, jobs, approvals, audit, memory, retrieval, automation, modelruntime, and operator APIs.
- **Production Kernel boundary** — `services/core/internal/forgekernel` is the sole boot-selectable semantic syscall ingress and decision owner for the production paths documented as live.
- **Bounded durable port** — Control Lane still implements temporary validation, apply, and SQLite mechanics beneath the Kernel; it is not an alternate production orchestrator.
- **Tool Gateway** — the only tool-execution authority. Models may format arguments only for the capability FORGE selected.
- **Runtime proposal boundary** — model output is proposal-only and must pass live binding and final-response guards before becoming visible.
- **Memory and retrieval** — evidence-oriented storage, retrieval, admission, provenance, and rebuildable acceleration projections.
- **Nix/NixOS substrate** — optional development shells, packages, operator-session scaffolding, and an OptiPlex workstation profile.

The simulator and target-architecture packages under `services/core/internal/forgek` remain non-authoritative. Detailed authority mapping lives in [`docs/status/current_authority_sources.md`](docs/status/current_authority_sources.md).

## Known alpha limits

The most important open areas are:

- **Chat grounding and continuation:** short replies such as “proceed” do not yet provide a universally reliable resume mechanism for pending work, and some system-inspection intents still need stronger deterministic routing.
- **Tool reliability:** the low-resource local-model path works, but real-model conformance and recovery benchmarks are still being expanded.
- **Recovery:** live raw row-merge restore is intentionally disabled. Complete daemon-stopped, generation-based whole-store recovery remains unfinished.
- **Packaging and releases:** Linux/Nix paths are the primary development target; a signed, version-unified public release channel is not complete.
- **Physical acceptance:** the OptiPlex profile builds reproducibly, while audio, removable media, printing/scanning, native-window behavior, and prolonged low-memory operation still require machine-side acceptance evidence.

These are blockers to a broad public “production-ready” claim, not reasons to hide the project. FORGE should be presented honestly as a serious engineering alpha.

## Requirements

- Go `1.24+`
- Node.js and npm
- Rust toolchain for Tauri desktop builds
- Optional: Nix for reproducible development and host profiles
- Optional: a local inference backend such as Ollama or a compatible managed runtime

GPU hardware, vLLM, and cloud providers are not required for default evaluation.

## Quick start

Install workspace dependencies:

```bash
npm install
```

Start the core:

```bash
npm run core
```

Start the desktop:

```bash
npm run desktop
```

Run the smoke path:

```bash
npm run smoke
```

Default endpoints:

- Core API: `http://127.0.0.1:18492`
- Desktop development server: `http://localhost:1420`
- Default shell route: `#/chat`

For the current operator procedure, use [`docs/runbooks/current_forge_bringup.md`](docs/runbooks/current_forge_bringup.md).

## Validation

Use the narrowest relevant test while developing, then run the broader gates before requesting review:

```bash
npm test
npm run lint
npm run validate:js
npm run validate:desktop
npm run validate:forgek
npm run validate:local
npm run smoke
```

Nix users can also run:

```bash
nix develop
nix develop .#core
nix develop .#desktop
nix build .#forge-core
nix build .#forge-desktop-shell
```

## Repository layout

- `apps/desktop` — Tauri + React operator shell.
- `services/core` — Go daemon and live production services.
- `services/core/internal/forgekernel` — production Kernel authority boundary.
- `services/core/internal/forgek` — simulator and target-architecture work; not production authority.
- `services/core/internal/gateway` — governed tool execution.
- `packages/shared` — shared contracts.
- `packages/ui` — shared UI primitives.
- `nix` — Nix/NixOS packages, modules, and host profiles.
- `docs` — architecture, status, runbooks, evidence, reviews, and historical phase material.

## Documentation map

Start with:

1. [`docs/status/CURRENT_STATE.md`](docs/status/CURRENT_STATE.md) — concise current posture and blockers.
2. [`docs/onboarding.md`](docs/onboarding.md) — contributor and agent read order.
3. [`docs/status/current_authority_sources.md`](docs/status/current_authority_sources.md) — detailed live authority map.
4. [`docs/reviews/current_phase_status.md`](docs/reviews/current_phase_status.md) — cumulative phase history and evidence notes.
5. [`docs/runbooks/current_forge_bringup.md`](docs/runbooks/current_forge_bringup.md) — operator bring-up and diagnosis.
6. [`docs/TOOL_GATEWAY.md`](docs/TOOL_GATEWAY.md) — tool boundary and execution model.
7. [`docs/MEMORY_ARCHITECTURE.md`](docs/MEMORY_ARCHITECTURE.md) — memory and evidence model.

Historical phase reports describe how FORGE reached the current state; they are not automatically current operating truth.

## Contributing and security

Read [`CONTRIBUTING.md`](CONTRIBUTING.md) before changing authority-sensitive code. Do not push feature work directly to `main`; use a focused branch and pull request with validation evidence.

Report security issues using the process in [`SECURITY.md`](SECURITY.md). Do not publish exploit details in a normal issue.
