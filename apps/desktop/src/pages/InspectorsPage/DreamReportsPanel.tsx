import { GhostButton, PrimaryButton } from "@forge/ui";

import { FoldSection } from "../../components/FoldSection";
import type { DreamReportDetail, DreamReportSummary } from "../../lib/api";
import { formatTime } from "../../lib/format";
import { CountPill, EmptyState, JsonBlock, MetricChip, Panel } from "./shared";

type DreamReportsPanelProps = {
  dreamWorkspaceId: string;
  dreamLaneId: string;
  dreamMode: string;
  selectedDreamReportId: string;
  dreamReports: DreamReportSummary[];
  dreamReportDetail: DreamReportDetail | null;
  dreamBusy: boolean;
  dreamErr: string | null;
  onWorkspaceIdChange: (value: string) => void;
  onLaneIdChange: (value: string) => void;
  onModeChange: (value: string) => void;
  onLoadDreamReports: () => void | Promise<void>;
  onClearDreamFilters: () => void;
  onSelectDreamReport: (reportId: string) => void;
};

export function DreamReportsPanel(props: DreamReportsPanelProps) {
  return (
    <Panel
      title="Dream Reports"
      subtitle="Inspect persisted Dream Mode dry-run reports as non-canonical evidence."
    >
      <div className="mb-4 flex flex-wrap gap-2">
        <CountPill label="Reports" value={props.dreamReports.length} />
        <CountPill
          label="Selected"
          value={props.selectedDreamReportId || "none"}
        />
        <CountPill label="Evidence" value="non-canonical" />
        <CountPill label="Commit mode" value="disabled" />
      </div>
      <div className="grid gap-2 md:grid-cols-3">
        <input
          className="forge-input"
          placeholder="workspace id"
          value={props.dreamWorkspaceId}
          onChange={(e) => props.onWorkspaceIdChange(e.target.value)}
        />
        <input
          className="forge-input"
          placeholder="lane id"
          value={props.dreamLaneId}
          onChange={(e) => props.onLaneIdChange(e.target.value)}
        />
        <input
          className="forge-input"
          placeholder="mode"
          value={props.dreamMode}
          onChange={(e) => props.onModeChange(e.target.value)}
        />
      </div>
      <div className="mt-3 flex flex-wrap gap-2">
        <PrimaryButton
          disabled={props.dreamBusy}
          onClick={() => void props.onLoadDreamReports()}
        >
          {props.dreamBusy ? "Loading…" : "Load Dream reports"}
        </PrimaryButton>
        <GhostButton onClick={props.onClearDreamFilters}>
          Clear Dream filters
        </GhostButton>
      </div>
      {props.dreamErr ? (
        <div className="mt-3">
          <EmptyState
            title="Dream report lookup failed"
            detail={props.dreamErr}
            tone="bad"
          />
        </div>
      ) : null}
      <div className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
        <div className="space-y-2">
          {props.dreamReports.length === 0 ? (
            <EmptyState
              title={
                props.dreamBusy
                  ? "Loading Dream reports"
                  : "No Dream reports matched"
              }
              detail={
                props.dreamBusy
                  ? "Fetching dry-run Dream reports for the current workspace scope."
                  : "Provide a workspace id or adjust lane and mode filters to inspect non-canonical Dream evidence."
              }
            />
          ) : null}
          {props.dreamReports.map((report) => (
            <button
              key={report.id}
              type="button"
              onClick={() => props.onSelectDreamReport(report.id)}
              className={[
                "w-full rounded border px-3 py-3 text-left transition",
                props.selectedDreamReportId === report.id
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
          {!props.dreamReportDetail ? (
            <EmptyState
              title="Select a Dream report"
              detail="Load a workspace and choose a report to inspect replay candidates, salience scores, proposals, warnings, and trace evidence."
            />
          ) : (
            <>
              <div className="grid gap-2 md:grid-cols-4">
                <MetricChip
                  label="Mode"
                  value={props.dreamReportDetail.mode || "—"}
                />
                <MetricChip
                  label="Status"
                  value={props.dreamReportDetail.status || "—"}
                />
                <MetricChip
                  label="Dry Run"
                  value={props.dreamReportDetail.dryRun ? "yes" : "no"}
                />
                <MetricChip
                  label="Canonical Commit"
                  value={
                    props.dreamReportDetail.canonicalWriteCommitted
                      ? "yes"
                      : "no"
                  }
                />
              </div>
              <div className="flex flex-wrap gap-2">
                <CountPill
                  label="Evidence"
                  value={
                    props.dreamReportDetail.evidenceClass ||
                    "non_canonical_evidence"
                  }
                />
                <CountPill
                  label="Candidates"
                  value={props.dreamReportDetail.candidates?.length ?? 0}
                />
                <CountPill
                  label="Salience"
                  value={props.dreamReportDetail.salienceScores?.length ?? 0}
                />
                <CountPill
                  label="Review"
                  value={
                    [
                      ...(props.dreamReportDetail.repairProposals ?? []),
                      ...(props.dreamReportDetail.snapshotHygieneProposals ??
                        []),
                    ].length
                  }
                />
              </div>
              <div className="rounded border border-white/10 bg-black/20 p-3 text-sm text-forge-mist">
                <div className="break-all font-mono text-[11px] text-forge-ash">
                  {props.dreamReportDetail.id}
                </div>
                <div className="mt-2 flex flex-wrap gap-2 text-xs">
                  <span>
                    workspace {props.dreamReportDetail.workspaceId || "—"}
                  </span>
                  <span>lane {props.dreamReportDetail.laneId || "—"}</span>
                  <span>
                    corr {props.dreamReportDetail.correlationId || "—"}
                  </span>
                  <span>trace {props.dreamReportDetail.traceId || "—"}</span>
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
                    value={props.dreamReportDetail.candidatesConsidered ?? 0}
                  />
                  <MetricChip
                    label="Generated"
                    value={props.dreamReportDetail.proposalsGenerated ?? 0}
                  />
                  <MetricChip
                    label="Warnings"
                    value={props.dreamReportDetail.warnings?.length ?? 0}
                  />
                </div>
                <div className="mt-3">
                  <JsonBlock
                    value={props.dreamReportDetail.summary}
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
                    value={props.dreamReportDetail.candidates}
                    empty="No replay candidates recorded."
                    maxHeightClass="max-h-[280px]"
                  />
                  <JsonBlock
                    value={props.dreamReportDetail.salienceScores}
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
                    value={props.dreamReportDetail.memoryTierProposals}
                    empty="No memory-tier proposals recorded."
                    maxHeightClass="max-h-[280px]"
                  />
                  <JsonBlock
                    value={props.dreamReportDetail.repairProposals}
                    empty="No repair proposals recorded."
                    maxHeightClass="max-h-[280px]"
                  />
                  <JsonBlock
                    value={props.dreamReportDetail.snapshotHygieneProposals}
                    empty="No snapshot hygiene proposals recorded."
                    maxHeightClass="max-h-[280px]"
                  />
                  <JsonBlock
                    value={props.dreamReportDetail.warnings}
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
                    value={props.dreamReportDetail.trace}
                    empty="No trace details recorded."
                  />
                  <JsonBlock
                    value={props.dreamReportDetail.metadata}
                    empty="No report details recorded."
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
