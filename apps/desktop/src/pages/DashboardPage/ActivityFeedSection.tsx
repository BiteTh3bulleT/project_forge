import type { DashboardSummary, MemoryObservation } from "@forge/shared";

import { formatTime } from "../../lib/format";
import { activityRows } from "./dashboardData";

export function ActivityFeedSection(props: {
  recentImports: DashboardSummary["recentImports"];
  automation: DashboardSummary["automationActivity"];
  observations: MemoryObservation[];
  onNavigate: (route: string) => void;
}) {
  return (
    <div className="forge-ops-panel">
      <div className="forge-ops-panel__head">
        <div>
          <div className="forge-ops-title">Activity Feed</div>
          <div className="mt-1 text-xs text-forge-mist/65">
            Recent imports, automation, and observations.
          </div>
        </div>
      </div>
      <div className="forge-ops-panel__body space-y-3">
        {[
          ...activityRows(
            props.recentImports,
            props.automation,
            props.observations,
          ),
        ]
          .slice(0, 6)
          .map((item) => (
            <button
              key={item.key}
              type="button"
              onClick={() => props.onNavigate(item.route)}
              className="flex w-full items-start justify-between gap-3 rounded-md border border-forge-platinum/10 bg-black/20 px-3 py-2.5 text-left transition hover:border-forge-ember/35 hover:bg-black/30"
            >
              <span className="min-w-0">
                <span className="block truncate text-sm font-semibold text-forge-ash">
                  {item.title}
                </span>
                <span className="mt-0.5 block truncate text-xs text-forge-mist/65">
                  {item.detail}
                </span>
              </span>
              <span className="shrink-0 text-[11px] text-forge-mist/55">
                {formatTime(item.createdAtMs)}
              </span>
            </button>
          ))}
      </div>
    </div>
  );
}
