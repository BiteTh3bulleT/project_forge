import { GhostButton } from "@forge/ui";

import { FoldSection } from "../../components/FoldSection";
import { MetricRow, MiniEmpty, Panel } from "./components";
import type { CoreMeta, PcDiagnostics } from "./types";

export function DiagnosticsSection(props: {
  pcDiagnostics: PcDiagnostics | null;
  refreshDiagnostics: () => void;
}) {
  const { pcDiagnostics, refreshDiagnostics } = props;

  return (
    <FoldSection
      title="Diagnostics"
      subtitle="Machine/runtime visibility and host metrics."
      defaultOpen
    >
      <Panel
        title="PC Diagnostics"
        subtitle="Local process-level and rendering context visibility used for monitor/resource audits."
      >
        <div className="rounded-xl border border-forge-platinum/10 bg-black/20 p-3">
          {pcDiagnostics ? (
            <div className="space-y-2 text-sm text-forge-mist">
              <MetricRow label="Platform" value={pcDiagnostics.platform} />
              <MetricRow label="Language" value={pcDiagnostics.language} />
              <MetricRow label="Languages" value={pcDiagnostics.languages} />
              <MetricRow label="Processor Cores" value={pcDiagnostics.cores} />
              <MetricRow label="Device Memory" value={pcDiagnostics.memoryGiB} />
              <MetricRow
                label="Screen"
                value={`${pcDiagnostics.screenWidth}×${pcDiagnostics.screenHeight} @ DPR ${pcDiagnostics.pixelRatio}`}
              />
              <MetricRow
                label="Screen Available"
                value={`${pcDiagnostics.availWidth}×${pcDiagnostics.availHeight} (${pcDiagnostics.colorDepth}-bit)`}
              />
              <MetricRow
                label="JS Heap Used / Limit"
                value={`${pcDiagnostics.memoryUsedMB} / ${pcDiagnostics.memoryLimitMB}`}
              />
              <MetricRow label="User Agent" value={pcDiagnostics.userAgent} />
              <MetricRow label="Runtime Origin" value={pcDiagnostics.runtime} />
              {pcDiagnostics.desktop ? (
                <div className="mt-3 border-t border-forge-platinum/10 pt-3">
                  <div className="mb-2 text-xs font-semibold uppercase tracking-[0.14em] text-forge-mist">
                    Host / process diagnostics
                  </div>
                  <MetricRow
                    label="Host"
                    value={`${pcDiagnostics.desktop.hostName} · ${pcDiagnostics.desktop.osName} ${pcDiagnostics.desktop.osVersion} ${pcDiagnostics.desktop.architecture ?? ""}`}
                  />
                  <MetricRow
                    label="Kernel"
                    value={pcDiagnostics.desktop.kernelVersion ?? "unavailable"}
                  />
                  <MetricRow
                    label="Uptime"
                    value={`${Math.floor(pcDiagnostics.desktop.uptimeSeconds / 3600)}h ${Math.floor((pcDiagnostics.desktop.uptimeSeconds % 3600) / 60)}m`}
                  />
                  <MetricRow
                    label="CPU"
                    value={`${pcDiagnostics.desktop.cpuCount} logical cores${pcDiagnostics.desktop.process ? ` · process ${pcDiagnostics.desktop.process.cpuUsagePercent.toFixed(1)}%` : ""}`}
                  />
                  <MetricRow
                    label="Memory"
                    value={`${Math.round(pcDiagnostics.desktop.usedMemoryBytes / 1024 / 1024 / 1024)} GB used / ${Math.round(pcDiagnostics.desktop.totalMemoryBytes / 1024 / 1024 / 1024)} GB total`}
                  />
                  <MetricRow
                    label="Swap"
                    value={`${Math.round(pcDiagnostics.desktop.usedSwapBytes / 1024 / 1024)} MB used / ${Math.round(pcDiagnostics.desktop.totalSwapBytes / 1024 / 1024)} MB total`}
                  />
                  <MetricRow
                    label="Process"
                    value={
                      pcDiagnostics.desktop.process
                        ? `${pcDiagnostics.desktop.process.name} (${pcDiagnostics.desktop.process.pid})`
                        : "Unavailable"
                    }
                  />
                </div>
              ) : null}
            </div>
          ) : (
            <MiniEmpty
              title="Diagnostics unavailable"
              detail="Refresh diagnostics to collect browser and desktop host metrics."
            />
          )}
        </div>
        <div className="mt-3">
          <GhostButton onClick={() => refreshDiagnostics()}>
            Refresh diagnostics
          </GhostButton>
        </div>
      </Panel>
    </FoldSection>
  );
}

export function WorkspacePathsSection(props: { meta: CoreMeta | null }) {
  return (
    <FoldSection
      title="Workspace Paths"
      subtitle="Core data/database/workspace roots."
    >
      <Panel
        title="Workspace"
        subtitle="Local paths used by FORGE core for persistence and context generation."
      >
        {props.meta ? (
          <div className="space-y-2 text-sm text-forge-mist">
            <div>
              <span className="text-forge-ash">FORGE_DATA_DIR:</span>{" "}
              <span className="font-mono text-xs text-forge-ash">
                {props.meta.dataDir}
              </span>
            </div>
            <div>
              <span className="text-forge-ash">Database:</span>{" "}
              <span className="font-mono text-xs text-forge-ash">
                {props.meta.dbPath}
              </span>
            </div>
            <div>
              <span className="text-forge-ash">Workspace:</span>{" "}
              <span className="font-mono text-xs text-forge-ash">
                {props.meta.workspaceDir}
              </span>
            </div>
          </div>
        ) : (
          <MiniEmpty
            title="Core metadata unavailable"
            detail="Start or reconnect core to show data, database, and workspace paths."
          />
        )}
      </Panel>
    </FoldSection>
  );
}
