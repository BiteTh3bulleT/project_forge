import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { api, type DiscordGatewayStatusResponse, type TelegramStatusResponse } from "../lib/api";
import { useUiStore } from "../stores/uiStore";

export function SettingsPage() {
  const [extensionsCsv, setExtensionsCsv] = useState("");
  const [theme, setTheme] = useState<"dark" | "light">("dark");
  const [ollamaBaseUrl, setOllamaBaseUrl] = useState("http://127.0.0.1:11434");
  const [ollamaModel, setOllamaModel] = useState("");
  const [embeddingProvider, setEmbeddingProvider] = useState("local_hash");
  const [embeddingModel, setEmbeddingModel] = useState("");
  const [embeddingDims, setEmbeddingDims] = useState("128");
  const [retrievalWeightKeyword, setRetrievalWeightKeyword] = useState("0.45");
  const [retrievalWeightSemantic, setRetrievalWeightSemantic] = useState("0.55");
  const [chatPersonalityPrompt, setChatPersonalityPrompt] = useState("");
  const [chatPromptDefault, setChatPromptDefault] = useState("");
  const [remoteAccessEnabled, setRemoteAccessEnabled] = useState(false);
  const [remoteAccessToken, setRemoteAccessToken] = useState("");
  const [remoteCrossChatContext, setRemoteCrossChatContext] = useState(false);
  const [remoteDefaultThreadId, setRemoteDefaultThreadId] = useState("");
  const [telegramBotToken, setTelegramBotToken] = useState("");
  const [telegramDefaultChatId, setTelegramDefaultChatId] = useState("");
  const [discordBotToken, setDiscordBotToken] = useState("");
  const [discordDefaultChannelId, setDiscordDefaultChannelId] = useState("");
  const [discordWebhookUrl, setDiscordWebhookUrl] = useState("");
  const [discordCrossChatContext, setDiscordCrossChatContext] = useState(false);
  const [remoteProbeMessage, setRemoteProbeMessage] = useState("FORGE remote ingress smoke test");
  const [remoteProbeTelegramChatId, setRemoteProbeTelegramChatId] = useState("");
  const [remoteProbeDiscordChannelId, setRemoteProbeDiscordChannelId] = useState("");
  const [remoteProbeStatus, setRemoteProbeStatus] = useState<string | null>(null);
  const [remoteProbeBusy, setRemoteProbeBusy] = useState<"telegram" | "discord" | null>(null);
  const [telegramStatus, setTelegramStatus] = useState<TelegramStatusResponse | null>(null);
  const [telegramStatusErr, setTelegramStatusErr] = useState<string | null>(null);
  const [discordStatus, setDiscordStatus] = useState<DiscordGatewayStatusResponse | null>(null);
  const [discordStatusErr, setDiscordStatusErr] = useState<string | null>(null);
  const [remoteStatusBusy, setRemoteStatusBusy] = useState(false);
  const [meta, setMeta] = useState<{ dataDir: string; dbPath: string; workspaceDir: string } | null>(null);
  const [ollamaModels, setOllamaModels] = useState<string[]>([]);
  const [ollamaModelsError, setOllamaModelsError] = useState<string | null>(null);
  const [ollamaModelsLoading, setOllamaModelsLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const setStatus = useUiStore((s) => s.setStatusLine);

  async function loadOllamaModelsFromAdapters() {
    const adapters = await api.adapters.list();
    const ollamaAdapter = adapters.adapters.find((adapter) => adapter.id === "ollama");
    const models = Array.isArray(ollamaAdapter?.config?.models) ? ollamaAdapter.config.models : [];
    return { models: models, baseUrl: String(ollamaAdapter?.config?.baseUrl ?? ""), error: undefined };
  }

  async function loadOllamaModels(baseUrl?: string) {
    const targetBaseUrl = (baseUrl ?? ollamaBaseUrl) || "http://127.0.0.1:11434";
    setOllamaModelsLoading(true);
    setOllamaModelsError(null);
    try {
      let result;
      try {
        result = await api.settings.ollamaModels(targetBaseUrl);
      } catch (e) {
        const message = e instanceof Error ? e.message : String(e);
        const isNotFound =
          message.includes("404") ||
          message.toLowerCase().includes("not found") ||
          message.toLowerCase().includes("does not exist");
        if (isNotFound) {
          result = await loadOllamaModelsFromAdapters();
        } else {
          throw e;
        }
      }
      const list = Array.isArray(result.models) ? result.models : [];
      setOllamaModels(list);
      setOllamaModelsError(result.error ?? null);
    } catch (e) {
      setOllamaModels([]);
      setOllamaModelsError(e instanceof Error ? e.message : String(e));
    } finally {
      setOllamaModelsLoading(false);
    }
  }

  async function load() {
    try {
      const s = await api.settings.get();
      setExtensionsCsv(s.extensionsCsv);
      setTheme(s.theme === "light" ? "light" : "dark");
      setOllamaBaseUrl(s.ollamaBaseUrl || "http://127.0.0.1:11434");
      setOllamaModel(s.ollamaModel || "");
      setEmbeddingProvider(s.embeddingProvider || "local_hash");
      setEmbeddingModel(s.embeddingModel || "");
      setEmbeddingDims(s.embeddingDims || "128");
      setRetrievalWeightKeyword(s.retrievalWeightKeyword || "0.45");
      setRetrievalWeightSemantic(s.retrievalWeightSemantic || "0.55");
      setChatPersonalityPrompt(s.chatPersonalityPrompt || "");
      setChatPromptDefault(s.chatPromptDefault || "");
      setRemoteAccessEnabled(Boolean(s.remoteAccessEnabled));
      setRemoteAccessToken(s.remoteAccessToken || "");
      setRemoteCrossChatContext(Boolean(s.remoteCrossChatContext));
      setRemoteDefaultThreadId(s.remoteDefaultThreadId || "");
      setTelegramBotToken(s.telegramBotToken || "");
      setTelegramDefaultChatId(s.telegramDefaultChatId || "");
      setDiscordBotToken(s.discordBotToken || "");
      setDiscordDefaultChannelId(s.discordDefaultChannelId || "");
      setDiscordWebhookUrl(s.discordWebhookUrl || "");
      setDiscordCrossChatContext(Boolean(s.discordCrossChatContext));
      await loadOllamaModels(s.ollamaBaseUrl || "http://127.0.0.1:11434");
      const m = await api.meta();
      setMeta(m);
      await refreshRemoteStatuses();
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  async function refreshRemoteStatuses() {
    setRemoteStatusBusy(true);
    const [telegramRes, discordRes] = await Promise.allSettled([api.telegram.status(), api.discord.status()]);
    if (telegramRes.status === "fulfilled") {
      setTelegramStatus(telegramRes.value);
      setTelegramStatusErr(null);
    } else {
      setTelegramStatus(null);
      setTelegramStatusErr(telegramRes.reason instanceof Error ? telegramRes.reason.message : String(telegramRes.reason));
    }
    if (discordRes.status === "fulfilled") {
      setDiscordStatus(discordRes.value);
      setDiscordStatusErr(null);
    } else {
      setDiscordStatus(null);
      setDiscordStatusErr(discordRes.reason instanceof Error ? discordRes.reason.message : String(discordRes.reason));
    }
    setRemoteStatusBusy(false);
  }

  useEffect(() => {
    void load();
  }, []);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  useEffect(() => {
    if (!ollamaBaseUrl) {
      return;
    }
    const id = window.setTimeout(() => {
      void loadOllamaModels(ollamaBaseUrl);
    }, 300);
    return () => window.clearTimeout(id);
  }, [ollamaBaseUrl]);

  const normalizedOllamaModel = ollamaModel.trim();
  const selectedInDropdown = normalizedOllamaModel && ollamaModels.includes(normalizedOllamaModel);

  async function probeTelegramIngress() {
    setRemoteProbeStatus(null);
    setRemoteProbeBusy("telegram");
    try {
      const chatIdRaw = (remoteProbeTelegramChatId || telegramDefaultChatId).trim();
      const chatId = Number(chatIdRaw);
      if (!Number.isFinite(chatId) || chatId <= 0) {
        throw new Error("Provide a valid Telegram chat ID for probe.");
      }
      if (!remoteAccessToken.trim()) {
        throw new Error("Set a remote access token before probing remote ingress.");
      }
      await api.remote.telegram(
        {
          message: {
            message_id: Date.now(),
            text: remoteProbeMessage.trim() || "FORGE remote ingress smoke test",
            chat: { id: chatId },
            from: { id: 1 },
          },
        },
        remoteAccessToken,
      );
      setRemoteProbeStatus("Telegram ingress probe accepted by core.");
      setStatus("Telegram ingress probe accepted.");
    } catch (e) {
      setRemoteProbeStatus(e instanceof Error ? e.message : String(e));
    } finally {
      setRemoteProbeBusy(null);
    }
  }

  async function probeDiscordIngress() {
    setRemoteProbeStatus(null);
    setRemoteProbeBusy("discord");
    try {
      const channelId = (remoteProbeDiscordChannelId || discordDefaultChannelId).trim();
      if (!channelId) {
        throw new Error("Provide a Discord channel ID for probe.");
      }
      if (!remoteAccessToken.trim()) {
        throw new Error("Set a remote access token before probing remote ingress.");
      }
      await api.remote.discord(
        {
          id: String(Date.now()),
          channel_id: channelId,
          content: remoteProbeMessage.trim() || "FORGE remote ingress smoke test",
          author: { id: "local-probe" },
        },
        remoteAccessToken,
      );
      setRemoteProbeStatus("Discord ingress probe accepted by core.");
      setStatus("Discord ingress probe accepted.");
    } catch (e) {
      setRemoteProbeStatus(e instanceof Error ? e.message : String(e));
    } finally {
      setRemoteProbeBusy(null);
    }
  }

  const discordSnapshot =
    discordStatus && typeof discordStatus.status === "object" && discordStatus.status != null ? discordStatus.status : null;
  const discordReady = Boolean(discordStatus?.enabled && discordSnapshot?.connected);
  const telegramReady = Boolean(telegramStatus?.ready);
  const remoteOverview = {
    endpointsEnabled: remoteAccessEnabled,
    tokenConfigured: remoteAccessToken.trim().length > 0,
    sharedThreading: remoteCrossChatContext,
    telegramReady,
    discordReady,
  };

  return (
    <div className="space-y-6">
      <Panel title="Settings" subtitle="Local-only configuration for indexing, adapter defaults, and workspace controls." actions={<GhostButton onClick={() => void load()}>Reload</GhostButton>}>
        {err ? <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
      </Panel>

      <Panel title="Indexing" subtitle="Comma-separated extension allowlist used by source scans.">
        <label className="text-xs font-semibold tracking-wide text-forge-mist">Supported extensions</label>
        <textarea className="forge-input mt-2 min-h-[96px] font-mono text-xs" value={extensionsCsv} onChange={(e) => setExtensionsCsv(e.target.value)} />
        <div className="mt-3 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              await api.settings.patch({ extensionsCsv });
              setStatus("Extensions updated.");
            }}
          >
            Save extensions
          </PrimaryButton>
        </div>
      </Panel>

        <Panel title="Ollama" subtitle="Local adapter endpoint + model defaults used for reasoning jobs.">
        <div className="grid gap-3 md:grid-cols-2">
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Base URL</label>
            <input
              className="forge-input mt-1"
              value={ollamaBaseUrl}
              onChange={(e) => setOllamaBaseUrl(e.target.value)}
              placeholder="http://127.0.0.1:11434"
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Model</label>
            <select className="forge-input mt-1" value={selectedInDropdown ? normalizedOllamaModel : ""} onChange={(e) => setOllamaModel(e.target.value)}>
              <option value="">{ollamaModelsLoading ? "Checking available models…" : "Select model…"}</option>
              {selectedInDropdown ? null : normalizedOllamaModel ? <option value={normalizedOllamaModel}>{normalizedOllamaModel}</option> : null}
              {ollamaModels.map((model) => (
                <option key={model} value={model}>
                  {model}
                </option>
              ))}
            </select>
            {!ollamaModelsLoading && ollamaModelsError ? (
              <div className="mt-1 text-xs text-forge-ember">
                Ollama model list unavailable: {ollamaModelsError}
              </div>
            ) : null}
            {!ollamaModelsLoading && ollamaModels.length === 0 ? (
              <div className="mt-1 text-xs text-forge-ash">No models detected at this base URL.</div>
            ) : null}
          </div>
        </div>
        <div className="mt-3 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              await api.settings.patch({ ollamaBaseUrl, ollamaModel });
              setStatus("Ollama settings saved.");
            }}
          >
            Save Ollama settings
          </PrimaryButton>
        </div>
      </Panel>

      <Panel title="Remote Channels" subtitle="Telegram + Discord transport controls with health checks, probes, and scoped threading behavior.">
        <div className="grid gap-3 md:grid-cols-2">
          <label className="flex items-center gap-2 md:col-span-2">
            <input
              type="checkbox"
              checked={remoteAccessEnabled}
              onChange={(e) => setRemoteAccessEnabled(e.target.checked)}
            />
            <span className="text-xs font-semibold tracking-wide text-forge-mist">Enable remote access endpoints</span>
          </label>
          <label className="flex items-center gap-2 md:col-span-2">
            <input
              type="checkbox"
              checked={remoteCrossChatContext}
              onChange={(e) => setRemoteCrossChatContext(e.target.checked)}
            />
            <span className="text-xs font-semibold tracking-wide text-forge-mist">Cross-chat context for remote ingress (share one thread per platform)</span>
          </label>
          <div className="md:col-span-2 grid gap-2 rounded border border-white/10 bg-black/20 p-3 text-xs md:grid-cols-5">
            <RemoteStateChip label="Endpoints" ok={remoteOverview.endpointsEnabled} okText="on" offText="off" />
            <RemoteStateChip label="Token" ok={remoteOverview.tokenConfigured} okText="set" offText="missing" />
            <RemoteStateChip label="Shared Context" ok={remoteOverview.sharedThreading} okText="on" offText="off" />
            <RemoteStateChip label="Telegram" ok={remoteOverview.telegramReady} okText="ready" offText="needs setup" />
            <RemoteStateChip label="Discord" ok={remoteOverview.discordReady} okText="ready" offText="needs setup" />
          </div>
          <div className="md:col-span-2">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Remote token</label>
            <input
              className="forge-input mt-1"
              value={remoteAccessToken}
              onChange={(e) => setRemoteAccessToken(e.target.value)}
              placeholder="Share with Telegram/Discord webhook callers"
              type="password"
            />
          </div>
          <div className="md:col-span-2">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Default thread ID (optional)</label>
            <input
              className="forge-input mt-1"
              value={remoteDefaultThreadId}
              onChange={(e) => setRemoteDefaultThreadId(e.target.value)}
              placeholder="Fallback thread for all remote conversations"
            />
          </div>
          <div className="rounded border border-white/10 bg-black/20 p-3">
            <div className="flex items-center justify-between">
              <div className="text-xs font-semibold uppercase tracking-[0.14em] text-forge-mist">Telegram</div>
              <StatusDot ok={telegramReady} label={telegramReady ? "Ready" : "Needs setup"} />
            </div>
            <div className="mt-3 space-y-3">
              <div>
                <label className="text-xs font-semibold tracking-wide text-forge-mist">Bot token</label>
                <input
                  className="forge-input mt-1"
                  value={telegramBotToken}
                  onChange={(e) => setTelegramBotToken(e.target.value)}
                  placeholder="Bot token from @BotFather"
                  type="password"
                />
              </div>
              <div>
                <label className="text-xs font-semibold tracking-wide text-forge-mist">Default chat ID</label>
                <input
                  className="forge-input mt-1"
                  value={telegramDefaultChatId}
                  onChange={(e) => setTelegramDefaultChatId(e.target.value)}
                  placeholder="Fallback chat id when payload lacks chat id"
                />
              </div>
              {telegramStatusErr ? (
                <div className="rounded border border-forge-ember/25 bg-forge-ember/10 px-2 py-1 text-[11px] text-forge-ash">
                  {telegramStatusErr}
                </div>
              ) : null}
              {telegramStatus ? (
                <div className="grid gap-1 text-[11px] text-forge-mist">
                  <StatusRow label="Remote endpoints" value={telegramStatus.remoteAccessEnabled ? "enabled" : "disabled"} />
                  <StatusRow label="Token configured" value={telegramStatus.tokenConfigured ? "yes" : "no"} />
                  <StatusRow label="Cross-chat context" value={telegramStatus.crossChatContext ? "on" : "off"} />
                  <StatusRow label="Default chat" value={telegramStatus.defaultChatId || "—"} />
                  {telegramStatus.bot ? <StatusRow label="Bot identity" value={`@${telegramStatus.bot.username} (${telegramStatus.bot.id})`} /> : null}
                  {telegramStatus.webhook?.url ? <StatusRow label="Webhook" value={telegramStatus.webhook.url} /> : null}
                  {telegramStatus.webhookError ? <StatusRow label="Webhook error" value={telegramStatus.webhookError} tone="warn" /> : null}
                  {telegramStatus.reason ? <StatusRow label="Reason" value={telegramStatus.reason} tone="warn" /> : null}
                </div>
              ) : (
                <div className="text-[11px] text-forge-mist">No Telegram status yet.</div>
              )}
            </div>
          </div>
          <div className="rounded border border-white/10 bg-black/20 p-3">
            <div className="flex items-center justify-between">
              <div className="text-xs font-semibold uppercase tracking-[0.14em] text-forge-mist">Discord</div>
              <StatusDot ok={discordReady} label={discordReady ? "Ready" : "Needs setup"} />
            </div>
            <div className="mt-3 space-y-3">
              <div>
                <label className="text-xs font-semibold tracking-wide text-forge-mist">Bot token</label>
                <input
                  className="forge-input mt-1"
                  value={discordBotToken}
                  onChange={(e) => setDiscordBotToken(e.target.value)}
                  placeholder="Bot token for gateway + outbound replies"
                  type="password"
                />
              </div>
              <div>
                <label className="text-xs font-semibold tracking-wide text-forge-mist">Default channel ID</label>
                <input
                  className="forge-input mt-1"
                  value={discordDefaultChannelId}
                  onChange={(e) => setDiscordDefaultChannelId(e.target.value)}
                  placeholder="Fallback channel when payload lacks channel_id"
                />
              </div>
              <div>
                <label className="text-xs font-semibold tracking-wide text-forge-mist">Webhook URL (optional)</label>
                <input
                  className="forge-input mt-1"
                  value={discordWebhookUrl}
                  onChange={(e) => setDiscordWebhookUrl(e.target.value)}
                  placeholder="Webhook URL for channel-less incoming messages"
                />
              </div>
              <label className="flex items-center gap-2 text-xs font-semibold tracking-wide text-forge-mist">
                <input type="checkbox" checked={discordCrossChatContext} onChange={(e) => setDiscordCrossChatContext(e.target.checked)} />
                Discord gateway cross-chat context
              </label>
              {discordStatusErr ? (
                <div className="rounded border border-forge-ember/25 bg-forge-ember/10 px-2 py-1 text-[11px] text-forge-ash">
                  {discordStatusErr}
                </div>
              ) : null}
              {discordStatus ? (
                typeof discordStatus.status === "string" ? (
                  <div className="grid gap-1 text-[11px] text-forge-mist">
                    <StatusRow label="Gateway" value="disabled" tone="warn" />
                    <StatusRow label="Reason" value={discordStatus.reason || "discord gateway not started"} tone="warn" />
                  </div>
                ) : (
                  <div className="grid gap-1 text-[11px] text-forge-mist">
                    <StatusRow label="Gateway" value={discordStatus.enabled ? "enabled" : "disabled"} />
                    <StatusRow label="Connected" value={discordStatus.status.connected ? "yes" : "no"} />
                    <StatusRow label="Guild" value={discordStatus.status.guildId || "any"} />
                    <StatusRow label="Prefix" value={discordStatus.status.commandPrefix || "!forge"} />
                    <StatusRow label="Slash commands" value={discordStatus.status.enableSlash ? "on" : "off"} />
                    <StatusRow label="Text commands" value={discordStatus.status.enableText ? "on" : "off"} />
                    <StatusRow label="Passive listen" value={discordStatus.status.enablePassive ? "on" : "off"} />
                    <StatusRow label="Cross-chat context" value={discordStatus.status.crossChatContext ? "on" : "off"} />
                    <StatusRow label="Inbound / Outbound" value={`${discordStatus.status.inboundCount ?? 0} / ${discordStatus.status.outboundCount ?? 0}`} />
                    <StatusRow label="Last inbound" value={discordStatus.status.lastInboundAtMs ? new Date(discordStatus.status.lastInboundAtMs).toLocaleString() : "—"} />
                    <StatusRow label="Last outbound" value={discordStatus.status.lastOutboundAtMs ? new Date(discordStatus.status.lastOutboundAtMs).toLocaleString() : "—"} />
                    {discordStatus.status.lastError ? <StatusRow label="Last error" value={discordStatus.status.lastError} tone="warn" /> : null}
                  </div>
                )
              ) : (
                <div className="text-[11px] text-forge-mist">No Discord status yet.</div>
              )}
            </div>
          </div>
        </div>
        <div className="mt-3 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              await api.settings.patch({
                remoteAccessEnabled,
                remoteAccessToken,
                remoteCrossChatContext,
                remoteDefaultThreadId,
                telegramBotToken,
                telegramDefaultChatId,
                discordBotToken,
                discordDefaultChannelId,
                discordWebhookUrl,
                discordCrossChatContext,
              });
              await refreshRemoteStatuses();
              setStatus("Remote access settings saved.");
            }}
          >
            Save remote access
          </PrimaryButton>
          <GhostButton onClick={() => void refreshRemoteStatuses()} disabled={remoteStatusBusy}>
            {remoteStatusBusy ? "Refreshing channel status..." : "Refresh Telegram + Discord status"}
          </GhostButton>
        </div>
        <div className="mt-4 grid gap-3 border-t border-white/10 pt-4 md:grid-cols-2">
          <div className="md:col-span-2">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Remote probe message</label>
            <input
              className="forge-input mt-1"
              value={remoteProbeMessage}
              onChange={(e) => setRemoteProbeMessage(e.target.value)}
              placeholder="Probe payload text"
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Telegram probe chat ID (optional override)</label>
            <input
              className="forge-input mt-1"
              value={remoteProbeTelegramChatId}
              onChange={(e) => setRemoteProbeTelegramChatId(e.target.value)}
              placeholder={telegramDefaultChatId || "Uses Telegram default chat id"}
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Discord probe channel ID (optional override)</label>
            <input
              className="forge-input mt-1"
              value={remoteProbeDiscordChannelId}
              onChange={(e) => setRemoteProbeDiscordChannelId(e.target.value)}
              placeholder={discordDefaultChannelId || "Uses Discord default channel id"}
            />
          </div>
          <div className="md:col-span-2 flex flex-wrap gap-2">
            <GhostButton onClick={() => void probeTelegramIngress()} disabled={remoteProbeBusy !== null}>
              {remoteProbeBusy === "telegram" ? "Probing Telegram…" : "Probe Telegram ingress"}
            </GhostButton>
            <GhostButton onClick={() => void probeDiscordIngress()} disabled={remoteProbeBusy !== null}>
              {remoteProbeBusy === "discord" ? "Probing Discord…" : "Probe Discord ingress"}
            </GhostButton>
          </div>
          {remoteProbeStatus ? (
            <div className="md:col-span-2 text-xs text-forge-mist">{remoteProbeStatus}</div>
          ) : null}
        </div>
      </Panel>

      <Panel title="Retrieval + Embeddings" subtitle="Hybrid ranking weights and semantic embedding provider configuration.">
        <div className="grid gap-3 md:grid-cols-2">
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Embedding provider</label>
            <select className="forge-input mt-1" value={embeddingProvider} onChange={(e) => setEmbeddingProvider(e.target.value)}>
              <option value="local_hash">local_hash</option>
              <option value="ollama">ollama</option>
            </select>
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Embedding model</label>
            <input className="forge-input mt-1" value={embeddingModel} onChange={(e) => setEmbeddingModel(e.target.value)} placeholder="optional for local_hash" />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Embedding dims</label>
            <input className="forge-input mt-1" value={embeddingDims} onChange={(e) => setEmbeddingDims(e.target.value)} />
          </div>
          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <label className="text-xs font-semibold tracking-wide text-forge-mist">Keyword weight</label>
              <input className="forge-input mt-1" value={retrievalWeightKeyword} onChange={(e) => setRetrievalWeightKeyword(e.target.value)} />
            </div>
            <div>
              <label className="text-xs font-semibold tracking-wide text-forge-mist">Semantic weight</label>
              <input className="forge-input mt-1" value={retrievalWeightSemantic} onChange={(e) => setRetrievalWeightSemantic(e.target.value)} />
            </div>
          </div>
        </div>
        <div className="mt-3 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              await api.settings.patch({
                embeddingProvider,
                embeddingModel,
                embeddingDims,
                retrievalWeightKeyword,
                retrievalWeightSemantic,
              });
              setStatus("Retrieval + embedding settings saved.");
            }}
          >
            Save retrieval settings
          </PrimaryButton>
        </div>
      </Panel>

      <Panel title="Chat Personality Prompt" subtitle="Live system prompt for chat replies. Changes apply on the next assistant response.">
        <label className="text-xs font-semibold tracking-wide text-forge-mist">System prompt</label>
        <textarea
          className="forge-input mt-2 min-h-[280px] font-mono text-xs leading-relaxed"
          value={chatPersonalityPrompt}
          onChange={(e) => setChatPersonalityPrompt(e.target.value)}
        />
        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton
            onClick={async () => {
              await api.settings.patch({ chatPersonalityPrompt });
              setStatus("Chat personality prompt saved.");
            }}
          >
            Save chat prompt
          </PrimaryButton>
          <GhostButton
            onClick={() => {
              setChatPersonalityPrompt(chatPromptDefault);
              setStatus("Restored default chat prompt in editor.");
            }}
          >
            Reset editor to default
          </GhostButton>
          <GhostButton
            onClick={async () => {
              await api.settings.patch({ chatPersonalityPrompt: chatPromptDefault });
              setChatPersonalityPrompt(chatPromptDefault);
              setStatus("Chat personality prompt reset to default.");
            }}
          >
            Save default prompt
          </GhostButton>
        </div>
      </Panel>

      <Panel title="Theme" subtitle="Dark is the intended operator default.">
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className={theme === "dark" ? "forge-btn forge-btn--primary" : "forge-btn forge-btn--ghost"}
            onClick={() => setTheme("dark")}
          >
            Dark
          </button>
          <button
            type="button"
            className={theme === "light" ? "forge-btn forge-btn--primary" : "forge-btn forge-btn--ghost"}
            onClick={() => setTheme("light")}
          >
            Light
          </button>
          <PrimaryButton
            onClick={async () => {
              await api.settings.patch({ theme });
              setStatus("Theme preference saved.");
            }}
          >
            Save theme
          </PrimaryButton>
        </div>
      </Panel>

      <Panel title="Workspace" subtitle="Local paths used by FORGE core for persistence and context generation.">
        {meta ? (
          <div className="space-y-2 text-sm text-forge-mist">
            <div>
              <span className="text-forge-ash">FORGE_DATA_DIR:</span> <span className="font-mono text-xs text-forge-ash">{meta.dataDir}</span>
            </div>
            <div>
              <span className="text-forge-ash">Database:</span> <span className="font-mono text-xs text-forge-ash">{meta.dbPath}</span>
            </div>
            <div>
              <span className="text-forge-ash">Workspace:</span> <span className="font-mono text-xs text-forge-ash">{meta.workspaceDir}</span>
            </div>
          </div>
        ) : (
          <div className="text-sm text-forge-mist">Core offline — metadata unavailable.</div>
        )}
      </Panel>
    </div>
  );
}

function RemoteStateChip(props: { label: string; ok: boolean; okText: string; offText: string }) {
  return (
    <div className="rounded border border-white/10 bg-black/25 px-3 py-2 text-[11px] text-forge-mist">
      <div className="uppercase tracking-[0.14em] text-forge-mist/70">{props.label}</div>
      <div className={props.ok ? "mt-1 font-semibold text-forge-ash" : "mt-1 font-semibold text-forge-emberSoft"}>
        {props.ok ? props.okText : props.offText}
      </div>
    </div>
  );
}

function StatusDot(props: { ok: boolean; label: string }) {
  return (
    <div className="inline-flex items-center gap-2 text-[11px]">
      <span className={props.ok ? "h-2 w-2 rounded-full bg-forge-electric" : "h-2 w-2 rounded-full bg-forge-emberSoft"} />
      <span className={props.ok ? "text-forge-ash" : "text-forge-emberSoft"}>{props.label}</span>
    </div>
  );
}

function StatusRow(props: { label: string; value: string; tone?: "normal" | "warn" }) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-white/5 pb-1">
      <span className="text-forge-mist/75">{props.label}</span>
      <span className={props.tone === "warn" ? "text-right text-forge-emberSoft" : "text-right text-forge-ash"}>{props.value}</span>
    </div>
  );
}
