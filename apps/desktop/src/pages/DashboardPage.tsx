import type { DashboardSummary, MemoryObservation } from "@forge/shared";
import { GhostButton, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  api,
  type ModelRuntimeHealth,
  type ModelRuntimeQueueStatus,
  type ModelRuntimeUsageSummary,
} from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

type CapabilityRecord = Awaited<
  ReturnType<typeof api.gateway.capabilities>
>["capabilities"][number];
type InvocationRecord = Awaited<
  ReturnType<typeof api.gateway.invocations>
>["invocations"][number];
type RunRow = {
  id: string;
  title: string;
  status: string;
  targetAdapter: string;
  createdAtMs: number;
  kind: "active" | "failed";
};

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
  const [shadowModeEnabled, setShadowModeEnabled] = useState(false);
  const [shadowModeSaving, setShadowModeSaving] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [lastUpdatedAt, setLastUpdatedAt] = useState(Date.now());

  async function load() {
    setLoading(true);
    try {
      const [dash, coreHealth, gatewayCaps, gatewayInvs, memoryObs, settings] =
        await Promise.all([
          api.dashboard.summary(),
          api.health().catch(() => null),
          api.gateway
            .capabilities()
            .catch(() => ({ capabilities: [] as CapabilityRecord[] })),
          api.gateway
            .invocations({ limit: 80 })
            .catch(() => ({ invocations: [] as InvocationRecord[] })),
          api.memory
            .listObservations({ limit: 48 })
            .catch(() => ({ observations: [] as MemoryObservation[] })),
          api.settings.get().catch(() => null),
        ]);
      const runtimeAvailable = coreHealth?.modelRuntime?.available === true;
      const [healthRes, queueRes, usageRes] = runtimeAvailable
        ? await Promise.all([
            api.modelRuntime
              .health()
              .catch(() => ({ health: null as ModelRuntimeHealth | null })),
            api.modelRuntime
              .queue()
              .catch(() => ({ queue: null as ModelRuntimeQueueStatus | null })),
            api.modelRuntime.usage().catch(() => ({
              usage: null as ModelRuntimeUsageSummary | null,
            })),
          ])
        : [
            {
              health: coreHealth
                ? ({
                    ok: false,
                    status: coreHealth.modelRuntime?.status || "not enabled",
                    backend: "not configured",
                  } satisfies ModelRuntimeHealth)
                : null,
            },
            {
              queue: {
                depth: 0,
                scheduler: "not enabled",
              } satisfies ModelRuntimeQueueStatus,
            },
            { usage: emptyUsage() },
          ];

      setSummary(dash);
      setHealth(healthRes.health);
      setQueue(queueRes.queue);
      setUsage(usageRes.usage);
      setCapabilities(
        Array.isArray(gatewayCaps.capabilities) ? gatewayCaps.capabilities : [],
      );
      setInvocations(
        Array.isArray(gatewayInvs.invocations) ? gatewayInvs.invocations : [],
      );
      setObservations(
        Array.isArray(memoryObs.observations) ? memoryObs.observations : [],
      );
      setShadowModeEnabled(Boolean(settings?.shadowMode?.enabled));
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

  async function toggleShadowMode() {
    const next = !shadowModeEnabled;
    setShadowModeSaving(true);
    try {
      const updated = await api.settings.patch({
        shadowMode: { enabled: next },
      });
      setShadowModeEnabled(Boolean(updated.shadowMode?.enabled));
      setStatus(
        `Shadow mode ${updated.shadowMode?.enabled ? "enabled" : "disabled"}.`,
      );
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setShadowModeSaving(false);
    }
  }

  const activeJobs = Array.isArray(summary?.activeJobs)
    ? summary.activeJobs
    : [];
  const recentFailures = Array.isArray(summary?.recentFailures)
    ? summary.recentFailures
    : [];
  const recentImports = Array.isArray(summary?.recentImports)
    ? summary.recentImports
    : [];
  const automation = Array.isArray(summary?.automationActivity)
    ? summary.automationActivity
    : [];
  const recommendations = Array.isArray(summary?.routingRecommendations)
    ? summary.routingRecommendations
    : [];
  const systemStatusRows = useMemo(
    () => flattenSystemStatus(summary?.systemStatus),
    [summary?.systemStatus],
  );
  const capabilityCounts = useMemo(
    () => countBy(capabilities, (row) => row.status || "unknown"),
    [capabilities],
  );
  const invocationCounts = useMemo(
    () => countBy(invocations, (row) => row.status || "unknown"),
    [invocations],
  );
  const failedInvocations =
    invocationCounts.failed ?? invocationCounts.error ?? 0;
  const successInvocations =
    invocationCounts.completed ??
    invocationCounts.success ??
    invocationCounts.ok ??
    0;
  const attentionCount =
    (summary?.approvalsPending ?? 0) +
    (summary?.reviewsPending ?? 0) +
    recentFailures.length +
    failedInvocations;
  const runtimeDegraded = Boolean(
    health &&
    (!health.ok ||
      String(health.status || "")
        .toLowerCase()
        .includes("degraded")),
  );
  const runRows: RunRow[] = [
    ...activeJobs.map((job) => ({ ...job, kind: "active" as const })),
    ...recentFailures.map((job) => ({ ...job, kind: "failed" as const })),
  ].slice(0, 8);
  const healthSegments = [
    {
      label: "Healthy",
      value: Math.max(
        successInvocations,
        capabilities.length - attentionCount,
        0,
      ),
      tone: "ok",
    },
    {
      label: "Warning",
      value: Math.max((queue?.depth ?? 0) + (summary?.reviewsPending ?? 0), 0),
      tone: "warn",
    },
    {
      label: "Failed",
      value: Math.max(recentFailures.length + failedInvocations, 0),
      tone: "bad",
    },
  ];
  const nextAction = getNextAction(
    summary,
    activeJobs,
    recentFailures,
    failedInvocations,
  );
  const summarySignals = summary
    ? [
        `${activeJobs.length} active`,
        `${summary.approvalsPending ?? 0} approvals`,
        `${summary.reviewsPending ?? 0} reviews`,
        `${recentFailures.length + failedInvocations} failures`,
      ]
    : ["Loading current operational state"];

  return (
    <div className="forge-ops-board space-y-5">
      <header className="forge-ops-panel p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div className="min-w-0">
            <div className="forge-ops-label">
              {uiMode === "metrics"
                ? "Metrics Overview"
                : "Operations Overview"}
            </div>
            <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
              FORGE dashboard
            </h1>
            <div className="mt-3 flex max-w-full flex-wrap gap-2">
              {summarySignals.map((item) => (
                <span
                  key={item}
                  className="rounded border border-forge-platinum/10 bg-black/25 px-2 py-1 text-[11px] font-medium text-forge-mist/75"
                >
                  {item}
                </span>
              ))}
            </div>
          </div>
          <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center sm:justify-end">
            <span
              className={[
                statusPillClass(attentionCount > 0 ? "warn" : "ok"),
                "w-full sm:w-auto",
              ].join(" ")}
            >
              {attentionCount > 0 ? `${attentionCount} attention` : "Clear"}
            </span>
            <button
              type="button"
              aria-pressed={shadowModeEnabled}
              aria-label={
                shadowModeEnabled ? "Disable shadow mode" : "Enable shadow mode"
              }
              className={[
                "flex w-full items-center justify-between gap-3 rounded border px-3 py-2 text-left text-xs font-semibold transition sm:w-auto sm:min-w-[11.5rem]",
                shadowModeEnabled
                  ? "border-forge-mint/35 bg-forge-mint/10 text-forge-ash"
                  : "border-forge-platinum/10 bg-black/25 text-forge-mist/75 hover:border-forge-ember/35 hover:text-forge-ash",
              ].join(" ")}
              onClick={() => void toggleShadowMode()}
              disabled={shadowModeSaving}
            >
              <span className="min-w-0">
                <span className="block text-[10px] uppercase tracking-wide text-forge-mist/60">
                  Shadow mode
                </span>
                <span className="block">
                  {shadowModeSaving
                    ? "Updating"
                    : shadowModeEnabled
                      ? "On"
                      : "Off"}
                </span>
              </span>
              <span
                className={[
                  "relative h-5 w-9 shrink-0 rounded-full border transition",
                  shadowModeEnabled
                    ? "border-forge-mint/50 bg-forge-mint/30"
                    : "border-forge-platinum/15 bg-black/40",
                ].join(" ")}
              >
                <span
                  className={[
                    "absolute top-1/2 h-3.5 w-3.5 -translate-y-1/2 rounded-full transition",
                    shadowModeEnabled
                      ? "left-[1.15rem] bg-forge-mint"
                      : "left-1 bg-forge-mist/70",
                  ].join(" ")}
                />
              </span>
            </button>
            <GhostButton
              className="w-full sm:w-auto"
              onClick={() => void load()}
              disabled={loading}
            >
              {loading ? "Refreshing" : "Refresh"}
            </GhostButton>
            <PrimaryButton
              className="w-full sm:w-auto"
              onClick={() => navigate(nextAction.route)}
            >
              {nextAction.label}
            </PrimaryButton>
          </div>
        </div>
        <div className="mt-4 grid min-w-0 gap-2 border-t border-forge-platinum/10 pt-3 text-[11px] text-forge-mist/65 sm:grid-cols-3">
          <DashboardSignal
            label="Queue depth"
            value={String(queue?.depth ?? 0)}
          />
          <DashboardSignal
            label="Gateway calls"
            value={String(invocations.length)}
          />
          <DashboardSignal
            label="Updated"
            value={formatTime(lastUpdatedAt)}
            alignRight
          />
        </div>
      </header>

      {err ? (
        <div className="forge-ops-panel border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
          {err}
        </div>
      ) : null}

      <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          label="Active Runs"
          value={String(activeJobs.length)}
          detail={`${queue?.depth ?? 0} queued`}
          tone={activeJobs.length > 0 ? "warn" : "ok"}
        />
        <MetricCard
          label="Success Signals"
          value={String(successInvocations)}
          detail={`${invocations.length} gateway calls`}
          tone="ok"
          spark
        />
        <MetricCard
          label="Failed Runs"
          value={String(recentFailures.length + failedInvocations)}
          detail={`${recentFailures.length} job failures`}
          tone={recentFailures.length + failedInvocations > 0 ? "bad" : "ok"}
          sparkBad
        />
        <MetricCard
          label="Runtime"
          value={health?.status || "unknown"}
          detail={health?.backend || "not configured"}
          tone={runtimeDegraded ? "warn" : "ok"}
        />
      </section>

      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.45fr)_minmax(22rem,0.75fr)]">
        <div className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Recent Runs</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Execution state from jobs and gateway outcomes.
              </div>
            </div>
            <button
              type="button"
              className="forge-inline-link text-xs"
              onClick={() => navigate("/jobs")}
            >
              View jobs
            </button>
          </div>
          <div className="overflow-x-auto">
            <table className="forge-ops-table">
              <thead>
                <tr>
                  <th>Run</th>
                  <th>Status</th>
                  <th>Triggered by</th>
                  <th>Started</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {runRows.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="text-forge-mist/70">
                      No active or failed runs.
                    </td>
                  </tr>
                ) : null}
                {runRows.map((row) => (
                  <tr key={`${row.kind}-${row.id}`}>
                    <td>
                      <div className="flex min-w-0 items-start gap-2">
                        <span
                          className={[
                            "mt-1.5 h-2 w-2 shrink-0 rounded-full",
                            statusDotClass(
                              row.kind === "failed" ? "bad" : row.status,
                            ),
                          ].join(" ")}
                        />
                        <div className="min-w-0">
                          <button
                            type="button"
                            className="max-w-sm truncate text-left font-semibold text-forge-ash hover:text-forge-electric"
                            onClick={() => navigate(`/jobs/${row.id}`)}
                          >
                            {row.title || row.id}
                          </button>
                          <div className="mt-0.5 truncate font-mono text-[11px] text-forge-mist/55">
                            #{row.id}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td>
                      <span
                        className={statusPillClass(
                          row.kind === "failed" ? "bad" : row.status,
                        )}
                      >
                        {row.kind === "failed" ? "Failed" : row.status}
                      </span>
                    </td>
                    <td>{row.targetAdapter || "system"}</td>
                    <td>{formatTime(row.createdAtMs)}</td>
                    <td className="text-right">
                      <button
                        type="button"
                        className="text-forge-mist/65 hover:text-forge-ash"
                        onClick={() => navigate(`/jobs/${row.id}`)}
                      >
                        Open
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className="grid gap-4">
          <div className="forge-ops-panel">
            <div className="forge-ops-panel__head">
              <div>
                <div className="forge-ops-title">System Health</div>
                <div className="mt-1 text-xs text-forge-mist/65">
                  Queue, runtime, gateway, and review pressure.
                </div>
              </div>
            </div>
            <div className="forge-ops-panel__body grid gap-4 sm:grid-cols-[8.5rem_minmax(0,1fr)] xl:grid-cols-1">
              <HealthDonut segments={healthSegments} />
              <div className="space-y-3">
                {healthSegments.map((item) => (
                  <HealthRow
                    key={item.label}
                    label={item.label}
                    value={item.value}
                    tone={item.tone}
                    total={sumHealth(healthSegments)}
                  />
                ))}
              </div>
            </div>
          </div>

          <div className="forge-ops-panel">
            <div className="forge-ops-panel__head">
              <div>
                <div className="forge-ops-title">Next Action</div>
                <div className="mt-1 text-xs text-forge-mist/65">
                  {nextAction.reason}
                </div>
              </div>
            </div>
            <div className="forge-ops-panel__body">
              <PrimaryButton
                className="w-full py-3"
                onClick={() => navigate(nextAction.route)}
              >
                {nextAction.label}
              </PrimaryButton>
            </div>
          </div>
        </div>
      </section>

      <section className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <div className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Gateway Activity</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Capability registry and invocation outcomes.
              </div>
            </div>
            <button
              type="button"
              className="forge-inline-link text-xs"
              onClick={() => navigate("/gateway")}
            >
              Gateway
            </button>
          </div>
          <div className="forge-ops-panel__body grid gap-4 md:grid-cols-2">
            <Distribution title="Capabilities" counts={capabilityCounts} />
            <Distribution title="Invocations" counts={invocationCounts} />
          </div>
        </div>

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
            {[...activityRows(recentImports, automation, observations)]
              .slice(0, 6)
              .map((item) => (
                <button
                  key={item.key}
                  type="button"
                  onClick={() => navigate(item.route)}
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
      </section>

      <section className="grid gap-4 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
        <div className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Runtime Queue</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Modelruntime state for local inference paths.
              </div>
            </div>
            <button
              type="button"
              className="forge-inline-link text-xs"
              onClick={() => navigate("/models")}
            >
              Models
            </button>
          </div>
          <div className="forge-ops-panel__body">
            <SectionRows
              rows={[
                ["Health", health?.status || (health?.ok ? "ok" : "unknown")],
                ["Backend", health?.backend || "unknown"],
                ["Scheduler", queue?.scheduler || "unknown"],
                ["Queue depth", String(queue?.depth ?? 0)],
                ["Registered", String(usage?.registered ?? 0)],
                ["Loaded", String(usage?.loaded ?? 0)],
              ]}
            />
          </div>
        </div>

        <div className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">System Fields</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Core status fields exposed by diagnostics.
              </div>
            </div>
            <button
              type="button"
              className="forge-inline-link text-xs"
              onClick={() => navigate("/inspectors")}
            >
              Inspect
            </button>
          </div>
          <div className="forge-ops-panel__body">
            <SectionRows
              rows={
                systemStatusRows.length > 0
                  ? systemStatusRows.slice(0, 8)
                  : [["Status", "No system status fields available"]]
              }
            />
          </div>
        </div>
      </section>

      {recommendations.length > 0 ? (
        <section className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Routing Recommendations</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Latest strategy hints from the core.
              </div>
            </div>
          </div>
          <div className="forge-ops-panel__body grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {recommendations.slice(0, 6).map((item) => (
              <button
                key={item.id}
                type="button"
                onClick={() => navigate("/strategies")}
                className="forge-ops-card p-3 text-left"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="truncate text-sm font-semibold text-forge-ash">
                    {item.taskType}
                  </span>
                  <span className="text-xs text-forge-electric">
                    {Math.round(item.confidence * 100)}%
                  </span>
                </div>
                <div className="mt-2 text-xs text-forge-mist/70">
                  {item.adapter}
                </div>
              </button>
            ))}
          </div>
        </section>
      ) : null}

      <div className="text-right text-[11px] text-forge-mist/50">
        Updated {formatTime(lastUpdatedAt)}
        <button
          type="button"
          className="ml-3 text-forge-electric hover:text-forge-ash"
          onClick={() =>
            setStatus(`Dashboard refreshed at ${formatTime(lastUpdatedAt)}`)
          }
        >
          Record status
        </button>
      </div>
    </div>
  );
}

function MetricCard(props: {
  label: string;
  value: string;
  detail: string;
  tone: string;
  spark?: boolean;
  sparkBad?: boolean;
}) {
  return (
    <div className="forge-ops-card min-h-[8.25rem] p-4">
      <span
        className={[
          "absolute inset-x-0 top-0 h-0.5",
          metricAccentClass(props.tone),
        ].join(" ")}
      />
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="forge-ops-label">{props.label}</div>
          <div className="mt-2 truncate text-3xl font-semibold tracking-normal text-forge-ash">
            {props.value}
          </div>
        </div>
        {props.spark || props.sparkBad ? (
          <div
            className={
              props.sparkBad
                ? "forge-ops-sparkline forge-ops-sparkline--bad"
                : "forge-ops-sparkline"
            }
          />
        ) : (
          <span className={statusPillClass(props.tone)}>{props.tone}</span>
        )}
      </div>
      <div className="mt-3 flex items-center gap-2 text-xs text-forge-mist/65">
        <span
          className={[
            "h-1.5 w-1.5 shrink-0 rounded-full",
            statusDotClass(props.tone),
          ].join(" ")}
        />
        <span className="min-w-0 truncate">{props.detail}</span>
      </div>
    </div>
  );
}

function DashboardSignal(props: {
  label: string;
  value: string;
  alignRight?: boolean;
}) {
  return (
    <div
      className={[
        "flex min-w-0 flex-wrap items-center justify-between gap-2 rounded border border-forge-platinum/10 bg-black/20 px-3 py-2",
        props.alignRight ? "sm:text-right" : "",
      ].join(" ")}
    >
      <span className="forge-ops-label shrink-0">{props.label}</span>
      <span className="w-full min-w-0 break-all font-mono text-forge-ash sm:w-auto sm:text-right">
        {props.value}
      </span>
    </div>
  );
}

function HealthDonut(props: {
  segments: Array<{ value: number; tone: string }>;
}) {
  const total = sumHealth(props.segments);
  const basis = Math.max(total, 1);
  const ok =
    ((props.segments.find((item) => item.tone === "ok")?.value ?? 0) / basis) *
    100;
  const warn =
    ((props.segments.find((item) => item.tone === "warn")?.value ?? 0) /
      basis) *
    100;
  const style = {
    background:
      total > 0
        ? `conic-gradient(rgb(var(--forge-mint-rgb)) 0 ${ok}%, rgb(var(--forge-amber-rgb)) ${ok}% ${ok + warn}%, rgb(var(--forge-danger-rgb)) ${ok + warn}% 100%)`
        : "conic-gradient(rgba(255,255,255,0.12) 0 100%)",
  };
  return (
    <div
      className="mx-auto grid h-32 w-32 place-items-center rounded-full"
      style={style}
    >
      <div className="grid h-20 w-20 place-items-center rounded-full bg-[#0b0e12] text-center">
        <div>
          <div className="text-xl font-semibold text-forge-ash">{total}</div>
          <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/55">
            signals
          </div>
        </div>
      </div>
    </div>
  );
}

function HealthRow(props: {
  label: string;
  value: number;
  tone: string;
  total: number;
}) {
  const pct =
    props.total <= 0 ? 0 : Math.round((props.value / props.total) * 100);
  return (
    <div>
      <div className="mb-1 flex items-center justify-between gap-3 text-xs">
        <span className="text-forge-mist/70">{props.label}</span>
        <span className="font-semibold text-forge-ash">
          {props.value} ({pct}%)
        </span>
      </div>
      <div className="forge-ops-progress">
        <span style={{ width: `${Math.max(4, pct)}%` }} />
      </div>
    </div>
  );
}

function Distribution(props: {
  title: string;
  counts: Record<string, number>;
}) {
  const entries = Object.entries(props.counts).sort((a, b) => b[1] - a[1]);
  const total = entries.reduce((sum, item) => sum + item[1], 0);
  return (
    <div>
      <div className="forge-ops-label">{props.title}</div>
      <div className="mt-3 space-y-3">
        {entries.length === 0 ? (
          <div className="text-xs text-forge-mist/65">No data.</div>
        ) : null}
        {entries.slice(0, 6).map(([label, value]) => {
          const pct = total <= 0 ? 0 : Math.round((value / total) * 100);
          return (
            <div key={label}>
              <div className="mb-1 flex justify-between text-xs">
                <span className="text-forge-mist/70">{label}</span>
                <span className="font-semibold text-forge-ash">{value}</span>
              </div>
              <div className="forge-ops-progress">
                <span style={{ width: `${Math.max(4, pct)}%` }} />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function SectionRows(props: { rows: Array<[string, string]> }) {
  return (
    <div className="grid gap-2 md:grid-cols-2">
      {props.rows.map(([label, value]) => (
        <div
          key={label}
          className="flex min-h-10 items-center justify-between gap-3 rounded-md border border-forge-platinum/10 bg-black/20 px-3 py-2 text-xs"
        >
          <span className="text-forge-mist/65">{label}</span>
          <span className="min-w-0 truncate text-right font-semibold text-forge-ash">
            {value}
          </span>
        </div>
      ))}
    </div>
  );
}

function activityRows(
  imports: DashboardSummary["recentImports"],
  automation: DashboardSummary["automationActivity"],
  observations: MemoryObservation[],
) {
  return [
    ...imports.map((item) => ({
      key: `import-${item.id}`,
      title: item.summary || item.adapterId,
      detail: `Import via ${item.adapterId}`,
      createdAtMs: item.createdAtMs,
      route: "/workbench",
    })),
    ...automation.map((item) => ({
      key: `automation-${item.id}`,
      title: item.message || `Automation ${item.status}`,
      detail: `Rule ${item.ruleId ?? "manual"} | ${item.status}`,
      createdAtMs: item.createdAtMs,
      route: "/automation",
    })),
    ...observations.map((item) => ({
      key: `obs-${item.id}`,
      title: item.summary || item.type || "Memory observation",
      detail: item.sourcePath || item.originKind || "Observation",
      createdAtMs: item.observedAtMs || item.createdAtMs,
      route: `/memory/${item.id}`,
    })),
  ].sort((a, b) => b.createdAtMs - a.createdAtMs);
}

function getNextAction(
  summary: DashboardSummary | null,
  activeJobs: DashboardSummary["activeJobs"],
  recentFailures: DashboardSummary["recentFailures"],
  failedInvocations: number,
) {
  if ((summary?.approvalsPending ?? 0) > 0)
    return {
      label: "Open Approvals",
      route: "/approvals",
      reason: "Execution is waiting on explicit gates.",
    };
  if ((summary?.reviewsPending ?? 0) > 0)
    return {
      label: "Open Reviews",
      route: "/reviews",
      reason: "Operator review is the next commit boundary.",
    };
  if (recentFailures.length > 0 || failedInvocations > 0)
    return {
      label: "Inspect Failures",
      route: "/events",
      reason: "Failure evidence should be inspected before new work starts.",
    };
  if (activeJobs.length > 0)
    return {
      label: "Track Runs",
      route: "/jobs",
      reason: "Active jobs are still moving through the run pipeline.",
    };
  return {
    label: "New Task",
    route: "/chat",
    reason: "No blocking operational work is visible.",
  };
}

function statusPillClass(status: string) {
  const normalized = String(status || "").toLowerCase();
  if (
    [
      "ok",
      "success",
      "completed",
      "ready",
      "online",
      "verified",
      "enabled",
      "clear",
    ].some((item) => normalized.includes(item))
  )
    return "forge-ops-status forge-ops-status--ok";
  if (
    ["fail", "error", "blocked", "denied", "bad"].some((item) =>
      normalized.includes(item),
    )
  )
    return "forge-ops-status forge-ops-status--bad";
  if (
    ["warn", "pending", "running", "queued", "degraded", "active"].some(
      (item) => normalized.includes(item),
    )
  )
    return "forge-ops-status forge-ops-status--warn";
  return "forge-ops-status forge-ops-status--muted";
}

function statusDotClass(status: string) {
  const className = statusPillClass(status);
  if (className.includes("--ok"))
    return "bg-forge-electric shadow-[0_0_14px_rgba(85,214,255,0.45)]";
  if (className.includes("--bad"))
    return "bg-forge-emberSoft shadow-[0_0_14px_rgba(255,122,51,0.42)]";
  if (className.includes("--warn"))
    return "bg-forge-ember shadow-[0_0_14px_rgba(255,154,61,0.38)]";
  return "bg-forge-platinum/35";
}

function metricAccentClass(status: string) {
  const className = statusPillClass(status);
  if (className.includes("--ok")) return "bg-forge-electric/80";
  if (className.includes("--bad")) return "bg-forge-emberSoft/90";
  if (className.includes("--warn")) return "bg-forge-ember/90";
  return "bg-forge-platinum/20";
}

function flattenSystemStatus(
  raw: Record<string, unknown> | null | undefined,
): Array<[string, string]> {
  if (!raw || typeof raw !== "object") return [];
  return Object.entries(raw).map(([key, value]) => [
    humanizeKey(key),
    summarizeValue(value),
  ]);
}

function summarizeValue(value: unknown) {
  if (value == null) return "-";
  if (
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  )
    return String(value);
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

function sumHealth(rows: Array<{ value: number }>) {
  return rows.reduce((sum, item) => sum + item.value, 0);
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
