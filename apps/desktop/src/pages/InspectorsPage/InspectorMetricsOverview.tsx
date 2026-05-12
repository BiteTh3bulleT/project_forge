import { InspectorMetric } from "./shared";

type InspectorMetricTone = "ok" | "warn" | "bad" | "muted";

export function InspectorMetricsOverview(props: {
  snapshotsCount: number;
  selectedSnapshotId: string;
  snapshotTone: InspectorMetricTone;
  dreamReportsCount: number;
  selectedDreamReportId: string;
  dreamTone: InspectorMetricTone;
  packetId: string | number;
  packetDetail: string;
  packetTone: InspectorMetricTone;
  traceReportCount: number;
  traceMode: string;
  traceTone: InspectorMetricTone;
  processReportCount: number;
  processRuntimeState: string;
  processTone: InspectorMetricTone;
}) {
  return (
    <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
      <InspectorMetric
        label="Snapshots"
        value={props.snapshotsCount}
        detail={props.selectedSnapshotId || "none selected"}
        tone={props.snapshotTone}
      />
      <InspectorMetric
        label="Dream Reports"
        value={props.dreamReportsCount}
        detail={props.selectedDreamReportId || "workspace scoped"}
        tone={props.dreamTone}
      />
      <InspectorMetric
        label="Packet"
        value={props.packetId}
        detail={props.packetDetail}
        tone={props.packetTone}
      />
      <InspectorMetric
        label="Trace Reports"
        value={props.traceReportCount}
        detail={props.traceMode}
        tone={props.traceTone}
      />
      <InspectorMetric
        label="Process"
        value={props.processReportCount}
        detail={props.processRuntimeState}
        tone={props.processTone}
      />
    </section>
  );
}
