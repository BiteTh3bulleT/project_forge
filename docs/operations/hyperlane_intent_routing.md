# Hyperlane Intent Routing

Status date: 2026-04-27.

Hyperlane intent routing is FORGE's deterministic CPU-side classifier for simple operator requests. It exists to avoid unnecessary modelruntime calls when the request can be routed from structured state or proposed to the gateway as a typed tool intent.

Hyperlane routes only. It does not execute tools, mutate canonical truth, call modelruntime, call the network, or bypass gateway policy.

## Runtime Boundary

- Structured status, diagnostics, local chat memory/history, restore, Dream, and modelruntime inspection requests route to no-model structured responders in chat POST and assistant-stream handling when the intent is deterministic and supported.
- Filesystem and process requests become gateway-bound proposals only.
- Gateway still owns tool execution, capability checks, workspace scope, approvals, and audit.
- Kernel/control lane remains the only authority for canonical truth.
- Model prose does not decide what is available. If modelruntime is asked for a governed tool call and returns a capability claim or refusal instead, chat handling discards that prose and lets FORGE deterministic routing, gateway policy, approval checks, and structured runtime state decide the result.

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

No-model chat responses copy the intent trace into assistant message metadata and `chatLatencyTrace`, and mark `modelruntime_avoided=true`, `gateway_avoided=true`, and `context_compile_avoided=true`. Responders read bounded structured state only; they do not execute tools, write files, run commands, or call modelruntime health/chat APIs.

Chat memory responders distinguish local persisted chat history from canonical memory. `chat_threads` and `chat_messages` are non-canonical conversation history. The `remoteCrossChatContext` and Discord cross-chat settings control remote-ingress thread sharing; they do not by themselves make semantic memory canonical.

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
