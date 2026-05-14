# Micro-Agent Charter Template

## Agent name

`<name>`

## Purpose

What latency or preparation problem does this worker solve?

## Trigger

Event-driven, schedule-driven, user-session-driven, startup, idle, or manual refresh.

## Inputs

List all inputs. Mark any sensitive input.

## Outputs

List exact outputs.

## Output authority

Choose one: cache only, proposal only, advisory only, evidence summary only, diagnostic only.

## Forbidden outputs

List what the worker must never produce.

## Storage

Where does output live: in-memory cache, SQLite diagnostic table, artifact, shadow report, or proposal queue?

## TTL / expiry

Define cache lifetime.

## Audit / provenance

What event or trace is recorded?

## Failure behavior

What happens on error?

## Anti-loop guard

How does it avoid repeated churn?

## Tests required

Happy path, missing input, stale output, error path, no authority mutation, no raw secret leak.

## WHAT NOT TO DO

Do not write canonical memory, approve, execute tools, mutate host, bypass Gateway, bypass modelruntime, or hide failure.
