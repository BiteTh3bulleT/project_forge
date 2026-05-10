# Remote Access (Telegram / Discord)

Forge can receive messages from Telegram and Discord through signed webhook endpoints in the Core API:

- `POST /api/remote/telegram`
- `POST /api/remote/discord`

Core base URL (default): `http://127.0.0.1:18492`

## Configure once in Settings

1. Open **Settings → Remote Access**.
2. Enable remote access.
3. Set a strong **Remote token**.
4. Set:
   - Telegram bot token and/or
   - Discord bot token and default channel, or Discord webhook URL.

Save remote settings.

## Telegram setup

1. Set Telegram webhook to Forge:

```bash
https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/setWebhook?url=<FORGE_PUBLIC_URL>/api/remote/telegram
```

2. Send secret token with webhook so Forge can authenticate calls:

```bash
curl -X POST "https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{"url":"<FORGE_PUBLIC_URL>/api/remote/telegram","secret_token":"<FORGE_REMOTE_TOKEN>"}'
```

`secret_token` is validated from `X-Telegram-Bot-Api-Secret-Token`.

### Telegram test payload

```bash
curl -X POST <FORGE_PUBLIC_URL>/api/remote/telegram \
  -H "X-Forge-Remote-Token: <FORGE_REMOTE_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"message":{"message_id":1,"text":"How are we looking?","chat":{"id":123456789},"from":{"id":123}}}'
```

## Discord setup

FORGE supports two input modes:

1. **Discord bot flow**
   - POSTs from your bot platform to `/api/remote/discord`.
   - Provide a reply-capable `discord bot token` and default channel in Settings.

2. **Incoming webhook flow**
   - POSTs to `/api/remote/discord`.
   - For channel-less senders, set `discordWebhookUrl` in Settings and Forge will use that for outbound replies.

### Discord test payload

```bash
curl -X POST <FORGE_PUBLIC_URL>/api/remote/discord \
  -H "X-Forge-Remote-Token: <FORGE_REMOTE_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"channel_id":"<DISCORD_CHANNEL_ID>","content":"How are we looking?","author":{"id":"12345"}}'
```

## Notes

- Replies are sent asynchronously to the configured platform channel.
- Remote calls are rejected if remote access is disabled.
- Remote tokens must be sent in `X-Forge-Remote-Token` or Telegram's `X-Telegram-Bot-Api-Secret-Token` header. URL query tokens are rejected so shared secrets do not leak through URL logs, browser history, or referrers.
- Use a dedicated thread in Forge by setting `Default thread ID`, or per platform chat/channel threads are auto-created on first message.
