import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";

import { FoldSection } from "../components/FoldSection";
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
      return "border-emerald-500/30 bg-emerald-500/10 text-emerald-200";
    case "loading":
    case "unloading":
    case "imported":
      return "border-forge-electric/30 bg-forge-electric/10 text-forge-electric";
    case "disabled":
    case "archived":
      return "border-forge-ember/30 bg-forge-ember/10 text-forge-ash";
    case "error":
    case "unavailable":
      return "border-forge-ember/40 bg-forge-ember/10 text-forge-emberSoft";
    default:
      return "border-white/10 bg-black/25 text-forge-mist";
  }
}

function summarizeList(values?: string[]) {
  if (!Array.isArray(values) || values.length === 0) return "none";
  return values.join(", ");
}

function summarizeValue(value: unknown) {
  if (value == null) return "—";
  if (typeof value === "string") return value || "—";
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (Array.isArray(value)) return value.length === 0 ? "[]" : JSON.stringify(value);
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

export function ModelsPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [models, setModels] = useState<ModelRuntimeModel[]>([]);
  const [selectedModelId, setSelectedModelId] = useState("");
  const [selectedModel, setSelectedModel] = useState<ModelRuntimeModel | null>(null);
  const [compatibility, setCompatibility] = useState<ModelRuntimeCompatibility | null>(null);
  const [health, setHealth] = useState<ModelRuntimeHealth | null>(null);
  const [queue, setQueue] = useState<ModelRuntimeQueueStatus | null>(null);
  const [loaded, setLoaded] = useState<ModelRuntimeLoadedStatus | null>(null);
  const [usage, setUsage] = useState<ModelRuntimeUsageSummary | null>(null);
  const [backends, setBackends] = useState<ModelRuntimeBackendStatus[]>([]);
  const [importPath, setImportPath] = useState("");
  const [importDisplayName, setImportDisplayName] = useState("");
  const [importFamily, setImportFamily] = useState("");
  const [importBackend, setImportBackend] = useState("");
  const [importCapabilities, setImportCapabilities] = useState("chat,completion");
  const [importPreferred, setImportPreferred] = useState(false);
  const [loading, setLoading] = useState(false);
  const [importBusy, setImportBusy] = useState(false);
  const [scanBusy, setScanBusy] = useState(false);
  const [actionBusy, setActionBusy] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [lastUpdatedAt, setLastUpdatedAt] = useState<number>(Date.now());

  const selectedLoadedRecord = useMemo(
    () => loaded?.models.find((item) => item.modelId === selectedModelId) ?? null,
    [loaded?.models, selectedModelId],
  );

  const modelCounts = useMemo(() => {
    return models.reduce<Record<string, number>>((acc, model) => {
      const key = normalizeStatus(model.status);
      acc[key] = (acc[key] ?? 0) + 1;
      return acc;
    }, {});
  }, [models]);

  async function refreshOverview(preserveSelection = true, preferredSelection?: string) {
    setLoading(true);
    try {
      const [modelsRes, healthRes, queueRes, loadedRes, usageRes, backendsRes] = await Promise.all([
        api.modelRuntime.list(),
        api.modelRuntime.health(),
        api.modelRuntime.queue(),
        api.modelRuntime.loaded(),
        api.modelRuntime.usage(),
        api.modelRuntime.backends(),
      ]);

      const nextModels = Array.isArray(modelsRes.models) ? modelsRes.models : [];
      setModels(nextModels);
      setHealth(healthRes.health);
      setQueue(queueRes.queue);
      setLoaded(loadedRes.loaded);
      setUsage(usageRes.usage);
      setBackends(Array.isArray(backendsRes.backends) ? backendsRes.backends : []);

      const preferredId = preferredSelection?.trim() || "";
      const nextSelectedId = preferredId && nextModels.some((model) => model.id === preferredId)
        ? preferredId
        : preserveSelection && nextModels.some((model) => model.id === selectedModelId)
          ? selectedModelId
          : nextModels[0]?.id ?? "";
      setSelectedModelId(nextSelectedId);
      setLastUpdatedAt(Date.now());
      setErr(null);
    } catch (error) {
      setErr(error instanceof Error ? error.message : String(error));
    } finally {
      setLoading(false);
    }
  }

  async function refreshSelectedModel(modelId: string) {
    const id = modelId.trim();
    if (!id) {
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
    const intervalId = window.setInterval(() => void refreshOverview(true), 10000);
    return () => window.clearInterval(intervalId);
  }, []);

  useEffect(() => {
    void refreshSelectedModel(selectedModelId);
  }, [selectedModelId]);

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
        actor: CONTROL_ACTOR,
        source: CONTROL_SOURCE,
      });
      setImportPath("");
      setImportDisplayName("");
      setImportFamily("");
      setImportBackend("");
      setImportCapabilities("chat,completion");
      setImportPreferred(false);
      await refreshOverview(true, result.result.model.id);
      setStatus(result.result.duplicate ? `Model already registered: ${result.result.model.id}` : `Imported model ${result.result.model.id}`);
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
      const result = await api.modelRuntime.scan({ actor: CONTROL_ACTOR, source: CONTROL_SOURCE });
      await refreshOverview(true);
      setStatus(`Scanned model home: ${result.count} registered models`);
      setErr(null);
    } catch (error) {
      setErr(error instanceof Error ? error.message : String(error));
    } finally {
      setScanBusy(false);
    }
  }

  async function runAction(modelId: string, action: "verify" | "enable" | "disable" | "archive" | "remove" | "load" | "unload") {
    const busyKey = `${action}:${modelId}`;
    setActionBusy(busyKey);
    try {
      switch (action) {
        case "verify":
          await api.modelRuntime.verify(modelId, { actor: CONTROL_ACTOR, source: CONTROL_SOURCE });
          break;
        case "enable":
          await api.modelRuntime.enable(modelId, { actor: CONTROL_ACTOR, source: CONTROL_SOURCE });
          break;
        case "disable":
          await api.modelRuntime.disable(modelId, { actor: CONTROL_ACTOR, source: CONTROL_SOURCE });
          break;
        case "archive":
          await api.modelRuntime.archive(modelId, { actor: CONTROL_ACTOR, source: CONTROL_SOURCE });
          break;
        case "remove":
          await api.modelRuntime.remove(modelId, { actor: CONTROL_ACTOR, source: CONTROL_SOURCE });
          if (selectedModelId === modelId) {
            setSelectedModelId("");
          }
          break;
        case "load":
          await api.modelRuntime.load(modelId, { actor: CONTROL_ACTOR, source: CONTROL_SOURCE });
          break;
        case "unload":
          await api.modelRuntime.unload(modelId, { actor: CONTROL_ACTOR, source: CONTROL_SOURCE });
          break;
      }
      const nextSelection = selectedModelId === modelId && action === "remove" ? "" : modelId;
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

  return (
    <div className="space-y-6">
      <Panel
        title="Models"
        subtitle="Governed FORGE model runtime surface: import, verify, load, disable, archive, inspect health, and track loaded/runtime state."
        actions={
          <div className="flex items-center gap-2">
            <GhostButton onClick={() => void handleScan()} disabled={scanBusy}>
              {scanBusy ? "Scanning..." : "Scan Model Home"}
            </GhostButton>
            <GhostButton onClick={() => void refreshOverview(true)} disabled={loading}>
              {loading ? "Refreshing..." : "Refresh"}
            </GhostButton>
          </div>
        }
      >
        {err ? <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
          <Metric label="Registered" value={usage?.registered ?? models.length} hint="known manifests" />
          <Metric label="Loaded" value={usage?.loaded ?? loaded?.count ?? 0} hint="active runtime state" />
          <Metric label="Queue Depth" value={queue?.depth ?? 0} hint={queue?.scheduler || "scheduler"} />
          <Metric label="Completed" value={usage?.completed ?? 0} hint="finished requests" />
          <Metric label="Health" value={health?.status || (health?.ok ? "ok" : "unknown")} hint={health?.backend || "runtime"} />
          <Metric label="Updated" value={formatTime(lastUpdatedAt)} hint="desktop poll timestamp" />
        </div>
        <div className="mt-4 grid gap-2 text-[11px] text-forge-mist md:grid-cols-2 xl:grid-cols-4">
          <div>Available: <span className="text-forge-ash">{modelCounts.available ?? usage?.available ?? 0}</span></div>
          <div>Imported: <span className="text-forge-ash">{modelCounts.imported ?? usage?.imported ?? 0}</span></div>
          <div>Verified: <span className="text-forge-ash">{modelCounts.verified ?? usage?.verified ?? 0}</span></div>
          <div>Disabled/Archived: <span className="text-forge-ash">{(usage?.disabled ?? 0) + (usage?.archived ?? 0)}</span></div>
        </div>
      </Panel>

      <Panel title="Import and Registration" subtitle="Local GGUF and manifest-backed model registration. File deletion remains intentionally out of scope.">
        <FoldSection title="Register Local Model" subtitle="Import a GGUF file or manifest-backed directory into FORGE-managed runtime metadata." defaultOpen>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            <label className="text-xs text-forge-mist">
              Path
              <input className="forge-input mt-1" value={importPath} onChange={(e) => setImportPath(e.target.value)} placeholder="/models/coder.gguf" />
            </label>
            <label className="text-xs text-forge-mist">
              Display name
              <input className="forge-input mt-1" value={importDisplayName} onChange={(e) => setImportDisplayName(e.target.value)} placeholder="Qwen Coder" />
            </label>
            <label className="text-xs text-forge-mist">
              Family
              <input className="forge-input mt-1" value={importFamily} onChange={(e) => setImportFamily(e.target.value)} placeholder="qwen" />
            </label>
            <label className="text-xs text-forge-mist">
              Backend
              <select className="forge-input mt-1" value={importBackend} onChange={(e) => setImportBackend(e.target.value)}>
                <option value="">manifest/default</option>
                <option value="llama_cpp">llama_cpp</option>
                <option value="openai_compat">openai_compat</option>
                <option value="vllm">vllm</option>
                <option value="fake">fake</option>
              </select>
            </label>
            <label className="text-xs text-forge-mist">
              Capabilities
              <input className="forge-input mt-1" value={importCapabilities} onChange={(e) => setImportCapabilities(e.target.value)} placeholder="chat,completion" />
            </label>
            <label className="flex items-center gap-2 self-end text-xs text-forge-mist">
              <input type="checkbox" checked={importPreferred} onChange={(e) => setImportPreferred(e.target.checked)} />
              Mark as preferred
            </label>
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            <PrimaryButton onClick={() => void handleImport()} disabled={importBusy}>
              {importBusy ? "Importing..." : "Import Model"}
            </PrimaryButton>
            <GhostButton onClick={() => void handleScan()} disabled={scanBusy}>
              {scanBusy ? "Scanning..." : "Reconcile Registry"}
            </GhostButton>
          </div>
        </FoldSection>
      </Panel>

      <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <Panel title="Registered Models" subtitle="Model lifecycle state and operational controls.">
          {models.length === 0 ? (
            <div className="text-sm text-forge-mist">No models registered yet. Import a local model or scan an existing model home.</div>
          ) : (
            <div className="space-y-3">
              {models.map((model) => {
                const isSelected = model.id === selectedModelId;
                const loadedRecord = loaded?.models.find((item) => item.modelId === model.id) ?? null;
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
                      "w-full rounded border px-3 py-3 text-left transition",
                      isSelected ? "border-forge-accent/55 bg-forge-slate/30" : "border-white/10 bg-black/20 hover:border-forge-accent/40",
                    )}
                  >
                    <div className="flex flex-wrap items-start justify-between gap-2">
                      <div>
                        <div className="font-mono text-sm text-forge-ash">{model.id}</div>
                        <div className="mt-1 text-xs text-forge-mist">
                          {(model.displayName || "Unnamed model")} · {model.family || "family unknown"} · {model.backend || "backend unset"} · {model.format || "format unknown"}
                        </div>
                      </div>
                      <span className={cx("rounded-full border px-2 py-1 text-[11px] font-medium", badgeClass(model.status))}>{model.status || "unknown"}</span>
                    </div>
                    <div className="mt-2 text-[11px] text-forge-mist">Capabilities: {summarizeList(model.capabilities)}</div>
                    <div className="mt-3 flex flex-wrap gap-2">
                      <GhostButton
                        onClick={(event) => {
                          event.stopPropagation();
                          void runAction(model.id, "verify");
                        }}
                        disabled={isBusy}
                      >
                        {busyPrefix === "verify" && isBusy ? "Verifying..." : "Verify"}
                      </GhostButton>
                      {normalizeStatus(model.status) === "disabled" ? (
                        <GhostButton
                          onClick={(event) => {
                            event.stopPropagation();
                            void runAction(model.id, "enable");
                          }}
                          disabled={isBusy}
                        >
                          {busyPrefix === "enable" && isBusy ? "Enabling..." : "Enable"}
                        </GhostButton>
                      ) : (
                        <GhostButton
                          onClick={(event) => {
                            event.stopPropagation();
                            void runAction(model.id, "disable");
                          }}
                          disabled={isBusy}
                        >
                          {busyPrefix === "disable" && isBusy ? "Disabling..." : "Disable"}
                        </GhostButton>
                      )}
                      {loadedRecord ? (
                        <PrimaryButton
                          onClick={(event) => {
                            event.stopPropagation();
                            void runAction(model.id, "unload");
                          }}
                          disabled={isBusy}
                        >
                          {busyPrefix === "unload" && isBusy ? "Unloading..." : "Unload"}
                        </PrimaryButton>
                      ) : (
                        <PrimaryButton
                          onClick={(event) => {
                            event.stopPropagation();
                            void runAction(model.id, "load");
                          }}
                          disabled={isBusy || normalizeStatus(model.status) === "archived"}
                        >
                          {busyPrefix === "load" && isBusy ? "Loading..." : "Load"}
                        </PrimaryButton>
                      )}
                      <GhostButton
                        onClick={(event) => {
                          event.stopPropagation();
                          void runAction(model.id, "archive");
                        }}
                        disabled={isBusy || normalizeStatus(model.status) === "archived"}
                      >
                        {busyPrefix === "archive" && isBusy ? "Archiving..." : "Archive"}
                      </GhostButton>
                      <GhostButton
                        onClick={(event) => {
                          event.stopPropagation();
                          if (!window.confirm(`Remove registration for ${model.id}? Model files will not be deleted.`)) return;
                          void runAction(model.id, "remove");
                        }}
                        disabled={isBusy}
                      >
                        {busyPrefix === "remove" && isBusy ? "Removing..." : "Remove"}
                      </GhostButton>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </Panel>

        <div className="space-y-6">
          <Panel title="Selected Model" subtitle="Compatibility, loaded state, metadata, and backend readiness for the focused model.">
            {!selectedModel ? (
              <div className="text-sm text-forge-mist">Select a registered model to inspect compatibility and detailed metadata.</div>
            ) : (
              <div className="space-y-4">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <div className="font-mono text-sm text-forge-ash">{selectedModel.id}</div>
                    <div className="mt-1 text-xs text-forge-mist">
                      {(selectedModel.displayName || "Unnamed model")} · {selectedModel.family || "family unknown"} · {selectedModel.backend || "backend unset"}
                    </div>
                  </div>
                  <span className={cx("rounded-full border px-2 py-1 text-[11px] font-medium", badgeClass(selectedModel.status))}>
                    {selectedModel.status || "unknown"}
                  </span>
                </div>
                <div className="grid gap-2 text-[11px] text-forge-mist sm:grid-cols-2">
                  <div>Format: <span className="text-forge-ash">{selectedModel.format || "unknown"}</span></div>
                  <div>Capabilities: <span className="text-forge-ash">{summarizeList(selectedModel.capabilities)}</span></div>
                  <div>Loaded state: <span className="text-forge-ash">{selectedLoadedRecord?.status || "not loaded"}</span></div>
                  <div>Loaded at: <span className="text-forge-ash">{selectedLoadedRecord?.loadedAtMs ? formatTime(selectedLoadedRecord.loadedAtMs) : "—"}</span></div>
                </div>
                {compatibility ? (
                  <div className="rounded border border-white/10 bg-black/25 p-3 text-xs text-forge-mist">
                    <div className="font-semibold text-forge-ash">Compatibility</div>
                    <div className="mt-2 grid gap-2 sm:grid-cols-2">
                      <div>Backend healthy: <span className="text-forge-ash">{compatibility.backendHealthy ? "yes" : "no"}</span></div>
                      <div>Configured: <span className="text-forge-ash">{compatibility.backendConfigured ? "yes" : "no"}</span></div>
                      <div>Supported by backend: <span className="text-forge-ash">{compatibility.supportedByBackend ? "yes" : "no"}</span></div>
                      <div>Can generate: <span className="text-forge-ash">{compatibility.canGenerate ? "yes" : "no"}</span></div>
                      <div>Preferred: <span className="text-forge-ash">{compatibility.preferred ? "yes" : "no"}</span></div>
                      <div>Backend: <span className="text-forge-ash">{compatibility.backend || "unknown"}</span></div>
                    </div>
                    {compatibility.warnings && compatibility.warnings.length > 0 ? (
                      <div className="mt-3 rounded border border-forge-ember/30 bg-forge-ember/10 px-2 py-2 text-[11px] text-forge-ash">
                        Warnings: {compatibility.warnings.join(" · ")}
                      </div>
                    ) : null}
                    {compatibility.details && Object.keys(compatibility.details).length > 0 ? (
                      <pre className="mt-3 max-h-[220px] overflow-auto rounded border border-white/10 bg-black/30 p-3 text-[11px] text-forge-mist">
                        {JSON.stringify(compatibility.details, null, 2)}
                      </pre>
                    ) : null}
                  </div>
                ) : null}
                {selectedModel.metadata && Object.keys(selectedModel.metadata).length > 0 ? (
                  <FoldSection title="Metadata" subtitle="Manifest and registry metadata for this model.">
                    <pre className="max-h-[260px] overflow-auto rounded border border-white/10 bg-black/30 p-3 text-[11px] text-forge-mist">
                      {JSON.stringify(selectedModel.metadata, null, 2)}
                    </pre>
                  </FoldSection>
                ) : null}
              </div>
            )}
          </Panel>

          <Panel title="Runtime State" subtitle="Queue, loaded models, usage counters, and backend health exposed by the runtime itself.">
            <div className="space-y-4 text-xs text-forge-mist">
              <div className="grid gap-3 sm:grid-cols-2">
                <StateBox title="Queue" rows={[
                  ["Depth", String(queue?.depth ?? 0)],
                  ["Scheduler", queue?.scheduler || "—"],
                  ["Policy", queue?.policyState || "—"],
                  ["Running", String(usage?.running ?? 0)],
                ]} />
                <StateBox title="Usage" rows={[
                  ["Registered", String(usage?.registered ?? models.length)],
                  ["Imported", String(usage?.imported ?? 0)],
                  ["Verified", String(usage?.verified ?? 0)],
                  ["Completed", String(usage?.completed ?? 0)],
                ]} />
              </div>
              <div className="rounded border border-white/10 bg-black/25 p-3">
                <div className="font-semibold text-forge-ash">Loaded models</div>
                {loaded?.models.length ? (
                  <div className="mt-2 space-y-2">
                    {loaded.models.map((item) => (
                      <div key={`${item.backend}:${item.modelId}`} className="rounded border border-white/10 bg-black/20 px-3 py-2">
                        <div className="font-mono text-forge-ash">{item.modelId}</div>
                        <div className="mt-1 text-[11px] text-forge-mist">
                          {item.backend || "backend unknown"} · {item.status || "status unknown"} · loaded {item.loadedAtMs ? formatTime(item.loadedAtMs) : "—"}
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="mt-2 text-[11px] text-forge-mist">No loaded models.</div>
                )}
              </div>
              <div className="rounded border border-white/10 bg-black/25 p-3">
                <div className="font-semibold text-forge-ash">Backend health</div>
                {backends.length ? (
                  <div className="mt-2 space-y-2">
                    {backends.map((backend) => (
                      <div key={`${backend.kind}:${backend.name}`} className="rounded border border-white/10 bg-black/20 px-3 py-2">
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <div className="font-mono text-forge-ash">{backend.name}</div>
                          <span className={cx("rounded-full border px-2 py-1 text-[11px] font-medium", badgeClass(backend.healthy ? "healthy" : "error"))}>
                            {backend.healthy ? "healthy" : "unhealthy"}
                          </span>
                        </div>
                        <div className="mt-1 text-[11px] text-forge-mist">
                          {backend.kind} · {backend.loadedModel || "no loaded model"} · {backend.detail || "no extra detail"}
                        </div>
                        {backend.meta && Object.keys(backend.meta).length > 0 ? (
                          <div className="mt-2 grid gap-1 text-[11px] text-forge-mist md:grid-cols-2">
                            {Object.entries(backend.meta).map(([key, value]) => (
                              <div key={key}>
                                {key}: <span className="text-forge-ash">{summarizeValue(value)}</span>
                              </div>
                            ))}
                          </div>
                        ) : null}
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="mt-2 text-[11px] text-forge-mist">No backend health records returned.</div>
                )}
              </div>
            </div>
          </Panel>
        </div>
      </div>
    </div>
  );
}

function Metric(props: { label: string; value: string | number; hint: string }) {
  return (
    <div className="rounded border border-white/10 bg-black/20 px-3 py-3">
      <div className="text-[11px] uppercase tracking-wide text-forge-mist">{props.label}</div>
      <div className="mt-1 text-lg font-semibold text-forge-ash">{props.value}</div>
      <div className="mt-1 text-[11px] text-forge-mist">{props.hint}</div>
    </div>
  );
}

function StateBox(props: { title: string; rows: Array<[string, string]> }) {
  return (
    <div className="rounded border border-white/10 bg-black/25 p-3">
      <div className="font-semibold text-forge-ash">{props.title}</div>
      <div className="mt-2 grid gap-2">
        {props.rows.map(([label, value]) => (
          <div key={label} className="flex items-center justify-between gap-3 text-[11px] text-forge-mist">
            <span>{label}</span>
            <span className="text-right text-forge-ash">{value}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
