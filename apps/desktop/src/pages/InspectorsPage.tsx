import type { PacketAlignmentNote, PacketGuidance, TaskPacket } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { FoldSection } from "../components/FoldSection";
import {
  api,
  type AuditTraceLookupReport,
  type AuditTraceLookupResponse,
  type ContextSnapshotInspectorDetail,
  type ContextSnapshotInspectorSummary,
} from "../lib/api";
import { formatTime } from "../lib/format";

function asRecord(value: unknown): Record<string, unknown> | null {
  if (value && typeof value === "object" && !Array.isArray(value)) return value as Record<string, unknown>;
  return null;
}

function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

function JsonBlock(props: { value: unknown; empty?: string; maxHeightClass?: string }) {
  const text = useMemo(() => {
    if (props.value == null) return "";
    try {
      return JSON.stringify(props.value, null, 2);
    } catch {
      return String(props.value);
    }
  }, [props.value]);
  if (!text || text === "{}" || text === "[]" || text === "null") {
    return <div className="text-xs text-forge-mist/75">{props.empty ?? "No recorded evidence."}</div>;
  }
  return (
    <pre
      className={[
        "overflow-auto rounded border border-white/10 bg-black/25 p-3 font-mono text-[11px] text-forge-mist",
        props.maxHeightClass ?? "max-h-[360px]",
      ].join(" ")}
    >
      {text}
    </pre>
  );
}

function MetricChip(props: { label: string; value: string | number }) {
  return (
    <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
      <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">{props.label}</div>
      <div className="mt-1 text-sm text-forge-ash">{props.value}</div>
    </div>
  );
}

function SummaryLink(props: { to: string; label: string }) {
  return (
    <Link
      to={props.to}
      className="rounded border border-white/15 bg-white/5 px-2.5 py-1 text-[11px] text-forge-mist transition hover:text-forge-ash"
    >
      {props.label}
    </Link>
  );
}

function parseInspectorReportSummary(report: AuditTraceLookupReport | null) {
  const raw = asRecord(report?.report);
  return {
    gatewayInvocations: asArray(raw?.gatewayInvocations),
    auditRecords: asArray(raw?.auditRecords),
    artifactRecords: asArray(raw?.artifactRecords),
    provenanceRecords: asArray(raw?.provenanceRecords),
    journalEvents: asArray(raw?.journalEvents),
    artifactRefs: asArray(raw?.artifactRefs),
    links: asRecord(raw?.links) ?? {},
  };
}

export function InspectorsPage() {
  const [params, setParams] = useSearchParams();

  const [snapshotWorkspaceId, setSnapshotWorkspaceId] = useState(() => params.get("workspaceId") ?? "");
  const [snapshotLaneId, setSnapshotLaneId] = useState(() => params.get("laneId") ?? "");
  const [snapshotKind, setSnapshotKind] = useState(() => params.get("snapshotKind") ?? "");
  const [snapshotQuery, setSnapshotQuery] = useState(() => params.get("snapshotQuery") ?? "");
  const [snapshotCorrelationId, setSnapshotCorrelationId] = useState(() => params.get("correlationId") ?? "");
  const [selectedSnapshotId, setSelectedSnapshotId] = useState(() => params.get("snapshotId") ?? "");
  const [snapshots, setSnapshots] = useState<ContextSnapshotInspectorSummary[]>([]);
  const [snapshotDetail, setSnapshotDetail] = useState<ContextSnapshotInspectorDetail | null>(null);
  const [snapshotBusy, setSnapshotBusy] = useState(false);
  const [snapshotErr, setSnapshotErr] = useState<string | null>(null);

  const [packetIdInput, setPacketIdInput] = useState(() => params.get("packetId") ?? "");
  const [packetBusy, setPacketBusy] = useState(false);
  const [packetErr, setPacketErr] = useState<string | null>(null);
  const [packet, setPacket] = useState<TaskPacket | null>(null);
  const [packetAlignment, setPacketAlignment] = useState<PacketAlignmentNote[]>([]);
  const [packetGuidance, setPacketGuidance] = useState<PacketGuidance[]>([]);

  const [correlationIdInput, setCorrelationIdInput] = useState(() => params.get("correlationId") ?? "");
  const [traceIdInput, setTraceIdInput] = useState(() => params.get("traceId") ?? "");
  const [traceBusy, setTraceBusy] = useState(false);
  const [traceErr, setTraceErr] = useState<string | null>(null);
  const [traceLookup, setTraceLookup] = useState<AuditTraceLookupResponse | null>(null);
  const [selectedTraceCorrelationId, setSelectedTraceCorrelationId] = useState("");

  const selectedTraceReport = useMemo(() => {
    if (!traceLookup?.reports?.length) return null;
    return traceLookup.reports.find((item) => item.correlationId === selectedTraceCorrelationId) ?? traceLookup.reports[0];
  }, [selectedTraceCorrelationId, traceLookup]);

  const parsedTraceSummary = useMemo(() => parseInspectorReportSummary(selectedTraceReport), [selectedTraceReport]);

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
    const effectiveCorrelationId = overrides?.correlationId ?? snapshotCorrelationId;
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
      setPacketAlignment(Array.isArray(alignmentRes.notes) ? alignmentRes.notes : []);
      setPacketGuidance(Array.isArray(guidanceRes.guidance) ? guidanceRes.guidance : []);
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

  async function loadTraceInspector(next?: { correlationId?: string; traceId?: string }) {
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

  useEffect(() => {
    void loadSnapshots(selectedSnapshotId);
  }, []);

  useEffect(() => {
    if (packetIdInput.trim()) {
      void loadPacketInspector(packetIdInput);
    }
  }, []);

  useEffect(() => {
    if (correlationIdInput.trim() || traceIdInput.trim()) {
      void loadTraceInspector({ correlationId: correlationIdInput, traceId: traceIdInput });
    }
  }, []);

  const packetRetrievedCount = packet?.retrievedContext?.length ?? 0;
  const packetReferenceCount = packet?.sourceReferences?.length ?? 0;

  return (
    <div className="space-y-6">
      <Panel
        title="Operator Inspectors"
        subtitle="Read-only evidence views for context snapshots, packet materialization, and correlation traces."
        actions={<GhostButton onClick={() => void Promise.all([loadSnapshots(selectedSnapshotId), correlationIdInput || traceIdInput ? loadTraceInspector() : Promise.resolve()])}>Refresh views</GhostButton>}
      >
        <div className="rounded border border-forge-electric/20 bg-forge-electric/5 p-3 text-sm text-forge-mist">
          These surfaces expose persisted evidence only. They do not execute tools, mutate truth, or bypass approvals.
        </div>
      </Panel>

      <Panel title="Snapshot Inspector" subtitle="Inspect persisted context compilation snapshots, restore scoring, and resume hints.">
        <div className="grid gap-2 md:grid-cols-5">
          <input className="forge-input" placeholder="workspace id" value={snapshotWorkspaceId} onChange={(e) => setSnapshotWorkspaceId(e.target.value)} />
          <input className="forge-input" placeholder="lane id" value={snapshotLaneId} onChange={(e) => setSnapshotLaneId(e.target.value)} />
          <input className="forge-input" placeholder="snapshot kind" value={snapshotKind} onChange={(e) => setSnapshotKind(e.target.value)} />
          <input className="forge-input" placeholder="query filter" value={snapshotQuery} onChange={(e) => setSnapshotQuery(e.target.value)} />
          <input
            className="forge-input"
            placeholder="correlation id"
            value={snapshotCorrelationId}
            onChange={(e) => setSnapshotCorrelationId(e.target.value)}
          />
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton disabled={snapshotBusy} onClick={() => void loadSnapshots()}>
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
        {snapshotErr ? <div className="mt-3 rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{snapshotErr}</div> : null}
        <div className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
          <div className="space-y-2">
            {snapshots.length === 0 ? <div className="text-sm text-forge-mist">No snapshots matched the current filters.</div> : null}
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
                  selectedSnapshotId === snapshot.id ? "border-white/20 bg-white/10" : "border-white/10 bg-black/20 hover:border-white/20",
                ].join(" ")}
              >
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div>
                    <div className="text-sm font-semibold text-forge-ash">{snapshot.query || snapshot.id}</div>
                    <div className="mt-1 font-mono text-[11px] text-forge-mist/80">{snapshot.id}</div>
                  </div>
                  <div className="text-[11px] text-forge-mist/75">{formatTime(snapshot.createdAtMs)}</div>
                </div>
                <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-forge-mist/80">
                  <span>{snapshot.snapshotKind || "kind: —"}</span>
                  <span>{snapshot.workspaceId || "workspace: —"}</span>
                  <span>{snapshot.laneId || "lane: —"}</span>
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
              <div className="rounded border border-dashed border-white/10 px-4 py-6 text-sm text-forge-mist">Select a snapshot to inspect its persisted evidence.</div>
            ) : (
              <>
                <div className="grid gap-2 md:grid-cols-3">
                  <MetricChip label="Kind" value={snapshotDetail.summary.snapshotKind || "—"} />
                  <MetricChip label="Workspace" value={snapshotDetail.summary.workspaceId || "—"} />
                  <MetricChip label="Lane" value={snapshotDetail.summary.laneId || "—"} />
                </div>
                <div className="rounded border border-white/10 bg-black/20 p-3 text-sm text-forge-mist">
                  <div className="font-medium text-forge-ash">{snapshotDetail.summary.query || snapshotDetail.summary.id}</div>
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
                          setSelectedSnapshotId(snapshotDetail.summary.parentSnapshotId);
                          void loadSnapshotDetail(snapshotDetail.summary.parentSnapshotId);
                        }}
                      >
                        Open parent {snapshotDetail.summary.parentSnapshotId}
                      </button>
                    ) : null}
                  </div>
                  <div className="mt-3 grid gap-2 md:grid-cols-2">
                    <div>
                      <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">Fingerprint</div>
                      <div className="mt-1 font-mono text-[11px] text-forge-ash">{snapshotDetail.summary.snapshotFingerprint || "—"}</div>
                    </div>
                    <div>
                      <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">Render Artifact Ref</div>
                      <div className="mt-1 font-mono text-[11px] text-forge-ash">{snapshotDetail.summary.renderArtifactRefId || "—"}</div>
                    </div>
                  </div>
                </div>

                <FoldSection title="Selection & Counts" subtitle="Scope paths, syscall lineage, and included evidence counts." defaultOpen>
                  <div className="grid gap-3 md:grid-cols-2">
                    <div>
                      <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">Selected Paths</div>
                      <div className="mt-2 flex flex-wrap gap-2">
                        {snapshotDetail.summary.selectedPaths.length === 0 ? (
                          <span className="text-xs text-forge-mist/75">No explicit path scope recorded.</span>
                        ) : (
                          snapshotDetail.summary.selectedPaths.map((path) => (
                            <span key={path} className="rounded border border-white/10 bg-black/25 px-2 py-1 font-mono text-[11px] text-forge-ash">
                              {path}
                            </span>
                          ))
                        )}
                      </div>
                    </div>
                    <div className="grid gap-2 grid-cols-2">
                      <MetricChip label="State" value={snapshotDetail.summary.counts.state} />
                      <MetricChip label="Open loops" value={snapshotDetail.summary.counts.openLoops} />
                      <MetricChip label="Notes" value={snapshotDetail.summary.counts.notes} />
                      <MetricChip label="Artifacts" value={snapshotDetail.summary.counts.artifacts} />
                    </div>
                  </div>
                  <div className="mt-3 grid gap-2 md:grid-cols-3">
                    <MetricChip label="Correlation" value={snapshotDetail.summary.correlationId || "—"} />
                    <MetricChip label="Trace" value={snapshotDetail.summary.traceId || "—"} />
                    <MetricChip label="Syscall" value={snapshotDetail.summary.syscallId || "—"} />
                  </div>
                </FoldSection>

                <FoldSection title="Budget & Inclusion" subtitle="Persisted packet budget and inclusion reasons." defaultOpen>
                  <div className="grid gap-4 md:grid-cols-2">
                    <JsonBlock value={snapshotDetail.budget} empty="No budget recorded." />
                    <JsonBlock value={snapshotDetail.inclusionReasons} empty="No inclusion reasons recorded." />
                  </div>
                </FoldSection>

                <FoldSection title="Restore Scoring" subtitle="Non-canonical restore scores and resume hints carried with the snapshot." defaultOpen>
                  <div className="grid gap-4 md:grid-cols-2">
                    <JsonBlock value={snapshotDetail.restoreScores} empty="No restore scores recorded." />
                    <JsonBlock value={snapshotDetail.resumeHints} empty="No resume hints recorded." />
                  </div>
                </FoldSection>

                <FoldSection title="Snapshot Evidence" subtitle="Header, graph, and delta structures captured for operator inspection.">
                  <div className="space-y-4">
                    <JsonBlock value={snapshotDetail.header} empty="No header evidence recorded." />
                    <JsonBlock value={snapshotDetail.graph} empty="No graph evidence recorded." />
                    <JsonBlock value={snapshotDetail.delta} empty="No delta evidence recorded." />
                  </div>
                </FoldSection>

                <FoldSection title="Raw Metadata" subtitle="Persisted metadata and included object ids.">
                  <div className="space-y-4">
                    <JsonBlock value={snapshotDetail.metadata} empty="No metadata recorded." />
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

      <Panel title="Context Packet Inspector" subtitle="Read-only task packet view with alignment notes and packet guidance evidence.">
        <div className="flex flex-wrap gap-2">
          <input
            className="forge-input min-w-[240px] flex-1"
            placeholder="packet id"
            value={packetIdInput}
            onChange={(e) => setPacketIdInput(e.target.value)}
          />
          <PrimaryButton disabled={packetBusy} onClick={() => void loadPacketInspector()}>
            {packetBusy ? "Loading…" : "Load packet"}
          </PrimaryButton>
        </div>
        {packetErr ? <div className="mt-3 rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{packetErr}</div> : null}
        {!packet ? (
          <div className="mt-4 text-sm text-forge-mist">Load a packet id to inspect its task packet evidence.</div>
        ) : (
          <div className="mt-4 space-y-4">
            <div className="grid gap-2 md:grid-cols-4">
              <MetricChip label="Packet" value={packet.id} />
              <MetricChip label="Risk" value={packet.riskClass} />
              <MetricChip label="Retrieved" value={packetRetrievedCount} />
              <MetricChip label="References" value={packetReferenceCount} />
            </div>
            <div className="rounded border border-white/10 bg-black/20 p-3 text-sm text-forge-mist">
              <div className="text-base font-semibold text-forge-ash">{packet.title}</div>
              <div className="mt-2">{packet.objective}</div>
              <div className="mt-2 text-xs text-forge-mist/80">Generated {formatTime(packet.generatedAtMs)} · adapter {packet.adapterTarget}</div>
            </div>

            <FoldSection title="Request" subtitle="User request, instructions, and selected path scope." defaultOpen>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-3 text-sm text-forge-mist">
                  <div>
                    <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">User Request</div>
                    <div className="mt-1 text-forge-ash">{packet.userRequest}</div>
                  </div>
                  <div>
                    <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">Instructions</div>
                    <div className="mt-1 whitespace-pre-wrap text-forge-ash">{packet.instructions || "—"}</div>
                  </div>
                </div>
                <div className="space-y-3">
                  <JsonBlock value={packet.expectedOutput} empty="No expected output contract." maxHeightClass="max-h-[220px]" />
                  <JsonBlock value={packet.scopeSnapshot} empty="No scope snapshot recorded." maxHeightClass="max-h-[220px]" />
                </div>
              </div>
              <div className="mt-3 flex flex-wrap gap-2">
                {packet.selectedPaths.length === 0 ? (
                  <span className="text-xs text-forge-mist/75">No selected paths recorded.</span>
                ) : (
                  packet.selectedPaths.map((path) => (
                    <span key={path} className="rounded border border-white/10 bg-black/25 px-2 py-1 font-mono text-[11px] text-forge-ash">
                      {path}
                    </span>
                  ))
                )}
              </div>
            </FoldSection>

            <FoldSection title="Payload & Retrieval" subtitle="Packet payload plus retrieved context and source references.">
              <div className="space-y-4">
                <JsonBlock value={packet.requestPayload} empty="No request payload recorded." />
                <JsonBlock value={packet.sourceReferences} empty="No source references recorded." />
                <JsonBlock value={packet.retrievedContext} empty="No retrieved context recorded." />
              </div>
            </FoldSection>

            <FoldSection title="Alignment Notes" subtitle="Why retrieved evidence was included in this packet." defaultOpen>
              {packetAlignment.length === 0 ? (
                <div className="text-sm text-forge-mist">No alignment notes recorded for this packet.</div>
              ) : (
                <div className="space-y-2">
                  {packetAlignment.map((note) => (
                    <div key={note.id} className="rounded border border-white/10 bg-black/20 p-3 text-sm text-forge-mist">
                      <div className="text-forge-ash">{note.note}</div>
                      <div className="mt-2 text-[11px] text-forge-mist/75">Recorded {formatTime(note.createdAtMs)}</div>
                    </div>
                  ))}
                </div>
              )}
            </FoldSection>

            <FoldSection title="Guidance" subtitle="Packet quality guidance, issues, and recommendations." defaultOpen>
              {packetGuidance.length === 0 ? (
                <div className="text-sm text-forge-mist">No guidance runs recorded for this packet.</div>
              ) : (
                <div className="space-y-3">
                  {packetGuidance.map((item) => (
                    <div key={item.id} className="rounded border border-white/10 bg-black/20 p-3">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="text-sm font-semibold text-forge-ash">Guidance score {item.guidanceScore}</div>
                        <div className="text-[11px] text-forge-mist/75">{formatTime(item.createdAtMs)}</div>
                      </div>
                      <div className="mt-2 text-sm text-forge-mist">
                        Issues: {item.issues.length > 0 ? item.issues.join(", ") : "none"}
                      </div>
                      <div className="mt-1 text-sm text-forge-mist">
                        Recommendations: {item.recommendations.length > 0 ? item.recommendations.join(", ") : "none"}
                      </div>
                      <div className="mt-3">
                        <JsonBlock value={item.evidence} empty="No guidance evidence recorded." maxHeightClass="max-h-[220px]" />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </FoldSection>
          </div>
        )}
      </Panel>

      <Panel title="Execution Trace" subtitle="Trace persisted execution evidence by correlation id or trace id.">
        <div className="grid gap-2 md:grid-cols-2">
          <input
            className="forge-input"
            placeholder="correlation id"
            value={correlationIdInput}
            onChange={(e) => setCorrelationIdInput(e.target.value)}
          />
          <input className="forge-input" placeholder="trace id" value={traceIdInput} onChange={(e) => setTraceIdInput(e.target.value)} />
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton disabled={traceBusy} onClick={() => void loadTraceInspector()}>
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
        {traceErr ? <div className="mt-3 rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{traceErr}</div> : null}
        {!traceLookup ? (
          <div className="mt-4 text-sm text-forge-mist">Load a correlation id or trace id to inspect the execution chain.</div>
        ) : (
          <div className="mt-4 space-y-4">
            {traceLookup.mode === "trace" && (traceLookup.correlationIds?.length ?? 0) > 1 ? (
              <div className="flex flex-wrap gap-2">
                {traceLookup.correlationIds?.map((correlationId) => (
                  <button
                    key={correlationId}
                    type="button"
                    onClick={() => setSelectedTraceCorrelationId(correlationId)}
                    className={[
                      "rounded border px-2.5 py-1 text-[11px]",
                      selectedTraceCorrelationId === correlationId ? "border-white/20 bg-white/10 text-forge-ash" : "border-white/10 bg-black/20 text-forge-mist",
                    ].join(" ")}
                  >
                    {correlationId}
                  </button>
                ))}
              </div>
            ) : null}

            {selectedTraceReport ? (
              <>
                <div className="grid gap-2 md:grid-cols-4">
                  <MetricChip label="Correlation" value={selectedTraceReport.correlationId} />
                  <MetricChip label="Audit Records" value={parsedTraceSummary.auditRecords.length} />
                  <MetricChip label="Gateway Calls" value={parsedTraceSummary.gatewayInvocations.length} />
                  <MetricChip label="Artifacts" value={parsedTraceSummary.artifactRecords.length} />
                </div>
                <div className="flex flex-wrap gap-2">
                  <SummaryLink to={`/audit?correlationId=${encodeURIComponent(selectedTraceReport.correlationId)}`} label="Open audit list" />
                  <SummaryLink
                    to={`/inspectors?correlationId=${encodeURIComponent(selectedTraceReport.correlationId)}`}
                    label="Filter snapshots by correlation"
                  />
                </div>

                <FoldSection title="Gateway Invocations" subtitle="Governed tool invocations tied to this correlation." defaultOpen>
                  <JsonBlock value={parsedTraceSummary.gatewayInvocations} empty="No gateway invocations linked to this correlation." />
                </FoldSection>

                <FoldSection title="Audit Records" subtitle="Append-only audit records for this correlation." defaultOpen>
                  <JsonBlock value={parsedTraceSummary.auditRecords} empty="No audit records linked to this correlation." />
                </FoldSection>

                <FoldSection title="Provenance & Journal" subtitle="Semantic provenance rows and journal events when present.">
                  <div className="space-y-4">
                    <JsonBlock value={parsedTraceSummary.provenanceRecords} empty="No provenance rows linked to this correlation." />
                    <JsonBlock value={parsedTraceSummary.journalEvents} empty="No journal events linked to this correlation." />
                  </div>
                </FoldSection>

                <FoldSection title="Artifacts & Links" subtitle="Stored artifacts, artifact refs, and inferred trace links.">
                  <div className="space-y-4">
                    <JsonBlock value={parsedTraceSummary.artifactRecords} empty="No artifact rows linked to this correlation." />
                    <JsonBlock value={parsedTraceSummary.artifactRefs} empty="No artifact refs linked to this correlation." />
                    <JsonBlock value={parsedTraceSummary.links} empty="No trace links recorded." />
                  </div>
                </FoldSection>
              </>
            ) : (
              <div className="text-sm text-forge-mist">No reports were found for the supplied correlation or trace id.</div>
            )}
          </div>
        )}
      </Panel>
    </div>
  );
}
