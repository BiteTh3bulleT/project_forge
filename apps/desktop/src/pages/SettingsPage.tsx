import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { api } from "../lib/api";
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
  const [remoteDefaultThreadId, setRemoteDefaultThreadId] = useState("");
  const [telegramBotToken, setTelegramBotToken] = useState("");
  const [telegramDefaultChatId, setTelegramDefaultChatId] = useState("");
  const [discordBotToken, setDiscordBotToken] = useState("");
  const [discordDefaultChannelId, setDiscordDefaultChannelId] = useState("");
  const [discordWebhookUrl, setDiscordWebhookUrl] = useState("");
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
      setRemoteDefaultThreadId(s.remoteDefaultThreadId || "");
      setTelegramBotToken(s.telegramBotToken || "");
      setTelegramDefaultChatId(s.telegramDefaultChatId || "");
      setDiscordBotToken(s.discordBotToken || "");
      setDiscordDefaultChannelId(s.discordDefaultChannelId || "");
      setDiscordWebhookUrl(s.discordWebhookUrl || "");
      await loadOllamaModels(s.ollamaBaseUrl || "http://127.0.0.1:11434");
      const m = await api.meta();
      setMeta(m);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
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

      <Panel title="Remote Access" subtitle="Configure Telegram or Discord ingress for Forge remote replies.">
        <div className="grid gap-3 md:grid-cols-2">
          <label className="flex items-center gap-2 md:col-span-2">
            <input
              type="checkbox"
              checked={remoteAccessEnabled}
              onChange={(e) => setRemoteAccessEnabled(e.target.checked)}
            />
            <span className="text-xs font-semibold tracking-wide text-forge-mist">Enable remote access endpoints</span>
          </label>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Remote token</label>
            <input
              className="forge-input mt-1"
              value={remoteAccessToken}
              onChange={(e) => setRemoteAccessToken(e.target.value)}
              placeholder="Share with Telegram/Discord webhook callers"
              type="password"
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Default thread ID (optional)</label>
            <input
              className="forge-input mt-1"
              value={remoteDefaultThreadId}
              onChange={(e) => setRemoteDefaultThreadId(e.target.value)}
              placeholder="Fallback thread for all remote conversations"
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Telegram bot token</label>
            <input
              className="forge-input mt-1"
              value={telegramBotToken}
              onChange={(e) => setTelegramBotToken(e.target.value)}
              placeholder="Bot token from @BotFather"
              type="password"
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Telegram default chat ID</label>
            <input
              className="forge-input mt-1"
              value={telegramDefaultChatId}
              onChange={(e) => setTelegramDefaultChatId(e.target.value)}
              placeholder="Fallback chat id when Telegram payload lacks chat id"
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Discord bot token</label>
            <input
              className="forge-input mt-1"
              value={discordBotToken}
              onChange={(e) => setDiscordBotToken(e.target.value)}
              placeholder="Bot token for Discord REST replies"
              type="password"
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Discord default channel ID</label>
            <input
              className="forge-input mt-1"
              value={discordDefaultChannelId}
              onChange={(e) => setDiscordDefaultChannelId(e.target.value)}
              placeholder="Fallback channel when incoming payload lacks channel_id"
            />
          </div>
          <div className="md:col-span-2">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Discord webhook URL (optional)</label>
            <input
              className="forge-input mt-1"
              value={discordWebhookUrl}
              onChange={(e) => setDiscordWebhookUrl(e.target.value)}
              placeholder="Webhook URL for channel-less incoming messages"
            />
          </div>
        </div>
        <div className="mt-3 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              await api.settings.patch({
                remoteAccessEnabled,
                remoteAccessToken,
                remoteDefaultThreadId,
                telegramBotToken,
                telegramDefaultChatId,
                discordBotToken,
                discordDefaultChannelId,
                discordWebhookUrl,
              });
              setStatus("Remote access settings saved.");
            }}
          >
            Save remote access
          </PrimaryButton>
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
