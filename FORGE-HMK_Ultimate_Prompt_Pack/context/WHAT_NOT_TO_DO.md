# WHAT NOT TO DO

## Do not violate authority

- Do not make FORGE-HMK live authority in early phases.
- Do not bypass FORGE-K / Control Lane.
- Do not let workers write canonical state.
- Do not let Crucible commit truth directly.
- Do not let cache/vector/VSA/snapshot output become truth.

## Do not create a swarm mess

- Do not create generic all-powerful agents.
- Do not allow unbounded worker fan-out.
- Do not allow workers to spawn unlimited child jobs.
- Do not run jobs without budgets, leases, cancellation, and scope.

## Do not flatten memory

- Do not collapse PhotoCell, KineticCell, TraceCell, ReplayCell into one blob.
- Do not overwrite old truth when superseding it.
- Do not delete contradiction evidence silently.
- Do not treat recency as truth.

## Do not wreck performance

- Do not recompute stable context every request.
- Do not full-scan archive when hot state is enough.
- Do not prewarm everything.
- Do not use one TTL policy for every cache layer.
- Do not hide cache invalidation.

## Do not hide behavior

- Do not create invisible memory promotions.
- Do not suppress validation warnings.
- Do not omit provenance from promotable claims.
- Do not skip telemetry because “it works.” That sentence is the preface to production pain.

## Do not put implementation code in chat

Write code to files. Summarize changed files and tests in chat.
