# Journal and Replay

## Goal

FORGE-K cannot be fully online until meaningful semantic state can be reconstructed from journaled transitions.

## Required capabilities

- canonical journal event schema
- hash-chain support
- audit linkage
- semantic syscall linkage
- gateway invocation linkage
- approval linkage
- modelruntime proposal linkage
- replay verifier
- replay dry-run command
- replay mismatch diagnostics
- corruption detection

## Done

A clean replay reconstructs semantic state without depending on raw chat transcript state.
