import type { RetrievalMode, RetrievalRun } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

const modes: RetrievalMode[] = ["keyword", "semantic", "hybrid"];

export function RetrievalRunsPage() {
  const [query, setQuery] = useState("project context execution pipeline");
  const [mode, setMode] = useState<RetrievalMode>("hybrid");
  const [dossierId, setDossierId] = useState("");
  const [runs, setRuns] = useState<RetrievalRun[]>([]);
  const [selectedRun, setSelectedRun] = useState<RetrievalRun | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [selectionByResult, setSelectionByResult] = useState<Record<number, Record<string, unknown>>>({});
  const [busy, setBusy] = useState(false);
  const setStatus = useUiStore((s) => s.setStatusLine);

  async function loadRuns() {
    try {
      const d = dossierId.trim() ? Number(dossierId.trim()) : undefined;
      const res = await api.retrieval.listRuns({ limit: 80, dossierId: Number.isFinite(d) ? d : undefined });
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
      return;
    }
    void (async () => {
      try {
        const selection = await api.memory.retrievalSelection(selectedRun.id);
        if (cancelled) return;
        const map: Record<number, Record<string, unknown>> = {};
        selection.selection.forEach((row) => {
          map[row.retrievalResultId] = row.reason ?? {};
        });
        setSelectionByResult(map);
      } catch {
        if (!cancelled) setSelectionByResult({});
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selectedRun?.id]);

  return (
    <div className="space-y-6">
      <Panel title="Retrieval Runs" subtitle="Keyword, semantic, and hybrid retrieval runs with persisted ranking evidence." actions={<GhostButton onClick={() => void loadRuns()}>Refresh</GhostButton>}>
        {err ? <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        <div className="grid gap-3 md:grid-cols-4">
          <div className="md:col-span-2">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Query</label>
            <input className="forge-input mt-1" value={query} onChange={(e) => setQuery(e.target.value)} />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Mode</label>
            <select className="forge-input mt-1" value={mode} onChange={(e) => setMode(e.target.value as RetrievalMode)}>
              {modes.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Dossier id (optional)</label>
            <input className="forge-input mt-1" value={dossierId} onChange={(e) => setDossierId(e.target.value)} placeholder="e.g. 1" />
          </div>
        </div>
        <div className="mt-3 flex gap-2">
          <PrimaryButton
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              try {
                const d = dossierId.trim() ? Number(dossierId.trim()) : undefined;
                const res = await api.retrieval.createRun({
                  query,
                  mode,
                  limit: 30,
                  selectForPacket: 8,
                  dossierId: Number.isFinite(d) ? d : undefined,
                });
                setStatus(`Retrieval run ${res.run.id} created (${res.run.mode}).`);
                await loadRuns();
                setSelectedRun(res.run);
                const selection = await api.memory.retrievalSelection(res.run.id);
                const map: Record<number, Record<string, unknown>> = {};
                selection.selection.forEach((row) => {
                  map[row.retrievalResultId] = row.reason ?? {};
                });
                setSelectionByResult(map);
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy ? "Running..." : "Run Retrieval"}
          </PrimaryButton>
        </div>
      </Panel>

      <div className="grid gap-6 xl:grid-cols-2">
        <Panel title="Run History" subtitle="Persisted retrieval evidence linked to jobs and packets.">
          {runs.length === 0 ? (
            <div className="text-sm text-forge-mist">No retrieval runs yet.</div>
          ) : (
            <div className="space-y-2">
              {runs.map((run) => (
                <button
                  key={run.id}
                  type="button"
                  className={[
                    "w-full rounded border px-3 py-2 text-left",
                    selectedRun?.id === run.id ? "border-forge-ember/40 bg-black/30" : "border-white/10 bg-black/20 hover:border-forge-ember/35",
                  ].join(" ")}
                  onClick={async () => {
                    const full = await api.retrieval.getRun(run.id);
                    setSelectedRun(full.run);
                    const selection = await api.memory.retrievalSelection(run.id);
                    const map: Record<number, Record<string, unknown>> = {};
                    selection.selection.forEach((row) => {
                      map[row.retrievalResultId] = row.reason ?? {};
                    });
                    setSelectionByResult(map);
                  }}
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="truncate text-sm font-semibold text-forge-ash">#{run.id} - {run.mode}</div>
                    <div className="text-[11px] text-forge-mist">{formatTime(run.createdAtMs)}</div>
                  </div>
                  <div className="mt-1 truncate text-xs text-forge-mist">{run.query}</div>
                  <div className="mt-1 text-[11px] text-forge-mist">results {run.results.length} | dossier {run.dossierId ?? "none"} | job {run.jobId ?? "none"}</div>
                </button>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="Run Detail" subtitle="Inspect ranking contributions and mark context usefulness.">
          {!selectedRun ? (
            <div className="text-sm text-forge-mist">Select a run.</div>
          ) : (
            <div className="space-y-2">
              <div className="rounded border border-white/10 bg-black/20 p-2 text-xs text-forge-mist">
                run #{selectedRun.id} | mode {selectedRun.mode} | packet {selectedRun.packetId ?? "none"} | job {selectedRun.jobId ?? "none"}
              </div>
              {selectedRun.results.length === 0 ? (
                <div className="text-sm text-forge-mist">No results in this run.</div>
              ) : (
                selectedRun.results.map((row) => (
                  <div key={row.id} className="rounded border border-white/10 bg-black/20 p-3">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="text-xs font-semibold text-forge-ash">rank {row.rankIndex + 1} - {row.relPath || row.absPath}</div>
                      <div className="text-[11px] text-forge-mist">selected {String(row.selectedForPacket)}</div>
                    </div>
                    <div className="mt-1 text-[11px] text-forge-mist">k {row.keywordScore.toFixed(3)} | s {row.semanticScore.toFixed(3)} | h {row.hybridScore.toFixed(3)}</div>
                    {row.observationId != null ? <div className="mt-1 text-[11px] text-forge-mist">observation #{row.observationId}</div> : null}
                    <div className="mt-2 text-xs text-forge-mist whitespace-pre-wrap">{row.snippet}</div>
                    {selectionByResult[row.id] ? (
                      <pre className="mt-2 overflow-x-auto rounded border border-white/10 bg-black/35 p-2 text-[11px] text-forge-mist">{JSON.stringify(selectionByResult[row.id], null, 2)}</pre>
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
                          className={row.usefulnessLabel === value ? "forge-btn forge-btn--primary" : "forge-btn forge-btn--ghost"}
                          onClick={async () => {
                            await api.retrieval.markUsefulness(row.id, {
                              label: value,
                              note: `Marked from retrieval UI at ${new Date().toISOString()}`,
                              jobId: selectedRun.jobId,
                              packetId: selectedRun.packetId,
                            });
                            const full = await api.retrieval.getRun(selectedRun.id);
                            setSelectedRun(full.run);
                            setStatus(`Usefulness updated for retrieval result ${row.id}.`);
                          }}
                        >
                          {label}
                        </button>
                      ))}
                    </div>
                  </div>
                ))
              )}
            </div>
          )}
        </Panel>
      </div>
    </div>
  );
}
