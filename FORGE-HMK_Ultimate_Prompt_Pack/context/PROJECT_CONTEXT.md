# Project Context

F.O.R.G.E. means **Foundry for Organic Reasoning, Growth, and Execution**.

FORGE is a personal AI operating layer that coordinates user context, files, tools, models, memory, projects, workflows, policies, and execution state. It should behave like an operating layer, not a chatbot with a search box stapled to it.

## FORGE-HMK definition

FORGE-HMK is the memory-side kernel responsible for:

- MemoryCells and Synapses
- PhotoCells, KineticCells, TraceCells, ReplayCells
- HKV cache manifests
- VSA / semantic algebra projection metadata
- memory job artifacts
- Neuron Mesh worker outputs
- non-canonical context assembly
- evidence packets for validation

FORGE-HMK is not RAG. It is not a vector DB wrapper. It is not a model. It is a managed cognitive memory substrate.

## Shadow-first doctrine

FORGE-HMK begins as read-mostly and shadow-first.

It may read, assemble, compare, cache, score, validate, and propose.

It may not initially overwrite canonical memory, bypass Control Lane, expose public mutation APIs, or treat cache/vector/VSA/snapshot output as truth.
