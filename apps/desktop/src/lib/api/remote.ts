import { j } from "./client";
import type {
  DiscordGatewayStatusResponse,
  RemoteDiscordPayload,
  RemoteTelegramPayload,
  TelegramStatusResponse,
} from "./types";

export const remoteApi = {
  telegram: (body: RemoteTelegramPayload, token?: string) =>
    j<{ ok: boolean }>("/api/remote/telegram", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...(token?.trim() ? { "X-Forge-Remote-Token": token.trim() } : {}),
      },
      body: JSON.stringify(body),
    }),
  discord: (body: RemoteDiscordPayload, token?: string) =>
    j<{ ok: boolean }>("/api/remote/discord", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...(token?.trim() ? { "X-Forge-Remote-Token": token.trim() } : {}),
      },
      body: JSON.stringify(body),
    }),
};

export const telegramApi = {
  status: () => j<TelegramStatusResponse>("/api/telegram/status"),
};

export const discordApi = {
  status: () => j<DiscordGatewayStatusResponse>("/api/discord/status"),
};
