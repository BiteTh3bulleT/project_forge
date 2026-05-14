import type { MemoryRepairRun, MemoryRepairRunDetail } from "@forge/shared";
import { PrimaryButton } from "@forge/ui";

import { api } from "../../lib/api";
import { formatTime } from "../../lib/format";
import { EmptyState, Panel } from "./shared";

export function RepairRunsPanel(props: {
  repairRuns: MemoryRepairRun[];
  selectedRepairId: number | null;
  setSelectedRepairId: (id: number) => void;
  repairBusy: boolean;
  setRepairBusy: (busy: boolean) => void;
  dossierId: string;
  setStatus: (status: string) => void;
  loadRepairRuns: () => Promise<void>;
  loadObservations: () => Promise<void>;
}) {
  return (
    <Panel
      title="Memory Repair Runs"
      subtitle="Drift correction runs with candidate/repair/skip/failure counts and persisted run history."
    >
      <div className="mb-3 flex gap-2">
        <PrimaryButton
          disabled={props.repairBusy}
          onClick={async () => {
            props.setRepairBusy(true);
            try {
              const did = props.dossierId.trim()
                ? Number(props.dossierId.trim())
                : undefined;
              const res = await api.memory.runRepair({
                dossierId: Number.isFinite(did) ? did : undefined,
                maxAgeDays: 14,
                limit: 120,
                note: "Manual repair run from Memory page",
              });
              props.setStatus(
                `Repair run ${res.detail.run.id}: repaired ${res.detail.run.repaired}, skipped ${res.detail.run.skipped}, failed ${res.detail.run.failed}.`,
              );
              props.setSelectedRepairId(res.detail.run.id);
              await props.loadRepairRuns();
              await props.loadObservations();
            } finally {
              props.setRepairBusy(false);
            }
          }}
        >
          {props.repairBusy ? "Running repair..." : "Run repair"}
        </PrimaryButton>
      </div>
      {props.repairRuns.length === 0 ? (
        <EmptyState
          title="No repair runs"
          detail="Run memory repair to create a persisted maintenance record with candidate, repair, skip, and failure counts."
        />
      ) : (
        <div className="space-y-2">
          {props.repairRuns.map((run) => (
            <button
              key={run.id}
              type="button"
              onClick={() => props.setSelectedRepairId(run.id)}
              className={[
                "w-full rounded border px-3 py-2 text-left",
                props.selectedRepairId === run.id
                  ? "border-forge-ember/40 bg-black/30"
                  : "border-forge-platinum/10 bg-black/20 hover:border-forge-ember/35",
              ].join(" ")}
            >
              <div className="break-words text-xs font-semibold text-forge-ash">
                run #{run.id} · {run.mode}
              </div>
              <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                candidates {run.candidates} · repaired {run.repaired} · skipped{" "}
                {run.skipped} · failed {run.failed}
              </div>
              <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                {formatTime(run.createdAtMs)} · {run.note || "no note"}
              </div>
            </button>
          ))}
        </div>
      )}
    </Panel>
  );
}

export function RepairRunDetailPanel(props: {
  repairDetail: MemoryRepairRunDetail | null;
}) {
  return (
    <Panel
      title="Repair Run Detail"
      subtitle="Per-observation repair actions with before/after fields for inspectable drift correction."
    >
      {!props.repairDetail ? (
        <EmptyState
          title="Select a repair run"
          detail="Choose a persisted repair run to inspect per-observation drift correction items."
        />
      ) : (
        <div className="space-y-2">
          <div className="rounded border border-forge-platinum/10 bg-black/20 p-2 text-xs text-forge-mist">
            run #{props.repairDetail.run.id} · mode{" "}
            {props.repairDetail.run.mode} · repaired{" "}
            {props.repairDetail.run.repaired} /{" "}
            {props.repairDetail.run.candidates}
          </div>
          {props.repairDetail.items.length === 0 ? (
            <EmptyState
              title="No repair items"
              detail="This repair run completed without itemized observation changes."
            />
          ) : (
            props.repairDetail.items.slice(0, 40).map((item) => (
              <div
                key={item.id}
                className="rounded border border-forge-platinum/10 bg-black/20 p-3"
              >
                <div className="text-xs font-semibold text-forge-ash">
                  item #{item.id} · obs {item.observationId} · {item.status}
                </div>
                <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                  {item.issue} · {item.note}
                </div>
              </div>
            ))
          )}
        </div>
      )}
    </Panel>
  );
}
