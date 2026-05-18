# Context Compiler Live Migration

## Goal

Move prompt/context construction from ad hoc transcript/context stitching into deterministic ContextBundles built from admitted evidence.

## Migration ladder

1. Shape validation only
2. Shadow ContextBundle generation
3. Compare against current prompt builder
4. Canary one route/thread type
5. Operator-visible context inspector
6. Default live prompt authority
7. Retire old context builder path

## Required outputs

- included refs
- excluded refs
- admission status
- budget reason
- source citations
- bundle hash
- prompt layout version
