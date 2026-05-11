# Chat Assistant Gateway Audit

Date: 2026-05-11

Scope: `services/core/internal/api/chat_assistant_gateway.go`

Review target: prompt-injection to effector paths, specifically whether model-controlled strings can become workspace paths, command arguments, URLs, or capability inputs without governed validation.

## Findings

- Chat tool calls resolve through `gateway.ResolveChatFunctionName`; unknown model function names are rejected before gateway dispatch.
- Tool execution is centralized through `s.gateway.Execute`, preserving gateway lanes, permission checks, approval checks, audit records, and bounded tool implementations.
- Chat-side path precheck rejects traversal or paths outside the configured workspace before dispatch, with gateway scope checks still applied after path resolution.
- Forced gateway routes discard mismatched model tool calls and discard model prose when the model omits a required tool call.
- Deterministic chat shortcuts still dispatch through gateway helpers and approval/status handling; they do not write directly from model output.

## Hardening Added

Model-supplied tool argument JSON is now bounded at the chat gateway boundary before JSON decoding and before gateway execution. Oversized argument payloads are rejected with a `tool_args_too_large` stage and do not create gateway invocations.

Model tool-call argument echoes emitted to stream/stage metadata are now trimmed before logging or event emission.

## Remaining Follow-Up

- Split `chat_assistant_gateway.go` so request flow, model-runtime fallback, tool dispatch, deterministic shortcuts, and output formatting can be reviewed independently.
- Continue raising end-to-end chat gateway tests around model-selected `fs.write`, `proc.run`, `net.fetch`, and `desktop.open` cases.
- Adopt structured logs with request/correlation IDs across the chat and gateway layers.
