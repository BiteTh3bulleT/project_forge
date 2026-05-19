# Chat Assistant Gateway Prompt-Injection Audit

Date: 2026-05-19

Scope: `services/core/internal/api/chat_assistant_*.go`, `services/core/internal/api/chat_post.go`, and model-response fields that can influence gateway execution.

Review target: whether model-controlled response fields can become command arguments, filesystem paths, URLs, or capability inputs without deterministic validation and gateway authority.

## Model-Output Trace Matrix

| Model response field | Assignment path | Effector surface | Guard and test evidence |
|---|---|---|---|
| `message.tool_calls[].function.name` | `completeAssistantWithGatewayTools` extracts `name`, resolves it through `gateway.ResolveChatFunctionName`, then passes the original function name to `dispatchToolCall`. | Gateway tool selection and lane selection. | Only chat-exposed gateway catalog names or legacy aliases resolve. Unknown names fail before gateway dispatch. Tests: `TestDispatchRejectsUnknownModelFunctionBeforeGateway`, `TestChatPostForcedToolMismatchUsesDeterministicFallback`. |
| `message.tool_calls[].function.arguments` | Arguments are accepted only as a string or JSON object, serialized to `argsStr`, size-bounded by `chatToolArgumentMaxBytes`, JSON-decoded into a map, and normalized by `normalizeChatInvokeArgs`. | `gateway.Request.Input` and `gateway.Request.Paths`. | Oversized and malformed arguments fail before gateway invocation. Stream/stage echoes are trimmed by `chatToolArgumentStageMaxBytes`. Tests: `TestDispatchRejectsOversizedModelToolArgumentsBeforeGateway`, `TestDispatchRejectsInvalidModelToolArgumentsBeforeGateway`. |
| `arguments.path` / `arguments.paths` | `normalizeChatInvokeArgs` separates these reserved fields from generic input and applies `normalizeChatPathAlias`; `dispatchToolCall` then prechecks every non-empty path with `pathAllowed`. | Filesystem, git, desktop path, and workspace-scoped gateway paths. | Outside-workspace and traversal paths fail before gateway invocation; gateway lane and permission checks still run after precheck. Tests: `TestDispatchRejectsModelPathTraversalBeforeGateway`, `TestPathAllowedNonRootWorkspaceRejectsOutsidePath`, `TestDispatchToolCallReadsThreadAttachmentWithoutWorkspacePath`. |
| `arguments.input.command` and other command-like fields | Reserved path fields are excluded from generic input; remaining model fields are copied into `gateway.Request.Input`. | Process/tool command arguments, service controls, capability-specific fields. | Chat never executes command text directly. It always goes through `s.gateway.Execute`, where lanes, permissions, risk, approval, and audit are enforced. Tests: `TestChatPostRepoExplorationUsesGatewayNotModelCommandSuggestions`, `TestGatewayPrivilegedToolRequiresApprovalWithoutCapabilityPolicy`, `TestGatewayDangerousCapabilityNotFreelyExecutable`. |
| `arguments.input.url` / `arguments.input.uri` / network fields | Model URL fields stay inside `gateway.Request.Input` and are interpreted only by gateway tools such as `net.fetch`, `web.search`, or `desktop.open`. | Network fetch/search or desktop URL open. | `net.fetch` validates outbound HTTP URLs and guarded HTTP clients in gateway code. `desktop.open` now applies a desktop-specific URL validator before launching browser/mail-client URLs, rejecting unsupported schemes, userinfo, missing hosts, empty `mailto:`, control characters, and oversized URLs. Forced remote-terminal style requests use `desktop.open`, not local file writes or process execution. Tests: `TestValidateDesktopOpenURL`, `TestChatPostRemoteSSHBannerUsesDesktopOpenNotLocalWrite`, gateway network URL validation tests, forced-chat web/browser routing tests. |
| `message.content` when tool calls are absent or mismatched | The assistant may append content only when no forced tool route or deterministic gateway fallback applies. For forced routes, omitted or mismatched tool calls discard model prose. | Visible assistant text only, not canonical state or tool execution. | Model prose cannot claim tool success or capability availability for forced routes. Tests: `TestChatPostForcedToolOmissionUsesForgeGatewayNotModelCapabilityClaim`, `TestChatPostForcedToolMismatchUsesDeterministicFallback`, sanitizer tests for copied transcripts. |

## Findings

- Model-selected tools are not authoritative. Tool names resolve through the chat gateway catalog and unknown names stop before dispatch.
- Model arguments are bounded, decoded as JSON, and split into paths versus capability input before dispatch.
- Filesystem paths get a chat-side workspace precheck before gateway execution, then gateway lanes and permission profiles enforce scope again.
- Command strings, URLs, and capability-specific fields are evidence for `gateway.Execute`, not direct execution instructions. URL sinks are validated in the relevant gateway tool before local fetch or desktop launch.
- Forced deterministic routes discard conflicting model tool calls and discard model prose when the model omits a required tool call.
- Deterministic chat shortcuts still dispatch through gateway helpers and preserve approval/status handling; they do not write directly from model output.

## Validation Evidence

- `TestDispatchRejectsUnknownModelFunctionBeforeGateway`
- `TestDispatchRejectsInvalidModelToolArgumentsBeforeGateway`
- `TestDispatchRejectsOversizedModelToolArgumentsBeforeGateway`
- `TestDispatchRejectsModelPathTraversalBeforeGateway`
- `TestChatPostForcedToolOmissionUsesForgeGatewayNotModelCapabilityClaim`
- `TestChatPostForcedToolMismatchUsesDeterministicFallback`
- `TestChatPostRemoteSSHBannerUsesDesktopOpenNotLocalWrite`
- `TestChatPostRepoExplorationUsesGatewayNotModelCommandSuggestions`
- `TestValidateDesktopOpenURL`

Focused validation command:

```powershell
cd services/core
go test ./internal/api -run "TestDispatch(RejectsUnknownModelFunctionBeforeGateway|RejectsInvalidModelToolArgumentsBeforeGateway|RejectsModelPathTraversalBeforeGateway|RejectsOversizedModelToolArgumentsBeforeGateway|SuppressesCompositeMkdirBeforeGateway)|TestPathAllowedNonRootWorkspaceRejectsOutsidePath|TestChatPostForcedToolOmissionUsesForgeGatewayNotModelCapabilityClaim|TestChatPostForcedToolMismatchUsesDeterministicFallback|TestChatPostRemoteSSHBannerUsesDesktopOpenNotLocalWrite|TestChatPostRepoExplorationUsesGatewayNotModelCommandSuggestions" -count=1
```

## Remaining Follow-Up

- Split `chat_assistant_gateway.go` so request flow, model-runtime fallback, tool dispatch, deterministic shortcuts, and output formatting can be reviewed independently.
- Add generated end-to-end attempts for each chat-exposed approval-only capability alias so representative approval tests become exhaustive.
- Adopt structured logs with request/correlation IDs across the chat and gateway layers.
