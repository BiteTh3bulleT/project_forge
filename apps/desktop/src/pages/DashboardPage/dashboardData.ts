import type { DashboardSummary, MemoryObservation } from "@forge/shared";

import type { ModelRuntimeUsageSummary } from "../../lib/api";
import { formatTime } from "../../lib/format";

export function activityRows(
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
      route: `/memory/chunk/${item.id}`,
    })),
  ].sort((a, b) => b.createdAtMs - a.createdAtMs);
}

export function buildActiveGoals(
  activeJobs: DashboardSummary["activeJobs"],
  recommendations: DashboardSummary["routingRecommendations"],
) {
  const goals = [
    ...activeJobs.slice(0, 4).map((job) => ({
      key: `job-${job.id}`,
      title: job.title || `Job ${job.id}`,
      detail: `${job.targetAdapter || "system"} | ${formatTime(job.createdAtMs)}`,
      status: job.status || "active",
      tone: job.status || "active",
      route: `/jobs/${job.id}`,
    })),
    ...recommendations.slice(0, Math.max(0, 4 - activeJobs.length)).map(
      (item) => ({
        key: `rec-${item.id}`,
        title: item.taskType || "Routing recommendation",
        detail: `${item.adapter} | ${Math.round(item.confidence * 100)}% confidence`,
        status: "route",
        tone: item.confidence >= 0.75 ? "ok" : "warn",
        route: "/strategies",
      }),
    ),
  ];

  if (goals.length > 0) return goals;

  return [
    {
      key: "idle-console",
      title: "Console idle",
      detail: "No active job or routing goal is visible.",
      status: "clear",
      tone: "ok",
      route: "/chat",
    },
  ];
}

export function getNextAction(
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

export function statusPillClass(status: string) {
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

export function statusDotClass(status: string) {
  const className = statusPillClass(status);
  if (className.includes("--ok"))
    return "bg-forge-electric shadow-[0_0_14px_rgba(85,214,255,0.45)]";
  if (className.includes("--bad"))
    return "bg-forge-emberSoft shadow-[0_0_14px_rgba(255,122,51,0.42)]";
  if (className.includes("--warn"))
    return "bg-forge-ember shadow-[0_0_14px_rgba(255,154,61,0.38)]";
  return "bg-forge-platinum/35";
}

export function metricAccentClass(status: string) {
  const className = statusPillClass(status);
  if (className.includes("--ok")) return "bg-forge-electric/80";
  if (className.includes("--bad")) return "bg-forge-emberSoft/90";
  if (className.includes("--warn")) return "bg-forge-ember/90";
  return "bg-forge-platinum/20";
}

export function flattenSystemStatus(
  raw: Record<string, unknown> | null | undefined,
): Array<[string, string]> {
  if (!raw || typeof raw !== "object") return [];
  return Object.entries(raw).map(([key, value]) => [
    humanizeKey(key),
    summarizeValue(value),
  ]);
}

export function shortPathLabel(path: string) {
  const parts = path.split(/[\\/]/).filter(Boolean);
  if (parts.length <= 2) return path || "-";
  return `.../${parts.slice(-2).join("/")}`;
}

export function countBy<T>(rows: T[], fn: (row: T) => string) {
  return rows.reduce<Record<string, number>>((acc, row) => {
    const key = fn(row);
    acc[key] = (acc[key] ?? 0) + 1;
    return acc;
  }, {});
}

export function sumHealth(rows: Array<{ value: number }>) {
  return rows.reduce((sum, item) => sum + item.value, 0);
}

export function emptyUsage(): ModelRuntimeUsageSummary {
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
