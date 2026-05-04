# ADR 0001 - FORGE-K Is a Cognitive Microkernel

Status: Accepted

Date: 2026-05-03

## Context

FORGE already separates approvals, packets, gateway execution, audit, semantic syscalls, model runtime management, and journaled semantic persistence. Phase 0 establishes FORGE-K as the durable architecture baseline for the next build sequence.

Without an explicit microkernel decision, future work could drift toward a conventional agent framework, a chatbot wrapper, or premature bare-metal operating system work.

## Decision

FORGE-K will be built as a userspace deterministic cognitive microkernel first, not as a bare-metal OS kernel and not as a conventional agent framework.

The Kernel owns canonical truth through semantic syscalls, deterministic validation, and journaled commit boundaries. Model runtimes, neurons, tools, and adapters operate outside commit authority unless they submit valid requests through the Kernel path.

## Consequences

- Early work focuses on contracts, envelopes, validation, admission, journaling, replay, and tests.
- Runtime drivers remain isolated from truth authority.
- Bare-metal and hardware work stays future research until userspace semantics are proven.
- The architecture favors small kernel contracts and bounded workers over monolithic agent loops.

## Alternatives considered

- Conventional agent framework: rejected because agent loops would blur proposal, validation, admission, and commit authority.
- Chatbot wrapper: rejected because raw conversation flow cannot own canonical state.
- Bare-metal OS first: rejected because semantic kernel contracts should be proven in userspace before hardware or kernel-level implementation.
