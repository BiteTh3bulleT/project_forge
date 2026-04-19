import type { AdapterInfo, DashboardSummary, ProjectContextRecord, SourceRow } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function StartPage() {
  const navigate = useNavigate();
  const setStatus = useUiStore((s) => s.setStatusLine);

  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [sources, setSources] = useState<SourceRow[]>([]);
  const [contextRecord, setContextRecord] = useState<ProjectContextRecord | null>(null);
  const [adapters, setAdapters] = useState<AdapterInfo[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState<string>("");

  async function load() {
    try {
      const [dash, src, ctx, ad] = await Promise.all([
        api.dashboard.summary(),
        api.sources.list(),
        api.projectContext.get(),
        api.adapters.list(),
      ]);
      setSummary(dash ?? null);
      setSources(Array.isArray(src?.sources) ? src.sources : []);
      setContextRecord(ctx?.record ?? null);
      setAdapters(Array.isArray(ad?.adapters) ? ad.adapters : []);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void load();
    const id = window.setInterval(() => void load(), 5000);
    return () => window.clearInterval(id);
  }, []);

  const activeJobs = Array.isArray(summary?.activeJobs) ? summary.activeJobs : [];
  const recentFailures = Array.isArray(summary?.recentFailures) ? summary.recentFailures : [];

  const readiness = useMemo(
    () => [
      {
        id: "sources",
        title: "Source folders",
        done: sources.length > 0,
        detail: sources.length > 0 ? `${sources.length} source(s) indexed` : "No sources configured yet",
        actionLabel: "Open Sources",
        onAction: () => navigate("/sources"),
      },
      {
        id: "context",
        title: "Project context",
        done: !!contextRecord,
        detail: contextRecord
          ? `Last normalized ${formatTime(contextRecord.generatedAtMs)}`
          : "Context not normalized yet",
        actionLabel: contextRecord ? "Open Project Context" : "Import Context",
        onAction: () => navigate("/project-context"),
      },
      {
        id: "adapters",
        title: "Adapters",
        done: adapters.length > 0 && adapters.some((a) => isAdapterReady(a.status)),
        detail: `${adapters.filter((a) => isAdapterReady(a.status)).length}/${adapters.length} ready`,
        actionLabel: "Open Adapters",
        onAction: () => navigate("/adapters"),
      },
      {
        id: "queue",
        title: "Execution queue",
        done: !!summary,
        detail: summary
          ? activeJobs.length > 0
            ? `${activeJobs.length} active · ${summary.approvalsPending} awaiting approval`
            : "Idle · no active jobs"
          : "Loading queue state",
        actionLabel: "Open Jobs",
        onAction: () => navigate("/jobs"),
      },
    ],
    [activeJobs.length, adapters, contextRecord, navigate, sources.length, summary],
  );

  async function runQuick(name: string, args: Record<string, unknown>, success: string, to: string) {
    setBusy(name);
    try {
      const res = await api.commands.execute(name, args);
      setStatus(res.jobId ? `${success} (${res.jobId})` : success);
      navigate(res.jobId ? `/jobs/${res.jobId}` : to);
      void load();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="space-y-6">
      <Panel
        className="forge-hero"
        title="Start Here"
        subtitle="FORGE in guided mode: connect context, run bounded jobs, review results, and keep control over risky actions."
        actions={<GhostButton onClick={() => void load()}>Refresh</GhostButton>}
      >
        {err ? <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          {readiness.map((row) => (
            <div key={row.id} className="rounded border border-white/10 bg-black/20 p-3">
              <div className="flex items-center justify-between gap-2">
                <div className="text-xs font-semibold uppercase tracking-wide text-forge-mist">{row.title}</div>
                <span
                  className={[
                    "rounded px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide",
                    row.done ? "bg-emerald-500/15 text-emerald-200" : "bg-white/10 text-forge-mist",
                  ].join(" ")}
                >
                  {row.done ? "ready" : "needs setup"}
                </span>
              </div>
              <div className="mt-2 text-xs text-forge-mist">{row.detail}</div>
              <button
                type="button"
                onClick={row.onAction}
                className="mt-3 rounded border border-white/10 bg-black/30 px-2.5 py-1 text-[11px] font-medium text-forge-mist transition hover:border-forge-ember/35 hover:text-forge-ash"
              >
                {row.actionLabel}
              </button>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Quick Actions" subtitle="One-click launches for common operator tasks.">
        <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
          <PrimaryButton onClick={() => navigate("/memory")}>Search Memory</PrimaryButton>
          <GhostButton
            onClick={() =>
              void runQuick(
                "search_packet",
                { query: "Build task packet from current project context" },
                "Packet job queued",
                "/jobs",
              )
            }
            disabled={busy !== ""}
          >
            {busy === "search_packet" ? "Queueing…" : "Build Packet Job"}
          </GhostButton>
          <GhostButton
            onClick={() =>
              void runQuick(
                "ollama_summary",
                { query: "Summarize relevant current context and pending work." },
                "Ollama summary job queued",
                "/jobs",
              )
            }
            disabled={busy !== ""}
          >
            {busy === "ollama_summary" ? "Queueing…" : "Run Ollama Summary"}
          </GhostButton>
          <GhostButton
            onClick={() =>
              void runQuick(
                "normalize_project_context",
                { query: "Normalize project context and refresh guidance files." },
                "Context normalization job queued",
                "/project-context",
              )
            }
            disabled={busy !== ""}
          >
            {busy === "normalize_project_context" ? "Queueing…" : "Normalize Project Context"}
          </GhostButton>
          <GhostButton onClick={() => navigate("/approvals")}>Open Approvals Queue</GhostButton>
          <GhostButton onClick={() => navigate("/reviews")}>Open Review Queue</GhostButton>
        </div>
      </Panel>

      <div className="grid gap-6 xl:grid-cols-2">
        <Panel title="Active Jobs" subtitle="Running or waiting jobs that may need attention.">
          {!summary || activeJobs.length === 0 ? (
            <div className="text-sm text-forge-mist">No active jobs.</div>
          ) : (
            <div className="space-y-2">
              {activeJobs.slice(0, 8).map((job) => (
                <button
                  key={job.id}
                  type="button"
                  onClick={() => navigate(`/jobs/${job.id}`)}
                  className="w-full rounded border border-white/10 bg-black/20 p-3 text-left transition hover:border-forge-ember/35"
                >
                  <div className="text-sm font-semibold text-forge-ash">{job.title}</div>
                  <div className="mt-1 text-xs text-forge-mist">
                    {job.status} · {job.targetAdapter}
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">{formatTime(job.createdAtMs)}</div>
                </button>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="Operator Queue" subtitle="Work blocked on approvals or reviews.">
          {!summary ? (
            <div className="text-sm text-forge-mist">Loading queue counters…</div>
          ) : (
            <div className="space-y-3">
              <QueueRow
                label="Approvals Pending"
                value={summary.approvalsPending}
                actionLabel="Open Approvals"
                onAction={() => navigate("/approvals")}
              />
              <QueueRow
                label="Reviews Pending"
                value={summary.reviewsPending}
                actionLabel="Open Reviews"
                onAction={() => navigate("/reviews")}
              />
              <QueueRow
                label="Recent Failures"
                value={recentFailures.length}
                actionLabel="Inspect Jobs"
                onAction={() => navigate("/jobs")}
              />
            </div>
          )}
        </Panel>
      </div>
    </div>
  );
}

function isAdapterReady(status: string | undefined): boolean {
  const normalized = String(status ?? "").toLowerCase();
  return normalized === "ready" || normalized === "degraded";
}

function QueueRow(props: { label: string; value: number; actionLabel: string; onAction: () => void }) {
  return (
    <div className="rounded border border-white/10 bg-black/20 p-3">
      <div className="flex items-center justify-between gap-2">
        <div className="text-xs uppercase tracking-wide text-forge-mist">{props.label}</div>
        <div className="text-lg font-semibold text-forge-ash">{props.value}</div>
      </div>
      <button
        type="button"
        onClick={props.onAction}
        className="mt-2 rounded border border-white/10 bg-black/30 px-2.5 py-1 text-[11px] font-medium text-forge-mist transition hover:border-forge-ember/35 hover:text-forge-ash"
      >
        {props.actionLabel}
      </button>
    </div>
  );
}
