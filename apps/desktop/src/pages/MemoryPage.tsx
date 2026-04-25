import type { MemoryObservation, ObservationVSADetail, SearchHit, VSAReindexRun } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { FoldSection } from "../components/FoldSection";
import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

function isOptionalEndpointMissing(error: unknown): boolean {
  const message = error instanceof Error ? error.message.toLowerCase() : String(error).toLowerCase();
  return message.includes("404") || message.includes("not found");
}

export function MemoryPage() {
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const q = useMemo(() => (params.get("q") ?? "").trim(), [params]);
  const [localQ, setLocalQ] = useState(q);
  const [hits, setHits] = useState<SearchHit[] | null>(null);
  const [observations, setObservations] = useState<MemoryObservation[]>([]);
  const [selectedObsId, setSelectedObsId] = useState<number | null>(null);
  const [obsDetail, setObsDetail] = useState<Awaited<ReturnType<typeof api.memory.getObservation>>["observation"] | null>(null);
  const [obsVSADetail, setObsVSADetail] = useState<ObservationVSADetail | null>(null);
  const [obsType, setObsType] = useState("");
  const [dossierId, setDossierId] = useState("");
  const [staleOnly, setStaleOnly] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [status, setStatus] = useState<string>("");
  const [repairRuns, setRepairRuns] = useState<Array<{ id: number; mode: string; repaired: number; skipped: number; failed: number; candidates: number; createdAtMs: number; completedAtMs: number | null; note: string }>>([]);
  const [selectedRepairId, setSelectedRepairId] = useState<number | null>(null);
  const [repairDetail, setRepairDetail] = useState<Awaited<ReturnType<typeof api.memory.getRepairRun>>["detail"] | null>(null);
  const [repairBusy, setRepairBusy] = useState(false);
  const [vsaRuns, setVSARuns] = useState<VSAReindexRun[]>([]);
  const [memoryView, setMemoryView] = useState<"all" | "inspect" | "search" | "maintenance">("inspect");
  const [selectedVSARunId, setSelectedVSARunId] = useState<number | null>(null);
  const [vsaRunDetail, setVSARunDetail] = useState<Awaited<ReturnType<typeof api.memory.getVSAReindexRun>>["detail"] | null>(null);
  const [vsaBusy, setVSABusy] = useState(false);
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

  async function loadVSARuns() {
    try {
      const did = dossierId.trim() ? Number(dossierId.trim()) : undefined;
      const res = await api.memory.listVSAReindexRuns({ limit: 60, dossierId: Number.isFinite(did) ? did : undefined });
      setVSARuns(res.runs);
      if (res.runs.length > 0 && selectedVSARunId == null) {
        setSelectedVSARunId(res.runs[0].id);
      }
    } catch (e) {
      if (isOptionalEndpointMissing(e)) {
        setVSARuns([]);
        setSelectedVSARunId(null);
        setVSARunDetail(null);
        return;
      }
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void loadObservations();
    void loadRepairRuns();
    void loadVSARuns();
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
      setObsVSADetail(null);
      return;
    }
    void (async () => {
      try {
        const [detail, vsa] = await Promise.all([
          api.memory.getObservation(selectedObsId),
          api.memory.getObservationVSA(selectedObsId).catch((error: unknown) => {
            if (isOptionalEndpointMissing(error)) {
              return { detail: null as ObservationVSADetail | null };
            }
            throw error;
          }),
        ]);
        if (cancelled) return;
        setObsDetail(detail.observation);
        setObsVSADetail(vsa.detail ?? detail.observation.vsa ?? null);
      } catch {
        if (!cancelled) {
          setObsDetail(null);
          setObsVSADetail(null);
        }
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

  useEffect(() => {
    let cancelled = false;
    if (selectedVSARunId == null) {
      setVSARunDetail(null);
      return;
    }
    void (async () => {
      try {
        const detail = await api.memory.getVSAReindexRun(selectedVSARunId);
        if (cancelled) return;
        setVSARunDetail(detail.detail);
      } catch (e) {
        if (!cancelled) {
          if (isOptionalEndpointMissing(e)) {
            setVSARunDetail(null);
            return;
          }
          setErr(e instanceof Error ? e.message : String(e));
          setVSARunDetail(null);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selectedVSARunId]);

  const maintenanceGates = useMemo(
    () => [
      { label: "if observations are loaded", pass: observations.length > 0 },
      { label: "and repair run is not already executing", pass: !repairBusy },
      { label: "and VSA reindex is not already executing", pass: !vsaBusy },
      { label: "and optional dossier filter is valid when set", pass: dossierId.trim() === "" || Number.isFinite(Number(dossierId.trim())) },
    ],
    [dossierId, observations.length, repairBusy, vsaBusy],
  );

  return (
    <div className="space-y-6">
      <Panel
        title="Memory"
        subtitle="Cognitive memory view: recent episodes, important observations, and active maintenance paths. Details stay collapsed until selected."
        actions={
          <div className="flex gap-2">
            <label className="text-[11px] text-forge-mist">
              View
              <select className="forge-input ml-2 px-2 py-1 text-[11px]" value={memoryView} onChange={(e) => setMemoryView(e.target.value as "all" | "inspect" | "search" | "maintenance")}>
                <option value="inspect">Recent episodes</option>
                <option value="search">Search chunks</option>
                <option value="all">All surfaces</option>
                <option value="maintenance">Maintenance</option>
              </select>
            </label>
            <GhostButton
              onClick={() => {
                void loadObservations();
                void loadRepairRuns();
                void loadVSARuns();
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
        <FoldSection title="Search and filter scope" subtitle="Set query and observation filters before running actions." defaultOpen>
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
            <GhostButton onClick={() => void loadVSARuns()}>Refresh VSA runs</GhostButton>
          </div>
        </FoldSection>
        <FoldSection title="Maintenance preflight gates" subtitle="If/and checks before running repair or VSA maintenance.">
          <div className="space-y-1 rounded border border-white/10 bg-black/25 p-3 text-xs">
            {maintenanceGates.map((gate, idx) => (
              <GateLine key={gate.label} prefix={idx === 0 ? "IF" : "AND"} label={gate.label} pass={gate.pass} />
            ))}
          </div>
        </FoldSection>
        {err ? <div className="mt-3 rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-xs text-forge-ash">{err}</div> : null}
      </Panel>

      {(memoryView === "all" || memoryView === "inspect") ? (
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
              <div className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-forge-ash">VSA Detail</div>
                {!obsVSADetail?.pointer ? (
                  <div className="mt-2 text-[11px]">No VSA pointer indexed for this observation yet.</div>
                ) : (
                  <div className="mt-2 space-y-2">
                    <div className="text-[11px]">
                      pointer #{obsVSADetail.pointer.id} · dims {obsVSADetail.pointer.dims} · norm {obsVSADetail.pointer.norm.toFixed(4)} · stale{" "}
                      {String(obsVSADetail.pointer.stale)}
                    </div>
                    <div className="text-[11px]">fingerprint {obsVSADetail.pointer.sourceFingerprint || "none"}</div>
                    <div className="text-[11px]">
                      vector preview [{obsVSADetail.pointer.pointer.slice(0, 8).map((v) => v.toFixed(3)).join(", ")}
                      {obsVSADetail.pointer.pointer.length > 8 ? ", ..." : ""}]
                    </div>
                    <div className="rounded border border-white/10 bg-black/30 p-2">
                      <div className="text-[11px] font-semibold text-forge-ash">Role bindings ({obsVSADetail.roleBindings.length})</div>
                      {obsVSADetail.roleBindings.length === 0 ? (
                        <div className="mt-1 text-[11px]">No role bindings.</div>
                      ) : (
                        <div className="mt-1 space-y-1 text-[11px]">
                          {obsVSADetail.roleBindings.slice(0, 10).map((binding) => (
                            <div key={binding.id}>
                              {binding.role}={binding.filler} · w {binding.weight.toFixed(2)} · support {binding.supportCount} · noise {binding.noiseCount}
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                    <div className="rounded border border-white/10 bg-black/30 p-2">
                      <div className="text-[11px] font-semibold text-forge-ash">Associations ({obsVSADetail.associations.length})</div>
                      {obsVSADetail.associations.length === 0 ? (
                        <div className="mt-1 text-[11px]">No associations.</div>
                      ) : (
                        <div className="mt-1 space-y-1 text-[11px]">
                          {obsVSADetail.associations.slice(0, 10).map((association) => (
                            <div key={association.id}>
                              {association.fromObservationId} {"->"} {association.toObservationId} · {association.associationType} · strength{" "}
                              {association.strength.toFixed(3)} · support {association.supportCount} · noise {association.noiseCount}
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
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
                      const [detail, vsa] = await Promise.all([
                        api.memory.getObservation(obsDetail.observation.id),
                        api.memory.getObservationVSA(obsDetail.observation.id).catch(() => ({ detail: null as ObservationVSADetail | null })),
                      ]);
                      setObsDetail(detail.observation);
                      if (vsa.detail) {
                        setObsVSADetail(vsa.detail);
                      }
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
                    const [detail, vsa] = await Promise.all([
                      api.memory.getObservation(obsDetail.observation.id),
                      api.memory.getObservationVSA(obsDetail.observation.id).catch(() => ({ detail: null as ObservationVSADetail | null })),
                    ]);
                    setObsDetail(detail.observation);
                    if (vsa.detail) {
                      setObsVSADetail(vsa.detail);
                    }
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
      ) : null}

      {(memoryView === "all" || memoryView === "search") && q && hits ? (
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

      {(memoryView === "all" || memoryView === "maintenance") ? (
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
      ) : null}

      {(memoryView === "all" || memoryView === "maintenance") ? (
      <div className="grid gap-6 xl:grid-cols-2">
        <Panel title="VSA Reindex Runs" subtitle="VSA pointer/binding/association refresh runs with inspectable outcomes.">
          <div className="mb-3 flex flex-wrap gap-2">
            <PrimaryButton
              disabled={vsaBusy}
              onClick={async () => {
                setVSABusy(true);
                try {
                  const did = dossierId.trim() ? Number(dossierId.trim()) : undefined;
                  const res = await api.memory.runVSAReindex({
                    dossierId: Number.isFinite(did) ? did : undefined,
                    mode: "manual",
                    triggeredBy: "operator",
                    reason: "manual_reindex",
                    note: "Manual VSA reindex from Memory page",
                    limit: 150,
                    staleOnly,
                  });
                  setStatus(
                    `VSA reindex run ${res.detail.run.id}: indexed ${res.detail.run.indexed}, skipped ${res.detail.run.skipped}, failed ${res.detail.run.failed}.`,
                  );
                  setSelectedVSARunId(res.detail.run.id);
                  await loadVSARuns();
                  if (selectedObsId != null) {
                    const vsa = await api.memory.getObservationVSA(selectedObsId).catch(() => ({ detail: null as ObservationVSADetail | null }));
                    setObsVSADetail(vsa.detail ?? null);
                  }
                } catch (e) {
                  if (isOptionalEndpointMissing(e)) {
                    setStatus("VSA reindex endpoints are unavailable on this core build.");
                  } else {
                    setErr(e instanceof Error ? e.message : String(e));
                  }
                } finally {
                  setVSABusy(false);
                }
              }}
            >
              {vsaBusy ? "Running VSA reindex..." : "Run VSA reindex"}
            </PrimaryButton>
            <GhostButton onClick={() => void loadVSARuns()}>Refresh VSA runs</GhostButton>
          </div>
          {vsaRuns.length === 0 ? (
            <div className="text-sm text-forge-mist">No VSA reindex runs yet.</div>
          ) : (
            <div className="space-y-2">
              {vsaRuns.map((run) => (
                <button
                  key={run.id}
                  type="button"
                  onClick={() => setSelectedVSARunId(run.id)}
                  className={[
                    "w-full rounded border px-3 py-2 text-left",
                    selectedVSARunId === run.id ? "border-forge-ember/40 bg-black/30" : "border-white/10 bg-black/20 hover:border-forge-ember/35",
                  ].join(" ")}
                >
                  <div className="text-xs font-semibold text-forge-ash">run #{run.id} · {run.mode} · {run.status}</div>
                  <div className="mt-1 text-[11px] text-forge-mist">
                    candidates {run.candidates} · indexed {run.indexed} · skipped {run.skipped} · failed {run.failed}
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">
                    {formatTime(run.createdAtMs)} · dossier {run.dossierId ?? "all"} · by {run.triggeredBy || "operator"}
                  </div>
                </button>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="VSA Reindex Detail" subtitle="Per-observation VSA fingerprint transitions and indexing status.">
          {!vsaRunDetail ? (
            <div className="text-sm text-forge-mist">Select a VSA reindex run.</div>
          ) : (
            <div className="space-y-2">
              <div className="rounded border border-white/10 bg-black/20 p-2 text-xs text-forge-mist">
                run #{vsaRunDetail.run.id} · status {vsaRunDetail.run.status} · indexed {vsaRunDetail.run.indexed} / {vsaRunDetail.run.candidates}
              </div>
              {vsaRunDetail.items.length === 0 ? (
                <div className="text-sm text-forge-mist">No VSA reindex items recorded.</div>
              ) : (
                vsaRunDetail.items.slice(0, 40).map((item) => (
                  <div key={item.id} className="rounded border border-white/10 bg-black/20 p-3">
                    <div className="text-xs font-semibold text-forge-ash">
                      item #{item.id} · obs {item.observationId} · {item.status}
                    </div>
                    <div className="mt-1 text-[11px] text-forge-mist">
                      {item.reason || "n/a"} · before {item.beforeFingerprint || "none"} · after {item.afterFingerprint || "none"}
                    </div>
                    {item.note ? <div className="mt-1 text-[11px] text-forge-mist">{item.note}</div> : null}
                  </div>
                ))
              )}
            </div>
          )}
        </Panel>
      </div>
      ) : null}
    </div>
  );
}

function GateLine(props: { prefix: "IF" | "AND"; label: string; pass: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-white/5 pb-1 last:border-b-0 last:pb-0">
      <div className="font-mono text-[11px] text-forge-mist">
        <span className="mr-2 text-forge-mist/60">{props.prefix}</span>
        {props.label}
      </div>
      <div className={props.pass ? "text-[11px] font-semibold text-forge-electric" : "text-[11px] font-semibold text-forge-emberSoft"}>
        {props.pass ? "pass" : "fail"}
      </div>
    </div>
  );
}
