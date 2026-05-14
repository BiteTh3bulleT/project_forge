import { GhostButton, PrimaryButton } from "@forge/ui";

import { FoldSection } from "../../components/FoldSection";
import type {
  ContextSnapshotInspectorDetail,
  ContextSnapshotInspectorSummary,
} from "../../lib/api";
import { formatTime } from "../../lib/format";
import type {
  RestoreScoreSummary,
  ResumeHintSummary,
} from "./inspectorsParsing";
import {
  CountPill,
  EmptyState,
  JsonBlock,
  MetricChip,
  Panel,
  SummaryLink,
} from "./shared";

type SnapshotInspectorPanelProps = {
  snapshotWorkspaceId: string;
  snapshotLaneId: string;
  snapshotKind: string;
  snapshotQuery: string;
  snapshotCorrelationId: string;
  selectedSnapshotId: string;
  snapshots: ContextSnapshotInspectorSummary[];
  snapshotDetail: ContextSnapshotInspectorDetail | null;
  snapshotBusy: boolean;
  snapshotErr: string | null;
  restoreScoreSummary: RestoreScoreSummary;
  resumeHintSummary: ResumeHintSummary;
  onWorkspaceIdChange: (value: string) => void;
  onLaneIdChange: (value: string) => void;
  onSnapshotKindChange: (value: string) => void;
  onSnapshotQueryChange: (value: string) => void;
  onCorrelationIdChange: (value: string) => void;
  onLoadSnapshots: () => void | Promise<void>;
  onClearFilters: () => void;
  onSelectSnapshot: (snapshotId: string) => void;
  onOpenParentSnapshot: (snapshotId: string) => void;
};

export function SnapshotInspectorPanel(props: SnapshotInspectorPanelProps) {
  return (
    <Panel
      title="Snapshot Inspector"
      subtitle="Inspect persisted context compilation snapshots, restore scoring, and resume hints."
    >
      <div className="mb-4 flex flex-wrap gap-2">
        <CountPill label="Snapshots" value={props.snapshots.length} />
        <CountPill
          label="Selected"
          value={props.selectedSnapshotId || "none"}
        />
        <CountPill
          label="Correlation filter"
          value={props.snapshotCorrelationId || "—"}
        />
      </div>
      <div className="grid gap-2 md:grid-cols-5">
        <input
          className="forge-input"
          placeholder="workspace id"
          value={props.snapshotWorkspaceId}
          onChange={(e) => props.onWorkspaceIdChange(e.target.value)}
        />
        <input
          className="forge-input"
          placeholder="lane id"
          value={props.snapshotLaneId}
          onChange={(e) => props.onLaneIdChange(e.target.value)}
        />
        <input
          className="forge-input"
          placeholder="snapshot kind"
          value={props.snapshotKind}
          onChange={(e) => props.onSnapshotKindChange(e.target.value)}
        />
        <input
          className="forge-input"
          placeholder="query filter"
          value={props.snapshotQuery}
          onChange={(e) => props.onSnapshotQueryChange(e.target.value)}
        />
        <input
          className="forge-input"
          placeholder="correlation id"
          value={props.snapshotCorrelationId}
          onChange={(e) => props.onCorrelationIdChange(e.target.value)}
        />
      </div>
      <div className="mt-3 flex flex-wrap gap-2">
        <PrimaryButton
          disabled={props.snapshotBusy}
          onClick={() => void props.onLoadSnapshots()}
        >
          {props.snapshotBusy ? "Loading…" : "Load snapshots"}
        </PrimaryButton>
        <GhostButton onClick={props.onClearFilters}>Clear filters</GhostButton>
      </div>
      {props.snapshotErr ? (
        <div className="mt-3">
          <EmptyState
            title="Snapshot lookup failed"
            detail={props.snapshotErr}
            tone="bad"
          />
        </div>
      ) : null}
      <div className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
        <div className="space-y-2">
          {props.snapshots.length === 0 ? (
            <EmptyState
              title={
                props.snapshotBusy
                  ? "Loading snapshots"
                  : "No snapshots matched"
              }
              detail={
                props.snapshotBusy
                  ? "Fetching persisted context snapshots for the current inspector scope."
                  : "Adjust workspace, lane, kind, query, or correlation filters and load snapshots again."
              }
            />
          ) : null}
          {props.snapshots.map((snapshot) => (
            <button
              key={snapshot.id}
              type="button"
              onClick={() => props.onSelectSnapshot(snapshot.id)}
              className={[
                "w-full rounded border px-3 py-3 text-left transition",
                props.selectedSnapshotId === snapshot.id
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
          {!props.snapshotDetail ? (
            <EmptyState
              title="Select a snapshot"
              detail="Choose a snapshot row to inspect persisted evidence, restore scoring, resume hints, graph, delta, and included ids."
            />
          ) : (
            <>
              <div className="grid gap-2 md:grid-cols-3">
                <MetricChip
                  label="Kind"
                  value={props.snapshotDetail.summary.snapshotKind || "—"}
                />
                <MetricChip
                  label="Workspace"
                  value={props.snapshotDetail.summary.workspaceId || "—"}
                />
                <MetricChip
                  label="Lane"
                  value={props.snapshotDetail.summary.laneId || "—"}
                />
              </div>
              <div className="flex flex-wrap gap-2">
                <CountPill
                  label="Evidence"
                  value={
                    props.snapshotDetail.summary.evidenceClass ||
                    "non_canonical_evidence"
                  }
                />
                <CountPill
                  label="Canonical commit"
                  value={
                    props.snapshotDetail.summary.nonCanonicalEvidence
                      ? "no"
                      : "unknown"
                  }
                />
              </div>
              <div className="rounded border border-white/10 bg-black/20 p-3 text-sm text-forge-mist">
                <div className="break-words font-medium text-forge-ash">
                  {props.snapshotDetail.summary.query ||
                    props.snapshotDetail.summary.id}
                </div>
                <div className="mt-2 flex flex-wrap gap-2">
                  {props.snapshotDetail.summary.correlationId ? (
                    <SummaryLink
                      to={`/inspectors?correlationId=${encodeURIComponent(props.snapshotDetail.summary.correlationId)}&snapshotId=${encodeURIComponent(props.snapshotDetail.summary.id)}`}
                      label={`Trace ${props.snapshotDetail.summary.correlationId}`}
                    />
                  ) : null}
                  {props.snapshotDetail.summary.parentSnapshotId ? (
                    <button
                      type="button"
                      className="rounded border border-white/15 bg-white/5 px-2.5 py-1 text-[11px] text-forge-mist transition hover:text-forge-ash"
                      onClick={() =>
                        props.onOpenParentSnapshot(
                          props.snapshotDetail!.summary.parentSnapshotId,
                        )
                      }
                    >
                      Open parent{" "}
                      {props.snapshotDetail.summary.parentSnapshotId}
                    </button>
                  ) : null}
                </div>
                <div className="mt-3 grid gap-2 md:grid-cols-2">
                  <div>
                    <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">
                      Fingerprint
                    </div>
                    <div className="mt-1 break-all font-mono text-[11px] text-forge-ash">
                      {props.snapshotDetail.summary.snapshotFingerprint || "—"}
                    </div>
                  </div>
                  <div>
                    <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">
                      Render Artifact Ref
                    </div>
                    <div className="mt-1 break-all font-mono text-[11px] text-forge-ash">
                      {props.snapshotDetail.summary.renderArtifactRefId || "—"}
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
                      {props.snapshotDetail.summary.selectedPaths.length ===
                      0 ? (
                        <span className="text-xs text-forge-mist/75">
                          No explicit path scope recorded.
                        </span>
                      ) : (
                        props.snapshotDetail.summary.selectedPaths.map(
                          (path) => (
                            <span
                              key={path}
                              className="min-w-0 break-all rounded border border-white/10 bg-black/25 px-2 py-1 font-mono text-[11px] text-forge-ash"
                            >
                              {path}
                            </span>
                          ),
                        )
                      )}
                    </div>
                  </div>
                  <div className="grid gap-2 grid-cols-2">
                    <MetricChip
                      label="State"
                      value={props.snapshotDetail.summary.counts.state}
                    />
                    <MetricChip
                      label="Open loops"
                      value={props.snapshotDetail.summary.counts.openLoops}
                    />
                    <MetricChip
                      label="Notes"
                      value={props.snapshotDetail.summary.counts.notes}
                    />
                    <MetricChip
                      label="Artifacts"
                      value={props.snapshotDetail.summary.counts.artifacts}
                    />
                  </div>
                </div>
                <div className="mt-3 grid gap-2 md:grid-cols-3">
                  <MetricChip
                    label="Correlation"
                    value={props.snapshotDetail.summary.correlationId || "—"}
                  />
                  <MetricChip
                    label="Trace"
                    value={props.snapshotDetail.summary.traceId || "—"}
                  />
                  <MetricChip
                    label="Syscall"
                    value={props.snapshotDetail.summary.syscallId || "—"}
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
                    value={props.snapshotDetail.budget}
                    empty="No budget recorded."
                  />
                  <JsonBlock
                    value={props.snapshotDetail.inclusionReasons}
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
                      value={props.restoreScoreSummary.decision || "—"}
                    />
                    <MetricChip
                      label="Threshold"
                      value={props.restoreScoreSummary.threshold}
                    />
                    <MetricChip
                      label="Top Score"
                      value={props.restoreScoreSummary.topScore.toFixed(3)}
                    />
                    <MetricChip
                      label="Candidates"
                      value={props.restoreScoreSummary.candidateCount}
                    />
                    <MetricChip
                      label="Selected"
                      value={props.restoreScoreSummary.selectedSnapshotId || "—"}
                    />
                    <MetricChip
                      label="Top Candidate"
                      value={props.restoreScoreSummary.topCandidateId || "—"}
                    />
                  </div>
                  {props.restoreScoreSummary.hasStructured ? (
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
                          {props.restoreScoreSummary.candidates.map(
                            (candidate) => (
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
                            ),
                          )}
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
                          value={props.resumeHintSummary.nextAction || "—"}
                        />
                        <MetricChip
                          label="Preferred Snapshot"
                          value={
                            props.resumeHintSummary.preferredSnapshotId || "—"
                          }
                        />
                        <MetricChip
                          label="Restore Confidence"
                          value={props.resumeHintSummary.restoreConfidence.toFixed(
                            3,
                          )}
                        />
                        <MetricChip
                          label="Requires Fresh Compile"
                          value={
                            props.resumeHintSummary.requiresFreshCompile
                              ? "yes"
                              : "no"
                          }
                        />
                      </div>
                      <div className="mt-2 flex flex-wrap gap-2 text-xs text-forge-mist">
                        <span>
                          Top blockers:{" "}
                          {props.resumeHintSummary.topBlockers.length > 0
                            ? props.resumeHintSummary.topBlockers.join(", ")
                            : "none"}
                        </span>
                        <span>
                          Dominant states:{" "}
                          {props.resumeHintSummary.dominantStateKeys.length > 0
                            ? props.resumeHintSummary.dominantStateKeys.join(
                                ", ",
                              )
                            : "none"}
                        </span>
                        <span>
                          Dominant loops:{" "}
                          {props.resumeHintSummary.dominantLoopIds.length > 0
                            ? props.resumeHintSummary.dominantLoopIds.join(", ")
                            : "none"}
                        </span>
                      </div>
                    </div>
                    <JsonBlock
                      value={props.snapshotDetail.resumeHints}
                      empty="No resume hints recorded."
                    />
                  </div>
                  <JsonBlock
                    value={props.snapshotDetail.restoreScores}
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
                    value={props.snapshotDetail.header}
                    empty="No header evidence recorded."
                  />
                  <JsonBlock
                    value={props.snapshotDetail.graph}
                    empty="No graph evidence recorded."
                  />
                  <JsonBlock
                    value={props.snapshotDetail.delta}
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
                    value={props.snapshotDetail.metadata}
                    empty="No snapshot details recorded."
                  />
                  <JsonBlock
                    value={{
                      state: props.snapshotDetail.includedStateIds,
                      openLoops: props.snapshotDetail.includedOpenLoops,
                      notes: props.snapshotDetail.includedNoteIds,
                      links: props.snapshotDetail.includedLinkIds,
                      models: props.snapshotDetail.includedModelIds,
                      artifacts: props.snapshotDetail.includedArtifactIds,
                      events: props.snapshotDetail.includedEventIds,
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
  );
}
