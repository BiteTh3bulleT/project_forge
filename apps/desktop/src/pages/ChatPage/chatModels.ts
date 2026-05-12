import type { ModelRuntimeModel } from "../../lib/api";

const CHAT_MODEL_SELECTION_CACHE_KEY = "forge.chat.requestedModelId.v1";

export function readCachedChatModelSelection(): string {
  if (typeof window === "undefined") return "";
  try {
    const raw = window.localStorage.getItem(CHAT_MODEL_SELECTION_CACHE_KEY);
    return typeof raw === "string" ? raw.trim() : "";
  } catch {
    return "";
  }
}

export function writeCachedChatModelSelection(value: string) {
  if (typeof window === "undefined") return;
  try {
    const trimmed = value.trim();
    if (!trimmed) {
      window.localStorage.removeItem(CHAT_MODEL_SELECTION_CACHE_KEY);
      return;
    }
    window.localStorage.setItem(CHAT_MODEL_SELECTION_CACHE_KEY, trimmed);
  } catch {
    return;
  }
}

export function supportsChatCapability(model: ModelRuntimeModel): boolean {
  const caps = Array.isArray(model.capabilities) ? model.capabilities : [];
  if (caps.length === 0) return true;
  return caps.some((cap) => {
    const normalized = String(cap).trim().toLowerCase();
    return normalized === "chat" || normalized === "completion";
  });
}

export function usableChatStatus(model: ModelRuntimeModel): boolean {
  const status = String(model.status ?? "")
    .trim()
    .toLowerCase();
  return (
    status !== "disabled" &&
    status !== "archived" &&
    status !== "unavailable" &&
    status !== "error"
  );
}

function chatModelStatusRank(status: string | undefined): number {
  switch (String(status ?? "").trim().toLowerCase()) {
    case "loaded":
      return 5;
    case "available":
      return 4;
    case "verified":
      return 3;
    case "imported":
      return 2;
    case "loading":
    case "unloading":
      return 1;
    default:
      return 0;
  }
}

export function preferredAutoChatModel(
  models: ModelRuntimeModel[],
): ModelRuntimeModel | null {
  const candidates = models
    .filter((model) => supportsChatCapability(model) && usableChatStatus(model))
    .slice()
    .sort((a, b) => {
      const rank = chatModelStatusRank(b.status) - chatModelStatusRank(a.status);
      if (rank !== 0) return rank;
      return a.id.localeCompare(b.id);
    });
  return candidates[0] ?? null;
}

export function describeChatModel(
  modelId: string,
  models: ModelRuntimeModel[],
): string {
  const id = modelId.trim();
  if (!id) {
    const preferred = preferredAutoChatModel(models);
    if (preferred) {
      const label = preferred.displayName?.trim() || preferred.id;
      const status = String(preferred.status ?? "").trim();
      return status.toLowerCase() === "loaded"
        ? `Auto will use active loaded model ${label}.`
        : `Auto will use ${label} unless the runtime selects a better route.`;
    }
    return "Auto routing uses the runtime default or configured adapter fallback.";
  }
  const selected = models.find((model) => model.id === id);
  if (!selected)
    return `Pinned to saved model ${id}. It is not present in the current runtime list.`;
  const label = selected.displayName?.trim() || selected.id;
  const backend = selected.backend?.trim();
  return backend
    ? `Pinned to ${label} on backend ${backend}.`
    : `Pinned to ${label}.`;
}

export function describeAssistantMode(props: {
  requestAssistant: boolean;
  assistantDryRun: boolean;
  streamAssistant: boolean;
  blockingAssistant: boolean;
}) {
  if (!props.requestAssistant) return "Message only";
  if (props.assistantDryRun) return "Dry-run only";
  if (props.blockingAssistant) return "Blocking response";
  if (props.streamAssistant) return "Streaming response";
  return "Async response";
}
