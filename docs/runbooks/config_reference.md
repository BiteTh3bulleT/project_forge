# FORGE Config Reference

_All env vars and durable settings the current FORGE system reads at
boot or runtime. Observed through 2026-08-14._

## Environment variables

### Core service

| Var | Consumer | Default | Purpose |
|---|---|---|---|
| `FORGE_DATA_DIR` | [config.go:16](../../services/core/internal/config/config.go#L16) | `${XDG_CONFIG_HOME}/forge` (typically `~/.config/forge`); falls back to CWD if `UserConfigDir` errors | Location of `forge.sqlite`, `backups/`, `exports/` |
| `FORGE_CORE_PORT` | [config.go:25](../../services/core/internal/config/config.go#L25) | `18492` | HTTP listen port |
| `FORGE_CORE_BIND_HOST` | [config.go](../../services/core/internal/config/config.go) | `127.0.0.1` | HTTP bind host. Wildcard hosts (`0.0.0.0`, `::`) fail closed unless `FORGE_ALLOW_WILDCARD_BIND=true` is also set and an API token is available. |
| `FORGE_ALLOW_WILDCARD_BIND` | [config.go](../../services/core/internal/config/config.go) + [main.go](../../services/core/main.go) | `false` | Explicit opt-in required before `forge-core` may bind every interface. Wildcard binds still require API auth. |
| `FORGE_API_TOKEN` | [config.go](../../services/core/internal/config/config.go) + [auth.go](../../services/core/internal/api/auth.go) | generated under `${FORGE_DATA_DIR}/auth/api_token` | Bearer token required for `/api/*`, `/forge/*`, and enabled `/v1/*` routes. `/health` remains public. |
| `FORGE_API_TOKEN_FILE` | [config.go](../../services/core/internal/config/config.go) | `${FORGE_DATA_DIR}/auth/api_token` | Optional explicit token file path. The token is read without logging and generated on first run when absent. |
| `FORGE_API_ACTOR` | [config.go](../../services/core/internal/config/config.go) | `operator` | Authenticated actor label recorded for approval and cancellation decisions. Request-body `actor` is not authority. |
| `FORGE_CORS_ALLOWED_ORIGINS` | [routes.go](../../services/core/internal/api/routes.go) | unset | Comma-separated exact origins allowed in addition to Tauri origins. |
| `FORGE_CORS_ALLOW_DEV_LOCALHOST` | [routes.go](../../services/core/internal/api/routes.go) | `false` | Explicit dev flag for `http://localhost:*` and `http://127.0.0.1:*` browser origins. CORS is not authentication. |
| `FORGE_ENABLE_METRICS_ENDPOINT` | [config.go](../../services/core/internal/config/config.go) + [metrics.go](../../services/core/internal/api/metrics.go) | `false` | Enables bearer-authenticated `GET /metrics` Prometheus text exposition with bounded non-secret process/build/scrape metrics. When unset, `/metrics` is not mounted and returns 404. |
| `FORGE_PROJECT_CONTEXT_ALLOWED_ROOTS` | [projectcontext/service.go](../../services/core/internal/projectcontext/service.go) | unset | Comma-separated extra roots allowed for project-context imports. The workspace root is always allowed; absolute paths, `..` escapes, and symlink escapes outside allowed roots are rejected. |
| `FORGE_WORKSPACE_DIR` | [config.go](../../services/core/internal/config/config.go) | `${FORGE_DATA_DIR}/workspace`; managed NixOS service defaults to `/forge/workspaces/default` | Workspace root for file-sensitive operations. Filesystem root is rejected at core startup unless `FORGE_ALLOW_ROOT_WORKSPACE=true`. |
| `FORGE_ALLOW_ROOT_WORKSPACE` | [config.go](../../services/core/internal/config/config.go) + [main.go](../../services/core/main.go) | `false` | Explicit unsafe opt-in required before `FORGE_WORKSPACE_DIR` may be `/` or a Windows volume root. |
| `FORGE_KERNEL_AUTHORITY_MODE` | [config.go](../../services/core/internal/config/config.go) + [forgekernel](../../services/core/internal/forgekernel/kernel.go) | `forge_k` | Selects exactly one semantic syscall owner at boot. `forge_k` enables K20A ingress plus K20B durable orchestration; `legacy_v1` is rollback only. There is no dual-write mode. |
| `FORGE_K_SHADOW_MODE_ENABLED` | [config.go](../../services/core/internal/config/config.go) | `false` | Enables disabled-by-default FORGE-K shadow diagnostics. Can also be toggled at runtime by the dashboard through the durable `forge_k_shadow_mode_enabled` setting. |
| `FORGE_K_SHADOW_CHAT_METADATA_ENABLED` | [config.go](../../services/core/internal/config/config.go) | `false` | Enables Phase 12H chat metadata diagnostics only when `FORGE_K_SHADOW_MODE_ENABLED=true`. Captures bounded metadata only, never chat content, prompts, completions, request/response bodies, tool payloads, retrieval content, memory content, auth headers, cookies, or secrets. |
| `FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED` | [config.go](../../services/core/internal/config/config.go) | `false` | Enables Phase 12K-L retrieval metadata diagnostics only when `FORGE_K_SHADOW_MODE_ENABLED=true`. Captures bounded refs/counts/classes/summaries only. |
| `FORGE_K_SHADOW_ADVISORY_ENABLED` | [config.go](../../services/core/internal/config/config.go) | `false` | Enables Phase 12M-Q internal shadow advisory reports only when `FORGE_K_SHADOW_MODE_ENABLED=true`. Advisories consume existing safe diagnostics only and do not force-enable chat or retrieval observers. |
| `FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED` | [config.go](../../services/core/internal/config/config.go) | `false` | Enables Phase 14D internal Control Lane validation summary diagnostics only when `FORGE_K_SHADOW_MODE_ENABLED=true`. Captures bounded scalar validation metadata only and does not alter Control Lane decisions. |
| `FORGE_TELEGRAM_GATEWAY_ENABLED` | telegram wire | `true` | Gateway feature flag; token absence still disables |
| `FORGE_TELEGRAM_BOT_TOKEN` | telegram wire | unset (= disabled) | Enables Telegram gateway |
| `FORGE_TELEGRAM_API_BASE`, `FORGE_TELEGRAM_POLL_TIMEOUT_S`, `FORGE_TELEGRAM_ALLOWED_CHATS` | telegram wire | standard telegram defaults | Optional tuning |
| `FORGE_DISCORD_ENABLED` | discord wire | `false` | Feature flag |
| `FORGE_DISCORD_BOT_TOKEN` | discord wire | unset | Enables Discord gateway |
| `FORGE_DISCORD_GUILD_ID`, `FORGE_DISCORD_DEFAULT_CHANNEL_ID` | discord wire | unset | Default routing |

### Model runtime

| Var | Consumer | Default | Purpose |
|---|---|---|---|
| `FORGE_ENABLE_MODEL_RUNTIME` | modelruntime init | `false` | Enables governed model runtime. Auto-enabled when OpenAI-compatible API or endpoint configuration requires it. |
| `FORGE_MODEL_HOME` | model store | `${FORGE_DATA_DIR}/models` | Managed model registry/import root. |
| `FORGE_MODEL_DEFAULT_BACKEND` | backend selection | inferred (`llama_cpp` unless compatible endpoint is configured) | Default backend kind. |
| `FORGE_MODEL_DEFAULT_ID` | model selection | unset | Preferred default model id. |
| `FORGE_LLAMA_CPP_ENDPOINT`, `FORGE_LLAMA_CPP_BINARY_PATH`, `FORGE_ALLOW_LLAMA_CPP_SPAWN` | llama.cpp backend | unset / `false` | Endpoint or explicitly allowed local backend process path. |
| `FORGE_MODEL_OPENAI_COMPAT_ENDPOINT`, `FORGE_MODEL_OPENAI_COMPAT_API_KEY` | OpenAI-compatible backend | unset | Remote-compatible backend endpoint and optional bearer token. |
| `FORGE_VLLM_BASE_URL`, `FORGE_VLLM_API_KEY` | vLLM-compatible backend | unset | Canonical M4 vLLM-compatible backend endpoint and optional bearer token. This selects the governed `interactive_vllm` backend profile and remains disabled when unset. |
| `FORGE_MODEL_VLLM_ENDPOINT`, `FORGE_MODEL_VLLM_API_KEY` | vLLM-compatible backend | unset | Legacy aliases for the vLLM-compatible endpoint and optional bearer token. Canonical `FORGE_VLLM_*` values win when both are set. |
| `FORGE_MODEL_MAX_PROMPT_TOKENS`, `FORGE_MODEL_MAX_OUTPUT_TOKENS`, `FORGE_MODEL_MAX_RESPONSE_BYTES` | runtime limits | `8192` / `1024` / `262144` | Request and response bounds. |
| `FORGE_MODEL_REQUEST_TIMEOUT_MS`, `FORGE_MODEL_LOAD_TIMEOUT_MS`, `FORGE_MODEL_UNLOAD_TIMEOUT_MS`, `FORGE_MODEL_IDLE_UNLOAD_MS` | runtime lifecycle | `30000` / `120000` / `30000` / `0` | Runtime request/load/unload/idle timing. |
| `FORGE_MODEL_MAX_LOADED_MODELS` | runtime lifecycle | `1` | Loaded-model cap. |
| `FORGE_MODEL_SCHEDULER_MAX_CONCURRENT`, `FORGE_MODEL_SCHEDULER_QUEUE_CAPACITY`, `FORGE_MODEL_SCHEDULER_DISPATCH_TIMEOUT_MS` | scheduler | `1` / `8` / `5000` | Admission and dispatch bounds. |
| `FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD`, `FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD`, `FORGE_MODEL_POLICY_ALLOW_CROSS_WORKSPACE`, `FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE` | runtime policy | `true` / `false` / `false` / `true` | Workspace and autoload policy controls. |
| `FORGE_MODEL_CHAT_MAX_ATTEMPTS`, `FORGE_MODEL_CHAT_RETRY_BACKOFF_MS`, `FORGE_MODEL_CHAT_PROVIDER_COOLDOWN_MS`, `FORGE_MODEL_CHAT_MODEL_COOLDOWN_MS`, `FORGE_MODEL_CHAT_CHECKPOINT_LIMIT` | chat orchestration | `3` / `250` / `5000` / `5000` / `128` | Retry pacing, cooldown, and checkpoint bounds. |
| `FORGE_ENABLE_OPENAI_COMPAT_API` | `/v1/*` routes | `false` | Enables gated OpenAI-compatible model API surface. |

Local dev startup note: `npm run core` auto-detects a running
Ollama OpenAI-compatible endpoint at `OLLAMA_BASE_URL` or
`http://127.0.0.1:11434` when no explicit modelruntime backend env vars are
set. In that case it enables modelruntime, configures
`FORGE_MODEL_OPENAI_COMPAT_ENDPOINT`, allows governed autoload for the local
dev session, and selects a non-embedding default model when available. Set
`FORGE_DISABLE_OLLAMA_AUTODETECT=true` or any explicit runtime backend env var
to bypass autodetect.

### Ollama native chat adapter

| Var | Consumer | Default | Purpose |
|---|---|---|---|
| `FORGE_OLLAMA_CHAT_NUM_PREDICT` | native Ollama `/api/chat` adapter | `96` | Sets Ollama `options.num_predict` for native chat and streaming chat calls. Set `0` to omit this option. |
| `FORGE_OLLAMA_CHAT_NUM_CTX` | native Ollama `/api/chat` adapter | `1024` | Sets Ollama `options.num_ctx` for native chat and streaming chat calls. Set `0` to omit this option. |
| `FORGE_OLLAMA_CHAT_THINK` | native Ollama `/api/chat` adapter | unset | When explicitly set to a boolean, forwards Ollama's top-level `think` control. The OptiPlex tool worker sets this to `false` so its bounded output budget is available for structured tool calls. |
| `FORGE_OLLAMA_CHAT_NUM_THREAD` | native Ollama `/api/chat` adapter | unset | Sets Ollama `options.num_thread` when positive. Unset or `0` omits this option. |

### Model runtime GPU/safe-mode policy

| Var | Consumer | Default | Purpose |
|---|---|---|---|
| `FORGE_GPU_ENABLED` | `services/core/internal/config/config.go` -> modelruntime init | `false` | Enables GPU-aware runtime policy path (kernel authority remains CPU/RAM). |
| `FORGE_NVIDIA_DCGM_ENABLED` | core GPU telemetry | `false` | Enables optional NVIDIA DCGM exporter telemetry. |
| `FORGE_NVIDIA_DCGM_ENDPOINT` | core GPU telemetry | unset | Prometheus metrics endpoint, usually `http://127.0.0.1:9400/metrics`. |
| `FORGE_NVIDIA_DCGM_TIMEOUT_MS` | core GPU telemetry | `1500` | HTTP timeout for DCGM metrics fetch. |
| `FORGE_INTEL_LEVEL_ZERO_ENABLED` | core GPU telemetry | `false` | Enables optional Intel Level Zero telemetry probe. |
| `FORGE_INTEL_LEVEL_ZERO_ZE_INFO_PATH` | core GPU telemetry | `ze_info` from `PATH` | Optional explicit `ze_info` path. |
| `FORGE_INTEL_GPU_TOP_PATH` | core GPU telemetry | `intel_gpu_top` from `PATH` | Optional explicit `intel_gpu_top` path for utilization sampling. |
| `FORGE_INTEL_GPU_TELEMETRY_TIMEOUT_MS` | core GPU telemetry | `1500` | Timeout for Intel telemetry command probes. |
| `FORGE_GPU_BACKGROUND_MEMORY_PRESSURE_BLOCK_THRESHOLD` | modelruntime admission policy | `0.90` | Defers background GPU jobs when DCGM memory pressure reaches this threshold. |
| `FORGE_GPU_REQUIRED_FOR_INTERACTIVE_INFERENCE` | modelruntime scheduler/policy | `false` | If `true`, interactive inference requests are rejected when GPU is unavailable. |
| `FORGE_GPU_VRAM_HEADROOM_FRACTION` | modelruntime policy metadata | `0.20` | VRAM headroom target fraction for guarded GPU scheduling policy. |
| `FORGE_GPU_BACKGROUND_JOBS_ENABLED` | modelruntime scheduler | `false` | Enables background GPU workload classes (embedding/rerank/distillation/eval/training). |
| `FORGE_GPU_BACKGROUND_IDLE_THRESHOLD_SECONDS` | modelruntime scheduler cooldown | `300` | Idle/cooldown threshold before background GPU classes are admitted. |
| `FORGE_GPU_MAX_BACKGROUND_JOBS` | modelruntime scheduler | `1` | Hard cap for concurrent background GPU jobs. |
| `FORGE_DREAM_MODE_ALLOW_GPU_SUBJOBS` | modelruntime dream workload classification | `false` | Allows Dream Mode to emit optional GPU-classified runtime jobs. |
| `FORGE_DREAM_MODE_GPU_ONLY_IN_DEEP_IDLE` | runtime policy metadata + operator health visibility | `true` | Declares Dream GPU work as deep-idle-only policy posture. |
| `FORGE_SAFE_MODE_FORCE_CPU_ONLY` | core + modelruntime init | `false` | Forces safe mode CPU-only runtime posture; GPU classes are disabled/deferred. |
| `FORGE_MODELRUNTIME_DEGRADED_ON_UNAVAILABLE_GPU` | modelruntime health state | `true` | Marks runtime state degraded/unavailable when GPU is expected but unavailable. |
| `FORGE_SCHEDULING_INTERACTIVE_PRIORITY_OVER_BACKGROUND` | modelruntime scheduler | `true` | Ensures interactive workload classes preempt/defer background classes. |

### Embedding providers

| Var | Consumer | Default | Purpose |
|---|---|---|---|
| `FORGE_EMBEDDING_PROVIDER` | embedding service settings bootstrap | unset (`local_hash` setting default) | Optional override such as `tei`. |
| `FORGE_EMBEDDING_MODEL` | embedding service settings bootstrap | unset | Optional provider model label. |
| `FORGE_EMBEDDING_DIMS` | embedding service settings bootstrap | `128` | Local hash vector dimensions and provider metadata default. |
| `FORGE_EMBEDDING_TEI_ENDPOINT` | TEI provider | unset | Hugging Face TEI endpoint. |
| `FORGE_EMBEDDING_TEI_API_KEY` | TEI provider | unset | Optional bearer token. |
| `FORGE_EMBEDDING_TEI_TIMEOUT_MS` | TEI provider | `30000` | TEI preflight/embed timeout. |

### Desktop (build-time; Vite inlines)

| Var | Consumer | Default | Purpose |
|---|---|---|---|
| `VITE_FORGE_API_URL` | [client.ts](../../apps/desktop/src/lib/api/client.ts) | `http://127.0.0.1:18492` | Backend URL baked into the built frontend. Native Tauri reads the local API token through `read_forge_api_token` and sends it as a bearer header. |

Template: [apps/desktop/.env.example](../../apps/desktop/.env.example). Copy
to `apps/desktop/.env.development` (or `.env.production`) to override.

### Orchestration scripts

| Var | Consumer | Default | Purpose |
|---|---|---|---|
| `FORGE_CORE_PORT` | `npm run up`, `npm run down` | `18492` | Health-check target / cleanup port |
| `FORGE_CORE_BIND_HOST` | `services/core` | `127.0.0.1` | Direct core bind host; orchestration health checks use loopback. |
| `FORGE_ALLOW_WILDCARD_BIND` | `services/core` | `false` | Required only when intentionally binding `FORGE_CORE_BIND_HOST` to `0.0.0.0` or `::`; wildcard still requires API auth. |

### OS integration readiness

Run `npm run test:os-integration` and then `npm run validate:os-integration`
before native desktop VM rebuilds or boot evidence capture. The test command
exercises the gate's failure behavior. The validation command performs
cross-platform static checks for the FORGE native desktop runtime, canonical
operator VM, operator shell package wiring, loopback-only service defaults,
local Ollama/modelruntime wiring, `/forge` durable storage roots, safe-mode
defaults, disabled autologin, TTY fallback markers, and disabled shell
host-mutation authority flags. It complements Nix flake checks and is usable on
hosts where `nix` is unavailable.

## Durable settings (stored in SQLite)

Selected keys the core reads via `loadSetting(db, key, default)`. Most
are editable via `PUT /api/settings` or the desktop Settings page.
Full list in [server.go handleGetSettings / handleUpdateSettings](../../services/core/internal/api/server.go).

| Key | Default | Effect |
|---|---|---|
| `autonomy_mode` | `observe` | `off` / `observe` / `propose` / `maintain` / `mission`. See safety note below. |
| `autonomy_dream_enabled` | `true` | Autonomy idle dream-loop goroutine on/off |
| `dream_mode_enabled` | `true` | Operator-facing `/api/dream/run` default availability flag exposed in settings |
| `dream_mode_default_dry_run` | `true` | Default posture for Dream Mode v0 reports |
| `dream_mode_mode` | `microdream` | Default Dream Mode v0 depth (`microdream`, `nap`, `deep_dream`) |
| `dream_mode_window_hours` | `6` | Default replay window override for Dream Mode settings surfaces |
| `dream_mode_max_candidates` | `8` | Default replay candidate limit for Dream Mode settings surfaces |
| `dream_mode_allow_long_term_promotion` | `false` | Allows long-term promotion proposals in dry-run reports |
| `dream_mode_require_operator_review_for_long_term` | `true` | Keeps long-term proposals review-routed unless explicitly relaxed |
| `dream_mode_allow_commits` | `false` | Reserved; Dream Mode v0 ignores commit requests and remains dry-run only |
| `forge_k_shadow_mode_enabled` | env/default `false` | Runtime dashboard toggle for global FORGE-K shadow diagnostics; updates the running observer without restarting |
| `forge_k_shadow_chat_metadata_enabled` | env/default `false` | Durable override for chat metadata diagnostics; still requires global shadow mode |
| `forge_k_shadow_retrieval_metadata_enabled` | env/default `false` | Durable override for retrieval metadata diagnostics; still requires global shadow mode |
| `extensions_csv` | (ingest defaults) | File types the watch/ingest pipeline considers |
| `theme` | `dark` | Desktop theme |
| `ollama_base_url` | `http://127.0.0.1:11434` | Local LLM endpoint |
| `ollama_model` | — | Persisted default for the Ollama adapter. A valid locally discovered model explicitly selected for a chat request takes precedence for that request; otherwise this value, then `OLLAMA_MODEL`, is used. |
| `embedding_provider` | `local_hash` | Retrieval embedding backend |
| `embedding_model` | provider default | Embedding provider model label |
| `embedding_dims` | `128` | Embedding vector size |
| `embedding_tei_endpoint` | unset | Hugging Face TEI endpoint |
| `embedding_tei_api_key` | unset | Optional TEI bearer token |
| `embedding_tei_timeout_ms` | `30000` | TEI HTTP timeout |
| `retrieval_weight_keyword` / `retrieval_weight_semantic` | `0.45` / `0.55` | Retrieval scoring mix |
| `retrieval_vsa_mode` | `off` | VSA index mode |
| `retrieval_vsa_dims` / `retrieval_vsa_seed` | `128` / `17` | VSA vector geometry |
| `chat_personality_prompt` | built-in default | System prompt |
| `remoteAccessEnabled` | `false` | External remote-access surfaces |
| `remoteAccessToken` | unset | Remote-access auth token |

## Safety defaults to preserve

- `autonomy_mode` should stay `observe` or safer (`off`) unless the
  operator explicitly flips to `propose`, `maintain`, or `mission` after
  reviewing durable charters/budgets suited to it.
- Dangerous tools (shell, process control, external effects, privileged
  operations) stay `approval_only` per
  [dangerous_capabilities.md](../status/dangerous_capabilities.md).
- Remote access stays `false` until the operator configures a token.
- Direct Go/dev `FORGE_WORKSPACE_DIR` defaults to `/`; managed NixOS
  `forge-core` defaults to `/forge/workspaces/default`. For any real
  project, scope it to a specific project directory or dedicated
  workspace path instead of host root.
