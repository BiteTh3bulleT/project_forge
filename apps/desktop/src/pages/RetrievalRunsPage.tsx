import type {
  RetrievalMode,
  RetrievalResultVSASignal,
  RetrievalRun,
} from "@forge/shared";
import { GhostButton, PrimaryButton } from "@forge/ui";
import { useEffect, useState, type ReactNode } from "react";

import { HumanDataView } from "../components/HumanDataView";
import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

const modes: RetrievalMode[] = ["keyword", "semantic", "hybrid"];

function isOptionalEndpointMissing(error: unknown): boolean {
  const message =
    error instanceof Error
      ? error.message.toLowerCase()
      : String(error).toLowerCase();
  return message.includes("404") || message.includes("not found");
}

export function RetrievalRunsPage() {
  const [query, setQuery] = useState("project context execution pipeline");
  const [mode, setMode] = useState<RetrievalMode>("hybrid");
  const [dossierId, setDossierId] = useState("");
  const [runs, setRuns] = useState<RetrievalRun[]>([]);
  const [selectedRun, setSelectedRun] = useState<RetrievalRun | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [selectionByResult, setSelectionByResult] = useState<
    Record<number, Record<string, unknown>>
  >({});
  const [vsaByResult, setVsaByResult] = useState<
    Record<number, RetrievalResultVSASignal>
  >({});
  const [observationInfoById, setObservationInfoById] = useState<
    Record<
      number,
      {
        type: string;
        summary: string;
        sourcePath: string;
        usefulnessScore: number;
      }
    >
  >({});
  const [busy, setBusy] = useState(false);
  const setStatus = useUiStore((s) => s.setStatusLine);

  async function loadRuns() {
    try {
      const d = dossierId.trim() ? Number(dossierId.trim()) : undefined;
      const res = await api.retrieval.listRuns({
        limit: 80,
        dossierId: Number.isFinite(d) ? d : undefined,
      });
      setRuns(res.runs);
      if (res.runs.length > 0 && !selectedRun) {
        setSelectedRun(res.runs[0]);
      }
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void loadRuns();
  }, []);

  useEffect(() => {
    let cancelled = false;
    if (!selectedRun) {
      setSelectionByResult({});
      setVsaByResult({});
      setObservationInfoById({});
      return;
    }
    void (async () => {
      try {
        const [selection, vsaSignals] = await Promise.all([
          api.memory.retrievalSelection(selectedRun.id),
          api.retrieval
            .getRunVSASignals(selectedRun.id)
            .catch((error: unknown) => {
              if (isOptionalEndpointMissing(error)) {
                return { signals: [] as RetrievalResultVSASignal[] };
              }
              throw error;
            }),
        ]);
        if (cancelled) return;
        const map: Record<number, Record<string, unknown>> = {};
        selection.selection.forEach((row) => {
          map[row.retrievalResultId] = row.reason ?? {};
        });
        setSelectionByResult(map);
        const vsaMap: Record<number, RetrievalResultVSASignal> = {};
        (vsaSignals.signals ?? []).forEach((signal) => {
          vsaMap[signal.retrievalResultId] = signal;
        });
        setVsaByResult(vsaMap);

        const observationIds = new Set<number>();
        selectedRun.results.forEach((row) => {
          if (row.observationId != null) observationIds.add(row.observationId);
          const signalObservationId =
            row.vsaSignal?.observationId ??
            vsaMap[row.id]?.observationId ??
            null;
          if (signalObservationId != null)
            observationIds.add(signalObservationId);
        });
        const details = await Promise.allSettled(
          Array.from(observationIds)
            .slice(0, 30)
            .map(async (id) => {
              const detail = await api.memory.getObservation(id);
              return {
                id,
                type: detail.observation.observation.type,
                summary: detail.observation.observation.summary,
                sourcePath: detail.observation.observation.sourcePath,
                usefulnessScore: detail.observation.observation.usefulnessScore,
              };
            }),
        );
        if (cancelled) return;
        const observationMap: Record<
          number,
          {
            type: string;
            summary: string;
            sourcePath: string;
            usefulnessScore: number;
          }
        > = {};
        details.forEach((item) => {
          if (item.status === "fulfilled") {
            observationMap[item.value.id] = {
              type: item.value.type,
              summary: item.value.summary,
              sourcePath: item.value.sourcePath,
              usefulnessScore: item.value.usefulnessScore,
            };
          }
        });
        setObservationInfoById(observationMap);
      } catch {
        if (!cancelled) {
          setSelectionByResult({});
          setVsaByResult({});
          setObservationInfoById({});
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selectedRun]);

  return (
    <div className="forge-ops-board space-y-5">
      <header className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div className="forge-ops-label">Retrieval Evidence</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Retrieval runs board
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            {runs.length} persisted runs loaded. Ranking, selection, VSA, and
            usefulness signals are evidence for packet construction.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className={statusPillClass(busy ? "warn" : "ok")}>
            {busy ? "running" : "ready"}
          </span>
          <GhostButton onClick={() => void loadRuns()}>Refresh</GhostButton>
        </div>
      </header>

      {err ? (
        <div className="forge-ops-panel border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
          {err}
        </div>
      ) : null}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricTile
          label="Runs"
          value={String(runs.length)}
          detail="history rows"
          tone="muted"
        />
        <MetricTile
          label="Selected"
          value={selectedRun ? `#${selectedRun.id}` : "none"}
          detail={selectedRun?.mode ?? "no run selected"}
          tone={selectedRun ? "ok" : "muted"}
        />
        <MetricTile
          label="Results"
          value={String(selectedRun?.results.length ?? 0)}
          detail="selected run"
          tone="muted"
        />
        <MetricTile
          label="Mode"
          value={mode}
          detail="next retrieval"
          tone={busy ? "warn" : "ok"}
        />
      </section>

      <OpsPanel
        title="Retrieval Runs"
        subtitle="Keyword, semantic, and hybrid retrieval runs with persisted ranking evidence."
      >
        <div className="forge-ops-card p-3">
          <div className="grid gap-3 md:grid-cols-4">
            <div className="md:col-span-2">
              <label className="forge-ops-label">Query</label>
              <input
                className="forge-input mt-1"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
              />
            </div>
            <div>
              <label className="forge-ops-label">Mode</label>
              <select
                className="forge-input mt-1"
                value={mode}
                onChange={(e) => setMode(e.target.value as RetrievalMode)}
              >
                {modes.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="forge-ops-label">Dossier id (optional)</label>
              <input
                className="forge-input mt-1"
                value={dossierId}
                onChange={(e) => setDossierId(e.target.value)}
                placeholder="e.g. 1"
              />
            </div>
          </div>
          <div className="mt-3 flex gap-2">
            <PrimaryButton
              className="w-full sm:w-auto"
              disabled={busy}
              onClick={async () => {
                setBusy(true);
                try {
                  const d = dossierId.trim()
                    ? Number(dossierId.trim())
                    : undefined;
                  const res = await api.retrieval.createRun({
                    query,
                    mode,
                    limit: 30,
                    selectForPacket: 8,
                    dossierId: Number.isFinite(d) ? d : undefined,
                  });
                  setStatus(
                    `Retrieval run ${res.run.id} created (${res.run.mode}).`,
                  );
                  await loadRuns();
                  setSelectedRun(res.run);
                } finally {
                  setBusy(false);
                }
              }}
            >
              {busy ? "Running..." : "Run Retrieval"}
            </PrimaryButton>
          </div>
        </div>
      </OpsPanel>

      <div className="grid gap-4 xl:grid-cols-[minmax(18rem,0.7fr)_minmax(0,1.3fr)]">
        <OpsPanel
          title="Run History"
          subtitle="Persisted retrieval evidence linked to jobs and packets."
        >
          {runs.length === 0 ? (
            <EmptyState
              title="No retrieval runs yet"
              detail="Run a keyword, semantic, or hybrid query to create persisted retrieval evidence."
            />
          ) : (
            <div className="space-y-2">
              {runs.map((run) => (
                <button
                  key={run.id}
                  type="button"
                  className={[
                    "forge-ops-card w-full px-3 py-2 text-left",
                    selectedRun?.id === run.id
                      ? "border-forge-ember/40 bg-forge-ember/10"
                      : "hover:border-forge-ember/35",
                  ].join(" ")}
                  onClick={async () => {
                    const full = await api.retrieval.getRun(run.id);
                    setSelectedRun(full.run);
                  }}
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="truncate text-sm font-semibold text-forge-ash">
                      #{run.id} - {run.mode}
                    </div>
                    <div className="text-[11px] text-forge-mist">
                      {formatTime(run.createdAtMs)}
                    </div>
                  </div>
                  <div className="mt-1 truncate text-xs text-forge-mist">
                    {run.query}
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">
                    results {run.results.length} | dossier{" "}
                    {run.dossierId ?? "none"} | job {run.jobId ?? "none"}
                  </div>
                </button>
              ))}
            </div>
          )}
        </OpsPanel>

        <OpsPanel
          title="Run Detail"
          subtitle="Inspect ranking contributions and mark context usefulness."
        >
          {!selectedRun ? (
            <EmptyState
              title="Select a run"
              detail="Pick a retrieval run from history to inspect ranked snippets and VSA signals."
            />
          ) : (
            <div className="space-y-2">
              <div className="rounded border border-white/10 bg-black/20 p-2 text-xs text-forge-mist">
                run #{selectedRun.id} | mode {selectedRun.mode} | packet{" "}
                {selectedRun.packetId ?? "none"} | job{" "}
                {selectedRun.jobId ?? "none"}
              </div>
              {selectedRun.results.length === 0 ? (
                <EmptyState
                  title="No results in this run"
                  detail="The selected run completed without candidate evidence rows."
                />
              ) : (
                selectedRun.results.map((row) => {
                  const vsaSignal = row.vsaSignal ?? vsaByResult[row.id];
                  const matchedObservationId =
                    vsaSignal?.observationId ?? row.observationId ?? null;
                  const matchedObservation =
                    matchedObservationId != null
                      ? observationInfoById[matchedObservationId]
                      : undefined;
                  return (
                    <div key={row.id} className="forge-ops-card p-3">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="text-xs font-semibold text-forge-ash">
                          rank {row.rankIndex + 1} -{" "}
                          {row.relPath || row.absPath}
                        </div>
                        <div className="text-[11px] text-forge-mist">
                          selected {String(row.selectedForPacket)}
                        </div>
                      </div>
                      <div className="mt-2 grid gap-2 sm:grid-cols-3">
                        <MiniStat
                          label="keyword"
                          value={row.keywordScore.toFixed(3)}
                        />
                        <MiniStat
                          label="semantic"
                          value={row.semanticScore.toFixed(3)}
                        />
                        <MiniStat
                          label="hybrid"
                          value={row.hybridScore.toFixed(3)}
                        />
                      </div>
                      {vsaSignal ? (
                        <div className="mt-1 rounded border border-white/10 bg-black/30 px-2 py-1 text-[11px] text-forge-mist">
                          VSA {vsaSignal.mode || "off"} | assoc{" "}
                          {vsaSignal.associativeScore.toFixed(3)} | role{" "}
                          {vsaSignal.roleMatchScore.toFixed(3)} | rel{" "}
                          {vsaSignal.relationalScore.toFixed(3)} | feedback{" "}
                          {vsaSignal.feedbackScore.toFixed(3)} | add{" "}
                          {vsaSignal.additiveScore.toFixed(3)} | applied{" "}
                          {vsaSignal.appliedScore.toFixed(3)}
                        </div>
                      ) : null}
                      {matchedObservationId != null ? (
                        <div className="mt-1 text-[11px] text-forge-mist">
                          matched observation #{matchedObservationId}
                          {matchedObservation
                            ? ` · ${matchedObservation.type} · usefulness ${matchedObservation.usefulnessScore.toFixed(2)} · ${
                                matchedObservation.summary ||
                                matchedObservation.sourcePath ||
                                "no summary"
                              }`
                            : ""}
                        </div>
                      ) : null}
                      <div className="mt-2 text-xs text-forge-mist whitespace-pre-wrap">
                        {row.snippet}
                      </div>
                      {selectionByResult[row.id] ? (
                        <div className="mt-2 overflow-x-auto rounded border border-white/10 bg-black/35 p-2 text-[11px] text-forge-mist">
                          <HumanDataView
                            value={selectionByResult[row.id]}
                            compact
                          />
                        </div>
                      ) : null}
                      {vsaSignal?.explain ? (
                        <div className="mt-2 overflow-x-auto rounded border border-white/10 bg-black/35 p-2 text-[11px] text-forge-mist">
                          <HumanDataView value={vsaSignal.explain} compact />
                        </div>
                      ) : null}
                      <div className="mt-2 flex flex-wrap gap-2">
                        {[
                          ["useful", "Useful"],
                          ["not_useful", "Not Useful"],
                          ["noisy", "Noisy"],
                          ["insufficient", "Insufficient"],
                        ].map(([value, label]) => (
                          <button
                            key={value}
                            type="button"
                            className={
                              row.usefulnessLabel === value
                                ? "forge-btn forge-btn--primary"
                                : "forge-btn forge-btn--ghost"
                            }
                            onClick={async () => {
                              await api.retrieval.markUsefulness(row.id, {
                                label: value,
                                note: `Marked from retrieval UI at ${new Date().toISOString()}`,
                                jobId: selectedRun.jobId,
                                packetId: selectedRun.packetId,
                              });
                              const full = await api.retrieval.getRun(
                                selectedRun.id,
                              );
                              setSelectedRun(full.run);
                              setStatus(
                                `Usefulness updated for retrieval result ${row.id}.`,
                              );
                            }}
                          >
                            {label}
                          </button>
                        ))}
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          )}
        </OpsPanel>
      </div>
    </div>
  );
}

function OpsPanel(props: {
  title: string;
  subtitle: string;
  children: ReactNode;
}) {
  return (
    <section className="forge-ops-panel">
      <div className="forge-ops-panel__head">
        <div>
          <div className="forge-ops-title">{props.title}</div>
          <div className="mt-1 text-xs text-forge-mist/65">
            {props.subtitle}
          </div>
        </div>
      </div>
      <div className="forge-ops-panel__body">{props.children}</div>
    </section>
  );
}

function MetricTile(props: {
  label: string;
  value: string;
  detail: string;
  tone: string;
}) {
  return (
    <div className="forge-ops-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="forge-ops-label">{props.label}</div>
          <div className="mt-2 truncate text-2xl font-semibold tracking-normal text-forge-ash">
            {props.value}
          </div>
        </div>
        <span className={statusPillClass(props.tone)}>{props.tone}</span>
      </div>
      <div className="mt-3 text-xs text-forge-mist/65">{props.detail}</div>
    </div>
  );
}

function EmptyState(props: { title: string; detail: string }) {
  return (
    <div className="forge-ops-card border-dashed p-4 text-sm">
      <div className="font-semibold text-forge-ash">{props.title}</div>
      <div className="mt-1 text-xs leading-5 text-forge-mist/70">
        {props.detail}
      </div>
    </div>
  );
}

function MiniStat(props: { label: string; value: string }) {
  return (
    <div className="rounded border border-white/10 bg-black/25 px-2 py-1.5 text-[11px] text-forge-mist">
      <div className="forge-ops-label">{props.label}</div>
      <div className="mt-1 font-semibold text-forge-ash">{props.value}</div>
    </div>
  );
}

function statusPillClass(status: string) {
  if (status === "ok" || status === "useful") {
    return "forge-ops-status forge-ops-status--ok";
  }
  if (status === "bad" || status === "not_useful" || status === "noisy") {
    return "forge-ops-status forge-ops-status--bad";
  }
  if (status === "warn" || status === "insufficient") {
    return "forge-ops-status forge-ops-status--warn";
  }
  return "forge-ops-status forge-ops-status--muted";
}
