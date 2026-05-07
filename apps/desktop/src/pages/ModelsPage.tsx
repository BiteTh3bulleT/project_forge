import { GhostButton, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";

import {
  HumanDataView,
  summarizeHumanValue,
} from "../components/HumanDataView";
import {
  api,
  type ModelRuntimeBackendStatus,
  type ModelRuntimeCompatibility,
  type ModelRuntimeHealth,
  type ModelRuntimeLoadedStatus,
  type ModelRuntimeModel,
  type ModelRuntimeQueueStatus,
  type ModelRuntimeUsageSummary,
} from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

const CONTROL_ACTOR = "operator";
const CONTROL_SOURCE = "desktop";
const MODEL_MANAGEMENT_CAPABILITY = "model.management";
const CHAT_MODEL_SELECTION_CACHE_KEY = "forge.chat.requestedModelId.v1";

function cx(...parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(" ");
}

function normalizeStatus(status?: string) {
  return (status ?? "unknown").trim().toLowerCase();
}

function badgeClass(status?: string) {
  switch (normalizeStatus(status)) {
    case "loaded":
    case "verified":
    case "available":
    case "ok":
    case "healthy":
      return "forge-ops-status forge-ops-status--ok";
    case "loading":
    case "unloading":
    case "imported":
      return "forge-ops-status forge-ops-status--warn";
    case "disabled":
    case "archived":
      return "forge-ops-status forge-ops-status--muted";
    case "error":
    case "unavailable":
      return "forge-ops-status forge-ops-status--bad";
    default:
      return "forge-ops-status forge-ops-status--muted";
  }
}

function summarizeList(values?: string[]) {
  if (!Array.isArray(values) || values.length === 0) return "none";
  return values.join(", ");
}

function summarizeValue(value: unknown) {
  const summary = summarizeHumanValue(value);
  return summary === "None" ? "—" : summary;
}

function EmptyState(props: {
  title: string;
  detail: string;
  tone?: "muted" | "warn" | "bad";
}) {
  const toneClass =
    props.tone === "bad"
      ? "border-forge-ember/30 bg-forge-ember/10"
      : props.tone === "warn"
        ? "border-forge-amber/30 bg-forge-amber/10"
        : "border-forge-platinum/10 bg-black/20";
  return (
    <div className={cx("rounded border border-dashed p-4", toneClass)}>
      <div className="text-sm font-semibold text-forge-ash">{props.title}</div>
      <div className="mt-1 text-xs leading-5 text-forge-mist/75">
        {props.detail}
      </div>
    </div>
  );
}

function supportsChatCapability(model: ModelRuntimeModel) {
  const capabilities = Array.isArray(model.capabilities)
    ? model.capabilities
    : [];
  if (capabilities.length === 0) return true;
  return capabilities.some((capability) => {
    const normalized = String(capability).trim().toLowerCase();
    return normalized === "chat" || normalized === "completion";
  });
}

function usableChatStatus(model: ModelRuntimeModel) {
  const status = normalizeStatus(model.status);
  return (
    status !== "disabled" &&
    status !== "archived" &&
    status !== "unavailable" &&
    status !== "error"
  );
}

function readCachedChatModelSelection() {
  if (typeof window === "undefined") return "";
  try {
    return (
      window.localStorage.getItem(CHAT_MODEL_SELECTION_CACHE_KEY) ?? ""
    ).trim();
  } catch {
    return "";
  }
}

function writeCachedChatModelSelection(value: string) {
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

function modelManagementRequest(metadata?: Record<string, unknown>) {
  return {
    actor: CONTROL_ACTOR,
    source: CONTROL_SOURCE,
    capabilityId: MODEL_MANAGEMENT_CAPABILITY,
    metadata,
  };
}

type ModelGovernanceDecision = {
  requiresApproval?: boolean;
  approved?: boolean;
  dryRun?: boolean;
  approvalRequestId?: number;
  operation?: string;
  reason?: string;
};

function modelGovernanceMessage(payload: unknown, label: string) {
  if (!payload || typeof payload !== "object") return null;
  const governance = (payload as { governance?: unknown }).governance;
  if (!governance || typeof governance !== "object") return null;
  const decision = governance as ModelGovernanceDecision;
  const approvalId =
    typeof decision.approvalRequestId === "number"
      ? ` #${decision.approvalRequestId}`
      : "";
  if (decision.requiresApproval && !decision.approved) {
    return `${label} requires approval${approvalId}.`;
  }
  if (decision.dryRun) {
    return `${label} dry run completed.`;
  }
  return null;
}

function emptyModelRuntimeUsage(): ModelRuntimeUsageSummary {
  return {
    registered: 0,
    imported: 0,
    verified: 0,
    available: 0,
    disabled: 0,
    archived: 0,
    loaded: 0,
    queueDepth: 0,
    running: 0,
    completed: 0,
    backends: {},
  };
}

export function ModelsPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const uiMode = useUiStore((s) => s.uiMode);
  const [searchParams, setSearchParams] = useSearchParams();
  const [models, setModels] = useState<ModelRuntimeModel[]>([]);
  const [selectedModelId, setSelectedModelId] = useState("");
  const [selectedModel, setSelectedModel] = useState<ModelRuntimeModel | null>(
    null,
  );
  const [compatibility, setCompatibility] =
    useState<ModelRuntimeCompatibility | null>(null);
  const [health, setHealth] = useState<ModelRuntimeHealth | null>(null);
  const [queue, setQueue] = useState<ModelRuntimeQueueStatus | null>(null);
  const [loaded, setLoaded] = useState<ModelRuntimeLoadedStatus | null>(null);
  const [usage, setUsage] = useState<ModelRuntimeUsageSummary | null>(null);
  const [backends, setBackends] = useState<ModelRuntimeBackendStatus[]>([]);
  const [runtimeAvailable, setRuntimeAvailable] = useState(true);
  const [importPath, setImportPath] = useState("");
  const [importDisplayName, setImportDisplayName] = useState("");
  const [importFamily, setImportFamily] = useState("");
  const [importBackend, setImportBackend] = useState("");
  const [importCapabilities, setImportCapabilities] =
    useState("chat,completion");
  const [importPreferred, setImportPreferred] = useState(false);
  const [loading, setLoading] = useState(false);
  const [importBusy, setImportBusy] = useState(false);
  const [scanBusy, setScanBusy] = useState(false);
  const [actionBusy, setActionBusy] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [lastUpdatedAt, setLastUpdatedAt] = useState<number>(Date.now());
  const [chatSelectedModelId, setChatSelectedModelId] = useState<string>(() =>
    readCachedChatModelSelection(),
  );
  const [runtimeGpuEnabled, setRuntimeGpuEnabled] = useState(false);
  const [runtimeAllowOllamaCloudModels, setRuntimeAllowOllamaCloudModels] =
    useState(false);
  const [runtimeEffectiveGpuEnabled, setRuntimeEffectiveGpuEnabled] =
    useState(false);
  const [runtimePolicyBusy, setRuntimePolicyBusy] = useState(false);

  const selectedLoadedRecord = useMemo(
    () =>
      loaded?.models.find((item) => item.modelId === selectedModelId) ?? null,
    [loaded?.models, selectedModelId],
  );

  const modelCounts = useMemo(() => {
    return models.reduce<Record<string, number>>((acc, model) => {
      const key = normalizeStatus(model.status);
      acc[key] = (acc[key] ?? 0) + 1;
      return acc;
    }, {});
  }, [models]);

  const chatSelectableModels = useMemo(
    () =>
      models.filter(
        (model) => supportsChatCapability(model) && usableChatStatus(model),
      ),
    [models],
  );

  const chatSelectionKnown = useMemo(
    () =>
      !chatSelectedModelId ||
      models.some((model) => model.id === chatSelectedModelId),
    [models, chatSelectedModelId],
  );

  const selectedRegistryModel = useMemo(
    () => models.find((model) => model.id === selectedModelId) ?? null,
    [models, selectedModelId],
  );

  const selectedModelDetail =
    selectedModel?.id === selectedModelId ? selectedModel : null;
  const selectedModelSummary = selectedModelDetail ?? selectedRegistryModel;
  const selectedCompatibility = selectedModelDetail ? compatibility : null;

  async function refreshOverview(
    preserveSelection = true,
    preferredSelection?: string,
  ) {
    setLoading(true);
    try {
      const settings = await api.settings.get();
      setRuntimeGpuEnabled(Boolean(settings.runtimeControls?.gpuEnabled));
      setRuntimeAllowOllamaCloudModels(
        Boolean(settings.runtimeControls?.allowOllamaCloudModels),
      );
      setRuntimeEffectiveGpuEnabled(
        Boolean(settings.runtimeControls?.effectiveGpuEnabled),
      );
      const coreHealth = await api.health();
      const available = coreHealth.modelRuntime?.available === true;
      setRuntimeAvailable(available);
      if (!available) {
        setModels([]);
        setSelectedModelId("");
        setSelectedModel(null);
        setCompatibility(null);
        setHealth({
          ok: false,
          status: coreHealth.modelRuntime?.status || "unavailable",
          backend: "disabled",
        });
        setQueue({ depth: 0, scheduler: "disabled" });
        setLoaded({ count: 0, models: [] });
        setUsage(emptyModelRuntimeUsage());
        setBackends([]);
        setLastUpdatedAt(Date.now());
        setErr(null);
        return;
      }
      const [modelsRes, healthRes, queueRes, loadedRes, usageRes, backendsRes] =
        await Promise.all([
          api.modelRuntime.list(),
          api.modelRuntime.health(),
          api.modelRuntime.queue(),
          api.modelRuntime.loaded(),
          api.modelRuntime.usage(),
          api.modelRuntime.backends(),
        ]);

      const nextModels = Array.isArray(modelsRes.models)
        ? modelsRes.models
        : [];
      setModels(nextModels);
      setHealth(healthRes.health);
      setQueue(queueRes.queue);
      setLoaded(loadedRes.loaded);
      setUsage(usageRes.usage);
      setBackends(
        Array.isArray(backendsRes.backends) ? backendsRes.backends : [],
      );

      const preferredId = preferredSelection?.trim() || "";
      const nextSelectedId =
        preferredId && nextModels.some((model) => model.id === preferredId)
          ? preferredId
          : preserveSelection &&
              nextModels.some((model) => model.id === selectedModelId)
            ? selectedModelId
            : (nextModels[0]?.id ?? "");
      setSelectedModelId(nextSelectedId);
      setLastUpdatedAt(Date.now());
      setErr(null);
    } catch (error) {
      setErr(error instanceof Error ? error.message : String(error));
    } finally {
      setLoading(false);
    }
  }

  async function saveRuntimePolicy(next?: {
    gpuEnabled?: boolean;
    allowOllamaCloudModels?: boolean;
  }) {
    const gpuEnabled = next?.gpuEnabled ?? runtimeGpuEnabled;
    const allowOllamaCloudModels =
      next?.allowOllamaCloudModels ?? runtimeAllowOllamaCloudModels;
    setRuntimePolicyBusy(true);
    try {
      const updated = await api.settings.patch({
        runtimeControls: {
          gpuEnabled,
          allowOllamaCloudModels,
        },
      });
      setRuntimeGpuEnabled(Boolean(updated.runtimeControls?.gpuEnabled));
      setRuntimeAllowOllamaCloudModels(
        Boolean(updated.runtimeControls?.allowOllamaCloudModels),
      );
      setRuntimeEffectiveGpuEnabled(
        Boolean(updated.runtimeControls?.effectiveGpuEnabled),
      );
      setStatus("Runtime model policy saved.");
      await refreshOverview(true);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setRuntimePolicyBusy(false);
    }
  }

  async function refreshSelectedModel(modelId: string) {
    const id = modelId.trim();
    if (!id || !runtimeAvailable) {
      setSelectedModel(null);
      setCompatibility(null);
      return;
    }
    try {
      const [modelRes, compatibilityRes] = await Promise.all([
        api.modelRuntime.get(id),
        api.modelRuntime.compatibility(id),
      ]);
      setSelectedModel(modelRes.model);
      setCompatibility(compatibilityRes.compatibility);
    } catch (error) {
      setSelectedModel(null);
      setCompatibility(null);
      setErr(error instanceof Error ? error.message : String(error));
    }
  }

  useEffect(() => {
    void refreshOverview(false);
    const intervalId = window.setInterval(
      () => void refreshOverview(true),
      10000,
    );
    return () => window.clearInterval(intervalId);
  }, []);

  useEffect(() => {
    void refreshSelectedModel(selectedModelId);
  }, [selectedModelId]);

  useEffect(() => {
    writeCachedChatModelSelection(chatSelectedModelId);
  }, [chatSelectedModelId]);

  async function handleImport() {
    const path = importPath.trim();
    if (!path) {
      setErr("Import path is required.");
      return;
    }
    setImportBusy(true);
    try {
      const result = await api.modelRuntime.import({
        path,
        displayName: importDisplayName.trim() || undefined,
        family: importFamily.trim() || undefined,
        backend: importBackend.trim() || undefined,
        capabilities: importCapabilities
          .split(",")
          .map((value) => value.trim())
          .filter(Boolean),
        preferred: importPreferred,
        ...modelManagementRequest(),
      });
      setImportPath("");
      setImportDisplayName("");
      setImportFamily("");
      setImportBackend("");
      setImportCapabilities("chat,completion");
      setImportPreferred(false);
      const governanceMessage = modelGovernanceMessage(result, "Model import");
      if (governanceMessage) {
        await refreshOverview(true);
        setStatus(governanceMessage);
        setErr(null);
        return;
      }
      if (!result.result?.model?.id) {
        throw new Error("Model import response did not include a model.");
      }
      await refreshOverview(true, result.result.model.id);
      setStatus(
        result.result.duplicate
          ? `Model already registered: ${result.result.model.id}`
          : `Imported model ${result.result.model.id}`,
      );
      setErr(null);
    } catch (error) {
      setErr(error instanceof Error ? error.message : String(error));
    } finally {
      setImportBusy(false);
    }
  }

  async function handleScan() {
    setScanBusy(true);
    try {
      const result = await api.modelRuntime.scan(modelManagementRequest());
      await refreshOverview(true);
      setStatus(`Scanned model home: ${result.count} registered models`);
      setErr(null);
    } catch (error) {
      setErr(error instanceof Error ? error.message : String(error));
    } finally {
      setScanBusy(false);
    }
  }

  async function runAction(
    modelId: string,
    action:
      | "verify"
      | "enable"
      | "disable"
      | "archive"
      | "remove"
      | "load"
      | "unload",
  ) {
    const busyKey = `${action}:${modelId}`;
    setActionBusy(busyKey);
    try {
      let result: unknown = null;
      switch (action) {
        case "verify":
          result = await api.modelRuntime.verify(
            modelId,
            modelManagementRequest(),
          );
          break;
        case "enable":
          result = await api.modelRuntime.enable(
            modelId,
            modelManagementRequest(),
          );
          break;
        case "disable":
          result = await api.modelRuntime.disable(
            modelId,
            modelManagementRequest(),
          );
          break;
        case "archive":
          result = await api.modelRuntime.archive(
            modelId,
            modelManagementRequest(),
          );
          break;
        case "remove":
          result = await api.modelRuntime.remove(
            modelId,
            modelManagementRequest(),
          );
          break;
        case "load":
          result = await api.modelRuntime.load(
            modelId,
            modelManagementRequest(),
          );
          break;
        case "unload":
          result = await api.modelRuntime.unload(
            modelId,
            modelManagementRequest(),
          );
          break;
      }
      const governanceMessage = modelGovernanceMessage(
        result,
        `Model ${action}`,
      );
      if (governanceMessage) {
        await refreshOverview(true, modelId);
        await refreshSelectedModel(modelId);
        setStatus(governanceMessage);
        setErr(null);
        return;
      }
      if (action === "remove" && selectedModelId === modelId) {
        setSelectedModelId("");
      }
      const nextSelection =
        selectedModelId === modelId && action === "remove" ? "" : modelId;
      await refreshOverview(true, nextSelection);
      await refreshSelectedModel(nextSelection);
      setStatus(`${action} completed for ${modelId}`);
      setErr(null);
    } catch (error) {
      setErr(error instanceof Error ? error.message : String(error));
    } finally {
      setActionBusy(null);
    }
  }

  const advancedView =
    uiMode === "metrics" || searchParams.get("view") === "registry";

  if (!advancedView) {
    return (
      <div className="forge-ops-board space-y-5">
        <header className="rounded border border-forge-platinum/10 bg-black/20 p-4 lg:flex lg:items-end lg:justify-between">
          <div className="min-w-0">
            <div className="forge-ops-label">Model Runtime</div>
            <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
              Runtime command board
            </h1>
            <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
              Cognitive model view with runtime status, active model,
              availability, and chat preference. Registry controls open only
              when requested.
            </p>
          </div>
          <div className="mt-4 flex flex-wrap items-center gap-2 lg:mt-0 lg:justify-end">
            <span
              className={badgeClass(
                health?.ok === false || !runtimeAvailable ? "error" : "ok",
              )}
            >
              {runtimeAvailable ? health?.status || "runtime" : "unavailable"}
            </span>
            <GhostButton
              onClick={() => void refreshOverview(true)}
              disabled={loading}
            >
              {loading ? "Refreshing" : "Refresh"}
            </GhostButton>
            <GhostButton onClick={() => setSearchParams({ view: "registry" })}>
              Open Registry
            </GhostButton>
          </div>
        </header>

        <section className="forge-ops-panel">
          {err ? (
            <div className="m-3">
              <EmptyState
                title="Runtime request failed"
                detail={err}
                tone="bad"
              />
            </div>
          ) : null}
          {!runtimeAvailable ? (
            <div className="m-3">
              <EmptyState
                title="Model runtime unavailable"
                detail="Core UI remains healthy; enable modelruntime before registry and load controls become active."
                tone="warn"
              />
            </div>
          ) : null}
          <div className="forge-ops-panel__body grid gap-3 lg:grid-cols-[minmax(0,1fr)_18rem]">
            <div className="rounded border border-forge-platinum/10 bg-black/20 p-4">
              <div className="forge-ops-label">Runtime Status</div>
              <div className="mt-2 text-2xl font-semibold text-forge-ash">
                {health?.status || (health?.ok ? "ok" : "unknown")}
              </div>
              <div className="mt-2 text-sm text-forge-mist">
                {loaded?.count ?? 0} loaded · {queue?.depth ?? 0} queued ·{" "}
                {chatSelectableModels.length} chat-capable
              </div>
              {health?.degradedReasons?.length ? (
                <div className="mt-3 rounded-md border border-forge-ember/30 bg-forge-ember/10 p-2 text-xs leading-5 text-forge-ash">
                  {health.degradedReasons.join("; ")}
                </div>
              ) : null}
              {health?.policyWarnings?.length ? (
                <div className="mt-2 rounded-md border border-white/10 bg-black/25 p-2 text-xs leading-5 text-forge-mist">
                  {health.policyWarnings.join("; ")}
                </div>
              ) : null}
              <div className="mt-4 grid gap-2 md:grid-cols-2">
                <StateBox
                  title="Active Model"
                  rows={[
                    [
                      "Selected",
                      selectedModelSummary?.displayName ||
                        selectedModelSummary?.id ||
                        "none",
                    ],
                    [
                      "Backend",
                      selectedModelSummary?.backend ||
                        health?.backend ||
                        "unknown",
                    ],
                    ["Loaded", selectedLoadedRecord?.status || "not loaded"],
                  ]}
                />
                <StateBox
                  title="Availability"
                  rows={[
                    ["Registered", String(models.length)],
                    [
                      "Available",
                      String(modelCounts.available ?? usage?.available ?? 0),
                    ],
                    [
                      "Disabled/Archived",
                      String((usage?.disabled ?? 0) + (usage?.archived ?? 0)),
                    ],
                  ]}
                />
              </div>
            </div>
            <div className="rounded border border-forge-platinum/10 bg-black/20 p-4">
              <div className="forge-ops-label">Chat Preference</div>
              <select
                className="forge-input mt-3"
                value={chatSelectedModelId}
                onChange={(event) => {
                  setChatSelectedModelId(event.target.value);
                  setStatus(
                    event.target.value
                      ? `Chat model preference set to ${event.target.value}`
                      : "Chat model preference cleared (auto)",
                  );
                }}
              >
                <option value="">Auto routing</option>
                {chatSelectableModels.map((model) => (
                  <option key={model.id} value={model.id}>
                    {model.displayName?.trim() || model.id}
                  </option>
                ))}
              </select>
              <GhostButton
                className="mt-3 w-full"
                onClick={() => setChatSelectedModelId("")}
                disabled={!chatSelectedModelId}
              >
                Clear preference
              </GhostButton>
            </div>
          </div>
        </section>

        <section className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Registry</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Compact model list. Select a model or open advanced registry for
                lifecycle controls.
              </div>
            </div>
            <span className="font-mono text-[11px] text-forge-mist/60">
              {models.length} registered
            </span>
          </div>
          <div className="forge-ops-panel__body">
            {models.length === 0 ? (
              <EmptyState
                title={loading ? "Refreshing registry" : "No models registered"}
                detail={
                  loading
                    ? "Waiting for the runtime registry, queue, loaded-models, and backend health calls to return."
                    : "Import a local model or open the registry to reconcile a configured model home."
                }
              />
            ) : (
              <div className="space-y-2">
                {models.slice(0, 12).map((model) => (
                  <button
                    key={model.id}
                    type="button"
                    onClick={() => setSelectedModelId(model.id)}
                    className={cx(
                      "flex w-full items-start justify-between gap-3 rounded border border-forge-platinum/10 bg-black/20 px-4 py-3 text-left transition hover:border-forge-ember/35 hover:bg-black/30 sm:items-center",
                      selectedModelId === model.id &&
                        "border-forge-ember/45 bg-forge-ember/10",
                    )}
                  >
                    <span className="min-w-0">
                      <span className="block break-all font-mono text-sm text-forge-ash">
                        {model.id}
                      </span>
                      <span className="mt-1 block break-words text-xs leading-5 text-forge-mist">
                        {model.displayName || "Unnamed model"} ·{" "}
                        {model.backend || "backend unset"}
                      </span>
                    </span>
                    <span className={cx("shrink-0", badgeClass(model.status))}>
                      {model.status || "unknown"}
                    </span>
                  </button>
                ))}
              </div>
            )}
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className="forge-ops-board space-y-5">
      <header className="rounded border border-forge-platinum/10 bg-black/20 p-4 lg:flex lg:items-end lg:justify-between">
        <div className="min-w-0">
          <div className="forge-ops-label">Models</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Model runtime board
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            Governed FORGE model runtime surface: import, verify, load, disable,
            archive, inspect health, and track loaded/runtime state.
          </p>
        </div>
        <div className="mt-4 flex flex-wrap items-center gap-2 lg:mt-0 lg:justify-end">
          <span
            className={badgeClass(
              health?.ok === false || !runtimeAvailable ? "error" : "ok",
            )}
          >
            {runtimeAvailable ? health?.status || "runtime" : "unavailable"}
          </span>
          <GhostButton onClick={() => void handleScan()} disabled={scanBusy}>
            {scanBusy ? "Scanning" : "Scan Model Home"}
          </GhostButton>
          <GhostButton
            onClick={() => void refreshOverview(true)}
            disabled={loading}
          >
            {loading ? "Refreshing" : "Refresh"}
          </GhostButton>
        </div>
      </header>

      {err ? (
        <div className="forge-ops-panel border-forge-ember/30 bg-forge-ember/10 p-3">
          <div className="text-sm font-semibold text-forge-ash">
            Runtime request failed
          </div>
          <div className="mt-1 break-words text-xs leading-5 text-forge-mist">
            {err}
          </div>
        </div>
      ) : null}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
        <Metric
          label="Registered"
          value={usage?.registered ?? models.length}
          hint="known manifests"
          tone="muted"
        />
        <Metric
          label="Loaded"
          value={usage?.loaded ?? loaded?.count ?? 0}
          hint="active runtime state"
          tone={(usage?.loaded ?? loaded?.count ?? 0) > 0 ? "ok" : "muted"}
        />
        <Metric
          label="Queue Depth"
          value={queue?.depth ?? 0}
          hint={queue?.scheduler || "scheduler"}
          tone={(queue?.depth ?? 0) > 0 ? "warn" : "ok"}
        />
        <Metric
          label="Completed"
          value={usage?.completed ?? 0}
          hint="finished requests"
          tone="ok"
        />
        <Metric
          label="Health"
          value={health?.status || (health?.ok ? "ok" : "unknown")}
          hint={health?.backend || "runtime"}
          tone={health?.ok === false ? "bad" : "ok"}
        />
        <Metric
          label="Chat Ready"
          value={chatSelectableModels.length}
          hint="eligible models"
          tone={chatSelectableModels.length > 0 ? "ok" : "warn"}
        />
      </section>

      <section className="grid gap-3 xl:grid-cols-[minmax(0,1.4fr)_minmax(320px,0.95fr)]">
        <div className="forge-ops-panel p-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="forge-ops-label">Runtime Overview</div>
              <div className="mt-2 max-w-2xl text-sm leading-6 text-forge-mist">
                Central control surface for model registration, operator chat
                preference, runtime availability, and backend posture.
              </div>
            </div>
            <div className="rounded-full border border-white/10 bg-black/25 px-3 py-1 text-[11px] text-forge-mist">
              Last poll{" "}
              <span className="text-forge-ash">
                {formatTime(lastUpdatedAt)}
              </span>
            </div>
          </div>
          <div className="mt-4 grid gap-2 rounded border border-white/10 bg-black/20 p-3 text-xs md:grid-cols-3">
            <label className="flex min-w-0 items-center gap-2">
              <input
                type="checkbox"
                checked={runtimeGpuEnabled}
                disabled={runtimePolicyBusy}
                onChange={(event) => {
                  const checked = event.target.checked;
                  setRuntimeGpuEnabled(checked);
                  void saveRuntimePolicy({ gpuEnabled: checked });
                }}
              />
              <span className="font-semibold tracking-wide text-forge-mist">
                GPU acceleration
              </span>
            </label>
            <label className="flex min-w-0 items-center gap-2">
              <input
                type="checkbox"
                checked={runtimeAllowOllamaCloudModels}
                disabled={runtimePolicyBusy}
                onChange={(event) => {
                  const checked = event.target.checked;
                  setRuntimeAllowOllamaCloudModels(checked);
                  void saveRuntimePolicy({ allowOllamaCloudModels: checked });
                }}
              />
              <span className="font-semibold tracking-wide text-forge-mist">
                Ollama cloud models
              </span>
            </label>
            <div className="min-w-0 rounded border border-white/10 bg-black/25 px-3 py-2 text-forge-mist">
              Effective GPU:{" "}
              <span className="text-forge-ash">
                {runtimeEffectiveGpuEnabled ? "on" : "off"}
              </span>
            </div>
          </div>
          {health?.degradedReasons?.length || health?.policyWarnings?.length ? (
            <div className="mt-4 grid gap-2 text-xs text-forge-mist md:grid-cols-2">
              {health.degradedReasons?.length ? (
                <div className="rounded border border-forge-ember/30 bg-forge-ember/10 px-3 py-2 text-forge-ash">
                  {health.degradedReasons.join("; ")}
                </div>
              ) : null}
              {health.policyWarnings?.length ? (
                <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                  {health.policyWarnings.join("; ")}
                </div>
              ) : null}
            </div>
          ) : null}
          <div className="mt-4 grid gap-2 text-[11px] text-forge-mist md:grid-cols-2 xl:grid-cols-4">
            <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
              Available:{" "}
              <span className="text-forge-ash">
                {modelCounts.available ?? usage?.available ?? 0}
              </span>
            </div>
            <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
              Imported:{" "}
              <span className="text-forge-ash">
                {modelCounts.imported ?? usage?.imported ?? 0}
              </span>
            </div>
            <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
              Verified:{" "}
              <span className="text-forge-ash">
                {modelCounts.verified ?? usage?.verified ?? 0}
              </span>
            </div>
            <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
              Disabled/Archived:{" "}
              <span className="text-forge-ash">
                {(usage?.disabled ?? 0) + (usage?.archived ?? 0)}
              </span>
            </div>
          </div>
          <div className="mt-4 rounded border border-white/10 bg-black/20 p-3">
            <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-forge-mist/80">
              Current Focus
            </div>
            {selectedModelSummary ? (
              <div className="mt-3 flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="break-all font-mono text-sm text-forge-ash">
                    {selectedModelSummary.id}
                  </div>
                  <div className="mt-1 break-words text-xs leading-5 text-forge-mist">
                    {selectedModelSummary.displayName || "Unnamed model"} ·{" "}
                    {selectedModelSummary.family || "family unknown"} ·{" "}
                    {selectedModelSummary.backend || "backend unset"}
                  </div>
                  <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-forge-mist">
                    <span className="min-w-0 rounded-full border border-white/10 bg-black/25 px-2 py-1">
                      Capabilities:{" "}
                      <span className="break-words text-forge-ash">
                        {summarizeList(selectedModelSummary.capabilities)}
                      </span>
                    </span>
                    <span className="rounded-full border border-white/10 bg-black/25 px-2 py-1">
                      Loaded:{" "}
                      <span className="text-forge-ash">
                        {selectedLoadedRecord?.status || "not loaded"}
                      </span>
                    </span>
                  </div>
                </div>
                <span
                  className={cx(
                    "rounded-full border px-2 py-1 text-[11px] font-medium",
                    badgeClass(selectedModelSummary.status),
                  )}
                >
                  {selectedModelSummary.status || "unknown"}
                </span>
              </div>
            ) : (
              <div className="mt-3 text-sm text-forge-mist">
                Select a registered model to pin detailed compatibility and
                runtime state on the right.
              </div>
            )}
          </div>
        </div>

        <div className="forge-ops-panel p-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="forge-ops-label">Chat Model Preference</div>
              <div className="mt-2 text-sm leading-6 text-forge-mist">
                Sets the desktop chat assistant’s requested model before adapter
                fallback.
              </div>
            </div>
            <span className="rounded-full border border-white/10 bg-black/30 px-3 py-1 text-[11px] text-forge-mist">
              {chatSelectedModelId ? "Manual pin" : "Auto routing"}
            </span>
          </div>
          <div className="relative z-20 mt-4 overflow-visible rounded border border-white/10 bg-black/20 p-3">
            <label className="block text-[11px] font-semibold uppercase tracking-[0.14em] text-forge-mist">
              Preferred chat model
            </label>
            <div className="mt-2 flex flex-col gap-2 sm:flex-row">
              <div className="relative z-20 min-w-0 flex-1 overflow-visible">
                <select
                  className="forge-input relative z-20 h-11 w-full min-w-0 py-2 text-sm"
                  value={chatSelectedModelId}
                  onChange={(event) => {
                    setChatSelectedModelId(event.target.value);
                    setStatus(
                      event.target.value
                        ? `Chat model preference set to ${event.target.value}`
                        : "Chat model preference cleared (auto)",
                    );
                  }}
                >
                  <option value="">
                    Auto (runtime default / adapter fallback)
                  </option>
                  {chatSelectedModelId &&
                  !chatSelectableModels.some(
                    (model) => model.id === chatSelectedModelId,
                  ) ? (
                    <option value={chatSelectedModelId}>
                      Saved: {chatSelectedModelId} (not in current runtime list)
                    </option>
                  ) : null}
                  {chatSelectableModels.map((model) => (
                    <option key={model.id} value={model.id}>
                      {model.displayName?.trim() || model.id}
                    </option>
                  ))}
                </select>
              </div>
              <GhostButton
                className="min-h-11 shrink-0 px-4"
                onClick={() => {
                  setChatSelectedModelId("");
                  setStatus("Chat model preference cleared (auto).");
                }}
                disabled={!chatSelectedModelId}
              >
                Clear
              </GhostButton>
            </div>
            <div className="mt-3 flex flex-wrap gap-2 text-[11px] text-forge-mist">
              <span className="min-w-0 rounded-full border border-white/10 bg-black/25 px-2 py-1">
                Active setting:{" "}
                <span className="break-all text-forge-ash">
                  {chatSelectedModelId || "auto"}
                </span>
              </span>
              <span className="rounded-full border border-white/10 bg-black/25 px-2 py-1">
                Eligible models:{" "}
                <span className="text-forge-ash">
                  {chatSelectableModels.length}
                </span>
              </span>
            </div>
          </div>
          {!chatSelectionKnown && chatSelectedModelId ? (
            <div className="mt-3 break-words rounded border border-forge-ember/30 bg-forge-ember/10 px-3 py-2 text-[11px] leading-5 text-forge-emberSoft">
              Saved chat model `{chatSelectedModelId}` is no longer registered;
              chat will fail over unless you clear or update it.
            </div>
          ) : null}
        </div>
      </section>

      <section className="forge-ops-panel">
        <div className="forge-ops-panel__head">
          <div>
            <div className="forge-ops-title">Import and Registration</div>
            <div className="mt-1 text-xs text-forge-mist/65">
              Local GGUF and manifest-backed model registration. File deletion
              remains intentionally out of scope.
            </div>
          </div>
          <span className="font-mono text-[11px] text-forge-mist/60">
            model.management
          </span>
        </div>
        <div className="forge-ops-panel__body">
          <div className="grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(240px,0.65fr)]">
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              <label className="text-xs text-forge-mist">
                Path
                <input
                  className="forge-input mt-1"
                  value={importPath}
                  onChange={(e) => setImportPath(e.target.value)}
                  placeholder="/models/coder.gguf"
                />
              </label>
              <label className="text-xs text-forge-mist">
                Display name
                <input
                  className="forge-input mt-1"
                  value={importDisplayName}
                  onChange={(e) => setImportDisplayName(e.target.value)}
                  placeholder="Qwen Coder"
                />
              </label>
              <label className="text-xs text-forge-mist">
                Family
                <input
                  className="forge-input mt-1"
                  value={importFamily}
                  onChange={(e) => setImportFamily(e.target.value)}
                  placeholder="qwen"
                />
              </label>
              <label className="relative z-10 overflow-visible text-xs text-forge-mist">
                Backend
                <select
                  className="forge-input relative z-20 mt-1 h-10 w-full"
                  value={importBackend}
                  onChange={(e) => setImportBackend(e.target.value)}
                >
                  <option value="">manifest/default</option>
                  <option value="llama_cpp">llama_cpp</option>
                  <option value="openai_compat">openai_compat</option>
                  <option value="vllm">vllm</option>
                </select>
              </label>
              <label className="text-xs text-forge-mist">
                Capabilities
                <input
                  className="forge-input mt-1"
                  value={importCapabilities}
                  onChange={(e) => setImportCapabilities(e.target.value)}
                  placeholder="chat,completion"
                />
              </label>
              <label className="flex items-center gap-2 self-end rounded border border-white/10 bg-black/20 px-3 py-2 text-xs text-forge-mist">
                <input
                  type="checkbox"
                  checked={importPreferred}
                  onChange={(e) => setImportPreferred(e.target.checked)}
                />
                Mark as preferred
              </label>
            </div>
            <div className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
              <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-forge-mist/80">
                Registration Notes
              </div>
              <div className="mt-3 space-y-2">
                <div>
                  Import records runtime details only; removal never deletes
                  model files.
                </div>
                <div>
                  Use <span className="text-forge-ash">preferred</span> when the
                  imported model should be favored by runtime compatibility
                  checks.
                </div>
                <div>
                  Capabilities should stay comma-separated so registry and chat
                  filtering remain aligned.
                </div>
              </div>
            </div>
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            <PrimaryButton
              className="min-h-11 px-4"
              onClick={() => void handleImport()}
              disabled={importBusy}
            >
              {importBusy ? "Importing..." : "Import Model"}
            </PrimaryButton>
            <GhostButton
              className="min-h-11 px-4"
              onClick={() => void handleScan()}
              disabled={scanBusy}
            >
              {scanBusy ? "Scanning..." : "Reconcile Registry"}
            </GhostButton>
          </div>
        </div>
      </section>

      <div className="grid gap-3 xl:grid-cols-[1.15fr_0.85fr]">
        <section className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Registered Models</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Model lifecycle state and operational controls.
              </div>
            </div>
            <span className="font-mono text-[11px] text-forge-mist/60">
              {models.length} rows
            </span>
          </div>
          <div className="forge-ops-panel__body">
            {models.length === 0 ? (
              <EmptyState
                title={
                  loading ? "Loading model registry" : "No registered models"
                }
                detail={
                  loading
                    ? "The runtime registry is refreshing. Lifecycle controls will appear once models are returned."
                    : "Import a local model or reconcile the model home to populate governed lifecycle controls."
                }
              />
            ) : (
              <div className="space-y-3">
                <div className="flex flex-wrap gap-2 text-[11px] text-forge-mist">
                  <span className="rounded-full border border-white/10 bg-black/20 px-3 py-1">
                    Registry size{" "}
                    <span className="text-forge-ash">{models.length}</span>
                  </span>
                  <span className="rounded-full border border-white/10 bg-black/20 px-3 py-1">
                    Loaded now{" "}
                    <span className="text-forge-ash">{loaded?.count ?? 0}</span>
                  </span>
                  <span className="rounded-full border border-white/10 bg-black/20 px-3 py-1">
                    Selected{" "}
                    <span className="text-forge-ash">
                      {selectedModelId || "none"}
                    </span>
                  </span>
                </div>
                {models.map((model) => {
                  const isSelected = model.id === selectedModelId;
                  const loadedRecord =
                    loaded?.models.find((item) => item.modelId === model.id) ??
                    null;
                  const busyPrefix = actionBusy?.split(":")[0] ?? "";
                  const isBusy = actionBusy?.endsWith(`:${model.id}`) ?? false;
                  return (
                    <div
                      key={model.id}
                      role="button"
                      tabIndex={0}
                      onClick={() => setSelectedModelId(model.id)}
                      onKeyDown={(event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          setSelectedModelId(model.id);
                        }
                      }}
                      className={cx(
                        "w-full rounded border px-4 py-4 text-left transition focus:outline-none focus:ring-2 focus:ring-forge-accent/40",
                        isSelected
                          ? "border-forge-accent/55 bg-[linear-gradient(135deg,rgba(27,29,31,0.9),rgba(9,10,11,0.92))] shadow-[0_0_0_1px_rgba(215,181,109,0.06)]"
                          : "border-white/10 bg-black/20 hover:border-forge-accent/40 hover:bg-black/25",
                      )}
                    >
                      <div className="flex flex-wrap items-start justify-between gap-3">
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <div className="break-all font-mono text-sm text-forge-ash">
                              {model.id}
                            </div>
                            {chatSelectedModelId === model.id ? (
                              <span className="rounded-full border border-forge-electric/35 bg-forge-electric/10 px-2 py-1 text-[10px] font-medium uppercase tracking-[0.12em] text-forge-electric">
                                Chat preferred
                              </span>
                            ) : null}
                            {isSelected ? (
                              <span className="rounded-full border border-forge-accent/35 bg-forge-accent/10 px-2 py-1 text-[10px] font-medium uppercase tracking-[0.12em] text-forge-accent">
                                Selected
                              </span>
                            ) : null}
                          </div>
                          <div className="mt-1 break-words text-xs leading-5 text-forge-mist">
                            {model.displayName || "Unnamed model"} ·{" "}
                            {model.family || "family unknown"} ·{" "}
                            {model.backend || "backend unset"} ·{" "}
                            {model.format || "format unknown"}
                          </div>
                        </div>
                        <span
                          className={cx(
                            "rounded-full border px-2 py-1 text-[11px] font-medium",
                            badgeClass(model.status),
                          )}
                        >
                          {model.status || "unknown"}
                        </span>
                      </div>
                      <div className="mt-3 flex flex-wrap gap-2 text-[11px] text-forge-mist">
                        <span className="min-w-0 rounded-full border border-white/10 bg-black/25 px-2 py-1">
                          Capabilities:{" "}
                          <span className="break-words text-forge-ash">
                            {summarizeList(model.capabilities)}
                          </span>
                        </span>
                        <span className="rounded-full border border-white/10 bg-black/25 px-2 py-1">
                          Loaded:{" "}
                          <span className="text-forge-ash">
                            {loadedRecord?.status || "not loaded"}
                          </span>
                        </span>
                      </div>
                      <div className="mt-4 flex flex-wrap gap-2">
                        <GhostButton
                          className="min-h-10 px-3"
                          onClick={(event) => {
                            event.stopPropagation();
                            void runAction(model.id, "verify");
                          }}
                          disabled={isBusy}
                        >
                          {busyPrefix === "verify" && isBusy
                            ? "Verifying..."
                            : "Verify"}
                        </GhostButton>
                        {normalizeStatus(model.status) === "disabled" ? (
                          <GhostButton
                            className="min-h-10 px-3"
                            onClick={(event) => {
                              event.stopPropagation();
                              void runAction(model.id, "enable");
                            }}
                            disabled={isBusy}
                          >
                            {busyPrefix === "enable" && isBusy
                              ? "Enabling..."
                              : "Enable"}
                          </GhostButton>
                        ) : (
                          <GhostButton
                            className="min-h-10 px-3"
                            onClick={(event) => {
                              event.stopPropagation();
                              void runAction(model.id, "disable");
                            }}
                            disabled={isBusy}
                          >
                            {busyPrefix === "disable" && isBusy
                              ? "Disabling..."
                              : "Disable"}
                          </GhostButton>
                        )}
                        {loadedRecord ? (
                          <PrimaryButton
                            className="min-h-10 px-3"
                            onClick={(event) => {
                              event.stopPropagation();
                              void runAction(model.id, "unload");
                            }}
                            disabled={isBusy}
                          >
                            {busyPrefix === "unload" && isBusy
                              ? "Unloading..."
                              : "Unload"}
                          </PrimaryButton>
                        ) : (
                          <PrimaryButton
                            className="min-h-10 px-3"
                            onClick={(event) => {
                              event.stopPropagation();
                              void runAction(model.id, "load");
                            }}
                            disabled={
                              isBusy ||
                              normalizeStatus(model.status) === "archived"
                            }
                          >
                            {busyPrefix === "load" && isBusy
                              ? "Loading..."
                              : "Load"}
                          </PrimaryButton>
                        )}
                        <GhostButton
                          className="min-h-10 px-3"
                          onClick={(event) => {
                            event.stopPropagation();
                            void runAction(model.id, "archive");
                          }}
                          disabled={
                            isBusy ||
                            normalizeStatus(model.status) === "archived"
                          }
                        >
                          {busyPrefix === "archive" && isBusy
                            ? "Archiving..."
                            : "Archive"}
                        </GhostButton>
                        <GhostButton
                          className="min-h-10 px-3"
                          onClick={(event) => {
                            event.stopPropagation();
                            if (
                              !window.confirm(
                                `Remove registration for ${model.id}? Model files will not be deleted.`,
                              )
                            )
                              return;
                            void runAction(model.id, "remove");
                          }}
                          disabled={isBusy}
                        >
                          {busyPrefix === "remove" && isBusy
                            ? "Removing..."
                            : "Remove"}
                        </GhostButton>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </section>

        <div className="space-y-3">
          <section className="forge-ops-panel">
            <div className="forge-ops-panel__head">
              <div>
                <div className="forge-ops-title">Selected Model</div>
                <div className="mt-1 text-xs text-forge-mist/65">
                  Compatibility, loaded state, model details, and backend
                  readiness for the focused model.
                </div>
              </div>
            </div>
            <div className="forge-ops-panel__body">
              {!selectedModelSummary ? (
                <EmptyState
                  title="No model selected"
                  detail="Select a registry row to inspect compatibility, manifest metadata, backend state, and lifecycle controls."
                />
              ) : (
                <div className="space-y-4">
                  <div className="rounded border border-forge-accent/20 bg-[linear-gradient(135deg,rgba(27,29,31,0.88),rgba(8,9,10,0.92))] p-4">
                    <div className="flex flex-wrap items-center justify-between gap-3">
                      <div className="min-w-0">
                        <div className="break-all font-mono text-sm text-forge-ash">
                          {selectedModelSummary.id}
                        </div>
                        <div className="mt-1 break-words text-xs leading-5 text-forge-mist">
                          {selectedModelSummary.displayName || "Unnamed model"}{" "}
                          · {selectedModelSummary.family || "family unknown"} ·{" "}
                          {selectedModelSummary.backend || "backend unset"}
                        </div>
                      </div>
                      <span
                        className={cx(
                          "rounded-full border px-2 py-1 text-[11px] font-medium",
                          badgeClass(selectedModelSummary.status),
                        )}
                      >
                        {selectedModelSummary.status || "unknown"}
                      </span>
                    </div>
                    <div className="mt-4 grid gap-2 text-[11px] text-forge-mist sm:grid-cols-2">
                      <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                        Format:{" "}
                        <span className="text-forge-ash">
                          {selectedModelSummary.format || "unknown"}
                        </span>
                      </div>
                      <div className="min-w-0 rounded border border-white/10 bg-black/20 px-3 py-2">
                        Capabilities:{" "}
                        <span className="break-words text-forge-ash">
                          {summarizeList(selectedModelSummary.capabilities)}
                        </span>
                      </div>
                      <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                        Loaded state:{" "}
                        <span className="text-forge-ash">
                          {selectedLoadedRecord?.status || "not loaded"}
                        </span>
                      </div>
                      <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                        Loaded at:{" "}
                        <span className="text-forge-ash">
                          {selectedLoadedRecord?.loadedAtMs
                            ? formatTime(selectedLoadedRecord.loadedAtMs)
                            : "—"}
                        </span>
                      </div>
                    </div>
                    <div className="mt-4 flex flex-wrap gap-2">
                      <GhostButton
                        className="min-h-10 px-3"
                        onClick={() =>
                          void runAction(selectedModelSummary.id, "verify")
                        }
                        disabled={
                          actionBusy?.endsWith(`:${selectedModelSummary.id}`) ??
                          false
                        }
                      >
                        {actionBusy === `verify:${selectedModelSummary.id}`
                          ? "Verifying..."
                          : "Verify"}
                      </GhostButton>
                      {normalizeStatus(selectedModelSummary.status) ===
                      "disabled" ? (
                        <GhostButton
                          className="min-h-10 px-3"
                          onClick={() =>
                            void runAction(selectedModelSummary.id, "enable")
                          }
                          disabled={
                            actionBusy?.endsWith(
                              `:${selectedModelSummary.id}`,
                            ) ?? false
                          }
                        >
                          {actionBusy === `enable:${selectedModelSummary.id}`
                            ? "Enabling..."
                            : "Enable"}
                        </GhostButton>
                      ) : (
                        <GhostButton
                          className="min-h-10 px-3"
                          onClick={() =>
                            void runAction(selectedModelSummary.id, "disable")
                          }
                          disabled={
                            actionBusy?.endsWith(
                              `:${selectedModelSummary.id}`,
                            ) ?? false
                          }
                        >
                          {actionBusy === `disable:${selectedModelSummary.id}`
                            ? "Disabling..."
                            : "Disable"}
                        </GhostButton>
                      )}
                      {selectedLoadedRecord ? (
                        <PrimaryButton
                          className="min-h-10 px-3"
                          onClick={() =>
                            void runAction(selectedModelSummary.id, "unload")
                          }
                          disabled={
                            actionBusy?.endsWith(
                              `:${selectedModelSummary.id}`,
                            ) ?? false
                          }
                        >
                          {actionBusy === `unload:${selectedModelSummary.id}`
                            ? "Unloading..."
                            : "Unload"}
                        </PrimaryButton>
                      ) : (
                        <PrimaryButton
                          className="min-h-10 px-3"
                          onClick={() =>
                            void runAction(selectedModelSummary.id, "load")
                          }
                          disabled={
                            (actionBusy?.endsWith(
                              `:${selectedModelSummary.id}`,
                            ) ??
                              false) ||
                            normalizeStatus(selectedModelSummary.status) ===
                              "archived"
                          }
                        >
                          {actionBusy === `load:${selectedModelSummary.id}`
                            ? "Loading..."
                            : "Load"}
                        </PrimaryButton>
                      )}
                      <GhostButton
                        className="min-h-10 px-3"
                        onClick={() =>
                          void runAction(selectedModelSummary.id, "archive")
                        }
                        disabled={
                          (actionBusy?.endsWith(
                            `:${selectedModelSummary.id}`,
                          ) ??
                            false) ||
                          normalizeStatus(selectedModelSummary.status) ===
                            "archived"
                        }
                      >
                        {actionBusy === `archive:${selectedModelSummary.id}`
                          ? "Archiving..."
                          : "Archive"}
                      </GhostButton>
                      <GhostButton
                        className="min-h-10 px-3"
                        onClick={() => {
                          if (
                            !window.confirm(
                              `Remove registration for ${selectedModelSummary.id}? Model files will not be deleted.`,
                            )
                          )
                            return;
                          void runAction(selectedModelSummary.id, "remove");
                        }}
                        disabled={
                          actionBusy?.endsWith(`:${selectedModelSummary.id}`) ??
                          false
                        }
                      >
                        {actionBusy === `remove:${selectedModelSummary.id}`
                          ? "Removing..."
                          : "Remove"}
                      </GhostButton>
                    </div>
                  </div>
                  {selectedCompatibility ? (
                    <div className="rounded border border-white/10 bg-black/25 p-3 text-xs text-forge-mist">
                      <div className="font-semibold text-forge-ash">
                        Compatibility
                      </div>
                      <div className="mt-3 grid gap-2 sm:grid-cols-2">
                        <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                          Backend healthy:{" "}
                          <span className="text-forge-ash">
                            {selectedCompatibility.backendHealthy
                              ? "yes"
                              : "no"}
                          </span>
                        </div>
                        <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                          Configured:{" "}
                          <span className="text-forge-ash">
                            {selectedCompatibility.backendConfigured
                              ? "yes"
                              : "no"}
                          </span>
                        </div>
                        <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                          Supported by backend:{" "}
                          <span className="text-forge-ash">
                            {selectedCompatibility.supportedByBackend
                              ? "yes"
                              : "no"}
                          </span>
                        </div>
                        <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                          Can generate:{" "}
                          <span className="text-forge-ash">
                            {selectedCompatibility.canGenerate ? "yes" : "no"}
                          </span>
                        </div>
                        <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                          Preferred:{" "}
                          <span className="text-forge-ash">
                            {selectedCompatibility.preferred ? "yes" : "no"}
                          </span>
                        </div>
                        <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                          Backend:{" "}
                          <span className="text-forge-ash">
                            {selectedCompatibility.backend || "unknown"}
                          </span>
                        </div>
                      </div>
                      {selectedCompatibility.warnings &&
                      selectedCompatibility.warnings.length > 0 ? (
                        <div className="mt-3 break-words rounded border border-forge-ember/30 bg-forge-ember/10 px-2 py-2 text-[11px] leading-5 text-forge-ash">
                          Warnings: {selectedCompatibility.warnings.join(" · ")}
                        </div>
                      ) : null}
                      {selectedCompatibility.details &&
                      Object.keys(selectedCompatibility.details).length > 0 ? (
                        <div className="mt-3 max-h-[220px] overflow-auto rounded border border-white/10 bg-black/30 p-3 text-[11px] text-forge-mist">
                          <HumanDataView
                            value={selectedCompatibility.details}
                            compact
                          />
                        </div>
                      ) : null}
                    </div>
                  ) : null}
                  {selectedModelSummary.metadata &&
                  Object.keys(selectedModelSummary.metadata).length > 0 ? (
                    <details className="rounded border border-forge-platinum/10 bg-black/25">
                      <summary className="cursor-pointer list-none px-3 py-2">
                        <div className="forge-ops-title text-sm">
                          Model Details
                        </div>
                        <div className="mt-1 text-xs text-forge-mist/65">
                          Manifest and registry details for this model.
                        </div>
                      </summary>
                      <div className="max-h-[260px] overflow-auto rounded border border-white/10 bg-black/30 p-3 text-[11px] text-forge-mist">
                        <HumanDataView value={selectedModelSummary.metadata} />
                      </div>
                    </details>
                  ) : null}
                </div>
              )}
            </div>
          </section>

          <section className="forge-ops-panel">
            <div className="forge-ops-panel__head">
              <div>
                <div className="forge-ops-title">Runtime State</div>
                <div className="mt-1 text-xs text-forge-mist/65">
                  Queue, loaded models, usage counters, and backend health
                  exposed by the runtime itself.
                </div>
              </div>
            </div>
            <div className="forge-ops-panel__body">
              <div className="space-y-4 text-xs text-forge-mist">
                <div className="grid gap-3 sm:grid-cols-2">
                  <StateBox
                    title="Queue"
                    rows={[
                      ["Depth", String(queue?.depth ?? 0)],
                      ["Scheduler", queue?.scheduler || "—"],
                      ["Policy", queue?.policyState || "—"],
                      ["Running", String(usage?.running ?? 0)],
                    ]}
                  />
                  <StateBox
                    title="Usage"
                    rows={[
                      [
                        "Registered",
                        String(usage?.registered ?? models.length),
                      ],
                      ["Imported", String(usage?.imported ?? 0)],
                      ["Verified", String(usage?.verified ?? 0)],
                      ["Completed", String(usage?.completed ?? 0)],
                    ]}
                  />
                </div>
                <div className="rounded border border-white/10 bg-black/25 p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="font-semibold text-forge-ash">
                      Loaded models
                    </div>
                    <div className="rounded-full border border-white/10 bg-black/20 px-2 py-1 text-[11px] text-forge-mist">
                      {loaded?.models.length ?? 0} active
                    </div>
                  </div>
                  {loaded?.models.length ? (
                    <div className="mt-3 space-y-2">
                      {loaded.models.map((item) => (
                        <div
                          key={`${item.backend}:${item.modelId}`}
                          className="rounded border border-white/10 bg-black/20 px-3 py-2"
                        >
                          <div className="flex flex-wrap items-center justify-between gap-2">
                            <div className="min-w-0 break-all font-mono text-forge-ash">
                              {item.modelId}
                            </div>
                            <span
                              className={cx(
                                "rounded-full border px-2 py-1 text-[11px] font-medium",
                                badgeClass(item.status),
                              )}
                            >
                              {item.status || "status unknown"}
                            </span>
                          </div>
                          <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                            {item.backend || "backend unknown"} · loaded{" "}
                            {item.loadedAtMs
                              ? formatTime(item.loadedAtMs)
                              : "—"}
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <EmptyState
                      title="No loaded models"
                      detail="Load a verified model to make it available to runtime requests."
                    />
                  )}
                </div>
                <div className="rounded border border-white/10 bg-black/25 p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="font-semibold text-forge-ash">
                      Backend health
                    </div>
                    <div className="rounded-full border border-white/10 bg-black/20 px-2 py-1 text-[11px] text-forge-mist">
                      {backends.length} reported
                    </div>
                  </div>
                  {backends.length ? (
                    <div className="mt-3 space-y-2">
                      {backends.map((backend) => (
                        <div
                          key={`${backend.kind}:${backend.name}`}
                          className="rounded border border-white/10 bg-black/20 px-3 py-2"
                        >
                          <div className="flex flex-wrap items-center justify-between gap-2">
                            <div className="min-w-0 break-all font-mono text-forge-ash">
                              {backend.name}
                            </div>
                            <span
                              className={cx(
                                "rounded-full border px-2 py-1 text-[11px] font-medium",
                                badgeClass(
                                  backend.healthy ? "healthy" : "error",
                                ),
                              )}
                            >
                              {backend.healthy ? "healthy" : "unhealthy"}
                            </span>
                          </div>
                          <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                            {backend.kind} ·{" "}
                            {backend.loadedModel || "no loaded model"} ·{" "}
                            {backend.detail || "no extra detail"}
                          </div>
                          {backend.meta &&
                          Object.keys(backend.meta).length > 0 ? (
                            <div className="mt-2 grid gap-1 text-[11px] text-forge-mist md:grid-cols-2">
                              {Object.entries(backend.meta).map(
                                ([key, value]) => (
                                  <div
                                    key={key}
                                    className="min-w-0 break-words"
                                  >
                                    {key}:{" "}
                                    <span className="break-words text-forge-ash">
                                      {summarizeValue(value)}
                                    </span>
                                  </div>
                                ),
                              )}
                            </div>
                          ) : null}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <EmptyState
                      title="No backend health records"
                      detail="The runtime did not report backend detail for this poll."
                    />
                  )}
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}

function Metric(props: {
  label: string;
  value: string | number;
  hint: string;
  tone?: "ok" | "warn" | "bad" | "muted";
}) {
  return (
    <div className="forge-ops-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="forge-ops-label">{props.label}</div>
          <div className="mt-2 break-words text-xl font-semibold tracking-normal text-forge-ash">
            {props.value}
          </div>
        </div>
        <span className={opsToneClass(props.tone ?? "muted")}>
          {props.tone ?? "muted"}
        </span>
      </div>
      <div className="mt-3 break-words text-xs leading-5 text-forge-mist/65">
        {props.hint}
      </div>
    </div>
  );
}

function StateBox(props: { title: string; rows: Array<[string, string]> }) {
  return (
    <div className="rounded border border-forge-platinum/10 bg-black/25 p-3">
      <div className="forge-ops-title text-sm">{props.title}</div>
      <div className="mt-2 grid gap-2">
        {props.rows.map(([label, value]) => (
          <div
            key={label}
            className="flex items-start justify-between gap-3 text-[11px] text-forge-mist"
          >
            <span className="shrink-0">{label}</span>
            <span className="min-w-0 break-words text-right text-forge-ash">
              {value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function opsToneClass(tone: "ok" | "warn" | "bad" | "muted") {
  if (tone === "ok") return "forge-ops-status forge-ops-status--ok";
  if (tone === "warn") return "forge-ops-status forge-ops-status--warn";
  if (tone === "bad") return "forge-ops-status forge-ops-status--bad";
  return "forge-ops-status forge-ops-status--muted";
}
