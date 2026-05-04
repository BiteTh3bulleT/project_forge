# FORGE-1 CPU Concept

Status: Phase 0 future research concept.

FORGE-1 is a future AI-native control processor concept. It is not an immediate implementation requirement and does not replace the current userspace FORGE-K roadmap.

FORGE-1 does not replace GPUs. GPUs compute tokens. FORGE-1 would accelerate governed execution: kernel validation, context shaping, snapshot handling, journal integrity, deterministic KV management, capability checks, and lane scheduling.

## Research Position

FORGE-1 is hardware/software co-design research. The near-term path remains a userspace deterministic cognitive microkernel, then a Rust Kernel Core, then a FORGE daemon, and only later simulator or prototype research.

## Candidate Hardware Blocks

- Cognitive Kernel Engine: deterministic syscall validation, admission checks, and lifecycle transition enforcement.
- Context Processing Unit: stable ContextBlock layout, token-addressed block assembly, budget contraction, and prompt-shape hashing.
- Snapshot and Journal Engine: append integrity, snapshot refs, source hashes, replay records, and compaction metadata.
- KV Memory Manager: deterministic cache manifest validation, tier routing, expiration, and backend assumption checks.
- Capability Fabric: policy, approval, scope, and capability checks with auditable decisions.
- Accelerator Memory Fabric: movement of token-shape and KV metadata between CPU memory, runtime backends, and accelerator memory.

## Candidate Future Instruction Families

- context instructions
- snapshot instructions
- KV instructions
- capability instructions
- journal instructions
- lane scheduling instructions

## Non-Goals

- Do not build FORGE-1 in Phase 0.
- Do not treat FORGE-1 as required for FORGE-K.
- Do not move truth authority into GPUs.
- Do not create speculative hardware code without tests, simulator contracts, and documented validation.
