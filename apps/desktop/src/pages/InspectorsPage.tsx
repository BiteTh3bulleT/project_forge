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
import { DreamReportsPanel } from "./InspectorsPage/DreamReportsPanel";
import { InspectorMetricsOverview } from "./InspectorsPage/InspectorMetricsOverview";
import { SnapshotInspectorPanel } from "./InspectorsPage/SnapshotInspectorPanel";
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

      <SnapshotInspectorPanel
        snapshotWorkspaceId={snapshotWorkspaceId}
        snapshotLaneId={snapshotLaneId}
        snapshotKind={snapshotKind}
        snapshotQuery={snapshotQuery}
        snapshotCorrelationId={snapshotCorrelationId}
        selectedSnapshotId={selectedSnapshotId}
        snapshots={snapshots}
        snapshotDetail={snapshotDetail}
        snapshotBusy={snapshotBusy}
        snapshotErr={snapshotErr}
        restoreScoreSummary={restoreScoreSummary}
        resumeHintSummary={resumeHintSummary}
        onWorkspaceIdChange={setSnapshotWorkspaceId}
        onLaneIdChange={setSnapshotLaneId}
        onSnapshotKindChange={setSnapshotKind}
        onSnapshotQueryChange={setSnapshotQuery}
        onCorrelationIdChange={setSnapshotCorrelationId}
        onLoadSnapshots={() => loadSnapshots()}
        onClearFilters={() => {
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
        onSelectSnapshot={(snapshotId) => {
          setSelectedSnapshotId(snapshotId);
          void loadSnapshotDetail(snapshotId);
        }}
        onOpenParentSnapshot={(snapshotId) => {
          setSelectedSnapshotId(snapshotId);
          void loadSnapshotDetail(snapshotId);
        }}
      />

      <DreamReportsPanel
        dreamWorkspaceId={dreamWorkspaceId}
        dreamLaneId={dreamLaneId}
        dreamMode={dreamMode}
        selectedDreamReportId={selectedDreamReportId}
        dreamReports={dreamReports}
        dreamReportDetail={dreamReportDetail}
        dreamBusy={dreamBusy}
        dreamErr={dreamErr}
        onWorkspaceIdChange={setDreamWorkspaceId}
        onLaneIdChange={setDreamLaneId}
        onModeChange={setDreamMode}
        onLoadDreamReports={() => loadDreamReports()}
        onClearDreamFilters={() => {
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
        onSelectDreamReport={(reportId) => {
          setSelectedDreamReportId(reportId);
          void loadDreamReportDetail(reportId);
        }}
      />

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
