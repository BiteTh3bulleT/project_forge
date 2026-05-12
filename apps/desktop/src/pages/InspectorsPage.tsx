import type {
  PacketAlignmentNote,
  PacketGuidance,
  TaskPacket,
} from "@forge/shared";
import { GhostButton, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";

import { FoldSection } from "../components/FoldSection";
import {
  api,
  type AuditTraceLookupResponse,
  type ContextSnapshotInspectorDetail,
  type ContextSnapshotInspectorSummary,
  type DreamReportDetail,
  type DreamReportSummary,
  type ProcessHealthTraceResponse,
  type ProcessHealthCorrelationReport,
} from "../lib/api";
import { formatTime } from "../lib/format";
import { InspectorMetricsOverview } from "./InspectorsPage/InspectorMetricsOverview";
import {
  CountPill,
  EmptyState,
  JsonBlock,
  MetricChip,
  OperatorStepCard,
  Panel,
  SummaryLink,
  parseProcessRuntimeLine,
} from "./InspectorsPage/shared";
import {
  parseInspectorReportSummary,
  parseRestoreScoreSummary,
  parseResumeHintSummary,
} from "./InspectorsPage/inspectorsParsing";

export function InspectorsPage() {
  const [params, setParams] = useSearchParams();

  const [snapshotWorkspaceId, setSnapshotWorkspaceId] = useState(
    () => params.get("workspaceId") ?? "",
  );
  const [snapshotLaneId, setSnapshotLaneId] = useState(
    () => params.get("laneId") ?? "",
  );
  const [snapshotKind, setSnapshotKind] = useState(
    () => params.get("snapshotKind") ?? "",
  );
  const [snapshotQuery, setSnapshotQuery] = useState(
    () => params.get("snapshotQuery") ?? "",
  );
  const [snapshotCorrelationId, setSnapshotCorrelationId] = useState(
    () => params.get("correlationId") ?? "",
  );
  const [selectedSnapshotId, setSelectedSnapshotId] = useState(
    () => params.get("snapshotId") ?? "",
  );
  const [snapshots, setSnapshots] = useState<ContextSnapshotInspectorSummary[]>(
    [],
  );
  const [snapshotDetail, setSnapshotDetail] =
    useState<ContextSnapshotInspectorDetail | null>(null);
  const [snapshotBusy, setSnapshotBusy] = useState(false);
  const [snapshotErr, setSnapshotErr] = useState<string | null>(null);

  const [dreamWorkspaceId, setDreamWorkspaceId] = useState(
    () => params.get("dreamWorkspaceId") ?? params.get("workspaceId") ?? "",
  );
  const [dreamLaneId, setDreamLaneId] = useState(
    () => params.get("dreamLaneId") ?? params.get("laneId") ?? "",
  );
  const [dreamMode, setDreamMode] = useState(
    () => params.get("dreamMode") ?? "",
  );
  const [selectedDreamReportId, setSelectedDreamReportId] = useState(
    () => params.get("dreamReportId") ?? "",
  );
  const [dreamReports, setDreamReports] = useState<DreamReportSummary[]>([]);
  const [dreamReportDetail, setDreamReportDetail] =
    useState<DreamReportDetail | null>(null);
  const [dreamBusy, setDreamBusy] = useState(false);
  const [dreamErr, setDreamErr] = useState<string | null>(null);

  const [packetIdInput, setPacketIdInput] = useState(
    () => params.get("packetId") ?? "",
  );
  const [packetBusy, setPacketBusy] = useState(false);
  const [packetErr, setPacketErr] = useState<string | null>(null);
  const [packet, setPacket] = useState<TaskPacket | null>(null);
  const [packetAlignment, setPacketAlignment] = useState<PacketAlignmentNote[]>(
    [],
  );
  const [packetGuidance, setPacketGuidance] = useState<PacketGuidance[]>([]);

  const [correlationIdInput, setCorrelationIdInput] = useState(
    () => params.get("correlationId") ?? "",
  );
  const [traceIdInput, setTraceIdInput] = useState(
    () => params.get("traceId") ?? "",
  );
  const [traceBusy, setTraceBusy] = useState(false);
  const [traceErr, setTraceErr] = useState<string | null>(null);
  const [traceLookup, setTraceLookup] =
    useState<AuditTraceLookupResponse | null>(null);
  const [selectedTraceCorrelationId, setSelectedTraceCorrelationId] =
    useState("");

  const [processCorrelationIdInput, setProcessCorrelationIdInput] = useState(
    () => params.get("processCorrelationId") ?? "",
  );
  const [processTraceIdInput, setProcessTraceIdInput] = useState(
    () => params.get("processTraceId") ?? "",
  );
  const [processBusy, setProcessBusy] = useState(false);
  const [processErr, setProcessErr] = useState<string | null>(null);
  const [processLookup, setProcessLookup] =
    useState<ProcessHealthTraceResponse | null>(null);

  const selectedTraceReport = useMemo(() => {
    if (!traceLookup?.reports?.length) return null;
    return (
      traceLookup.reports.find(
        (item) => item.correlationId === selectedTraceCorrelationId,
      ) ?? traceLookup.reports[0]
    );
  }, [selectedTraceCorrelationId, traceLookup]);

  const parsedTraceSummary = useMemo(
    () => parseInspectorReportSummary(selectedTraceReport),
    [selectedTraceReport],
  );
  const processParsedRuntime = useMemo(() => {
    const reports: ProcessHealthCorrelationReport[] =
      processLookup?.reports ?? [];
    const totalProcessInvocations = reports.reduce(
      (acc, item) => acc + item.processInvocationCount,
      0,
    );
    const totalGatewayInvocations = reports.reduce(
      (acc, item) => acc + item.totalInvocations,
      0,
    );
    return { totalProcessInvocations, totalGatewayInvocations };
  }, [processLookup]);
  const restoreScoreSummary = useMemo(
    () => parseRestoreScoreSummary(snapshotDetail?.restoreScores),
    [snapshotDetail?.restoreScores],
  );
  const resumeHintSummary = useMemo(
    () => parseResumeHintSummary(snapshotDetail?.resumeHints),
    [snapshotDetail?.resumeHints],
  );

  function syncParams(next: Record<string, string>) {
    const merged = new URLSearchParams(params);
    for (const [key, value] of Object.entries(next)) {
      const trimmed = value.trim();
      if (!trimmed) merged.delete(key);
      else merged.set(key, trimmed);
    }
    setParams(merged, { replace: true });
  }

  async function loadSnapshots(
    preferredSnapshotId?: string,
    overrides?: Partial<{
      workspaceId: string;
      laneId: string;
      snapshotKind: string;
      snapshotQuery: string;
      correlationId: string;
    }>,
  ) {
    const effectiveWorkspaceId = overrides?.workspaceId ?? snapshotWorkspaceId;
    const effectiveLaneId = overrides?.laneId ?? snapshotLaneId;
    const effectiveSnapshotKind = overrides?.snapshotKind ?? snapshotKind;
    const effectiveSnapshotQuery = overrides?.snapshotQuery ?? snapshotQuery;
    const effectiveCorrelationId =
      overrides?.correlationId ?? snapshotCorrelationId;
    setSnapshotBusy(true);
    try {
      const res = await api.contextInspector.listSnapshots({
        limit: 60,
        workspaceId: effectiveWorkspaceId.trim() || undefined,
        laneId: effectiveLaneId.trim() || undefined,
        correlationId: effectiveCorrelationId.trim() || undefined,
        snapshotKind: effectiveSnapshotKind.trim() || undefined,
        query: effectiveSnapshotQuery.trim() || undefined,
      });
      const next = Array.isArray(res.snapshots) ? res.snapshots : [];
      setSnapshots(next);
      const nextSelectedId =
        preferredSnapshotId?.trim() ||
        selectedSnapshotId.trim() ||
        next[0]?.id ||
        "";
      if (nextSelectedId) {
        setSelectedSnapshotId(nextSelectedId);
        await loadSnapshotDetail(nextSelectedId);
      } else {
        setSnapshotDetail(null);
      }
      setSnapshotErr(null);
      syncParams({
        workspaceId: effectiveWorkspaceId,
        laneId: effectiveLaneId,
        snapshotKind: effectiveSnapshotKind,
        snapshotQuery: effectiveSnapshotQuery,
        snapshotId: nextSelectedId,
        correlationId: effectiveCorrelationId || correlationIdInput,
      });
    } catch (error) {
      setSnapshotErr(error instanceof Error ? error.message : String(error));
    } finally {
      setSnapshotBusy(false);
    }
  }

  async function loadSnapshotDetail(snapshotId: string) {
    const id = snapshotId.trim();
    if (!id) {
      setSnapshotDetail(null);
      return;
    }
    try {
      const res = await api.contextInspector.getSnapshot(id);
      setSnapshotDetail(res.snapshot);
      setSnapshotErr(null);
      syncParams({ snapshotId: id });
    } catch (error) {
      setSnapshotDetail(null);
      setSnapshotErr(error instanceof Error ? error.message : String(error));
    }
  }

  async function loadDreamReports(preferredReportId?: string) {
    const workspaceId = dreamWorkspaceId.trim();
    if (!workspaceId) {
      setDreamErr("Workspace id is required to inspect Dream reports.");
      return;
    }
    setDreamBusy(true);
    try {
      const res = await api.dreamReports.list({
        workspaceId,
        laneId: dreamLaneId.trim() || undefined,
        mode: dreamMode.trim() || undefined,
        limit: 60,
      });
      const next = Array.isArray(res.reports) ? res.reports : [];
      setDreamReports(next);
      const nextSelectedId =
        preferredReportId?.trim() ||
        selectedDreamReportId.trim() ||
        next[0]?.id ||
        "";
      if (nextSelectedId) {
        setSelectedDreamReportId(nextSelectedId);
        await loadDreamReportDetail(nextSelectedId, workspaceId, dreamLaneId);
      } else {
        setDreamReportDetail(null);
      }
      setDreamErr(null);
      syncParams({
        dreamWorkspaceId: workspaceId,
        dreamLaneId,
        dreamMode,
        dreamReportId: nextSelectedId,
      });
    } catch (error) {
      setDreamReports([]);
      setDreamReportDetail(null);
      setDreamErr(error instanceof Error ? error.message : String(error));
    } finally {
      setDreamBusy(false);
    }
  }

  async function loadDreamReportDetail(
    reportId: string,
    workspaceOverride?: string,
    laneOverride?: string,
  ) {
    const id = reportId.trim();
    const workspaceId = (workspaceOverride ?? dreamWorkspaceId).trim();
    const laneId = (laneOverride ?? dreamLaneId).trim();
    if (!id) {
      setDreamReportDetail(null);
      return;
    }
    if (!workspaceId) {
      setDreamErr("Workspace id is required to inspect Dream reports.");
      return;
    }
    try {
      const res = await api.dreamReports.get(id, {
        workspaceId,
        laneId: laneId || undefined,
      });
      setDreamReportDetail(res);
      setDreamErr(null);
      syncParams({
        dreamReportId: id,
        dreamWorkspaceId: workspaceId,
        dreamLaneId: laneId,
      });
    } catch (error) {
      setDreamReportDetail(null);
      setDreamErr(error instanceof Error ? error.message : String(error));
    }
  }

  async function loadPacketInspector(packetIdText = packetIdInput) {
    const packetId = Number(packetIdText.trim());
    if (!Number.isFinite(packetId) || packetId <= 0) {
      setPacketErr("Packet id must be a positive integer.");
      return;
    }
    setPacketBusy(true);
    try {
      const [packetRes, alignmentRes, guidanceRes] = await Promise.all([
        api.packets.get(packetId),
        api.memory.packetAlignment(packetId, 80),
        api.packetGuidance.list({ limit: 20, packetId }),
      ]);
      setPacket(packetRes);
      setPacketAlignment(
        Array.isArray(alignmentRes.notes) ? alignmentRes.notes : [],
      );
      setPacketGuidance(
        Array.isArray(guidanceRes.guidance) ? guidanceRes.guidance : [],
      );
      setPacketErr(null);
      syncParams({ packetId: String(packetId) });
    } catch (error) {
      setPacketErr(error instanceof Error ? error.message : String(error));
      setPacket(null);
      setPacketAlignment([]);
      setPacketGuidance([]);
    } finally {
      setPacketBusy(false);
    }
  }

  async function loadTraceInspector(next?: {
    correlationId?: string;
    traceId?: string;
  }) {
    const correlationId = next?.correlationId ?? correlationIdInput;
    const traceId = next?.traceId ?? traceIdInput;
    if (!correlationId.trim() && !traceId.trim()) {
      setTraceErr("Provide a correlation id or trace id.");
      return;
    }
    setTraceBusy(true);
    try {
      const res = await api.audit.lookup({
        correlationId: correlationId.trim() || undefined,
        traceId: traceId.trim() || undefined,
      });
      setTraceLookup(res);
      setSelectedTraceCorrelationId(res.reports[0]?.correlationId ?? "");
      setTraceErr(null);
      syncParams({ correlationId, traceId });
    } catch (error) {
      setTraceLookup(null);
      setSelectedTraceCorrelationId("");
      setTraceErr(error instanceof Error ? error.message : String(error));
    } finally {
      setTraceBusy(false);
    }
  }

  async function loadProcessHealthInspector(next?: {
    correlationId?: string;
    traceId?: string;
  }) {
    const correlationId = next?.correlationId ?? processCorrelationIdInput;
    const traceId = next?.traceId ?? processTraceIdInput;
    if (!correlationId.trim() && !traceId.trim()) {
      setProcessErr("Provide a correlation id or trace id.");
      return;
    }
    setProcessBusy(true);
    try {
      const res = await api.processHealth({
        correlationId: correlationId.trim() || undefined,
        traceId: traceId.trim() || undefined,
      });
      setProcessLookup(res);
      setProcessErr(null);
      syncParams({
        processCorrelationId: correlationId,
        processTraceId: traceId,
      });
    } catch (error) {
      setProcessLookup(null);
      setProcessErr(error instanceof Error ? error.message : String(error));
    } finally {
      setProcessBusy(false);
    }
  }

  useEffect(() => {
    void loadSnapshots(selectedSnapshotId);
  }, []);

  useEffect(() => {
    if (dreamWorkspaceId.trim()) {
      void loadDreamReports(selectedDreamReportId);
    }
  }, []);

  useEffect(() => {
    if (packetIdInput.trim()) {
      void loadPacketInspector(packetIdInput);
    }
  }, []);

  useEffect(() => {
    if (correlationIdInput.trim() || traceIdInput.trim()) {
      void loadTraceInspector({
        correlationId: correlationIdInput,
        traceId: traceIdInput,
      });
    }
  }, []);

  useEffect(() => {
    if (processCorrelationIdInput.trim() || processTraceIdInput.trim()) {
      void loadProcessHealthInspector({
        correlationId: processCorrelationIdInput,
        traceId: processTraceIdInput,
      });
    }
  }, []);

  const packetRetrievedCount = packet?.retrievedContext?.length ?? 0;
  const packetReferenceCount = packet?.sourceReferences?.length ?? 0;
  const traceReportCount = traceLookup?.reports?.length ?? 0;
  const processReportCount = processLookup?.reports?.length ?? 0;
  const busyCount =
    (snapshotBusy ? 1 : 0) +
    (dreamBusy ? 1 : 0) +
    (packetBusy ? 1 : 0) +
    (traceBusy ? 1 : 0) +
    (processBusy ? 1 : 0);
  const errorCount = [
    snapshotErr,
    dreamErr,
    packetErr,
    traceErr,
    processErr,
  ].filter(Boolean).length;

  return (
    <div className="forge-ops-board space-y-5">
      <header className="rounded border border-forge-platinum/10 bg-black/20 p-4 lg:flex lg:items-end lg:justify-between">
        <div className="min-w-0">
          <div className="forge-ops-label">Diagnostics</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Inspector command board
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            Read-only evidence surfaces for snapshots, packets, Dream reports,
            audit traces, process invocations, and runtime health.
          </p>
        </div>
        <div className="mt-4 flex flex-wrap items-center gap-2 lg:mt-0 lg:justify-end">
          <span
            className={
              errorCount > 0
                ? "forge-ops-status forge-ops-status--bad"
                : busyCount > 0
                  ? "forge-ops-status forge-ops-status--warn"
                  : "forge-ops-status forge-ops-status--ok"
            }
          >
            {errorCount > 0
              ? `${errorCount} errors`
              : busyCount > 0
                ? `${busyCount} loading`
                : "clear"}
          </span>
          <GhostButton
            onClick={() =>
              void Promise.all([
                loadSnapshots(selectedSnapshotId),
                dreamWorkspaceId
                  ? loadDreamReports(selectedDreamReportId)
                  : Promise.resolve(),
                correlationIdInput || traceIdInput
                  ? loadTraceInspector()
                  : Promise.resolve(),
                processCorrelationIdInput || processTraceIdInput
                  ? loadProcessHealthInspector()
                  : Promise.resolve(),
              ])
            }
          >
            Refresh
          </GhostButton>
        </div>
      </header>

      <InspectorMetricsOverview
        snapshotsCount={snapshots.length}
        selectedSnapshotId={selectedSnapshotId}
        snapshotTone={snapshotErr ? "bad" : snapshotBusy ? "warn" : "muted"}
        dreamReportsCount={dreamReports.length}
        selectedDreamReportId={selectedDreamReportId}
        dreamTone={dreamErr ? "bad" : dreamBusy ? "warn" : "muted"}
        packetId={packet?.id ?? "none"}
        packetDetail={`${packetRetrievedCount} retrieved / ${packetReferenceCount} refs`}
        packetTone={
          packetErr ? "bad" : packetBusy ? "warn" : packet ? "ok" : "muted"
        }
        traceReportCount={traceReportCount}
        traceMode={traceLookup?.mode ?? "idle"}
        traceTone={
          traceErr ? "bad" : traceBusy ? "warn" : traceLookup ? "ok" : "muted"
        }
        processReportCount={processReportCount}
        processRuntimeState={processLookup?.runtime?.state || "runtime unknown"}
        processTone={
          processErr
            ? "bad"
            : processBusy
              ? "warn"
              : processLookup
                ? "ok"
                : "muted"
        }
      />

      <Panel
        title="Inspector Scope"
        subtitle="Read-only evidence views for context snapshots, packet materialization, and correlation traces."
        actions={
          <GhostButton
            onClick={() =>
              void Promise.all([
                loadSnapshots(selectedSnapshotId),
                dreamWorkspaceId
                  ? loadDreamReports(selectedDreamReportId)
                  : Promise.resolve(),
                correlationIdInput || traceIdInput
                  ? loadTraceInspector()
                  : Promise.resolve(),
                processCorrelationIdInput || processTraceIdInput
                  ? loadProcessHealthInspector()
                  : Promise.resolve(),
              ])
            }
          >
            Refresh views
          </GhostButton>
        }
      >
        <div className="rounded border border-forge-electric/20 bg-forge-electric/5 p-3 text-sm text-forge-mist">
          These surfaces expose persisted evidence only. They do not execute
          tools, mutate truth, or bypass approvals.
        </div>
        <div className="mt-4 grid gap-3 lg:grid-cols-4">
          <OperatorStepCard
            step="01"
            title="Locate the snapshot"
            detail="Start with context snapshots when you need to see what scope, restore hints, and evidence counts were materialized."
          />
          <OperatorStepCard
            step="02"
            title="Check the packet"
            detail="Use packet inspection to confirm the request contract, retrieved evidence, and quality guidance that reached execution."
          />
          <OperatorStepCard
            step="03"
            title="Inspect Dream evidence"
            detail="Review dry-run reports, replay candidates, salience scores, proposed memory-tier moves, and warnings as non-canonical evidence."
          />
          <OperatorStepCard
            step="04"
            title="Trace process/runtime"
            detail="Follow correlation or trace ids to audit rows, process invocations, and model-runtime health in one flow."
          />
        </div>
      </Panel>

      <Panel
        title="Snapshot Inspector"
        subtitle="Inspect persisted context compilation snapshots, restore scoring, and resume hints."
      >
        <div className="mb-4 flex flex-wrap gap-2">
          <CountPill label="Snapshots" value={snapshots.length} />
          <CountPill label="Selected" value={selectedSnapshotId || "none"} />
          <CountPill
            label="Correlation filter"
            value={snapshotCorrelationId || "—"}
          />
        </div>
        <div className="grid gap-2 md:grid-cols-5">
          <input
            className="forge-input"
            placeholder="workspace id"
            value={snapshotWorkspaceId}
            onChange={(e) => setSnapshotWorkspaceId(e.target.value)}
          />
          <input
            className="forge-input"
            placeholder="lane id"
            value={snapshotLaneId}
            onChange={(e) => setSnapshotLaneId(e.target.value)}
          />
          <input
            className="forge-input"
            placeholder="snapshot kind"
            value={snapshotKind}
            onChange={(e) => setSnapshotKind(e.target.value)}
          />
          <input
            className="forge-input"
            placeholder="query filter"
            value={snapshotQuery}
            onChange={(e) => setSnapshotQuery(e.target.value)}
          />
          <input
            className="forge-input"
            placeholder="correlation id"
            value={snapshotCorrelationId}
            onChange={(e) => setSnapshotCorrelationId(e.target.value)}
          />
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton
            disabled={snapshotBusy}
            onClick={() => void loadSnapshots()}
          >
            {snapshotBusy ? "Loading…" : "Load snapshots"}
          </PrimaryButton>
          <GhostButton
            onClick={() => {
              setSnapshotWorkspaceId("");
              setSnapshotLaneId("");
              setSnapshotKind("");
              setSnapshotQuery("");
              setSnapshotCorrelationId("");
              setSelectedSnapshotId("");
              setSnapshotDetail(null);
              void loadSnapshots("", {
                workspaceId: "",
                laneId: "",
                snapshotKind: "",
                snapshotQuery: "",
                correlationId: "",
              });
            }}
          >
            Clear filters
          </GhostButton>
        </div>
        {snapshotErr ? (
          <div className="mt-3">
            <EmptyState
              title="Snapshot lookup failed"
              detail={snapshotErr}
              tone="bad"
            />
          </div>
        ) : null}
        <div className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
          <div className="space-y-2">
            {snapshots.length === 0 ? (
              <EmptyState
                title={
                  snapshotBusy ? "Loading snapshots" : "No snapshots matched"
                }
                detail={
                  snapshotBusy
                    ? "Fetching persisted context snapshots for the current inspector scope."
                    : "Adjust workspace, lane, kind, query, or correlation filters and load snapshots again."
                }
              />
            ) : null}
            {snapshots.map((snapshot) => (
              <button
                key={snapshot.id}
                type="button"
                onClick={() => {
                  setSelectedSnapshotId(snapshot.id);
                  void loadSnapshotDetail(snapshot.id);
                }}
                className={[
                  "w-full rounded border px-3 py-3 text-left transition",
                  selectedSnapshotId === snapshot.id
                    ? "border-white/20 bg-white/10"
                    : "border-white/10 bg-black/20 hover:border-white/20",
                ].join(" ")}
              >
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div>
                    <div className="break-words text-sm font-semibold text-forge-ash">
                      {snapshot.query || snapshot.id}
                    </div>
                    <div className="mt-1 break-all font-mono text-[11px] text-forge-mist/80">
                      {snapshot.id}
                    </div>
                  </div>
                  <div className="text-[11px] text-forge-mist/75">
                    {formatTime(snapshot.createdAtMs)}
                  </div>
                </div>
                <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-forge-mist/80">
                  <span>{snapshot.snapshotKind || "kind: —"}</span>
                  <span>{snapshot.workspaceId || "workspace: —"}</span>
                  <span>{snapshot.laneId || "lane: —"}</span>
                </div>
                <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-forge-mist/80">
                  {snapshot.correlationId ? (
                    <span className="break-all font-mono">
                      corr {snapshot.correlationId}
                    </span>
                  ) : (
                    <span>corr —</span>
                  )}
                  {snapshot.traceId ? (
                    <span className="break-all font-mono">
                      trace {snapshot.traceId}
                    </span>
                  ) : (
                    <span>trace —</span>
                  )}
                </div>
                <div className="mt-2 grid grid-cols-3 gap-2 text-[11px] text-forge-mist/80">
                  <span>notes {snapshot.counts.notes}</span>
                  <span>artifacts {snapshot.counts.artifacts}</span>
                  <span>events {snapshot.counts.events}</span>
                </div>
              </button>
            ))}
          </div>

          <div className="space-y-4">
            {!snapshotDetail ? (
              <EmptyState
                title="Select a snapshot"
                detail="Choose a snapshot row to inspect persisted evidence, restore scoring, resume hints, graph, delta, and included ids."
              />
            ) : (
              <>
                <div className="grid gap-2 md:grid-cols-3">
                  <MetricChip
                    label="Kind"
                    value={snapshotDetail.summary.snapshotKind || "—"}
                  />
                  <MetricChip
                    label="Workspace"
                    value={snapshotDetail.summary.workspaceId || "—"}
                  />
                  <MetricChip
                    label="Lane"
                    value={snapshotDetail.summary.laneId || "—"}
                  />
                </div>
                <div className="flex flex-wrap gap-2">
                  <CountPill
                    label="Evidence"
                    value={
                      snapshotDetail.summary.evidenceClass ||
                      "non_canonical_evidence"
                    }
                  />
                  <CountPill
                    label="Canonical commit"
                    value={
                      snapshotDetail.summary.nonCanonicalEvidence
                        ? "no"
                        : "unknown"
                    }
                  />
                </div>
                <div className="rounded border border-white/10 bg-black/20 p-3 text-sm text-forge-mist">
                  <div className="break-words font-medium text-forge-ash">
                    {snapshotDetail.summary.query || snapshotDetail.summary.id}
                  </div>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {snapshotDetail.summary.correlationId ? (
                      <SummaryLink
                        to={`/inspectors?correlationId=${encodeURIComponent(snapshotDetail.summary.correlationId)}&snapshotId=${encodeURIComponent(snapshotDetail.summary.id)}`}
                        label={`Trace ${snapshotDetail.summary.correlationId}`}
                      />
                    ) : null}
                    {snapshotDetail.summary.parentSnapshotId ? (
                      <button
                        type="button"
                        className="rounded border border-white/15 bg-white/5 px-2.5 py-1 text-[11px] text-forge-mist transition hover:text-forge-ash"
                        onClick={() => {
                          setSelectedSnapshotId(
                            snapshotDetail.summary.parentSnapshotId,
                          );
                          void loadSnapshotDetail(
                            snapshotDetail.summary.parentSnapshotId,
                          );
                        }}
                      >
                        Open parent {snapshotDetail.summary.parentSnapshotId}
                      </button>
                    ) : null}
                  </div>
                  <div className="mt-3 grid gap-2 md:grid-cols-2">
                    <div>
                      <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">
                        Fingerprint
                      </div>
                      <div className="mt-1 break-all font-mono text-[11px] text-forge-ash">
                        {snapshotDetail.summary.snapshotFingerprint || "—"}
                      </div>
                    </div>
                    <div>
                      <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">
                        Render Artifact Ref
                      </div>
                      <div className="mt-1 break-all font-mono text-[11px] text-forge-ash">
                        {snapshotDetail.summary.renderArtifactRefId || "—"}
                      </div>
                    </div>
                  </div>
                </div>

                <FoldSection
                  title="Selection & Counts"
                  subtitle="Scope paths, syscall lineage, and included evidence counts."
                  defaultOpen
                >
                  <div className="grid gap-3 md:grid-cols-2">
                    <div>
                      <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">
                        Selected Paths
                      </div>
                      <div className="mt-2 flex flex-wrap gap-2">
                        {snapshotDetail.summary.selectedPaths.length === 0 ? (
                          <span className="text-xs text-forge-mist/75">
                            No explicit path scope recorded.
                          </span>
                        ) : (
                          snapshotDetail.summary.selectedPaths.map((path) => (
                            <span
                              key={path}
                              className="min-w-0 break-all rounded border border-white/10 bg-black/25 px-2 py-1 font-mono text-[11px] text-forge-ash"
                            >
                              {path}
                            </span>
                          ))
                        )}
                      </div>
                    </div>
                    <div className="grid gap-2 grid-cols-2">
                      <MetricChip
                        label="State"
                        value={snapshotDetail.summary.counts.state}
                      />
                      <MetricChip
                        label="Open loops"
                        value={snapshotDetail.summary.counts.openLoops}
                      />
                      <MetricChip
                        label="Notes"
                        value={snapshotDetail.summary.counts.notes}
                      />
                      <MetricChip
                        label="Artifacts"
                        value={snapshotDetail.summary.counts.artifacts}
                      />
                    </div>
                  </div>
                  <div className="mt-3 grid gap-2 md:grid-cols-3">
                    <MetricChip
                      label="Correlation"
                      value={snapshotDetail.summary.correlationId || "—"}
                    />
                    <MetricChip
                      label="Trace"
                      value={snapshotDetail.summary.traceId || "—"}
                    />
                    <MetricChip
                      label="Syscall"
                      value={snapshotDetail.summary.syscallId || "—"}
                    />
                  </div>
                </FoldSection>

                <FoldSection
                  title="Budget & Inclusion"
                  subtitle="Persisted packet budget and inclusion reasons."
                  defaultOpen
                >
                  <div className="grid gap-4 md:grid-cols-2">
                    <JsonBlock
                      value={snapshotDetail.budget}
                      empty="No budget recorded."
                    />
                    <JsonBlock
                      value={snapshotDetail.inclusionReasons}
                      empty="No inclusion reasons recorded."
                    />
                  </div>
                </FoldSection>

                <FoldSection
                  title="Restore Scoring"
                  subtitle="Non-canonical restore scores and resume hints carried with the snapshot."
                  defaultOpen
                >
                  <div className="space-y-4">
                    <div className="grid gap-3 md:grid-cols-3">
                      <MetricChip
                        label="Decision"
                        value={restoreScoreSummary.decision || "—"}
                      />
                      <MetricChip
                        label="Threshold"
                        value={restoreScoreSummary.threshold}
                      />
                      <MetricChip
                        label="Top Score"
                        value={restoreScoreSummary.topScore.toFixed(3)}
                      />
                      <MetricChip
                        label="Candidates"
                        value={restoreScoreSummary.candidateCount}
                      />
                      <MetricChip
                        label="Selected"
                        value={restoreScoreSummary.selectedSnapshotId || "—"}
                      />
                      <MetricChip
                        label="Top Candidate"
                        value={restoreScoreSummary.topCandidateId || "—"}
                      />
                    </div>
                    {restoreScoreSummary.hasStructured ? (
                      <div className="overflow-auto rounded border border-white/10 bg-black/20">
                        <table className="min-w-full text-xs">
                          <thead>
                            <tr className="border-b border-white/10 bg-black/25 text-left text-forge-mist/70">
                              <th className="px-2 py-2">Snapshot</th>
                              <th className="px-2 py-2">Score</th>
                              <th className="px-2 py-2">Query</th>
                              <th className="px-2 py-2">Scope</th>
                              <th className="px-2 py-2">Kind</th>
                              <th className="px-2 py-2">Lineage</th>
                              <th className="px-2 py-2">State</th>
                              <th className="px-2 py-2">Loop</th>
                              <th className="px-2 py-2">Artifact</th>
                              <th className="px-2 py-2">Penalties</th>
                              <th className="px-2 py-2">Selected</th>
                            </tr>
                          </thead>
                          <tbody>
                            {restoreScoreSummary.candidates.map((candidate) => (
                              <tr
                                key={candidate.snapshotId}
                                className="border-b border-white/10 last:border-b-0"
                              >
                                <td className="px-2 py-1.5">
                                  <div className="text-[11px] text-forge-ash">
                                    {candidate.snapshotId}
                                  </div>
                                  <div className="text-[10px] text-forge-mist/70">
                                    {formatTime(candidate.createdAt)}
                                  </div>
                                  <div className="text-[10px] text-forge-mist/70">
                                    {candidate.snapshotKind || "restore"}
                                  </div>
                                </td>
                                <td className="px-2 py-1.5 text-forge-ash">
                                  {candidate.total.toFixed(3)}
                                </td>
                                <td className="px-2 py-1.5 text-forge-mist">
                                  {candidate.queryScore.toFixed(3)}
                                </td>
                                <td className="px-2 py-1.5 text-forge-mist">
                                  {candidate.scopeScore.toFixed(3)}
                                </td>
                                <td className="px-2 py-1.5 text-forge-mist">
                                  {candidate.kindScore.toFixed(3)}
                                </td>
                                <td className="px-2 py-1.5 text-forge-mist">
                                  {candidate.lineageScore.toFixed(3)}
                                </td>
                                <td className="px-2 py-1.5 text-forge-mist">
                                  {candidate.stateOverlapScore.toFixed(3)}
                                </td>
                                <td className="px-2 py-1.5 text-forge-mist">
                                  {candidate.loopOverlapScore.toFixed(3)}
                                </td>
                                <td className="px-2 py-1.5 text-forge-mist">
                                  {candidate.artifactOverlapScore.toFixed(3)}
                                </td>
                                <td className="px-2 py-1.5 text-forge-mist">
                                  {(
                                    candidate.stalenessPenalty +
                                    candidate.contradictionPenalty +
                                    candidate.headerOnlyPenalty
                                  ).toFixed(3)}
                                </td>
                                <td className="px-2 py-1.5 text-forge-mist">
                                  {candidate.selected ? "yes" : "—"}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    ) : null}
                    <div className="grid gap-4 md:grid-cols-2">
                      <div>
                        <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">
                          Resume Hints (parsed)
                        </div>
                        <div className="mt-2 grid gap-2 md:grid-cols-2">
                          <MetricChip
                            label="Next Action"
                            value={resumeHintSummary.nextAction || "—"}
                          />
                          <MetricChip
                            label="Preferred Snapshot"
                            value={resumeHintSummary.preferredSnapshotId || "—"}
                          />
                          <MetricChip
                            label="Restore Confidence"
                            value={resumeHintSummary.restoreConfidence.toFixed(
                              3,
                            )}
                          />
                          <MetricChip
                            label="Requires Fresh Compile"
                            value={
                              resumeHintSummary.requiresFreshCompile
                                ? "yes"
                                : "no"
                            }
                          />
                        </div>
                        <div className="mt-2 flex flex-wrap gap-2 text-xs text-forge-mist">
                          <span>
                            Top blockers:{" "}
                            {resumeHintSummary.topBlockers.length > 0
                              ? resumeHintSummary.topBlockers.join(", ")
                              : "none"}
                          </span>
                          <span>
                            Dominant states:{" "}
                            {resumeHintSummary.dominantStateKeys.length > 0
                              ? resumeHintSummary.dominantStateKeys.join(", ")
                              : "none"}
                          </span>
                          <span>
                            Dominant loops:{" "}
                            {resumeHintSummary.dominantLoopIds.length > 0
                              ? resumeHintSummary.dominantLoopIds.join(", ")
                              : "none"}
                          </span>
                        </div>
                      </div>
                      <JsonBlock
                        value={snapshotDetail.resumeHints}
                        empty="No resume hints recorded."
                      />
                    </div>
                    <JsonBlock
                      value={snapshotDetail.restoreScores}
                      empty="No restore scores recorded."
                    />
                  </div>
                </FoldSection>

                <FoldSection
                  title="Snapshot Evidence"
                  subtitle="Header, graph, and delta structures captured for operator inspection."
                >
                  <div className="space-y-4">
                    <JsonBlock
                      value={snapshotDetail.header}
                      empty="No header evidence recorded."
                    />
                    <JsonBlock
                      value={snapshotDetail.graph}
                      empty="No graph evidence recorded."
                    />
                    <JsonBlock
                      value={snapshotDetail.delta}
                      empty="No delta evidence recorded."
                    />
                  </div>
                </FoldSection>

                <FoldSection
                  title="Snapshot Details"
                  subtitle="Persisted details and included object ids."
                >
                  <div className="space-y-4">
                    <JsonBlock
                      value={snapshotDetail.metadata}
                      empty="No snapshot details recorded."
                    />
                    <JsonBlock
                      value={{
                        state: snapshotDetail.includedStateIds,
                        openLoops: snapshotDetail.includedOpenLoops,
                        notes: snapshotDetail.includedNoteIds,
                        links: snapshotDetail.includedLinkIds,
                        models: snapshotDetail.includedModelIds,
                        artifacts: snapshotDetail.includedArtifactIds,
                        events: snapshotDetail.includedEventIds,
                      }}
                      empty="No included ids recorded."
                    />
                  </div>
                </FoldSection>
              </>
            )}
          </div>
        </div>
      </Panel>

      <Panel
        title="Dream Reports"
        subtitle="Inspect persisted Dream Mode dry-run reports as non-canonical evidence."
      >
        <div className="mb-4 flex flex-wrap gap-2">
          <CountPill label="Reports" value={dreamReports.length} />
          <CountPill label="Selected" value={selectedDreamReportId || "none"} />
          <CountPill label="Evidence" value="non-canonical" />
          <CountPill label="Commit mode" value="disabled" />
        </div>
        <div className="grid gap-2 md:grid-cols-3">
          <input
            className="forge-input"
            placeholder="workspace id"
            value={dreamWorkspaceId}
            onChange={(e) => setDreamWorkspaceId(e.target.value)}
          />
          <input
            className="forge-input"
            placeholder="lane id"
            value={dreamLaneId}
            onChange={(e) => setDreamLaneId(e.target.value)}
          />
          <input
            className="forge-input"
            placeholder="mode"
            value={dreamMode}
            onChange={(e) => setDreamMode(e.target.value)}
          />
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton
            disabled={dreamBusy}
            onClick={() => void loadDreamReports()}
          >
            {dreamBusy ? "Loading…" : "Load Dream reports"}
          </PrimaryButton>
          <GhostButton
            onClick={() => {
              setDreamWorkspaceId("");
              setDreamLaneId("");
              setDreamMode("");
              setSelectedDreamReportId("");
              setDreamReports([]);
              setDreamReportDetail(null);
              setDreamErr(null);
              syncParams({
                dreamWorkspaceId: "",
                dreamLaneId: "",
                dreamMode: "",
                dreamReportId: "",
              });
            }}
          >
            Clear Dream filters
          </GhostButton>
        </div>
        {dreamErr ? (
          <div className="mt-3">
            <EmptyState
              title="Dream report lookup failed"
              detail={dreamErr}
              tone="bad"
            />
          </div>
        ) : null}
        <div className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
          <div className="space-y-2">
            {dreamReports.length === 0 ? (
              <EmptyState
                title={
                  dreamBusy
                    ? "Loading Dream reports"
                    : "No Dream reports matched"
                }
                detail={
                  dreamBusy
                    ? "Fetching dry-run Dream reports for the current workspace scope."
                    : "Provide a workspace id or adjust lane and mode filters to inspect non-canonical Dream evidence."
                }
              />
            ) : null}
            {dreamReports.map((report) => (
              <button
                key={report.id}
                type="button"
                onClick={() => {
                  setSelectedDreamReportId(report.id);
                  void loadDreamReportDetail(report.id);
                }}
                className={[
                  "w-full rounded border px-3 py-3 text-left transition",
                  selectedDreamReportId === report.id
                    ? "border-white/20 bg-white/10"
                    : "border-white/10 bg-black/20 hover:border-white/20",
                ].join(" ")}
              >
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div>
                    <div className="break-words text-sm font-semibold text-forge-ash">
                      {report.mode || "dream report"}
                    </div>
                    <div className="mt-1 break-all font-mono text-[11px] text-forge-mist/80">
                      {report.id}
                    </div>
                  </div>
                  <div className="text-[11px] text-forge-mist/75">
                    {formatTime(report.createdAt)}
                  </div>
                </div>
                <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-forge-mist/80">
                  <span>{report.workspaceId || "workspace: —"}</span>
                  <span>{report.laneId || "lane: —"}</span>
                  <span>{report.dryRun ? "dry-run" : "not dry-run"}</span>
                  <span>{report.status || "status: —"}</span>
                </div>
                <div className="mt-2 grid grid-cols-3 gap-2 text-[11px] text-forge-mist/80">
                  <span>candidates {report.candidatesConsidered ?? 0}</span>
                  <span>proposals {report.proposalsGenerated ?? 0}</span>
                  <span>warnings {report.warnings?.length ?? 0}</span>
                </div>
              </button>
            ))}
          </div>

          <div className="space-y-4">
            {!dreamReportDetail ? (
              <EmptyState
                title="Select a Dream report"
                detail="Load a workspace and choose a report to inspect replay candidates, salience scores, proposals, warnings, and trace evidence."
              />
            ) : (
              <>
                <div className="grid gap-2 md:grid-cols-4">
                  <MetricChip
                    label="Mode"
                    value={dreamReportDetail.mode || "—"}
                  />
                  <MetricChip
                    label="Status"
                    value={dreamReportDetail.status || "—"}
                  />
                  <MetricChip
                    label="Dry Run"
                    value={dreamReportDetail.dryRun ? "yes" : "no"}
                  />
                  <MetricChip
                    label="Canonical Commit"
                    value={
                      dreamReportDetail.canonicalWriteCommitted ? "yes" : "no"
                    }
                  />
                </div>
                <div className="flex flex-wrap gap-2">
                  <CountPill
                    label="Evidence"
                    value={
                      dreamReportDetail.evidenceClass ||
                      "non_canonical_evidence"
                    }
                  />
                  <CountPill
                    label="Candidates"
                    value={dreamReportDetail.candidates?.length ?? 0}
                  />
                  <CountPill
                    label="Salience"
                    value={dreamReportDetail.salienceScores?.length ?? 0}
                  />
                  <CountPill
                    label="Review"
                    value={
                      [
                        ...(dreamReportDetail.repairProposals ?? []),
                        ...(dreamReportDetail.snapshotHygieneProposals ?? []),
                      ].length
                    }
                  />
                </div>
                <div className="rounded border border-white/10 bg-black/20 p-3 text-sm text-forge-mist">
                  <div className="break-all font-mono text-[11px] text-forge-ash">
                    {dreamReportDetail.id}
                  </div>
                  <div className="mt-2 flex flex-wrap gap-2 text-xs">
                    <span>
                      workspace {dreamReportDetail.workspaceId || "—"}
                    </span>
                    <span>lane {dreamReportDetail.laneId || "—"}</span>
                    <span>corr {dreamReportDetail.correlationId || "—"}</span>
                    <span>trace {dreamReportDetail.traceId || "—"}</span>
                  </div>
                </div>

                <FoldSection
                  title="Run Summary"
                  subtitle="Dry-run report totals and summary details."
                  defaultOpen
                >
                  <div className="grid gap-3 md:grid-cols-3">
                    <MetricChip
                      label="Considered"
                      value={dreamReportDetail.candidatesConsidered ?? 0}
                    />
                    <MetricChip
                      label="Generated"
                      value={dreamReportDetail.proposalsGenerated ?? 0}
                    />
                    <MetricChip
                      label="Warnings"
                      value={dreamReportDetail.warnings?.length ?? 0}
                    />
                  </div>
                  <div className="mt-3">
                    <JsonBlock
                      value={dreamReportDetail.summary}
                      empty="No Dream summary details recorded."
                      maxHeightClass="max-h-[220px]"
                    />
                  </div>
                </FoldSection>

                <FoldSection
                  title="Replay & Salience"
                  subtitle="Replay candidates and deterministic salience evidence."
                  defaultOpen
                >
                  <div className="grid gap-4 md:grid-cols-2">
                    <JsonBlock
                      value={dreamReportDetail.candidates}
                      empty="No replay candidates recorded."
                      maxHeightClass="max-h-[280px]"
                    />
                    <JsonBlock
                      value={dreamReportDetail.salienceScores}
                      empty="No salience scores recorded."
                      maxHeightClass="max-h-[280px]"
                    />
                  </div>
                </FoldSection>

                <FoldSection
                  title="Proposals & Review"
                  subtitle="Proposal records only. The inspector does not apply Dream Mode changes."
                  defaultOpen
                >
                  <div className="grid gap-4 md:grid-cols-2">
                    <JsonBlock
                      value={dreamReportDetail.memoryTierProposals}
                      empty="No memory-tier proposals recorded."
                      maxHeightClass="max-h-[280px]"
                    />
                    <JsonBlock
                      value={dreamReportDetail.repairProposals}
                      empty="No repair proposals recorded."
                      maxHeightClass="max-h-[280px]"
                    />
                    <JsonBlock
                      value={dreamReportDetail.snapshotHygieneProposals}
                      empty="No snapshot hygiene proposals recorded."
                      maxHeightClass="max-h-[280px]"
                    />
                    <JsonBlock
                      value={dreamReportDetail.warnings}
                      empty="No warnings recorded."
                      maxHeightClass="max-h-[280px]"
                    />
                  </div>
                </FoldSection>

                <FoldSection
                  title="Trace & Details"
                  subtitle="Correlation, trace, and non-canonical report details."
                >
                  <div className="grid gap-4 md:grid-cols-2">
                    <JsonBlock
                      value={dreamReportDetail.trace}
                      empty="No trace details recorded."
                    />
                    <JsonBlock
                      value={dreamReportDetail.metadata}
                      empty="No report details recorded."
                    />
                  </div>
                </FoldSection>
              </>
            )}
          </div>
        </div>
      </Panel>

      <Panel
        title="Context Packet Inspector"
        subtitle="Read-only task packet view with alignment notes and packet guidance evidence."
      >
        <div className="mb-4 flex flex-wrap gap-2">
          <CountPill label="Packet" value={packet?.id ?? "none"} />
          <CountPill label="Retrieved context" value={packetRetrievedCount} />
          <CountPill label="Source refs" value={packetReferenceCount} />
        </div>
        <div className="flex flex-wrap gap-2">
          <input
            className="forge-input min-w-[240px] flex-1"
            placeholder="packet id"
            value={packetIdInput}
            onChange={(e) => setPacketIdInput(e.target.value)}
          />
          <PrimaryButton
            disabled={packetBusy}
            onClick={() => void loadPacketInspector()}
          >
            {packetBusy ? "Loading…" : "Load packet"}
          </PrimaryButton>
        </div>
        {packetErr ? (
          <div className="mt-3">
            <EmptyState
              title="Packet lookup failed"
              detail={packetErr}
              tone="bad"
            />
          </div>
        ) : null}
        {!packet ? (
          <div className="mt-4">
            <EmptyState
              title="No packet loaded"
              detail="Load a packet id to inspect request scope, retrieved context, source references, alignment notes, and guidance evidence."
            />
          </div>
        ) : (
          <div className="mt-4 space-y-4">
            <div className="grid gap-2 md:grid-cols-4">
              <MetricChip label="Packet" value={packet.id} />
              <MetricChip label="Risk" value={packet.riskClass} />
              <MetricChip label="Retrieved" value={packetRetrievedCount} />
              <MetricChip label="References" value={packetReferenceCount} />
            </div>
            <div className="rounded border border-white/10 bg-black/20 p-3 text-sm text-forge-mist">
              <div className="break-words text-base font-semibold text-forge-ash">
                {packet.title}
              </div>
              <div className="mt-2 break-words">{packet.objective}</div>
              <div className="mt-2 text-xs text-forge-mist/80">
                Generated {formatTime(packet.generatedAtMs)} · adapter{" "}
                {packet.adapterTarget}
              </div>
            </div>

            <FoldSection
              title="Request"
              subtitle="User request, instructions, and selected path scope."
              defaultOpen
            >
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-3 text-sm text-forge-mist">
                  <div>
                    <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">
                      User Request
                    </div>
                    <div className="mt-1 break-words text-forge-ash">
                      {packet.userRequest}
                    </div>
                  </div>
                  <div>
                    <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">
                      Instructions
                    </div>
                    <div className="mt-1 whitespace-pre-wrap break-words text-forge-ash">
                      {packet.instructions || "—"}
                    </div>
                  </div>
                </div>
                <div className="space-y-3">
                  <JsonBlock
                    value={packet.expectedOutput}
                    empty="No expected output contract."
                    maxHeightClass="max-h-[220px]"
                  />
                  <JsonBlock
                    value={packet.scopeSnapshot}
                    empty="No scope snapshot recorded."
                    maxHeightClass="max-h-[220px]"
                  />
                </div>
              </div>
              <div className="mt-3 flex flex-wrap gap-2">
                {packet.selectedPaths.length === 0 ? (
                  <span className="text-xs text-forge-mist/75">
                    No selected paths recorded.
                  </span>
                ) : (
                  packet.selectedPaths.map((path) => (
                    <span
                      key={path}
                      className="min-w-0 break-all rounded border border-white/10 bg-black/25 px-2 py-1 font-mono text-[11px] text-forge-ash"
                    >
                      {path}
                    </span>
                  ))
                )}
              </div>
            </FoldSection>

            <FoldSection
              title="Request & Retrieval"
              subtitle="Packet request details plus retrieved context and source references."
            >
              <div className="space-y-4">
                <JsonBlock
                  value={packet.requestPayload}
                  empty="No request details recorded."
                />
                <JsonBlock
                  value={packet.sourceReferences}
                  empty="No source references recorded."
                />
                <JsonBlock
                  value={packet.retrievedContext}
                  empty="No retrieved context recorded."
                />
              </div>
            </FoldSection>

            <FoldSection
              title="Alignment Notes"
              subtitle="Why retrieved evidence was included in this packet."
              defaultOpen
            >
              {packetAlignment.length === 0 ? (
                <EmptyState
                  title="No alignment notes"
                  detail="This packet does not have recorded evidence inclusion notes."
                />
              ) : (
                <div className="space-y-2">
                  {packetAlignment.map((note) => (
                    <div
                      key={note.id}
                      className="rounded border border-white/10 bg-black/20 p-3 text-sm text-forge-mist"
                    >
                      <div className="break-words text-forge-ash">
                        {note.note}
                      </div>
                      <div className="mt-2 text-[11px] text-forge-mist/75">
                        Recorded {formatTime(note.createdAtMs)}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </FoldSection>

            <FoldSection
              title="Guidance"
              subtitle="Packet quality guidance, issues, and recommendations."
              defaultOpen
            >
              {packetGuidance.length === 0 ? (
                <EmptyState
                  title="No guidance runs"
                  detail="This packet has no recorded packet-quality guidance runs."
                />
              ) : (
                <div className="space-y-3">
                  {packetGuidance.map((item) => (
                    <div
                      key={item.id}
                      className="rounded border border-white/10 bg-black/20 p-3"
                    >
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="text-sm font-semibold text-forge-ash">
                          Guidance score {item.guidanceScore}
                        </div>
                        <div className="text-[11px] text-forge-mist/75">
                          {formatTime(item.createdAtMs)}
                        </div>
                      </div>
                      <div className="mt-2 text-sm text-forge-mist">
                        Issues:{" "}
                        {item.issues.length > 0
                          ? item.issues.join(", ")
                          : "none"}
                      </div>
                      <div className="mt-1 text-sm text-forge-mist">
                        Recommendations:{" "}
                        {item.recommendations.length > 0
                          ? item.recommendations.join(", ")
                          : "none"}
                      </div>
                      <div className="mt-3">
                        <JsonBlock
                          value={item.evidence}
                          empty="No guidance evidence recorded."
                          maxHeightClass="max-h-[220px]"
                        />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </FoldSection>
          </div>
        )}
      </Panel>

      <Panel
        title="Execution Trace"
        subtitle="Trace persisted execution evidence by correlation id or trace id."
      >
        <div className="mb-4 grid gap-3 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
          <div className="rounded border border-white/10 bg-black/20 p-3 text-sm leading-6 text-forge-mist">
            Correlation trace is the fastest path from operator question to
            execution truth: audit rows, gateway calls, artifact refs,
            provenance, and journal entries.
          </div>
          <div className="rounded border border-white/10 bg-black/20 p-3">
            <div className="text-[10px] uppercase tracking-[0.14em] text-forge-mist/65">
              Lookup state
            </div>
            <div className="mt-2 flex flex-wrap gap-2">
              <CountPill label="Mode" value={traceLookup?.mode ?? "idle"} />
              <CountPill
                label="Reports"
                value={traceLookup?.reports?.length ?? 0}
              />
              <CountPill
                label="Correlation input"
                value={correlationIdInput || "—"}
              />
            </div>
          </div>
        </div>
        <div className="grid gap-2 md:grid-cols-2">
          <input
            className="forge-input"
            placeholder="correlation id"
            value={correlationIdInput}
            onChange={(e) => setCorrelationIdInput(e.target.value)}
          />
          <input
            className="forge-input"
            placeholder="trace id"
            value={traceIdInput}
            onChange={(e) => setTraceIdInput(e.target.value)}
          />
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton
            disabled={traceBusy}
            onClick={() => void loadTraceInspector()}
          >
            {traceBusy ? "Loading…" : "Load trace"}
          </PrimaryButton>
          <GhostButton
            onClick={() => {
              setCorrelationIdInput("");
              setTraceIdInput("");
              setTraceLookup(null);
              setSelectedTraceCorrelationId("");
              syncParams({ correlationId: "", traceId: "" });
            }}
          >
            Clear trace inputs
          </GhostButton>
        </div>
        {traceErr ? (
          <div className="mt-3">
            <EmptyState
              title="Trace lookup failed"
              detail={traceErr}
              tone="bad"
            />
          </div>
        ) : null}
        {!traceLookup ? (
          <div className="mt-4">
            <EmptyState
              title="No trace loaded"
              detail="Load a correlation id or trace id to inspect audit rows, gateway calls, artifact references, provenance, and journal entries."
            />
          </div>
        ) : (
          <div className="mt-4 space-y-4">
            {traceLookup.mode === "trace" &&
            (traceLookup.correlationIds?.length ?? 0) > 1 ? (
              <div className="flex flex-wrap gap-2">
                {traceLookup.correlationIds?.map((correlationId) => (
                  <button
                    key={correlationId}
                    type="button"
                    onClick={() => setSelectedTraceCorrelationId(correlationId)}
                    className={[
                      "rounded border px-2.5 py-1 text-[11px]",
                      selectedTraceCorrelationId === correlationId
                        ? "border-white/20 bg-white/10 text-forge-ash"
                        : "border-white/10 bg-black/20 text-forge-mist",
                    ].join(" ")}
                  >
                    {correlationId}
                  </button>
                ))}
              </div>
            ) : null}

            {selectedTraceReport ? (
              <>
                <div className="rounded border border-white/10 bg-black/20 p-3">
                  <div className="text-[10px] uppercase tracking-[0.14em] text-forge-mist/65">
                    Selected execution chain
                  </div>
                  <div className="mt-2 flex flex-wrap gap-2">
                    <CountPill
                      label="Correlation"
                      value={selectedTraceReport.correlationId}
                    />
                    <CountPill
                      label="Trace id"
                      value={traceLookup.traceId ?? traceIdInput ?? "—"}
                    />
                    <CountPill label="Mode" value={traceLookup.mode} />
                  </div>
                </div>
                <div className="grid gap-2 md:grid-cols-4">
                  <MetricChip
                    label="Correlation"
                    value={selectedTraceReport.correlationId}
                  />
                  <MetricChip
                    label="Audit Records"
                    value={parsedTraceSummary.auditRecords.length}
                  />
                  <MetricChip
                    label="Gateway Calls"
                    value={parsedTraceSummary.gatewayInvocations.length}
                  />
                  <MetricChip
                    label="Artifacts"
                    value={parsedTraceSummary.artifactRecords.length}
                  />
                </div>
                <div className="flex flex-wrap gap-2">
                  <SummaryLink
                    to={`/audit?correlationId=${encodeURIComponent(selectedTraceReport.correlationId)}`}
                    label="Open audit list"
                  />
                  <SummaryLink
                    to={`/inspectors?correlationId=${encodeURIComponent(selectedTraceReport.correlationId)}`}
                    label="Filter snapshots by correlation"
                  />
                </div>

                <FoldSection
                  title="Gateway Invocations"
                  subtitle="Governed tool invocations tied to this correlation."
                  defaultOpen
                >
                  <JsonBlock
                    value={parsedTraceSummary.gatewayInvocations}
                    empty="No gateway invocations linked to this correlation."
                  />
                </FoldSection>

                <FoldSection
                  title="Audit Records"
                  subtitle="Append-only audit records for this correlation."
                  defaultOpen
                >
                  <JsonBlock
                    value={parsedTraceSummary.auditRecords}
                    empty="No audit records linked to this correlation."
                  />
                </FoldSection>

                <FoldSection
                  title="Provenance & Journal"
                  subtitle="Semantic provenance rows and journal events when present."
                >
                  <div className="space-y-4">
                    <JsonBlock
                      value={parsedTraceSummary.provenanceRecords}
                      empty="No provenance rows linked to this correlation."
                    />
                    <JsonBlock
                      value={parsedTraceSummary.journalEvents}
                      empty="No journal events linked to this correlation."
                    />
                  </div>
                </FoldSection>

                <FoldSection
                  title="Artifacts & Links"
                  subtitle="Stored artifacts, artifact refs, and inferred trace links."
                >
                  <div className="space-y-4">
                    <JsonBlock
                      value={parsedTraceSummary.artifactRecords}
                      empty="No artifact rows linked to this correlation."
                    />
                    <JsonBlock
                      value={parsedTraceSummary.artifactRefs}
                      empty="No artifact refs linked to this correlation."
                    />
                    <JsonBlock
                      value={parsedTraceSummary.links}
                      empty="No trace links recorded."
                    />
                  </div>
                </FoldSection>
              </>
            ) : (
              <EmptyState
                title="No trace reports"
                detail="No persisted execution reports were found for the supplied correlation or trace id."
              />
            )}
          </div>
        )}
      </Panel>

      <Panel
        title="Process & Health Trace"
        subtitle="Trace process tool invocations and model-runtime condition for a correlation id or trace id."
      >
        <div className="mb-4 grid gap-3 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
          <div className="rounded border border-white/10 bg-black/20 p-3 text-sm leading-6 text-forge-mist">
            Process trace shows only governed process invocations, while
            model-runtime entries are non-mutating read-only diagnostics.
          </div>
          <div className="rounded border border-white/10 bg-black/20 p-3">
            <div className="text-[10px] uppercase tracking-[0.14em] text-forge-mist/65">
              Runtime projection
            </div>
            <div className="mt-2 flex flex-wrap gap-2">
              <CountPill
                label="Process invocations"
                value={processParsedRuntime.totalProcessInvocations}
              />
              <CountPill
                label="Gateway invocations"
                value={processParsedRuntime.totalGatewayInvocations}
              />
              <CountPill
                label="Runtime available"
                value={processLookup?.runtime?.available ? "yes" : "no"}
              />
              <CountPill
                label="Runtime state"
                value={processLookup?.runtime?.state || "unknown"}
              />
              <CountPill
                label="Safe mode"
                value={processLookup?.runtime?.safeMode ? "on" : "off"}
              />
              <CountPill
                label="GPU aware"
                value={processLookup?.runtime?.gpuAware ? "yes" : "no"}
              />
            </div>
            {processLookup?.runtime?.safeModeReasons?.length ? (
              <div className="mt-2 text-xs text-forge-mist">
                {processLookup.runtime.safeModeReasons.join("; ")}
              </div>
            ) : null}
          </div>
        </div>
        <div className="grid gap-2 md:grid-cols-2">
          <input
            className="forge-input"
            placeholder="correlation id"
            value={processCorrelationIdInput}
            onChange={(e) => setProcessCorrelationIdInput(e.target.value)}
          />
          <input
            className="forge-input"
            placeholder="trace id"
            value={processTraceIdInput}
            onChange={(e) => setProcessTraceIdInput(e.target.value)}
          />
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton
            disabled={processBusy}
            onClick={() => void loadProcessHealthInspector()}
          >
            {processBusy ? "Loading…" : "Load process trace"}
          </PrimaryButton>
          <GhostButton
            onClick={() => {
              setProcessCorrelationIdInput("");
              setProcessTraceIdInput("");
              setProcessLookup(null);
              setProcessErr(null);
              syncParams({ processCorrelationId: "", processTraceId: "" });
            }}
          >
            Clear process inputs
          </GhostButton>
        </div>
        {processErr ? (
          <div className="mt-3">
            <EmptyState
              title="Process trace lookup failed"
              detail={processErr}
              tone="bad"
            />
          </div>
        ) : null}
        {!processLookup ? (
          <div className="mt-4">
            <EmptyState
              title="No process trace loaded"
              detail="Load a correlation id or trace id to inspect governed process invocations and model-runtime diagnostics."
            />
          </div>
        ) : (
          <div className="mt-4 space-y-4">
            <div className="grid gap-2 md:grid-cols-2">
              <MetricChip
                label="Correlation"
                value={processLookup.correlationId || "—"}
              />
              <MetricChip
                label="Trace id"
                value={processLookup.traceId || "—"}
              />
              <MetricChip
                label="Input mode"
                value={processLookup.correlationId ? "correlation" : "trace"}
              />
              <MetricChip
                label="Correlation count"
                value={processLookup.correlationIds?.length ?? 0}
              />
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <JsonBlock
                value={processLookup.runtime.health || null}
                empty="Runtime health unavailable."
              />
              <JsonBlock
                value={processLookup.runtime.usage || null}
                empty="Runtime usage unavailable."
              />
              <JsonBlock
                value={processLookup.runtime.queue || null}
                empty="Runtime queue unavailable."
              />
              <JsonBlock
                value={processLookup.runtime.loaded || null}
                empty="Loaded models unavailable."
              />
            </div>
            {processLookup.runtime.error ? (
              <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm leading-6 text-forge-ash">
                {processLookup.runtime.error}
              </div>
            ) : null}
            {processLookup.reports.length === 0 ? (
              <EmptyState
                title="No process invocations"
                detail="No governed process invocations were captured for this correlation family."
              />
            ) : (
              processLookup.reports.map((report) => (
                <FoldSection
                  key={report.correlationId}
                  title={`Process invocations (${report.correlationId})`}
                  subtitle={`Process: ${report.processInvocationCount} · Gateway total: ${report.totalInvocations}`}
                  defaultOpen
                >
                  <div className="mb-2">
                    <div className="flex flex-wrap gap-2">
                      <SummaryLink
                        to={`/audit?correlationId=${encodeURIComponent(report.correlationId)}`}
                        label="Open audit list"
                      />
                    </div>
                  </div>
                  <div className="overflow-auto rounded border border-white/10 bg-black/20">
                    <table className="min-w-full text-xs">
                      <thead>
                        <tr className="border-b border-white/10 bg-black/25 text-left text-forge-mist/70">
                          <th className="px-2 py-2">Invocation</th>
                          <th className="px-2 py-2">Tool</th>
                          <th className="px-2 py-2">Action</th>
                          <th className="px-2 py-2">Domain</th>
                          <th className="px-2 py-2">Status</th>
                          <th className="px-2 py-2">Policy</th>
                          <th className="px-2 py-2">Duration</th>
                          <th className="px-2 py-2">Trace</th>
                        </tr>
                      </thead>
                      <tbody>
                        {parseProcessRuntimeLine(report.processInvocations)}
                      </tbody>
                    </table>
                  </div>
                  <FoldSection
                    title="Process Invocation Details"
                    subtitle="Traceable invocation details for this correlation."
                  >
                    <JsonBlock
                      value={report.processInvocations}
                      empty="No process invocations recorded for this correlation."
                    />
                  </FoldSection>
                </FoldSection>
              ))
            )}
          </div>
        )}
      </Panel>
    </div>
  );
}
