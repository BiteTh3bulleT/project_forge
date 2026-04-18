import type { JobRecord, JobStatus } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
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

  return (
    <div className="space-y-6">
      <Panel
        title="Jobs"
        subtitle="Execution queue, status projections, and replayable event streams."
        actions={<GhostButton onClick={() => void refresh()}>Refresh</GhostButton>}
      >
        <div className="flex flex-wrap items-end gap-3">
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Status filter</label>
            <select aria-label="Job status filter" className="forge-input mt-1" value={filter} onChange={(e) => setFilter(e.target.value as "" | JobStatus)}>
              {statuses.map((s) => (
                <option key={s.label} value={s.value}>
                  {s.label}
                </option>
              ))}
            </select>
          </div>
          <PrimaryButton
            onClick={async () => {
              try {
                const res = await api.jobs.create({
                  templateId: "search_packet",
                  title: "Memory packet",
                  userRequest: "Build a fresh task packet from current memory",
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

        <div className="mt-4 flex flex-wrap gap-2 text-xs text-forge-mist">
          <span className="rounded border border-white/10 px-2 py-1">running {counts.running ?? 0}</span>
          <span className="rounded border border-white/10 px-2 py-1">awaiting {counts.awaiting_approval ?? 0}</span>
          <span className="rounded border border-white/10 px-2 py-1">queued {counts.queued ?? 0}</span>
          <span className="rounded border border-white/10 px-2 py-1">failed {counts.failed ?? 0}</span>
        </div>

        {err ? <div className="mt-4 rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
      </Panel>

      <Panel title="Recent Jobs" subtitle="Select a row to inspect packet, approval state, events, and artifacts.">
        {jobs.length === 0 ? (
          <div className="text-sm text-forge-mist">No jobs yet.</div>
        ) : (
          <div className="space-y-2">
            {jobs.map((j) => (
              <button
                key={j.id}
                type="button"
                className="w-full rounded-md border border-white/10 bg-black/20 p-3 text-left hover:border-forge-ember/35"
                onClick={() => navigate(`/jobs/${j.id}`)}
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="text-sm font-semibold text-forge-ash">{j.title}</div>
                  <div className="text-[11px] text-forge-mist">{j.status}</div>
                </div>
                <div className="mt-2 text-xs text-forge-mist">
                  {j.id} · {j.targetAdapter} · {j.riskClass} · {formatTime(j.createdAtMs)}
                </div>
                {j.lastFailureCode ? (
                  <div className="mt-2 text-xs text-forge-emberSoft">failure: {String(j.lastFailureCode)}</div>
                ) : null}
              </button>
            ))}
          </div>
        )}
      </Panel>
    </div>
  );
}
