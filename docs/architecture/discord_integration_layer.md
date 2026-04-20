# Discord Integration Layer

## Purpose

Discord is an external I/O surface for FORGE. It is not the reasoning core.

Boundary rule:

- Discord transport receives/sends messages.
- FORGE gateway normalizes and routes events.
- FORGE services handle intent, memory, policy, and audit.
- Durable truth changes still go through semantic syscall + kernel validation paths.

## Runtime architecture

```text
Discord Bot Session (discordgo)
  -> DiscordGateway transport handlers
  -> canonical discordEventEnvelope
  -> discordIntent routing + permission check
  -> FORGE services (chat/search/dashboard/adapters)
  -> discordResponse formatter
  -> Discord outbound sender (channel reply or interaction response)

Every step emits event/audit correlation metadata.
```

## Core modules

- `services/core/internal/api/discord_gateway_types.go`
  - gateway config, event envelope, intent, response, status contracts
- `services/core/internal/api/discord_gateway_translate.go`
  - canonical normalization (`MessageCreate`/`InteractionCreate` to envelope)
- `services/core/internal/api/discord_gateway_router.go`
  - text/slash command parsing into `discordIntent`
- `services/core/internal/api/discord_gateway_permissions.go`
  - actor/role permission scaffolding (admin command gate)
- `services/core/internal/api/discord_gateway_service.go`
  - bot lifecycle, handlers, intent execution, outbound response, audit/event logging
- `services/core/internal/api/discord_gateway_server.go`
  - server startup wiring, status endpoint, conversation enqueue bridge

## Canonical event envelope

Inbound Discord payloads are normalized into `discordEventEnvelope` with:

- `source=discord`
- event type (`message_create` or `interaction_create`)
- guild/channel/user identity
- message/interaction id
- timestamp
- raw content
- metadata
- correlation id + trace id
- permission context
- normalized actor identity (id/name/roles/bot flag)

This normalization is deterministic and unit tested.

## Intent routing

Supported intent classes:

- `direct_command`
- `conversational_input`
- `system_query`
- `agent_control_request`
- `memory_query`
- `automation_event`

Initial command set:

- `/forge ping`
- `/forge status`
- `/forge memory query <text>`
- `/forge agents` (admin-gated)
- `/forge help`

Text fallback:

- `!forge ping|status|memory query <text>|agents|help`

Conversation fallback:

- bot mention (`<@botId> ...`) routes to conversational intent
- optional passive listening can also route conversation input

## Response pipeline

Responses are generated as `discordResponse` contracts and formatted by kind:

- plain
- status
- error
- diagnostic

Outbound path supports:

- channel message replies
- interaction responses (ephemeral or public)

Rich embeds are intentionally deferred but the response contract is metadata-ready.

## Permission and safety

Current scaffolding:

- actor identity derives from Discord user + roles
- admin-restricted command category (`agents`) checks configured admin user/role ids
- unknown command and denied actions return deterministic error responses
- denials are auditable (`discord_gateway` audit category)

Future extension:

- map Discord identities into permission profiles and capability checks
- gate destructive/external actions through approval and autonomy charter checks

## Observability and audit

Gateway emits structured events:

- `discord.gateway.started`
- `discord.gateway.inbound`
- `discord.gateway.outbound`
- `discord.gateway.error`
- `discord.gateway.intent.enqueued`

Gateway writes audit entries for accepted/denied intents with:

- correlation id
- actor
- command
- outcome
- payload context

API surface:

- `GET /api/discord/status` exposes runtime status/counters and registration state.

## Configuration

Environment variables:

- `FORGE_DISCORD_ENABLED` (`true|false`)
- `FORGE_DISCORD_BOT_TOKEN`
- `FORGE_DISCORD_APP_ID`
- `FORGE_DISCORD_GUILD_ID` (optional; restrict bot to one guild)
- `FORGE_DISCORD_DEFAULT_CHANNEL_ID` (optional fallback)
- `FORGE_DISCORD_COMMAND_PREFIX` (default `!forge`)
- `FORGE_DISCORD_ENABLE_SLASH_COMMANDS` (default `true`)
- `FORGE_DISCORD_ENABLE_TEXT_COMMANDS` (default `true`)
- `FORGE_DISCORD_ENABLE_PASSIVE_LISTENING` (default `false`)
- `FORGE_DISCORD_ENABLE_OUTBOUND_POSTING` (default `true`)
- `FORGE_DISCORD_ADMIN_USER_IDS` (CSV)
- `FORGE_DISCORD_ADMIN_ROLE_IDS` (CSV)

Settings fallback is available for bot token/default channel/remote enablement to keep current desktop settings behavior compatible.

## Operator setup

1. Create a Discord application and bot in Discord Developer Portal.
2. Enable bot intents needed for your mode:
   - Guild Messages
   - Direct Messages
   - Message Content (if using text commands/passive listening)
3. Add the bot to your target guild.
4. Set FORGE environment variables (at minimum token + enabled flag).
5. Start FORGE core.
6. Verify `GET /api/discord/status` reports `enabled=true` and `connected=true`.
7. Run `/forge ping` (or `!forge ping`) in Discord.

## Forge integration notes

- Conversation intents use existing chat assistant async path (`enqueueDiscordConversation`) so Discord remains transport-only.
- Memory queries use existing search service.
- Status queries use existing dashboard + autonomy status.
- Discord interactions do not bypass approvals, permissions, or audit subsystems.

## Extension points

- Add new command handlers in `executeIntent`.
- Expand role-to-policy mapping in `authorizeDiscordIntent`.
- Route selected intents into autonomy intent queue for operator-approved self-actions.
- Add embed/structured card response adapter while keeping core response contract stable.
