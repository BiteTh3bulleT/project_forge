import type { JobRecord, JobStatus } from "@forge/shared";
import { GhostButton, PrimaryButton } from "@forge/ui";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

const statuses: Array<{ value: "" | JobStatus; label: string }> = [
  { value: "", label: "All" },
  { value: "queued", label: "Queued" },
  { value: "preparing", label: "Preparing" },
  { value: "awaiting_approval", label: "Awaiting approval" },
  { value: "running", label: "Running" },
  { value: "succeeded", label: "Succeeded" },
  { value: "failed", label: "Failed" },
  { value: "cancelled", label: "Cancelled" },
];

export function JobsPage() {
  const navigate = useNavigate();
  const [filter, setFilter] = useState<"" | JobStatus>("");
  const [jobs, setJobs] = useState<JobRecord[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const setStatus = useUiStore((s) => s.setStatusLine);

  const refresh = useCallback(async () => {
    try {
      const res = await api.jobs.list(filter, 180);
      const rows = res?.jobs;
      setJobs(Array.isArray(rows) ? rows : []);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setJobs([]);
    }
  }, [filter]);

  useEffect(() => {
    void refresh();
    const id = window.setInterval(() => void refresh(), 2500);
    return () => window.clearInterval(id);
  }, [refresh]);

  const counts = useMemo(() => {
    const out: Record<string, number> = {};
    for (const j of jobs) {
      const st = j.status ?? "unknown";
      out[st] = (out[st] ?? 0) + 1;
    }
    return out;
  }, [jobs]);

  const attentionCount = (counts.awaiting_approval ?? 0) + (counts.failed ?? 0);
  const activeCount =
    (counts.queued ?? 0) + (counts.preparing ?? 0) + (counts.running ?? 0);
  const latestJob = jobs[0] ?? null;
  const filterLabel = statuses.find((s) => s.value === filter)?.label ?? "All";

  return (
    <div className="forge-ops-board space-y-5">
      <header className="forge-ops-panel p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div className="min-w-0">
            <div className="forge-ops-label">Pipeline Runs</div>
            <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
              Jobs command board
            </h1>
            <p className="mt-2 max-w-3xl text-sm leading-6 text-forge-mist/75">
              {jobs.length > 0
                ? `${jobs.length} ${filter ? filterLabel.toLowerCase() : "recent"} runs loaded. Latest projection: ${latestJob?.title ?? latestJob?.id}.`
                : `No ${filter ? filterLabel.toLowerCase() : "recent"} runs loaded.`}
            </p>
          </div>
          <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center sm:justify-end">
            <span
              className={[
                statusPillClass(
                  attentionCount > 0 ? "awaiting_approval" : "succeeded",
                ),
                "w-full sm:w-auto",
              ].join(" ")}
            >
              {attentionCount > 0 ? `${attentionCount} attention` : "Clear"}
            </span>
            <GhostButton
              className="w-full sm:w-auto"
              onClick={() => void refresh()}
            >
              Refresh
            </GhostButton>
            <PrimaryButton
              className="w-full sm:w-auto"
              onClick={async () => {
                try {
                  const res = await api.jobs.create({
                    templateId: "search_packet",
                    title: "Memory packet",
                    userRequest:
                      "Build a fresh task packet from current memory",
                    objective: "Assemble contextual packet for follow-on work",
                    query: "project context",
                    initiatingSource: "jobs_page",
                  });
                  setStatus(`Job queued: ${res.job.id}`);
                  navigate(`/jobs/${res.job.id}`);
                } catch (e) {
                  setErr(e instanceof Error ? e.message : String(e));
                }
              }}
            >
              New Packet Job
            </PrimaryButton>
          </div>
        </div>
        <div className="mt-4 grid gap-2 border-t border-forge-platinum/10 pt-3 text-[11px] text-forge-mist/65 sm:grid-cols-3">
          <HeaderSignal label="Running" value={String(counts.running ?? 0)} />
          <HeaderSignal
            label="Awaiting"
            value={String(counts.awaiting_approval ?? 0)}
          />
          <HeaderSignal label="Filter" value={filterLabel} />
        </div>
      </header>

      {err ? (
        <div className="forge-ops-panel border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
          {err}
        </div>
      ) : null}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <RunMetric
          label="Loaded"
          value={String(jobs.length)}
          detail={filter ? `${filterLabel} filter` : "all recent runs"}
          tone="muted"
        />
        <RunMetric
          label="Active"
          value={String(activeCount)}
          detail={`${counts.running ?? 0} running`}
          tone={activeCount > 0 ? "warn" : "ok"}
        />
        <RunMetric
          label="Approval Gate"
          value={String(counts.awaiting_approval ?? 0)}
          detail="waiting on operator"
          tone={(counts.awaiting_approval ?? 0) > 0 ? "warn" : "ok"}
        />
        <RunMetric
          label="Failed"
          value={String(counts.failed ?? 0)}
          detail={`${counts.cancelled ?? 0} cancelled`}
          tone={(counts.failed ?? 0) > 0 ? "bad" : "ok"}
        />
      </section>

      <section className="forge-ops-panel">
        <div className="forge-ops-panel__head flex-col items-stretch sm:flex-row sm:items-center">
          <div>
            <div className="forge-ops-title">Run Filters</div>
            <div className="mt-1 text-xs text-forge-mist/65">
              Status-scoped job projections from the execution queue.
            </div>
          </div>
          <label className="sr-only" htmlFor="job-status-filter">
            Status filter
          </label>
          <select
            id="job-status-filter"
            aria-label="Job status filter"
            className="forge-input w-full sm:w-56"
            value={filter}
            onChange={(e) => setFilter(e.target.value as "" | JobStatus)}
          >
            {statuses.map((s) => (
              <option key={s.label} value={s.value}>
                {s.label}
              </option>
            ))}
          </select>
        </div>
        <div className="forge-ops-panel__body">
          <div className="flex gap-2 overflow-x-auto pb-1">
            {statuses.map((s) => {
              const selected = filter === s.value;
              const count = s.value ? (counts[s.value] ?? 0) : jobs.length;
              return (
                <button
                  key={s.label}
                  type="button"
                  className={[
                    "min-w-[7.5rem] shrink-0 rounded border px-3 py-2 text-left text-xs transition",
                    selected
                      ? "border-forge-ember/45 bg-forge-ember/15 text-forge-ash shadow-[inset_0_1px_0_rgba(255,255,255,0.06)]"
                      : "border-forge-platinum/10 bg-black/20 text-forge-mist/75 hover:border-forge-ember/30 hover:text-forge-ash",
                  ].join(" ")}
                  onClick={() => setFilter(s.value)}
                >
                  <span className="flex items-center justify-between gap-3">
                    <span className="font-semibold">{s.label}</span>
                    <span
                      className={[
                        "h-1.5 w-1.5 rounded-full",
                        statusDotClass(s.value || "muted"),
                      ].join(" ")}
                    />
                  </span>
                  <span className="mt-1.5 block font-mono text-[11px] text-forge-mist/55">
                    {count} run{count === 1 ? "" : "s"}
                  </span>
                </button>
              );
            })}
          </div>
        </div>
      </section>

      <section className="forge-ops-panel">
        <div className="forge-ops-panel__head">
          <div>
            <div className="forge-ops-title">Recent Runs</div>
            <div className="mt-1 text-xs text-forge-mist/65">
              Open a run to inspect packet, approval state, events, and
              artifacts.
            </div>
          </div>
          <span className="font-mono text-[11px] text-forge-mist/60">
            limit 180
          </span>
        </div>
        {jobs.length === 0 ? (
          <div className="forge-ops-panel__body text-sm text-forge-mist">
            No jobs yet.
          </div>
        ) : (
          <div>
            <div className="hidden overflow-x-auto md:block">
              <table className="forge-ops-table">
                <thead>
                  <tr>
                    <th>Run</th>
                    <th>Status</th>
                    <th>Boundary</th>
                    <th>Adapter</th>
                    <th>Updated</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {jobs.map((j) => (
                    <tr key={j.id} className="align-top">
                      <td>
                        <div className="flex min-w-0 items-start gap-2">
                          <span
                            className={[
                              "mt-1.5 h-2 w-2 shrink-0 rounded-full",
                              statusDotClass(j.status),
                            ].join(" ")}
                          />
                          <div className="min-w-0">
                            <button
                              type="button"
                              className="max-w-md truncate text-left font-semibold text-forge-ash hover:text-forge-electric"
                              onClick={() => navigate(`/jobs/${j.id}`)}
                            >
                              {j.title || j.id}
                            </button>
                            <div className="mt-1 flex flex-wrap gap-2 text-[11px] text-forge-mist/55">
                              <span className="font-mono">#{j.id}</span>
                              <span>{j.initiatingSource || "system"}</span>
                              {j.writeIntent ? (
                                <span className="text-forge-emberSoft">
                                  write intent
                                </span>
                              ) : null}
                            </div>
                            {j.lastFailureCode || j.lastError ? (
                              <div className="mt-1 text-[11px] text-forge-emberSoft">
                                failure: {j.lastFailureCode || j.lastError}
                              </div>
                            ) : null}
                          </div>
                        </div>
                      </td>
                      <td>
                        <span className={statusPillClass(j.status)}>
                          {statusLabel(j.status)}
                        </span>
                      </td>
                      <td>
                        <div className="text-forge-ash">
                          {j.executionBoundary || "bounded"}
                        </div>
                        <div className="mt-1 text-[11px] text-forge-mist/55">
                          {j.riskClass}
                        </div>
                      </td>
                      <td>{j.targetAdapter || "gateway"}</td>
                      <td>
                        <div>{formatTime(j.updatedAtMs || j.createdAtMs)}</div>
                        <div className="mt-1 text-[11px] text-forge-mist/55">
                          created {formatTime(j.createdAtMs)}
                        </div>
                      </td>
                      <td className="text-right">
                        <button
                          type="button"
                          className="text-xs font-semibold text-forge-emberSoft hover:text-forge-ash"
                          onClick={() => navigate(`/jobs/${j.id}`)}
                        >
                          Open
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="space-y-2 p-3 md:hidden">
              {jobs.map((j) => (
                <button
                  key={j.id}
                  type="button"
                  className="w-full rounded border border-forge-platinum/10 bg-black/25 p-3 text-left transition hover:border-forge-ember/35 hover:bg-black/35"
                  onClick={() => navigate(`/jobs/${j.id}`)}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex min-w-0 items-start gap-2">
                      <span
                        className={[
                          "mt-1.5 h-2 w-2 shrink-0 rounded-full",
                          statusDotClass(j.status),
                        ].join(" ")}
                      />
                      <div className="min-w-0">
                        <div className="truncate text-sm font-semibold text-forge-ash">
                          {j.title || j.id}
                        </div>
                        <div className="mt-1 truncate font-mono text-[11px] text-forge-mist/55">
                          #{j.id}
                        </div>
                      </div>
                    </div>
                    <span className={statusPillClass(j.status)}>
                      {statusLabel(j.status)}
                    </span>
                  </div>
                  <div className="mt-3 grid grid-cols-2 gap-2 text-[11px] text-forge-mist/70">
                    <RunDatum
                      label="adapter"
                      value={j.targetAdapter || "gateway"}
                    />
                    <RunDatum label="risk" value={j.riskClass} />
                    <RunDatum
                      label="boundary"
                      value={j.executionBoundary || "bounded"}
                    />
                    <RunDatum
                      label="updated"
                      value={formatTime(j.updatedAtMs || j.createdAtMs)}
                    />
                  </div>
                  {j.lastFailureCode || j.lastError ? (
                    <div className="mt-2 text-xs text-forge-emberSoft">
                      failure: {j.lastFailureCode || j.lastError}
                    </div>
                  ) : null}
                </button>
              ))}
            </div>
          </div>
        )}
      </section>
    </div>
  );
}

function RunMetric(props: {
  label: string;
  value: string;
  detail: string;
  tone: "ok" | "warn" | "bad" | "muted";
}) {
  return (
    <div className="forge-ops-card min-h-[8rem] p-4">
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
        <span className={statusPillClass(props.tone)}>{props.tone}</span>
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

function HeaderSignal(props: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded border border-forge-platinum/10 bg-black/20 px-3 py-2">
      <span className="forge-ops-label">{props.label}</span>
      <span className="min-w-0 truncate font-mono text-forge-ash">
        {props.value}
      </span>
    </div>
  );
}

function RunDatum(props: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded border border-forge-platinum/10 bg-black/20 px-2 py-1.5">
      <div className="forge-ops-label">{props.label}</div>
      <div className="mt-1 truncate text-forge-ash">{props.value}</div>
    </div>
  );
}

function statusPillClass(status: JobStatus | "ok" | "warn" | "bad" | "muted") {
  if (status === "succeeded" || status === "ok")
    return "forge-ops-status forge-ops-status--ok";
  if (status === "failed" || status === "bad")
    return "forge-ops-status forge-ops-status--bad";
  if (
    status === "queued" ||
    status === "preparing" ||
    status === "awaiting_approval" ||
    status === "running" ||
    status === "warn"
  ) {
    return "forge-ops-status forge-ops-status--warn";
  }
  return "forge-ops-status forge-ops-status--muted";
}

function statusDotClass(
  status: JobStatus | "ok" | "warn" | "bad" | "muted" | "",
) {
  const className = statusPillClass(status || "muted");
  if (className.includes("--ok"))
    return "bg-forge-electric shadow-[0_0_14px_rgba(85,214,255,0.45)]";
  if (className.includes("--bad"))
    return "bg-forge-emberSoft shadow-[0_0_14px_rgba(255,122,51,0.42)]";
  if (className.includes("--warn"))
    return "bg-forge-ember shadow-[0_0_14px_rgba(255,154,61,0.38)]";
  return "bg-forge-platinum/35";
}

function metricAccentClass(
  status: JobStatus | "ok" | "warn" | "bad" | "muted",
) {
  const className = statusPillClass(status);
  if (className.includes("--ok")) return "bg-forge-electric/80";
  if (className.includes("--bad")) return "bg-forge-emberSoft/90";
  if (className.includes("--warn")) return "bg-forge-ember/90";
  return "bg-forge-platinum/20";
}

function statusLabel(status: JobStatus) {
  return status.replace(/_/g, " ");
}
