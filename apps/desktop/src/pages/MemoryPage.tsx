import type { MemoryObservation, SearchHit } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function MemoryPage() {
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const q = useMemo(() => (params.get("q") ?? "").trim(), [params]);
  const [localQ, setLocalQ] = useState(q);
  const [hits, setHits] = useState<SearchHit[] | null>(null);
  const [observations, setObservations] = useState<MemoryObservation[]>([]);
  const [selectedObsId, setSelectedObsId] = useState<number | null>(null);
  const [obsDetail, setObsDetail] = useState<Awaited<ReturnType<typeof api.memory.getObservation>>["observation"] | null>(null);
  const [obsType, setObsType] = useState("");
  const [dossierId, setDossierId] = useState("");
  const [staleOnly, setStaleOnly] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [status, setStatus] = useState<string>("");
  const [repairRuns, setRepairRuns] = useState<Array<{ id: number; mode: string; repaired: number; skipped: number; failed: number; candidates: number; createdAtMs: number; completedAtMs: number | null; note: string }>>([]);
  const [selectedRepairId, setSelectedRepairId] = useState<number | null>(null);
  const [repairDetail, setRepairDetail] = useState<Awaited<ReturnType<typeof api.memory.getRepairRun>>["detail"] | null>(null);
  const [repairBusy, setRepairBusy] = useState(false);
  const setStatusLine = useUiStore((s) => s.setStatusLine);

  useEffect(() => setLocalQ(q), [q]);

  async function loadObservations() {
    try {
      const did = dossierId.trim() ? Number(dossierId.trim()) : undefined;
      const res = await api.memory.listObservations({
        limit: 120,
        dossierId: Number.isFinite(did) ? did : undefined,
        type: obsType.trim() || undefined,
        staleOnly,
      });
      setObservations(res.observations);
      if (res.observations.length > 0 && selectedObsId == null) {
        setSelectedObsId(res.observations[0].id);
      }
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  async function loadRepairRuns() {
    try {
      const did = dossierId.trim() ? Number(dossierId.trim()) : undefined;
      const res = await api.memory.listRepairRuns({ limit: 60, dossierId: Number.isFinite(did) ? did : undefined });
      setRepairRuns(res.runs);
      if (res.runs.length > 0 && selectedRepairId == null) {
        setSelectedRepairId(res.runs[0].id);
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void loadObservations();
    void loadRepairRuns();
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!q) {
        setHits(null);
        return;
      }
      try {
        const res = await api.search(q, 60);
        if (cancelled) return;
        setHits(res.hits);
        setStatusLine(`Search finished: ${res.hits.length} hit(s).`);
      } catch (e) {
        if (cancelled) return;
        setHits(null);
        setErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [q, setStatusLine]);

  useEffect(() => {
    let cancelled = false;
    if (selectedObsId == null) {
      setObsDetail(null);
      return;
    }
    void (async () => {
      try {
        const detail = await api.memory.getObservation(selectedObsId);
        if (cancelled) return;
        setObsDetail(detail.observation);
      } catch {
        if (!cancelled) setObsDetail(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selectedObsId]);

  useEffect(() => {
    let cancelled = false;
    if (selectedRepairId == null) {
      setRepairDetail(null);
      return;
    }
    void (async () => {
      try {
        const detail = await api.memory.getRepairRun(selectedRepairId);
        if (cancelled) return;
        setRepairDetail(detail.detail);
      } catch {
        if (!cancelled) setRepairDetail(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selectedRepairId]);

  return (
    <div className="space-y-6">
      <Panel
        title="Memory Retrieval"
        subtitle="Hybrid evidence access. Search chunks, inspect observations, and mark usefulness/noise so retrieval quality improves over time."
        actions={
          <div className="flex gap-2">
            <GhostButton
              onClick={() => {
                void loadObservations();
                void loadRepairRuns();
              }}
            >
              Refresh observations
            </GhostButton>
            <GhostButton
              onClick={() => {
                setParams({});
                setLocalQ("");
                setHits(null);
              }}
            >
              Clear query
            </GhostButton>
          </div>
        }
      >
        <div className="grid gap-3 md:grid-cols-6">
          <div className="md:col-span-3">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Query</label>
            <input
              className="forge-input mt-1"
              value={localQ}
              onChange={(e) => setLocalQ(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  const v = localQ.trim();
                  if (v) setParams({ q: v });
                  else setParams({});
                }
              }}
              placeholder="search code and indexed content"
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Observation type</label>
            <input className="forge-input mt-1" value={obsType} onChange={(e) => setObsType(e.target.value)} placeholder="retrieval_result" />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Dossier id</label>
            <input className="forge-input mt-1" value={dossierId} onChange={(e) => setDossierId(e.target.value)} placeholder="optional" />
          </div>
          <div className="flex items-end gap-2">
            <label className="inline-flex items-center gap-2 text-xs text-forge-mist">
              <input type="checkbox" checked={staleOnly} onChange={(e) => setStaleOnly(e.target.checked)} />
              Stale only
            </label>
          </div>
        </div>
        <div className="mt-3 flex gap-2">
          <PrimaryButton
            onClick={() => {
              const v = localQ.trim();
              if (v) setParams({ q: v });
              else setParams({});
            }}
          >
            Run search
          </PrimaryButton>
          <GhostButton onClick={() => void loadObservations()}>Apply observation filters</GhostButton>
        </div>
        {err ? <div className="mt-3 rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-xs text-forge-ash">{err}</div> : null}
      </Panel>

      <div className="grid gap-6 xl:grid-cols-2">
        <Panel title="Observations" subtitle="Cold+warm memory records with structural metadata and staleness/usefulness state.">
          {observations.length === 0 ? (
            <div className="text-sm text-forge-mist">No observations match this filter.</div>
          ) : (
            <div className="space-y-2">
              {observations.map((obs) => (
                <button
                  key={obs.id}
                  type="button"
                  onClick={() => setSelectedObsId(obs.id)}
                  className={[
                    "w-full rounded border px-3 py-2 text-left",
                    selectedObsId === obs.id ? "border-forge-ember/40 bg-black/30" : "border-white/10 bg-black/20 hover:border-forge-ember/35",
                  ].join(" ")}
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="truncate text-xs font-semibold text-forge-ash">#{obs.id} · {obs.type}</div>
                    <div className="text-[11px] text-forge-mist">{formatTime(obs.observedAtMs)}</div>
                  </div>
                  <div className="mt-1 truncate text-[11px] text-forge-mist">{obs.summary || obs.sourcePath || "(no summary)"}</div>
                  <div className="mt-1 text-[11px] text-forge-mist">dossier {obs.dossierId ?? "none"} · useful {obs.usefulnessCount} · noise {obs.noiseCount} · stale {String(obs.stale)}</div>
                </button>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="Observation Detail" subtitle="Inspect lineage, links, and usefulness events. Mark stale/useful/noisy to repair memory drift.">
          {!obsDetail ? (
            <div className="text-sm text-forge-mist">Select an observation.</div>
          ) : (
            <div className="space-y-3">
              <div className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                <div>ID #{obsDetail.observation.id} · {obsDetail.observation.type} · verification {obsDetail.observation.verificationState}</div>
                <div className="mt-1">origin {obsDetail.observation.originKind || "none"}:{obsDetail.observation.originId || "none"}</div>
                <div className="mt-1">score {obsDetail.observation.usefulnessScore.toFixed(2)} · useful {obsDetail.observation.usefulnessCount} · noisy {obsDetail.observation.noiseCount}</div>
              </div>
              <div className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist whitespace-pre-wrap">{obsDetail.observation.rawContent || "(no raw content)"}</div>
              <div className="flex flex-wrap gap-2">
                {[
                  ["useful", "Useful"],
                  ["noisy", "Noisy"],
                  ["not_useful", "Not useful"],
                  ["insufficient", "Insufficient"],
                ].map(([value, label]) => (
                  <button
                    key={value}
                    type="button"
                    className="forge-btn forge-btn--ghost"
                    onClick={async () => {
                      await api.memory.markObservationUsefulness(obsDetail.observation.id, {
                        signal: value,
                        note: `Marked from memory page at ${new Date().toISOString()}`,
                      });
                      const detail = await api.memory.getObservation(obsDetail.observation.id);
                      setObsDetail(detail.observation);
                      setStatus(`Observation ${obsDetail.observation.id} marked ${value}.`);
                    }}
                  >
                    {label}
                  </button>
                ))}
                <button
                  type="button"
                  className="forge-btn forge-btn--ghost"
                  onClick={async () => {
                    await api.memory.patchObservation(obsDetail.observation.id, {
                      stale: !obsDetail.observation.stale,
                      lastVerifiedAtMs: Date.now(),
                    });
                    const detail = await api.memory.getObservation(obsDetail.observation.id);
                    setObsDetail(detail.observation);
                    setStatus(`Observation ${obsDetail.observation.id} stale=${String(detail.observation.observation.stale)}.`);
                  }}
                >
                  Toggle stale
                </button>
              </div>
              {status ? <div className="rounded border border-white/10 bg-black/20 p-2 text-xs text-forge-mist">{status}</div> : null}
            </div>
          )}
        </Panel>
      </div>

      {q && hits ? (
        <Panel title="Chunk Search Results" subtitle="Raw retrieval candidates from indexed source chunks.">
          {hits.length === 0 ? (
            <div className="text-sm text-forge-mist">No chunk hits.</div>
          ) : (
            <div className="space-y-2">
              {hits.slice(0, 40).map((h) => (
                <button
                  key={h.chunkId}
                  type="button"
                  onClick={() => navigate(`/memory/chunk/${h.chunkId}`)}
                  className="w-full rounded border border-white/10 bg-black/20 p-3 text-left hover:border-forge-ember/35"
                >
                  <div className="text-xs font-semibold text-forge-ash">{h.relPath || h.absPath}</div>
                  <div className="mt-1 text-[11px] text-forge-mist">chunk {h.chunkIndex} · score {h.score.toFixed(3)} · {formatTime(Math.floor(h.mtimeNs / 1_000_000))}</div>
                  <div className="mt-1 text-xs text-forge-mist whitespace-pre-wrap">{h.snippet || h.content.slice(0, 220)}</div>
                </button>
              ))}
            </div>
          )}
        </Panel>
      ) : null}

      <div className="grid gap-6 xl:grid-cols-2">
        <Panel title="Memory Repair Runs" subtitle="Drift correction runs with candidate/repair/skip/failure counts and persisted run history.">
          <div className="mb-3 flex gap-2">
            <PrimaryButton
              disabled={repairBusy}
              onClick={async () => {
                setRepairBusy(true);
                try {
                  const did = dossierId.trim() ? Number(dossierId.trim()) : undefined;
                  const res = await api.memory.runRepair({
                    dossierId: Number.isFinite(did) ? did : undefined,
                    maxAgeDays: 14,
                    limit: 120,
                    note: "Manual repair run from Memory page",
                  });
                  setStatus(`Repair run ${res.detail.run.id}: repaired ${res.detail.run.repaired}, skipped ${res.detail.run.skipped}, failed ${res.detail.run.failed}.`);
                  setSelectedRepairId(res.detail.run.id);
                  await loadRepairRuns();
                  await loadObservations();
                } finally {
                  setRepairBusy(false);
                }
              }}
            >
              {repairBusy ? "Running repair..." : "Run repair"}
            </PrimaryButton>
          </div>
          {repairRuns.length === 0 ? (
            <div className="text-sm text-forge-mist">No repair runs yet.</div>
          ) : (
            <div className="space-y-2">
              {repairRuns.map((run) => (
                <button
                  key={run.id}
                  type="button"
                  onClick={() => setSelectedRepairId(run.id)}
                  className={[
                    "w-full rounded border px-3 py-2 text-left",
                    selectedRepairId === run.id ? "border-forge-ember/40 bg-black/30" : "border-white/10 bg-black/20 hover:border-forge-ember/35",
                  ].join(" ")}
                >
                  <div className="text-xs font-semibold text-forge-ash">run #{run.id} · {run.mode}</div>
                  <div className="mt-1 text-[11px] text-forge-mist">
                    candidates {run.candidates} · repaired {run.repaired} · skipped {run.skipped} · failed {run.failed}
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">{formatTime(run.createdAtMs)} · {run.note || "no note"}</div>
                </button>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="Repair Run Detail" subtitle="Per-observation repair actions with before/after fields for inspectable drift correction.">
          {!repairDetail ? (
            <div className="text-sm text-forge-mist">Select a repair run.</div>
          ) : (
            <div className="space-y-2">
              <div className="rounded border border-white/10 bg-black/20 p-2 text-xs text-forge-mist">
                run #{repairDetail.run.id} · mode {repairDetail.run.mode} · repaired {repairDetail.run.repaired} / {repairDetail.run.candidates}
              </div>
              {repairDetail.items.length === 0 ? (
                <div className="text-sm text-forge-mist">No repair items recorded.</div>
              ) : (
                repairDetail.items.slice(0, 40).map((item) => (
                  <div key={item.id} className="rounded border border-white/10 bg-black/20 p-3">
                    <div className="text-xs font-semibold text-forge-ash">
                      item #{item.id} · obs {item.observationId} · {item.status}
                    </div>
                    <div className="mt-1 text-[11px] text-forge-mist">{item.issue} · {item.note}</div>
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
