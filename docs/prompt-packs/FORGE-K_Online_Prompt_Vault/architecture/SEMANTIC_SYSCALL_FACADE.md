# Semantic Syscall Facade

## Purpose

Normalize live semantic writes into typed syscall-shaped requests before migrating Kernel commit authority.

## Requirements

Every live semantic syscall request must include:

- syscall id
- operation
- actor
- workspace id
- capability scope
- provenance
- correlation id
- trace id
- idempotency key
- refs
- expected effect
- dry-run flag
- rollback metadata

## Rule

The facade can live in Control Lane first. It must not import simulator Kernel services.
