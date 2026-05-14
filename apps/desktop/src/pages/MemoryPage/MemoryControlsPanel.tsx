import { GhostButton, PrimaryButton } from "@forge/ui";

import { FoldSection } from "../../components/FoldSection";
import {
  EmptyState,
  GateLine,
  Panel,
  type MaintenanceGate,
  type MemoryView,
} from "./shared";

export function MemoryControlsPanel(props: {
  memoryView: MemoryView;
  setMemoryView: (view: MemoryView) => void;
  localQ: string;
  setLocalQ: (value: string) => void;
  obsType: string;
  setObsType: (value: string) => void;
  dossierId: string;
  setDossierId: (value: string) => void;
  staleOnly: boolean;
  setStaleOnly: (value: boolean) => void;
  maintenanceGates: MaintenanceGate[];
  err: string | null;
  onRefreshAll: () => void;
  onClearQuery: () => void;
  onRunSearch: () => void;
  onApplyObservationFilters: () => void;
  onRefreshVSARuns: () => void;
}) {
  return (
    <Panel
      title="Memory Controls"
      subtitle="Search scope, observation filters, view selection, and maintenance preflight gates."
      actions={
        <div className="flex flex-wrap gap-2">
          <label className="text-[11px] text-forge-mist">
            View
            <select
              className="forge-input ml-2 px-2 py-1 text-[11px]"
              value={props.memoryView}
              onChange={(e) => props.setMemoryView(e.target.value as MemoryView)}
            >
              <option value="inspect">Recent episodes</option>
              <option value="search">Search chunks</option>
              <option value="all">All surfaces</option>
              <option value="maintenance">Maintenance</option>
            </select>
          </label>
          <GhostButton onClick={props.onRefreshAll}>
            Refresh observations
          </GhostButton>
          <GhostButton onClick={props.onClearQuery}>Clear query</GhostButton>
        </div>
      }
    >
      <FoldSection
        title="Search and filter scope"
        subtitle="Set query and observation filters before running actions."
        defaultOpen
      >
        <div className="grid gap-3 md:grid-cols-6">
          <div className="md:col-span-3">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">
              Query
            </label>
            <input
              className="forge-input mt-1"
              value={props.localQ}
              onChange={(e) => props.setLocalQ(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  props.onRunSearch();
                }
              }}
              placeholder="search code and indexed content"
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">
              Observation type
            </label>
            <input
              className="forge-input mt-1"
              value={props.obsType}
              onChange={(e) => props.setObsType(e.target.value)}
              placeholder="retrieval_result"
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">
              Dossier id
            </label>
            <input
              className="forge-input mt-1"
              value={props.dossierId}
              onChange={(e) => props.setDossierId(e.target.value)}
              placeholder="optional"
            />
          </div>
          <div className="flex items-end gap-2">
            <label className="inline-flex items-center gap-2 text-xs text-forge-mist">
              <input
                type="checkbox"
                checked={props.staleOnly}
                onChange={(e) => props.setStaleOnly(e.target.checked)}
              />
              Stale only
            </label>
          </div>
        </div>
        <div className="mt-3 flex gap-2">
          <PrimaryButton onClick={props.onRunSearch}>Run search</PrimaryButton>
          <GhostButton onClick={props.onApplyObservationFilters}>
            Apply observation filters
          </GhostButton>
          <GhostButton onClick={props.onRefreshVSARuns}>
            Refresh VSA runs
          </GhostButton>
        </div>
      </FoldSection>
      <FoldSection
        title="Maintenance preflight gates"
        subtitle="If/and checks before running repair or VSA maintenance."
      >
        <div className="space-y-1 rounded border border-forge-platinum/10 bg-black/25 p-3 text-xs">
          {props.maintenanceGates.map((gate, idx) => (
            <GateLine
              key={gate.label}
              prefix={idx === 0 ? "IF" : "AND"}
              label={gate.label}
              pass={gate.pass}
            />
          ))}
        </div>
      </FoldSection>
      {props.err ? (
        <div className="mt-3">
          <EmptyState
            title="Memory request failed"
            detail={props.err}
            tone="bad"
          />
        </div>
      ) : null}
    </Panel>
  );
}
