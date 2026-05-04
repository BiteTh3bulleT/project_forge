# ADR 0006 - Rust Kernel Core Boundary

Status: Accepted

Date: 2026-05-04

## Context

FORGE-K Phase 1-10 exists as a Go simulator authority under `services/core/internal/forgek`. It covers the Kernel simulator, Neuron Fabric, Courthouse, Memory Palace, Semantic Algebra, Snapshots, Context Compiler, Deterministic KV metadata, Runtime Driver Boundary, and Lymphatic Lane.

Rust may be valuable for deterministic kernel primitives such as canonical serialization, hashing, validation, manifest identity, and journal integrity. These primitives are small, testable, and sensitive to ordering, schema drift, and accidental nondeterminism.

The live daemon is still not FORGE-K-governed. ADR 0005 states that live state mutation continues through existing AI-OS, gateway, permissions, lane, audit, model runtime, and API authority paths until a later explicit `LIVE_INTEGRATION` phase changes that boundary.

Rust must not become a second ungoverned authority path.

## Decision

FORGE-K will prefer a Rust library boundary for deterministic validation primitives only.

Initial Rust candidates are:

- canonical serialization
- hashing
- manifest validation
- capability check primitives
- journal integrity primitives
- KV nine-gate validation primitives

The recommended first implementation, if Phase 11B is approved, is a standalone Rust crate with a CLI test harness and shared fixture corpus. It should not use cgo, should not be imported by live daemon code, and should not mutate state.

## Non-Goals

- No live daemon integration.
- No model runtime calls.
- No gateway or tool execution.
- No route or API changes.
- No replacement of the Go simulator yet.
- No live state mutation.
- No cgo or service sidecar until a later explicit phase approves it.

## Consequences

Positive:

- stronger invariant enforcement
- clearer stable kernel contracts
- easier future hardware/software co-design path
- better deterministic test corpus
- lower risk of serialization and hash drift after parity fixtures exist

Negative:

- adds language and tooling complexity
- requires stable serialization contracts
- can slow development if started too early
- FFI boundary must be carefully tested before any integration
- Go/Rust drift becomes a maintenance risk without shared fixtures

## Alternatives Considered

### Keep Everything In Go

This is viable for the current simulator and remains the default live behavior. It avoids toolchain complexity but delays cross-language deterministic validation.

### Rewrite All FORGE-K In Rust

Rejected. It would throw away working simulator tests, broaden scope, and risk creating a second authority path.

### Start With A Rust Daemon

Rejected. A daemon would introduce lifecycle, IPC, deployment, and authority risks before primitive parity is proven.

### Start With A Tiny Rust Validation Library

Accepted as the preferred direction for Phase 11B, with a CLI harness first and no live daemon integration.
