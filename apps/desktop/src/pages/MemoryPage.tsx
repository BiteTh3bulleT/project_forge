import type {
  MemoryObservation,
  MemoryObservationDetail,
  MemoryRepairRun,
  MemoryRepairRunDetail,
  ObservationVSADetail,
  SearchHit,
  VSAReindexRun,
  VSAReindexRunDetail,
} from "@forge/shared";
import { GhostButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { api } from "../lib/api";
import { useUiStore } from "../stores/uiStore";
import { MemoryControlsPanel } from "./MemoryPage/MemoryControlsPanel";
import { MemoryNoteComposer } from "./MemoryPage/MemoryNoteComposer";
import {
  ObservationDetailPanel,
  ObservationsPanel,
} from "./MemoryPage/ObservationPanels";
import {
  RepairRunDetailPanel,
  RepairRunsPanel,
} from "./MemoryPage/RepairPanels";
import { SearchResultsPanel } from "./MemoryPage/SearchResultsPanel";
import { MemoryMetric, type MemoryView } from "./MemoryPage/shared";
import { isOptionalEndpointMissing } from "./MemoryPage/utils";
import {
  VsaReindexDetailPanel,
  VsaReindexRunsPanel,
} from "./MemoryPage/VsaPanels";

export function MemoryPage() {
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const q = useMemo(() => (params.get("q") ?? "").trim(), [params]);
  const [localQ, setLocalQ] = useState(q);
  const [hits, setHits] = useState<SearchHit[] | null>(null);
  const [observations, setObservations] = useState<MemoryObservation[]>([]);
  const [selectedObsId, setSelectedObsId] = useState<number | null>(null);
  const [obsDetail, setObsDetail] = useState<MemoryObservationDetail | null>(
    null,
  );
  const [obsVSADetail, setObsVSADetail] = useState<ObservationVSADetail | null>(
    null,
  );
  const [obsType, setObsType] = useState("");
  const [dossierId, setDossierId] = useState("");
  const [staleOnly, setStaleOnly] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [status, setStatus] = useState<string>("");
  const [repairRuns, setRepairRuns] = useState<MemoryRepairRun[]>([]);
  const [selectedRepairId, setSelectedRepairId] = useState<number | null>(null);
  const [repairDetail, setRepairDetail] =
    useState<MemoryRepairRunDetail | null>(null);
  const [repairBusy, setRepairBusy] = useState(false);
  const [vsaRuns, setVSARuns] = useState<VSAReindexRun[]>([]);
  const [memoryView, setMemoryView] = useState<MemoryView>("inspect");
  const [selectedVSARunId, setSelectedVSARunId] = useState<number | null>(null);
  const [vsaRunDetail, setVSARunDetail] =
    useState<VSAReindexRunDetail | null>(null);
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
      const res = await api.memory.listRepairRuns({
        limit: 60,
        dossierId: Number.isFinite(did) ? did : undefined,
      });
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
      const res = await api.memory.listVSAReindexRuns({
        limit: 60,
        dossierId: Number.isFinite(did) ? did : undefined,
      });
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
          api.memory
            .getObservationVSA(selectedObsId)
            .catch((error: unknown) => {
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
      {
        label: "and optional dossier filter is valid when set",
        pass:
          dossierId.trim() === "" || Number.isFinite(Number(dossierId.trim())),
      },
    ],
    [dossierId, observations.length, repairBusy, vsaBusy],
  );
  const staleCount = useMemo(
    () => observations.filter((obs) => obs.stale).length,
    [observations],
  );
  const maintenanceBusyCount = (repairBusy ? 1 : 0) + (vsaBusy ? 1 : 0);

  const refreshAll = () => {
    void loadObservations();
    void loadRepairRuns();
    void loadVSARuns();
  };
  const runSearch = () => {
    const v = localQ.trim();
    if (v) setParams({ q: v });
    else setParams({});
  };
  const clearQuery = () => {
    setParams({});
    setLocalQ("");
    setHits(null);
  };

  return (
    <div className="forge-ops-board space-y-5">
      <header className="rounded border border-forge-platinum/10 bg-black/20 p-4 lg:flex lg:items-end lg:justify-between">
        <div className="min-w-0">
          <div className="forge-ops-label">Memory Operations</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Memory command board
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            Inspect observations, chunk retrieval, repair history, and VSA
            maintenance without changing canonical memory outside governed API
            calls.
          </p>
        </div>
        <div className="mt-4 flex flex-wrap items-center gap-2 lg:mt-0 lg:justify-end">
          <span
            className={
              maintenanceBusyCount > 0
                ? "forge-ops-status forge-ops-status--warn"
                : "forge-ops-status forge-ops-status--ok"
            }
          >
            {maintenanceBusyCount > 0 ? "maintenance running" : "idle"}
          </span>
          <GhostButton onClick={refreshAll}>Refresh</GhostButton>
        </div>
      </header>

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MemoryMetric
          label="Observations"
          value={observations.length}
          detail={`${staleCount} stale`}
          tone={staleCount > 0 ? "warn" : "ok"}
        />
        <MemoryMetric
          label="Chunk Hits"
          value={hits?.length ?? 0}
          detail={q ? `query: ${q}` : "no active query"}
          tone={hits ? "ok" : "muted"}
        />
        <MemoryMetric
          label="Repair Runs"
          value={repairRuns.length}
          detail={
            selectedRepairId ? `selected #${selectedRepairId}` : "none selected"
          }
          tone={repairBusy ? "warn" : "muted"}
        />
        <MemoryMetric
          label="VSA Runs"
          value={vsaRuns.length}
          detail={
            selectedVSARunId
              ? `selected #${selectedVSARunId}`
              : "optional endpoint"
          }
          tone={vsaBusy ? "warn" : "muted"}
        />
      </section>

      <MemoryControlsPanel
        memoryView={memoryView}
        setMemoryView={setMemoryView}
        localQ={localQ}
        setLocalQ={setLocalQ}
        obsType={obsType}
        setObsType={setObsType}
        dossierId={dossierId}
        setDossierId={setDossierId}
        staleOnly={staleOnly}
        setStaleOnly={setStaleOnly}
        maintenanceGates={maintenanceGates}
        err={err}
        onRefreshAll={refreshAll}
        onClearQuery={clearQuery}
        onRunSearch={runSearch}
        onApplyObservationFilters={() => void loadObservations()}
        onRefreshVSARuns={() => void loadVSARuns()}
      />

      {memoryView === "all" || memoryView === "inspect" ? (
        <MemoryNoteComposer
          dossierId={dossierId}
          loadObservations={loadObservations}
          setSelectedObsId={setSelectedObsId}
          setErr={setErr}
          setStatus={setStatus}
          setStatusLine={setStatusLine}
        />
      ) : null}

      {memoryView === "all" || memoryView === "inspect" ? (
        <div className="grid gap-6 xl:grid-cols-2">
          <ObservationsPanel
            observations={observations}
            selectedObsId={selectedObsId}
            setSelectedObsId={setSelectedObsId}
          />
          <ObservationDetailPanel
            obsDetail={obsDetail}
            obsVSADetail={obsVSADetail}
            setObsDetail={setObsDetail}
            setObsVSADetail={setObsVSADetail}
            status={status}
            setStatus={setStatus}
          />
        </div>
      ) : null}

      {(memoryView === "all" || memoryView === "search") && q && hits ? (
        <SearchResultsPanel
          hits={hits}
          navigateToChunk={(chunkId) => navigate(`/memory/chunk/${chunkId}`)}
        />
      ) : null}

      {memoryView === "all" || memoryView === "maintenance" ? (
        <div className="grid gap-6 xl:grid-cols-2">
          <RepairRunsPanel
            repairRuns={repairRuns}
            selectedRepairId={selectedRepairId}
            setSelectedRepairId={setSelectedRepairId}
            repairBusy={repairBusy}
            setRepairBusy={setRepairBusy}
            dossierId={dossierId}
            setStatus={setStatus}
            loadRepairRuns={loadRepairRuns}
            loadObservations={loadObservations}
          />
          <RepairRunDetailPanel repairDetail={repairDetail} />
        </div>
      ) : null}

      {memoryView === "all" || memoryView === "maintenance" ? (
        <div className="grid gap-6 xl:grid-cols-2">
          <VsaReindexRunsPanel
            vsaRuns={vsaRuns}
            selectedVSARunId={selectedVSARunId}
            setSelectedVSARunId={setSelectedVSARunId}
            vsaBusy={vsaBusy}
            setVSABusy={setVSABusy}
            dossierId={dossierId}
            staleOnly={staleOnly}
            selectedObsId={selectedObsId}
            setObsVSADetail={setObsVSADetail}
            setStatus={setStatus}
            setErr={setErr}
            loadVSARuns={loadVSARuns}
          />
          <VsaReindexDetailPanel vsaRunDetail={vsaRunDetail} />
        </div>
      ) : null}
    </div>
  );
}
