# FORGE Config Reference

_All env vars and durable settings the current FORGE system reads at
boot or runtime. Observed 2026-04-21._

## Environment variables

### Core service

| Var | Consumer | Default | Purpose |
|---|---|---|---|
| `FORGE_DATA_DIR` | [config.go:16](services/core/internal/config/config.go#L16) | `${XDG_CONFIG_HOME}/forge` (typically `~/.config/forge`); falls back to CWD if `UserConfigDir` errors | Location of `forge.sqlite`, `backups/`, `exports/` |
| `FORGE_CORE_PORT` | [config.go:25](services/core/internal/config/config.go#L25) | `18492` | HTTP listen port |
| `FORGE_WORKSPACE_DIR` | [config.go:30](services/core/internal/config/config.go#L30) | `/` | Workspace root for file-sensitive operations |
| `FORGE_K_SHADOW_MODE_ENABLED` | [config.go:175](services/core/internal/config/config.go#L175) | `false` | Enables Phase 12B read-only `/health` metadata shadow diagnostics. Disabled by default; no public API, route, response, memory, retrieval, gateway, or modelruntime behavior changes. |
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
| `FORGE_MODEL_VLLM_ENDPOINT`, `FORGE_MODEL_VLLM_API_KEY` | vLLM-compatible backend | unset | vLLM-compatible backend endpoint and optional bearer token. |
| `FORGE_MODEL_MAX_PROMPT_TOKENS`, `FORGE_MODEL_MAX_OUTPUT_TOKENS`, `FORGE_MODEL_MAX_RESPONSE_BYTES` | runtime limits | `8192` / `1024` / `262144` | Request and response bounds. |
| `FORGE_MODEL_REQUEST_TIMEOUT_MS`, `FORGE_MODEL_LOAD_TIMEOUT_MS`, `FORGE_MODEL_UNLOAD_TIMEOUT_MS`, `FORGE_MODEL_IDLE_UNLOAD_MS` | runtime lifecycle | `30000` / `120000` / `30000` / `0` | Runtime request/load/unload/idle timing. |
| `FORGE_MODEL_MAX_LOADED_MODELS` | runtime lifecycle | `1` | Loaded-model cap. |
| `FORGE_MODEL_SCHEDULER_MAX_CONCURRENT`, `FORGE_MODEL_SCHEDULER_QUEUE_CAPACITY`, `FORGE_MODEL_SCHEDULER_DISPATCH_TIMEOUT_MS` | scheduler | `1` / `8` / `5000` | Admission and dispatch bounds. |
| `FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD`, `FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD`, `FORGE_MODEL_POLICY_ALLOW_CROSS_WORKSPACE`, `FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE` | runtime policy | `true` / `false` / `false` / `true` | Workspace and autoload policy controls. |
| `FORGE_MODEL_CHAT_MAX_ATTEMPTS`, `FORGE_MODEL_CHAT_RETRY_BACKOFF_MS`, `FORGE_MODEL_CHAT_PROVIDER_COOLDOWN_MS`, `FORGE_MODEL_CHAT_MODEL_COOLDOWN_MS`, `FORGE_MODEL_CHAT_CHECKPOINT_LIMIT` | chat orchestration | `3` / `250` / `5000` / `5000` / `128` | Retry pacing, cooldown, and checkpoint bounds. |
| `FORGE_ENABLE_OPENAI_COMPAT_API` | `/v1/*` routes | `false` | Enables gated OpenAI-compatible model API surface. |

Local dev startup note: `scripts/forge-core.sh` auto-detects a running
Ollama OpenAI-compatible endpoint at `OLLAMA_BASE_URL` or
`http://127.0.0.1:11434` when no explicit modelruntime backend env vars are
set. In that case it enables modelruntime, configures
`FORGE_MODEL_OPENAI_COMPAT_ENDPOINT`, allows governed autoload for the local
dev session, and selects a non-embedding default model when available. Set
`FORGE_DISABLE_OLLAMA_AUTODETECT=true` or any explicit runtime backend env var
to bypass autodetect.

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
| `VITE_FORGE_API_URL` | [api.ts:258](apps/desktop/src/lib/api.ts#L258) | `http://127.0.0.1:18492` | Backend URL baked into the built frontend |

Template: [apps/desktop/.env.example](apps/desktop/.env.example). Copy
to `apps/desktop/.env.development` (or `.env.production`) to override.

### Orchestration scripts

| Var | Consumer | Default | Purpose |
|---|---|---|---|
| `FORGE_CORE_PORT` | `scripts/forge-up.sh`, `scripts/forge-down.sh` | `18492` | Health-check target / cleanup port |

## Durable settings (stored in SQLite)

Selected keys the core reads via `loadSetting(db, key, default)`. Most
are editable via `PUT /api/settings` or the desktop Settings page.
Full list in [server.go handleGetSettings / handleUpdateSettings](services/core/internal/api/server.go).

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
| `extensions_csv` | (ingest defaults) | File types the watch/ingest pipeline considers |
| `theme` | `dark` | Desktop theme |
| `ollama_base_url` | `http://127.0.0.1:11434` | Local LLM endpoint |
| `ollama_model` | — | Model override for ollama adapter |
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
  [dangerous_capabilities.md](dangerous_capabilities.md).
- Remote access stays `false` until the operator configures a token.
- `FORGE_WORKSPACE_DIR` defaults to `/`. For anything beyond dev
  exploration, scope it to a specific project directory.
