import type { JobLineage, JobRecord } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useCallback, useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { AuditJobLink } from "../components/AuditLinks";
import { HumanDataView } from "../components/HumanDataView";
import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function LineagePage() {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [jobId, setJobId] = useState("");
  const [lineage, setLineage] = useState<JobLineage | null>(null);
  const [recentJobs, setRecentJobs] = useState<JobRecord[]>([]);
  const [retryQuery, setRetryQuery] = useState("");
  const [err, setErr] = useState<string | null>(null);

  const loadRecent = useCallback(async () => {
    try {
      const res = await api.jobs.list("", 80);
      setRecentJobs(res.jobs);
      const fromUrl = params.get("jobId")?.trim();
      if (fromUrl) {
        setJobId(fromUrl);
      } else {
        setJobId((prev) =>
          prev.trim() ? prev : res.jobs.length > 0 ? res.jobs[0].id : "",
        );
      }
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }, [params]);

  async function loadLineage(id: string) {
    if (!id.trim()) return;
    try {
      const line = await api.lineage.byJob(id.trim());
      setLineage(line);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setLineage(null);
    }
  }

  useEffect(() => {
    void loadRecent();
  }, [loadRecent]);

  useEffect(() => {
    if (jobId.trim()) {
      void loadLineage(jobId.trim());
    }
  }, [jobId]);

  return (
    <div className="forge-ops-board space-y-5">
      <Panel
        title="Lineage"
        subtitle="Retry/replay chains with explicit parent-child relationships and change summaries."
        actions={
          <GhostButton onClick={() => void loadRecent()}>Refresh</GhostButton>
        }
      >
        {err ? (
          <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
            {err}
          </div>
        ) : null}
        <div className="grid gap-3 md:grid-cols-3">
          <div className="md:col-span-2">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">
              Job id
            </label>
            <input
              aria-label="Lineage job id"
              className="forge-input mt-1"
              value={jobId}
              onChange={(e) => setJobId(e.target.value)}
              placeholder="job_..."
            />
          </div>
          <div className="md:col-span-1">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">
              Retry query override (optional)
            </label>
            <input
              aria-label="Retry query override"
              className="forge-input mt-1"
              value={retryQuery}
              onChange={(e) => setRetryQuery(e.target.value)}
            />
          </div>
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton
            onClick={async () => {
              if (!jobId.trim()) return;
              const res = await api.jobs.retry(jobId.trim(), {
                query: retryQuery || undefined,
                note: "Retry from lineage page",
              });
              setStatus(`Retry job created: ${res.job.id}`);
              setJobId(res.job.id);
              navigate(`/jobs/${res.job.id}`);
            }}
          >
            Retry Job
          </PrimaryButton>
          <GhostButton
            onClick={async () => {
              if (!jobId.trim()) return;
              const res = await api.jobs.replay(jobId.trim(), {
                note: "Replay from lineage page",
              });
              setStatus(`Replay job created: ${res.job.id}`);
              setJobId(res.job.id);
              navigate(`/jobs/${res.job.id}`);
            }}
          >
            Replay Job
          </GhostButton>
          <GhostButton onClick={() => void loadLineage(jobId)}>
            Load Lineage
          </GhostButton>
          {jobId.trim() ? (
            <AuditJobLink
              jobId={jobId.trim()}
              className="forge-btn forge-btn--ghost inline-flex"
            />
          ) : null}
        </div>
      </Panel>

      <Panel
        title="Recent Jobs"
        subtitle="Pick an origin run to inspect and extend lineage."
      >
        {recentJobs.length === 0 ? (
          <div className="text-sm text-forge-mist">No jobs yet.</div>
        ) : (
          <div className="space-y-2">
            {recentJobs.map((j) => (
              <button
                key={j.id}
                type="button"
                className={[
                  "w-full rounded border px-3 py-2 text-left",
                  j.id === jobId
                    ? "border-forge-ember/40 bg-black/30"
                    : "border-forge-platinum/10 bg-black/20 hover:border-forge-ember/35",
                ].join(" ")}
                onClick={() => setJobId(j.id)}
              >
                <div className="text-sm font-semibold text-forge-ash">
                  {j.title}
                </div>
                <div className="mt-1 text-xs text-forge-mist">
                  {j.id} | {j.status} | {j.targetAdapter} |{" "}
                  {formatTime(j.createdAtMs)}
                </div>
              </button>
            ))}
          </div>
        )}
      </Panel>

      <Panel
        title="Lineage Detail"
        subtitle="Parent/child edges with stored change summaries for replayability and debugging."
      >
        {!lineage ? (
          <div className="text-sm text-forge-mist">No lineage loaded.</div>
        ) : (
          <div className="grid gap-4 xl:grid-cols-3">
            <div className="rounded border border-forge-platinum/10 bg-black/20 p-3">
              <div className="text-xs font-semibold tracking-wide text-forge-mist">
                Parents
              </div>
              {lineage.parents.length === 0 ? (
                <div className="mt-2 text-xs text-forge-mist">No parents.</div>
              ) : (
                <div className="mt-2 space-y-2">
                  {lineage.parents.map((edge) => (
                    <div
                      key={edge.id}
                      className="rounded border border-forge-platinum/10 bg-black/30 p-2 text-xs text-forge-mist"
                    >
                      <div>
                        {edge.parentJobId} {"->"} {edge.childJobId}
                      </div>
                      <div className="mt-1">
                        {edge.relationType} | {formatTime(edge.createdAtMs)}
                      </div>
                      <div className="mt-1 max-h-40 overflow-auto text-[11px] text-forge-ash">
                        <HumanDataView value={edge.changeSummary} compact />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div className="rounded border border-forge-platinum/10 bg-black/20 p-3">
              <div className="text-xs font-semibold tracking-wide text-forge-mist">
                Children
              </div>
              {lineage.children.length === 0 ? (
                <div className="mt-2 text-xs text-forge-mist">No children.</div>
              ) : (
                <div className="mt-2 space-y-2">
                  {lineage.children.map((edge) => (
                    <div
                      key={edge.id}
                      className="rounded border border-forge-platinum/10 bg-black/30 p-2 text-xs text-forge-mist"
                    >
                      <div>
                        {edge.parentJobId} {"->"} {edge.childJobId}
                      </div>
                      <div className="mt-1">
                        {edge.relationType} | {formatTime(edge.createdAtMs)}
                      </div>
                      <div className="mt-1 max-h-40 overflow-auto text-[11px] text-forge-ash">
                        <HumanDataView value={edge.changeSummary} compact />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div className="rounded border border-forge-platinum/10 bg-black/20 p-3">
              <div className="text-xs font-semibold tracking-wide text-forge-mist">
                Related Jobs
              </div>
              {lineage.relatedJobs.length === 0 ? (
                <div className="mt-2 text-xs text-forge-mist">
                  No related jobs.
                </div>
              ) : (
                <div className="mt-2 space-y-2">
                  {lineage.relatedJobs.map((j) => (
                    <div
                      key={j.id}
                      className="rounded border border-forge-platinum/10 bg-black/30 p-2 text-xs text-forge-mist"
                    >
                      <button
                        type="button"
                        className="w-full text-left"
                        onClick={() => navigate(`/jobs/${j.id}`)}
                      >
                        <div className="font-semibold text-forge-ash">
                          {j.id}
                        </div>
                        <div className="mt-1">
                          {j.status} | {j.targetAdapter}
                        </div>
                      </button>
                      <div className="mt-2">
                        <AuditJobLink jobId={j.id} />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </Panel>
    </div>
  );
}
