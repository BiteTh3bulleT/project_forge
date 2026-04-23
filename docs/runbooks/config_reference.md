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
| `FORGE_TELEGRAM_GATEWAY_ENABLED` | telegram wire | `true` | Gateway feature flag; token absence still disables |
| `FORGE_TELEGRAM_BOT_TOKEN` | telegram wire | unset (= disabled) | Enables Telegram gateway |
| `FORGE_TELEGRAM_API_BASE`, `FORGE_TELEGRAM_POLL_TIMEOUT_S`, `FORGE_TELEGRAM_ALLOWED_CHATS` | telegram wire | standard telegram defaults | Optional tuning |
| `FORGE_DISCORD_ENABLED` | discord wire | `false` | Feature flag |
| `FORGE_DISCORD_BOT_TOKEN` | discord wire | unset | Enables Discord gateway |
| `FORGE_DISCORD_GUILD_ID`, `FORGE_DISCORD_DEFAULT_CHANNEL_ID` | discord wire | unset | Default routing |

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
| `autonomy_mode` | `maintain` | `off` / `observe` / `propose` / `maintain` / `mission`. See safety note below. |
| `autonomy_dream_enabled` | `true` | Dream-loop goroutine on/off |
| `extensions_csv` | (ingest defaults) | File types the watch/ingest pipeline considers |
| `theme` | `dark` | Desktop theme |
| `ollama_base_url` | `http://127.0.0.1:11434` | Local LLM endpoint |
| `ollama_model` | — | Model override for ollama adapter |
| `embedding_provider` | `local_hash` | Retrieval embedding backend |
| `embedding_dims` | `128` | Embedding vector size |
| `retrieval_weight_keyword` / `retrieval_weight_semantic` | `0.45` / `0.55` | Retrieval scoring mix |
| `retrieval_vsa_mode` | `off` | VSA index mode |
| `retrieval_vsa_dims` / `retrieval_vsa_seed` | `128` / `17` | VSA vector geometry |
| `chat_personality_prompt` | built-in default | System prompt |
| `remoteAccessEnabled` | `false` | External remote-access surfaces |
| `remoteAccessToken` | unset | Remote-access auth token |

## Safety defaults to preserve

- `autonomy_mode` should stay `maintain` or safer (`off`/`observe`/`propose`)
  unless the operator explicitly flips to `mission` after installing
  durable charters/budgets suited to it.
- Dangerous tools (shell, file write, process control) stay
  `approval_only` or `stubbed` per
  [dangerous_capabilities.md](dangerous_capabilities.md).
- Remote access stays `false` until the operator configures a token.
- `FORGE_WORKSPACE_DIR` defaults to `/`. For anything beyond dev
  exploration, scope it to a specific project directory.
