import type {
  ObservationVSADetail,
  VSAReindexRun,
  VSAReindexRunDetail,
} from "@forge/shared";
import { GhostButton, PrimaryButton } from "@forge/ui";

import { api } from "../../lib/api";
import { formatTime } from "../../lib/format";
import { EmptyState, Panel } from "./shared";
import { isOptionalEndpointMissing } from "./utils";

export function VsaReindexRunsPanel(props: {
  vsaRuns: VSAReindexRun[];
  selectedVSARunId: number | null;
  setSelectedVSARunId: (id: number) => void;
  vsaBusy: boolean;
  setVSABusy: (busy: boolean) => void;
  dossierId: string;
  staleOnly: boolean;
  selectedObsId: number | null;
  setObsVSADetail: (detail: ObservationVSADetail | null) => void;
  setStatus: (status: string) => void;
  setErr: (error: string | null) => void;
  loadVSARuns: () => Promise<void>;
}) {
  return (
    <Panel
      title="VSA Reindex Runs"
      subtitle="VSA pointer/binding/association refresh runs with inspectable outcomes."
    >
      <div className="mb-3 flex flex-wrap gap-2">
        <PrimaryButton
          disabled={props.vsaBusy}
          onClick={async () => {
            props.setVSABusy(true);
            try {
              const did = props.dossierId.trim()
                ? Number(props.dossierId.trim())
                : undefined;
              const res = await api.memory.runVSAReindex({
                dossierId: Number.isFinite(did) ? did : undefined,
                mode: "manual",
                triggeredBy: "operator",
                reason: "manual_reindex",
                note: "Manual VSA reindex from Memory page",
                limit: 150,
                staleOnly: props.staleOnly,
              });
              props.setStatus(
                `VSA reindex run ${res.detail.run.id}: indexed ${res.detail.run.indexed}, skipped ${res.detail.run.skipped}, failed ${res.detail.run.failed}.`,
              );
              props.setSelectedVSARunId(res.detail.run.id);
              await props.loadVSARuns();
              if (props.selectedObsId != null) {
                const vsa = await api.memory
                  .getObservationVSA(props.selectedObsId)
                  .catch(() => ({
                    detail: null as ObservationVSADetail | null,
                  }));
                props.setObsVSADetail(vsa.detail ?? null);
              }
            } catch (e) {
              if (isOptionalEndpointMissing(e)) {
                props.setStatus(
                  "VSA reindex endpoints are unavailable on this core build.",
                );
              } else {
                props.setErr(e instanceof Error ? e.message : String(e));
              }
            } finally {
              props.setVSABusy(false);
            }
          }}
        >
          {props.vsaBusy ? "Running VSA reindex..." : "Run VSA reindex"}
        </PrimaryButton>
        <GhostButton onClick={() => void props.loadVSARuns()}>
          Refresh VSA runs
        </GhostButton>
      </div>
      {props.vsaRuns.length === 0 ? (
        <EmptyState
          title="No VSA reindex runs"
          detail="Run VSA reindex to create an inspectable pointer, binding, and association refresh record."
        />
      ) : (
        <div className="space-y-2">
          {props.vsaRuns.map((run) => (
            <button
              key={run.id}
              type="button"
              onClick={() => props.setSelectedVSARunId(run.id)}
              className={[
                "w-full rounded border px-3 py-2 text-left",
                props.selectedVSARunId === run.id
                  ? "border-forge-ember/40 bg-black/30"
                  : "border-forge-platinum/10 bg-black/20 hover:border-forge-ember/35",
              ].join(" ")}
            >
              <div className="break-words text-xs font-semibold text-forge-ash">
                run #{run.id} · {run.mode} · {run.status}
              </div>
              <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                candidates {run.candidates} · indexed {run.indexed} · skipped{" "}
                {run.skipped} · failed {run.failed}
              </div>
              <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                {formatTime(run.createdAtMs)} · dossier{" "}
                {run.dossierId ?? "all"} · by {run.triggeredBy || "operator"}
              </div>
            </button>
          ))}
        </div>
      )}
    </Panel>
  );
}

export function VsaReindexDetailPanel(props: {
  vsaRunDetail: VSAReindexRunDetail | null;
}) {
  return (
    <Panel
      title="VSA Reindex Detail"
      subtitle="Per-observation VSA fingerprint transitions and indexing status."
    >
      {!props.vsaRunDetail ? (
        <EmptyState
          title="Select a VSA reindex run"
          detail="Choose a persisted VSA maintenance run to inspect fingerprint transitions and indexing outcomes."
        />
      ) : (
        <div className="space-y-2">
          <div className="rounded border border-forge-platinum/10 bg-black/20 p-2 text-xs text-forge-mist">
            run #{props.vsaRunDetail.run.id} · status{" "}
            {props.vsaRunDetail.run.status} · indexed{" "}
            {props.vsaRunDetail.run.indexed} /{" "}
            {props.vsaRunDetail.run.candidates}
          </div>
          {props.vsaRunDetail.items.length === 0 ? (
            <EmptyState
              title="No VSA reindex items"
              detail="This reindex run did not record itemized observation transitions."
            />
          ) : (
            props.vsaRunDetail.items.slice(0, 40).map((item) => (
              <div
                key={item.id}
                className="rounded border border-forge-platinum/10 bg-black/20 p-3"
              >
                <div className="text-xs font-semibold text-forge-ash">
                  item #{item.id} · obs {item.observationId} · {item.status}
                </div>
                <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                  {item.reason || "n/a"} · before{" "}
                  {item.beforeFingerprint || "none"} · after{" "}
                  {item.afterFingerprint || "none"}
                </div>
                {item.note ? (
                  <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                    {item.note}
                  </div>
                ) : null}
              </div>
            ))
          )}
        </div>
      )}
    </Panel>
  );
}
