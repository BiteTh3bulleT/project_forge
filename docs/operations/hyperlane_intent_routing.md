# Hyperlane Intent Routing

Status date: 2026-04-27.

Hyperlane intent routing is FORGE's deterministic CPU-side classifier for simple operator requests. It exists to avoid unnecessary modelruntime calls when the request can be routed from structured state or proposed to the gateway as a typed tool intent.

Hyperlane routes only. It does not execute tools, mutate canonical truth, call modelruntime, call the network, or bypass gateway policy.

## Runtime Boundary

- Structured status, diagnostics, restore, Dream, and modelruntime inspection requests can route to no-model structured responders.
- Filesystem and process requests become gateway-bound proposals only.
- Gateway still owns tool execution, capability checks, workspace scope, approvals, and audit.
- Kernel/control lane remains the only authority for canonical truth.

## Safety Rules

Path fields are normalized before a route proposal is returned:

- slashes are normalized
- surrounding quotes and punctuation are trimmed
- empty paths are rejected
- traversal segments are rejected
- absolute user paths are rejected unless the path came from an explicit gateway-derived context hint
- shell metacharacters are rejected in path fields
- file-name-only fields cannot smuggle directories

Shell command intents are classified as `run_command`, route to `proc.run`, use risk class `high`, and carry `requires_approval_hint=true`. Hyperlane never runs the command.

## Trace

Every parse includes:

- parser version
- matched rule
- confidence
- route
- warnings
- rejected reason when unknown or unsafe

The trace is compact and suitable for chat/process diagnostics. It is not a durable authority record.

## Templates

Deterministic demo/template content is isolated in gateway template helpers. Templates are not general intelligence and are not executed automatically. A template result is only proposed as a `generate_template` intent routed through `fs.write`.

## Adding An Intent Safely

Add a deterministic intent only when all of the following are true:

- classification can be done with CPU-local string/regex checks
- no model call is required
- no tool execution is performed by the parser
- route maps to an existing structured route or existing gateway tool id
- risk class and approval hint are conservative
- unsafe paths or arguments are rejected before route proposal
- tests cover match, trace, unsafe rejection, and compatibility behavior
