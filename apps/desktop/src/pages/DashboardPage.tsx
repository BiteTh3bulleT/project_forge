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
import { useWorkspaceStore } from "../stores/workspaceStore";
import { ActivityFeedSection } from "./DashboardPage/ActivityFeedSection";
import { DashboardSignal } from "./DashboardPage/DashboardSignal";
import { Distribution } from "./DashboardPage/Distribution";
import { HealthDonut } from "./DashboardPage/HealthDonut";
import { HealthRow } from "./DashboardPage/HealthRow";
import { MetricCard } from "./DashboardPage/MetricCard";
import { SectionRows } from "./DashboardPage/SectionRows";
import {
  buildActiveGoals,
  countBy,
  emptyUsage,
  flattenSystemStatus,
  getNextAction,
  shortPathLabel,
  statusDotClass,
  statusPillClass,
  sumHealth,
} from "./DashboardPage/dashboardData";
import type {
  CapabilityRecord,
  InvocationRecord,
  RunRow,
} from "./DashboardPage/types";

export function DashboardPage() {
  const navigate = useNavigate();
  const uiMode = useUiStore((s) => s.uiMode);
  const setStatus = useUiStore((s) => s.setStatusLine);
  const coreState = useWorkspaceStore((s) => s.core);
  const workspaceMeta = useWorkspaceStore((s) => s.meta);

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
              queue: null as ModelRuntimeQueueStatus | null,
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
  const dossierHealth = Array.isArray(summary?.dossierHealth)
    ? summary.dossierHealth
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
  const queueDepth = queue?.depth;
  const queueDepthLabel =
    typeof queueDepth === "number" ? String(queueDepth) : "unavailable";
  const queueDetailLabel =
    typeof queueDepth === "number" ? `${queueDepth} queued` : "queue unavailable";
  const queueSchedulerLabel = queue?.scheduler || "unavailable";
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
  const staleObservations = observations.filter((item) => item.stale).length;
  const verifiedObservations = observations.filter((item) =>
    String(item.verificationState || "")
      .toLowerCase()
      .includes("verified"),
  ).length;
  const workspaceLabel = workspaceMeta?.workspaceDir
    ? shortPathLabel(workspaceMeta.workspaceDir)
    : "Workspace unknown";
  const workspaceRows: Array<[string, string]> = [
    ["Core", coreState],
    ["Workspace", workspaceLabel],
    [
      "Data",
      workspaceMeta?.dataDir ? shortPathLabel(workspaceMeta.dataDir) : "-",
    ],
    [
      "Database",
      workspaceMeta?.dbPath ? shortPathLabel(workspaceMeta.dbPath) : "-",
    ],
  ];
  const activeGoalRows = buildActiveGoals(activeJobs, recommendations);
  const decisionRows = [
    {
      label: "Approvals",
      value: summary?.approvalsPending ?? 0,
      route: "/approvals",
      tone: (summary?.approvalsPending ?? 0) > 0 ? "warn" : "ok",
    },
    {
      label: "Reviews",
      value: summary?.reviewsPending ?? 0,
      route: "/reviews",
      tone: (summary?.reviewsPending ?? 0) > 0 ? "warn" : "ok",
    },
    {
      label: "Failures",
      value: recentFailures.length + failedInvocations,
      route: "/events",
      tone: recentFailures.length + failedInvocations > 0 ? "bad" : "ok",
    },
  ];
  const openLoopRows = [
    ["Active jobs", String(activeJobs.length)],
    ["Runtime queue", queueDepthLabel],
    ["Gateway failures", String(failedInvocations)],
    ["Stale memory", String(staleObservations)],
  ] satisfies Array<[string, string]>;
  const stateRows = [
    ["Mode", uiMode],
    ["Attention", attentionCount > 0 ? `${attentionCount} item(s)` : "clear"],
    ["Runtime", health?.status || "unknown"],
    ["Backend", health?.backend || "not configured"],
    ["Capabilities", String(capabilities.length)],
    ["Observations", String(observations.length)],
  ] satisfies Array<[string, string]>;
  const contextCards = [
    {
      title: "Memory graph",
      value: String(observations.length),
      detail: `${verifiedObservations} verified / ${staleObservations} stale`,
      route: "/memory",
      tone: staleObservations > 0 ? "warn" : "ok",
    },
    {
      title: "Context",
      value: String(dossierHealth.length),
      detail:
        dossierHealth.length > 0
          ? "dossier health records visible"
          : "no dossier health rows",
      route: "/dossiers",
      tone: dossierHealth.length > 0 ? "ok" : "muted",
    },
    {
      title: "Workspaces",
      value: workspaceLabel,
      detail: workspaceMeta?.workspaceDir || "core metadata unavailable",
      route: "/layouts",
      tone: coreState === "online" ? "ok" : "warn",
    },
    {
      title: "Artifacts",
      value: String(recentImports.length + automation.length),
      detail: `${recentImports.length} imports / ${automation.length} automation`,
      route: "/workbench",
      tone: recentImports.length + automation.length > 0 ? "ok" : "muted",
    },
  ];
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
      value: Math.max((queueDepth ?? 0) + (summary?.reviewsPending ?? 0), 0),
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
                ? "Telemetry Dense View"
                : "Kernel Cockpit"}
            </div>
            <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
              Cognition Console
            </h1>
            <p className="mt-2 max-w-3xl text-sm text-forge-mist/70">
              Goals, workspace truth, decision gates, runtime pressure, and
              memory context.
            </p>
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
            value={queueDepthLabel}
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

      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(21rem,0.8fr)_minmax(22rem,0.85fr)]">
        <div className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Active Goals</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Current objectives assembled from running jobs and route hints.
              </div>
            </div>
            <button
              type="button"
              className="forge-inline-link text-xs"
              onClick={() => navigate("/jobs")}
            >
              Jobs
            </button>
          </div>
          <div className="forge-ops-panel__body space-y-2">
            {activeGoalRows.map((goal) => (
              <button
                key={goal.key}
                type="button"
                onClick={() => navigate(goal.route)}
                className="forge-ops-card flex w-full items-start justify-between gap-3 p-3 text-left"
              >
                <span className="min-w-0">
                  <span className="block truncate text-sm font-semibold text-forge-ash">
                    {goal.title}
                  </span>
                  <span className="mt-1 block truncate text-xs text-forge-mist/65">
                    {goal.detail}
                  </span>
                </span>
                <span className={statusPillClass(goal.tone)}>{goal.status}</span>
              </button>
            ))}
          </div>
        </div>

        <div className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Workspace</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Current local boundary and daemon connection state.
              </div>
            </div>
            <button
              type="button"
              className="forge-inline-link text-xs"
              onClick={() => navigate("/settings")}
            >
              Settings
            </button>
          </div>
          <div className="forge-ops-panel__body">
            <SectionRows rows={workspaceRows} />
          </div>
        </div>

        <div className="grid gap-4">
          <div className="forge-ops-panel">
            <div className="forge-ops-panel__head">
              <div>
                <div className="forge-ops-title">Decisions</div>
                <div className="mt-1 text-xs text-forge-mist/65">
                  Gates and review pressure before state changes.
                </div>
              </div>
            </div>
            <div className="forge-ops-panel__body grid gap-2 sm:grid-cols-3 xl:grid-cols-1">
              {decisionRows.map((row) => (
                <button
                  key={row.label}
                  type="button"
                  onClick={() => navigate(row.route)}
                  className="flex min-h-12 items-center justify-between gap-3 rounded-md border border-forge-platinum/10 bg-black/20 px-3 py-2 text-left text-xs hover:border-forge-ember/35"
                >
                  <span className="text-forge-mist/65">{row.label}</span>
                  <span className={statusPillClass(row.tone)}>
                    {row.value}
                  </span>
                </button>
              ))}
            </div>
          </div>

          <div className="forge-ops-panel">
            <div className="forge-ops-panel__head">
              <div>
                <div className="forge-ops-title">Open Loops</div>
                <div className="mt-1 text-xs text-forge-mist/65">
                  Outstanding work that still needs closure.
                </div>
              </div>
            </div>
            <div className="forge-ops-panel__body">
              <SectionRows rows={openLoopRows} />
            </div>
          </div>
        </div>
      </section>

      <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
        <MetricCard
          label="State"
          value={attentionCount > 0 ? "watch" : "clear"}
          detail={`${attentionCount} decision/open-loop signal(s)`}
          tone={attentionCount > 0 ? "warn" : "ok"}
        />
        <MetricCard
          label="Active Goals"
          value={String(activeJobs.length)}
          detail={queueDetailLabel}
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

        <ActivityFeedSection
          recentImports={recentImports}
          automation={automation}
          observations={observations}
          onNavigate={navigate}
        />
      </section>

      <section className="grid gap-4 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
        <div className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Runtime Monitor</div>
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
                ["Scheduler", queueSchedulerLabel],
                ["Queue depth", queueDepthLabel],
                ["Registered", String(usage?.registered ?? 0)],
                ["Loaded", String(usage?.loaded ?? 0)],
              ]}
            />
          </div>
        </div>

        <div className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">State Fields</div>
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
              rows={[
                ...stateRows,
                ...(systemStatusRows.length > 0
                  ? systemStatusRows.slice(0, 4)
                  : ([
                      ["Diagnostics", "No system status fields available"],
                    ] satisfies Array<[string, string]>)),
              ]}
            />
          </div>
        </div>
      </section>

      <section className="forge-ops-panel">
        <div className="forge-ops-panel__head">
          <div>
            <div className="forge-ops-title">Cognitive Surfaces</div>
            <div className="mt-1 text-xs text-forge-mist/65">
              Memory, context, workspace, and artifact state.
            </div>
          </div>
        </div>
        <div className="forge-ops-panel__body grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          {contextCards.map((card) => (
            <button
              key={card.title}
              type="button"
              onClick={() => navigate(card.route)}
              className="forge-ops-card min-h-[8.5rem] p-4 text-left"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="forge-ops-label">{card.title}</div>
                  <div className="mt-2 truncate text-xl font-semibold text-forge-ash">
                    {card.value}
                  </div>
                </div>
                <span className={statusPillClass(card.tone)}>{card.tone}</span>
              </div>
              <div className="mt-3 line-clamp-2 text-xs text-forge-mist/65">
                {card.detail}
              </div>
            </button>
          ))}
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
