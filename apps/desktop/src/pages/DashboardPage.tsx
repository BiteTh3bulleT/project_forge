import type { DashboardSummary, MemoryObservation } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { FoldSection } from "../components/FoldSection";
import { api, type ModelRuntimeHealth, type ModelRuntimeQueueStatus, type ModelRuntimeUsageSummary } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

type CapabilityRecord = Awaited<ReturnType<typeof api.gateway.capabilities>>["capabilities"][number];
type InvocationRecord = Awaited<ReturnType<typeof api.gateway.invocations>>["invocations"][number];

export function DashboardPage() {
  const navigate = useNavigate();
  const uiMode = useUiStore((s) => s.uiMode);
  const setStatus = useUiStore((s) => s.setStatusLine);

  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [health, setHealth] = useState<ModelRuntimeHealth | null>(null);
  const [queue, setQueue] = useState<ModelRuntimeQueueStatus | null>(null);
  const [usage, setUsage] = useState<ModelRuntimeUsageSummary | null>(null);
  const [capabilities, setCapabilities] = useState<CapabilityRecord[]>([]);
  const [invocations, setInvocations] = useState<InvocationRecord[]>([]);
  const [observations, setObservations] = useState<MemoryObservation[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [lastUpdatedAt, setLastUpdatedAt] = useState(Date.now());

  async function load() {
    setLoading(true);
    try {
      const [dash, coreHealth, gatewayCaps, gatewayInvs, memoryObs] = await Promise.all([
        api.dashboard.summary(),
        api.health().catch(() => null),
        api.gateway.capabilities().catch(() => ({ capabilities: [] as CapabilityRecord[] })),
        api.gateway.invocations({ limit: 80 }).catch(() => ({ invocations: [] as InvocationRecord[] })),
        api.memory.listObservations({ limit: 48 }).catch(() => ({ observations: [] as MemoryObservation[] })),
      ]);
      const runtimeAvailable = coreHealth?.modelRuntime?.available === true;
      const [healthRes, queueRes, usageRes] = runtimeAvailable
        ? await Promise.all([
            api.modelRuntime.health().catch(() => ({ health: null as ModelRuntimeHealth | null })),
            api.modelRuntime.queue().catch(() => ({ queue: null as ModelRuntimeQueueStatus | null })),
            api.modelRuntime.usage().catch(() => ({ usage: null as ModelRuntimeUsageSummary | null })),
          ])
        : [
            { health: coreHealth ? ({ ok: false, status: coreHealth.modelRuntime?.status || "not enabled", backend: "not configured" } satisfies ModelRuntimeHealth) : null },
            { queue: { depth: 0, scheduler: "not enabled" } satisfies ModelRuntimeQueueStatus },
            { usage: emptyUsage() },
          ];
      setSummary(dash);
      setHealth(healthRes.health);
      setQueue(queueRes.queue);
      setUsage(usageRes.usage);
      setCapabilities(Array.isArray(gatewayCaps.capabilities) ? gatewayCaps.capabilities : []);
      setInvocations(Array.isArray(gatewayInvs.invocations) ? gatewayInvs.invocations : []);
      setObservations(Array.isArray(memoryObs.observations) ? memoryObs.observations : []);
      setLastUpdatedAt(Date.now());
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
    const id = window.setInterval(() => void load(), 5000);
    return () => window.clearInterval(id);
  }, []);

  const activeJobs = Array.isArray(summary?.activeJobs) ? summary.activeJobs : [];
  const recentFailures = Array.isArray(summary?.recentFailures) ? summary.recentFailures : [];
  const recentImports = Array.isArray(summary?.recentImports) ? summary.recentImports : [];
  const attention = (summary?.approvalsPending ?? 0) + (summary?.reviewsPending ?? 0) + recentFailures.length;
  const currentState = attention > 0 ? "blocked" : activeJobs.length > 0 ? "running" : "idle";
  const topBlocker = getTopBlocker(summary, recentFailures);
  const nextAction = getNextAction(summary, activeJobs, recentFailures);
  const systemStatusRows = useMemo(() => flattenSystemStatus(summary?.systemStatus), [summary?.systemStatus]);
  const capabilityStatusCounts = useMemo(() => countBy(capabilities, (row) => row.status || "unknown"), [capabilities]);
  const invocationStatusCounts = useMemo(() => countBy(invocations, (row) => row.status || "unknown"), [invocations]);
  const degradedRuntime = health && (!health.ok || String(health.status || "").toLowerCase().includes("degraded"));

  if (uiMode === "cognitive") {
    return (
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-5">
        <Panel
          title="Current State"
          subtitle="Cognitive mode keeps the operator on context, active work, blockers, and the next action."
          actions={<GhostButton onClick={() => void load()} disabled={loading}>{loading ? "Refreshing..." : "Refresh"}</GhostButton>}
        >
          {err ? <div className="mb-3 rounded-xl border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
          {!summary ? (
            <div className="text-sm text-forge-mist">Loading current state...</div>
          ) : (
            <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_18rem]">
              <div className="space-y-4">
                <div className="rounded-2xl bg-black/20 p-4">
                  <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-forge-mist/60">Status</div>
                  <div className="mt-2 text-2xl font-semibold text-forge-ash">{currentState}</div>
                  <div className="mt-2 text-sm leading-6 text-forge-mist">{summaryLine(summary, activeJobs, recentFailures)}</div>
                </div>
                <div className="grid gap-3 md:grid-cols-2">
                  <FocusRow label="Active task" value={activeJobs[0]?.title || "No active job"} onClick={() => navigate(activeJobs[0] ? `/jobs/${activeJobs[0].id}` : "/jobs")} />
                  <FocusRow label="Top blocker" value={topBlocker} onClick={() => navigate((summary?.approvalsPending ?? 0) > 0 ? "/approvals" : (summary?.reviewsPending ?? 0) > 0 ? "/reviews" : "/events")} />
                  <FocusRow label="Restore summary" value="Header-first restore package available through snapshot inspector" onClick={() => navigate("/inspectors")} />
                  <FocusRow label="Suggested next action" value={nextAction.label} onClick={() => navigate(nextAction.route)} />
                </div>
              </div>
              <div className="rounded-2xl bg-black/20 p-4">
                <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-forge-mist/60">Next Action</div>
                <div className="mt-3 text-sm font-semibold text-forge-ash">{nextAction.label}</div>
                <div className="mt-2 text-xs leading-5 text-forge-mist">{nextAction.reason}</div>
                <PrimaryButton className="mt-4 w-full" onClick={() => navigate(nextAction.route)}>Open</PrimaryButton>
              </div>
            </div>
          )}
        </Panel>

        {attention > 0 ? (
          <Panel title="Attention" subtitle="Shown only when operator attention is needed.">
            <div className="space-y-2">
              {(summary?.approvalsPending ?? 0) > 0 ? <AttentionRow label="Approvals pending" value={summary?.approvalsPending ?? 0} route="/approvals" /> : null}
              {(summary?.reviewsPending ?? 0) > 0 ? <AttentionRow label="Reviews pending" value={summary?.reviewsPending ?? 0} route="/reviews" /> : null}
              {recentFailures.length > 0 ? <AttentionRow label="Recent failures" value={recentFailures.length} route="/events" /> : null}
            </div>
          </Panel>
        ) : null}
      </div>
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-5">
      <Panel
        title="System Metrics Board"
        subtitle="Secondary mode for operational telemetry. Sections stay collapsed or compact until inspection is needed."
        actions={
          <div className="flex items-center gap-2">
            <GhostButton onClick={() => void load()} disabled={loading}>{loading ? "Refreshing..." : "Refresh"}</GhostButton>
            <GhostButton onClick={() => setStatus(`Metrics refreshed at ${formatTime(lastUpdatedAt)}`)}>Updated {formatTime(lastUpdatedAt)}</GhostButton>
          </div>
        }
      >
        {err ? <div className="rounded-xl border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <CompactMetric label="CPU/RAM core" value={summary ? "online" : "unknown"} />
          <CompactMetric label="Runtime" value={health?.status || (health?.ok ? "ok" : "unknown")} warn={Boolean(degradedRuntime)} />
          <CompactMetric label="Queue depth" value={String(queue?.depth ?? activeJobs.length)} />
          <CompactMetric label="Gateway calls" value={String(invocations.length)} />
        </div>
      </Panel>

      <FoldSection title="Resource Usage" subtitle="Host and core status fields from diagnostics." defaultOpen>
        <SectionRows rows={systemStatusRows.length > 0 ? systemStatusRows : [["Status", "No system status fields available"]]} />
      </FoldSection>

      <FoldSection title="Job Pipeline" subtitle="Active job queue and recent execution pressure." defaultOpen>
        {activeJobs.length === 0 ? (
          <div className="text-sm text-forge-mist">No active jobs.</div>
        ) : (
          <div className="space-y-2">
            {activeJobs.slice(0, 10).map((job) => (
              <button key={job.id} type="button" onClick={() => navigate(`/jobs/${job.id}`)} className="forge-window-action">
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-semibold text-forge-ash">{job.title}</span>
                  <span className="mt-1 block truncate text-[11px]">{job.status} | {job.targetAdapter} | {formatTime(job.createdAtMs)}</span>
                </span>
              </button>
            ))}
          </div>
        )}
      </FoldSection>

      <FoldSection title="Model Runtime Health" subtitle="Runtime, loaded-model, and scheduler state.">
        {health && !health.ok && (health.backend === "not configured" || health.status === "unavailable") ? (
          <div className="mb-3 rounded-xl border border-forge-platinum/10 bg-black/20 p-3 text-sm text-forge-mist">
            FORGE core is online. Governed modelruntime is not enabled for this process. Chat may still use the configured Ollama adapter, but registry/load controls require
            `FORGE_ENABLE_MODEL_RUNTIME=true` plus a backend endpoint such as `FORGE_LLAMA_CPP_ENDPOINT` or `FORGE_MODEL_OPENAI_COMPAT_ENDPOINT`.
          </div>
        ) : null}
        <SectionRows
          rows={[
            ["Health", health?.status || (health?.ok ? "ok" : "unknown")],
            ["Backend", health?.backend || "unknown"],
            ["Registered models", String(usage?.registered ?? 0)],
            ["Loaded models", String(usage?.loaded ?? 0)],
            ["Queue scheduler", queue?.scheduler || "unknown"],
            ["Queue depth", String(queue?.depth ?? 0)],
          ]}
        />
      </FoldSection>

      <FoldSection title="Vector DB State" subtitle="Retrieval evidence remains non-canonical; this section reports index freshness and observation flow.">
        <SectionRows
          rows={[
            ["Recent observations", String(observations.length)],
            ["Stale observations", String(observations.filter((item) => item.stale).length)],
            ["Recent imports", String(recentImports.length)],
            ["Truth authority", "kernel only"],
          ]}
        />
      </FoldSection>

      <FoldSection title="Gateway Activity" subtitle="Capability registry and invocation outcomes.">
        <div className="grid gap-4 lg:grid-cols-2">
          <Distribution title="Capability Status" counts={capabilityStatusCounts} />
          <Distribution title="Invocation Outcomes" counts={invocationStatusCounts} />
        </div>
      </FoldSection>

      <FoldSection title="Diagnostics" subtitle="Routes for detailed drill-down open as focused surfaces or floating windows.">
        <div className="flex flex-wrap gap-2">
          <GhostButton onClick={() => navigate("/inspectors")}>Inspectors</GhostButton>
          <GhostButton onClick={() => navigate("/events")}>Logs</GhostButton>
          <GhostButton onClick={() => navigate("/gateway")}>Gateway</GhostButton>
          <GhostButton onClick={() => navigate("/models")}>Models</GhostButton>
        </div>
      </FoldSection>
    </div>
  );
}

function FocusRow(props: { label: string; value: string; onClick: () => void }) {
  return (
    <button type="button" onClick={props.onClick} className="rounded-2xl bg-black/20 p-4 text-left transition hover:bg-black/30">
      <div className="text-[10px] font-semibold uppercase tracking-[0.16em] text-forge-mist/55">{props.label}</div>
      <div className="mt-2 line-clamp-3 text-sm text-forge-ash">{props.value}</div>
    </button>
  );
}

function AttentionRow(props: { label: string; value: number; route: string }) {
  const navigate = useNavigate();
  return (
    <button type="button" onClick={() => navigate(props.route)} className="flex w-full items-center justify-between rounded-2xl border border-forge-ember/25 bg-forge-ember/10 px-4 py-3 text-left">
      <span className="text-sm text-forge-ash">{props.label}</span>
      <span className="forge-chip forge-chip--warn">{props.value}</span>
    </button>
  );
}

function CompactMetric(props: { label: string; value: string; warn?: boolean }) {
  return (
    <div className="rounded-2xl bg-black/20 p-3">
      <div className="text-[10px] font-semibold uppercase tracking-[0.14em] text-forge-mist/55">{props.label}</div>
      <div className={props.warn ? "mt-1 text-sm font-semibold text-forge-emberSoft" : "mt-1 text-sm font-semibold text-forge-ash"}>{props.value}</div>
    </div>
  );
}

function SectionRows(props: { rows: Array<[string, string]> }) {
  return (
    <div className="grid gap-2 md:grid-cols-2">
      {props.rows.map(([label, value]) => (
        <div key={label} className="flex items-center justify-between gap-3 rounded-xl bg-black/20 px-3 py-2 text-xs text-forge-mist">
          <span>{label}</span>
          <span className="text-right font-semibold text-forge-ash">{value}</span>
        </div>
      ))}
    </div>
  );
}

function Distribution(props: { title: string; counts: Record<string, number> }) {
  const entries = Object.entries(props.counts).sort((a, b) => b[1] - a[1]);
  const total = entries.reduce((sum, item) => sum + item[1], 0);
  return (
    <div className="rounded-2xl bg-black/20 p-4">
      <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-forge-mist/65">{props.title}</div>
      <div className="mt-3 space-y-2">
        {entries.length === 0 ? <div className="text-xs text-forge-mist">No data.</div> : null}
        {entries.map(([label, value]) => {
          const width = total <= 0 ? 0 : Math.max(4, Math.round((value / total) * 100));
          return (
            <div key={label}>
              <div className="flex justify-between text-xs text-forge-mist">
                <span>{label}</span>
                <span className="text-forge-ash">{value}</span>
              </div>
              <div className="mt-1 h-1.5 rounded bg-forge-platinum/10">
                <div className="h-full rounded bg-forge-electric/70" style={{ width: `${width}%` }} />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function getTopBlocker(summary: DashboardSummary | null, recentFailures: unknown[]) {
  if ((summary?.approvalsPending ?? 0) > 0) return `${summary?.approvalsPending ?? 0} approval gate(s) waiting`;
  if ((summary?.reviewsPending ?? 0) > 0) return `${summary?.reviewsPending ?? 0} review item(s) waiting`;
  if (recentFailures.length > 0) return `${recentFailures.length} recent failure(s) need inspection`;
  return "No active blocker";
}

function getNextAction(summary: DashboardSummary | null, activeJobs: DashboardSummary["activeJobs"], recentFailures: unknown[]) {
  if ((summary?.approvalsPending ?? 0) > 0) return { label: "Resolve approval queue", route: "/approvals", reason: "Execution is gated until explicit operator decisions are recorded." };
  if ((summary?.reviewsPending ?? 0) > 0) return { label: "Review pending outputs", route: "/reviews", reason: "Generated/imported outputs need explicit disposition." };
  if (recentFailures.length > 0) return { label: "Inspect recent failures", route: "/events", reason: "Failure evidence should be inspected before starting more work." };
  if (activeJobs.length > 0) return { label: "Track active task", route: "/jobs", reason: "A job is already running; stay with its event stream and artifacts." };
  return { label: "Continue in chat", route: "/chat", reason: "No blockers are visible; the cognitive workspace is ready for the next operator request." };
}

function summaryLine(summary: DashboardSummary, activeJobs: DashboardSummary["activeJobs"], recentFailures: unknown[]) {
  const parts = [
    `${activeJobs.length} active job(s)`,
    `${summary.approvalsPending ?? 0} approval(s)`,
    `${summary.reviewsPending ?? 0} review(s)`,
    `${recentFailures.length} recent failure(s)`,
  ];
  return parts.join(" | ");
}

function flattenSystemStatus(raw: Record<string, unknown> | null | undefined): Array<[string, string]> {
  if (!raw || typeof raw !== "object") return [];
  return Object.entries(raw).map(([key, value]) => [humanizeKey(key), summarizeValue(value)]);
}

function summarizeValue(value: unknown) {
  if (value == null) return "-";
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") return String(value);
  if (Array.isArray(value)) return `${value.length} item(s)`;
  return "available";
}

function humanizeKey(key: string) {
  return key
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/[_-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .replace(/^\w/, (c) => c.toUpperCase());
}

function countBy<T>(rows: T[], fn: (row: T) => string) {
  return rows.reduce<Record<string, number>>((acc, row) => {
    const key = fn(row);
    acc[key] = (acc[key] ?? 0) + 1;
    return acc;
  }, {});
}

function emptyUsage(): ModelRuntimeUsageSummary {
  return {
    registered: 0,
    imported: 0,
    verified: 0,
    available: 0,
    disabled: 0,
    archived: 0,
    loaded: 0,
    queueDepth: 0,
    running: 0,
    completed: 0,
  };
}
